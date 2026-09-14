import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';
import '../../core/shifts_api_client.dart';
import 'shift_models.dart';

/// Chi può vedere il calendario dei turni — permesso `shifts:read`, che
/// (nel Comitato di Pavullo) ha ogni ruolo: stesso pattern di
/// `canManageUsersProvider` in `create_user_screen.dart`.
final canViewShiftsProvider = Provider<bool>(
  (ref) => ref.watch(
    authControllerProvider.select(
      (session) => session.claims?.hasPermission('shifts:read') ?? false,
    ),
  ),
);

/// Chi può richiedere di prenotarsi su uno slot — permesso `shifts:request`,
/// solo `emergency_volunteer` nel Comitato di Pavullo. Chi NON ce l'ha vede
/// comunque il calendario (in sola lettura): niente checkbox, niente barra
/// di selezione.
final canRequestShiftsProvider = Provider<bool>(
  (ref) => ref.watch(
    authControllerProvider.select(
      (session) => session.claims?.hasPermission('shifts:request') ?? false,
    ),
  ),
);

/// Chi può approvare/rifiutare richieste e prenotare direttamente un
/// volontario — permesso `shifts:write`, il gestore turni.
final canManageShiftsProvider = Provider<bool>(
  (ref) => ref.watch(
    authControllerProvider.select(
      (session) => session.claims?.hasPermission('shifts:write') ?? false,
    ),
  ),
);

/// Chi può creare/modificare i turni-template — permesso `shifts:configure`,
/// distinto da `shifts:write` (approvare richieste non implica poter
/// cambiare gli orari dei turni, e viceversa).
final canConfigureShiftTemplatesProvider = Provider<bool>(
  (ref) => ref.watch(
    authControllerProvider.select(
      (session) => session.claims?.hasPermission('shifts:configure') ?? false,
    ),
  ),
);

/// Chi vede la schermata "Gestione turni" in home — basta uno dei due
/// permessi da gestore, le singole tab si nascondono/disabilitano da sole
/// in base al permesso specifico (vedi `ShiftManagerScreen`).
final canAccessShiftManagementProvider = Provider<bool>(
  (ref) =>
      ref.watch(canManageShiftsProvider) ||
      ref.watch(canConfigureShiftTemplatesProvider),
);

String _formatDate(DateTime d) =>
    '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

ApiException _shiftsError(DioException error) => ApiException.fromDioException(
  error,
  statusToKey: const {400: 'errors.invalid_payload'},
);

/// Range di date da interrogare — chiave del `family` sotto: ogni vista
/// (giorno/settimana/mese/lista) richiede il proprio intervallo, ciascuno
/// cacheato ed invalidato separatamente da Riverpod.
typedef OccurrenceRange = ({DateTime from, DateTime to});

/// `GET /v1/shift-occurrences?from=&to=` per un range di date. `.autoDispose`
/// perché ogni vista chiede il proprio range: senza, ogni cambio di
/// mese/settimana/giorno lascerebbe indietro provider mai più letti.
final occurrencesProvider = FutureProvider.autoDispose
    .family<List<ShiftOccurrence>, OccurrenceRange>((ref, range) async {
      try {
        final response = await ref
            .watch(shiftsDioProvider)
            .get<List<dynamic>>(
              '/v1/shift-occurrences',
              queryParameters: {
                'from': _formatDate(range.from),
                'to': _formatDate(range.to),
              },
            );
        return response.data!
            .map((e) => ShiftOccurrence.fromJson(e as Map<String, dynamic>))
            .toList();
      } on DioException catch (error) {
        throw _shiftsError(error);
      }
    });

/// `POST /v1/shift-bookings/bulk` — crea tutte le prenotazioni scelte
/// (occorrenza+figura, vedi ADR-0025 "Aggiornamento") in un'unica
/// richiesta atomica (vedi `docs/backlog.md` "Gestione turni", voce 7): o
/// vengono create tutte, o nessuna.
Future<void> requestBulkBookings(
  WidgetRef ref,
  List<BookingSelection> selection,
) async {
  try {
    await ref
        .read(shiftsDioProvider)
        .post<void>(
          '/v1/shift-bookings/bulk',
          data: {
            'bookings': selection
                .map(
                  (s) => {
                    'template_id': s.occurrence.templateId,
                    'date': s.occurrence.dateKey,
                    'role': s.role.name,
                  },
                )
                .toList(),
          },
        );
  } on DioException catch (error) {
    throw _shiftsError(error);
  }
}

/// Una richiesta pending così come la restituisce
/// `GET /v1/shift-bookings/pending` (`shift.Booking` lato backend) — solo i
/// campi che servono alla tab "Richieste": nome volontario e label del
/// turno NON sono qui, il backend resta disaccoppiato da come `registry`
/// formatta un nome (vedi ADR-0025 "Aggiornamento") — la UI li risolve da
/// `usersProvider`/`templatesProvider` per id.
class PendingBooking {
  const PendingBooking({
    required this.id,
    required this.templateId,
    required this.volunteerId,
    required this.role,
    required this.date,
  });

  final String id;
  final String templateId;
  final String volunteerId;
  final ShiftRole role;
  final DateTime date;

  factory PendingBooking.fromJson(Map<String, dynamic> json) {
    final role = parseShiftRole(json['role'] as String);
    if (role == null) {
      throw FormatException('unknown role in response: ${json['role']}');
    }
    return PendingBooking(
      id: json['id'] as String,
      templateId: json['template_id'] as String,
      volunteerId: json['volunteer_id'] as String,
      role: role,
      date: DateTime.parse(json['date'] as String),
    );
  }
}

/// `GET /v1/shift-bookings/pending` — la lista di dettaglio dietro la tab
/// "Richieste" del gestore (`shifts:write`). Il conteggio per il badge
/// della tab è semplicemente `.length` di questa lista: niente chiamata
/// separata a `/pending-count`.
final pendingBookingsProvider =
    FutureProvider.autoDispose<List<PendingBooking>>((ref) async {
      try {
        final response = await ref
            .watch(shiftsDioProvider)
            .get<List<dynamic>>('/v1/shift-bookings/pending');
        return response.data!
            .map((e) => PendingBooking.fromJson(e as Map<String, dynamic>))
            .toList();
      } on DioException catch (error) {
        throw _shiftsError(error);
      }
    });

/// `PATCH /v1/shift-bookings/{id}` — approva o rifiuta una richiesta
/// pending.
Future<void> decideBooking(
  WidgetRef ref,
  String bookingId, {
  required bool approve,
}) async {
  try {
    await ref
        .read(shiftsDioProvider)
        .patch<void>(
          '/v1/shift-bookings/$bookingId',
          data: {'status': approve ? 'confirmed' : 'rejected'},
        );
  } on DioException catch (error) {
    throw _shiftsError(error);
  }
}

/// `POST /v1/shift-bookings/direct` — il gestore prenota direttamente un
/// volontario scelto su una figura, già confermata (nessun passaggio
/// "in attesa").
Future<void> assignVolunteer(
  WidgetRef ref, {
  required String templateId,
  required String volunteerId,
  required String date,
  required ShiftRole role,
}) async {
  try {
    await ref
        .read(shiftsDioProvider)
        .post<void>(
          '/v1/shift-bookings/direct',
          data: {
            'template_id': templateId,
            'volunteer_id': volunteerId,
            'date': date,
            'role': role.name,
          },
        );
  } on DioException catch (error) {
    throw _shiftsError(error);
  }
}

/// Un turno-template così come lo restituisce `GET /v1/shift-templates`
/// (`shift.ShiftTemplate` lato backend) — orario ricorrente settimanale da
/// cui `ListOccurrences` genera le occorrenze concrete sul calendario.
class ShiftTemplateSummary {
  const ShiftTemplateSummary({
    required this.id,
    required this.weekday,
    required this.startTime,
    required this.endTime,
    required this.label,
    required this.active,
  });

  final String id;

  /// 0 = domenica .. 6 = sabato, stessa convenzione di `time.Weekday` in Go.
  final int weekday;
  final String startTime;
  final String endTime;
  final String label;
  final bool active;

  factory ShiftTemplateSummary.fromJson(Map<String, dynamic> json) =>
      ShiftTemplateSummary(
        id: json['id'] as String,
        weekday: json['weekday'] as int,
        startTime: json['start_time'] as String,
        endTime: json['end_time'] as String,
        label: json['label'] as String,
        active: json['active'] as bool,
      );
}

/// `GET /v1/shift-templates` — dietro la tab "Turni-template" del gestore
/// (`shifts:configure`), ma leggibile da chiunque abbia `shifts:read` (lo
/// stesso endpoint che alimenta anche `ListOccurrences` lato backend).
final templatesProvider =
    FutureProvider.autoDispose<List<ShiftTemplateSummary>>((ref) async {
      try {
        final response = await ref
            .watch(shiftsDioProvider)
            .get<List<dynamic>>('/v1/shift-templates');
        return response.data!
            .map(
              (e) => ShiftTemplateSummary.fromJson(e as Map<String, dynamic>),
            )
            .toList();
      } on DioException catch (error) {
        throw _shiftsError(error);
      }
    });

/// `POST /v1/shift-templates`.
Future<void> createTemplate(
  WidgetRef ref, {
  required int weekday,
  required String startTime,
  required String endTime,
  required String label,
}) async {
  try {
    await ref
        .read(shiftsDioProvider)
        .post<void>(
          '/v1/shift-templates',
          data: {
            'weekday': weekday,
            'start_time': startTime,
            'end_time': endTime,
            'label': label,
          },
        );
  } on DioException catch (error) {
    throw _shiftsError(error);
  }
}

/// `PATCH /v1/shift-templates/{id}` — sostituzione completa: il backend
/// richiede sempre tutti i campi insieme, non un patch parziale.
Future<void> updateTemplate(
  WidgetRef ref,
  String id, {
  required int weekday,
  required String startTime,
  required String endTime,
  required String label,
  required bool active,
}) async {
  try {
    await ref
        .read(shiftsDioProvider)
        .patch<void>(
          '/v1/shift-templates/$id',
          data: {
            'weekday': weekday,
            'start_time': startTime,
            'end_time': endTime,
            'label': label,
            'active': active,
          },
        );
  } on DioException catch (error) {
    throw _shiftsError(error);
  }
}
