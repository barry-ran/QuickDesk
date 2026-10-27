import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/controller/remote_text_input.dart';

void main() {
  test(
    'IME candidates stay local; composition-only commit sends exactly once',
    () {
      final controller = TextEditingController();
      final sent = <String>[];
      final relay = RemoteTextInputRelay(controller, sent.add, () {});
      controller.value = const TextEditingValue(
        text: 'ni',
        composing: TextRange(start: 0, end: 2),
      );
      controller.value = const TextEditingValue(
        text: '你',
        composing: TextRange(start: 0, end: 1),
      );
      expect(sent, isEmpty);
      expect(controller.text, '你');
      controller.value = const TextEditingValue(text: '你');
      expect(sent, ['你']);
      expect(controller.text, RemoteTextInputRelay.sentinel);
      controller.value = TextEditingValue.empty;
      expect(sent, ['你']);
      relay.dispose();
      controller.dispose();
    },
  );

  test('paste and emoji commit as intact strings; disposal removes relay', () {
    final controller = TextEditingController();
    final sent = <String>[];
    final relay = RemoteTextInputRelay(controller, sent.add, () {});
    controller.text = 'Hello 👨‍👩‍👧‍👦\n你好';
    expect(sent, ['Hello 👨‍👩‍👧‍👦\n你好']);
    relay.dispose();
    controller.text = 'local only';
    expect(sent, hasLength(1));
    controller.dispose();
  });
  test(
    'soft keyboard deleting sentinel sends one backspace and restores it',
    () {
      final controller = TextEditingController();
      var backspaces = 0;
      final relay = RemoteTextInputRelay(
        controller,
        (_) {},
        () => backspaces++,
      );
      controller.value = TextEditingValue.empty;
      expect(backspaces, 1);
      expect(controller.text, RemoteTextInputRelay.sentinel);
      controller.value = TextEditingValue.empty;
      expect(backspaces, 2);
      relay.dispose();
      controller.dispose();
    },
  );
  test('canceling IME preedit does not delete remote committed text', () {
    final controller = TextEditingController();
    var backspaces = 0;
    final sent = <String>[];
    final relay = RemoteTextInputRelay(
      controller,
      sent.add,
      () => backspaces++,
    );
    controller.value = const TextEditingValue(
      text: 'ni',
      composing: TextRange(start: 0, end: 2),
    );
    controller.value = TextEditingValue.empty;
    expect(backspaces, 0);
    expect(sent, isEmpty);
    expect(controller.text, RemoteTextInputRelay.sentinel);
    relay.dispose();
    controller.dispose();
  });
}
