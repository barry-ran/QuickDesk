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
  final VoidCallback onDisconnect;

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
    required this.onDisconnect,
  });

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.black54,
      borderRadius: BorderRadius.circular(24),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (displays.length > 1)
            PopupMenuButton<int>(
              icon: const Icon(Icons.desktop_windows, color: Colors.white),
              tooltip: L10n.t('remote.switchDisplay'),
              onSelected: onSelectDisplay,
              itemBuilder: (context) => [
                for (var i = 0; i < displays.length; i++)
                  PopupMenuItem<int>(
                    value: i,
                    child: Row(
                      children: [
                        Icon(
                          i == activeDisplayIndex
                              ? Icons.radio_button_checked
                              : Icons.radio_button_unchecked,
                          size: 18,
                        ),
                        const SizedBox(width: 8),
                        Text('${L10n.t('remote.display', {'n': i + 1})}'
                            '${(displays[i].width ?? 0) > 0 ? '  ${displays[i].width}×${displays[i].height}' : ''}'),
                      ],
                    ),
                  ),
              ],
            ),
          IconButton(
            icon: Icon(
              landscape
                  ? Icons.stay_current_portrait
                  : Icons.stay_current_landscape,
              color: Colors.white,
            ),
            tooltip: L10n.t('remote.rotate'),
            onPressed: onToggleOrientation,
          ),
          if (zoomed)
            IconButton(
              icon: const Icon(Icons.zoom_out_map, color: Colors.white),
              tooltip: L10n.t('remote.resetZoom'),
              onPressed: onResetZoom,
            ),
          IconButton(
            icon: Icon(Icons.keyboard,
                color: keyboardVisible ? Colors.lightBlueAccent : Colors.white),
            tooltip: L10n.t('remote.keyboard'),
            onPressed: onToggleKeyboard,
          ),
          IconButton(
            icon: const Icon(Icons.content_paste_go, color: Colors.white),
            tooltip: L10n.t('remote.sendClipboard'),
            onPressed: () async {
              await onSendClipboard();
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                      content: Text(L10n.t('remote.clipboardSent')),
                      duration: const Duration(seconds: 1)),
                );
              }
            },
          ),
          IconButton(
            icon: Icon(Icons.insights,
                color: statsVisible ? Colors.lightBlueAccent : Colors.white),
            tooltip: L10n.t('remote.stats'),
            onPressed: onToggleStats,
          ),
          if (supportsFileTransfer)
            PopupMenuButton<String>(
              icon: const Icon(Icons.folder_open, color: Colors.white),
              tooltip: L10n.t('file.transfer'),
              onSelected: (v) {
                if (v == 'upload') onUpload();
                if (v == 'download') onDownload();
              },
              itemBuilder: (context) => [
                PopupMenuItem(
                    value: 'upload', child: Text(L10n.t('file.upload'))),
                PopupMenuItem(
                    value: 'download', child: Text(L10n.t('file.download'))),
              ],
            ),
          IconButton(
            icon: const Icon(Icons.close, color: Colors.white),
            tooltip: L10n.t('remote.disconnect'),
            onPressed: onDisconnect,
          ),
        ],
      ),
    );
  }
}
