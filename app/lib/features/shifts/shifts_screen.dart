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
import 'shifts_filter.dart';
import 'shifts_legend.dart';
import 'shifts_providers.dart';
import 'week_view.dart';

enum ShiftsView { day, week, month, list }

/// Schermata Turni del volontario (`docs/backlog.md`, "Turni" del
/// frontend, voce 10): 4 viste (Giorno/Settimana/Mese/Lista, design
/// approvato via mockup), filtro Tutti/Liberi/Miei, selezione multipla di
/// figure (autista/leader/soccorritore/osservatore, vedi ADR-0025
/// "Aggiornamento") che alimenta `POST /v1/shift-bookings/bulk`. In sola
/// lettura per chi non ha `shifts:request` (niente checkbox/barra di
/// selezione, vedi `canRequestShiftsProvider`) — la rotta stessa resta
/// protetta da `shifts:read` (vedi `router.dart`).
class ShiftsScreen extends ConsumerStatefulWidget {
  const ShiftsScreen({super.key});

  @override
  ConsumerState<ShiftsScreen> createState() => _ShiftsScreenState();
}

class _ShiftsScreenState extends ConsumerState<ShiftsScreen> {
  ShiftsView _view = ShiftsView.month;
  ShiftsFilter _filter = ShiftsFilter.all;

  // Chiave = ShiftOccurrence.keyFor(role) — una selezione è sempre una
  // COPPIA occorrenza+figura, non solo un'occorrenza (un turno ha 4 figure
  // prenotabili indipendentemente, vedi ADR-0025 "Aggiornamento").
  final Map<String, BookingSelection> _selected = {};
  bool _submitting = false;

  void _toggle(ShiftOccurrence occurrence, ShiftRole role) {
    final key = occurrence.keyFor(role);
    setState(() {
      if (_selected.containsKey(key)) {
        _selected.remove(key);
      } else {
        _selected[key] = (occurrence: occurrence, role: role);
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
                  child: ShiftsLegend(),
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
