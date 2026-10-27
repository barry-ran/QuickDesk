import 'package:shared_preferences/shared_preferences.dart';

/// Stable reorder, preserving RTX/FEC and every fallback codec.
List<T> preferCodec<T>(
  List<T> codecs,
  String preferred,
  String? Function(T) mime,
) {
  final target = 'video/${preferred.toLowerCase()}';
  return [
    ...codecs.where((c) => mime(c)?.toLowerCase() == target),
    ...codecs.where((c) => mime(c)?.toLowerCase() != target),
  ];
}

class QualitySettings {
  static const codecs = ['', 'H264', 'VP8', 'VP9', 'AV1'];
  static const _codecKey = 'preferred_video_codec';
  static Future<String> loadCodec() async {
    try {
      final value =
          (await SharedPreferences.getInstance()).getString(_codecKey) ?? '';
      return codecs.contains(value) ? value : '';
    } catch (_) {
      // Optional preferences must not prevent a remote connection.
      return '';
    }
  }

  static Future<void> saveCodec(String value) async {
    if (!codecs.contains(value)) throw ArgumentError.value(value);
    await (await SharedPreferences.getInstance()).setString(_codecKey, value);
  }
}
