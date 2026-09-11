import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'shift_models.dart';
import 'shift_status_style.dart';

/// Card espandibile per un'occorrenza: chiusa mostra un riepilogo
/// compatto (quante figure sono ancora libere, o "Tuo turno" se il
/// chiamante ha già una figura lì); aperta rivela le 4 righe-figura
/// (autista/leader/soccorritore/osservatore), ciascuna con stato e
/// checkbox/badge indipendenti — design approvato via mockup interattivo
/// prima dell'implementazione (vedi ADR-0025 "Aggiornamento").
class OccurrenceCard extends StatefulWidget {
  const OccurrenceCard({
    super.key,
    required this.occurrence,
    required this.selectable,
    required this.selection,
    required this.onToggle,
    this.onAssign,
  });

  final ShiftOccurrence occurrence;

  /// false se il volontario corrente non ha `shifts:request` (calendario in
  /// sola lettura) — in quel caso non si mostra mai la checkbox, nemmeno su
  /// una figura libera.
  final bool selectable;

  /// Chiavi (`ShiftOccurrence.keyFor(role)`) delle figure attualmente
  /// selezionate, di qualunque occorrenza — il genitore possiede lo stato,
  /// questa card si limita a controllare le proprie.
  final Set<String> selection;
  final void Function(ShiftOccurrence occurrence, ShiftRole role) onToggle;

  /// Non-null solo nella schermata del gestore turni (`shifts:write`): al
  /// posto di checkbox/badge, ogni figura non ancora confermata mostra un
  /// bottone "Assegna" che apre il selettore volontario — un'azione diretta
  /// e immediata (`POST /v1/shift-bookings/direct`), non una selezione da
  /// accumulare, quindi ignora `selectable`/`selection`/`onToggle` quando è
  /// presente.
  final void Function(ShiftOccurrence occurrence, ShiftRole role)? onAssign;

  @override
  State<OccurrenceCard> createState() => _OccurrenceCardState();
}

class _OccurrenceCardState extends State<OccurrenceCard> {
  bool _expanded = false;

  /// La figura già selezionata su questa occorrenza (se una c'è) — cerca
  /// nel set di selezione GLOBALE (di tutte le card) una chiave che inizi
  /// con questa occorrenza, indipendentemente dalla figura, dato che una
  /// selezione è sempre una coppia occorrenza+figura (vedi
  /// `ShiftOccurrence.keyFor`).
  ShiftRole? _selectedRoleFor(ShiftOccurrence occurrence) {
    final prefix = '${occurrence.templateId}|${occurrence.dateKey}|';
    for (final key in widget.selection) {
      if (key.startsWith(prefix)) {
        final roleName = key.substring(prefix.length);
        for (final role in ShiftRole.values) {
          if (role.name == roleName) return role;
        }
      }
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final occurrence = widget.occurrence;
    final dateLabel = DateFormat.MMMd(context.locale.toString())
        .format(occurrence.date);
    final openCount = occurrence.roles
        .where((rc) => rc.status == ShiftOccurrenceStatus.free)
        .length;
    final mine = myAggregateStatus(occurrence);

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(14),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Material(
            color: Colors.transparent,
            borderRadius: BorderRadius.circular(14),
            child: InkWell(
              borderRadius: BorderRadius.circular(14),
              onTap: () => setState(() => _expanded = !_expanded),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 12,
                  vertical: 10,
                ),
                child: Row(
                  children: [
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
                            '$dateLabel · ${occurrence.startTime}–${occurrence.endTime}',
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 10,
                        vertical: 4,
                      ),
                      decoration: BoxDecoration(
                        color: mine != null
                            ? theme.colorScheme.primaryContainer
                            : theme.colorScheme.surfaceContainerHighest,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text(
                        mine == MyBookingStatus.pending
                            ? 'shifts.mine_badge_pending'.tr()
                            : mine == MyBookingStatus.confirmed
                            ? 'shifts.mine_badge_confirmed'.tr()
                            // Al completo appena autista/leader/soccorritore
                            // sono confermati (vedi `isComplete`), anche se
                            // l'osservatore resta libero — niente "1/4
                            // libero" fuorviante in quel caso.
                            : occurrence.isComplete
                            ? 'shifts.status_confirmed'.tr()
                            : 'shifts.card_open_count'.tr(
                                namedArgs: {'count': '$openCount'},
                              ),
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: mine != null
                              ? theme.colorScheme.onPrimaryContainer
                              : theme.colorScheme.onSurfaceVariant,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    Icon(
                      _expanded
                          ? Icons.keyboard_arrow_up_rounded
                          : Icons.keyboard_arrow_down_rounded,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ],
                ),
              ),
            ),
          ),
          AnimatedSize(
            duration: const Duration(milliseconds: 180),
            child: !_expanded
                ? const SizedBox(width: double.infinity)
                : Padding(
                    padding: const EdgeInsets.fromLTRB(12, 0, 12, 10),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        const Divider(height: 16),
                        for (final coverage in occurrence.roles)
                          _RoleRow(
                            coverage: coverage,
                            selectable: widget.selectable,
                            // Un volontario può tenere al più una figura
                            // per occorrenza (vedi ADR-0025
                            // "Aggiornamento"): se ne ha già selezionata
                            // una qui, le checkbox delle altre 3 restano
                            // visibili ma disabilitate — bloccarlo qui,
                            // non solo lasciare che il backend rifiuti la
                            // richiesta bulk al momento dell'invio.
                            locked:
                                _selectedRoleFor(occurrence) != null &&
                                _selectedRoleFor(occurrence) != coverage.role,
                            selected: widget.selection.contains(
                              occurrence.keyFor(coverage.role),
                            ),
                            onToggle: () =>
                                widget.onToggle(occurrence, coverage.role),
                            onAssign: widget.onAssign == null
                                ? null
                                : () => widget.onAssign!(
                                    occurrence,
                                    coverage.role,
                                  ),
                          ),
                      ],
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}

class _RoleRow extends StatelessWidget {
  const _RoleRow({
    required this.coverage,
    required this.selectable,
    required this.locked,
    required this.selected,
    required this.onToggle,
    this.onAssign,
  });

  final RoleCoverage coverage;
  final bool selectable;

  /// true se un'ALTRA figura di questa stessa occorrenza è già stata
  /// selezionata — la checkbox resta visibile ma disabilitata (un
  /// volontario tiene al più una figura per turno, vedi ADR-0025
  /// "Aggiornamento"). Non riguarda mai la riga già selezionata, quella
  /// resta sempre deselezionabile.
  final bool locked;
  final bool selected;
  final VoidCallback onToggle;

  /// Non-null solo nella schermata del gestore turni — vedi
  /// `OccurrenceCard.onAssign`.
  final VoidCallback? onAssign;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final showAssignButton =
        onAssign != null && coverage.status != ShiftOccurrenceStatus.confirmed;
    final showCheckbox = onAssign == null && selectable && coverage.isBookable;
    final showMineBadge = onAssign == null && coverage.myBookingStatus != null;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          Icon(
            roleIcon(coverage.role),
            size: 18,
            color: theme.colorScheme.onSurfaceVariant,
          ),
          const SizedBox(width: 8),
          Container(
            width: 9,
            height: 9,
            margin: const EdgeInsets.only(right: 8),
            decoration: roleIsOutlineOnly(coverage)
                ? BoxDecoration(
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: roleColor(context, coverage),
                      width: 1.5,
                    ),
                  )
                : BoxDecoration(
                    color: roleColor(context, coverage),
                    shape: BoxShape.circle,
                  ),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  roleLabel(coverage.role),
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface,
                  ),
                ),
                Text(
                  roleStatusLabel(coverage),
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
          if (showAssignButton)
            OutlinedButton(
              onPressed: onAssign,
              style: OutlinedButton.styleFrom(
                minimumSize: const Size(0, 32),
                padding: const EdgeInsets.symmetric(horizontal: 12),
                textStyle: theme.textTheme.labelSmall,
              ),
              child: Text('shifts.assign'.tr()),
            )
          else if (showCheckbox)
            Checkbox(
              value: selected,
              onChanged: (locked && !selected) ? null : (_) => onToggle(),
            )
          else if (showMineBadge)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: theme.colorScheme.primaryContainer,
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                coverage.myBookingStatus == MyBookingStatus.pending
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
