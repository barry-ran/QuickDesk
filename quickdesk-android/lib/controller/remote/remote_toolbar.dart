/// remote_toolbar.dart - 远程页浮动工具条
library;

import 'package:flutter/material.dart';

import '../../l10n/app_strings.dart';
import '../../protocol/proto/protobuf_messages.dart';

class RemoteToolbar extends StatelessWidget {
  final List<VideoTrackLayout> displays;
  final int activeDisplayIndex;
  final bool landscape;
  final bool zoomed;
  final bool keyboardVisible;
  final bool statsVisible;
  final bool supportsFileTransfer;
  final ValueChanged<int> onSelectDisplay;
  final VoidCallback onToggleOrientation;
  final VoidCallback onResetZoom;
  final VoidCallback onToggleKeyboard;
  final Future<void> Function() onSendClipboard;
  final VoidCallback onToggleStats;
  final VoidCallback onUpload;
  final VoidCallback onDownload;
  final VoidCallback? onTransfers;
  final VoidCallback onDisconnect;
  final VoidCallback onHelp;
  final VoidCallback? onQuality;
  final VoidCallback onLeftClick;
  final VoidCallback onRightClick;
  final VoidCallback onToggleDrag;
  final bool dragArmed;
  final ValueChanged<int> onControlKey;

  const RemoteToolbar({
    super.key,
    required this.displays,
    required this.activeDisplayIndex,
    required this.landscape,
    required this.zoomed,
    required this.keyboardVisible,
    required this.statsVisible,
    required this.supportsFileTransfer,
    required this.onSelectDisplay,
    required this.onToggleOrientation,
    required this.onResetZoom,
    required this.onToggleKeyboard,
    required this.onSendClipboard,
    required this.onToggleStats,
    required this.onUpload,
    required this.onDownload,
    this.onTransfers,
    required this.onDisconnect,
    required this.onHelp,
    this.onQuality,
    required this.onLeftClick,
    required this.onRightClick,
    required this.onToggleDrag,
    required this.dragArmed,
    required this.onControlKey,
  });

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.black54,
      borderRadius: BorderRadius.circular(24),
      child: Wrap(
        alignment: WrapAlignment.end,
        children: [
          IconButton(
            icon: const Icon(Icons.mouse, color: Colors.white),
            tooltip: L10n.t('remote.leftClick'),
            onPressed: onLeftClick,
          ),
          IconButton(
            icon: const Icon(Icons.ads_click, color: Colors.white),
            tooltip: L10n.t('remote.rightClick'),
            onPressed: onRightClick,
          ),
          IconButton(
            icon: Icon(
              Icons.pan_tool,
              color: dragArmed ? Colors.lightBlueAccent : Colors.white,
            ),
            tooltip: L10n.t(dragArmed ? 'remote.cancelDrag' : 'remote.armDrag'),
            onPressed: onToggleDrag,
          ),
          IconButton(
            icon: Icon(
              Icons.keyboard,
              color: keyboardVisible ? Colors.lightBlueAccent : Colors.white,
            ),
            tooltip: L10n.t('remote.keyboard'),
            onPressed: onToggleKeyboard,
          ),
          IconButton(
            icon: const Icon(Icons.zoom_out_map, color: Colors.white),
            tooltip: L10n.t('remote.resetZoom'),
            onPressed: onResetZoom,
          ),
          IconButton(
            icon: const Icon(Icons.help_outline, color: Colors.white),
            tooltip: L10n.t('remote.gestureHelp'),
            onPressed: onHelp,
          ),
          PopupMenuButton<String>(
            icon: const Icon(Icons.more_vert, color: Colors.white),
            tooltip: L10n.t('remote.more'),
            onSelected: (value) async {
              if (value.startsWith('display:')) {
                onSelectDisplay(int.parse(value.split(':').last));
              } else if (value.startsWith('key:')) {
                onControlKey(int.parse(value.split(':').last));
              } else {
                switch (value) {
                  case 'quality':
                    onQuality?.call();
                  case 'rotate':
                    onToggleOrientation();
                  case 'clipboard':
                    await onSendClipboard();
                  case 'stats':
                    onToggleStats();
                  case 'upload':
                    onUpload();
                  case 'download':
                    onDownload();
                  case 'transfers':
                    onTransfers?.call();
                  case 'disconnect':
                    onDisconnect();
                }
              }
            },
            itemBuilder: (context) => [
              for (final entry in const <int, String>{
                0x29: 'Esc',
                0x2b: 'Tab',
                0x28: 'Enter',
                0x2a: '⌫ Backspace',
                0x50: '←',
                0x52: '↑',
                0x4f: '→',
                0x51: '↓',
              }.entries)
                PopupMenuItem(
                  value: 'key:${entry.key}',
                  child: Text(entry.value),
                ),
              const PopupMenuDivider(),
              if (onQuality != null)
                PopupMenuItem(
                  value: 'quality',
                  child: Text(L10n.t('quality.title')),
                ),
              for (var i = 0; i < displays.length; i++)
                CheckedPopupMenuItem(
                  value: 'display:$i',
                  checked: i == activeDisplayIndex,
                  child: Text(L10n.t('remote.display', {'n': i + 1})),
                ),
              PopupMenuItem(
                value: 'rotate',
                child: Text(L10n.t('remote.rotate')),
              ),
              PopupMenuItem(
                value: 'clipboard',
                child: Text(L10n.t('remote.sendClipboard')),
              ),
              CheckedPopupMenuItem(
                value: 'stats',
                checked: statsVisible,
                child: Text(L10n.t('remote.stats')),
              ),
              if (supportsFileTransfer) ...[
                PopupMenuItem(
                  value: 'upload',
                  child: Text(L10n.t('file.upload')),
                ),
                PopupMenuItem(
                  value: 'download',
                  child: Text(L10n.t('file.download')),
                ),
              ],
              if (onTransfers != null)
                PopupMenuItem(
                  value: 'transfers',
                  child: Text(L10n.t('file.transfers')),
                ),
              const PopupMenuDivider(),
              PopupMenuItem(
                value: 'disconnect',
                child: Text(L10n.t('remote.disconnect')),
              ),
            ],
          ),
        ],
      ),
    );
  }
}
