import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'shift_models.dart';
import 'shift_status_style.dart';

/// Una riga-slot: pallino di stato, label + data + orario + stato
/// testuale, e a destra una checkbox (se prenotabile) o il badge "Tuo" (se
/// è una mia prenotazione) — nient'altro se non prenotabile da nessuno
/// (es. confermato per un altro volontario). Condivisa dalle 4 viste
/// (giorno/settimana/mese/lista): stessa identica riga ovunque, cambia solo
/// come vengono raggruppate.
class OccurrenceRow extends StatelessWidget {
  const OccurrenceRow({
    super.key,
    required this.occurrence,
    required this.selectable,
    required this.selected,
    required this.onSelectedChanged,
  });

  final ShiftOccurrence occurrence;

  /// false se il volontario corrente non ha `shifts:request` (calendario in
  /// sola lettura) — in quel caso non si mostra mai la checkbox, nemmeno su
  /// uno slot libero.
  final bool selectable;
  final bool selected;
  final ValueChanged<bool> onSelectedChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final dateLabel = DateFormat.MMMd(context.locale.toString())
        .format(occurrence.date);
    final showCheckbox = selectable && occurrence.isBookable;
    final showMineBadge = occurrence.myBookingStatus != null;

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Row(
        children: [
          Container(
            width: 10,
            height: 10,
            margin: const EdgeInsets.only(right: 10),
            decoration: occurrenceIsOutlineOnly(occurrence)
                ? BoxDecoration(
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: occurrenceColor(context, occurrence),
                      width: 1.5,
                    ),
                  )
                : BoxDecoration(
                    color: occurrenceColor(context, occurrence),
                    shape: BoxShape.circle,
                  ),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  occurrence.label,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  '$dateLabel · ${occurrence.startTime}–${occurrence.endTime} · '
                  '${occurrenceStatusLabel(occurrence)}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          if (showCheckbox)
            Checkbox(
              value: selected,
              onChanged: (value) => onSelectedChanged(value ?? false),
            )
          else if (showMineBadge)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: theme.colorScheme.primaryContainer,
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                occurrence.myBookingStatus == MyBookingStatus.pending
                    ? 'shifts.mine_badge_pending'.tr()
                    : 'shifts.mine_badge_confirmed'.tr(),
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onPrimaryContainer,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
        ],
      ),
    );
  }
}
