import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/account/account_page.dart';
import 'package:quickdesk_android/account/account_service.dart';
import 'package:quickdesk_android/l10n/app_strings.dart';

class ResetAccount extends AccountService {
  ResetAccount() : super(signalingUrl: 'https://a.example');
  final gate = Completer<Uri>();
  @override
  Future<Uri> passwordResetUri() => gate.future;
}

void main() {
  setUp(() => L10n.locale = AppLocale.en);
  testWidgets(
    'reset opens only the confirmed configured URL, without credentials',
    (tester) async {
      final account = ResetAccount();
      final opened = <Uri>[];
      final uri = Uri.parse('https://web.example/#/reset-password');
      account.gate.complete(uri);
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: AccountLoginForm(
                account: account,
                onAuthenticated: () async {},
                openUrl: (url) async {
                  opened.add(url);
                  return true;
                },
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.byKey(const Key('account-forgot-password')));
      await tester.pumpAndSettle();
      expect(find.textContaining(uri.toString()), findsOneWidget);
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(opened, isEmpty);
      await tester.tap(find.byKey(const Key('account-forgot-password')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(opened, [uri]);
      await tester.pumpWidget(const SizedBox());
      account.dispose();
    },
  );
  testWidgets('dismissed login never opens a delayed reset URL', (
    tester,
  ) async {
    final account = ResetAccount();
    final opened = <Uri>[];
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: AccountLoginForm(
              account: account,
              onAuthenticated: () async {},
              openUrl: (url) async {
                opened.add(url);
                return true;
              },
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.byKey(const Key('account-forgot-password')));
    await tester.pump();
    await tester.pumpWidget(const SizedBox());
    account.gate.complete(Uri.parse('https://web.example/#/reset-password'));
    await tester.pumpAndSettle();
    expect(opened, isEmpty);
    account.dispose();
  });
}
