/// websocket_transport.dart - 信令 WebSocket 传输层
///
/// 对照 WebClient/js/signaling/websocket-transport.js：
///   1. 建立 WS 到 /v1/realtime/signal（URL 不带 token/code）
///   2. 首帧发送 {type:"auth", signal_token, role, device_id, client_id}
///   3. 服务端回 {type:"auth_ok"} 后才允许发送 Jingle XML
///   4. auth_ok 超时（服务端 5s，客户端 6s）则失败
///   5. signal_token：client 角色可续期、host 角色一次性
///
/// 信令帧封装（§2.26，Chromium host 的 SendJingleEnvelope 格式）：
///   发送：`{client_id, payload:"<iq...>"}`
///     - client 角色：client_id 填自己（服务端会用连接身份覆盖）
///     - host 角色：client_id 填目标客户端，服务端据此路由
///   接收：`{client_id, payload}` → 解包出 XML，并带上来源 client_id
///     （host 角色用它区分多个并发客户端）
library;

import 'dart:async';
import 'dart:convert';
import 'dart:io';

const _authTimeout = Duration(seconds: 6);

class WebSocketTransport {
  final String signalingUrl;

  /// 业务消息回调：(xmlPayload, fromClientId)
  /// fromClientId 仅 host 角色有意义（区分并发客户端），client 角色可忽略。
  final void Function(String message, String? fromClientId) onMessage;
  final void Function()? onAuthOk;
  final void Function(int? code, String? reason)? onClose;
  final void Function(Object error)? onError;

  WebSocket? _ws;
  bool _authOk = false;
  bool _closed = false;
  Timer? _authTimer;
  String _selfClientId = '';

  WebSocketTransport({
    required this.signalingUrl,
    required this.onMessage,
    this.onAuthOk,
    this.onClose,
    this.onError,
  });

  bool get isConnected => _ws != null && _authOk && !_closed;

  /// 连接并完成首帧认证。返回的 Future 在 auth_ok 后 resolve。
  Future<void> connect({
    required String deviceId,
    required String signalToken,
    String clientId = '',
    String role = 'client',
  }) async {
    _closed = false;
    _authOk = false;
    _selfClientId = clientId;

    final base = signalingUrl.replaceAll(RegExp(r'/+$'), '');
    final wsUrl = '$base/v1/realtime/signal';

    _ws = await WebSocket.connect(wsUrl);

    final authCompleter = Completer<void>();

    _authTimer = Timer(_authTimeout, () {
      if (!authCompleter.isCompleted) {
        authCompleter.completeError(TimeoutException('signaling auth_ok timeout'));
        _ws?.close(4401, 'auth timeout');
      }
    });

    _ws!.listen(
      (data) {
        final message = data is String ? data : utf8.decode(data as List<int>);
        _handleRawMessage(message, authCompleter);
      },
      onDone: () {
        _authTimer?.cancel();
        if (!authCompleter.isCompleted) {
          authCompleter.completeError(StateError(
              'signaling closed during auth: ${_ws?.closeCode} ${_ws?.closeReason}'));
        }
        if (!_closed) {
          onClose?.call(_ws?.closeCode, _ws?.closeReason);
        }
      },
      onError: (e) {
        _authTimer?.cancel();
        if (!authCompleter.isCompleted) authCompleter.completeError(e as Object);
        onError?.call(e as Object);
      },
    );

    // 首帧认证
    final authFrame = <String, dynamic>{
      'type': 'auth',
      'signal_token': signalToken,
      'role': role,
      'device_id': deviceId,
    };
    if (clientId.isNotEmpty) {
      authFrame['client_id'] = clientId;
    }
    _ws!.add(jsonEncode(authFrame));

    return authCompleter.future;
  }

  void _handleRawMessage(String message, Completer<void> authCompleter) {
    final trimmed = message.trim();

    if (trimmed.startsWith('{')) {
      Map<String, dynamic>? json;
      try {
        json = jsonDecode(trimmed) as Map<String, dynamic>;
      } catch (_) {
        json = null;
      }
      if (json != null) {
        final type = json['type'];
        if (type == 'auth_ok') {
          _authOk = true;
          _authTimer?.cancel();
          onAuthOk?.call();
          if (!authCompleter.isCompleted) authCompleter.complete();
          return;
        }
        if (type == 'error') {
          if (!_authOk) {
            final code = json['code'] ?? (json['data'] is Map ? json['data']['code'] : null);
            if (!authCompleter.isCompleted) {
              authCompleter.completeError(StateError('signaling auth failed: $code'));
            }
            return;
          }
          // auth_ok 之后的错误帧（HOST_OFFLINE/PEER_DISCONNECTED…）转交上层
          onMessage(trimmed, json['client_id'] as String?);
          return;
        }
        // 承载 Jingle XML 的信令封装 {client_id, payload}
        final payload = json['payload'];
        if (payload is String) {
          if (!_authOk) return;
          onMessage(payload, json['client_id'] as String?);
          return;
        }
        // 其它 JSON 控制帧
        if (_authOk) onMessage(trimmed, json['client_id'] as String?);
        return;
      }
    }

    // 裸 XML（兜底：服务端理论上不会直接发裸 XML）
    if (!_authOk) return;
    onMessage(trimmed, null);
  }

  /// 发送一条 Jingle XML。
  /// [targetClientId] 仅 host 角色需要（指定回给哪个客户端）；client 角色留空。
  void send(String message, {String? targetClientId}) {
    if (!isConnected) {
      throw StateError('WebSocket not connected/authenticated');
    }
    final clientId = targetClientId ?? _selfClientId;
    final envelope = <String, dynamic>{
      'client_id': clientId,
      'payload': message,
    };
    _ws!.add(jsonEncode(envelope));
  }

  void disconnect() {
    _closed = true;
    _authTimer?.cancel();
    _ws?.close(1000, 'disconnect');
    _ws = null;
    _authOk = false;
  }
}
