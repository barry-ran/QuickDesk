/// Relay committed IME text, never intermediate composing candidates.
library;

import 'package:flutter/widgets.dart';

class RemoteTextInputRelay {
  static const sentinel = '\u200b';
  final TextEditingController controller;
  final ValueChanged<String> sendText;
  final VoidCallback sendBackspace;
  bool _resetting = false;
  bool _wasComposing = false;

  RemoteTextInputRelay(this.controller, this.sendText, this.sendBackspace) {
    _reset();
    controller.addListener(_onChanged);
  }

  void _reset() {
    _resetting = true;
    controller.value = const TextEditingValue(
      text: sentinel,
      selection: TextSelection.collapsed(offset: 1),
    );
    _resetting = false;
  }

  void _onChanged() {
    if (_resetting) return;
    final value = controller.value;
    if (value.composing.isValid && !value.composing.isCollapsed) {
      _wasComposing = true;
      return;
    }
    final wasComposing = _wasComposing;
    _wasComposing = false;
    if (value.text.isEmpty) {
      if (!wasComposing) sendBackspace();
      _reset();
      return;
    }
    final text = value.text.startsWith(sentinel)
        ? value.text.substring(1)
        : value.text;
    if (text.isEmpty) return;
    sendText(text);
    _reset();
  }

  void dispose() => controller.removeListener(_onChanged);
}
