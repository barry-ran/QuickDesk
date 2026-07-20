/// spake2_authenticator.dart - SPAKE2 认证状态机
///
/// 对照 WebClient/js/auth/spake2.js 的 Spake2Authenticator
/// （其参照 Chromium NegotiatingClientAuthenticator + Spake2Authenticator）。
///
/// 同时支持两个角色：
///   - Client（Alice / initiator）：M1 主控端使用
///   - Host（Bob / responder）：M3 被控端使用
library;

import 'dart:convert';
import 'dart:typed_data';

import 'spake2.dart';

enum AuthState {
  messageReady,
  waitingMessage,
  accepted,
  rejected,
}

/// 认证消息（对应 Jingle XML `<authentication>` 元素的内容）
class AuthMessage {
  String? supportedMethods;
  String? method;
  String? spakeMessage; // base64
  String? verificationHash; // base64
  String? certificate; // base64

  AuthMessage({
    this.supportedMethods,
    this.method,
    this.spakeMessage,
    this.verificationHash,
    this.certificate,
  });

  bool get isEmpty =>
      supportedMethods == null &&
      method == null &&
      spakeMessage == null &&
      verificationHash == null &&
      certificate == null;
}

const String kSpake2Method = 'spake2_curve25519';

/// Client 角色（Alice）认证器，对照 JS Spake2Authenticator
class Spake2ClientAuthenticator {
  final String localId;
  final String remoteId;
  final Uint8List sharedSecretHash;

  late final Spake2Context _ctx;
  Uint8List? _localSpakeMessage;
  bool _spakeMessageSent = false;
  Uint8List? _outgoingVerificationHash;
  AuthState _state = AuthState.messageReady;
  String? rejectionReason;

  bool _methodSelected = false;

  Spake2ClientAuthenticator(this.localId, this.remoteId, this.sharedSecretHash) {
    _ctx = Spake2Context(spake2RoleAlice, localId, remoteId);
    _localSpakeMessage = _ctx.generateMessage(sharedSecretHash);
  }

  /// 首条协商消息：仅 supported-methods
  AuthMessage getFirstNegotiationMessage() {
    _state = AuthState.waitingMessage;
    return AuthMessage(supportedMethods: kSpake2Method);
  }

  /// 处理 Host 回复的认证消息
  void processMessage(AuthMessage message) {
    if (!_methodSelected && message.method != null) {
      if (message.method != kSpake2Method) {
        _state = AuthState.rejected;
        rejectionReason = 'Unsupported method: ${message.method}';
        return;
      }
      _methodSelected = true;
    }

    if (message.spakeMessage != null) {
      try {
        _ctx.processMessage(base64Decode(message.spakeMessage!));
        _outgoingVerificationHash = _ctx.getOutgoingVerificationHash();
      } catch (e) {
        _state = AuthState.rejected;
        rejectionReason = 'Failed to process SPAKE2 message: $e';
        return;
      }
    }

    if (message.verificationHash != null) {
      final valid = _ctx.verifyHash(base64Decode(message.verificationHash!));
      if (!valid) {
        _state = AuthState.rejected;
        rejectionReason = 'Verification hash mismatch';
        return;
      }
      _state = AuthState.accepted;
      return;
    }

    _state = AuthState.messageReady;
  }

  /// 取下一条要发送的认证消息
  AuthMessage getNextMessage() {
    final message = AuthMessage();

    if (_methodSelected) {
      message.method = kSpake2Method;
    }

    if (!_spakeMessageSent) {
      message.spakeMessage = base64Encode(_localSpakeMessage!);
      _spakeMessageSent = true;
    }

    if (_outgoingVerificationHash != null) {
      message.verificationHash = base64Encode(_outgoingVerificationHash!);
      _outgoingVerificationHash = null;
    }

    if (_state != AuthState.accepted) {
      _state = AuthState.waitingMessage;
    }

    return message;
  }

  /// SPAKE2 协商出的 auth key（64 字节），用于 SDP 签名
  Uint8List? get authKey => _ctx.authKey;

  AuthState get state {
    if (_state == AuthState.accepted && _outgoingVerificationHash != null) {
      return AuthState.messageReady;
    }
    return _state;
  }
}

/// Host 角色（Bob）认证器，为 M3 被控端准备。
///
/// 对照 Chromium NegotiatingHostAuthenticator 的最小流程:
///   1. 收 client 首条消息（supported-methods）→ 回复 method + 自己的 SPAKE2 消息
///   2. 收 client 的 SPAKE2 消息 → 计算 key，回复 verification-hash
///   3. 收 client 的 verification-hash → 校验，通过则 ACCEPTED
class Spake2HostAuthenticator {
  final String localId;
  final String remoteId;
  final Uint8List sharedSecretHash;
  final String certificate;

  late final Spake2Context _ctx;
  Uint8List? _localSpakeMessage;
  bool _spakeMessageSent = false;
  Uint8List? _outgoingVerificationHash;
  AuthState _state = AuthState.waitingMessage;
  String? rejectionReason;

  bool _methodSelected = false;
  bool _theirSpakeProcessed = false;

  Spake2HostAuthenticator(
    this.localId,
    this.remoteId,
    this.sharedSecretHash, {
    required this.certificate,
  }) {
    if (certificate.isEmpty) {
      throw ArgumentError.value(certificate, 'certificate', 'must not be empty');
    }
    _ctx = Spake2Context(spake2RoleBob, localId, remoteId);
    _localSpakeMessage = _ctx.generateMessage(sharedSecretHash);
  }

  void processMessage(AuthMessage message) {
    // 客户端首条可发送 supported-methods，或直接携带已选中的 method。
    if (!_methodSelected && message.supportedMethods != null) {
      final methods = message.supportedMethods!.split(RegExp(r'[\s,]+'));
      if (!methods.contains(kSpake2Method)) {
        _state = AuthState.rejected;
        rejectionReason = 'No supported method in: ${message.supportedMethods}';
        return;
      }
      _methodSelected = true;
      _state = AuthState.messageReady;
      return;
    }
    if (!_methodSelected && message.method != null) {
      if (message.method != kSpake2Method) {
        _state = AuthState.rejected;
        rejectionReason = 'Unsupported method: ${message.method}';
        return;
      }
      _methodSelected = true;
      _state = AuthState.messageReady;
      if (message.spakeMessage == null && message.verificationHash == null) {
        return;
      }
    }

    if (message.spakeMessage != null && !_theirSpakeProcessed) {
      try {
        _ctx.processMessage(base64Decode(message.spakeMessage!));
        _theirSpakeProcessed = true;
        _outgoingVerificationHash = _ctx.getOutgoingVerificationHash();
      } catch (e) {
        _state = AuthState.rejected;
        rejectionReason = 'Failed to process SPAKE2 message: $e';
        return;
      }
    }

    if (message.verificationHash != null) {
      final valid = _ctx.verifyHash(base64Decode(message.verificationHash!));
      if (!valid) {
        _state = AuthState.rejected;
        rejectionReason = 'Verification hash mismatch';
        return;
      }
      _state = AuthState.accepted;
      return;
    }

    _state = _theirSpakeProcessed || _methodSelected
        ? AuthState.messageReady
        : AuthState.waitingMessage;
  }

  AuthMessage getNextMessage() {
    final message = AuthMessage(method: kSpake2Method);

    if (!_spakeMessageSent) {
      message.certificate = certificate;
      message.spakeMessage = base64Encode(_localSpakeMessage!);
      _spakeMessageSent = true;
    }

    if (_outgoingVerificationHash != null) {
      message.verificationHash = base64Encode(_outgoingVerificationHash!);
      _outgoingVerificationHash = null;
    }

    if (_state != AuthState.accepted) {
      _state = AuthState.waitingMessage;
    }

    return message;
  }

  Uint8List? get authKey => _ctx.authKey;

  AuthState get state {
    if (_state == AuthState.accepted && _outgoingVerificationHash != null) {
      return AuthState.messageReady;
    }
    return _state;
  }
}
