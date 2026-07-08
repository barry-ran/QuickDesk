import 'package:flutter/material.dart';

import 'controller/home_page.dart';
import 'l10n/app_strings.dart';
import 'l10n/locale_controller.dart';
import 'theme/app_theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await LocaleController.instance.load();
  runApp(const QuickDeskApp());
}

class QuickDeskApp extends StatelessWidget {
  const QuickDeskApp({super.key});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<AppLocale>(
      valueListenable: LocaleController.instance.locale,
      builder: (context, _, __) {
        return MaterialApp(
          title: 'QuickDesk',
          debugShowCheckedModeBanner: false,
          theme: AppTheme.light(),
          darkTheme: AppTheme.dark(),
          home: const HomePage(),
        );
      },
    );
  }
}
