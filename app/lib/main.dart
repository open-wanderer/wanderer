import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/toast_overlay.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'i18n/app_localizations.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'theme/theme.dart';
import 'provider/router_provider.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final container = ProviderContainer();
  await container.read(cookieJarProvider.future);

  runApp(const ProviderScope(child: MainApp()));
}

class MainApp extends ConsumerWidget {
  const MainApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final goRouter = ref.watch(routerProvider);

    return MaterialApp.router(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('de')],
      debugShowCheckedModeBanner: false,
      theme: AppTheme.createTheme(Brightness.light),
      darkTheme: AppTheme.createTheme(Brightness.dark),
      themeMode: ThemeMode.system,

      routerConfig: goRouter,

      builder: (context, child) {
        return ToastOverlay(child: child!);
      },
    );
  }
}
