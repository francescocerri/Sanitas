import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sanitas_app/core/auth/auth_controller.dart';
import 'package:sanitas_app/core/auth/auth_state.dart';
import 'package:sanitas_app/core/jwt.dart';
import 'package:sanitas_app/core/shifts_api_client.dart';
import 'package:sanitas_app/core/theme/committee_theme.dart';
import 'package:sanitas_app/features/shifts/shifts_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

class _Translations extends AssetLoader {
  const _Translations();
  @override
  Future<Map<String, dynamic>> load(String path, Locale locale) async =>
      jsonDecode(File('$path/${locale.languageCode}.json').readAsStringSync())
          as Map<String, dynamic>;
}

class _Auth extends AuthController {
  _Auth(this.permissions);
  final List<String> permissions;
  @override
  AuthSession build() => AuthSession(
    status: AuthStatus.authenticated,
    claims: JwtClaims(
      subject: 'test-volunteer',
      username: 'test',
      roles: const [],
      permissions: permissions,
      expiresAt: DateTime(2099),
    ),
  );
}

Map<String, dynamic> _role(
  String role,
  String status, {
  String? myBookingStatus,
}) => {'role': role, 'status': status, 'my_booking_status': ?myBookingStatus};

/// Un backend `shifts` finto che restituisce sempre 3 occorrenze sulla
/// data `from` richiesta (per ogni vista, qualunque sia il range che
/// interroga) — così i test non dipendono dal giorno della settimana in
/// cui girano: non serve calcolare un vero giovedì/sabato futuro.
///
/// - tpl-free: tutte e 4 le figure libere.
/// - tpl-confirmed: tutte e 4 confermate (da un altro volontario) — turno
///   al completo, nessuna checkbox né badge da nessuna parte.
/// - tpl-mine: autista confermato per il chiamante, le altre 3 libere.
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
                    _role('driver', 'free'),
                    _role('leader', 'free'),
                    _role('rescuer', 'free'),
                    _role('observer', 'free'),
                  ],
                },
                {
                  'template_id': 'tpl-confirmed',
                  'date': '${from}T00:00:00Z',
                  'weekday': 4,
                  'start_time': '08:00',
                  'end_time': '14:00',
                  'label': 'Turno Mattina',
                  'roles': [
                    _role('driver', 'confirmed'),
                    _role('leader', 'confirmed'),
                    _role('rescuer', 'confirmed'),
                    _role('observer', 'confirmed'),
                  ],
                },
                {
                  'template_id': 'tpl-mine',
                  'date': '${from}T00:00:00Z',
                  'weekday': 4,
                  'start_time': '14:00',
                  'end_time': '20:00',
                  'label': 'Turno Pomeriggio',
                  'roles': [
                    _role('driver', 'confirmed', myBookingStatus: 'confirmed'),
                    _role('leader', 'free'),
                    _role('rescuer', 'free'),
                    _role('observer', 'free'),
                  ],
                },
              ],
            ),
          );
          return;
        }
        if (options.path == '/v1/shift-bookings/bulk') {
          handler.resolve(Response(requestOptions: options, statusCode: 201));
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
    Dio dio, {
    required List<String> permissions,
  }) async {
    tester.view.physicalSize = const Size(500, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.runAsync(() async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(() => _Auth(permissions)),
            shiftsDioProvider.overrideWithValue(dio),
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
                home: const ShiftsScreen(),
              ),
            ),
          ),
        ),
      );
      await Future<void>.delayed(const Duration(milliseconds: 100));
    });
    await tester.pumpAndSettle();
  }

  testWidgets('read-only for a volunteer without shifts:request', (
    tester,
  ) async {
    final requests = <RequestOptions>[];
    await mount(
      tester,
      _fakeShiftsDio(requests),
      permissions: const ['shifts:read'],
    );

    await tester.tap(find.text('Lista'));
    await tester.pumpAndSettle();
    // Espande la card completamente libera: senza shifts:request non deve
    // comparire nessuna checkbox, nemmeno lì.
    await tester.tap(find.text('Turno Serale'));
    await tester.pumpAndSettle();

    expect(find.byType(Checkbox), findsNothing);
    expect(find.text('Invia richiesta'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'shows a badge (not a checkbox) for the volunteer\'s own confirmed '
    'role, and neither for a role confirmed by someone else',
    (tester) async {
      final requests = <RequestOptions>[];
      await mount(
        tester,
        _fakeShiftsDio(requests),
        permissions: const ['shifts:read', 'shifts:request'],
      );

      await tester.tap(find.text('Lista'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Turno Pomeriggio')); // tpl-mine
      await tester.pumpAndSettle();

      // Il badge "Confermato" compare 2 volte per tpl-mine: nel riepilogo
      // della card (sempre visibile) e sulla riga della figura autista
      // (visibile solo da espansa) — la legenda usa un testo diverso
      // ("Confermato per te"), niente collisione.
      expect(find.text('Confermato'), findsNWidgets(2));
      // Le altre 3 figure di tpl-mine sono libere e prenotabili.
      expect(find.byType(Checkbox), findsNWidgets(3));

      await tester.tap(find.text('Turno Mattina')); // tpl-confirmed
      await tester.pumpAndSettle();

      // tpl-confirmed è tutto confermato da altri: nessuna checkbox in
      // più, nessun badge "Confermato" in più (non è mio).
      expect(find.byType(Checkbox), findsNWidgets(3));
      expect(find.text('Confermato'), findsNWidgets(2));
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('selecting a free role and submitting calls the bulk endpoint', (
    tester,
  ) async {
    final requests = <RequestOptions>[];
    await mount(
      tester,
      _fakeShiftsDio(requests),
      permissions: const ['shifts:read', 'shifts:request'],
    );

    await tester.tap(find.text('Lista'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Turno Serale')); // tpl-free, expand
    await tester.pumpAndSettle();

    // Il primo checkbox è quello dell'autista (primo in ShiftRole.values).
    await tester.tap(find.byType(Checkbox).first);
    await tester.pumpAndSettle();

    expect(find.text('1 posizione selezionata'), findsOneWidget);
    expect(find.text('Invia richiesta'), findsOneWidget);

    await tester.tap(find.text('Invia richiesta'));
    await tester.pumpAndSettle();

    final bulk = requests.singleWhere(
      (r) => r.path == '/v1/shift-bookings/bulk',
    );
    final body = bulk.data as Map<String, dynamic>;
    final bookings = body['bookings'] as List<dynamic>;
    expect(bookings, hasLength(1));
    expect((bookings.first as Map)['template_id'], 'tpl-free');
    expect((bookings.first as Map)['role'], 'driver');

    expect(find.text('Richiesta inviata.'), findsOneWidget);
    expect(find.text('Invia richiesta'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('selecting one role locks the other roles on the same slot '
      '(a volunteer can only hold one role per occurrence)', (tester) async {
    final requests = <RequestOptions>[];
    await mount(
      tester,
      _fakeShiftsDio(requests),
      permissions: const ['shifts:read', 'shifts:request'],
    );

    await tester.tap(find.text('Lista'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Turno Serale')); // tpl-free, expand
    await tester.pumpAndSettle();

    final before = tester.widgetList<Checkbox>(find.byType(Checkbox)).toList();
    expect(before, hasLength(4));
    expect(before.every((c) => c.onChanged != null), isTrue);

    await tester.tap(find.byType(Checkbox).first);
    await tester.pumpAndSettle();

    final afterSelect = tester
        .widgetList<Checkbox>(find.byType(Checkbox))
        .toList();
    expect(afterSelect, hasLength(4));
    // Quella appena selezionata resta interattiva (per poterla
    // deselezionare); le altre 3 sulla stessa occorrenza sono bloccate.
    expect(afterSelect.first.onChanged, isNotNull);
    expect(afterSelect.skip(1).every((c) => c.onChanged == null), isTrue);

    // Deselezionandola si sblocca di nuovo tutto.
    await tester.tap(find.byType(Checkbox).first);
    await tester.pumpAndSettle();
    final afterDeselect = tester
        .widgetList<Checkbox>(find.byType(Checkbox))
        .toList();
    expect(afterDeselect.every((c) => c.onChanged != null), isTrue);

    expect(tester.takeException(), isNull);
  });
}
