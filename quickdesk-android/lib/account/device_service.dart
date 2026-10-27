import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';

import 'account_service.dart';
import 'device_sort.dart';

/// The event transport is injectable; tests never connect to a real account.
abstract class DeviceEventSocket {
  Stream<dynamic> get messages;
  void send(String message);
  Future<void> close();
}

class _IoDeviceEventSocket implements DeviceEventSocket {
  final WebSocket socket;
  _IoDeviceEventSocket(this.socket);
  @override
  Stream<dynamic> get messages => socket;
  @override
  void send(String message) => socket.add(message);
  @override
  Future<void> close() async {
    await socket.close();
  }
}

typedef DeviceSocketConnector = Future<DeviceEventSocket> Function(String url);

/// Account-scoped cloud data. Secrets are never persisted here or logged.
/// HTTP lists provide initial/offline fallback only; a realtime snapshot and
/// revision-ordered patches are authoritative once the stream is established.
class DeviceService extends ChangeNotifier {
  final AccountService account;
  final DeviceSocketConnector _connect;
  List<Map<String, dynamic>> _devices = [];
  List<Map<String, dynamic>> _favorites = [];
  List<Map<String, dynamic>> _connections = [];
  List<Map<String, dynamic>> get devices =>
      List.unmodifiable(sortedDeviceRows(_devices));
  List<Map<String, dynamic>> get favorites =>
      List.unmodifiable(sortedDeviceRows(_favorites, favorites: true));
  List<Map<String, dynamic>> get connections => List.unmodifiable(_connections);
  bool loading = false;
  Object? error;
  bool syncConnected = false;
  DeviceEventSocket? _socket;
  StreamSubscription<dynamic>? _subscription;
  Timer? _retry;
  int _epoch = -1;
  int _revision = 0;
  int _favoritesRevision = 0;
  int _connectionGeneration = 0;
  int _refreshGeneration = 0;
  int _attempt = 0;
  bool _disposed = false;
  bool _started = false;
  bool _hasSnapshot = false;
  String? _family;

  DeviceService(this.account, {DeviceSocketConnector? connector})
    : _connect =
          connector ??
          ((url) async => _IoDeviceEventSocket(
            await WebSocket.connect(url).timeout(const Duration(seconds: 10)),
          )) {
    account.addListener(_accountChanged);
  }

  bool _current(int epoch) =>
      !_disposed &&
      account.isLoggedIn &&
      account.sessionEpoch == epoch &&
      _epoch == epoch;

  void start() {
    if (_started || _disposed) {
      return;
    }
    final alreadyObserved =
        _epoch == account.sessionEpoch && account.isLoggedIn;
    _started = true;
    _accountChanged();
    if (alreadyObserved) {
      unawaited(refresh());
      unawaited(_open());
    }
  }

  void _accountChanged() {
    if (_disposed) {
      return;
    }
    if (_epoch != account.sessionEpoch || !account.isLoggedIn) {
      _closeTransport();
      ++_refreshGeneration;
      _epoch = account.sessionEpoch;
      _devices = [];
      _favorites = [];
      _connections = [];
      _revision = 0;
      _favoritesRevision = 0;
      _hasSnapshot = false;
      loading = false;
      error = null;
      notifyListeners();
      if (_started && account.isLoggedIn) {
        unawaited(refresh());
        unawaited(_open());
      }
    }
  }

  void stop() {
    _started = false;
    _closeTransport();
  }

  void _closeTransport() {
    ++_connectionGeneration;
    _retry?.cancel();
    _retry = null;
    unawaited(_subscription?.cancel());
    _subscription = null;
    unawaited(_socket?.close());
    _socket = null;
    syncConnected = false;
    _family = null;
  }

  static List<Map<String, dynamic>> _rows(dynamic value) {
    if (value is! List) {
      throw const FormatException('Invalid cloud list');
    }
    return value
        .map(
          (row) => Map<String, dynamic>.unmodifiable(
            Map<String, dynamic>.from(row as Map),
          ),
        )
        .toList();
  }

  Future<List<Map<String, dynamic>>> _list(String path) async {
    final rows = <Map<String, dynamic>>[];
    final cursors = <String>{};
    String? cursor;
    do {
      final suffix = cursor == null
          ? ''
          : '?cursor=${Uri.encodeQueryComponent(cursor)}';
      final response = await account.request('GET', '$path$suffix');
      rows.addAll(_rows(response['items']));
      cursor = response['next_cursor']?.toString();
      if (cursor != null && cursor.isNotEmpty && !cursors.add(cursor)) {
        throw const FormatException('Repeated pagination cursor');
      }
    } while (cursor != null && cursor.isNotEmpty);
    return rows;
  }

  Future<void> refresh() async {
    if (!account.isLoggedIn) {
      return;
    }
    final epoch = account.sessionEpoch;
    final generation = ++_refreshGeneration;
    final revision = _revision;
    final favoritesRevision = _favoritesRevision;
    loading = true;
    error = null;
    notifyListeners();
    try {
      final lists = await Future.wait([
        _list('/v1/me/devices'),
        _list('/v1/me/favorites'),
        _list('/v1/me/connections'),
      ]);
      if (!_current(epoch) || generation != _refreshGeneration) {
        return;
      }
      // Never overwrite realtime presence with a cross-channel HTTP result.
      if ((!_hasSnapshot || !syncConnected) && revision == _revision) {
        _devices = lists[0];
      }
      if (favoritesRevision == _favoritesRevision) {
        _favorites = lists[1];
      }
      _connections = lists[2];
    } catch (e) {
      if (_current(epoch) && generation == _refreshGeneration) {
        error = e;
      }
    } finally {
      if (_current(epoch) && generation == _refreshGeneration) {
        loading = false;
        notifyListeners();
      }
    }
  }

  String _devicePath(String id) => '/v1/me/devices/${Uri.encodeComponent(id)}';
  String _favoritePath(String id) =>
      '/v1/me/favorites/${Uri.encodeComponent(id)}';

  Future<void> bind(String id) async {
    await account.request('POST', '/v1/me/devices', body: {'device_id': id});
    await refresh();
  }

  Future<void> setRemark(String id, String remark) async {
    final epoch = account.sessionEpoch;
    final revision = _revision;
    await account.request('PATCH', _devicePath(id), body: {'remark': remark});
    if (!_current(epoch)) {
      return;
    }
    if (_revision == revision) {
      _patchDevice(id, {'remark': remark});
    }
    notifyListeners();
  }

  Future<void> unbind(String id) async {
    final epoch = account.sessionEpoch;
    final revision = _revision;
    await account.request('DELETE', _devicePath(id));
    if (!_current(epoch) || revision != _revision) {
      return;
    }
    _devices.removeWhere((row) => row['device_id'] == id);
    notifyListeners();
  }

  /// Fetch only the current code. Do not replace newer realtime presence.
  Future<String> accessCode(String id) async {
    final epoch = account.sessionEpoch;
    final response = await account.request('GET', _devicePath(id));
    if (!_current(epoch)) {
      throw StateError('Account changed');
    }
    final code = response['access_code'];
    if (code is! String || code.isEmpty) {
      throw const FormatException('Device has no access code');
    }
    return code;
  }

  Future<void> addFavorite(
    String id, {
    String name = '',
    String password = '',
  }) async {
    await account.request(
      'POST',
      '/v1/me/favorites',
      body: {
        'device_id': id,
        if (name.isNotEmpty) 'device_name': name,
        if (password.isNotEmpty) 'access_password': password,
      },
    );
    await _refreshFavorites();
  }

  Future<void> updateFavorite(
    String id, {
    String? name,
    String? password,
  }) async {
    await account.request(
      'PATCH',
      _favoritePath(id),
      body: {
        if (name != null) 'device_name': name,
        if (password != null) 'access_password': password,
      },
    );
    await _refreshFavorites();
  }

  Future<void> removeFavorite(String id) async {
    final epoch = account.sessionEpoch;
    final revision = _revision;
    await account.request('DELETE', _favoritePath(id));
    if (!_current(epoch) || revision != _revision) {
      return;
    }
    _favorites.removeWhere((row) => row['device_id'] == id);
    notifyListeners();
  }

  Future<void> _refreshFavorites() async {
    final epoch = account.sessionEpoch;
    final revision = _favoritesRevision;
    final rows = await _list('/v1/me/favorites');
    if (!_current(epoch) || _favoritesRevision != revision) {
      return;
    }
    _favorites = rows;
    notifyListeners();
  }

  Future<void> _open() async {
    final epoch = account.sessionEpoch;
    final generation = ++_connectionGeneration;
    bool current() => _current(epoch) && generation == _connectionGeneration;
    try {
      final token = await account.validAccessToken();
      if (!current()) {
        return;
      }
      final base = account.signalingWsUrl.replaceAll(RegExp(r'/+$'), '');
      final socket = await _connect('$base/v1/realtime/events');
      if (!current()) {
        await socket.close();
        return;
      }
      _socket = socket;
      _retry = Timer(const Duration(seconds: 15), () {
        if (current()) {
          _disconnected();
        }
      });
      socket.send(
        jsonEncode({
          'type': 'auth',
          'access_token': token,
          if (_revision > 0) 'since_rev': _revision,
        }),
      );
      _subscription = socket.messages.listen(
        (raw) {
          if (!current()) {
            return;
          }
          try {
            final frame = jsonDecode(raw as String);
            if (frame is Map<String, dynamic>) {
              _frame(frame);
            }
          } catch (_) {
            /* Ignore malformed frames, never print secret payloads. */
          }
        },
        onError: (Object _) {
          if (current()) {
            _disconnected();
          }
        },
        onDone: () {
          if (current()) {
            _disconnected();
          }
        },
      );
    } catch (_) {
      if (current()) {
        _disconnected();
      }
    }
  }

  void _disconnected() {
    _closeTransport();
    if (!_current(_epoch) || !_started) {
      return;
    }
    final exponent = _attempt < 5 ? _attempt++ : 5;
    final seconds = 1 << exponent;
    _retry = Timer(Duration(seconds: seconds), () => unawaited(_open()));
    notifyListeners();
  }

  void _patchDevice(String id, Map<String, dynamic> fields) {
    final index = _devices.indexWhere((row) => row['device_id'] == id);
    final row = <String, dynamic>{
      if (index >= 0) ..._devices[index],
      'device_id': id,
      ...fields,
    };
    if (index < 0) {
      _devices.add(Map.unmodifiable(row));
    } else {
      _devices[index] = Map.unmodifiable(row);
    }
  }

  void _frame(Map<String, dynamic> frame) {
    final type = frame['type'];
    final data = Map<String, dynamic>.from(frame['data'] as Map? ?? {});
    final rev = (frame['server_rev'] as num?)?.toInt() ?? 0;
    if (type == 'auth_ok') {
      _family = data['family_id'] as String?;
      _attempt = 0;
      return;
    }
    if (type == 'bootstrap_done') {
      _retry?.cancel();
      _retry = null;
      if (rev > _revision) {
        _revision = rev;
      }
      syncConnected = true;
      notifyListeners();
      return;
    }
    if (type == 'error') {
      // A REST validation engages the same refresh/retry policy as Qt.
      unawaited(
        account
            .request('GET', '/v1/me')
            .then<void>((_) {}, onError: (Object _) {}),
      );
      _disconnected();
      return;
    }
    if (rev <= _revision || rev <= 0) {
      return;
    }
    if (type == 'snapshot_required') {
      return;
    }
    if (type == 'snapshot') {
      final devices = _rows(data['devices']);
      final favorites = _rows(data['favorites']);
      _devices = devices;
      _favorites = favorites;
      ++_favoritesRevision;
      _hasSnapshot = true;
    } else if (type == 'session.revoked') {
      final family = data['family_id'];
      if (family == null || family == _family || _family == null) {
        unawaited(account.handleSessionRevoked().catchError((Object _) {}));
        return;
      }
    } else if (type is String && type.startsWith('device.')) {
      final id = data['device_id'];
      if (id is! String || id.isEmpty) {
        return;
      }
      if (type == 'device.unbound' || type == 'device.ownership.lost') {
        _devices.removeWhere((row) => row['device_id'] == id);
      } else {
        final fields = Map<String, dynamic>.from(data)..remove('access_code');
        if (type == 'device.access_code.changed') {
          // Code is intentionally absent on the wire. Read it at connect time.
          fields['access_code'] = null;
        }
        if (fields['online'] == false) {
          fields['logged_in'] = false;
        }
        _patchDevice(id, fields);
      }
    } else if (type is String && type.startsWith('favorite.')) {
      ++_favoritesRevision;
      if (type == 'favorite.removed') {
        _favorites.removeWhere((row) => row['device_id'] == data['device_id']);
      } else {
        // Favorite event payloads can omit private fields.
        scheduleMicrotask(
          () => unawaited(_refreshFavorites().catchError((Object _) {})),
        );
      }
    }
    _revision = rev;
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    account.removeListener(_accountChanged);
    stop();
    super.dispose();
  }
}
