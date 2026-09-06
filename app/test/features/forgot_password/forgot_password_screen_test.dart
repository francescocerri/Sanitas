import 'dart:async';

import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/auth/auth_controller.dart';
import 'package:sanitas_app/core/auth/auth_state.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/forgot_password/forgot_password_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _testCommitteeTheme = CommitteeTheme(
  committeeName: 'Test',
  defaultLocale: 'it',
  primary: Colors.blue,
  secondary: Colors.indigo,
  surface: Colors.white,
);

class _RequestResetController extends AuthController {
  final calls = <String>[];
  final completion = Completer<void>();

  @override
  AuthSession build() => const AuthSession.unauthenticated();

  @override
  Future<void> requestPasswordReset({required String identifier}) {
    calls.add(identifier);
    return completion.future;
  }
}

void main() {
  late _RequestResetController controller;
  setUp(() => controller = _RequestResetController());
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
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
        fallbackLocale: const Locale('it'),
        startLocale: const Locale('it'),
        child: Builder(
          builder: (context) => MaterialApp(
            localizationsDelegates: context.localizationDelegates,
            supportedLocales: context.supportedLocales,
            locale: context.locale,
            home: const ForgotPasswordScreen(),
          ),
        ),
      ),
    );
  }

  testWidgets('mostra un errore di validazione se il campo è vuoto', (
    tester,
  ) async {
    // `runAsync` + un piccolo ritardo: `AuthShell` usa `google_fonts`, che
    // tenta un caricamento di rete reale al primo utilizzo — senza questo,
    // `pumpAndSettle` non riesce ad aspettarlo e un secondo test nello
    // stesso file può trovare l'albero dei widget ancora incompleto (stesso
    // problema già visto in activate_account_screen_test.dart).
    await tester.runAsync(() async {
      await tester.pumpWidget(buildTestApp());
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pumpAndSettle();

    await tester.tap(find.byType(FilledButton));
    await tester.pumpAndSettle();

    expect(find.text('Inserisci email o username'), findsOneWidget);
    expect(controller.calls, isEmpty);
  });

  testWidgets('mostra sempre lo stesso messaggio di conferma dopo l\'invio', (
    tester,
  ) async {
    await tester.runAsync(() async {
      await tester.pumpWidget(buildTestApp());
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextFormField), 'mario.rossi');
    await tester.tap(find.byType(FilledButton));
    await tester.pump();

    expect(controller.calls, ['mario.rossi']);

    await tester.runAsync(() async {
      controller.completion.complete();
      await controller.completion.future;
    });
    await tester.pump();
    await tester.pumpAndSettle();

    expect(
      find.textContaining('Se l\'account esiste, controlla la tua email'),
      findsOneWidget,
    );
  });
}
