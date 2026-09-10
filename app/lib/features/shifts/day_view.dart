import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'date_math.dart';
import 'occurrence_card.dart';
import 'occurrences_async_builder.dart';
import 'shift_models.dart';
import 'shifts_filter.dart';

/// Vista Giorno: un giorno alla volta, navigabile avanti/indietro — il
/// massimo dettaglio fra le 4 viste.
class DayView extends StatefulWidget {
  const DayView({
    super.key,
    required this.filter,
    required this.selection,
    required this.selectable,
    required this.onToggle,
  });

  final ShiftsFilter filter;
  final Set<String> selection;
  final bool selectable;
  final void Function(ShiftOccurrence occurrence, ShiftRole role) onToggle;

  @override
  State<DayView> createState() => _DayViewState();
}

class _DayViewState extends State<DayView> {
  DateTime _day = dateOnly(DateTime.now());

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            IconButton(
              icon: const Icon(Icons.chevron_left_rounded),
              onPressed: () => setState(() => _day = addDays(_day, -1)),
            ),
            Text(
              DateFormat.yMMMMEEEEd(context.locale.toString()).format(_day),
              style: theme.textTheme.titleSmall,
            ),
            IconButton(
              icon: const Icon(Icons.chevron_right_rounded),
              onPressed: () => setState(() => _day = addDays(_day, 1)),
            ),
          ],
        ),
        const SizedBox(height: 8),
        OccurrencesAsyncBuilder(
          range: (from: _day, to: _day),
          filter: widget.filter,
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
            return Column(
              children: [
                for (final occurrence in items)
                  OccurrenceCard(
                    occurrence: occurrence,
                    selectable: widget.selectable,
                    selection: widget.selection,
                    onToggle: widget.onToggle,
                  ),
              ],
            );
          },
        ),
      ],
    );
  }
}
