import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'shift_models.dart';

/// Colore per la "categoria visiva" di una figura — sempre
/// `myBookingStatus` se presente (è una MIA prenotazione, indipendentemente
/// da chi altro ha richiesto la stessa figura), altrimenti lo stato
/// aggregato. Sia "mia in attesa" che "mia confermata" usano il colore del
/// comitato (per distinguerle da chiunque altro): [roleIsOutlineOnly]
/// dice a chi disegna il pallino se riempirlo (confermata) o solo
/// bordarlo (in attesa) — altrimenti le due sarebbero visivamente
/// identiche, cosa segnalata esplicitamente come confusa.
///
/// I colori "successo/attenzione" non fanno parte della palette del
/// comitato (`CommitteeTheme` ha solo primary/secondary/surface, vedi
/// `core/theme/committee_theme.dart`): sono un linguaggio semantico
/// generico (libero=verde, in attesa=ambra), non un vincolo di brand, quindi
/// restano costanti Material qui invece di finire nel tema per-comitato.
Color roleColor(BuildContext context, RoleCoverage coverage) =>
    statusColor(context, coverage.status, coverage.myBookingStatus);

/// Stessa logica di [roleColor], ma sui soli stati invece che su una
/// `RoleCoverage` intera — usata anche dalla legenda della schermata, che
/// non ha (né le serve) una vera figura per ogni voce.
Color statusColor(
  BuildContext context,
  ShiftOccurrenceStatus status,
  MyBookingStatus? mine,
) {
  if (mine != null) {
    return Theme.of(context).colorScheme.primary;
  }
  switch (status) {
    case ShiftOccurrenceStatus.free:
      return Colors.green.shade600;
    case ShiftOccurrenceStatus.pending:
      return Colors.amber.shade700;
    case ShiftOccurrenceStatus.confirmed:
      return Theme.of(context).colorScheme.onSurfaceVariant;
  }
}

/// true solo per una MIA prenotazione ancora in attesa: chi disegna un
/// pallino/marcatore lo rende un anello vuoto invece che pieno, così "in
/// attesa" (contorno) e "confermato" (pieno) restano distinguibili anche
/// quando condividono lo stesso colore primario.
bool roleIsOutlineOnly(RoleCoverage coverage) =>
    coverage.myBookingStatus == MyBookingStatus.pending;

/// [volunteerName] è lo username già risolto di `coverage.volunteerId`
/// (solo per una figura confermata non mia, vedi
/// `RoleCoverage.volunteerId`) — se assente (figura non confermata, o
/// ancora in attesa di risoluzione) si ricade sul testo generico "Completo".
String roleStatusLabel(RoleCoverage coverage, {String? volunteerName}) {
  switch (coverage.myBookingStatus) {
    case MyBookingStatus.pending:
      return 'shifts.status_mine_pending'.tr();
    case MyBookingStatus.confirmed:
      return 'shifts.status_mine_confirmed'.tr();
    case null:
      break;
  }
  if (coverage.status == ShiftOccurrenceStatus.confirmed &&
      volunteerName != null) {
    return 'shifts.status_confirmed_by'.tr(namedArgs: {'name': volunteerName});
  }
  switch (coverage.status) {
    case ShiftOccurrenceStatus.free:
      return 'shifts.status_free'.tr();
    case ShiftOccurrenceStatus.pending:
      return 'shifts.status_pending'.tr();
    case ShiftOccurrenceStatus.confirmed:
      return 'shifts.status_confirmed'.tr();
  }
}

String roleLabel(ShiftRole role) {
  switch (role) {
    case ShiftRole.driver:
      return 'shifts.role_driver'.tr();
    case ShiftRole.leader:
      return 'shifts.role_leader'.tr();
    case ShiftRole.rescuer:
      return 'shifts.role_rescuer'.tr();
    case ShiftRole.observer:
      return 'shifts.role_observer'.tr();
  }
}

IconData roleIcon(ShiftRole role) {
  switch (role) {
    case ShiftRole.driver:
      return Icons.airport_shuttle_outlined;
    case ShiftRole.leader:
      return Icons.flag_outlined;
    case ShiftRole.rescuer:
      return Icons.medical_services_outlined;
    case ShiftRole.observer:
      return Icons.visibility_outlined;
  }
}

/// Il mio stato aggregato su un'intera occorrenza: confermato se ho una
/// figura confermata (a prescindere dalle altre), altrimenti in attesa se
/// ne ho una in attesa, altrimenti nessuno — un volontario ha al più una
/// figura per occorrenza (vedi ADR-0025 "Aggiornamento"), quindi al più uno
/// di questi due casi si applica mai.
MyBookingStatus? myAggregateStatus(ShiftOccurrence occurrence) {
  for (final rc in occurrence.roles) {
    if (rc.myBookingStatus == MyBookingStatus.confirmed) {
      return MyBookingStatus.confirmed;
    }
  }
  for (final rc in occurrence.roles) {
    if (rc.myBookingStatus == MyBookingStatus.pending) {
      return MyBookingStatus.pending;
    }
  }
  return null;
}

/// Stato aggettivo del turno nel suo complesso — indipendente da chi guarda
/// (a differenza di [myAggregateStatus]), calcolato in un unico punto e
/// riusato identico dal pallino Mese ([occurrenceCardColor]) e dal
/// riepilogo della card chiusa in tutte le viste (vedi
/// `occurrenceSummaryLabel`). `pending` vince SEMPRE su `free` non appena
/// una delle 3 figure operative (vedi [requiredRolesForCompletion]) è in
/// attesa, anche se resta un'altra figura libera altrove: prima di questa
/// unificazione un turno con 2 figure confermate e 1 in attesa poteva
/// ancora apparire "libero" solo perché l'osservatore lo era — bug
/// segnalato esplicitamente dall'utente (vedi ADR-0025 "Aggiornamento
/// (stato aggregato del turno...)").
enum OccurrenceAggregateStatus { free, pending, complete }

OccurrenceAggregateStatus aggregateStatus(ShiftOccurrence occurrence) {
  if (occurrence.isComplete) return OccurrenceAggregateStatus.complete;
  final anyOperativePending = occurrence.roles
      .where((rc) => requiredRolesForCompletion.contains(rc.role))
      .any((rc) => rc.status == ShiftOccurrenceStatus.pending);
  return anyOperativePending
      ? OccurrenceAggregateStatus.pending
      : OccurrenceAggregateStatus.free;
}

/// Colore del singolo pallino aggregato per turno usato dalla vista Mese
/// (un pallino per turno, non uno per figura — il dettaglio per figura
/// resta nella card espansa). Priorità: [ShiftOccurrence.operationalStatus]
/// (se presente, l'occorrenza è passata: l'esito oggettivo — si è svolta o
/// no — conta più di chi ci fosse) vince su tutto; altrimenti una mia
/// figura confermata o in attesa vince sul resto; altrimenti segue
/// [aggregateStatus]. "Ridotto" usa un blu dedicato (non fa parte della
/// palette per-comitato, stesso principio di verde/ambra qui sopra);
/// "chiuso" riusa il grigio di "completo" — la X viene disegnata sopra da
/// chi consuma [occurrenceCardClosedMarker], il colore da solo non basta a
/// distinguerli.
Color occurrenceCardColor(BuildContext context, ShiftOccurrence occurrence) {
  final operational = occurrence.operationalStatus;
  if (operational != null) {
    switch (operational) {
      case ShiftOperationalStatus.complete:
      case ShiftOperationalStatus.closed:
        return Theme.of(context).colorScheme.onSurfaceVariant;
      case ShiftOperationalStatus.reduced:
        return Colors.blue.shade600;
    }
  }
  if (myAggregateStatus(occurrence) != null) {
    return Theme.of(context).colorScheme.primary;
  }
  switch (aggregateStatus(occurrence)) {
    case OccurrenceAggregateStatus.complete:
      return Theme.of(context).colorScheme.onSurfaceVariant;
    case OccurrenceAggregateStatus.pending:
      return Colors.amber.shade700;
    case OccurrenceAggregateStatus.free:
      return Colors.green.shade600;
  }
}

/// true solo quando l'aggregato è "mia richiesta in attesa" — stesso
/// trattamento anello-vs-pieno di [roleIsOutlineOnly], applicato al
/// pallino per turno invece che a quello per figura. Mai true su
/// un'occorrenza passata (`operationalStatus` non nullo): l'esito
/// oggettivo prende il posto di "mio" anche qui, niente anello.
bool occurrenceCardOutline(ShiftOccurrence occurrence) =>
    occurrence.operationalStatus == null &&
    myAggregateStatus(occurrence) == MyBookingStatus.pending;

/// true solo per un'occorrenza passata risultata "chiusa" (autista o
/// leader non confermati) — chi disegna il pallino Mese ci sovrappone una
/// piccola X, il colore da solo (uguale a "completo") non li distingue.
bool occurrenceCardClosedMarker(ShiftOccurrence occurrence) =>
    occurrence.operationalStatus == ShiftOperationalStatus.closed;

/// Testo del riepilogo compatto di una card chiusa (tutte le viste: Mese,
/// Settimana, Giorno, Lista) — stessa priorità di [occurrenceCardColor],
/// così pallino e card non divergono mai. [openCount] è precalcolato dal
/// chiamante e conta SOLO le figure operative libere (vedi
/// `requiredRolesForCompletion`, mai l'osservatore — un "3/3 libero" con
/// l'osservatore ancora libero sarebbe fuorviante): usato dal caso `free`
/// ("N/3 libero") ma anche da `pending` e dai due casi "mio" (in
/// attesa/confermato per me), per non far sparire quante figure operative
/// restano comunque libere — richiesto esplicitamente dall'utente dopo il
/// fix della priorità pending-su-libero: sapere "sono confermato" o "c'è
/// una decisione in sospeso" non deve nascondere che restano altri posti
/// da coprire.
String occurrenceSummaryLabel(
  ShiftOccurrence occurrence, {
  required int openCount,
}) {
  final mine = myAggregateStatus(occurrence);
  if (mine == MyBookingStatus.pending) {
    return openCount > 0
        ? 'shifts.card_pending_count'.tr(namedArgs: {'count': '$openCount'})
        : 'shifts.mine_badge_pending'.tr();
  }
  if (mine == MyBookingStatus.confirmed) {
    return openCount > 0
        ? 'shifts.card_mine_confirmed_count'.tr(
            namedArgs: {'count': '$openCount'},
          )
        : 'shifts.mine_badge_confirmed'.tr();
  }
  switch (aggregateStatus(occurrence)) {
    case OccurrenceAggregateStatus.complete:
      return 'shifts.status_confirmed'.tr();
    case OccurrenceAggregateStatus.pending:
      return openCount > 0
          ? 'shifts.card_pending_count'.tr(namedArgs: {'count': '$openCount'})
          : 'shifts.status_pending'.tr();
    case OccurrenceAggregateStatus.free:
      return 'shifts.card_open_count'.tr(namedArgs: {'count': '$openCount'});
  }
}

/// Sfondo, testo, icona opzionale ed etichetta del badge di riepilogo
/// della card chiusa (tutte le viste) — un solo punto che decide TUTTO lo
/// stile del badge, così `occurrence_card.dart` si limita a disegnarlo.
/// Se `operationalStatus` è presente (occorrenza passata) ha priorità
/// assoluta, stesso principio di [occurrenceCardColor]: niente più "mio"
/// né libero/in attesa, solo l'esito oggettivo — "chiuso" è l'unico caso
/// con un'icona (una X, richiesta esplicitamente dall'utente nel mockup
/// approvato). Altrimenti ricade sulla stessa logica "mio"/[aggregateStatus]
/// di sempre via [occurrenceSummaryLabel].
typedef OccurrenceSummaryBadge = ({
  Color background,
  Color foreground,
  IconData? icon,
  String label,
});

OccurrenceSummaryBadge occurrenceSummaryBadge(
  BuildContext context,
  ShiftOccurrence occurrence, {
  required int openCount,
}) {
  final theme = Theme.of(context);
  final operational = occurrence.operationalStatus;
  if (operational != null) {
    switch (operational) {
      case ShiftOperationalStatus.complete:
        return (
          background: theme.colorScheme.surfaceContainerHighest,
          foreground: theme.colorScheme.onSurfaceVariant,
          icon: null,
          label: 'shifts.status_confirmed'.tr(),
        );
      case ShiftOperationalStatus.reduced:
        return (
          background: Colors.blue.shade50,
          foreground: Colors.blue.shade800,
          icon: null,
          label: 'shifts.status_reduced'.tr(),
        );
      case ShiftOperationalStatus.closed:
        return (
          background: Colors.red.shade50,
          foreground: Colors.red.shade800,
          icon: Icons.close_rounded,
          label: 'shifts.status_closed'.tr(),
        );
    }
  }
  final mine = myAggregateStatus(occurrence);
  return (
    background: mine != null
        ? theme.colorScheme.primaryContainer
        : theme.colorScheme.surfaceContainerHighest,
    foreground: mine != null
        ? theme.colorScheme.onPrimaryContainer
        : theme.colorScheme.onSurfaceVariant,
    icon: null,
    label: occurrenceSummaryLabel(occurrence, openCount: openCount),
  );
}
