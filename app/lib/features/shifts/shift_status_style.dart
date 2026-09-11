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

String roleStatusLabel(RoleCoverage coverage) {
  switch (coverage.myBookingStatus) {
    case MyBookingStatus.pending:
      return 'shifts.status_mine_pending'.tr();
    case MyBookingStatus.confirmed:
      return 'shifts.status_mine_confirmed'.tr();
    case null:
      break;
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

/// Colore del singolo pallino aggregato per turno usato dalla vista Mese
/// (un pallino per turno, non uno per figura — il dettaglio per figura
/// resta nella card espansa). Priorità: una mia figura confermata o in
/// attesa vince su tutto; altrimenti il turno è "al completo" (grigio, vedi
/// `ShiftOccurrence.isComplete`) appena autista/leader/soccorritore sono
/// confermati, indipendentemente dall'osservatore — solo se non è ancora al
/// completo si guarda se resta almeno una figura libera (verde) o solo
/// figure in attesa (ambra).
Color occurrenceCardColor(BuildContext context, ShiftOccurrence occurrence) {
  if (myAggregateStatus(occurrence) != null) {
    return Theme.of(context).colorScheme.primary;
  }
  if (occurrence.isComplete) {
    return Theme.of(context).colorScheme.onSurfaceVariant;
  }
  if (occurrence.roles.any((rc) => rc.status == ShiftOccurrenceStatus.free)) {
    return Colors.green.shade600;
  }
  if (occurrence.roles.any(
    (rc) => rc.status == ShiftOccurrenceStatus.pending,
  )) {
    return Colors.amber.shade700;
  }
  return Theme.of(context).colorScheme.onSurfaceVariant;
}

/// true solo quando l'aggregato è "mia richiesta in attesa" — stesso
/// trattamento anello-vs-pieno di [roleIsOutlineOnly], applicato al
/// pallino per turno invece che a quello per figura.
bool occurrenceCardOutline(ShiftOccurrence occurrence) =>
    myAggregateStatus(occurrence) == MyBookingStatus.pending;
