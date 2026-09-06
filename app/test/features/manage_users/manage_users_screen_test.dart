import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/api_client.dart';
import 'package:sanitas_app/core/auth/auth_controller.dart';
import 'package:sanitas_app/core/auth/auth_state.dart';
import 'package:sanitas_app/core/jwt.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/manage_users/manage_users_screen.dart';
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
  }) async {
    tester.view.physicalSize = const Size(500, 1200);
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
                home: const ManageUsersScreen(),
              ),
            ),
          ),
        ),
      );
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pumpAndSettle();
  }

  testWidgets('groups users by role and replaces roles on save', (
    tester,
  ) async {
    final requests = <RequestOptions>[];
    final dio = Dio();
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          if (options.path == '/v1/roles') {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: [
                  {'slug': 'president', 'display_name': 'Presidente'},
                  {'slug': 'base_volunteer', 'display_name': 'Volontario base'},
                ],
              ),
            );
            return;
          }
          if (options.path == '/v1/users' && options.method == 'GET') {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: [
                  {
                    'id': 'u1',
                    'username': 'mario',
                    'email': 'mario@example.org',
                    'roles': ['president'],
                  },
                  {
                    'id': 'u2',
                    'username': 'giulia',
                    'email': 'giulia@example.org',
                    'roles': [],
                  },
                ],
              ),
            );
            return;
          }
          if (options.path == '/v1/users/u1/roles' &&
              options.method == 'PATCH') {
            handler.resolve(Response(requestOptions: options, statusCode: 200));
            return;
          }
          handler.resolve(Response(requestOptions: options, statusCode: 200));
        },
      ),
    );

    await mount(tester, dio);

    expect(find.text('PRESIDENTE · 1'), findsOneWidget);
    expect(find.text('SENZA RUOLO · 1'), findsOneWidget);
    expect(find.text('mario'), findsOneWidget);
    expect(find.text('giulia'), findsOneWidget);

    // La matitina (non il tocco sulla riga) entra in modifica: mario è nella
    // prima sezione (PRESIDENTE), quindi la sua è la prima matitina.
    await tester.tap(find.byIcon(Icons.edit_outlined).first);
    await tester.pumpAndSettle();

    expect(find.widgetWithText(FilterChip, 'Presidente'), findsOneWidget);
    expect(find.widgetWithText(FilterChip, 'Volontario base'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilterChip, 'Volontario base'));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(FilledButton, 'Salva'));
    await tester.pumpAndSettle();

    final patch = requests.singleWhere((r) => r.method == 'PATCH');
    expect(patch.path, '/v1/users/u1/roles');
    expect(patch.data, {
      'roles': ['president', 'base_volunteer'],
    });
    expect(find.text('Ruoli aggiornati.'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('tapping the row only shows read-only info, not editable chips', (
    tester,
  ) async {
    final dio = Dio();
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (options.path == '/v1/roles') {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: [
                  {'slug': 'president', 'display_name': 'Presidente'},
                ],
              ),
            );
            return;
          }
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: [
                {
                  'id': 'u1',
                  'username': 'mario',
                  'email': 'mario@example.org',
                  'roles': ['president'],
                },
              ],
            ),
          );
        },
      ),
    );

    await mount(tester, dio);
    await tester.tap(find.text('mario'));
    await tester.pumpAndSettle();

    expect(find.text('mario@example.org'), findsOneWidget);
    expect(find.widgetWithText(Chip, 'Presidente'), findsOneWidget);
    expect(find.widgetWithText(FilterChip, 'Presidente'), findsNothing);
    expect(find.widgetWithText(FilledButton, 'Salva'), findsNothing);
  });

  testWidgets('filters the list via the search field', (tester) async {
    final dio = Dio();
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (options.path == '/v1/roles') {
            handler.resolve(
              Response(requestOptions: options, statusCode: 200, data: []),
            );
            return;
          }
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: [
                {
                  'id': 'u1',
                  'username': 'mario',
                  'email': 'mario@example.org',
                  'roles': [],
                },
                {
                  'id': 'u2',
                  'username': 'giulia',
                  'email': 'giulia@example.org',
                  'roles': [],
                },
              ],
            ),
          );
        },
      ),
    );

    await mount(tester, dio);
    expect(find.text('mario'), findsOneWidget);
    expect(find.text('giulia'), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'giu');
    await tester.pumpAndSettle();

    expect(find.text('mario'), findsNothing);
    expect(find.text('giulia'), findsOneWidget);
  });

  testWidgets(
    'users without users:manage can see the list but not the edit pencil',
    (tester) async {
      final dio = Dio();
      dio.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            if (options.path == '/v1/roles') {
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: 200,
                  data: [
                    {'slug': 'president', 'display_name': 'Presidente'},
                  ],
                ),
              );
              return;
            }
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: [
                  {
                    'id': 'u1',
                    'username': 'mario',
                    'email': 'mario@example.org',
                    'roles': ['president'],
                  },
                ],
              ),
            );
          },
        ),
      );

      await mount(tester, dio, allowed: false);

      // La lista è comunque visibile: solo la matitina è nascosta.
      expect(find.text('mario'), findsOneWidget);
      expect(find.byIcon(Icons.edit_outlined), findsNothing);

      // Il tocco sulla riga resta utile: mostra i ruoli in sola lettura.
      await tester.tap(find.text('mario'));
      await tester.pumpAndSettle();
      expect(find.widgetWithText(Chip, 'Presidente'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Salva'), findsNothing);
    },
  );
}
