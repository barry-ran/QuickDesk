/// remote_page.dart - 远程桌面页
///
/// 视频渲染（RTCVideoRenderer）+ 触控板输入 + 虚拟键盘 + 工具条。
library;

import 'dart:async';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:path_provider/path_provider.dart';
import 'package:wakelock_plus/wakelock_plus.dart';

import '../api/signaling_api.dart';
import '../l10n/app_strings.dart';
import '../protocol/client_session.dart';
import '../protocol/file_transfer.dart';
import '../protocol/proto/protobuf_messages.dart';
import 'clipboard_sync.dart';
import 'connection_store.dart';
import 'keycode_mapper.dart';
import 'touch_input.dart';
import 'video_stats.dart';

class RemotePage extends StatefulWidget {
  final String signalingUrl;
  final String deviceId;
  final String accessCode;
  final String signalToken;
  final List<IceServerEntry> iceServers;

  const RemotePage({
    super.key,
    required this.signalingUrl,
    required this.deviceId,
    required this.accessCode,
    required this.signalToken,
    required this.iceServers,
  });

  @override
  State<RemotePage> createState() => _RemotePageState();
}

class _RemotePageState extends State<RemotePage> {
  late final ClientSession _session;
  late final TouchInputController _touch;
  final _renderer = RTCVideoRenderer();

  SessionState _state = SessionState.idle;
  String _statusText = '';
  bool _rendererReady = false;
  final _textInputCtrl = TextEditingController();
  final _textFocus = FocusNode();
  bool _keyboardVisible = false;
  final List<StreamSubscription> _subs = [];

  // 多显示器
  List<VideoTrackLayout> _displays = [];
  int _activeDisplayIndex = 0;
  String? _selectedStreamId;

  // 剪贴板同步
  ClipboardSync? _clipboard;
  final ConnectionStore _history = ConnectionStore();
  bool _historyRecorded = false;

  // 性能统计
  VideoStatsCollector? _stats;
  VideoStatsData? _statsData;
  bool _statsVisible = false;

  // 文件传输
  FileTransferManager? _fileTransfer;
  String _hostCaps = '';

  @override
  void initState() {
    super.initState();
    _statusText = L10n.t('remote.connecting');
    _session = ClientSession(
      signalingUrl: widget.signalingUrl,
      iceServers: widget.iceServers,
    );
    _touch = TouchInputController(_session.dcHandler);
    _touch.onCursorMoved = () {
      if (mounted) setState(() {});
    };
    _init();
  }

  Future<void> _init() async {
    await _renderer.initialize();
    setState(() => _rendererReady = true);

    WakelockPlus.enable();

    _subs.add(_session.onStateChange.listen((s) {
      if (!mounted) return;
      setState(() {
        _state = s;
        _statusText = switch (s) {
          SessionState.connecting => L10n.t('remote.stateConnecting'),
          SessionState.initiating => L10n.t('remote.stateInitiating'),
          SessionState.accepting => L10n.t('remote.stateAccepting'),
          SessionState.authenticating => L10n.t('remote.stateAuthenticating'),
          SessionState.connected => '',
          SessionState.failed =>
            '${L10n.t('remote.stateFailed')}${_session.failureReason != null ? ': ${_session.failureReason}' : ''}',
          SessionState.closed => L10n.t('remote.stateClosed'),
          _ => '',
        };
      });
      if (s == SessionState.connected) {
        _onConnected();
      }
      if (s == SessionState.closed && mounted) {
        Navigator.of(context).maybePop();
      }
    }));

    // 新流到达 → 刷新渲染
    _subs.add(_session.onRemoteStreamsChanged.listen((_) => _applyStream()));

    // 视频尺寸变化 → 更新触控坐标系。
    // 无 VideoLayout，或 VideoLayout 未给出有效尺寸（如被控端采集尺寸为 0）时，
    // 用解码帧的真实尺寸兜底，避免触控分辨率停留在 0 导致光标无法移动。
    _renderer.onResize = () {
      final vw = _renderer.videoWidth.toInt();
      final vh = _renderer.videoHeight.toInt();
      if ((_displays.isEmpty || _touch.remoteWidth <= 0 || _touch.remoteHeight <= 0) &&
          vw > 0 &&
          vh > 0) {
        _touch.setRemoteResolution(vw, vh);
      }
      if (mounted) setState(() {});
    };

    // control 通道就绪后发送初始配置（对照 remote-main.js _sendInitialConfig）
    _subs.add(_session.dcHandler.onControlReady.listen((_) {
      _session.dcHandler.sendCapabilities('');
      _session.dcHandler.sendAudioControl(enable: true);
    }));

    // 记录 host 能力（用于判断是否支持文件传输）
    _subs.add(_session.dcHandler.onCapabilities.listen((caps) {
      if (mounted) setState(() => _hostCaps = caps);
    }));

    _subs.add(_session.dcHandler.onVideoLayout.listen(_onVideoLayout));

    try {
      await _session.connect(widget.deviceId, widget.accessCode, widget.signalToken);
    } catch (_) {
      // 状态流里已处理
    }
  }

  void _onConnected() {
    // 记录连接历史（一次）
    if (!_historyRecorded) {
      _historyRecorded = true;
      _history.recordConnection(widget.deviceId);
    }
    // 启用剪贴板同步
    _clipboard ??= ClipboardSync(_session.dcHandler)..enable();
    // 文件传输管理器
    _fileTransfer ??= FileTransferManager(() => _session.peerConnection!);
  }

  bool get _supportsFileTransfer => _hostCaps.contains('fileTransfer');

  Future<void> _pickAndUpload() async {
    final result = await FilePicker.platform.pickFiles(withData: false);
    final picked = (result != null && result.files.isNotEmpty) ? result.files.first : null;
    if (picked == null || picked.path == null || _fileTransfer == null) return;
    final bytes = await File(picked.path!).readAsBytes();
    if (!mounted) return;
    _showTransferProgress(
      title: L10n.t('file.uploading'),
      stream: _fileTransfer!.upload(picked.name, bytes),
    );
  }

  Future<void> _startDownload() async {
    if (_fileTransfer == null) return;
    final dir = await _downloadDir();
    if (!mounted) return;
    _showTransferProgress(
      title: L10n.t('file.downloading'),
      stream: _fileTransfer!.download(dir),
    );
  }

  Future<String> _downloadDir() async {
    // 优先外部 Downloads，退回应用文档目录
    try {
      final ext = await getExternalStorageDirectory();
      if (ext != null) {
        final downloads = Directory('${ext.path}${Platform.pathSeparator}QuickDesk');
        if (!downloads.existsSync()) downloads.createSync(recursive: true);
        return downloads.path;
      }
    } catch (_) {}
    final docs = await getApplicationDocumentsDirectory();
    return docs.path;
  }

  void _showTransferProgress({
    required String title,
    required Stream<FileTransferProgress> stream,
  }) {
    showModalBottomSheet<void>(
      context: context,
      isDismissible: false,
      enableDrag: false,
      builder: (ctx) {
        return _TransferSheet(title: title, stream: stream);
      },
    );
  }

  void _toggleStats() {
    setState(() => _statsVisible = !_statsVisible);
    if (_statsVisible) {
      _stats ??= VideoStatsCollector(
        fetch: _session.getStats,
        onUpdate: (d) {
          if (mounted) setState(() => _statsData = d);
        },
      );
      _stats!.start();
    } else {
      _stats?.stop();
    }
  }

  // ==================== 多显示器 ====================

  void _onVideoLayout(VideoLayoutMsg layout) {
    if (layout.videoTracks.isEmpty) return;
    _displays = layout.videoTracks;

    VideoTrackLayout? selected;
    var index = 0;

    if (_selectedStreamId != null) {
      for (var i = 0; i < _displays.length; i++) {
        if (_displays[i].mediaStreamId == _selectedStreamId) {
          selected = _displays[i];
          index = i;
          break;
        }
      }
    }
    if (selected == null && layout.primaryScreenId != null) {
      for (var i = 0; i < _displays.length; i++) {
        if (_displays[i].screenId == layout.primaryScreenId) {
          selected = _displays[i];
          index = i;
          break;
        }
      }
    }
    selected ??= _displays.first;

    _activeDisplayIndex = index;
    _selectedStreamId = selected.mediaStreamId;
    if ((selected.width ?? 0) > 0 && (selected.height ?? 0) > 0) {
      _touch.setRemoteResolution(selected.width!, selected.height!);
    }
    if (mounted) setState(() {});
    _applyStream();
  }

  void _selectDisplay(int index) {
    if (index < 0 || index >= _displays.length) return;
    final track = _displays[index];
    _selectedStreamId = track.mediaStreamId;
    _activeDisplayIndex = index;
    if ((track.width ?? 0) > 0 && (track.height ?? 0) > 0) {
      _touch.setRemoteResolution(track.width!, track.height!);
    }
    _applyStream();
    if (mounted) setState(() {});
  }

  void _applyStream() {
    MediaStream? stream;
    if (_selectedStreamId != null) {
      stream = _session.remoteStreams[_selectedStreamId];
    }
    stream ??= _session.remoteStreams.values.isEmpty
        ? null
        : _session.remoteStreams.values.first;
    if (stream == null) return;
    if (_renderer.srcObject?.id != stream.id) {
      _renderer.srcObject = stream;
      if (mounted) setState(() {});
    }
  }

  // ==================== 键盘 ====================

  void _toggleKeyboard() {
    setState(() => _keyboardVisible = !_keyboardVisible);
    if (_keyboardVisible) {
      _textFocus.requestFocus();
    } else {
      _textFocus.unfocus();
    }
  }

  /// 软键盘按键 → USB HID keycode 注入；可打印字符走 TextEvent
  KeyEventResult _onKey(FocusNode node, KeyEvent event) {
    final usb = KeycodeMapper.logicalToUsb(event.logicalKey);
    if (usb == null) return KeyEventResult.ignored;

    if (event is KeyDownEvent) {
      _session.dcHandler.sendKeyEvent(KeyEventMsg(pressed: true, usbKeycode: usb));
    } else if (event is KeyUpEvent) {
      _session.dcHandler.sendKeyEvent(KeyEventMsg(pressed: false, usbKeycode: usb));
    }
    return KeyEventResult.handled;
  }

  void _onTextChanged(String value) {
    if (value.isEmpty) return;
    // 输入框只是文本中转：把增量文本发送到远端后立即清空
    _session.dcHandler.sendTextEvent(value);
    _textInputCtrl.clear();
  }

  // ==================== 触控 ====================

  int _pointerCount = 0;
  final Map<int, Offset> _pointers = {};

  Offset? get _twoFingerCenter {
    if (_pointers.length < 2) return null;
    final positions = _pointers.values.toList();
    return (positions[0] + positions[1]) / 2;
  }

  void _handlePointerDown(PointerDownEvent e) {
    _pointers[e.pointer] = e.position;
    _pointerCount = _pointers.length;
    _touch.onPointerDown(e, _pointerCount);
    if (_pointerCount == 2) {
      _touch.onTwoFingerStart(_twoFingerCenter!);
    }
  }

  void _handlePointerMove(PointerMoveEvent e) {
    _pointers[e.pointer] = e.position;
    _touch.onPointerMove(e, _pointerCount, twoFingerCenter: _twoFingerCenter);
  }

  void _handlePointerUp(PointerUpEvent e) {
    _pointers.remove(e.pointer);
    _touch.onPointerUp(e, _pointers.length);
    _pointerCount = _pointers.length;
  }

  // ==================== UI ====================

  @override
  Widget build(BuildContext context) {
    final connected = _state == SessionState.connected;

    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        await _session.disconnect();
        if (context.mounted) Navigator.of(context).pop();
      },
      child: Scaffold(
        backgroundColor: Colors.black,
        body: SafeArea(
          child: Stack(
            children: [
              // 视频画面（InteractiveViewer 提供捏合缩放/平移浏览）
              if (_rendererReady)
                Positioned.fill(
                  child: Listener(
                    onPointerDown: _handlePointerDown,
                    onPointerMove: _handlePointerMove,
                    onPointerUp: _handlePointerUp,
                    onPointerCancel: (e) {
                      _pointers.remove(e.pointer);
                      _pointerCount = _pointers.length;
                    },
                    behavior: HitTestBehavior.opaque,
                    child: RTCVideoView(
                      _renderer,
                      objectFit: RTCVideoViewObjectFit.RTCVideoViewObjectFitContain,
                    ),
                  ),
                ),

              // 虚拟光标
              if (connected && _touch.remoteWidth > 0)
                _buildVirtualCursor(context),

              // 连接状态遮罩
              if (!connected && _state != SessionState.closed)
                Positioned.fill(
                  child: Container(
                    color: Colors.black87,
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        if (_state != SessionState.failed)
                          const CircularProgressIndicator(),
                        const SizedBox(height: 24),
                        Text(_statusText,
                            style: const TextStyle(color: Colors.white70, fontSize: 16)),
                        if (_state == SessionState.failed) ...[
                          const SizedBox(height: 24),
                          FilledButton(
                            onPressed: () => Navigator.of(context).pop(),
                            child: Text(L10n.t('common.back')),
                          ),
                        ],
                      ],
                    ),
                  ),
                ),

              // 隐藏的文本输入框（软键盘中转）
              if (_keyboardVisible)
                Positioned(
                  left: 0,
                  right: 0,
                  bottom: 0,
                  child: Container(
                    color: Theme.of(context).colorScheme.surface,
                    padding: const EdgeInsets.all(8),
                    child: Focus(
                      onKeyEvent: _onKey,
                      child: TextField(
                        controller: _textInputCtrl,
                        focusNode: _textFocus,
                        autofocus: true,
                        onChanged: _onTextChanged,
                        decoration: InputDecoration(
                          hintText: L10n.t('remote.textInputHint'),
                          border: const OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                    ),
                  ),
                ),

              // 性能统计叠层
              if (connected && _statsVisible)
                Positioned(
                  top: 56,
                  right: 8,
                  child: _buildStatsPanel(),
                ),

              // 浮动工具条
              if (connected)
                Positioned(
                  top: 8,
                  right: 8,
                  child: _buildToolbar(context),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildStatsPanel() {
    final d = _statsData;
    final rows = <(String, String)>[
      (L10n.t('stats.resolution'), d?.resolution ?? '—'),
      (L10n.t('stats.codec'), d?.codec ?? '—'),
      (L10n.t('stats.fps'), d != null ? '${d.fps} fps' : '—'),
      (L10n.t('stats.bitrate'), d != null ? _fmtBitrate(d.bitrateKbps) : '—'),
      (L10n.t('stats.rtt'), d != null ? '${d.rttMs} ms' : '—'),
      (L10n.t('stats.jitter'), d != null ? '${d.jitterMs.toStringAsFixed(1)} ms' : '—'),
      (L10n.t('stats.packetsLost'), d != null ? '${d.packetsLost}' : '—'),
      (L10n.t('stats.framesDropped'), d != null ? '${d.framesDropped}' : '—'),
      (L10n.t('stats.route'), d?.routeType ?? '—'),
      (L10n.t('stats.protocol'), d?.protocol ?? '—'),
    ];

    return Material(
      color: Colors.black.withValues(alpha: 0.72),
      borderRadius: BorderRadius.circular(10),
      child: Container(
        width: 220,
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(L10n.t('stats.title'),
                style: const TextStyle(
                    color: Colors.white, fontWeight: FontWeight.bold, fontSize: 13)),
            const SizedBox(height: 8),
            for (final (label, value) in rows)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(label,
                        style: const TextStyle(color: Colors.white54, fontSize: 12)),
                    const SizedBox(width: 8),
                    Flexible(
                      child: Text(
                        value,
                        textAlign: TextAlign.right,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(color: Colors.white, fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }

  String _fmtBitrate(int kbps) {
    if (kbps <= 0) return '—';
    if (kbps >= 1000) return '${(kbps / 1000).toStringAsFixed(1)} Mbps';
    return '$kbps kbps';
  }

  Widget _buildVirtualCursor(BuildContext context) {
    // 将远程坐标映射到屏幕坐标（contain 模式黑边补偿）
    return LayoutBuilder(builder: (context, constraints) {
      final vw = _touch.remoteWidth.toDouble();
      final vh = _touch.remoteHeight.toDouble();
      final cw = constraints.maxWidth;
      final ch = constraints.maxHeight;
      if (vw <= 0 || vh <= 0) return const SizedBox.shrink();

      final videoAspect = vw / vh;
      final containerAspect = cw / ch;
      double renderW, renderH, offX, offY;
      if (containerAspect > videoAspect) {
        renderH = ch;
        renderW = renderH * videoAspect;
        offX = (cw - renderW) / 2;
        offY = 0;
      } else {
        renderW = cw;
        renderH = renderW / videoAspect;
        offX = 0;
        offY = (ch - renderH) / 2;
      }

      final sx = offX + (_touch.cursorX / vw) * renderW;
      final sy = offY + (_touch.cursorY / vh) * renderH;

      return Positioned(
        left: sx - 3,
        top: sy - 1,
        child: IgnorePointer(
          child: CustomPaint(size: const Size(20, 20), painter: _CursorPainter()),
        ),
      );
    });
  }

  Widget _buildToolbar(BuildContext context) {
    return Material(
      color: Colors.black54,
      borderRadius: BorderRadius.circular(24),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (_displays.length > 1)
            PopupMenuButton<int>(
              icon: const Icon(Icons.desktop_windows, color: Colors.white),
              tooltip: L10n.t('remote.switchDisplay'),
              onSelected: _selectDisplay,
              itemBuilder: (context) => [
                for (var i = 0; i < _displays.length; i++)
                  PopupMenuItem<int>(
                    value: i,
                    child: Row(
                      children: [
                        Icon(
                          i == _activeDisplayIndex
                              ? Icons.radio_button_checked
                              : Icons.radio_button_unchecked,
                          size: 18,
                        ),
                        const SizedBox(width: 8),
                        Text('${L10n.t('remote.display', {'n': i + 1})}'
                            '${(_displays[i].width ?? 0) > 0 ? '  ${_displays[i].width}×${_displays[i].height}' : ''}'),
                      ],
                    ),
                  ),
              ],
            ),
          IconButton(
            icon: Icon(Icons.keyboard, color: _keyboardVisible ? Colors.lightBlueAccent : Colors.white),
            tooltip: L10n.t('remote.keyboard'),
            onPressed: _toggleKeyboard,
          ),
          IconButton(
            icon: const Icon(Icons.content_paste_go, color: Colors.white),
            tooltip: L10n.t('remote.sendClipboard'),
            onPressed: () async {
              await _clipboard?.syncNow();
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(L10n.t('remote.clipboardSent')), duration: const Duration(seconds: 1)),
                );
              }
            },
          ),
          IconButton(
            icon: Icon(Icons.insights,
                color: _statsVisible ? Colors.lightBlueAccent : Colors.white),
            tooltip: L10n.t('remote.stats'),
            onPressed: _toggleStats,
          ),
          if (_supportsFileTransfer)
            PopupMenuButton<String>(
              icon: const Icon(Icons.folder_open, color: Colors.white),
              tooltip: L10n.t('file.transfer'),
              onSelected: (v) {
                if (v == 'upload') _pickAndUpload();
                if (v == 'download') _startDownload();
              },
              itemBuilder: (context) => [
                PopupMenuItem(value: 'upload', child: Text(L10n.t('file.upload'))),
                PopupMenuItem(value: 'download', child: Text(L10n.t('file.download'))),
              ],
            ),
          IconButton(
            icon: const Icon(Icons.close, color: Colors.white),
            tooltip: L10n.t('remote.disconnect'),
            onPressed: () async {
              await _session.disconnect();
              if (context.mounted) Navigator.of(context).pop();
            },
          ),
        ],
      ),
    );
  }

  @override
  void dispose() {
    for (final s in _subs) {
      s.cancel();
    }
    _clipboard?.dispose();
    _stats?.stop();
    WakelockPlus.disable();
    _touch.dispose();
    _session.dispose();
    _renderer.dispose();
    _textInputCtrl.dispose();
    _textFocus.dispose();
    super.dispose();
  }
}

/// 文件传输进度底部弹层
class _TransferSheet extends StatefulWidget {
  final String title;
  final Stream<FileTransferProgress> stream;

  const _TransferSheet({required this.title, required this.stream});

  @override
  State<_TransferSheet> createState() => _TransferSheetState();
}

class _TransferSheetState extends State<_TransferSheet> {
  FileTransferProgress? _p;
  StreamSubscription<FileTransferProgress>? _sub;

  @override
  void initState() {
    super.initState();
    _sub = widget.stream.listen((p) {
      if (mounted) setState(() => _p = p);
    });
  }

  @override
  Widget build(BuildContext context) {
    final p = _p;
    final done = p?.done == true;
    final error = p?.error;
    final finished = done || error != null;

    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(widget.title, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 12),
            if (p != null) Text(p.filename, style: const TextStyle(fontWeight: FontWeight.w500)),
            const SizedBox(height: 8),
            if (error != null)
              Text('${L10n.t('file.failed')}: $error',
                  style: TextStyle(color: Theme.of(context).colorScheme.error))
            else ...[
              LinearProgressIndicator(value: p != null && p.totalBytes > 0 ? p.fraction : null),
              const SizedBox(height: 8),
              Text(_progressText(p, done)),
            ],
            const SizedBox(height: 16),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton(
                onPressed: finished ? () => Navigator.of(context).pop() : null,
                child: Text(finished ? L10n.t('common.back') : L10n.t('common.cancel')),
              ),
            ),
          ],
        ),
      ),
    );
  }

  String _progressText(FileTransferProgress? p, bool done) {
    if (p == null) return '...';
    if (done) {
      return p.savedPath != null
          ? L10n.t('file.savedTo', {'path': p.savedPath})
          : L10n.t('file.done');
    }
    final kb = (p.bytes / 1024).toStringAsFixed(0);
    final total = p.totalBytes > 0 ? (p.totalBytes / 1024).toStringAsFixed(0) : '?';
    return '$kb KB / $total KB';
  }

  @override
  void dispose() {
    _sub?.cancel();
    super.dispose();
  }
}

/// 标准箭头光标（对照 touch-handler.js 的 SVG path）
class _CursorPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final path = Path()
      ..moveTo(3, 2)
      ..lineTo(3, 17)
      ..lineTo(7.5, 12.5)
      ..lineTo(11, 19)
      ..lineTo(13.5, 18)
      ..lineTo(10, 11.5)
      ..lineTo(16, 11.5)
      ..close();

    canvas.drawPath(path, Paint()..color = Colors.white);
    canvas.drawPath(
      path,
      Paint()
        ..color = Colors.black
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.2
        ..strokeJoin = StrokeJoin.round,
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
