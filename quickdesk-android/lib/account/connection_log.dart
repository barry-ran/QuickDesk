import 'account_service.dart';

/// A best-effort, account/session-scoped final record for one connection attempt.
/// The existing API has no idempotency key. Never replay an ambiguous failed
/// POST: it may already have committed. Authentication's rejected 401 retry is
/// handled by AccountService; no credentials or raw errors enter this payload.
class ConnectionLogAttempt {
  final AccountService _account;
  final int _epoch;
  final String deviceId;
  final Stopwatch _duration = Stopwatch();
  final Duration Function()? _elapsed;
  bool _connected = false;
  Future<bool>? _finished;

  ConnectionLogAttempt._(this._account, this.deviceId, this._elapsed)
    : _epoch = _account.sessionEpoch;

  static ConnectionLogAttempt? capture({
    required AccountService? account,
    required String signalingUrl,
    required String? apiKey,
    required String deviceId,
    Duration Function()? elapsed,
  }) {
    if (account == null || !account.isLoggedIn || deviceId.isEmpty) {
      return null;
    }
    // Match the full configuration, not just hostname. In particular never
    // send an old account token after editing the server form.
    final uri = Uri.tryParse(signalingUrl.trim());
    if (uri == null) {
      return null;
    }
    final base = uri
        .replace(
          scheme: switch (uri.scheme) {
            'ws' => 'http',
            'wss' => 'https',
            _ => uri.scheme,
          },
        )
        .toString()
        .replaceFirst(RegExp(r'/+$'), '');
    if (base != account.httpBase || (apiKey ?? '') != (account.apiKey ?? '')) {
      return null;
    }
    return ConnectionLogAttempt._(account, deviceId, elapsed);
  }

  void connected() {
    if (_finished != null || _connected) {
      return;
    }
    _connected = true;
    _duration.start();
  }

  static const _safeFailures = {
    'DEVICE_NOT_FOUND',
    'HOST_OFFLINE',
    'INVALID_CODE',
    'TOO_MANY_ATTEMPTS',
    'FORBIDDEN',
    'NETWORK_ERROR',
    'TIMEOUT',
    'SESSION_FAILED',
    'CANCELLED',
  };

  /// Returns false when skipped or unconfirmed. Repeated terminal callbacks
  /// share one future and cannot create duplicate rows.
  Future<bool> finish({String? failureCode}) =>
      _finished ??= _write(failureCode);

  Future<bool> _write(String? failureCode) async {
    _duration.stop();
    if (!_account.isLoggedIn || _account.sessionEpoch != _epoch) {
      return false;
    }
    final failure = failureCode == null
        ? (_connected ? null : 'CANCELLED')
        : _safeFailures.contains(failureCode)
        ? failureCode
        : 'SESSION_FAILED';
    final seconds = _connected
        ? (_elapsed?.call() ?? _duration.elapsed).inSeconds.clamp(0, 2147483647)
        : 0;
    try {
      await _account.request(
        'POST',
        '/v1/me/connections',
        body: {
          'device_id': deviceId,
          'duration': seconds,
          'status': failure == null
              ? 'success'
              : failure == 'TIMEOUT'
              ? 'timeout'
              : 'failed',
          if (failure != null) 'error_msg': failure,
        },
      );
      return true;
    } catch (_) {
      // No raw error, access code, device name, token or URL is persisted.
      return false;
    }
  }
}
