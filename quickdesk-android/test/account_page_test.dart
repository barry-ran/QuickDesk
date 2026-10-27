import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:quickdesk_android/account/device_service.dart';
import 'package:quickdesk_android/core/app_settings.dart';
import 'package:quickdesk_android/account/account_page.dart';
import 'package:quickdesk_android/account/account_service.dart';
import 'package:quickdesk_android/l10n/app_strings.dart';

class _Account extends AccountService {
  _Account() : super(signalingUrl: 'https://example.test');
  bool signedIn = false;
  @override
  bool get isLoggedIn => signedIn;
  @override
  String get username => 'alice';
  @override
  Future<void> initialize() async {}
  int loginCalls = 0;
  int registerCalls = 0;
  Completer<void>? pending;
  @override
  Future<void> login(String identifier, String password) async {
    loginCalls++;
    await pending?.future;
  }

  @override
  Future<void> register({
    required String username,
    required String password,
    String phone = '',
    String email = '',
    String smsCode = '',
  }) async {
    registerCalls++;
  }
}

class _Devices extends DeviceService {
  _Devices(super.account);
  int edits = 0;
  @override
  List<Map<String, dynamic>> get devices => [
    {
      'device_id': '123',
      'device_name': 'Office',
      'online': true,
      'logged_in': false,
    },
  ];
  @override
  void start() {}
  @override
  Future<void> refresh() async {}
  @override
  Future<void> setRemark(String id, String remark) async {
    edits++;
  }
}

void main() {
  setUp(() {
    L10n.locale = AppLocale.en;
    SharedPreferences.setMockInitialValues({});
  });
  Future<void> mount(
    WidgetTester tester,
    _Account account, {
    Future<void> Function()? onAuthenticated,
  }) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: AccountLoginForm(
              account: account,
              onAuthenticated: onAuthenticated ?? () async {},
            ),
          ),
        ),
      ),
    );
  }

  testWidgets('validation, repeated submit, failed login and retry', (
    tester,
  ) async {
    final account = _Account();
    addTearDown(account.dispose);
    await mount(tester, account);
    await tester.tap(find.byKey(const Key('account-submit')));
    await tester.pump();
    expect(account.loginCalls, 0);
    await tester.enterText(find.byKey(const Key('account-username')), 'alice');
    await tester.enterText(
      find.byKey(const Key('account-password')),
      'wrongpass',
    );
    account.pending = Completer<void>();
    await tester.tap(find.byKey(const Key('account-submit')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('account-submit')));
    expect(account.loginCalls, 1);
    account.pending!.completeError(
      const AccountException(401, 'INVALID_CREDENTIALS', 'Invalid credentials'),
    );
    await tester.pumpAndSettle();
    expect(find.text('Invalid username or password'), findsOneWidget);
    account.pending = null;
    await tester.tap(find.byKey(const Key('account-submit')));
    await tester.pumpAndSettle();
    expect(account.loginCalls, 2);
    expect(
      tester
          .widget<TextFormField>(find.byKey(const Key('account-password')))
          .controller!
          .text,
      isEmpty,
    );
  });

  testWidgets(
    'registration requires password, permits Unicode and feature gates SMS',
    (tester) async {
      final account = _Account();
      addTearDown(account.dispose);
      await mount(tester, account);
      expect(find.text('SMS sign-in'), findsNothing);
      await tester.tap(find.text('Register'));
      await tester.pumpAndSettle();
      await tester.enterText(
        find.byKey(const Key('account-username')),
        'alice',
      );
      await tester.enterText(
        find.byKey(const Key('account-password')),
        'short',
      );
      await tester.ensureVisible(find.byKey(const Key('account-submit')));
      await tester.tap(find.byKey(const Key('account-submit')));
      await tester.pump();
      expect(account.registerCalls, 0);
      await tester.enterText(
        find.byKey(const Key('account-password')),
        '中文密码123',
      );
      await tester.ensureVisible(find.byKey(const Key('account-submit')));
      await tester.tap(find.byKey(const Key('account-submit')));
      await tester.pumpAndSettle();
      expect(account.registerCalls, 1);
    },
  );

  testWidgets('narrow large-text form remains scrollable without overflow', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 568);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final account = _Account()..smsEnabled = true;
    addTearDown(account.dispose);
    await tester.pumpWidget(
      MaterialApp(
        home: MediaQuery(
          data: const MediaQueryData(textScaler: TextScaler.linear(2)),
          child: Scaffold(
            body: SingleChildScrollView(
              child: AccountLoginForm(
                account: account,
                onAuthenticated: () async {},
              ),
            ),
          ),
        ),
      ),
    );
    await tester.ensureVisible(find.text('Register'));
    await tester.tap(find.text('Register'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.byKey(const Key('account-submit')));
    expect(tester.takeException(), isNull);
  });
  testWidgets('cancel and server switch discard pending device edits', (
    tester,
  ) async {
    final account = _Account()..signedIn = true;
    final devices = _Devices(account);
    addTearDown(account.dispose);
    addTearDown(devices.dispose);
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: AccountPage(account: account, devices: devices),
        ),
      ),
    );
    await tester.pumpAndSettle();
    final connect = tester.widget<TextButton>(
      find.widgetWithText(TextButton, 'Connect'),
    );
    expect(connect.onPressed, isNull); // online is not the same as logged in
    await tester.tap(find.text('Edit label'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();
    expect(devices.edits, 0);
    await tester.tap(find.text('Edit label'));
    await tester.pumpAndSettle();
    await AppSettings.save(
      signalingUrl: 'https://another.example.test',
      apiKey: '',
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm'));
    await tester.pumpAndSettle();
    expect(devices.edits, 0);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox.shrink());
  });
}
