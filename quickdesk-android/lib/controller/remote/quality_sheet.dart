import 'package:flutter/material.dart';

import '../../core/quality_settings.dart';
import '../../l10n/app_strings.dart';

/// Qt has no per-setting capability acknowledgements. These are requests, not
/// promises of a resulting resolution, FPS or bandwidth.
class QualitySheet extends StatefulWidget {
  final bool canRequest;
  final bool muted;
  final ValueChanged<bool> onMute;
  final ValueChanged<int> onFramerate;
  final ValueChanged<int> onBitrate;
  final void Function(int width, int height) onResolution;
  const QualitySheet({
    super.key,
    required this.canRequest,
    required this.muted,
    required this.onMute,
    required this.onFramerate,
    required this.onBitrate,
    required this.onResolution,
  });

  @override
  State<QualitySheet> createState() => _QualitySheetState();
}

class _QualitySheetState extends State<QualitySheet> {
  late bool _muted = widget.muted;
  String _codec = '';
  bool _saving = false;
  bool _loaded = false;
  @override
  void initState() {
    super.initState();
    QualitySettings.loadCodec().then((value) {
      if (mounted) {
        setState(() {
          _codec = value;
          _loaded = true;
        });
      }
    });
  }

  @override
  Widget build(BuildContext context) => SafeArea(
    child: SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            L10n.t('quality.title'),
            style: Theme.of(context).textTheme.titleLarge,
          ),
          Text(
            L10n.t(
              widget.canRequest ? 'quality.requests' : 'quality.unavailable',
            ),
          ),
          SwitchListTile(
            title: Text(L10n.t('quality.mute')),
            value: _muted,
            onChanged: (value) {
              setState(() => _muted = value);
              widget.onMute(value);
            },
          ),
          DropdownButtonFormField<int>(
            decoration: InputDecoration(labelText: L10n.t('quality.fps')),
            items: [
              for (final fps in [5, 15, 30, 60])
                DropdownMenuItem(value: fps, child: Text('$fps FPS')),
            ],
            onChanged: widget.canRequest
                ? (value) {
                    if (value != null) widget.onFramerate(value);
                  }
                : null,
          ),
          DropdownButtonFormField<int>(
            decoration: InputDecoration(labelText: L10n.t('quality.bitrate')),
            items: [
              for (final mb in [0, 2, 5, 10, 50, 100])
                DropdownMenuItem(
                  value: mb * 1024 * 1024,
                  child: Text(
                    mb == 0 ? L10n.t('quality.auto') : '$mb × 1024² bit/s',
                  ),
                ),
            ],
            onChanged: widget.canRequest
                ? (value) {
                    if (value != null) widget.onBitrate(value);
                  }
                : null,
          ),
          DropdownButtonFormField<String>(
            decoration: InputDecoration(
              labelText: L10n.t('quality.resolution'),
            ),
            items: [
              for (final res in [
                '1280x720',
                '1600x900',
                '1920x1080',
                '2560x1440',
                '3840x2160',
              ])
                DropdownMenuItem(value: res, child: Text(res)),
            ],
            onChanged: widget.canRequest
                ? (value) {
                    if (value != null) {
                      final size = value.split('x');
                      widget.onResolution(
                        int.parse(size[0]),
                        int.parse(size[1]),
                      );
                    }
                  }
                : null,
          ),
          const SizedBox(height: 16),
          DropdownButtonFormField<String>(
            key: ValueKey(_codec),
            initialValue: _codec,
            decoration: InputDecoration(labelText: L10n.t('quality.codec')),
            items: [
              for (final codec in QualitySettings.codecs)
                DropdownMenuItem(
                  value: codec,
                  child: Text(codec.isEmpty ? L10n.t('quality.auto') : codec),
                ),
            ],
            onChanged: (_saving || !_loaded)
                ? null
                : (value) async {
                    if (value == null) return;
                    setState(() => _saving = true);
                    try {
                      await QualitySettings.saveCodec(value);
                      if (mounted) setState(() => _codec = value);
                    } catch (_) {
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text(L10n.t('quality.saveFailed'))),
                        );
                      }
                    } finally {
                      if (mounted) setState(() => _saving = false);
                    }
                  },
          ),
          Text(L10n.t('quality.codecHint')),
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: Text(L10n.t('quality.close')),
          ),
        ],
      ),
    ),
  );
}

/// Compatibility heuristic only: Qt advertises action tokens, not quality
/// tokens. Empty/unknown capability sets (including current Android host) are
/// conservatively excluded; the UI still labels enabled controls as requests.
bool canRequestDesktopQuality(String capabilities) => capabilities
    .split(RegExp(r'\s+'))
    .any(
      const {
        'sendAttentionSequenceAction',
        'lockWorkstationAction',
        'fileTransfer',
        'privacyScreen',
        'virtualDisplay',
      }.contains,
    );
