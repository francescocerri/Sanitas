import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:table_calendar/table_calendar.dart';

import 'date_math.dart';
import 'occurrence_card.dart';
import 'occurrences_async_builder.dart';
import 'shift_models.dart';
import 'shift_status_style.dart';
import 'shifts_filter.dart';

/// Vista Mese: griglia calendario (pacchetto `table_calendar`, vedi
/// `docs/backlog.md`) con un pallino colorato per turno sotto ogni giorno
/// con turni (un pallino aggregato, non uno per figura — vedi
/// `occurrenceCardColor`); toccare un giorno apre sotto il pannello con le
/// sue occorrenze — stessa `OccurrenceCard` espandibile delle altre viste,
/// senza una seconda chiamata di rete (riusa la lista già scaricata per il
/// mese).
class MonthView extends StatefulWidget {
  const MonthView({
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
  State<MonthView> createState() => _MonthViewState();
}

class _MonthViewState extends State<MonthView> {
  DateTime _focusedDay = dateOnly(DateTime.now());
  DateTime? _selectedDay;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final firstOfMonth = DateTime(_focusedDay.year, _focusedDay.month, 1);
    final lastOfMonth = DateTime(_focusedDay.year, _focusedDay.month + 1, 0);

    return OccurrencesAsyncBuilder(
      range: (from: firstOfMonth, to: lastOfMonth),
      filter: widget.filter,
      builder: (context, items) {
        final byDay = <DateTime, List<ShiftOccurrence>>{};
        for (final occurrence in items) {
          byDay
              .putIfAbsent(dateOnly(occurrence.date), () => [])
              .add(occurrence);
        }

        final selectedItems = _selectedDay == null
            ? const <ShiftOccurrence>[]
            : (byDay[dateOnly(_selectedDay!)] ?? const []);

        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TableCalendar<ShiftOccurrence>(
              locale: context.locale.toString(),
              firstDay: DateTime.now().subtract(const Duration(days: 365)),
              lastDay: DateTime.now().add(const Duration(days: 365)),
              focusedDay: _focusedDay,
              // Stessa convenzione Lun-Dom già usata dalla vista Settimana
              // (vedi date_math.dart/_startOfWeek), non la domenica di
              // default del pacchetto.
              startingDayOfWeek: StartingDayOfWeek.monday,
              headerStyle: const HeaderStyle(
                formatButtonVisible: false,
                titleCentered: true,
              ),
              // Le decorazioni di default del pacchetto sono pesanti
              // (cerchi pieni, quella "selezionato" pure di un blu che non
              // c'entra col tema) e sovradimensionate rispetto ai puntini
              // di stato: "oggi" resta un anello sottile nel colore del
              // comitato; "selezionato" un riempimento leggero (stesso
              // colore, non il blu di default) con testo in grassetto —
              // nessuno dei due un cerchio pieno e marcato. `cellMargin`
              // più ampio del default riduce anche la dimensione fisica di
              // entrambi rispetto alla cella.
              calendarStyle: CalendarStyle(
                outsideDaysVisible: false,
                cellMargin: const EdgeInsets.all(8),
                todayDecoration: BoxDecoration(
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: theme.colorScheme.primary,
                    width: 1.5,
                  ),
                ),
                todayTextStyle: TextStyle(color: theme.colorScheme.primary),
                selectedDecoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: theme.colorScheme.primary.withValues(alpha: 0.16),
                ),
                selectedTextStyle: TextStyle(
                  color: theme.colorScheme.primary,
                  fontWeight: FontWeight.w700,
                ),
              ),
              eventLoader: (day) => byDay[dateOnly(day)] ?? const [],
              selectedDayPredicate: (day) =>
                  _selectedDay != null && isSameDay(_selectedDay, day),
              onDaySelected: (selectedDay, focusedDay) {
                setState(() {
                  _selectedDay = dateOnly(selectedDay);
                  _focusedDay = dateOnly(focusedDay);
                });
              },
              onPageChanged: (focusedDay) {
                setState(() {
                  _focusedDay = dateOnly(focusedDay);
                  _selectedDay = null;
                });
              },
              calendarBuilders: CalendarBuilders(
                markerBuilder: (context, day, events) {
                  if (events.isEmpty) return null;
                  return Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      for (final occurrence in events.take(3))
                        Container(
                          width: 6,
                          height: 6,
                          margin: const EdgeInsets.symmetric(horizontal: 1),
                          decoration: occurrenceCardOutline(occurrence)
                              ? BoxDecoration(
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                    color: occurrenceCardColor(
                                      context,
                                      occurrence,
                                    ),
                                    width: 1.2,
                                  ),
                                )
                              : BoxDecoration(
                                  color: occurrenceCardColor(
                                    context,
                                    occurrence,
                                  ),
                                  shape: BoxShape.circle,
                                ),
                        ),
                    ],
                  );
                },
              ),
            ),
            if (_selectedDay != null) ...[
              const Divider(height: 32),
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Text(
                  DateFormat.yMMMMEEEEd(context.locale.toString())
                      .format(_selectedDay!),
                  style: theme.textTheme.titleSmall,
                ),
              ),
              if (selectedItems.isEmpty)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 8),
                  child: Text(
                    'shifts.empty'.tr(),
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                )
              else
                for (final occurrence in selectedItems)
                  OccurrenceCard(
                    occurrence: occurrence,
                    selectable: widget.selectable,
                    selection: widget.selection,
                    onToggle: widget.onToggle,
                  ),
            ],
          ],
        );
      },
    );
  }
}
