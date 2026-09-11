import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'date_math.dart';
import 'occurrence_card.dart';
import 'occurrences_async_builder.dart';
import 'shift_models.dart';
import 'shifts_filter.dart';

/// Lunedì della settimana che contiene [d] — stessa convenzione ISO usata
/// nel mockup approvato (settimana Lun-Dom, non Dom-Sab).
DateTime _startOfWeek(DateTime d) =>
    addDays(dateOnly(d), -(d.weekday - DateTime.monday));

/// Vista Settimana: i 7 giorni della settimana corrente, ciascuno con la
/// propria intestazione e le proprie occorrenze (o "Nessun turno") —
/// struttura intera della settimana sempre visibile, non solo i giorni con
/// turni.
class WeekView extends StatefulWidget {
  const WeekView({
    super.key,
    required this.filter,
    required this.selection,
    required this.selectable,
    required this.onToggle,
    this.onAssign,
  });

  final ShiftsFilter filter;
  final Set<String> selection;
  final bool selectable;
  final void Function(ShiftOccurrence occurrence, ShiftRole role) onToggle;
  final void Function(ShiftOccurrence occurrence, ShiftRole role)? onAssign;

  @override
  State<WeekView> createState() => _WeekViewState();
}

class _WeekViewState extends State<WeekView> {
  DateTime _weekStart = _startOfWeek(DateTime.now());

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final weekEnd = addDays(_weekStart, 6);
    final locale = context.locale.toString();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            IconButton(
              icon: const Icon(Icons.chevron_left_rounded),
              onPressed: () =>
                  setState(() => _weekStart = addDays(_weekStart, -7)),
            ),
            Text(
              '${DateFormat.MMMd(locale).format(_weekStart)} '
              '– ${DateFormat.MMMd(locale).format(weekEnd)}',
              style: theme.textTheme.titleSmall,
            ),
            IconButton(
              icon: const Icon(Icons.chevron_right_rounded),
              onPressed: () =>
                  setState(() => _weekStart = addDays(_weekStart, 7)),
            ),
          ],
        ),
        const SizedBox(height: 8),
        OccurrencesAsyncBuilder(
          range: (from: _weekStart, to: weekEnd),
          filter: widget.filter,
          builder: (context, items) {
            return Column(
              children: [
                for (var i = 0; i < 7; i++) ...[
                  _DayGroup(
                    day: addDays(_weekStart, i),
                    items: items
                        .where(
                          (o) => dateOnly(o.date) == addDays(_weekStart, i),
                        )
                        .toList(),
                    selectable: widget.selectable,
                    selection: widget.selection,
                    onToggle: widget.onToggle,
                    onAssign: widget.onAssign,
                  ),
                ],
              ],
            );
          },
        ),
      ],
    );
  }
}

class _DayGroup extends StatelessWidget {
  const _DayGroup({
    required this.day,
    required this.items,
    required this.selectable,
    required this.selection,
    required this.onToggle,
    this.onAssign,
  });

  final DateTime day;
  final List<ShiftOccurrence> items;
  final bool selectable;
  final Set<String> selection;
  final void Function(ShiftOccurrence occurrence, ShiftRole role) onToggle;
  final void Function(ShiftOccurrence occurrence, ShiftRole role)? onAssign;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.only(top: 8, bottom: 4),
            child: Text(
              DateFormat.yMMMMEEEEd(context.locale.toString()).format(day),
              style: theme.textTheme.labelMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ),
          if (items.isEmpty)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                'shifts.empty'.tr(),
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            )
          else
            for (final occurrence in items)
              OccurrenceCard(
                occurrence: occurrence,
                selectable: selectable,
                selection: selection,
                onToggle: onToggle,
                onAssign: onAssign,
              ),
        ],
      ),
    );
  }
}
