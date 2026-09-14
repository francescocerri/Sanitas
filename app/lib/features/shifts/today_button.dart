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
    return IconButton(
      tooltip: 'shifts.today'.tr(),
      icon: const Icon(Icons.today_rounded),
      onPressed: onPressed,
    );
  }
}
