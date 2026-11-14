Mobile IME device smoke test

This entrypoint uses the production RemoteKeyboard and RemoteTextInputRelay.
It is a component harness, not a remote connection. The visible label explicitly
states "no host". Do not distribute this harness as the production application.

Build:
flutter build apk --debug --target-platform android-x64 -t test_driver/mobile_input_smoke.dart

Run on a disposable emulator. Tap Keyboard, type hello, verify Received: hello,
then dismiss. Tap Rotate and repeat in landscape. The colored test surface must
remain visible and the accessory must use a single 48 logical-pixel row. Android
input_method diagnostics should report inputType=0x1, not password variations.

The Chinese composition contract is covered by remote_text_input_test.dart and
remote_keyboard_test.dart. The AOSP emulator keyboard used here is Latin-only;
this smoke test does not claim Chinese IME UI verification.

Restore a normal APK built with -t lib/main.dart after this test. Both entrypoints
use the same app id, so installing the normal APK replaces the harness.
