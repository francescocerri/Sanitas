import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/theme/committee_theme.dart';
import 'core/theme/theme_mode_controller.dart';
import 'router.dart';

Future<void> main() async {
  // `runApp` normalmente si può chiamare subito, ma qui prima serve fare
  // due cose asincrone (leggere il tema del comitato da un asset, e
  // inizializzare easy_localization) — `WidgetsFlutterBinding.ensureInitialized()`
  // prepara i binding di Flutter perché quel codice asincrono possa girare
  // PRIMA che l'albero dei widget esista ancora.
  WidgetsFlutterBinding.ensureInitialized();
  await EasyLocalization.ensureInitialized();

  final committeeTheme = await loadCommitteeTheme();

  runApp(
    // `ProviderScope` è il widget radice richiesto da Riverpod: senza,
    // nessun `ref.watch`/`ref.read` nel resto dell'app funzionerebbe. Gli
    // `overrides` sostituiscono un provider "segnaposto"
    // (`committeeThemeProvider`, vedi quel file) con il valore vero appena
    // caricato — l'unico modo pulito di far arrivare un dato caricato in
    // modo asincrono PRIMA di `runApp` dentro il sistema di provider.
    ProviderScope(
      overrides: [committeeThemeProvider.overrideWithValue(committeeTheme)],
      child: EasyLocalization(
        supportedLocales: const [Locale('it'), Locale('en')],
        path: 'assets/translations',
        fallbackLocale: const Locale('it'),
        // Ogni comitato può avere una lingua di partenza diversa (vedi
        // `default_locale` in `config/<slug>/app/theme.json`); il progetto
        // di riferimento la imposta a italiano.
        startLocale: Locale(committeeTheme.defaultLocale),
        child: const SanitasApp(),
      ),
    ),
  );
}

class SanitasApp extends ConsumerWidget {
  const SanitasApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final committeeTheme = ref.watch(committeeThemeProvider);
    final themeMode = ref.watch(themeModeControllerProvider);
    final router = ref.watch(routerProvider);

    // `MaterialApp.router` (invece del più comune `MaterialApp` con `home:`)
    // è la variante che delega la navigazione a un `RouterConfig` esterno —
    // qui `go_router` tramite `routerProvider` (vedi `router.dart`).
    return MaterialApp.router(
      onGenerateTitle: (context) => 'app.title'.tr(),
      // easy_localization richiede questi 3 parametri per sapere in quale
      // lingua tradurre `.tr()` e con quale convenzione di sistema.
      localizationsDelegates: context.localizationDelegates,
      supportedLocales: context.supportedLocales,
      locale: context.locale,
      theme: committeeTheme.toThemeData(Brightness.light),
      darkTheme: committeeTheme.toThemeData(Brightness.dark),
      themeMode: themeMode,
      routerConfig: router,
    );
  }
}
