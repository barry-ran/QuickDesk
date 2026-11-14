// Device-only component harness; not imported by the production entrypoint.
// Build with flutter build apk --debug -t test_driver/mobile_input_smoke.dart.
// This does not establish a remote session or simulate one.
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:quickdesk_android/controller/remote/remote_keyboard.dart';
import 'package:quickdesk_android/controller/remote_text_input.dart';

void main() => runApp(const MaterialApp(home: InputSmoke()));

class InputSmoke extends StatefulWidget {
  const InputSmoke({super.key});
  @override
  State<InputSmoke> createState() => _InputSmokeState();
}

class _InputSmokeState extends State<InputSmoke> {
  final controller = TextEditingController();
  final focus = FocusNode();
  late final RemoteTextInputRelay relay;
  bool keyboard = false;
  bool landscape = false;
  String received = '';
  @override
  void initState() {
    super.initState();
    relay = RemoteTextInputRelay(controller, (text) {
      setState(() => received += text);
    }, () => setState(() => received = 'Backspace'));
  }

  @override
  void dispose() {
    relay.dispose();
    controller.dispose();
    focus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('IME component test — no host')),
        body: SafeArea(
            child: Column(children: [
          Expanded(
              child: ColoredBox(
                  color: const Color(0xff17354d),
                  child: SizedBox.expand(
                    child: Center(
                        child: Text('VISIBLE TEST SURFACE\nReceived: $received',
                            textAlign: TextAlign.center,
                            style: const TextStyle(
                                color: Colors.white, fontSize: 18))),
                  ))),
          if (!keyboard)
            Row(children: [
              TextButton(
                  onPressed: () => setState(() => keyboard = true),
                  child: const Text('Keyboard')),
              TextButton(
                  onPressed: () async {
                    landscape = !landscape;
                    await SystemChrome.setPreferredOrientations(landscape
                        ? [DeviceOrientation.landscapeLeft]
                        : [DeviceOrientation.portraitUp]);
                  },
                  child: const Text('Rotate')),
            ]),
          if (keyboard)
            RemoteKeyboard(
                controller: controller,
                focusNode: focus,
                onKey: (node, event) => KeyEventResult.ignored,
                onFocusLost: () {},
                onDismiss: () {
                  focus.unfocus();
                  setState(() => keyboard = false);
                },
                onControlKey: (key) => setState(() => received = 'key: $key')),
        ])),
      );
}
