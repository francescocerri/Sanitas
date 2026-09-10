import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'shift_models.dart';

/// Colore per la "categoria visiva" di un'occorrenza — sempre
/// `myBookingStatus` se presente (è una MIA prenotazione, indipendentemente
/// da chi altro ha richiesto lo stesso slot), altrimenti lo stato
/// aggregato. Sia "mia in attesa" che "mia confermata" usano il colore del
/// comitato (per distinguerle da chiunque altro): [occurrenceIsOutlineOnly]
/// dice a chi disegna il pallino se riempirlo (confermata) o solo
/// bordarlo (in attesa) — altrimenti le due sarebbero visivamente
/// identiche, cosa segnalata esplicitamente come confusa.
///
/// I colori "successo/attenzione" non fanno parte della palette del
/// comitato (`CommitteeTheme` ha solo primary/secondary/surface, vedi
/// `core/theme/committee_theme.dart`): sono un linguaggio semantico
/// generico (libero=verde, in attesa=ambra), non un vincolo di brand, quindi
/// restano costanti Material qui invece di finire nel tema per-comitato.
Color occurrenceColor(BuildContext context, ShiftOccurrence occurrence) =>
    statusColor(context, occurrence.status, occurrence.myBookingStatus);

/// Stessa logica di [occurrenceColor], ma sui soli stati invece che su
/// un'occorrenza intera — usata anche dalla legenda della schermata, che
/// non ha (né le serve) una vera `ShiftOccurrence` per ogni voce.
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
bool occurrenceIsOutlineOnly(ShiftOccurrence occurrence) =>
    occurrence.myBookingStatus == MyBookingStatus.pending;

String occurrenceStatusLabel(ShiftOccurrence occurrence) {
  switch (occurrence.myBookingStatus) {
    case MyBookingStatus.pending:
      return 'shifts.status_mine_pending'.tr();
    case MyBookingStatus.confirmed:
      return 'shifts.status_mine_confirmed'.tr();
    case null:
      break;
  }
  switch (occurrence.status) {
    case ShiftOccurrenceStatus.free:
      return 'shifts.status_free'.tr();
    case ShiftOccurrenceStatus.pending:
      return 'shifts.status_pending'.tr();
    case ShiftOccurrenceStatus.confirmed:
      return 'shifts.status_confirmed'.tr();
  }
}
