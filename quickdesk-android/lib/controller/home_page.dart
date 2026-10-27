/// home_page.dart - 顶层页面：主控 / 被控 两个模式切换
library;

import 'package:flutter/material.dart';

import '../l10n/app_strings.dart';
import '../account/account_page.dart';
import '../l10n/locale_controller.dart';
import 'connect_page.dart';
import 'host_page.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _index = 0;

  // IndexedStack 保活两个页面：被控页在切走后仍保持会话
  final _pages = const [ConnectPage(), HostPage(), AccountPage()];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(
          _index == 0
              ? L10n.t('home.titleController')
              : _index == 1
              ? L10n.t('home.titleHost')
              : L10n.t('account.title'),
        ),
        actions: [
          TextButton(
            onPressed: () => LocaleController.instance.toggle(),
            child: Text(
              L10n.locale == AppLocale.zh ? 'EN' : '中',
              style: TextStyle(
                color: Theme.of(context).colorScheme.onSurface,
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
        ],
      ),
      body: SafeArea(
        child: IndexedStack(index: _index, children: _pages),
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (i) => setState(() => _index = i),
        destinations: [
          NavigationDestination(
            icon: const Icon(Icons.desktop_windows_outlined),
            label: L10n.t('home.controller'),
          ),
          NavigationDestination(
            icon: const Icon(Icons.smartphone_outlined),
            label: L10n.t('home.host'),
          ),
          NavigationDestination(
            icon: const Icon(Icons.person_outline),
            label: L10n.t('account.title'),
          ),
        ],
      ),
    );
  }
}
