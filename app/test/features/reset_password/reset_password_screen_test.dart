import 'dart:async';

import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/auth/auth_controller.dart';
import 'package:sanitas_app/core/auth/auth_state.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/reset_password/reset_password_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _testCommitteeTheme = CommitteeTheme(
  committeeName: 'Test',
  defaultLocale: 'it',
  primary: Colors.blue,
  secondary: Colors.indigo,
  surface: Colors.white,
);

class _ConfirmResetController extends AuthController {
  final calls = <String>[];
  final completion = Completer<void>();

  @override
  AuthSession build() => const AuthSession.unauthenticated();

  @override
  Future<void> confirmPasswordReset({
    required String token,
    required String password,
  }) {
    calls.add(password);
    return completion.future;
  }
}

void main() {
  late _ConfirmResetController controller;
  setUp(() => controller = _ConfirmResetController());
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    SharedPreferences.setMockInitialValues({});
    await EasyLocalization.ensureInitialized();
  });

  Widget buildTestApp({String? resetToken = 'test-reset-token'}) {
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
            home: ResetPasswordScreen(resetToken: resetToken),
          ),
        ),
      ),
    );
  }

  testWidgets('mostra un errore se il token manca dal link', (tester) async {
    // Vedi commento in forgot_password_screen_test.dart: `AuthShell` usa
    // `google_fonts`, il cui caricamento di rete reale va aspettato con
    // `runAsync`, altrimenti un secondo test nello stesso file può trovare
    // l'albero dei widget ancora incompleto.
    await tester.runAsync(() async {
      await tester.pumpWidget(buildTestApp(resetToken: null));
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pumpAndSettle();

    expect(find.text('Link non valido: token mancante'), findsOneWidget);
  });

  testWidgets(
    'richiede una password valida e coincidente prima di abilitare il submit',
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

      expect(enabled(), isFalse);
      await enter('1234567', '1234567');
      expect(enabled(), isFalse);
      await enter('12345678', '12345679');
      expect(enabled(), isFalse);
      await enter('12345678', '12345678');
      expect(enabled(), isTrue);

      expect(controller.calls, isEmpty);
      await tester.ensureVisible(button);
      await tester.tap(button);
      await tester.pump();
      expect(enabled(), isFalse);
      expect(controller.calls, ['12345678']);

      await tester.runAsync(() async {
        controller.completion.complete();
        await controller.completion.future;
      });
      await tester.pump();
      await tester.pumpAndSettle();
      expect(
        find.text('Password reimpostata: ora puoi accedere'),
        findsOneWidget,
      );
    },
  );
}
