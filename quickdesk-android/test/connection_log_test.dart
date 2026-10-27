import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quickdesk_android/account/account_service.dart';
import 'package:quickdesk_android/account/connection_log.dart';

class _Tokens implements AccountTokenStore {
  @override
  Future<String?> read(String key) async => null;
  @override
  Future<void> write(String key, String value) async {}
  @override
  Future<void> delete(String key) async {}
}

void main() {
  late AccountService account;
  late List<http.Request> logs;
  late int response;
  late Completer<void>? gate;
  setUp(() async {
    logs = [];
    response = 200;
    gate = null;
    account = AccountService(
      signalingUrl: 'wss://server.test/prefix',
      apiKey: 'key',
      tokenStore: _Tokens(),
      client: MockClient((request) async {
        if (request.url.path.endsWith('/auth/sessions')) {
          return http.Response(
            jsonEncode({
              'access_token': 'token',
              'refresh_token': 'refresh',
              'user': {'id': 1, 'username': 'test'},
            }),
            200,
          );
        }
        logs.add(request);
        await gate?.future;
        return http.Response('{}', response);
      }),
    );
    await account.login('user', 'password');
  });
  tearDown(() => account.dispose());

  ConnectionLogAttempt capture({
    Duration elapsed = const Duration(seconds: 42),
  }) => ConnectionLogAttempt.capture(
    account: account,
    signalingUrl: 'wss://server.test/prefix',
    apiKey: 'key',
    deviceId: '123456789',
    elapsed: () => elapsed,
  )!;

  test(
    'one terminal record, connected duration, no credentials or names',
    () async {
      final attempt = capture();
      attempt.connected();
      attempt.connected();
      expect(logs, isEmpty);
      gate = Completer<void>();
      final first = attempt.finish();
      final repeated = attempt.finish(failureCode: 'SESSION_FAILED');
      expect(identical(first, repeated), true);
      gate!.complete();
      expect(await first, true);
      expect(logs, hasLength(1));
      expect(jsonDecode(logs.single.body), {
        'device_id': '123456789',
        'duration': 42,
        'status': 'success',
      });
      expect(logs.single.headers['Authorization'], 'Bearer token');
    },
  );

  test('failed verification has zero duration and allowlisted error', () async {
    await capture().finish(failureCode: 'INVALID_CODE');
    expect(jsonDecode(logs.single.body), {
      'device_id': '123456789',
      'duration': 0,
      'status': 'failed',
      'error_msg': 'INVALID_CODE',
    });
  });

  test('remote failure duration and raw failure privacy', () async {
    final attempt = capture();
    attempt.connected();
    await attempt.finish(failureCode: 'secret token https://private.example');
    expect(jsonDecode(logs.single.body)['error_msg'], 'SESSION_FAILED');
    expect(jsonDecode(logs.single.body)['duration'], 42);
  });

  test('back before connected is a cancelled attempt, never success', () async {
    await capture().finish();
    expect(jsonDecode(logs.single.body)['status'], 'failed');
    expect(jsonDecode(logs.single.body)['error_msg'], 'CANCELLED');
  });

  test('logout and account switch discard old attempt', () async {
    final old = capture();
    await account.clearSession();
    expect(await old.finish(), false);
    await account.login('other', 'password');
    final switched = capture();
    await account.login('third', 'password');
    expect(await switched.finish(), false);
    expect(logs, isEmpty);
  });

  test('in-flight old 401 never retries with a new account token', () async {
    gate = Completer<void>();
    response = 401;
    final attempt = capture();
    attempt.connected();
    final writing = attempt.finish();
    await Future<void>.delayed(Duration.zero);
    await account.login('new-user', 'password');
    gate!.complete();
    expect(await writing, false);
    expect(logs, hasLength(1));
  });

  test('server path and API key mismatch cannot capture old account', () {
    for (final url in ['wss://other.test/prefix', 'wss://server.test/other']) {
      expect(
        ConnectionLogAttempt.capture(
          account: account,
          signalingUrl: url,
          apiKey: 'key',
          deviceId: '123',
        ),
        isNull,
      );
    }
    expect(
      ConnectionLogAttempt.capture(
        account: account,
        signalingUrl: account.signalingUrl,
        apiKey: 'other',
        deviceId: '123',
      ),
      isNull,
    );
    expect(
      ConnectionLogAttempt.capture(
        account: null,
        signalingUrl: account.signalingUrl,
        apiKey: 'key',
        deviceId: '123',
      ),
      isNull,
    );
  });

  test('ambiguous server failure is not replayed', () async {
    response = 500;
    final attempt = capture();
    attempt.connected();
    expect(await attempt.finish(), false);
    expect(await attempt.finish(), false);
    expect(logs, hasLength(1));
  });

  test('negative clock and huge duration are bounded', () async {
    final attempt = capture(elapsed: const Duration(seconds: -1));
    attempt.connected();
    await attempt.finish();
    expect(jsonDecode(logs.single.body)['duration'], 0);
  });
  test('identified transport timeout uses backend timeout outcome', () async {
    await capture().finish(failureCode: 'TIMEOUT');
    expect(jsonDecode(logs.single.body)['status'], 'timeout');
    expect(jsonDecode(logs.single.body)['error_msg'], 'TIMEOUT');
  });
}
