import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'date_math.dart';
import 'occurrence_row.dart';
import 'occurrences_async_builder.dart';
import 'shift_models.dart';
import 'shifts_filter.dart';

/// Vista Lista: agenda scorrevole sui prossimi 30 giorni, comoda per
/// selezionare più slot su settimane diverse in un colpo solo (il caso
/// d'uso della richiesta bulk) senza dover cambiare mese/settimana più
/// volte. A differenza di Giorno/Settimana, i giorni senza turni non
/// compaiono affatto (altrimenti, su 30 giorni con turni solo 3 giorni a
/// settimana, sarebbe perlopiù vuota).
class AgendaListView extends StatelessWidget {
  const AgendaListView({
    super.key,
    required this.filter,
    required this.selection,
    required this.selectable,
    required this.onToggle,
  });

  final ShiftsFilter filter;
  final Set<String> selection;
  final bool selectable;
  final void Function(ShiftOccurrence occurrence) onToggle;

  static const _rangeDays = 30;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final from = dateOnly(DateTime.now());
    final to = addDays(from, _rangeDays - 1);

    return OccurrencesAsyncBuilder(
      range: (from: from, to: to),
      filter: filter,
      builder: (context, items) {
        if (items.isEmpty) {
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 24),
            child: Center(
              child: Text(
                'shifts.empty'.tr(),
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          );
        }

        final byDay = <DateTime, List<ShiftOccurrence>>{};
        for (final occurrence in items) {
          byDay
              .putIfAbsent(dateOnly(occurrence.date), () => [])
              .add(occurrence);
        }
        final days = byDay.keys.toList()..sort();

        return Column(
          children: [
            for (final day in days) ...[
              Padding(
                padding: const EdgeInsets.only(top: 8, bottom: 4),
                child: Align(
                  alignment: Alignment.centerLeft,
                  child: Text(
                    DateFormat.yMMMMEEEEd(context.locale.toString())
                        .format(day),
                    style: theme.textTheme.labelMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
              ),
              for (final occurrence in byDay[day]!)
                OccurrenceRow(
                  occurrence: occurrence,
                  selectable: selectable,
                  selected: selection.contains(occurrence.key),
                  onSelectedChanged: (_) => onToggle(occurrence),
                ),
            ],
          ],
        );
      },
    );
  }
}
