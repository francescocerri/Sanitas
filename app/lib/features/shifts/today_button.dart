import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';

/// Bottone "torna a oggi" per l'intestazione di Giorno/Settimana/Mese —
/// spostarsi tra le viste per verificare la copertura di date lontane
/// (es. il mese prossimo) rende lento tornare a "adesso" senza, richiesto
/// esplicitamente dall'utente. Non compare in Lista, già sempre ancorata
/// a oggi (i prossimi 30 giorni).
class TodayButton extends StatelessWidget {
  const TodayButton({super.key, required this.onPressed});

  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    // Pillola con etichetta invece della sola icona — rifinitura richiesta
    // esplicitamente dall'utente dopo il mockup ("i bottoni erano fatti
    // meglio"), stesso trattamento di `FilterChipPill`/`ShiftsToolbarCard`
    // (bordo sottile, angoli pieni, nessun riempimento marcato).
    return OutlinedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.today_rounded, size: 16),
      label: Text('shifts.today'.tr()),
      style: OutlinedButton.styleFrom(
        foregroundColor: scheme.onSurfaceVariant,
        side: BorderSide(color: scheme.outlineVariant),
        shape: const StadiumBorder(),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        textStyle: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600),
      ),
    );
  }
}
