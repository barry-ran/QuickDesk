import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/l10n/app_strings.dart';
import 'package:quickdesk_android/controller/remote/remote_toolbar.dart';

void main() {
  testWidgets('compact toolbar fits portrait and exposes help and controls', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    var help = 0;
    var fit = 0;
    int? key;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: RemoteToolbar(
            displays: const [],
            activeDisplayIndex: 0,
            landscape: false,
            zoomed: false,
            keyboardVisible: false,
            statsVisible: false,
            supportsFileTransfer: true,
            onSelectDisplay: (_) {},
            onToggleOrientation: () {},
            onResetZoom: () => fit++,
            onToggleKeyboard: () {},
            onSendClipboard: () async {},
            onToggleStats: () {},
            onUpload: () {},
            onDownload: () {},
            onDisconnect: () {},
            onHelp: () => help++,
            onLeftClick: () {},
            onRightClick: () {},
            onToggleDrag: () {},
            dragArmed: false,
            onControlKey: (value) => key = value,
          ),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    final keyboardX = tester.getTopLeft(find.byIcon(Icons.keyboard)).dx;
    final leftX = tester.getTopLeft(find.byIcon(Icons.mouse)).dx;
    final rightX = tester.getTopLeft(find.byIcon(Icons.ads_click)).dx;
    final dragX = tester.getTopLeft(find.byIcon(Icons.pan_tool)).dx;
    expect(keyboardX, lessThan(leftX));
    expect(leftX, lessThan(rightX));
    expect(rightX, lessThan(dragX));
    expect(tester.getBottomRight(find.byIcon(Icons.more_vert)).dx,
        lessThanOrEqualTo(320));
    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text(L10n.t('remote.gestureHelp')));
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.zoom_out_map));
    expect(help, 1);
    expect(fit, 1);
    await tester.ensureVisible(find.byIcon(Icons.more_vert));
    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Esc'));
    await tester.pumpAndSettle();
    expect(key, 0x29);
    expect(tester.takeException(), isNull);
  });
}
