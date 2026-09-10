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
