/// transfer_sheet.dart - 文件传输进度底部弹层
library;

import 'dart:async';

import 'package:flutter/material.dart';

import '../../l10n/app_strings.dart';
import '../../protocol/file_transfer.dart';

class TransferSheet extends StatefulWidget {
  final String title;
  final Stream<FileTransferProgress> stream;

  const TransferSheet({super.key, required this.title, required this.stream});

  @override
  State<TransferSheet> createState() => _TransferSheetState();
}

class _TransferSheetState extends State<TransferSheet> {
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
            if (p != null)
              Text(p.filename,
                  style: const TextStyle(fontWeight: FontWeight.w500)),
            const SizedBox(height: 8),
            if (error != null)
              Text('${L10n.t('file.failed')}: $error',
                  style: TextStyle(color: Theme.of(context).colorScheme.error))
            else ...[
              LinearProgressIndicator(
                  value: p != null && p.totalBytes > 0 ? p.fraction : null),
              const SizedBox(height: 8),
              Text(_progressText(p, done)),
            ],
            const SizedBox(height: 16),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton(
                onPressed: finished ? () => Navigator.of(context).pop() : null,
                child:
                    Text(finished ? L10n.t('common.back') : L10n.t('common.cancel')),
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
    final total =
        p.totalBytes > 0 ? (p.totalBytes / 1024).toStringAsFixed(0) : '?';
    return '$kb KB / $total KB';
  }

  @override
  void dispose() {
    _sub?.cancel();
    super.dispose();
  }
}
