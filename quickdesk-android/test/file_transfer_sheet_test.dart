import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/controller/remote/transfer_sheet.dart';
import 'package:quickdesk_android/protocol/file_transfer.dart';

import 'file_transfer_test.dart' show FakeFileChannel;

void main() {
  testWidgets('queue sheet exposes active cancellation and may close independently', (tester) async {
    final channel = FakeFileChannel();
    final manager = FileTransferManager.withChannelFactory((_) async => channel);
    addTearDown(manager.dispose);
    final id = manager.enqueueDownload('unused-until-metadata');
    await tester.pumpWidget(MaterialApp(home: Scaffold(body: TransferSheet(manager: manager))));
    expect(find.byKey(ValueKey(id)), findsOneWidget);
    expect(find.byIcon(Icons.cancel_outlined), findsOneWidget);
    await tester.tap(find.byIcon(Icons.cancel_outlined));
    await tester.pump();
    expect(manager.tasks.single.status, FileTransferStatus.cancelled);
    expect(find.byIcon(Icons.cancel_outlined), findsNothing);
    expect(find.byIcon(Icons.close), findsOneWidget);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
