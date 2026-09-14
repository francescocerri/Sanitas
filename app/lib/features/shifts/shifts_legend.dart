import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

import 'shift_models.dart';
import 'shift_status_style.dart';

/// Legenda dei colori/anelli usati sia dal calendario del volontario sia da
/// quello del gestore turni — senza, "in attesa (mia)" e "confermato (mio)"
/// condividerebbero lo stesso colore del comitato distinguibili solo da
/// anello-vs-pieno, una sottigliezza facile da perdere senza una
/// spiegazione esplicita a schermo (segnalato direttamente dall'utente).
/// Estratta da `shifts_screen.dart` (era `_Legend`, privata) per essere
/// riusata da `ShiftManagerScreen`.
class ShiftsLegend extends StatelessWidget {
  const ShiftsLegend({super.key, this.showMineEntries = true});

  /// false nella vista del gestore: le voci "mio in attesa"/"mio
  /// confermato" non hanno senso lì (il gestore non prenota per sé, vedi
  /// `OccurrenceCard.onAssign`).
  final bool showMineEntries;

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
        if (showMineEntries) ...[
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
      ],
    );
  }
}
