/// host_input.dart - 被控端收到的输入事件模型
///
/// HostSession 从 client 的 'event' DataChannel 解出 protobuf 输入事件后，
/// 归一化为 HostInputEvent 交给平台注入层（AccessibilityService / Shizuku）。
///
/// 坐标说明：mouse.x/y 是相对被控屏幕的**绝对像素坐标**（client 已按
/// VideoLayout 的 width/height 做过映射），注入层可直接使用。
library;

enum HostInputType { mouse, key, text }

class HostInputEvent {
  final HostInputType type;

  // mouse
  final int? x;
  final int? y;
  final int? button; // MouseButton.value：1=left 2=middle 3=right
  final bool? buttonDown;
  final double? wheelDeltaX;
  final double? wheelDeltaY;

  // key
  final int? usbKeycode; // USB HID usage（0x07 页）
  final bool? pressed;

  // text
  final String? text;

  const HostInputEvent._({
    required this.type,
    this.x,
    this.y,
    this.button,
    this.buttonDown,
    this.wheelDeltaX,
    this.wheelDeltaY,
    this.usbKeycode,
    this.pressed,
    this.text,
  });

  factory HostInputEvent.mouse({
    int? x,
    int? y,
    int? button,
    bool? buttonDown,
    double? wheelDeltaX,
    double? wheelDeltaY,
  }) =>
      HostInputEvent._(
        type: HostInputType.mouse,
        x: x,
        y: y,
        button: button,
        buttonDown: buttonDown,
        wheelDeltaX: wheelDeltaX,
        wheelDeltaY: wheelDeltaY,
      );

  factory HostInputEvent.key({required int usbKeycode, required bool pressed}) =>
      HostInputEvent._(
        type: HostInputType.key,
        usbKeycode: usbKeycode,
        pressed: pressed,
      );

  factory HostInputEvent.text(String text) =>
      HostInputEvent._(type: HostInputType.text, text: text);

  @override
  String toString() {
    switch (type) {
      case HostInputType.mouse:
        return 'Mouse(x=$x,y=$y,btn=$button,down=$buttonDown,wheel=$wheelDeltaX/$wheelDeltaY)';
      case HostInputType.key:
        return 'Key(usb=0x${usbKeycode?.toRadixString(16)},pressed=$pressed)';
      case HostInputType.text:
        return 'Text("$text")';
    }
  }
}
