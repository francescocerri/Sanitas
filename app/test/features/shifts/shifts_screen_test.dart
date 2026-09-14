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
import 'package:sanitas_app/features/shifts/occurrence_card.dart';
import 'package:sanitas_app/features/shifts/shift_models.dart';
import 'package:sanitas_app/features/shifts/shifts_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:table_calendar/table_calendar.dart';

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
  String? volunteerId,
}) => {
  'role': role,
  'status': status,
  'my_booking_status': ?myBookingStatus,
  'volunteer_id': ?volunteerId,
};

/// Un backend `registry` finto — solo `GET /v1/users`, per risolvere il
/// nome di chi è confermato su una figura (vedi
/// `OccurrenceCard`/`usersProvider`). Vuoto di default: le figure
/// confermate senza un utente corrispondente qui ricadono sul testo
/// generico "Completo", comportamento già atteso dai test che non se ne
/// occupano.
Dio _fakeRegistryDio({List<Map<String, dynamic>> users = const []}) {
  final dio = Dio();
  dio.interceptors.add(
    InterceptorsWrapper(
      onRequest: (options, handler) {
        if (options.path == '/v1/users') {
          handler.resolve(
            Response(requestOptions: options, statusCode: 200, data: users),
          );
          return;
        }
        handler.resolve(Response(requestOptions: options, statusCode: 200));
      },
    ),
  );
  return dio;
}

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
                    _role('driver', 'confirmed', volunteerId: 'other-vol'),
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
    Dio? registryDio,
    // Stretto (mobile) di default, come sempre — solo il test dedicato
    // allo schermo largo lo sovrascrive.
    Size size = const Size(500, 1200),
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    await tester.runAsync(() async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authControllerProvider.overrideWith(() => _Auth(permissions)),
            shiftsDioProvider.overrideWithValue(dio),
            apiDioProvider.overrideWithValue(registryDio ?? _fakeRegistryDio()),
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

      // Il badge "Confermato" (senza conteggio) compare sulla riga della
      // figura autista (visibile solo da espansa) — la legenda usa un testo
      // diverso ("Confermato per te"), niente collisione. Il riepilogo
      // della card mostra invece la variante con quante figure OPERATIVE
      // restano libere (leader/rescuer = 2, l'osservatore libero non conta),
      // non il semplice "Confermato".
      expect(find.text('Confermato'), findsOneWidget);
      expect(find.text('Confermato · 2/3 libero'), findsOneWidget);
      // Le altre 3 figure di tpl-mine sono libere e prenotabili.
      expect(find.byType(Checkbox), findsNWidgets(3));

      await tester.tap(find.text('Turno Mattina')); // tpl-confirmed
      await tester.pumpAndSettle();

      // tpl-confirmed è tutto confermato da altri: nessuna checkbox in
      // più, nessun badge "Confermato" in più (non è mio).
      expect(find.byType(Checkbox), findsNWidgets(3));
      expect(find.text('Confermato'), findsOneWidget);
      expect(find.text('Confermato · 2/3 libero'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'shows the resolved volunteer name for a role confirmed by someone '
    'else, but the generic label while any of it is only pending',
    (tester) async {
      final requests = <RequestOptions>[];
      await mount(
        tester,
        _fakeShiftsDio(requests),
        permissions: const ['shifts:read'],
        registryDio: _fakeRegistryDio(
          users: [
            {
              'id': 'other-vol',
              'username': 'giuliarossi',
              'email': 'giulia@example.org',
              'roles': <String>[],
            },
          ],
        ),
      );

      await tester.tap(find.text('Lista'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Turno Mattina')); // tpl-confirmed
      await tester.pumpAndSettle();

      // L'autista ha un volunteer_id risolvibile: si vede il nome.
      expect(find.text('Confermato: giuliarossi'), findsOneWidget);
      // Le altre 3 figure sono confermate ma senza un utente corrispondente
      // nel fake registry: ricadono sul testo generico "Completo", mai su
      // un id grezzo — 5 in tutto: la legenda (1), il riepilogo della card
      // (1, il turno è al completo) e le 3 righe-figura senza nome.
      expect(find.text('Completo'), findsNWidgets(5));
      expect(find.textContaining('other-vol'), findsNothing);
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

  testWidgets(
    'no checkbox (nor "Assegna" button) on a past occurrence, even if a '
    'role is otherwise free — the backend always rejects a booking on a '
    'past date, and offering the checkbox there used to fail silently with '
    'just the generic "request failed" message',
    (tester) async {
      final requests = <RequestOptions>[];
      await mount(
        tester,
        _fakeShiftsDio(requests),
        permissions: const ['shifts:read', 'shifts:request'],
      );

      await tester.tap(find.text('Giorno'));
      await tester.pumpAndSettle();
      // Torna indietro di un giorno: la stessa occorrenza tpl-free (4
      // figure libere) viene comunque restituita dal fake backend, ma per
      // ieri.
      await tester.tap(find.byIcon(Icons.chevron_left_rounded));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Turno Serale')); // tpl-free, expand
      await tester.pumpAndSettle();

      expect(find.byType(Checkbox), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'a past occurrence with operational_status "closed" shows "Chiuso" '
    'with an X icon on the closed-card summary, overriding both the "mine" '
    'badge (the caller had a confirmed role there) and the ordinary '
    'libero/in attesa aggregate — the objective outcome wins once the '
    'shift is over',
    (tester) async {
      final requests = <RequestOptions>[];
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
                      'template_id': 'tpl-closed',
                      'date': '${from}T00:00:00Z',
                      'weekday': 4,
                      'start_time': '20:00',
                      'end_time': '08:00',
                      'label': 'Turno Passato',
                      // Chiuso: manca l'autista (solo leader confermato,
                      // dal chiamante stesso) — deve vincere su "Confermato
                      // per te".
                      'operational_status': 'closed',
                      'roles': [
                        _role('driver', 'free'),
                        _role(
                          'leader',
                          'confirmed',
                          myBookingStatus: 'confirmed',
                        ),
                        _role('rescuer', 'free'),
                        _role('observer', 'free'),
                      ],
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
      await mount(tester, dio, permissions: const ['shifts:read']);

      await tester.tap(find.text('Lista'));
      await tester.pumpAndSettle();

      expect(find.text('Chiuso'), findsOneWidget);
      expect(find.byIcon(Icons.close_rounded), findsOneWidget);
      // "Confermato per te" compare comunque nella legenda (sempre
      // visibile, indipendente dai dati) — quello che conta è che il
      // riepilogo della card NON lo usi per questa occorrenza chiusa.
      expect(find.text('Confermato'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

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

  testWidgets('tapping "Oggi" brings the Giorno view back to today after '
      'navigating away', (tester) async {
    final requests = <RequestOptions>[];
    await mount(
      tester,
      _fakeShiftsDio(requests),
      permissions: const ['shifts:read'],
    );

    await tester.tap(find.text('Giorno'));
    await tester.pumpAndSettle();

    final todayLabel = DateFormat.yMMMMEEEEd('it').format(DateTime.now());
    expect(find.text(todayLabel), findsOneWidget);

    await tester.tap(find.byIcon(Icons.chevron_right_rounded));
    await tester.pumpAndSettle();
    expect(find.text(todayLabel), findsNothing);

    await tester.tap(find.text('Oggi'));
    await tester.pumpAndSettle();
    expect(find.text(todayLabel), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'the "Liberi" filter hides a shift once driver/leader/rescuer are all '
    'confirmed, even if the observer role is still free',
    (tester) async {
      final requests = <RequestOptions>[];
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
                      'template_id': 'tpl-open',
                      'date': '${from}T00:00:00Z',
                      'weekday': 4,
                      'start_time': '20:00',
                      'end_time': '08:00',
                      'label': 'Turno Aperto',
                      'roles': [
                        _role('driver', 'free'),
                        _role('leader', 'free'),
                        _role('rescuer', 'free'),
                        _role('observer', 'free'),
                      ],
                    },
                    {
                      'template_id': 'tpl-complete-but-observer',
                      'date': '${from}T00:00:00Z',
                      'weekday': 4,
                      'start_time': '08:00',
                      'end_time': '14:00',
                      'label': 'Turno Quasi Completo',
                      'roles': [
                        _role('driver', 'confirmed'),
                        _role('leader', 'confirmed'),
                        _role('rescuer', 'confirmed'),
                        // Osservatore ancora libero: non deve bastare a
                        // farlo comparire sotto "Liberi", il turno è già
                        // operativo senza — richiesto esplicitamente
                        // dall'utente.
                        _role('observer', 'free'),
                      ],
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
      await mount(tester, dio, permissions: const ['shifts:read']);

      await tester.tap(find.text('Lista'));
      await tester.pumpAndSettle();
      expect(find.text('Turno Aperto'), findsOneWidget);
      expect(find.text('Turno Quasi Completo'), findsOneWidget);

      await tester.tap(find.text('Liberi'));
      await tester.pumpAndSettle();

      expect(find.text('Turno Aperto'), findsOneWidget);
      expect(find.text('Turno Quasi Completo'), findsNothing);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('the closed card summary shows "In attesa · N/3 libero" when an '
      'operative role is pending, counting only the 3 operative roles (never '
      'the observer) — reproduces the bug where it showed "2/4 libero" '
      'instead, and keeps the still-open count visible instead of just "In '
      'attesa"', (tester) async {
    final requests = <RequestOptions>[];
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
                    'template_id': 'tpl-pending-with-free',
                    'date': '${from}T00:00:00Z',
                    'weekday': 4,
                    'start_time': '20:00',
                    'end_time': '08:00',
                    'label': 'Turno Ambiguo',
                    'roles': [
                      _role('driver', 'confirmed'),
                      // Figura operativa in attesa: deve far vincere "In
                      // attesa" sul riepilogo, non "2/4 libero".
                      _role('leader', 'pending'),
                      _role('rescuer', 'free'),
                      // Libero ma è l'osservatore: non deve contare nel
                      // conteggio del riepilogo ("1/3", non "2/3").
                      _role('observer', 'free'),
                    ],
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
    await mount(tester, dio, permissions: const ['shifts:read']);

    await tester.tap(find.text('Lista'));
    await tester.pumpAndSettle();

    // La legenda mostra sempre "In attesa" da sola (1); il riepilogo
    // della card chiusa mostra la variante con il conteggio dei soli
    // liberi OPERATIVI rimasti (solo rescuer, l'osservatore libero non
    // conta) — le due stringhe sono diverse, niente collisione da
    // contare insieme.
    expect(find.text('In attesa'), findsOneWidget);
    expect(find.text('In attesa · 1/3 libero'), findsOneWidget);
    expect(find.text('2/4 libero'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('on a wide screen (desktop breakpoint) the Mese day detail shows '
      'occurrence cards two per row instead of stacked one under the other '
      '— fixes the calendar feeling too small/stretched on a laptop, '
      'reported by the user', (tester) async {
    final requests = <RequestOptions>[];
    await mount(
      tester,
      _fakeShiftsDio(requests),
      permissions: const ['shifts:read'],
      size: const Size(1400, 900),
    );

    // Mese è già la vista di default. Tutte e 3 le occorrenze del fake
    // backend cadono il primo giorno del range interrogato (vedi
    // `_fakeShiftsDio`), cioè il 1° del mese mostrato — con
    // `outsideDaysVisible: false` è l'unico "1" nella griglia.
    await tester.tap(
      find
          .descendant(
            of: find.byType(TableCalendar<ShiftOccurrence>),
            matching: find.text('1'),
          )
          .first,
    );
    await tester.pumpAndSettle();

    final cards = find.byType(OccurrenceCard);
    expect(cards, findsNWidgets(3));
    final firstTop = tester.getTopLeft(cards.at(0));
    final secondTop = tester.getTopLeft(cards.at(1));
    final thirdTop = tester.getTopLeft(cards.at(2));
    // Prime due card affiancate (stessa riga)...
    expect(firstTop.dy, secondTop.dy);
    expect(secondTop.dx, greaterThan(firstTop.dx));
    // ...la terza va a capo sulla riga successiva.
    expect(thirdTop.dy, greaterThan(firstTop.dy));
    expect(tester.takeException(), isNull);
  });
}
