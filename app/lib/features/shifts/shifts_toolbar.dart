import 'package:flutter/material.dart';

/// Card che racchiude il gruppo viste + i filtri in un unico blocco visivo
/// (bordo, angoli arrotondati, ombra leggera) invece di lasciarli come
/// widget nudi sullo sfondo della schermata — rifinitura richiesta
/// esplicitamente dall'utente dopo aver visto il mockup approvato ("i
/// filtri erano fatti molto meglio"), riusata identica da `ShiftsScreen` e
/// `ShiftManagerScreen` (stessa barra, duplicata in entrambe le
/// schermate).
class ShiftsToolbarCard extends StatelessWidget {
  const ShiftsToolbarCard({super.key, required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: theme.colorScheme.outlineVariant),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.05),
            blurRadius: 14,
            offset: const Offset(0, 3),
          ),
        ],
      ),
      child: Wrap(
        spacing: 12,
        runSpacing: 10,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: children,
      ),
    );
  }
}

/// Stile condiviso del gruppo di viste (Giorno/Settimana/Mese/Lista) —
/// evita di duplicare lo `ButtonStyle` fra `ShiftsScreen` e
/// `ShiftManagerScreen`. Il `SegmentedButton` di Material 3 di default
/// (segmento selezionato con tonalità primaria + spunta) risultava più
/// "amministrativo" del gruppo a pillola del mockup — stessa struttura del
/// widget (Material gestisce già bordo esterno/divisori condivisi), solo
/// colori/forma cambiano; `showSelectedIcon: false` toglie la spunta.
ButtonStyle segmentedButtonStyle(ThemeData theme) {
  final scheme = theme.colorScheme;
  return SegmentedButton.styleFrom(
    backgroundColor: scheme.surfaceContainerHighest.withValues(alpha: 0.5),
    selectedBackgroundColor: scheme.surface,
    selectedForegroundColor: scheme.onSurface,
    foregroundColor: scheme.onSurfaceVariant,
    side: BorderSide(color: scheme.outlineVariant),
    textStyle: theme.textTheme.labelMedium?.copyWith(
      fontWeight: FontWeight.w600,
    ),
    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
  );
}

/// Chip di filtro (Tutti/Liberi/Miei) in stile "pillola piena" quando
/// selezionata (sfondo scuro, testo chiaro) invece del tonale chiaro di
/// default di `ChoiceChip` — stessa rifinitura di [segmentedButtonStyle],
/// riusata da `ShiftsScreen` e `ShiftManagerScreen`.
class FilterChipPill extends StatelessWidget {
  const FilterChipPill({
    super.key,
    required this.label,
    required this.selected,
    required this.onSelected,
  });

  final String label;
  final bool selected;
  final ValueChanged<bool> onSelected;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return ChoiceChip(
      label: Text(label),
      selected: selected,
      onSelected: onSelected,
      showCheckmark: false,
      backgroundColor: scheme.surface,
      selectedColor: scheme.onSurface,
      side: BorderSide(color: scheme.outlineVariant),
      shape: const StadiumBorder(),
      labelStyle: TextStyle(
        color: selected ? scheme.surface : scheme.onSurfaceVariant,
        fontWeight: FontWeight.w600,
      ),
    );
  }
}
