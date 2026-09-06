import 'dart:convert';
import 'dart:io';

import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/activate_account/activate_account_screen.dart';
import 'package:sanitas_app/core/auth/auth_controller.dart';
import 'package:sanitas_app/core/auth/auth_state.dart';

import 'dart:async';

import 'package:shared_preferences/shared_preferences.dart';

const _testCommitteeTheme = CommitteeTheme(
  committeeName: 'Test',
  defaultLocale: 'it',
  primary: Colors.blue,
  secondary: Colors.indigo,
  surface: Colors.white,
);

class _TestTranslations extends AssetLoader {
  const _TestTranslations();

  @override
  Future<Map<String, dynamic>> load(String path, Locale locale) async =>
      jsonDecode(File('$path/${locale.languageCode}.json').readAsStringSync())
          as Map<String, dynamic>;
}

class _ActivationController extends AuthController {
  final calls = <String>[];
  final completion = Completer<void>();

  @override
  AuthSession build() => const AuthSession.unauthenticated();

  @override
  Future<void> activateAccount({
    required String token,
    required String password,
  }) {
    calls.add(password);
    return completion.future;
  }
}

void main() {
  late _ActivationController controller;
  setUp(() => controller = _ActivationController());
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
        authControllerProvider.overrideWith(() => controller),
      ],
      child: EasyLocalization(
        supportedLocales: const [Locale('it'), Locale('en')],
        path: 'assets/translations',
        assetLoader: const _TestTranslations(),
        fallbackLocale: const Locale('it'),
        startLocale: const Locale('it'),
        child: Builder(
          builder: (context) => MaterialApp(
            localizationsDelegates: context.localizationDelegates,
            supportedLocales: context.supportedLocales,
            locale: context.locale,
            home: const ActivateAccountScreen(inviteToken: 'test-invite'),
          ),
        ),
      ),
    );
  }

  testWidgets(
    'activation requires a valid matching password and prevents duplicate submits',
    (tester) async {
      await tester.runAsync(() async {
        await tester.pumpWidget(buildTestApp());
        await Future<void>.delayed(const Duration(milliseconds: 100));
      });
      await tester.pumpAndSettle();
      final fields = find.byType(TextFormField);
      final button = find.byType(FilledButton);
      bool enabled() => tester.widget<FilledButton>(button).onPressed != null;
      Future<void> enter(String password, String confirmation) async {
        await tester.enterText(fields.at(0), password);
        await tester.enterText(fields.at(1), confirmation);
        await tester.pumpAndSettle();
      }

      expect(
        find.textContaining('La password deve contenere almeno 8 caratteri'),
        findsOneWidget,
      );
      expect(enabled(), isFalse);
      await enter('1234567', '1234567');
      expect(enabled(), isFalse);
      expect(find.text('Usa almeno 8 caratteri'), findsOneWidget);
      await enter('12345678', '');
      expect(enabled(), isFalse);
      await enter('12345678', '12345679');
      expect(enabled(), isFalse);
      await enter('12345678', '12345678');
      expect(enabled(), isTrue);
      await tester.enterText(fields.first, 'short');
      await tester.pumpAndSettle();
      expect(enabled(), isFalse);
      await enter('a' * 73, 'a' * 73);
      expect(enabled(), isFalse);
      await enter('é' * 37, 'é' * 37);
      expect(enabled(), isFalse);
      await enter('é' * 36, 'é' * 36);
      expect(enabled(), isTrue);
      expect(controller.calls, isEmpty);
      await tester.ensureVisible(button);
      await tester.tap(button);
      await tester.pump();
      expect(enabled(), isFalse);
      expect(controller.calls, ['é' * 36]);
      await tester.runAsync(() async {
        controller.completion.complete();
        await controller.completion.future;
      });
      await tester.pump();
      await tester.pumpAndSettle();
      expect(find.text('Account attivato: ora puoi accedere'), findsOneWidget);
    },
  );
}
