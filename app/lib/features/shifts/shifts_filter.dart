import 'shift_models.dart';

/// I 3 filtri della schermata Turni, applicati client-side sulla lista già
/// scaricata per il range corrente — nessun parametro server-side, il
/// volume di dati è piccolo (poche occorrenze a settimana). Un'occorrenza
/// ha 4 figure indipendenti (vedi ADR-0025 "Aggiornamento"): "Liberi"
/// mostra i turni con ALMENO UNA figura ancora libera, "Miei" quelli dove
/// il chiamante ha ALMENO UNA figura (in attesa o confermata) — il
/// dettaglio per figura resta nella card espansa, il filtro decide solo
/// quali card mostrare.
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
          .where(
            (o) => o.roles.any((rc) => rc.status == ShiftOccurrenceStatus.free),
          )
          .toList();
    case ShiftsFilter.mine:
      return items
          .where((o) => o.roles.any((rc) => rc.myBookingStatus != null))
          .toList();
  }
}
