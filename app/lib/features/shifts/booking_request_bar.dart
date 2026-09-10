import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

/// Barra fissa in fondo alla schermata Turni: contatore della selezione
/// corrente + bottone di invio. Visibile solo se `ShiftsScreen` ha almeno
/// uno slot selezionato (e il volontario ha `shifts:request` — controllato
/// lì, non qui: questo widget non sa nulla di permessi).
class BookingRequestBar extends StatelessWidget {
  const BookingRequestBar({
    super.key,
    required this.count,
    required this.submitting,
    required this.onSubmit,
  });

  final int count;
  final bool submitting;
  final VoidCallback onSubmit;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final label = count == 1
        ? 'shifts.selection_count_one'.tr()
        : 'shifts.selection_count_other'.tr(namedArgs: {'count': '$count'});

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          Expanded(
            child: Text(
              label,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onPrimaryContainer,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
          FilledButton(
            // Il tema dell'app dà ai `FilledButton` `minimumSize:
            // Size.fromHeight(56)` (larghezza infinita, pensato per i
            // bottoni a piena larghezza come "Salva"/"Accedi") — qui il
            // bottone sta in una `Row` accanto al contatore, serve una
            // dimensione minima compatta o il layout esplode (larghezza
            // infinita dentro un `Row`).
            style: FilledButton.styleFrom(minimumSize: const Size(64, 44)),
            onPressed: submitting ? null : onSubmit,
            child: submitting
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : Text('shifts.submit_request'.tr()),
          ),
        ],
      ),
    );
  }
}
