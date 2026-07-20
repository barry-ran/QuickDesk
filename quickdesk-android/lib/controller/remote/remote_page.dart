/// remote_page.dart - 远程桌面页
///
/// 视频渲染（RTCVideoRenderer）+ 触控板输入 + 虚拟键盘；
/// 工具条 / 统计面板 / 文件传输弹层见同目录组件。
library;

import 'dart:async';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:path_provider/path_provider.dart';
import 'package:wakelock_plus/wakelock_plus.dart';

import '../../api/signaling_api.dart';
import '../../core/geometry.dart';
import '../../l10n/app_strings.dart';
import '../../protocol/client_session.dart';
import '../../protocol/file_transfer.dart';
import '../../protocol/proto/protobuf_messages.dart';
import '../clipboard_sync.dart';
import '../connection_store.dart';
import '../keycode_mapper.dart';
import '../touch_input.dart';
import '../video_stats.dart';
import 'cursor_painter.dart';
import 'remote_toolbar.dart';
import 'stats_panel.dart';
import 'transfer_sheet.dart';

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

  bool _firstFrame = false;
  /// true = 横屏锁定；false = 竖屏锁定。进入远程页默认横屏（桌面被控更合适）。
  bool _landscape = true;

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
    _touch.onTransformChanged = () {
      if (mounted) setState(() {});
    };
    _applyOrientation();
    _init();
  }

  Future<void> _applyOrientation() async {
    await SystemChrome.setPreferredOrientations(
      _landscape
          ? const [
              DeviceOrientation.landscapeLeft,
              DeviceOrientation.landscapeRight,
            ]
          : const [
              DeviceOrientation.portraitUp,
              DeviceOrientation.portraitDown,
            ],
    );
    // 沉浸式：横屏时少占一点边距
    await SystemChrome.setEnabledSystemUIMode(
      SystemUiMode.immersiveSticky,
    );
  }

  Future<void> _toggleOrientation() async {
    setState(() => _landscape = !_landscape);
    await _applyOrientation();
  }

  Future<void> _restoreOrientation() async {
    await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    await SystemChrome.setPreferredOrientations(DeviceOrientation.values);
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

    // 首帧已绘制 → 去掉「等待画面」遮罩
    _renderer.onFirstFrameRendered = () {
      if (!_firstFrame) {
        _firstFrame = true;
        if (mounted) setState(() {});
      }
    };

    // 视频尺寸变化 → 更新触控坐标系。
    // 无 VideoLayout，或 VideoLayout 未给出有效尺寸（如被控端采集尺寸为 0）时，
    // 用解码帧的真实尺寸兜底，避免触控分辨率停留在 0 导致光标无法移动。
    _renderer.onResize = () {
      final vw = _renderer.videoWidth.toInt();
      final vh = _renderer.videoHeight.toInt();
      if (vw > 0 && vh > 0) {
        _firstFrame = true;
      }
      if ((_displays.isEmpty ||
              _touch.remoteWidth <= 0 ||
              _touch.remoteHeight <= 0) &&
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
      await _session.connect(
          widget.deviceId, widget.accessCode, widget.signalToken);
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
    final picked =
        (result != null && result.files.isNotEmpty) ? result.files.first : null;
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
        final downloads =
            Directory('${ext.path}${Platform.pathSeparator}QuickDesk');
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
      builder: (ctx) => TransferSheet(title: title, stream: stream),
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

  Future<void> _applyStream() async {
    MediaStream? stream;
    if (_selectedStreamId != null) {
      stream = _session.remoteStreams[_selectedStreamId];
    }
    stream ??= _session.remoteStreams.values.isEmpty
        ? null
        : _session.remoteStreams.values.first;
    if (stream == null) return;

    final videoTracks = stream.getVideoTracks();
    final trackId = videoTracks.isEmpty ? null : videoTracks.first.id;

    // 显式绑定 video track，避免仅设 stream 时 Android 纹理未正确挂上
    try {
      await _renderer.setSrcObject(stream: stream, trackId: trackId);
    } catch (_) {
      _renderer.srcObject = stream;
    }
    if (mounted) setState(() {});
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
      _session.dcHandler
          .sendKeyEvent(KeyEventMsg(pressed: true, usbKeycode: usb));
    } else if (event is KeyUpEvent) {
      _session.dcHandler
          .sendKeyEvent(KeyEventMsg(pressed: false, usbKeycode: usb));
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
    return twoFingerCenterOf(positions[0], positions[1]);
  }

  double? get _twoFingerDistance {
    if (_pointers.length < 2) return null;
    final positions = _pointers.values.toList();
    return twoFingerDistance(positions[0], positions[1]);
  }

  void _handlePointerDown(PointerDownEvent e) {
    _pointers[e.pointer] = e.position;
    _pointerCount = _pointers.length;
    _touch.onPointerDown(e, _pointerCount);
    if (_pointerCount == 2) {
      final center = _twoFingerCenter!;
      final dist = _twoFingerDistance!;
      _touch.onTwoFingerStart(center, dist);
    }
  }

  void _handlePointerMove(PointerMoveEvent e) {
    _pointers[e.pointer] = e.position;
    _touch.onPointerMove(
      e,
      _pointerCount,
      twoFingerCenter: _twoFingerCenter,
      twoFingerDistance: _twoFingerDistance,
    );
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
              // 视频画面：捏合缩放 / 平移由 TouchInputController 驱动
              if (_rendererReady)
                Positioned.fill(
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      _touch.setViewportSize(
                          constraints.maxWidth, constraints.maxHeight);
                      return Listener(
                        onPointerDown: _handlePointerDown,
                        onPointerMove: _handlePointerMove,
                        onPointerUp: _handlePointerUp,
                        onPointerCancel: (e) {
                          _pointers.remove(e.pointer);
                          _pointerCount = _pointers.length;
                        },
                        behavior: HitTestBehavior.opaque,
                        child: ClipRect(
                          child: Transform(
                            transform: _touch.buildTransformMatrix(),
                            child: _buildRemoteVideo(),
                          ),
                        ),
                      );
                    },
                  ),
                ),

              // 已连接但还没收到远程画面时的提示
              if (connected && !_firstFrame)
                Positioned.fill(
                  child: IgnorePointer(
                    child: Container(
                      color: Colors.black,
                      alignment: Alignment.center,
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          const CircularProgressIndicator(),
                          const SizedBox(height: 20),
                          Text(L10n.t('remote.waitingFrame'),
                              style: const TextStyle(
                                  color: Colors.white70, fontSize: 15)),
                        ],
                      ),
                    ),
                  ),
                ),

              // 虚拟光标（随画布缩放平移）
              if (connected && _firstFrame && _touch.remoteWidth > 0)
                _buildVirtualCursor(context),

              // 左键按住提示
              if (connected && _touch.leftButtonHeld)
                Positioned(
                  left: 12,
                  bottom: 12,
                  child: Material(
                    color: Colors.orange.withValues(alpha: 0.85),
                    borderRadius: BorderRadius.circular(8),
                    child: Padding(
                      padding: const EdgeInsets.symmetric(
                          horizontal: 10, vertical: 6),
                      child: Text(
                        L10n.t('remote.leftHold'),
                        style:
                            const TextStyle(color: Colors.white, fontSize: 12),
                      ),
                    ),
                  ),
                ),

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
                            style: const TextStyle(
                                color: Colors.white70, fontSize: 16)),
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
                  child: StatsPanel(data: _statsData),
                ),

              // 浮动工具条
              if (connected)
                Positioned(
                  top: 8,
                  right: 8,
                  child: RemoteToolbar(
                    displays: _displays,
                    activeDisplayIndex: _activeDisplayIndex,
                    landscape: _landscape,
                    zoomed: _touch.isZoomed,
                    keyboardVisible: _keyboardVisible,
                    statsVisible: _statsVisible,
                    supportsFileTransfer: _supportsFileTransfer,
                    onSelectDisplay: _selectDisplay,
                    onToggleOrientation: _toggleOrientation,
                    onResetZoom: () {
                      _touch.resetZoom();
                      setState(() {});
                    },
                    onToggleKeyboard: _toggleKeyboard,
                    onSendClipboard: () async => _clipboard?.syncNow(),
                    onToggleStats: _toggleStats,
                    onUpload: _pickAndUpload,
                    onDownload: _startDownload,
                    onDisconnect: () async {
                      await _session.disconnect();
                      if (context.mounted) Navigator.of(context).pop();
                    },
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  /// 自定义视频层：按视频宽高比 letterbox，Texture 强制铺满计算出的区域。
  Widget _buildRemoteVideo() {
    return ColoredBox(
      color: Colors.black,
      child: ListenableBuilder(
        listenable: _renderer,
        builder: (context, _) {
          final textureId = _renderer.textureId;
          if (textureId == null || !_renderer.renderVideo) {
            return const SizedBox.expand();
          }

          final vw = _renderer.videoWidth.toDouble();
          final vh = _renderer.videoHeight.toDouble();
          final aspect =
              (vw > 0 && vh > 0) ? vw / vh : _renderer.value.aspectRatio;

          return LayoutBuilder(
            builder: (context, constraints) {
              final maxW = constraints.maxWidth;
              final maxH = constraints.maxHeight;
              if (!maxW.isFinite || !maxH.isFinite || maxW <= 0 || maxH <= 0) {
                return SizedBox.expand(
                  child: Texture(textureId: textureId),
                );
              }

              // contain：完整显示远程画面，必要时留黑边；
              // 与触控坐标映射共用 fitContain，保证光标与画面对齐
              final rect = fitContain(
                contentW: aspect,
                contentH: 1,
                boxW: maxW,
                boxH: maxH,
              );

              return Center(
                child: SizedBox(
                  width: rect.width,
                  height: rect.height,
                  child: Texture(
                    textureId: textureId,
                    filterQuality: FilterQuality.low,
                  ),
                ),
              );
            },
          );
        },
      ),
    );
  }

  Widget _buildVirtualCursor(BuildContext context) {
    return Positioned.fill(
      child: LayoutBuilder(builder: (context, constraints) {
        _touch.setViewportSize(constraints.maxWidth, constraints.maxHeight);
        final screen = _touch.cursorScreenPosition();
        if (screen == null) return const SizedBox.shrink();

        return Stack(
          children: [
            Positioned(
              left: screen.dx - 3,
              top: screen.dy - 1,
              child: IgnorePointer(
                child: CustomPaint(
                    size: const Size(20, 20), painter: const CursorPainter()),
              ),
            ),
          ],
        );
      }),
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
    _restoreOrientation();
    _touch.dispose();
    _session.dispose();
    _renderer.dispose();
    _textInputCtrl.dispose();
    _textFocus.dispose();
    super.dispose();
  }
}
