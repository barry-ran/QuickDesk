// Shared contract: test/fixtures/device-sort.json. Locale-independent ordering.
String _clean(dynamic value) => value is String
    ? value.replaceAll(RegExp(r'^[ \t\r\n]+|[ \t\r\n]+$'), '')
    : '';
String _fold(String value) =>
    value.replaceAllMapped(RegExp('[A-Z]'), (match) => match[0]!.toLowerCase());
int _natural(String a, String b) {
  final chunks = RegExp(r'[0-9]+|[^0-9]+');
  final aa = chunks.allMatches(_fold(a)).map((m) => m[0]!).toList();
  final bb = chunks.allMatches(_fold(b)).map((m) => m[0]!).toList();
  final digits = RegExp(r'^[0-9]+$');
  for (var i = 0; i < aa.length && i < bb.length; i++) {
    final x = aa[i], y = bb[i];
    int result;
    if (digits.hasMatch(x) && digits.hasMatch(y)) {
      final sx = x.replaceFirst(RegExp(r'^0+'), '');
      final sy = y.replaceFirst(RegExp(r'^0+'), '');
      final nx = sx.isEmpty ? '0' : sx;
      final ny = sy.isEmpty ? '0' : sy;
      result = nx.length.compareTo(ny.length);
      if (result == 0) {
        result = nx.compareTo(ny);
      }
      if (result == 0) {
        result = x.length.compareTo(y.length);
      }
    } else {
      result = x.compareTo(y);
    }
    if (result != 0) {
      return result;
    }
  }
  return aa.length.compareTo(bb.length);
}

String deviceSortDisplayName(
  Map<String, dynamic> row, {
  bool favorites = false,
}) {
  for (final value in [
    if (!favorites) row['remark'],
    row['device_name'],
    row['device_id'],
  ]) {
    final name = _clean(value);
    if (name.isNotEmpty) {
      return name;
    }
  }
  return '';
}

/// Sort a copy, never the realtime cache. Favorites have no presence contract.
List<Map<String, dynamic>> sortedDeviceRows(
  Iterable<Map<String, dynamic>> rows, {
  bool favorites = false,
}) {
  return rows.toList()..sort((a, b) {
    if (!favorites) {
      final presence = (b['online'] == true ? 1 : 0).compareTo(
        a['online'] == true ? 1 : 0,
      );
      if (presence != 0) {
        return presence;
      }
    }
    final names = _natural(
      deviceSortDisplayName(a, favorites: favorites),
      deviceSortDisplayName(b, favorites: favorites),
    );
    return names != 0
        ? names
        : _clean(a['device_id']).compareTo(_clean(b['device_id']));
  });
}
