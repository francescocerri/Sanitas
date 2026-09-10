import 'shift_models.dart';

/// I 3 filtri della schermata Turni, applicati client-side sulla lista già
/// scaricata per il range corrente — nessun parametro server-side, il
/// volume di dati è piccolo (poche occorrenze a settimana).
enum ShiftsFilter { all, free, mine }

List<ShiftOccurrence> filterOccurrences(
  List<ShiftOccurrence> items,
  ShiftsFilter filter,
) {
  switch (filter) {
    case ShiftsFilter.all:
      return items;
    case ShiftsFilter.free:
      return items
          .where((o) => o.status == ShiftOccurrenceStatus.free)
          .toList();
    case ShiftsFilter.mine:
      return items.where((o) => o.myBookingStatus != null).toList();
  }
}
