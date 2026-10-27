import 'dart:async';
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quickdesk_android/account/account_service.dart';

class MemoryTokens implements AccountTokenStore {
  final values = <String, String>{};
  bool failDelete = false;
  Completer<void>? writeGate;
  Completer<void>? writeStarted;
  @override
  Future<String?> read(String key) async => values[key];
  @override
  Future<void> write(String key, String value) async {
    writeStarted?.complete();
    if (writeGate != null) {
      await writeGate!.future;
    }
    values[key] = value;
  }

  @override
  Future<void> delete(String key) async {
    if (failDelete) {
      throw StateError('storage unavailable');
    }
    values.remove(key);
  }
}

Map<String, dynamic> tokens([
  String access = 'access',
  String refresh = 'refresh',
]) => {
  'access_token': access,
  'refresh_token': refresh,
  'access_expires_at': DateTime.now()
      .add(const Duration(hours: 2))
      .toIso8601String(),
  'refresh_expires_at': DateTime.now()
      .add(const Duration(days: 30))
      .toIso8601String(),
  'user': {'id': 1, 'username': 'tester'},
};
http.Response ok(Object data) => http.Response(jsonEncode(data), 200);
void main() {
  test('password login and register use canonical payload and secure server-scoped persistence', () async {
    final requests = <http.Request>[];
    final store = MemoryTokens();
    final service = AccountService(
      signalingUrl: 'wss://example.test',
      apiKey: 'api',
      tokenStore: store,
      client: MockClient((r) async {
        requests.add(r);
        return ok(tokens());
      }),
    );
    await service.login(' user ', ' password ');
    expect(jsonDecode(requests.single.body), {
      'identifier': 'user',
      'password': ' password ',
    });
    expect(requests.single.headers['X-API-Key'], 'api');
    expect(requests.single.headers.containsKey('Authorization'), false);
    expect(service.isLoggedIn, true);
    expect(service.username, 'tester');
    expect(store.values.values.single, isNot(contains(' password ')));
    await service.register(
      username: 'new',
      password: 'secret',
      phone: '13800138000',
      smsCode: '123456',
    );
    expect(requests.last.url.path, '/v1/auth/register');
    expect(jsonDecode(requests.last.body)['sms_code'], '123456');
    service.dispose();
  });

  test('SMS feature and auth use documented endpoints', () async {
    final requests = <http.Request>[];
    final service = AccountService(
      signalingUrl: 'wss://example.test',
      tokenStore: MemoryTokens(),
      client: MockClient((r) async {
        requests.add(r);
        return ok(
          r.url.path == '/v1/features'
              ? {'sms_enabled': true, 'register_enabled': false}
              : tokens(),
        );
      }),
    );
    await service.fetchFeatures();
    expect(service.smsEnabled, true);
    expect(service.registerEnabled, false);
    await service.sendSmsCode('13800138000', 'login');
    expect(requests.last.url.path, '/v1/verification-codes');
    expect(jsonDecode(requests.last.body)['scene'], 'login');
    await service.loginSms('13800138000', '123456');
    expect(requests.last.url.path, '/v1/auth/sessions:sms');
    service.dispose();
  });

  test('concurrent 401 responses coalesce refresh and retry once', () async {
    var refreshCount = 0;
    final refreshReply = Completer<http.Response>();
    final service = AccountService(
      signalingUrl: 'wss://example.test',
      tokenStore: MemoryTokens(),
      client: MockClient((r) async {
        if (r.url.path == '/v1/auth/sessions') {
          return ok(tokens());
        }
        if (r.url.path == '/v1/auth/tokens:refresh') {
          refreshCount++;
          return refreshReply.future;
        }
        if (r.headers['Authorization'] == 'Bearer access') {
          return http.Response('{"code":"TOKEN_EXPIRED"}', 401);
        }
        return ok({'items': []});
      }),
    );
    await service.login('user', 'password');
    final epoch = service.sessionEpoch;
    final a = service.request('GET', '/v1/me/devices');
    final b = service.request('GET', '/v1/me/favorites');
    await Future<void>.delayed(Duration.zero);
    expect(refreshCount, 1);
    refreshReply.complete(ok(tokens('new', 'new-refresh')));
    await Future.wait([a, b]);
    expect(service.accessToken, 'new');
    expect(service.sessionEpoch, epoch);
    service.dispose();
  });

  test(
    'refresh network failure preserves session; explicit retry recovers',
    () async {
      var unavailable = true;
      final service = AccountService(
        signalingUrl: 'wss://example.test',
        tokenStore: MemoryTokens(),
        client: MockClient((r) async {
          if (r.url.path == '/v1/auth/sessions') {
            return ok(tokens());
          }
          if (r.url.path == '/v1/auth/tokens:refresh') {
            return unavailable
                ? http.Response('{}', 503)
                : ok(tokens('new', 'new-refresh'));
          }
          return r.headers['Authorization'] == 'Bearer access'
              ? http.Response('{}', 401)
              : ok({});
        }),
      );
      await service.login('user', 'password');
      await expectLater(
        service.request('GET', '/v1/me'),
        throwsA(isA<AccountException>()),
      );
      expect(service.isLoggedIn, true);
      unavailable = false;
      await service.request('GET', '/v1/me');
      expect(service.accessToken, 'new');
      service.dispose();
    },
  );

  for (final refreshRejected in [true, false]) {
    test(
      'session cleared for ${refreshRejected ? 'rejected refresh' : 'second 401'}',
      () async {
        final store = MemoryTokens();
        final service = AccountService(
          signalingUrl: 'wss://example.test',
          tokenStore: store,
          client: MockClient((r) async {
            if (r.url.path == '/v1/auth/sessions') {
              return ok(tokens());
            }
            if (r.url.path == '/v1/auth/tokens:refresh' && !refreshRejected) {
              return ok(tokens('new', 'new-refresh'));
            }
            return http.Response('{}', 401);
          }),
        );
        await service.login('user', 'password');
        await expectLater(
          service.request('GET', '/v1/me'),
          throwsA(isA<AccountException>()),
        );
        expect(service.isLoggedIn, false);
        expect(store.values, isEmpty);
        service.dispose();
      },
    );
  }

  test('logout revokes device intent then current session and clears despite failures', () async {
    final paths = <String>[];
    final store = MemoryTokens();
    final service = AccountService(
      signalingUrl: 'wss://example.test',
      tokenStore: store,
      client: MockClient((r) async {
        if (r.url.path == '/v1/auth/sessions') {
          return ok(tokens());
        }
        paths.add(r.url.path);
        return http.Response('{}', 503);
      }),
    );
    await service.login('user', 'password');
    await expectLater(
      service.logout(deviceId: '123'),
      throwsA(isA<AccountException>()),
    );
    expect(paths, ['/v1/me/devices/123/session', '/v1/me/sessions/current']);
    expect(service.isLoggedIn, false);
    expect(store.values, isEmpty);
    service.dispose();
  });

  test(
    'logout cancels pending login and prevents session resurrection',
    () async {
      final reply = Completer<http.Response>();
      final store = MemoryTokens();
      final service = AccountService(
        signalingUrl: 'wss://example.test',
        tokenStore: store,
        client: MockClient((r) => reply.future),
      );
      final login = service.login('user', 'password');
      final assertion = expectLater(login, throwsA(isA<AccountException>()));
      await service.logout();
      reply.complete(ok(tokens()));
      await assertion;
      expect(service.isLoggedIn, false);
      expect(store.values, isEmpty);
      service.dispose();
    },
  );

  test(
    'restores same server only and never forwards redirects or external paths',
    () async {
      final store = MemoryTokens();
      final requests = <http.Request>[];
      final client = MockClient((r) async {
        requests.add(r);
        if (r.url.path == '/v1/auth/sessions') {
          return ok(tokens());
        }
        if (r.url.path == '/v1/me') {
          return ok({'username': 'restored'});
        }
        return ok({'sms_enabled': false});
      });
      final first = AccountService(
        signalingUrl: 'wss://one.test',
        client: client,
        tokenStore: store,
      );
      await first.login('user', 'password');
      first.dispose();
      final second = AccountService(
        signalingUrl: 'wss://two.test',
        client: client,
        tokenStore: store,
      );
      await second.initialize();
      expect(second.isLoggedIn, false);
      expect(requests.last.headers.containsKey('Authorization'), false);
      second.dispose();
      final restored = AccountService(
        signalingUrl: 'wss://one.test',
        client: client,
        tokenStore: store,
      );
      await restored.initialize();
      expect(restored.username, 'restored');
      expect(requests.every((r) => !r.followRedirects), true);
      for (final path in ['https://evil.test/v1/me', '/v1/../../other']) {
        await expectLater(
          restored.request('GET', path),
          throwsA(isA<AccountException>()),
        );
      }
      restored.dispose();
    },
  );

  test(
    'restore failure still loads features and malformed storage is discarded',
    () async {
      final store = MemoryTokens();
      var failProfile = false;
      final client = MockClient((r) async {
        if (r.url.path == '/v1/auth/sessions') {
          return ok(tokens());
        }
        if (r.url.path == '/v1/me' && failProfile) {
          return http.Response('{}', 503);
        }
        return ok({'sms_enabled': true});
      });
      final first = AccountService(
        signalingUrl: 'wss://test.example',
        client: client,
        tokenStore: store,
      );
      await first.login('user', 'password');
      first.dispose();
      failProfile = true;
      final restored = AccountService(
        signalingUrl: 'wss://test.example',
        client: client,
        tokenStore: store,
      );
      await expectLater(
        restored.initialize(),
        throwsA(isA<AccountException>()),
      );
      expect(restored.featuresLoaded, true);
      expect(restored.smsEnabled, true);
      restored.dispose();
      store.values[store.values.keys.single] = jsonEncode({
        'tokens': {'access_token': 123},
        'user': {},
      });
      final malformed = AccountService(
        signalingUrl: 'wss://test.example',
        client: client,
        tokenStore: store,
      );
      await malformed.initialize();
      expect(malformed.isLoggedIn, false);
      expect(store.values, isEmpty);
      malformed.dispose();
    },
  );

  test(
    'logout refreshes expired access and revokes despite local storage failure',
    () async {
      final store = MemoryTokens();
      final paths = <String>[];
      final service = AccountService(
        signalingUrl: 'wss://test.example',
        tokenStore: store,
        client: MockClient((r) async {
          paths.add(r.url.path);
          if (r.url.path == '/v1/auth/sessions') {
            return ok({
              ...tokens(),
              'access_expires_at': '2000-01-01T00:00:00Z',
            });
          }
          if (r.url.path == '/v1/auth/tokens:refresh') {
            return ok(tokens('fresh', 'fresh-refresh'));
          }
          expect(r.headers['Authorization'], 'Bearer fresh');
          return ok({});
        }),
      );
      await service.login('user', 'password');
      store.failDelete = true;
      await expectLater(
        service.logout(),
        throwsA(
          isA<AccountException>().having(
            (e) => e.code,
            'code',
            'SECURE_STORAGE',
          ),
        ),
      );
      expect(paths, [
        '/v1/auth/sessions',
        '/v1/auth/tokens:refresh',
        '/v1/me/sessions/current',
      ]);
      expect(service.isLoggedIn, false);
      service.dispose();
    },
  );

  test(
    'queued logout deletion survives dispose and precedes another restore',
    () async {
      final store = MemoryTokens()
        ..writeGate = Completer<void>()
        ..writeStarted = Completer<void>();
      final client = MockClient(
        (r) async => ok(r.url.path == '/v1/auth/sessions' ? tokens() : {}),
      );
      final service = AccountService(
        signalingUrl: 'wss://race.example',
        client: client,
        tokenStore: store,
      );
      final login = service.login('user', 'password');
      final rejected = expectLater(login, throwsA(isA<AccountException>()));
      await store.writeStarted!.future;
      final clear = service.clearSession();
      service.dispose();
      final next = AccountService(
        signalingUrl: 'wss://race.example',
        client: client,
        tokenStore: store,
      );
      final restored = next.initialize();
      store.writeGate!.complete();
      await Future.wait([clear, rejected, restored]);
      expect(store.values, isEmpty);
      expect(next.isLoggedIn, false);
      next.dispose();
    },
  );

  test('errors never expose response bodies or credentials', () async {
    final service = AccountService(
      signalingUrl: 'wss://example.test',
      tokenStore: MemoryTokens(),
      client: MockClient(
        (r) async => http.Response(
          '{"detail":"secret password", "code":"INVALID_PASSWORD"}',
          400,
        ),
      ),
    );
    try {
      await service.login('user', 'secret');
      fail('expected error');
    } on AccountException catch (error) {
      expect(error.code, 'INVALID_PASSWORD');
      expect(error.toString(), isNot(contains('secret')));
    }
    service.dispose();
  });
}
