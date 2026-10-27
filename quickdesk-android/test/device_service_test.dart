import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/account/account_service.dart';
import 'package:quickdesk_android/account/device_service.dart';

class FakeAccount extends AccountService {
  FakeAccount() : super(signalingUrl: 'wss://example.invalid');
  void announceRestored() => notifyListeners();
  bool loggedIn = true;
  int epoch = 1;
  final calls = <(String, String, Map<String, dynamic>?)>[];
  Future<Map<String, dynamic>> Function(String, String, Map<String, dynamic>?)?
  handler;
  @override
  bool get isLoggedIn => loggedIn;
  @override
  int get sessionEpoch => epoch;
  @override
  Future<String> validAccessToken() async => 'fixture-token';
  @override
  Future<Map<String, dynamic>> request(
    String method,
    String path, {
    Map<String, dynamic>? body,
  }) async {
    calls.add((method, path, body));
    return handler == null
        ? {'items': <dynamic>[]}
        : handler!(method, path, body);
  }

  @override
  Future<void> handleSessionRevoked() async {
    loggedIn = false;
    epoch++;
    notifyListeners();
  }
}

class FakeSocket implements DeviceEventSocket {
  final controller = StreamController<dynamic>();
  final sent = <Map<String, dynamic>>[];
  @override
  Stream<dynamic> get messages => controller.stream;
  @override
  void send(String message) =>
      sent.add(jsonDecode(message) as Map<String, dynamic>);
  @override
  Future<void> close() async {}
  void frame(String type, int revision, Map<String, dynamic> data) {
    controller.add(
      jsonEncode({'type': type, 'server_rev': revision, 'data': data}),
    );
  }
}

Future<void> settle() async {
  for (var i = 0; i < 8; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}

void main() {
  late FakeAccount account;
  late FakeSocket socket;
  late DeviceService devices;
  setUp(() {
    account = FakeAccount();
    socket = FakeSocket();
    devices = DeviceService(
      account,
      connector: (url) async {
        expect(url, 'wss://example.invalid/v1/realtime/events');
        return socket;
      },
    );
  });
  tearDown(() async {
    devices.dispose();
    account.dispose();
    await socket.controller.close();
  });

  test(
    'list pagination collects all pages and encodes opaque cursor',
    () async {
      account.handler = (_, path, _) async {
        if (path == '/v1/me/devices') {
          return {
            'items': [
              {'device_id': 'a'},
            ],
            'next_cursor': 'next/page',
          };
        }
        if (path == '/v1/me/devices?cursor=next%2Fpage') {
          return {
            'items': [
              {'device_id': 'b'},
            ],
            'next_cursor': null,
          };
        }
        return {'items': []};
      };
      devices.start();
      await settle();
      expect(devices.devices.map((row) => row['device_id']), ['a', 'b']);
      expect(devices.error, isNull);
    },
  );

  test('repeated pagination cursor fails safely instead of looping', () async {
    account.handler = (_, path, _) async => path.startsWith('/v1/me/devices')
        ? {'items': [], 'next_cursor': 'same'}
        : {'items': []};
    devices.start();
    await settle();
    expect(devices.error, isA<FormatException>());
    expect(
      account.calls.where((call) => call.$2.startsWith('/v1/me/devices')),
      hasLength(2),
    );
    expect(devices.loading, false);
  });

  test(
    'a restored account observed before start still starts data sync',
    () async {
      account.announceRestored();
      expect(account.calls, isEmpty);
      devices.start();
      await settle();
      expect(
        account.calls.where((call) => call.$2 == '/v1/me/devices'),
        hasLength(1),
      );
      expect(socket.sent, hasLength(1));
    },
  );

  test(
    'first-frame auth, snapshot replacement and ordered presence patches',
    () async {
      devices.start();
      await settle();
      expect(socket.sent.single['access_token'], 'fixture-token');
      socket.frame('auth_ok', 10, {'family_id': 'family-a'});
      socket.frame('snapshot', 10, {
        'devices': [
          {'device_id': 'a', 'online': true, 'logged_in': true},
        ],
        'favorites': [],
      });
      socket.frame('bootstrap_done', 10, {});
      socket.frame('device.online.changed', 11, {
        'device_id': 'a',
        'online': false,
      });
      socket.frame('device.online.changed', 9, {
        'device_id': 'a',
        'online': true,
      });
      await settle();
      expect(devices.devices.single['online'], false);
      expect(devices.devices.single['logged_in'], false);
      expect(devices.syncConnected, true);
      socket.frame('snapshot', 12, {'devices': [], 'favorites': []});
      await settle();
      expect(devices.devices, isEmpty);
    },
  );

  test('HTTP response cannot replace a newer realtime snapshot', () async {
    final pending = Completer<Map<String, dynamic>>();
    account.handler = (_, path, _) async =>
        path == '/v1/me/devices' ? pending.future : {'items': []};
    devices.start();
    await settle();
    socket.frame('snapshot', 3, {
      'devices': [
        {'device_id': 'new', 'online': true},
      ],
      'favorites': [],
    });
    await settle();
    pending.complete({
      'items': [
        {'device_id': 'old', 'online': false},
      ],
    });
    await settle();
    expect(devices.devices.single['device_id'], 'new');
  });

  test('logout clears caches and ignores in-flight requests', () async {
    devices.start();
    await settle();
    socket.frame('snapshot', 1, {
      'devices': [
        {'device_id': 'a'},
      ],
      'favorites': [],
    });
    await settle();
    final pending = Completer<Map<String, dynamic>>();
    account.handler = (_, path, _) async =>
        path == '/v1/me/devices' ? pending.future : {'items': []};
    final refresh = devices.refresh();
    await account.handleSessionRevoked();
    pending.complete({
      'items': [
        {'device_id': 'private'},
      ],
    });
    await refresh;
    expect(devices.devices, isEmpty);
    expect(devices.favorites, isEmpty);
    expect(devices.loading, false);
  });

  test('only matching family revocation signs out', () async {
    devices.start();
    await settle();
    socket.frame('auth_ok', 1, {'family_id': 'family-a'});
    socket.frame('session.revoked', 2, {'family_id': 'family-b'});
    await settle();
    expect(account.loggedIn, true);
    socket.frame('session.revoked', 3, {'family_id': 'family-a'});
    await settle();
    expect(account.loggedIn, false);
    expect(devices.devices, isEmpty);
  });

  test('failure preserves data and retry clears error', () async {
    devices.start();
    await settle();
    account.handler = (_, _, _) async =>
        throw const AccountException(503, 'UNAVAILABLE', 'Try again');
    await devices.refresh();
    expect(devices.error, isA<AccountException>());
    expect(devices.loading, false);
    account.handler = (_, _, _) async => {'items': []};
    await devices.refresh();
    expect(devices.error, isNull);
  });

  test(
    'manual refresh restores HTTP fallback while realtime is disconnected',
    () async {
      devices.start();
      await settle();
      socket.frame('snapshot', 1, {
        'devices': [
          {'device_id': 'old'},
        ],
        'favorites': [],
      });
      socket.frame('bootstrap_done', 1, {});
      await settle();
      devices.stop();
      account.handler = (_, path, _) async => {
        'items': path == '/v1/me/devices'
            ? [
                {'device_id': 'fresh'},
              ]
            : [],
      };
      await devices.refresh();
      expect(devices.devices.single['device_id'], 'fresh');
    },
  );

  test('late unbind response cannot erase a newer realtime rebind', () async {
    devices.start();
    await settle();
    final pending = Completer<Map<String, dynamic>>();
    account.handler = (method, _, _) async =>
        method == 'DELETE' ? pending.future : {'items': []};
    final removal = devices.unbind('a');
    socket.frame('device.bound', 1, {'device_id': 'a', 'online': true});
    await settle();
    pending.complete({});
    await removal;
    expect(devices.devices.single['device_id'], 'a');
  });

  test('presence events do not discard a pending favorites fetch', () async {
    devices.start();
    await settle();
    final pending = Completer<Map<String, dynamic>>();
    account.handler = (method, path, _) async =>
        path == '/v1/me/favorites' && method == 'GET'
        ? pending.future
        : {'items': []};
    final addition = devices.addFavorite('a');
    await settle();
    socket.frame('device.online.changed', 1, {
      'device_id': 'a',
      'online': true,
    });
    await settle();
    pending.complete({
      'items': [
        {'device_id': 'favorite'},
      ],
    });
    await addition;
    expect(devices.favorites.single['device_id'], 'favorite');
  });

  test(
    'device and favorite actions use exact API payloads and escaped IDs',
    () async {
      devices.start();
      await settle();
      account.handler = (_, path, _) async => path == '/v1/me/devices/a%2Fb'
          ? {'access_code': '123456'}
          : {'items': []};
      await devices.setRemark('a/b', 'Office');
      expect(account.calls.last, (
        'PATCH',
        '/v1/me/devices/a%2Fb',
        {'remark': 'Office'},
      ));
      expect(await devices.accessCode('a/b'), '123456');
      await devices.addFavorite('a', name: 'Home', password: 'secret-fixture');
      expect(
        account.calls.any(
          (call) =>
              call.$1 == 'POST' &&
              call.$2 == '/v1/me/favorites' &&
              call.$3?['access_password'] == 'secret-fixture',
        ),
        true,
      );
      await devices.updateFavorite('a', name: '', password: '');
      expect(
        account.calls.any(
          (call) => call.$1 == 'PATCH' && call.$3?['access_password'] == '',
        ),
        true,
      );
      await devices.removeFavorite('a');
      expect(account.calls.last.$1, 'DELETE');
      await devices.unbind('a/b');
      expect(account.calls.last.$2, '/v1/me/devices/a%2Fb');
    },
  );
}
