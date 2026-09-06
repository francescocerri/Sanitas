import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/api_client.dart';
import 'package:sanitas_app/core/auth/auth_controller.dart';
import 'package:sanitas_app/core/auth/auth_state.dart';
import 'package:sanitas_app/core/jwt.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/create_user/create_user_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _Translations extends AssetLoader {
  const _Translations();
  @override
  Future<Map<String, dynamic>> load(String path, Locale locale) async =>
      jsonDecode(File('$path/${locale.languageCode}.json').readAsStringSync())
          as Map<String, dynamic>;
}

class _Auth extends AuthController {
  _Auth(this.allowed);
  final bool allowed;
  @override
  AuthSession build() => AuthSession(
    status: AuthStatus.authenticated,
    claims: JwtClaims(
      subject: 'test-user',
      username: 'test',
      roles: const [],
      permissions: allowed ? ['users:manage'] : [],
      expiresAt: DateTime(2099),
    ),
  );
}

void main() {
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    SharedPreferences.setMockInitialValues({});
    await EasyLocalization.ensureInitialized();
  });

  Future<void> mount(
    WidgetTester tester,
    Dio dio, {
    bool allowed = true,
    double width = 1000,
  }) async {
    tester.view.physicalSize = Size(width, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.runAsync(() async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(() => _Auth(allowed)),
            apiDioProvider.overrideWithValue(dio),
          ],
          child: EasyLocalization(
            supportedLocales: const [Locale('it')],
            startLocale: const Locale('it'),
            path: 'assets/translations',
            assetLoader: const _Translations(),
            child: Builder(
              builder: (context) => MaterialApp(
                localizationsDelegates: context.localizationDelegates,
                supportedLocales: context.supportedLocales,
                locale: context.locale,
                theme:
                    const CommitteeTheme(
                          committeeName: 'Test',
                          defaultLocale: 'it',
                          primary: Colors.red,
                          secondary: Colors.black,
                          surface: Colors.white,
                        )
                        .toThemeData(Brightness.light)
                        .copyWith(textTheme: ThemeData.light().textTheme),
                home: const CreateUserScreen(),
              ),
            ),
          ),
        ),
      );
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pumpAndSettle();
  }

  for (final width in [390.0, 1000.0]) {
    testWidgets(
      'creates user with backend role slugs and copies invite at $width px',
      (tester) async {
        final dio = Dio();
        final requests = <RequestOptions>[];
        dio.interceptors.add(
          InterceptorsWrapper(
            onRequest: (options, handler) {
              requests.add(options);
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: options.method == 'GET' ? 200 : 201,
                  data: options.method == 'GET'
                      ? [
                          {
                            'slug': 'custom_role',
                            'display_name': 'Ruolo di prova',
                          },
                        ]
                      : {
                          'invite_url':
                              'https://example.org/user-activation?token=test',
                        },
                ),
              );
            },
          ),
        );
        await mount(tester, dio, width: width);
        final submit = find.widgetWithText(
          FilledButton,
          'Crea utente e genera link',
        );
        expect(tester.widget<FilledButton>(submit).onPressed, isNull);
        await tester.enterText(find.byType(TextFormField).at(0), 'invalid');
        await tester.enterText(find.byType(TextFormField).at(1), 'test.user');
        await tester.pumpAndSettle();
        expect(tester.widget<FilledButton>(submit).onPressed, isNull);
        await tester.enterText(
          find.byType(TextFormField).at(0),
          'test@example.org',
        );
        await tester.ensureVisible(find.byType(FilterChip));
        await tester.tap(find.byType(FilterChip));
        await tester.pumpAndSettle();
        await tester.ensureVisible(submit);
        await tester.tap(submit);
        await tester.pumpAndSettle();
        final post = requests.singleWhere(
          (request) => request.method == 'POST',
        );
        expect(post.path, '/v1/users');
        expect(post.data, {
          'email': 'test@example.org',
          'username': 'test.user',
          'roles': ['custom_role'],
        });
        expect(find.text('Un nuovo inizio.'), findsOneWidget);
        String? copied;
        tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          SystemChannels.platform,
          (call) async {
            if (call.method == 'Clipboard.setData') {
              copied = (call.arguments as Map)['text'] as String;
            }
            return null;
          },
        );
        addTearDown(
          () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
            SystemChannels.platform,
            null,
          ),
        );
        await tester.tap(find.text('Copia link di attivazione'));
        await tester.pumpAndSettle();
        expect(copied, 'https://example.org/user-activation?token=test');
        await tester.tap(find.text('Crea un altro utente'));
        await tester.pumpAndSettle();
        expect(tester.widget<FilledButton>(submit).onPressed, isNull);
        expect(tester.takeException(), isNull);
      },
    );
  }

  testWidgets('denies users without permission without fetching roles', (
    tester,
  ) async {
    final dio = Dio();
    var requested = false;
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requested = true;
          handler.resolve(Response(requestOptions: options, data: []));
        },
      ),
    );
    await mount(tester, dio, allowed: false);
    expect(
      find.text('Non hai il permesso di gestire gli utenti.'),
      findsOneWidget,
    );
    expect(find.byType(TextFormField), findsNothing);
    expect(requested, isFalse);
  });

  testWidgets('role loading error can retry and duplicate keeps form data', (
    tester,
  ) async {
    final dio = Dio();
    var failRoles = true;
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (options.method == 'GET' && !failRoles) {
            handler.resolve(
              Response(requestOptions: options, statusCode: 200, data: []),
            );
          } else {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.badResponse,
                response: Response(
                  requestOptions: options,
                  statusCode: options.method == 'GET' ? 500 : 409,
                ),
              ),
            );
          }
        },
      ),
    );
    await mount(tester, dio);
    expect(find.text('Riprova'), findsOneWidget);
    failRoles = false;
    await tester.tap(find.text('Riprova'));
    await tester.pumpAndSettle();
    expect(
      find.text(
        'Nessun ruolo disponibile. Puoi creare un account senza ruoli.',
      ),
      findsOneWidget,
    );
    await tester.enterText(
      find.byType(TextFormField).at(0),
      'test@example.org',
    );
    await tester.enterText(find.byType(TextFormField).at(1), 'test.user');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Crea utente e genera link'));
    await tester.pumpAndSettle();
    expect(
      find.text('Email o username già in uso. Controlla i dati inseriti.'),
      findsOneWidget,
    );
    expect(find.text('test@example.org'), findsOneWidget);
  });
}
