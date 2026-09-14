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
import 'package:sanitas_app/core/shifts_api_client.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/shifts/manage/shift_manager_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _Translations extends AssetLoader {
  const _Translations();
  @override
  Future<Map<String, dynamic>> load(String path, Locale locale) async =>
      jsonDecode(File('$path/${locale.languageCode}.json').readAsStringSync())
          as Map<String, dynamic>;
}

class _Auth extends AuthController {
  @override
  AuthSession build() => AuthSession(
    status: AuthStatus.authenticated,
    claims: JwtClaims(
      subject: 'test-manager',
      username: 'manager',
      roles: const [],
      permissions: const ['shifts:read', 'shifts:write', 'shifts:configure'],
      expiresAt: DateTime(2099),
    ),
  );
}

/// Un backend `registry` finto (solo `GET /v1/users`, per risolvere i nomi
/// dei volontari nella tab Richieste e per il selettore volontario della
/// tab Copertura).
Dio _fakeRegistryDio() {
  final dio = Dio();
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        if (options.path == '/v1/users') {
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: [
                {
                  'id': 'u1',
                  'username': 'mario',
                  'email': 'mario@example.org',
                  'roles': <String>[],
                },
              ],
            ),
          );
          return;
        }
        handler.resolve(Response(requestOptions: options, statusCode: 200));
      },
    ),
  );
  return dio;
}

/// Un backend `shifts` finto — un'occorrenza con la figura autista
/// libera (tab Copertura), un template attivo e uno inattivo (tab
/// Turni-template), una richiesta pending per "mario" (tab Richieste).
Dio _fakeShiftsDio(List<RequestOptions> requests) {
  final dio = Dio();
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        requests.add(options);
        if (options.path == '/v1/shift-occurrences') {
          final from = options.queryParameters['from'] as String;
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: [
                {
                  'template_id': 'tpl-free',
                  'date': '${from}T00:00:00Z',
                  'weekday': 4,
                  'start_time': '20:00',
                  'end_time': '08:00',
                  'label': 'Turno Serale',
                  'roles': [
                    {'role': 'driver', 'status': 'free'},
                    {'role': 'leader', 'status': 'free'},
                    {'role': 'rescuer', 'status': 'free'},
                    {'role': 'observer', 'status': 'free'},
                  ],
                },
              ],
            ),
          );
          return;
        }
        if (options.path == '/v1/shift-bookings/pending') {
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: [
                {
                  'id': 'b1',
                  'template_id': 'tpl-free',
                  'volunteer_id': 'u1',
                  'role': 'leader',
                  'date': '2026-09-10T00:00:00Z',
                  'start_time': '20:00',
                  'end_time': '08:00',
                  'status': 'pending',
                },
              ],
            ),
          );
          return;
        }
        if (options.path == '/v1/shift-templates') {
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: [
                {
                  'id': 'tpl-free',
                  'weekday': 4,
                  'start_time': '20:00',
                  'end_time': '08:00',
                  'label': 'Turno Serale',
                  'active': true,
                },
                {
                  'id': 'tpl-old',
                  'weekday': 0,
                  'start_time': '14:00',
                  'end_time': '20:00',
                  'label': 'Vecchio turno',
                  'active': false,
                },
              ],
            ),
          );
          return;
        }
        if (options.path == '/v1/shift-bookings/direct' ||
            (options.path.startsWith('/v1/shift-bookings/') &&
                options.method == 'PATCH')) {
          handler.resolve(Response(requestOptions: options, statusCode: 200));
          return;
        }
        handler.resolve(Response(requestOptions: options, statusCode: 200));
      },
    ),
  );
  return dio;
}

void main() {
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    SharedPreferences.setMockInitialValues({});
    await EasyLocalization.ensureInitialized();
  });

  Future<void> mount(
    WidgetTester tester,
    Dio shiftsDio,
    Dio registryDio,
  ) async {
    tester.view.physicalSize = const Size(500, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.runAsync(() async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(_Auth.new),
            shiftsDioProvider.overrideWithValue(shiftsDio),
            apiDioProvider.overrideWithValue(registryDio),
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
                home: const ShiftManagerScreen(),
              ),
            ),
          ),
        ),
      );
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pumpAndSettle();
  }

  testWidgets(
    'Copertura tab shows Assegna on a free role and assigns the chosen volunteer',
    (tester) async {
      final requests = <RequestOptions>[];
      await mount(tester, _fakeShiftsDio(requests), _fakeRegistryDio());

      // Tab Copertura è la prima, già selezionata di default: si passa
      // dalla vista Mese (default) alla Lista per non dover toccare un
      // giorno specifico nel calendario.
      await tester.tap(find.text('Lista'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Turno Serale'));
      await tester.pumpAndSettle();

      expect(find.widgetWithText(OutlinedButton, 'Assegna'), findsWidgets);
      // Niente checkbox: la card è in modalità gestore, non selezione.
      expect(find.byType(Checkbox), findsNothing);

      await tester.tap(find.widgetWithText(OutlinedButton, 'Assegna').first);
      await tester.pumpAndSettle();

      expect(find.text('Assegna un volontario'), findsOneWidget);
      await tester.tap(find.text('mario'));
      await tester.pumpAndSettle();

      final direct = requests.singleWhere(
        (r) => r.path == '/v1/shift-bookings/direct',
      );
      final body = direct.data as Map<String, dynamic>;
      expect(body['template_id'], 'tpl-free');
      expect(body['volunteer_id'], 'u1');
      expect(body['role'], 'driver');
      expect(find.text('Volontario assegnato.'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'Richieste tab shows a pending request with resolved volunteer name and '
    'approving calls the decide endpoint',
    (tester) async {
      final requests = <RequestOptions>[];
      await mount(tester, _fakeShiftsDio(requests), _fakeRegistryDio());

      await tester.tap(find.text('Richieste'));
      await tester.pumpAndSettle();

      expect(find.text('mario'), findsOneWidget);
      expect(find.textContaining('Turno Serale'), findsOneWidget);

      await tester.tap(find.widgetWithText(OutlinedButton, 'Approva'));
      await tester.pumpAndSettle();

      final decide = requests.singleWhere(
        (r) => r.path == '/v1/shift-bookings/b1',
      );
      expect(decide.method, 'PATCH');
      expect(decide.data, {'status': 'confirmed'});
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('Turni-template tab dims an inactive template', (tester) async {
    final requests = <RequestOptions>[];
    await mount(tester, _fakeShiftsDio(requests), _fakeRegistryDio());

    await tester.tap(find.text('Turni-template'));
    await tester.pumpAndSettle();

    expect(find.text('Turno Serale'), findsOneWidget);
    expect(find.text('Vecchio turno'), findsOneWidget);
    expect(find.text('Inattivo'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
