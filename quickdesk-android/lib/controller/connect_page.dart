/// connect_page.dart - 连接页：输入设备 ID + 访问码发起连接
library;

import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/signaling_api.dart';
import '../l10n/app_strings.dart';
import 'connection_store.dart';
import 'remote_page.dart';

const kDefaultSignalingUrl = 'wss://qd.quickcoder.cn';

class ConnectPage extends StatefulWidget {
  const ConnectPage({super.key});

  @override
  State<ConnectPage> createState() => _ConnectPageState();
}

class _ConnectPageState extends State<ConnectPage> {
  final _deviceIdCtrl = TextEditingController();
  final _accessCodeCtrl = TextEditingController();
  final _serverCtrl = TextEditingController(text: kDefaultSignalingUrl);
  final _apiKeyCtrl = TextEditingController();

  bool _connecting = false;
  String? _error;

  final ConnectionStore _store = ConnectionStore();
  List<ConnectionEntry> _entries = [];

  @override
  void initState() {
    super.initState();
    _loadPrefs();
    _loadHistory();
  }

  Future<void> _loadHistory() async {
    final entries = await _store.loadSorted();
    if (!mounted) return;
    setState(() => _entries = entries);
  }

  Future<void> _loadPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() {
      _serverCtrl.text = prefs.getString('signaling_url') ?? kDefaultSignalingUrl;
      _apiKeyCtrl.text = prefs.getString('api_key') ?? '';
      _deviceIdCtrl.text = prefs.getString('last_device_id') ?? '';
    });
  }

  Future<void> _savePrefs() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('signaling_url', _serverCtrl.text.trim());
    await prefs.setString('api_key', _apiKeyCtrl.text.trim());
    await prefs.setString('last_device_id', _deviceIdCtrl.text.trim());
  }

  Future<void> _connect() async {
    final deviceId = _deviceIdCtrl.text.trim().replaceAll(' ', '');
    final accessCode = _accessCodeCtrl.text.trim();
    final serverUrl = _serverCtrl.text.trim();

    if (deviceId.isEmpty || accessCode.isEmpty) {
      setState(() => _error = L10n.t('connect.needIdAndCode'));
      return;
    }

    setState(() {
      _connecting = true;
      _error = null;
    });
    await _savePrefs();

    final api = SignalingApi(serverUrl,
        apiKey: _apiKeyCtrl.text.trim().isEmpty ? null : _apiKeyCtrl.text.trim());

    try {
      // 1. 校验访问码换 signal_token
      final signalToken = await api.verifyAccessCode(deviceId, accessCode);
      // 2. 拉取 ICE 配置
      final iceServers = await api.getIceServers();

      if (!mounted) return;
      await Navigator.of(context).push(MaterialPageRoute(
        builder: (_) => RemotePage(
          signalingUrl: serverUrl,
          deviceId: deviceId,
          accessCode: accessCode,
          signalToken: signalToken,
          iceServers: iceServers,
        ),
      ));
      await _loadHistory();
    } on SignalingApiException catch (e) {
      setState(() => _error = switch (e.code) {
            'DEVICE_NOT_FOUND' => L10n.t('connect.errDeviceNotFound'),
            'HOST_OFFLINE' => L10n.t('connect.errHostOffline'),
            'INVALID_CODE' => L10n.t('connect.errInvalidCode'),
            'TOO_MANY_ATTEMPTS' => L10n.t('connect.errTooManyAttempts'),
            _ => L10n.t('connect.errGeneric', {'code': e.code}),
          });
    } catch (e) {
      setState(() => _error = L10n.t('connect.errGeneric', {'code': '$e'}));
    } finally {
      if (mounted) setState(() => _connecting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SingleChildScrollView(
      padding: const EdgeInsets.fromLTRB(20, 24, 20, 32),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _buildHero(context, scheme),
          const SizedBox(height: 24),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  TextField(
                    controller: _deviceIdCtrl,
                    keyboardType: TextInputType.number,
                    style: const TextStyle(
                      letterSpacing: 1.5,
                      fontFeatures: [FontFeature.tabularFigures()],
                    ),
                    decoration: InputDecoration(
                      labelText: L10n.t('connect.deviceId'),
                      hintText: L10n.t('connect.deviceIdHint'),
                      prefixIcon: const Icon(Icons.tag),
                    ),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: _accessCodeCtrl,
                    keyboardType: TextInputType.number,
                    obscureText: true,
                    decoration: InputDecoration(
                      labelText: L10n.t('connect.accessCode'),
                      prefixIcon: const Icon(Icons.password),
                    ),
                  ),
                  const SizedBox(height: 20),
                  FilledButton.icon(
                    onPressed: _connecting ? null : _connect,
                    style: FilledButton.styleFrom(
                        padding: const EdgeInsets.symmetric(vertical: 16)),
                    icon: _connecting
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : const Icon(Icons.cast_connected),
                    label: Text(L10n.t('connect.connect')),
                  ),
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Row(
                      children: [
                        Icon(Icons.error_outline, size: 16, color: scheme.error),
                        const SizedBox(width: 6),
                        Expanded(
                          child: Text(_error!, style: TextStyle(color: scheme.error)),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          Card(
            child: ExpansionTile(
              leading: const Icon(Icons.dns_outlined),
              title: Text(L10n.t('connect.serverSettings')),
              shape: const Border(),
              childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
              children: [
                TextField(
                  controller: _serverCtrl,
                  keyboardType: TextInputType.url,
                  decoration: InputDecoration(
                    labelText: L10n.t('connect.serverUrl'),
                    hintText: 'wss://your-server.com:8000',
                  ),
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: _apiKeyCtrl,
                  decoration: InputDecoration(labelText: L10n.t('connect.apiKey')),
                ),
              ],
            ),
          ),
          if (_entries.isNotEmpty) ...[
            const SizedBox(height: 24),
            Text(L10n.t('connect.recentAndFav'),
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Card(
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 4),
                child: Column(children: _entries.map(_buildHistoryTile).toList()),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildHero(BuildContext context, ColorScheme scheme) {
    return Row(
      children: [
        Container(
          width: 48,
          height: 48,
          decoration: BoxDecoration(
            color: scheme.primaryContainer,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Icon(Icons.cast, color: scheme.onPrimaryContainer),
        ),
        const SizedBox(width: 16),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(L10n.t('connect.title'),
                  style: Theme.of(context).textTheme.titleLarge?.copyWith(
                        fontWeight: FontWeight.w600,
                      )),
              const SizedBox(height: 2),
              Text('QuickDesk',
                  style: TextStyle(color: scheme.outline, fontSize: 13)),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildHistoryTile(ConnectionEntry e) {
    final title = e.name.isNotEmpty ? e.name : e.deviceId;
    return Dismissible(
      key: ValueKey(e.deviceId),
      direction: DismissDirection.endToStart,
      background: Container(
        alignment: Alignment.centerRight,
        padding: const EdgeInsets.only(right: 20),
        color: Theme.of(context).colorScheme.errorContainer,
        child: const Icon(Icons.delete_outline),
      ),
      onDismissed: (_) async {
        await _store.remove(e.deviceId);
        await _loadHistory();
      },
      child: ListTile(
        contentPadding: EdgeInsets.zero,
        leading: IconButton(
          icon: Icon(e.favorite ? Icons.star : Icons.star_border,
              color: e.favorite ? Colors.amber : null),
          tooltip: e.favorite ? L10n.t('connect.unfavorite') : L10n.t('connect.favorite'),
          onPressed: () async {
            await _store.toggleFavorite(e.deviceId);
            await _loadHistory();
          },
        ),
        title: Text(title),
        subtitle: e.name.isNotEmpty ? Text(e.deviceId) : null,
        trailing: const Icon(Icons.north_east, size: 16),
        onTap: () {
          _deviceIdCtrl.text = e.deviceId;
          _accessCodeCtrl.clear();
          FocusScope.of(context).requestFocus(FocusNode());
        },
      ),
    );
  }

  @override
  void dispose() {
    _deviceIdCtrl.dispose();
    _accessCodeCtrl.dispose();
    _serverCtrl.dispose();
    _apiKeyCtrl.dispose();
    super.dispose();
  }
}
