import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/controller/remote/remote_keyboard.dart';

void main() {
  for (final width in [320.0, 844.0]) {
    testWidgets('compact keyboard at width $width preserves IME and dismiss',
        (tester) async {
      tester.view.physicalSize = Size(width, 390);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final controller = TextEditingController();
      final focus = FocusNode();
      addTearDown(controller.dispose);
      addTearDown(focus.dispose);
      final keys = <int>[];
      var dismissed = false;
      await tester.pumpWidget(MaterialApp(
          home: Scaffold(
              body: Column(children: [
        const Expanded(child: ColoredBox(color: Colors.black)),
        RemoteKeyboard(
            controller: controller,
            focusNode: focus,
            onKey: (node, event) => KeyEventResult.ignored,
            onFocusLost: () {},
            onDismiss: () => dismissed = true,
            onControlKey: keys.add),
      ]))));
      await tester.pump();
      expect(tester.takeException(), isNull);
      expect(tester.getSize(find.byType(RemoteKeyboard)).height,
          lessThanOrEqualTo(48));
      expect(focus.hasFocus, isTrue);
      final field = tester.widget<TextField>(find.byType(TextField));
      expect(field.enableSuggestions, isTrue);
      expect(field.obscureText, isFalse);
      controller.value = const TextEditingValue(
          text: 'ni', composing: TextRange(start: 0, end: 2));
      await tester.tap(find.text('Esc'));
      expect(keys, isEmpty);
      controller.value = const TextEditingValue(text: '你');
      await tester.tap(find.text('Esc'));
      expect(keys, [0x29]);
      await tester.tap(find.byIcon(Icons.keyboard_hide));
      expect(dismissed, isTrue);
      expect(tester.takeException(), isNull);
    });
  }
}
