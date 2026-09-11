import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/api_exception.dart';
import '../../../core/widgets/error_banner.dart';
import '../../../core/widgets/theme_toggle_button.dart';
import '../../manage_users/manage_users_screen.dart' show usersProvider;
import '../agenda_list_view.dart';
import '../day_view.dart';
import '../month_view.dart';
import '../shift_models.dart';
import '../shift_status_style.dart';
import '../shifts_filter.dart';
import '../shifts_legend.dart';
import '../shifts_providers.dart';
import '../shifts_screen.dart' show ShiftsView;
import '../week_view.dart';
import 'shift_templates_tab.dart';
import 'volunteer_picker.dart';

/// Schermata del gestore turni (`docs/backlog.md`, voce 11): 3 tab —
/// Copertura (calendario completo, "Assegna" al posto della checkbox del
/// volontario), Richieste (approva/rifiuta, badge col conteggio) e
/// Turni-template (CRUD, voce 2). Raggiungibile solo con `shifts:write`
/// e/o `shifts:configure` (vedi `canAccessShiftManagementProvider`,
/// applicato già a livello di rotta in `router.dart`).
class ShiftManagerScreen extends ConsumerStatefulWidget {
  const ShiftManagerScreen({super.key});

  @override
  ConsumerState<ShiftManagerScreen> createState() => _ShiftManagerScreenState();
}

class _ShiftManagerScreenState extends ConsumerState<ShiftManagerScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabController;
  int _tabIndex = 0;

  @override
  void initState() {
    super.initState();
    _tabController = TabController(length: 3, vsync: this)
      ..addListener(() {
        if (_tabController.indexIsChanging) return;
        setState(() => _tabIndex = _tabController.index);
      });
  }

  @override
  void dispose() {
    _tabController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final pendingCount = ref
        .watch(pendingBookingsProvider)
        .maybeWhen(data: (items) => items.length, orElse: () => 0);
    final canConfigureTemplates = ref.watch(canConfigureShiftTemplatesProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'home.title'.tr(),
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: () => context.go('/home'),
        ),
        title: Text('shifts.manage_title'.tr()),
        actions: const [ThemeToggleButton(), SizedBox(width: 8)],
        bottom: TabBar(
          controller: _tabController,
          tabs: [
            Tab(text: 'shifts.tab_coverage'.tr()),
            Tab(
              child: Badge(
                label: Text('$pendingCount'),
                isLabelVisible: pendingCount > 0,
                alignment: AlignmentDirectional.topEnd,
                offset: const Offset(12, -8),
                child: Text('shifts.tab_requests'.tr()),
              ),
            ),
            Tab(text: 'shifts.tab_templates'.tr()),
          ],
        ),
      ),
      body: SafeArea(
        child: TabBarView(
          controller: _tabController,
          children: const [_CoverageTab(), _RequestsTab(), ShiftTemplatesTab()],
        ),
      ),
      floatingActionButton: _tabIndex == 2 && canConfigureTemplates
          ? const NewTemplateFab()
          : null,
    );
  }
}

class _CoverageTab extends ConsumerStatefulWidget {
  const _CoverageTab();

  @override
  ConsumerState<_CoverageTab> createState() => _CoverageTabState();
}

class _CoverageTabState extends ConsumerState<_CoverageTab> {
  ShiftsView _view = ShiftsView.month;
  ShiftsFilter _filter = ShiftsFilter.all;

  Future<void> _handleAssign(ShiftOccurrence occurrence, ShiftRole role) async {
    final volunteerId = await showVolunteerPicker(context);
    if (volunteerId == null || !mounted) return;
    try {
      await assignVolunteer(
        ref,
        templateId: occurrence.templateId,
        volunteerId: volunteerId,
        date: occurrence.dateKey,
        role: role,
      );
      ref.invalidate(occurrencesProvider);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('shifts.assign_success'.tr())));
    } on ApiException catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.translationKey.tr())));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Center(
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
                ],
              ),
            ),
            const Padding(
              padding: EdgeInsets.fromLTRB(16, 10, 16, 0),
              child: ShiftsLegend(showMineEntries: false),
            ),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
                child: switch (_view) {
                  ShiftsView.day => DayView(
                    filter: _filter,
                    selection: const {},
                    selectable: false,
                    onToggle: (_, _) {},
                    onAssign: _handleAssign,
                  ),
                  ShiftsView.week => WeekView(
                    filter: _filter,
                    selection: const {},
                    selectable: false,
                    onToggle: (_, _) {},
                    onAssign: _handleAssign,
                  ),
                  ShiftsView.month => MonthView(
                    filter: _filter,
                    selection: const {},
                    selectable: false,
                    onToggle: (_, _) {},
                    onAssign: _handleAssign,
                  ),
                  ShiftsView.list => AgendaListView(
                    filter: _filter,
                    selection: const {},
                    selectable: false,
                    onToggle: (_, _) {},
                    onAssign: _handleAssign,
                  ),
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _RequestsTab extends ConsumerWidget {
  const _RequestsTab();

  Future<void> _decide(
    BuildContext context,
    WidgetRef ref,
    String bookingId, {
    required bool approve,
  }) async {
    try {
      await decideBooking(ref, bookingId, approve: approve);
      ref.invalidate(pendingBookingsProvider);
      ref.invalidate(occurrencesProvider);
    } on ApiException catch (error) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.translationKey.tr())));
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final pending = ref.watch(pendingBookingsProvider);
    final users = ref.watch(usersProvider);
    final templates = ref.watch(templatesProvider);

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 720),
        child: pending.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (error, _) => Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  ErrorBanner(
                    message:
                        (error is ApiException
                                ? error.translationKey
                                : 'errors.unknown')
                            .tr(),
                  ),
                  const SizedBox(height: 12),
                  TextButton(
                    onPressed: () => ref.invalidate(pendingBookingsProvider),
                    child: Text('common.retry'.tr()),
                  ),
                ],
              ),
            ),
          ),
          data: (items) {
            if (items.isEmpty) {
              return Center(child: Text('shifts.no_pending_requests'.tr()));
            }
            // users/templates possono ancora essere in caricamento quando
            // pending è già arrivato: si mostra l'id grezzo come fallback
            // finché non arrivano, niente spinner bloccante per l'intera
            // lista solo per due lookup accessori.
            final usernameById = {
              for (final u in users.value ?? const []) u.id: u.username,
            };
            final templateById = {
              for (final t in templates.value ?? const []) t.id: t,
            };
            return ListView.separated(
              padding: const EdgeInsets.all(16),
              itemCount: items.length,
              separatorBuilder: (_, _) => const SizedBox(height: 8),
              itemBuilder: (context, index) {
                final booking = items[index];
                final template = templateById[booking.templateId];
                final dateLabel = DateFormat.yMMMd(context.locale.toString())
                    .format(booking.date);
                return Container(
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.surfaceContainerHighest
                        .withValues(alpha: 0.4),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        usernameById[booking.volunteerId] ??
                            booking.volunteerId,
                        style: Theme.of(context).textTheme.titleSmall
                            ?.copyWith(fontWeight: FontWeight.w700),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        [
                          if (template != null) template.label,
                          dateLabel,
                          roleLabel(booking.role),
                        ].join(' · '),
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: Theme.of(context).colorScheme.onSurfaceVariant,
                        ),
                      ),
                      const SizedBox(height: 10),
                      Row(
                        children: [
                          Expanded(
                            child: OutlinedButton(
                              onPressed: () => _decide(
                                context,
                                ref,
                                booking.id,
                                approve: true,
                              ),
                              child: Text('shifts.approve'.tr()),
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: OutlinedButton(
                              onPressed: () => _decide(
                                context,
                                ref,
                                booking.id,
                                approve: false,
                              ),
                              child: Text('shifts.reject'.tr()),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                );
              },
            );
          },
        ),
      ),
    );
  }
}
