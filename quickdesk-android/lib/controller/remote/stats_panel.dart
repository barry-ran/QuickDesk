/// stats_panel.dart - 远程页性能统计叠层
library;

import 'package:flutter/material.dart';

import '../../l10n/app_strings.dart';
import '../video_stats.dart';

class StatsPanel extends StatelessWidget {
  final VideoStatsData? data;

  const StatsPanel({super.key, this.data});

  @override
  Widget build(BuildContext context) {
    final d = data;
    final rows = <(String, String)>[
      (L10n.t('stats.resolution'), d?.resolution ?? '—'),
      (L10n.t('stats.codec'), d?.codec ?? '—'),
      (L10n.t('stats.decoder'), d?.decoder ?? '—'),
      (L10n.t('stats.fps'), d != null ? '${d.fps} fps' : '—'),
      (L10n.t('stats.bitrate'), d != null ? _fmtBitrate(d.bitrateKbps) : '—'),
      (L10n.t('stats.rtt'), d != null ? '${d.rttMs} ms' : '—'),
      (
        L10n.t('stats.jitter'),
        d != null ? '${d.jitterMs.toStringAsFixed(1)} ms' : '—'
      ),
      (L10n.t('stats.packetsLost'), d != null ? '${d.packetsLost}' : '—'),
      (L10n.t('stats.framesDropped'), d != null ? '${d.framesDropped}' : '—'),
      (L10n.t('stats.route'), d?.routeType ?? '—'),
      (L10n.t('stats.protocol'), d?.protocol ?? '—'),
      (L10n.t('stats.localAddr'), d?.localAddr ?? '—'),
      (L10n.t('stats.remoteAddr'), d?.remoteAddr ?? '—'),
    ];

    return Material(
      color: Colors.black.withValues(alpha: 0.72),
      borderRadius: BorderRadius.circular(10),
      child: Container(
        width: 240,
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(L10n.t('stats.title'),
                style: const TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.bold,
                    fontSize: 13)),
            const SizedBox(height: 8),
            for (final (label, value) in rows)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(label,
                        style: const TextStyle(
                            color: Colors.white54, fontSize: 12)),
                    const SizedBox(width: 8),
                    Flexible(
                      child: Text(
                        value,
                        textAlign: TextAlign.right,
                        overflow: TextOverflow.ellipsis,
                        style:
                            const TextStyle(color: Colors.white, fontSize: 12),
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
}
