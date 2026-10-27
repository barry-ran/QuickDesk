import 'dart:async';
import 'dart:convert';

import 'package:crypto/crypto.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;

/// Safe to display: never includes response bodies, credentials or request URLs.
class AccountException implements Exception {
  final int statusCode;
  final String code;
  final String message;
  const AccountException(this.statusCode, this.code, this.message);
  @override
  String toString() => message;
}

abstract class AccountTokenStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
  Future<void> delete(String key);
}

class SecureAccountTokenStore implements AccountTokenStore {
  final FlutterSecureStorage storage;
  const SecureAccountTokenStore({this.storage = const FlutterSecureStorage()});
  @override
  Future<String?> read(String key) => storage.read(key: key);
  @override
  Future<void> write(String key, String value) =>
      storage.write(key: key, value: value);
  @override
  Future<void> delete(String key) => storage.delete(key: key);
}

/// A session belongs to one server configuration. Dispose and create a new
/// instance when changing servers; tokens are never sent to a different origin.
class AccountService extends ChangeNotifier {
  final String signalingUrl;
  final String? apiKey;
  final http.Client _client;
  final AccountTokenStore _store;
  final bool _ownsClient;
  late final String httpBase;
  late final String _storageKey;
  Map<String, dynamic> _tokens = {};
  Map<String, dynamic> _user = {};
  Future<void>? _refreshing;
  Timer? _refreshTimer;
  static final Map<String, Future<void>> _storageWork = {};
  int _generation = 0;
  bool _disposed = false;
  bool _authenticating = false;
  bool _loggingOut = false;
  bool smsEnabled = false;
  bool registerEnabled = false;
  bool featuresLoaded = false;

  AccountService({
    required this.signalingUrl,
    this.apiKey,
    http.Client? client,
    AccountTokenStore? tokenStore,
  }) : _client = client ?? http.Client(),
       _ownsClient = client == null,
       _store = tokenStore ?? const SecureAccountTokenStore() {
    final uri = Uri.parse(signalingUrl.trim());
    if (!['ws', 'wss', 'http', 'https'].contains(uri.scheme) ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasQuery ||
        uri.hasFragment) {
      throw const AccountException(
        0,
        'INVALID_SERVER',
        'Invalid server address.',
      );
    }
    httpBase = uri
        .replace(
          scheme: uri.scheme == 'wss'
              ? 'https'
              : uri.scheme == 'ws'
              ? 'http'
              : uri.scheme,
        )
        .toString()
        .replaceFirst(RegExp(r'/+$'), '');
    _storageKey =
        'account_v1_${sha256.convert(utf8.encode('$httpBase\n${apiKey ?? ''}'))}';
  }

  bool get isLoggedIn => (_tokens['access_token'] as String? ?? '').isNotEmpty;
  String get username => _user['username'] as String? ?? '';
  Map<String, dynamic> get user => Map.unmodifiable(_user);
  int get sessionEpoch => _generation;
  String get signalingWsUrl =>
      Uri.parse(httpBase)
          .replace(scheme: Uri.parse(httpBase).scheme == 'https' ? 'wss' : 'ws')
          .toString();
  Future<void> handleSessionRevoked() => clearSession();
  Future<String> validAccessToken() async {
    final generation = _generation;
    final expiry = DateTime.tryParse(
      _tokens['access_expires_at'] as String? ?? '',
    );
    if (!isLoggedIn ||
        expiry == null ||
        expiry.isBefore(DateTime.now().add(const Duration(minutes: 1)))) {
      await refreshAccessToken();
    }
    _check(generation);
    return accessToken;
  }

  String get accessToken => _tokens['access_token'] as String? ?? '';
  void _notify() {
    if (!_disposed) {
      notifyListeners();
    }
  }

  void _check(int generation) {
    if (_disposed || generation != _generation) {
      throw const AccountException(
        0,
        'CANCELLED',
        'The account operation was cancelled.',
      );
    }
  }

  Future<void> _persist(int generation, {bool clear = false}) {
    final snapshot = jsonEncode({'tokens': _tokens, 'user': _user});
    final work = (_storageWork[_storageKey] ?? Future<void>.value())
        .catchError((Object _) {})
        .then((_) async {
          // Logout deletion remains durable after navigation/disposal. Queue order
          // guarantees a later login writes after this deletion, never before it.
          if (!clear && (_disposed || generation != _generation)) {
            return;
          }
          if (clear) {
            await _store.delete(_storageKey);
          } else {
            await _store.write(_storageKey, snapshot);
          }
        });
    _storageWork[_storageKey] = work;
    return work;
  }

  Future<void> fetchFeatures() async {
    final generation = _generation;
    final data = await _raw('GET', '/v1/features');
    _check(generation);
    smsEnabled = data['sms_enabled'] == true;
    registerEnabled = data['register_enabled'] != false;
    featuresLoaded = true;
    _notify();
  }

  Future<void> initialize() async {
    try {
      await _restoreSession();
    } finally {
      if (!_disposed) {
        await fetchFeatures();
      }
    }
  }

  Future<void> _restoreSession() async {
    final generation = _generation;
    String? saved;
    try {
      await (_storageWork[_storageKey] ?? Future<void>.value()).catchError(
        (Object _) {},
      );
      saved = await _store.read(_storageKey);
    } catch (_) {
      throw const AccountException(
        0,
        'SECURE_STORAGE',
        'Unable to read the saved account securely.',
      );
    }
    _check(generation);
    if (saved != null) {
      try {
        final data = jsonDecode(saved) as Map<String, dynamic>;
        _tokens = Map<String, dynamic>.from(data['tokens'] as Map);
        _user = Map<String, dynamic>.from(data['user'] as Map);
        for (final key in [
          'access_token',
          'refresh_token',
          'access_expires_at',
          'refresh_expires_at',
        ]) {
          if (_tokens[key] != null && _tokens[key] is! String) {
            throw const FormatException();
          }
        }
        if (_user['username'] != null && _user['username'] is! String) {
          throw const FormatException();
        }
      } catch (_) {
        await clearSession();
      }
      if (isLoggedIn ||
          (_tokens['refresh_token'] as String? ?? '').isNotEmpty) {
        try {
          final expiry = DateTime.tryParse(
            _tokens['access_expires_at'] as String? ?? '',
          );
          if (!isLoggedIn ||
              expiry == null ||
              expiry.isBefore(DateTime.now().add(const Duration(minutes: 1)))) {
            await refreshAccessToken();
          }
          final profile = await request('GET', '/v1/me');
          _check(generation);
          _user = Map<String, dynamic>.from(
            profile['user'] is Map ? profile['user'] as Map : profile,
          );
          await _persist(generation);
          _scheduleRefresh();
        } finally {
          _notify();
        }
      }
    }
  }

  Future<void> login(String identifier, String password) => _authenticate(
    '/v1/auth/sessions',
    {'identifier': identifier.trim(), 'password': password},
  );
  Future<void> register({
    required String username,
    required String password,
    String phone = '',
    String email = '',
    String smsCode = '',
  }) => _authenticate('/v1/auth/register', {
    'username': username.trim(),
    'password': password,
    if (phone.isNotEmpty) 'phone': phone.trim(),
    if (email.isNotEmpty) 'email': email.trim(),
    if (smsCode.isNotEmpty) 'sms_code': smsCode.trim(),
  });
  Future<void> loginSms(String phone, String smsCode) => _authenticate(
    '/v1/auth/sessions:sms',
    {'phone': phone.trim(), 'sms_code': smsCode.trim()},
  );
  Future<void> sendSmsCode(String phone, String scene) async {
    await _raw(
      'POST',
      '/v1/verification-codes',
      body: {'phone': phone.trim(), 'scene': scene},
    );
  }

  Future<void> _authenticate(String path, Map<String, dynamic> body) async {
    if (_authenticating || _loggingOut) {
      throw const AccountException(
        0,
        'BUSY',
        'Please wait for the current request.',
      );
    }
    _authenticating = true;
    final generation = ++_generation;
    try {
      final result = await _raw('POST', path, body: body);
      _check(generation);
      if (_loggingOut) {
        throw const AccountException(
          0,
          'CANCELLED',
          'The account operation was cancelled.',
        );
      }
      await _applyTokens(result, generation);
    } finally {
      _authenticating = false;
    }
  }

  Future<void> _applyTokens(Map<String, dynamic> result, int generation) async {
    _check(generation);
    if (result['access_token'] is! String ||
        (result['access_token'] as String).isEmpty ||
        result['refresh_token'] is! String ||
        (result['refresh_token'] as String).isEmpty) {
      throw const AccountException(
        0,
        'INVALID_RESPONSE',
        'Invalid account response.',
      );
    }
    _tokens = {
      for (final key in [
        'access_token',
        'refresh_token',
        'access_expires_at',
        'refresh_expires_at',
      ])
        key: result[key],
    };
    if (result['user'] is Map) {
      _user = Map<String, dynamic>.from(result['user'] as Map);
    }
    try {
      await _persist(generation);
    } catch (_) {
      _check(generation);
      try {
        await clearSession();
      } catch (_) {
        /* Report secure-storage failure below. */
      }
      throw const AccountException(
        0,
        'SECURE_STORAGE',
        'Unable to save the account securely.',
      );
    }
    _check(generation);
    _scheduleRefresh();
    _notify();
  }

  void _scheduleRefresh({bool retry = false}) {
    _refreshTimer?.cancel();
    if (_disposed || !isLoggedIn) {
      return;
    }
    final expiry = DateTime.tryParse(
      _tokens['access_expires_at'] as String? ?? '',
    );
    if (expiry == null && !retry) {
      return;
    }
    var delay = retry
        ? const Duration(minutes: 1)
        : expiry!.difference(DateTime.now()) - const Duration(minutes: 5);
    if (delay < const Duration(seconds: 1)) {
      delay = const Duration(seconds: 1);
    }
    _refreshTimer = Timer(delay, () async {
      try {
        await refreshAccessToken();
      } catch (_) {
        if (!_disposed && isLoggedIn) {
          _scheduleRefresh(retry: true);
        }
      }
    });
  }

  Future<void> refreshAccessToken() {
    return _refreshing ??= _refresh().whenComplete(() {
      _refreshing = null;
    });
  }

  Future<void> _refresh() async {
    final generation = _generation;
    final refresh = _tokens['refresh_token'] as String? ?? '';
    if (refresh.isEmpty) {
      await clearSession();
      throw const AccountException(
        401,
        'SESSION_EXPIRED',
        'Please sign in again.',
      );
    }
    try {
      final result = await _raw(
        'POST',
        '/v1/auth/tokens:refresh',
        body: {'refresh_token': refresh},
      );
      await _applyTokens(result, generation);
    } on AccountException catch (error) {
      if (error.statusCode == 401 && generation == _generation) {
        await clearSession();
      }
      rethrow;
    }
  }

  Future<Map<String, dynamic>> request(
    String method,
    String path, {
    Map<String, dynamic>? body,
  }) async {
    final generation = _generation;
    _check(generation);
    if (!isLoggedIn) {
      throw const AccountException(
        401,
        'SESSION_EXPIRED',
        'Please sign in again.',
      );
    }
    final originalToken = accessToken;
    try {
      final result = await _raw(
        method,
        path,
        body: body,
        bearer: originalToken,
      );
      _check(generation);
      return result;
    } on AccountException catch (error) {
      _check(generation);
      if (error.statusCode != 401) {
        rethrow;
      }
      // A concurrent request may already have rotated this access token.
      if (originalToken == accessToken) {
        await refreshAccessToken();
      }
      _check(generation);
      try {
        final result = await _raw(
          method,
          path,
          body: body,
          bearer: accessToken,
        );
        _check(generation);
        return result;
      } on AccountException catch (retryError) {
        if (retryError.statusCode == 401 && generation == _generation) {
          await clearSession();
        }
        rethrow;
      }
    }
  }

  /// Clear local state even if the server is unreachable. A failed revocation
  /// is reported so the UI does not claim a remote logout succeeded.
  Future<void> logout({String? deviceId}) async {
    if (_loggingOut) {
      return;
    }
    _loggingOut = true;
    var bearer = accessToken;
    try {
      AccountException? failure;
      if (bearer.isNotEmpty) {
        try {
          bearer = await validAccessToken();
        } on AccountException catch (error) {
          failure = error;
        }
      }
      AccountException? storageFailure;
      try {
        await clearSession();
      } on AccountException catch (error) {
        storageFailure = error;
      }
      // Even a Keystore failure must not prevent remote session revocation.
      if (bearer.isNotEmpty) {
        if (deviceId != null && deviceId.isNotEmpty) {
          try {
            await _raw(
              'DELETE',
              '/v1/me/devices/${Uri.encodeComponent(deviceId)}/session',
              bearer: bearer,
            );
          } on AccountException catch (error) {
            if (error.statusCode != 404) {
              failure = error;
            }
          }
        }
        try {
          await _raw('DELETE', '/v1/me/sessions/current', bearer: bearer);
        } on AccountException catch (error) {
          failure = error;
        }
      }
      if (storageFailure != null) {
        throw storageFailure;
      }
      if (failure != null) {
        throw const AccountException(
          0,
          'LOGOUT_OFFLINE',
          'Signed out locally. Server session revocation could not be confirmed.',
        );
      }
    } finally {
      _loggingOut = false;
    }
  }

  Future<void> clearSession() async {
    final generation = ++_generation;
    _tokens = {};
    _user = {};
    _refreshTimer?.cancel();
    _notify();
    try {
      await _persist(generation, clear: true);
    } catch (_) {
      throw const AccountException(
        0,
        'SECURE_STORAGE',
        'Signed out in memory, but saved credentials could not be removed.',
      );
    }
  }

  Future<Map<String, dynamic>> _raw(
    String method,
    String path, {
    Map<String, dynamic>? body,
    String? bearer,
  }) async {
    final target = Uri.parse('$httpBase$path');
    final base = Uri.parse(httpBase);
    if (!path.startsWith('/v1/') ||
        path.contains('://') ||
        Uri.decodeComponent(path).contains('..') ||
        path.contains('\\') ||
        target.origin != base.origin ||
        target.hasFragment ||
        !target.path.startsWith('${base.path}/v1/')) {
      throw const AccountException(
        0,
        'INVALID_PATH',
        'Invalid account request.',
      );
    }
    try {
      final request = http.Request(method, Uri.parse('$httpBase$path'))
        ..followRedirects = false
        ..headers.addAll({
          'Content-Type': 'application/json',
          if (apiKey?.isNotEmpty ?? false) 'X-API-Key': apiKey!,
          if (bearer != null && bearer.isNotEmpty)
            'Authorization': 'Bearer $bearer',
        });
      if (body != null) {
        request.body = jsonEncode(body);
      }
      final response = await _client
          .send(request)
          .then(http.Response.fromStream)
          .timeout(const Duration(seconds: 10));
      Map<String, dynamic> data = {};
      try {
        final decoded = jsonDecode(response.body);
        if (decoded is Map<String, dynamic>) {
          data = decoded;
        }
      } catch (_) {
        /* No raw body in errors. */
      }
      if (response.statusCode < 200 || response.statusCode >= 300) {
        final rawCode = data['code'];
        final code =
            rawCode is String && RegExp(r'^[A-Z0-9_]{1,64}$').hasMatch(rawCode)
            ? rawCode
            : 'REQUEST_FAILED';
        throw AccountException(
          response.statusCode,
          code,
          'Account request failed ($code, ${response.statusCode}).',
        );
      }
      return data;
    } on AccountException {
      rethrow;
    } catch (_) {
      throw const AccountException(
        0,
        'NETWORK_ERROR',
        'Unable to reach the server. Please retry.',
      );
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _refreshTimer?.cancel();
    ++_generation;
    if (_ownsClient) {
      _client.close();
    }
    super.dispose();
  }
}
