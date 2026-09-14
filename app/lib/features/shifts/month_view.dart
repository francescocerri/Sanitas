import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:table_calendar/table_calendar.dart';

import 'date_math.dart';
import 'occurrence_card.dart';
import 'occurrences_async_builder.dart';
import 'shift_models.dart';
import 'shift_status_style.dart';
import 'shifts_filter.dart';
import 'today_button.dart';

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
    required this.isWide,
    this.onAssign,
  });

  final ShiftsFilter filter;
  final Set<String> selection;
  final bool selectable;
  final void Function(ShiftOccurrence occurrence, ShiftRole role) onToggle;
  final void Function(ShiftOccurrence occurrence, ShiftRole role)? onAssign;

  /// Sopra la soglia di `ShiftsScreen` (schermo largo, es. laptop):
  /// celle/pallini più grandi e le card del dettaglio giorno affiancate a
  /// coppie invece che impilate — la shell attorno è già più larga
  /// (`ConstrainedBox` in `shifts_screen.dart`), qui si usa davvero quello
  /// spazio in più. Sotto la soglia il rendering resta identico a prima
  /// (tablet/mobile verificati, non da toccare — richiesto esplicitamente
  /// dall'utente).
  final bool isWide;

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
            // `TableCalendar` ha già un proprio header con frecce
            // mese-precedente/successivo e il titolo — nessuno slot per un
            // terzo bottone lì dentro, quindi "torna a oggi" vive in una
            // riga sottile sopra, allineata a destra.
            Align(
              alignment: Alignment.centerRight,
              child: TodayButton(
                onPressed: () => setState(() {
                  _focusedDay = dateOnly(DateTime.now());
                  _selectedDay = dateOnly(DateTime.now());
                }),
              ),
            ),
            TableCalendar<ShiftOccurrence>(
              locale: context.locale.toString(),
              firstDay: DateTime.now().subtract(const Duration(days: 365)),
              lastDay: DateTime.now().add(const Duration(days: 365)),
              focusedDay: _focusedDay,
              // Stessa convenzione Lun-Dom già usata dalla vista Settimana
              // (vedi date_math.dart/_startOfWeek), non la domenica di
              // default del pacchetto.
              startingDayOfWeek: StartingDayOfWeek.monday,
              // 52 è il default del pacchetto, reso esplicito qui — su
              // schermo largo le celle diventano davvero più grandi (non
              // solo più larghe), richiesto esplicitamente dall'utente dopo
              // aver visto il calendario troppo piccolo su un laptop da
              // 15".
              rowHeight: widget.isWide ? 96 : 52,
              headerStyle: const HeaderStyle(
                formatButtonVisible: false,
                titleCentered: true,
              ),
              // Le decorazioni di default del pacchetto sono pesanti
              // (cerchi pieni, quella "selezionato" pure di un blu che non
              // c'entra col tema) e sovradimensionate rispetto ai puntini
              // di stato: "oggi" resta un anello sottile nel colore del
              // comitato su schermo stretto (comportamento verificato,
              // invariato); su schermo largo diventa un cerchio pieno, più
              // invitante con celle così più grandi (mockup approvato).
              // "selezionato" un riempimento leggero (stesso colore, non il
              // blu di default) con testo in grassetto in entrambi i casi.
              // `cellMargin` più ampio del default riduce anche la
              // dimensione fisica di entrambi rispetto alla cella.
              calendarStyle: CalendarStyle(
                outsideDaysVisible: false,
                cellMargin: const EdgeInsets.all(8),
                todayDecoration: widget.isWide
                    ? BoxDecoration(
                        shape: BoxShape.circle,
                        color: theme.colorScheme.primary,
                      )
                    : BoxDecoration(
                        shape: BoxShape.circle,
                        border: Border.all(
                          color: theme.colorScheme.primary,
                          width: 1.5,
                        ),
                      ),
                todayTextStyle: TextStyle(
                  color: widget.isWide
                      ? theme.colorScheme.onPrimary
                      : theme.colorScheme.primary,
                ),
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
                  final dotSize = widget.isWide ? 8.0 : 6.0;
                  final xSize = widget.isWide ? 12.0 : 9.0;
                  return Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      for (final occurrence in events.take(3))
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 1),
                          // Un turno "chiuso" (vedi
                          // `occurrenceCardClosedMarker`) è una X diretta,
                          // non un pallino con una X sovrapposta: a 6px un
                          // cerchio più un'icona sopra risultava illeggibile
                          // — segnalato esplicitamente dall'utente.
                          child: occurrenceCardClosedMarker(occurrence)
                              ? Icon(
                                  Icons.close_rounded,
                                  size: xSize,
                                  color: Colors.red.shade700,
                                )
                              : Container(
                                  width: dotSize,
                                  height: dotSize,
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
              else if (!widget.isWide)
                for (final occurrence in selectedItems)
                  OccurrenceCard(
                    occurrence: occurrence,
                    selectable: widget.selectable,
                    selection: widget.selection,
                    onToggle: widget.onToggle,
                    onAssign: widget.onAssign,
                  )
              else
                // Schermo largo: le card (ad altezza variabile, si
                // espandono al tocco) si affiancano a coppie invece di
                // restare impilate in un'unica colonna stretta — niente
                // `GridView` (richiede un aspect ratio fisso, qui l'altezza
                // cambia quando una card si espande): ogni riga è
                // semplicemente larga quanto la sua card più alta, la
                // successiva parte sotto.
                for (var i = 0; i < selectedItems.length; i += 2)
                  // `OccurrenceCard` porta già il proprio margine inferiore
                  // (8px): niente Padding aggiuntivo qui, altrimenti lo
                  // spazio fra una riga e la successiva raddoppierebbe
                  // rispetto alla colonna singola di sotto la soglia.
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(
                        child: OccurrenceCard(
                          occurrence: selectedItems[i],
                          selectable: widget.selectable,
                          selection: widget.selection,
                          onToggle: widget.onToggle,
                          onAssign: widget.onAssign,
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: i + 1 < selectedItems.length
                            ? OccurrenceCard(
                                occurrence: selectedItems[i + 1],
                                selectable: widget.selectable,
                                selection: widget.selection,
                                onToggle: widget.onToggle,
                                onAssign: widget.onAssign,
                              )
                            : const SizedBox.shrink(),
                      ),
                    ],
                  ),
            ],
          ],
        );
      },
    );
  }
}
