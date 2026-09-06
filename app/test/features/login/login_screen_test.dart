import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/login/login_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// `AuthShell` (usato da `LoginScreen`) legge `committeeThemeProvider` per
/// il pannello brandizzato e il monogramma — in `main.dart` viene
/// sovrascritto con il tema vero letto da `config/<slug>/app/theme.json`
/// (vedi quel file), qui basta un valore qualunque sintatticamente valido.
const _testCommitteeTheme = CommitteeTheme(
  committeeName: 'Test',
  defaultLocale: 'it',
  primary: Colors.blue,
  secondary: Colors.indigo,
  surface: Colors.white,
);

void main() {
  setUpAll(() async {
    // Nei widget test easy_localization va inizializzato esplicitamente,
    // proprio come in `main.dart` — senza, ogni `.tr()' nella schermata
    // lancerebbe un'eccezione perché non troverebbe alcuna lingua caricata.
    TestWidgetsFlutterBinding.ensureInitialized();
    // easy_localization usa shared_preferences sotto al cofano per
    // ricordare la lingua scelta: in un test Dart puro non esistono i
    // canali nativi veri, quindi shared_preferences va "finto" con questo
    // helper ufficiale del pacchetto, altrimenti ogni lettura fallirebbe
    // con un MissingPluginException.
    SharedPreferences.setMockInitialValues({});
    await EasyLocalization.ensureInitialized();
  });

  Widget buildTestApp() {
    return ProviderScope(
      overrides: [
        committeeThemeProvider.overrideWithValue(_testCommitteeTheme),
      ],
      child: EasyLocalization(
        supportedLocales: const [Locale('it'), Locale('en')],
        path: 'assets/translations',
        fallbackLocale: const Locale('it'),
        startLocale: const Locale('it'),
        child: Builder(
          builder: (context) => MaterialApp(
            localizationsDelegates: context.localizationDelegates,
            supportedLocales: context.supportedLocales,
            locale: context.locale,
            home: const LoginScreen(),
          ),
        ),
      ),
    );
  }

  testWidgets('mostra gli errori di validazione se si invia il form vuoto', (
    tester,
  ) async {
    await tester.pumpWidget(buildTestApp());
    await tester.pumpAndSettle();

    await tester.tap(find.byType(FilledButton));
    await tester.pumpAndSettle();

    expect(find.text('Inserisci email o username'), findsOneWidget);
    expect(find.text('Inserisci la password'), findsOneWidget);
  });
}
