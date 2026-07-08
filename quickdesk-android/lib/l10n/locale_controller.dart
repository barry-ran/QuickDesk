/// locale_controller.dart - 全局语言状态（持久化 + 通知重建）
library;

import 'package:flutter/widgets.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'app_strings.dart';

class LocaleController {
  LocaleController._();
  static final LocaleController instance = LocaleController._();

  final ValueNotifier<AppLocale> locale = ValueNotifier(AppLocale.zh);

  static const _key = 'app_locale';

  Future<void> load() async {
    final prefs = await SharedPreferences.getInstance();
    final v = prefs.getString(_key);
    final loc = v == 'en' ? AppLocale.en : AppLocale.zh;
    L10n.locale = loc;
    locale.value = loc;
  }

  Future<void> set(AppLocale loc) async {
    L10n.locale = loc;
    locale.value = loc;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_key, loc == AppLocale.en ? 'en' : 'zh');
  }

  void toggle() {
    set(locale.value == AppLocale.zh ? AppLocale.en : AppLocale.zh);
  }
}
