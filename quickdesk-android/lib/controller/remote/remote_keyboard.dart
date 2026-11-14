import 'package:flutter/material.dart';
import '../../l10n/app_strings.dart';

/// Single-row accessory shared by portrait and landscape remote sessions.
/// The real text field stays mounted for the platform IME, without a second row.
class RemoteKeyboard extends StatelessWidget {
  final TextEditingController controller;
  final FocusNode focusNode;
  final FocusOnKeyEventCallback onKey;
  final VoidCallback onFocusLost;
  final VoidCallback onDismiss;
  final ValueChanged<int> onControlKey;
  const RemoteKeyboard(
      {super.key,
      required this.controller,
      required this.focusNode,
      required this.onKey,
      required this.onFocusLost,
      required this.onDismiss,
      required this.onControlKey});

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Theme.of(context).colorScheme.surface,
      padding: const EdgeInsets.symmetric(horizontal: 4),
      child: Row(
        children: [
          Expanded(
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
                  for (final entry in const <int, String>{
                    0x29: 'Esc',
                    0x2b: 'Tab',
                    0x28: 'Enter',
                    0x2a: '⌫',
                    0x50: '←',
                    0x52: '↑',
                    0x51: '↓',
                    0x4f: '→',
                  }.entries)
                    TextButton(
                      style: TextButton.styleFrom(
                        minimumSize: const Size(48, 48),
                      ),
                      onPressed: () {
                        final composing = controller.value.composing;
                        if (composing.isValid && !composing.isCollapsed) return;
                        onControlKey(entry.key);
                      },
                      child: Text(entry.value),
                    ),
                ],
              ),
            ),
          ),
          SizedBox(
            width: 1,
            height: 1,
            child: Opacity(
              opacity: 0,
              child: Focus(
                onKeyEvent: onKey,
                onFocusChange: (focused) {
                  if (!focused) onFocusLost();
                },
                child: TextField(
                  controller: controller,
                  focusNode: focusNode,
                  autofocus: true,
                  autocorrect: false,
                  // Disabling suggestions adds Android visible-password flags,
                  // which can force secure keyboards and prevent Chinese IME input.
                  enableSuggestions: true,
                  textInputAction: TextInputAction.done,
                  onSubmitted: (_) => onControlKey(0x28),
                  decoration: const InputDecoration.collapsed(hintText: ''),
                ),
              ),
            ),
          ),
          IconButton(
            tooltip: L10n.t('remote.hideKeyboard'),
            onPressed: onDismiss,
            icon: const Icon(Icons.keyboard_hide),
          ),
        ],
      ),
    );
  }
}
