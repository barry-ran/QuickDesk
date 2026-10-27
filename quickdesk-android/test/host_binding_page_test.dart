import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:quickdesk_android/account/account_page.dart';
import 'package:quickdesk_android/account/account_service.dart';
import 'package:quickdesk_android/account/device_service.dart';
import 'package:quickdesk_android/core/app_settings.dart';
import 'package:quickdesk_android/host/host_credential_store.dart';
import 'package:quickdesk_android/l10n/app_strings.dart';

class BindingAccount extends AccountService {
  BindingAccount() : super(signalingUrl: 'https://a.example');
  int epoch = 1;
  bool signedIn = true;
  String? logoutDevice;
  @override
  int get sessionEpoch => epoch;
  @override
  bool get isLoggedIn => signedIn;
  @override
  String get username => 'alice';
  @override
  Future<void> initialize() async {}
  @override
  Future<void> logout({String? deviceId}) async {
    logoutDevice = deviceId;
    signedIn = false;
    epoch++;
    notifyListeners();
  }

  void switchAccount() {
    epoch++;
    notifyListeners();
  }
}

class BindingDevices extends DeviceService {
  BindingDevices(super.account);
  final bound = <String>[];
  @override
  void start() {}
  @override
  Future<void> refresh() async {}
  @override
  Future<void> bind(String id) async {
    bound.add(id);
  }
}

class BindingStore extends HostCredentialStore {
  BindingStore() : super(signalingUrl: 'https://a.example');
  HostCredentials credentials = HostCredentials(
    deviceUuid: 'uuid',
    deviceId: '123',
    deviceSecret: 'secret',
  );
  bool failRead = false;
  @override
  Future<HostCredentials> load() async {
    if (failRead) throw StateError('Keystore failure');
    return credentials;
  }
}

void main() {
  setUp(() {
    L10n.locale = AppLocale.en;
    SharedPreferences.setMockInitialValues({
      'signaling_url': 'https://a.example',
    });
  });
  Future<void> mount(
    WidgetTester tester,
    BindingAccount account,
    BindingDevices devices,
    BindingStore store,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: AccountPage(
            account: account,
            devices: devices,
            hostStore: (_) => store,
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  Future<void> open(WidgetTester tester) async {
    await tester.tap(find.byKey(const Key('account-bind-host')));
    await tester.pumpAndSettle();
  }

  testWidgets(
    'binding is explicit and cancel never posts; confirmation posts once',
    (tester) async {
      final account = BindingAccount();
      final devices = BindingDevices(account);
      final store = BindingStore();
      await mount(tester, account, devices, store);
      expect(devices.bound, isEmpty); // no auto-bind on restored login
      await open(tester);
      expect(find.textContaining('transfers ownership'), findsOneWidget);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(devices.bound, isEmpty);
      await open(tester);
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(devices.bound, ['123']);
      await tester.pumpWidget(const SizedBox());
      devices.dispose();
      account.dispose();
    },
  );
  testWidgets(
    'account epoch change while dialog open invalidates binding consent',
    (tester) async {
      final account = BindingAccount();
      final devices = BindingDevices(account);
      final store = BindingStore();
      await mount(tester, account, devices, store);
      await open(tester);
      account.switchAccount();
      await tester.pump();
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(devices.bound, isEmpty);
      await tester.pumpWidget(const SizedBox());
      devices.dispose();
      account.dispose();
    },
  );
  testWidgets(
    'device identity change while dialog open invalidates binding consent',
    (tester) async {
      final account = BindingAccount();
      final devices = BindingDevices(account);
      final store = BindingStore();
      await mount(tester, account, devices, store);
      await open(tester);
      store.credentials = HostCredentials(
        deviceUuid: 'new-uuid',
        deviceId: '456',
        deviceSecret: 'new-secret',
      );
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(devices.bound, isEmpty);
      await tester.pumpWidget(const SizedBox());
      devices.dispose();
      account.dispose();
    },
  );
  testWidgets('server change while dialog open invalidates binding consent', (
    tester,
  ) async {
    final account = BindingAccount();
    final devices = BindingDevices(account);
    final store = BindingStore();
    await mount(tester, account, devices, store);
    await open(tester);
    await AppSettings.save(signalingUrl: 'https://b.example', apiKey: '');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(devices.bound, isEmpty);
    await tester.pumpWidget(const SizedBox());
    devices.dispose();
    account.dispose();
  });
  testWidgets('logout clears the current scoped host session', (tester) async {
    final account = BindingAccount();
    final devices = BindingDevices(account);
    final store = BindingStore();
    await mount(tester, account, devices, store);
    await tester.tap(find.text('Sign out'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(account.logoutDevice, '123');
    await tester.pumpWidget(const SizedBox());
    devices.dispose();
    account.dispose();
  });
  testWidgets('host Keystore failure does not prevent account logout', (
    tester,
  ) async {
    final account = BindingAccount();
    final devices = BindingDevices(account);
    final store = BindingStore()..failRead = true;
    await mount(tester, account, devices, store);
    await tester.tap(find.text('Sign out'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(account.signedIn, isFalse);
    expect(account.logoutDevice, isNull);
    expect(find.textContaining('revocation was not confirmed'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    devices.dispose();
    account.dispose();
  });
}
