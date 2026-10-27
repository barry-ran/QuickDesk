import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/account/device_sort.dart';

void main() {
  final fixtures = jsonDecode(
    File('../test/fixtures/device-sort.json').readAsStringSync(),
  ) as Map<String, dynamic>;
  for (final dynamic value in fixtures['cases'] as List) {
    final fixture = value as Map<String, dynamic>;
    test(fixture['name'] as String, () {
      final rows = (fixture['rows'] as List).cast<Map<String, dynamic>>();
      final original = jsonEncode(rows);
      for (final input in [
        rows,
        rows.reversed.toList(),
        [...rows.skip(1), rows.first],
      ]) {
        final result = sortedDeviceRows(
          input,
          favorites: fixture['favorites'] as bool,
        );
        expect(
          result.map((row) => row['device_id']).toList(),
          fixture['expected'],
        );
        expect(identical(result, input), isFalse);
      }
      expect(jsonEncode(rows), original);
    });
  }
  test('presence and remark updates reorder without mutating input', () {
    final rows = <Map<String, dynamic>>[
      {'device_id': '1', 'device_name': 'B', 'online': false},
      {'device_id': '2', 'device_name': 'A', 'online': true},
    ];
    expect(sortedDeviceRows(rows).first['device_id'], '2');
    rows[0]['online'] = true;
    rows[0]['remark'] = '0';
    expect(sortedDeviceRows(rows).first['device_id'], '1');
    rows[0]['online'] = false;
    expect(sortedDeviceRows(rows).first['device_id'], '2');
  });
}
