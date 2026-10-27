/// Account UI uses the same server configuration as controller/host, but never
/// transfers account tokens to a new server. Secrets are never rendered in rows.
library;

import 'dart:async';

import 'package:flutter/material.dart';

import '../api/signaling_api.dart';
import '../controller/remote/remote_page.dart';
import '../core/app_settings.dart';
import '../l10n/app_strings.dart';
import 'account_service.dart';
import 'device_service.dart';
import 'device_sort.dart';

class AccountPage extends StatefulWidget {
  const AccountPage({super.key, this.account, this.devices});
  final AccountService? account;
  final DeviceService? devices;

  @override
  State<AccountPage> createState() => _AccountPageState();
}

class _AccountPageState extends State<AccountPage> {
  AccountService? _account;
  DeviceService? _devices;
  AppSettings? _settings;
  bool _busy = false;
  String? _error;
  int _generation = 0;
  int _configurationRequest = 0;

  @override
  void initState() {
    super.initState();
    AppSettings.changes.addListener(_configure);
    _configure();
  }

  Future<void> _configure() async {
    final request = ++_configurationRequest;
    final settings = await AppSettings.load();
    if (!mounted || request != _configurationRequest) {
      return;
    }
    if (_account != null &&
        _settings?.signalingUrl == settings.signalingUrl &&
        _settings?.apiKey == settings.apiKey) {
      return;
    }
    final generation = ++_generation;
    _release();
    _account = null;
    _devices = null;
    _settings = settings;
    setState(() {
      _error = null;
      _busy = true;
    });
    try {
      _account =
          widget.account ??
          AccountService(
            signalingUrl: settings.signalingUrl,
            apiKey: settings.apiKeyOrNull,
          );
      _devices = widget.devices ?? DeviceService(_account!);
      _account!.addListener(_changed);
      _devices!.addListener(_changed);
      await _account!.initialize();
      if (!mounted || generation != _generation) {
        return;
      }
      _devices!.start();
    } catch (_) {
      if (mounted && generation == _generation) {
        setState(() => _error = L10n.t('account.loadFailed'));
      }
    } finally {
      if (mounted && generation == _generation) {
        _devices?.start();
        setState(() => _busy = false);
      }
    }
  }

  bool _current(DeviceService service, int generation, int epoch) =>
      mounted &&
      identical(_devices, service) &&
      _generation == generation &&
      _account?.sessionEpoch == epoch &&
      _account!.isLoggedIn;

  void _changed() {
    if (mounted) {
      setState(() {});
    }
  }

  void _release() {
    _account?.removeListener(_changed);
    _devices?.removeListener(_changed);
    if (widget.devices == null) {
      _devices?.dispose();
    }
    if (widget.account == null) {
      _account?.dispose();
    }
  }

  @override
  void dispose() {
    _generation++;
    AppSettings.changes.removeListener(_configure);
    _release();
    super.dispose();
  }

  Future<void> _run(Future<void> Function() action) async {
    if (_busy) {
      return;
    }
    final generation = _generation;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await action();
    } catch (e) {
      if (mounted && generation == _generation) {
        setState(
          () => _error = L10n.t(
            e is AccountException && e.code == 'LOGOUT_OFFLINE'
                ? 'account.logoutOffline'
                : e is AccountException && e.code == 'SECURE_STORAGE'
                ? 'account.storageFailed'
                : 'account.actionFailed',
          ),
        );
      }
    } finally {
      if (mounted && generation == _generation) {
        setState(() => _busy = false);
      }
    }
  }

  Future<bool> _confirm(String message) async =>
      await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: Text(message),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(L10n.t('common.cancel')),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: Text(L10n.t('account.confirm')),
            ),
          ],
        ),
      ) ??
      false;

  Future<void> _edit(Map<String, dynamic> row, bool favorite) async {
    final service = _devices!;
    final generation = _generation;
    final epoch = _account!.sessionEpoch;
    final result = await showDialog<List<String>>(
      context: context,
      builder: (_) => _FavoriteDialog(row: row, favorite: favorite),
    );
    if (result == null || !_current(service, generation, epoch)) {
      return;
    }
    await _run(
      () => favorite
          ? service.updateFavorite(
              row['device_id'] as String,
              name: result[1],
              password: result[2].isEmpty ? null : result[2],
            )
          : service.setRemark(row['device_id'] as String, result[1]),
    );
  }

  Future<void> _connect(Map<String, dynamic> row, bool favorite) =>
      _run(() async {
        final devices = _devices!;
        final settings = _settings!;
        final generation = _generation;
        final epoch = _account!.sessionEpoch;
        final id = row['device_id'] as String;
        var code = favorite
            ? (row['access_password'] as String? ?? '')
            : await devices.accessCode(id);
        if (!_current(devices, generation, epoch)) {
          return;
        }
        if (code.isEmpty) {
          if (!mounted) {
            return;
          }
          final result = await showDialog<List<String>>(
            context: context,
            builder: (_) =>
                _FavoriteDialog(row: row, favorite: true, connect: true),
          );
          if (result == null || !_current(devices, generation, epoch)) {
            return;
          }
          code = result[2];
        }
        final api = SignalingApi(
          settings.signalingUrl,
          apiKey: settings.apiKeyOrNull,
        );
        final token = await api.verifyAccessCode(id, code);
        if (!_current(devices, generation, epoch)) {
          return;
        }
        final ice = await api.getIceServers();
        if (!_current(devices, generation, epoch)) {
          return;
        }
        if (!mounted) {
          return;
        }
        await Navigator.of(context).push(
          MaterialPageRoute<void>(
            builder: (_) => RemotePage(
              signalingUrl: settings.signalingUrl,
              deviceId: id,
              accessCode: code,
              signalToken: token,
              iceServers: ice,
            ),
          ),
        );
      });

  Widget _row(Map<String, dynamic> row, bool favorite) {
    final id = row['device_id'] as String? ?? '';
    final title = deviceSortDisplayName(row, favorites: favorite);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title.isEmpty ? id : title,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            SelectableText(id),
            if (!favorite)
              Text(
                L10n.t(
                  row['online'] == true ? 'account.online' : 'account.offline',
                ),
              ),
            if (!favorite && row['logged_in'] == true)
              Text(L10n.t('account.deviceLoggedIn')),
            Wrap(
              spacing: 8,
              children: [
                TextButton.icon(
                  onPressed:
                      _busy ||
                          (!favorite &&
                              (row['online'] != true ||
                                  row['logged_in'] != true))
                      ? null
                      : () => _connect(row, favorite),
                  icon: const Icon(Icons.cast),
                  label: Text(L10n.t('connect.connect')),
                ),
                TextButton(
                  onPressed: _busy ? null : () => _edit(row, favorite),
                  child: Text(L10n.t('account.edit')),
                ),
                TextButton(
                  onPressed: _busy
                      ? null
                      : () async {
                          final service = _devices!;
                          final generation = _generation;
                          final epoch = _account!.sessionEpoch;
                          if (!await _confirm(
                                L10n.t(
                                  favorite
                                      ? 'account.removeConfirm'
                                      : 'account.unbindConfirm',
                                ),
                              ) ||
                              !_current(service, generation, epoch)) {
                            return;
                          }
                          await _run(
                            () => favorite
                                ? service.removeFavorite(id)
                                : service.unbind(id),
                          );
                        },
                  child: Text(
                    L10n.t(favorite ? 'account.remove' : 'account.unbind'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final account = _account;
    final devices = _devices;
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 720),
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              L10n.t('account.title'),
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 8),
            Text(
              _settings?.signalingUrl ?? '',
              key: const Key('account-server'),
            ),
            Text(L10n.t('account.serverHint')),
            if (_settings != null &&
                !{
                  'https',
                  'wss',
                }.contains(Uri.tryParse(_settings!.signalingUrl)?.scheme))
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Text(
                  L10n.t('account.insecure'),
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            if (_busy || devices?.loading == true)
              const LinearProgressIndicator(),
            if (_error != null || devices?.error != null)
              Semantics(
                liveRegion: true,
                child: Text(
                  _error ?? L10n.t('account.loadFailed'),
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            const SizedBox(height: 16),
            if (_error != null && account?.isLoggedIn != true)
              TextButton(
                onPressed: _busy
                    ? null
                    : () => _run(() async {
                        if (account != null) {
                          await account.initialize();
                          _devices?.start();
                        }
                      }),
                child: Text(L10n.t('account.refresh')),
              ),
            if (account != null && !account.isLoggedIn && !_busy)
              AccountLoginForm(
                key: ObjectKey(account),
                account: account,
                onAuthenticated: () async {
                  devices?.start();
                  await devices?.refresh();
                },
              ),
            if (account?.isLoggedIn == true && devices != null) ...[
              Text(
                account!.username,
                style: Theme.of(context).textTheme.titleLarge,
              ),
              Wrap(
                spacing: 8,
                children: [
                  OutlinedButton.icon(
                    onPressed: _busy ? null : () => _run(devices.refresh),
                    icon: const Icon(Icons.refresh),
                    label: Text(L10n.t('account.refresh')),
                  ),
                  TextButton(
                    onPressed: _busy
                        ? null
                        : () async {
                            final generation = _generation;
                            final epoch = account.sessionEpoch;
                            if (!await _confirm(
                                  L10n.t('account.logoutConfirm'),
                                ) ||
                                !_current(devices, generation, epoch)) {
                              return;
                            }
                            await _run(() => account.logout());
                          },
                    child: Text(L10n.t('account.logout')),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              Text(
                L10n.t('account.devices'),
                style: Theme.of(context).textTheme.titleLarge,
              ),
              if (devices.devices.isEmpty) Text(L10n.t('account.noDevices')),
              ...sortedDeviceRows(devices.devices)
                  .map((row) => _row(row, false)),
              const SizedBox(height: 20),
              Text(
                L10n.t('account.favorites'),
                style: Theme.of(context).textTheme.titleLarge,
              ),
              Text(L10n.t('account.favoriteHint')),
              Align(
                alignment: AlignmentDirectional.centerStart,
                child: TextButton.icon(
                  onPressed: _busy
                      ? null
                      : () async {
                          final generation = _generation;
                          final epoch = account.sessionEpoch;
                          final result = await showDialog<List<String>>(
                            context: context,
                            builder: (_) =>
                                const _FavoriteDialog(favorite: true),
                          );
                          if (result == null ||
                              !_current(devices, generation, epoch)) {
                            return;
                          }
                          await _run(
                            () => devices.addFavorite(
                              result[0],
                              name: result[1],
                              password: result[2],
                            ),
                          );
                        },
                  icon: const Icon(Icons.add),
                  label: Text(L10n.t('account.addFavorite')),
                ),
              ),
              if (devices.favorites.isEmpty)
                Text(L10n.t('account.noFavorites')),
              ...sortedDeviceRows(
                devices.favorites,
                favorites: true,
              ).map((row) => _row(row, true)),
              const SizedBox(height: 20),
              Text(
                L10n.t('account.history'),
                style: Theme.of(context).textTheme.titleLarge,
              ),
              if (devices.connections.isEmpty)
                Text(L10n.t('account.noHistory')),
              ...devices.connections
                  .take(20)
                  .map(
                    (row) => ListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text(row['device_id'] as String? ?? ''),
                      subtitle: Text(
                        [
                          if (row['created_at'] is String)
                            row['created_at'] as String,
                          L10n.t(
                            row['status'] == 'success'
                                ? 'account.success'
                                : 'account.failed',
                          ),
                          if (row['duration'] is num) '${row['duration']} s',
                        ].join(' · '),
                      ),
                    ),
                  ),
            ],
          ],
        ),
      ),
    );
  }
}

class _FavoriteDialog extends StatefulWidget {
  const _FavoriteDialog({
    this.row,
    required this.favorite,
    this.connect = false,
  });
  final Map<String, dynamic>? row;
  final bool favorite;
  final bool connect;
  @override
  State<_FavoriteDialog> createState() => _FavoriteDialogState();
}

class _FavoriteDialogState extends State<_FavoriteDialog> {
  final _form = GlobalKey<FormState>();
  late final _id = TextEditingController(
    text: widget.row?['device_id'] as String?,
  );
  late final _name = TextEditingController(
    text: widget.row?[widget.favorite ? 'device_name' : 'remark'] as String?,
  );
  final _password = TextEditingController();
  @override
  void dispose() {
    _id.dispose();
    _name.dispose();
    _password.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
    title: Text(L10n.t(widget.connect ? 'connect.connect' : 'account.edit')),
    content: SingleChildScrollView(
      child: Form(
        key: _form,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (widget.row == null)
              TextFormField(
                controller: _id,
                decoration: InputDecoration(
                  labelText: L10n.t('connect.deviceId'),
                ),
                validator: (v) => v == null || v.trim().isEmpty
                    ? L10n.t('account.required')
                    : null,
              ),
            if (!widget.connect)
              TextFormField(
                controller: _name,
                decoration: InputDecoration(
                  labelText: L10n.t('account.remark'),
                ),
              ),
            if (widget.favorite)
              TextFormField(
                controller: _password,
                obscureText: true,
                enableSuggestions: false,
                autocorrect: false,
                decoration: InputDecoration(
                  labelText: L10n.t('connect.accessCode'),
                  helperText: widget.row != null && !widget.connect
                      ? L10n.t('account.keepPassword')
                      : null,
                ),
                validator: (v) => widget.connect && (v == null || v.isEmpty)
                    ? L10n.t('account.required')
                    : null,
              ),
          ],
        ),
      ),
    ),
    actions: [
      TextButton(
        onPressed: () => Navigator.pop(context),
        child: Text(L10n.t('common.cancel')),
      ),
      FilledButton(
        onPressed: () {
          if (_form.currentState!.validate()) {
            Navigator.pop(context, [
              _id.text.trim(),
              _name.text.trim(),
              _password.text,
            ]);
          }
        },
        child: Text(L10n.t('account.confirm')),
      ),
    ],
  );
}

/// Separate widget keeps form state stable when account/device notifications arrive.
class AccountLoginForm extends StatefulWidget {
  const AccountLoginForm({
    super.key,
    required this.account,
    required this.onAuthenticated,
  });
  final AccountService account;
  final Future<void> Function() onAuthenticated;
  @override
  State<AccountLoginForm> createState() => _AccountLoginFormState();
}

class _AccountLoginFormState extends State<AccountLoginForm> {
  final _form = GlobalKey<FormState>();
  final _username = TextEditingController();
  final _password = TextEditingController();
  final _phone = TextEditingController();
  final _email = TextEditingController();
  final _code = TextEditingController();
  String _mode = 'login';
  String? _error;
  bool _busy = false;
  int _seconds = 0;
  Timer? _timer;
  @override
  void dispose() {
    _timer?.cancel();
    for (final c in [_username, _password, _phone, _email, _code]) {
      c.dispose();
    }
    super.dispose();
  }

  String? _required(String? v) =>
      v == null || v.trim().isEmpty ? L10n.t('account.required') : null;
  String _message(Object e) {
    if (e is AccountException) {
      final key = 'account.error.${e.code}';
      final translated = L10n.t(key);
      if (translated != key) {
        return translated;
      }
    }
    return L10n.t('account.actionFailed');
  }

  Future<void> _submit() async {
    if (_busy || !_form.currentState!.validate()) {
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      switch (_mode) {
        case 'register':
          await widget.account.register(
            username: _username.text.trim(),
            password: _password.text,
            phone: _phone.text.trim(),
            email: _email.text.trim(),
            smsCode: _code.text.trim(),
          );
        case 'sms':
          await widget.account.loginSms(_phone.text.trim(), _code.text.trim());
        default:
          await widget.account.login(_username.text.trim(), _password.text);
      }
      if (!mounted) {
        return;
      }
      _password.clear();
      _code.clear();
      await widget.onAuthenticated();
    } catch (e) {
      if (mounted) {
        setState(() => _error = _message(e));
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _sendSms() async {
    if (_busy || _seconds > 0 || _phone.text.trim().isEmpty) {
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.account.sendSmsCode(
        _phone.text.trim(),
        _mode == 'register' ? 'register' : 'login',
      );
      if (!mounted) {
        return;
      }
      setState(() => _seconds = 60);
      _timer?.cancel();
      _timer = Timer.periodic(const Duration(seconds: 1), (timer) {
        if (!mounted) {
          timer.cancel();
          return;
        }
        setState(() => _seconds--);
        if (_seconds <= 0) {
          timer.cancel();
        }
      });
    } catch (e) {
      if (mounted) {
        setState(() => _error = _message(e));
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  void _switch(String mode) => setState(() {
    _mode = mode;
    _error = null;
    _password.clear();
    _code.clear();
    _form.currentState?.reset();
  });
  @override
  Widget build(BuildContext context) => AutofillGroup(
    child: Form(
      key: _form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            L10n.t(_mode == 'register' ? 'account.register' : 'account.login'),
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 16),
          if (_mode != 'sms') ...[
            TextFormField(
              key: const Key('account-username'),
              controller: _username,
              enabled: !_busy,
              autocorrect: false,
              textInputAction: TextInputAction.next,
              autofillHints: const [AutofillHints.username],
              validator: _required,
              decoration: InputDecoration(
                labelText: L10n.t('account.username'),
              ),
            ),
            const SizedBox(height: 12),
            TextFormField(
              key: const Key('account-password'),
              controller: _password,
              enabled: !_busy,
              obscureText: true,
              autocorrect: false,
              enableSuggestions: false,
              autofillHints: [
                _mode == 'register'
                    ? AutofillHints.newPassword
                    : AutofillHints.password,
              ],
              validator: (v) {
                if (v == null || v.isEmpty) {
                  return L10n.t('account.required');
                }
                return null;
              },
              onFieldSubmitted: (_) => _submit(),
              decoration: InputDecoration(
                labelText: L10n.t('account.password'),
                helperText: _mode == 'register'
                    ? L10n.t('account.passwordRule')
                    : null,
                helperMaxLines: 3,
              ),
            ),
          ],
          if (_mode == 'register' || _mode == 'sms') ...[
            const SizedBox(height: 12),
            TextFormField(
              controller: _phone,
              enabled: !_busy,
              keyboardType: TextInputType.phone,
              autofillHints: const [AutofillHints.telephoneNumber],
              validator: _mode == 'sms' ? _required : null,
              decoration: InputDecoration(
                labelText: L10n.t(
                  _mode == 'sms' ? 'account.phone' : 'account.phoneOptional',
                ),
              ),
            ),
            if (widget.account.smsEnabled) ...[
              const SizedBox(height: 12),
              TextFormField(
                controller: _code,
                enabled: !_busy,
                keyboardType: TextInputType.number,
                autofillHints: const [AutofillHints.oneTimeCode],
                validator: (value) =>
                    _mode == 'sms' || _phone.text.trim().isNotEmpty
                    ? _required(value)
                    : null,
                decoration: InputDecoration(
                  labelText: L10n.t('account.smsCode'),
                ),
              ),
              Align(
                alignment: AlignmentDirectional.centerStart,
                child: TextButton(
                  onPressed: _busy || _seconds > 0 ? null : _sendSms,
                  child: Text(
                    _seconds > 0 ? '${_seconds}s' : L10n.t('account.sendSms'),
                  ),
                ),
              ),
            ],
          ],
          if (_mode == 'register') ...[
            const SizedBox(height: 12),
            TextFormField(
              controller: _email,
              enabled: !_busy,
              keyboardType: TextInputType.emailAddress,
              autofillHints: const [AutofillHints.email],
              decoration: InputDecoration(
                labelText: L10n.t('account.emailOptional'),
              ),
            ),
          ],
          if (_error != null)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Semantics(
                liveRegion: true,
                child: Text(
                  _error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
            ),
          const SizedBox(height: 16),
          FilledButton(
            key: const Key('account-submit'),
            onPressed: _busy ? null : _submit,
            child: Text(
              L10n.t(
                _busy
                    ? 'account.working'
                    : _mode == 'register'
                    ? 'account.register'
                    : 'account.login',
              ),
            ),
          ),
          Wrap(
            spacing: 8,
            children: [
              if (_mode != 'login')
                TextButton(
                  onPressed: _busy ? null : () => _switch('login'),
                  child: Text(L10n.t('account.passwordLogin')),
                ),
              if (_mode != 'register' && widget.account.registerEnabled)
                TextButton(
                  onPressed: _busy ? null : () => _switch('register'),
                  child: Text(L10n.t('account.register')),
                ),
              if (_mode != 'sms' && widget.account.smsEnabled)
                TextButton(
                  onPressed: _busy ? null : () => _switch('sms'),
                  child: Text(L10n.t('account.smsLogin')),
                ),
            ],
          ),
        ],
      ),
    ),
  );
}
