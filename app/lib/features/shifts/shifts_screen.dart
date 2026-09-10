import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_exception.dart';
import '../../core/widgets/theme_toggle_button.dart';
import 'agenda_list_view.dart';
import 'booking_request_bar.dart';
import 'day_view.dart';
import 'month_view.dart';
import 'shift_models.dart';
import 'shift_status_style.dart';
import 'shifts_filter.dart';
import 'shifts_providers.dart';
import 'week_view.dart';

enum ShiftsView { day, week, month, list }

/// Schermata Turni del volontario (`docs/backlog.md`, "Turni" del
/// frontend, voce 10): 4 viste (Giorno/Settimana/Mese/Lista, design
/// approvato via mockup), filtro Tutti/Liberi/Miei, selezione multipla di
/// slot che alimenta `POST /v1/shift-bookings/bulk`. In sola lettura per chi
/// non ha `shifts:request` (niente checkbox/barra di selezione, vedi
/// `canRequestShiftsProvider`) — la rotta stessa resta protetta da
/// `shifts:read` (vedi `router.dart`).
class ShiftsScreen extends ConsumerStatefulWidget {
  const ShiftsScreen({super.key});

  @override
  ConsumerState<ShiftsScreen> createState() => _ShiftsScreenState();
}

class _ShiftsScreenState extends ConsumerState<ShiftsScreen> {
  ShiftsView _view = ShiftsView.month;
  ShiftsFilter _filter = ShiftsFilter.all;
  final Map<String, ShiftOccurrence> _selected = {};
  bool _submitting = false;

  void _toggle(ShiftOccurrence occurrence) {
    setState(() {
      if (_selected.containsKey(occurrence.key)) {
        _selected.remove(occurrence.key);
      } else {
        _selected[occurrence.key] = occurrence;
      }
    });
  }

  Future<void> _submit() async {
    setState(() => _submitting = true);
    try {
      await requestBulkBookings(ref, _selected.values.toList());
      ref.invalidate(occurrencesProvider);
      if (!mounted) return;
      setState(() => _selected.clear());
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('shifts.request_sent'.tr())));
    } on ApiException catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('shifts.request_failed'.tr())));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final canRequest = ref.watch(canRequestShiftsProvider);
    final selection = _selected.keys.toSet();

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'home.title'.tr(),
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: () => context.go('/home'),
        ),
        title: Text('shifts.title'.tr()),
        actions: const [ThemeToggleButton(), SizedBox(width: 8)],
      ),
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 720),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                  child: SegmentedButton<ShiftsView>(
                    segments: [
                      ButtonSegment(
                        value: ShiftsView.day,
                        label: Text('shifts.view_day'.tr()),
                      ),
                      ButtonSegment(
                        value: ShiftsView.week,
                        label: Text('shifts.view_week'.tr()),
                      ),
                      ButtonSegment(
                        value: ShiftsView.month,
                        label: Text('shifts.view_month'.tr()),
                      ),
                      ButtonSegment(
                        value: ShiftsView.list,
                        label: Text('shifts.view_list'.tr()),
                      ),
                    ],
                    selected: {_view},
                    onSelectionChanged: (selection) =>
                        setState(() => _view = selection.first),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                  child: Wrap(
                    spacing: 8,
                    children: [
                      ChoiceChip(
                        label: Text('shifts.filter_all'.tr()),
                        selected: _filter == ShiftsFilter.all,
                        onSelected: (_) =>
                            setState(() => _filter = ShiftsFilter.all),
                      ),
                      ChoiceChip(
                        label: Text('shifts.filter_free'.tr()),
                        selected: _filter == ShiftsFilter.free,
                        onSelected: (_) =>
                            setState(() => _filter = ShiftsFilter.free),
                      ),
                      ChoiceChip(
                        label: Text('shifts.filter_mine'.tr()),
                        selected: _filter == ShiftsFilter.mine,
                        onSelected: (_) =>
                            setState(() => _filter = ShiftsFilter.mine),
                      ),
                    ],
                  ),
                ),
                const Padding(
                  padding: EdgeInsets.fromLTRB(16, 10, 16, 0),
                  child: _Legend(),
                ),
                Expanded(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
                    child: switch (_view) {
                      ShiftsView.day => DayView(
                        filter: _filter,
                        selection: selection,
                        selectable: canRequest,
                        onToggle: _toggle,
                      ),
                      ShiftsView.week => WeekView(
                        filter: _filter,
                        selection: selection,
                        selectable: canRequest,
                        onToggle: _toggle,
                      ),
                      ShiftsView.month => MonthView(
                        filter: _filter,
                        selection: selection,
                        selectable: canRequest,
                        onToggle: _toggle,
                      ),
                      ShiftsView.list => AgendaListView(
                        filter: _filter,
                        selection: selection,
                        selectable: canRequest,
                        onToggle: _toggle,
                      ),
                    },
                  ),
                ),
                if (canRequest && _selected.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
                    child: BookingRequestBar(
                      count: _selected.length,
                      submitting: _submitting,
                      onSubmit: _submit,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Legenda dei 5 stati visivi: senza, "in attesa (mia)" e "confermato
/// (mio)" condividerebbero lo stesso colore del comitato distinguibili
/// solo da anello-vs-pieno — una sottigliezza facile da perdere senza una
/// spiegazione esplicita a schermo (segnalato direttamente dall'utente).
class _Legend extends StatelessWidget {
  const _Legend();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final textStyle = theme.textTheme.bodySmall?.copyWith(
      color: theme.colorScheme.onSurfaceVariant,
    );

    Widget dot(Color color, {bool outline = false}) => Container(
      width: 8,
      height: 8,
      decoration: outline
          ? BoxDecoration(
              shape: BoxShape.circle,
              border: Border.all(color: color, width: 1.5),
            )
          : BoxDecoration(color: color, shape: BoxShape.circle),
    );

    Widget item(Widget dot, String label) => Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        dot,
        const SizedBox(width: 5),
        Text(label, style: textStyle),
      ],
    );

    return Wrap(
      spacing: 14,
      runSpacing: 4,
      children: [
        item(
          dot(statusColor(context, ShiftOccurrenceStatus.free, null)),
          'shifts.status_free'.tr(),
        ),
        item(
          dot(statusColor(context, ShiftOccurrenceStatus.pending, null)),
          'shifts.status_pending'.tr(),
        ),
        item(
          dot(statusColor(context, ShiftOccurrenceStatus.confirmed, null)),
          'shifts.status_confirmed'.tr(),
        ),
        item(
          dot(
            statusColor(
              context,
              ShiftOccurrenceStatus.free,
              MyBookingStatus.pending,
            ),
            outline: true,
          ),
          'shifts.status_mine_pending'.tr(),
        ),
        item(
          dot(
            statusColor(
              context,
              ShiftOccurrenceStatus.free,
              MyBookingStatus.confirmed,
            ),
          ),
          'shifts.status_mine_confirmed'.tr(),
        ),
      ],
    );
  }
}
