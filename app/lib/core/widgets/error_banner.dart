import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';

/// Banner d'errore riusato identico in tutte le schermate con un form
/// (login, attivazione account, cambio password): sfondo tenue del colore
/// d'errore di Material invece di solo testo rosso su sfondo neutro — più
/// visibile, e coerente con lo stile "riempito" del resto della UI (vedi
/// `committee_theme.dart`).
class ErrorBanner extends StatelessWidget {
  const ErrorBanner({required this.message, super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
          width: double.infinity,
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: colorScheme.errorContainer,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Row(
            children: [
              Icon(
                Icons.error_outline_rounded,
                color: colorScheme.onErrorContainer,
                size: 20,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  message,
                  style: TextStyle(color: colorScheme.onErrorContainer),
                ),
              ),
            ],
          ),
        )
        // Ogni volta che questo banner COMPARE (es. dopo un tentativo di
        // login fallito) riparte da capo con dissolvenza + un piccolo
        // "scatto" orizzontale (`shake`): un tocco leggero che richiama
        // l'attenzione senza essere invadente.
        .animate()
        .fadeIn(duration: 200.ms)
        .shake(hz: 4, curve: Curves.easeInOut, duration: 300.ms);
  }
}

/// Variante "successo" dello stesso banner (usata dopo il cambio password,
/// l'attivazione account, ecc.): stessa forma, colori del `primary` invece
/// che dell'errore.
class SuccessBanner extends StatelessWidget {
  const SuccessBanner({required this.message, super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
          width: double.infinity,
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: colorScheme.primaryContainer,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Row(
            children: [
              Icon(
                Icons.check_circle_outline_rounded,
                color: colorScheme.onPrimaryContainer,
                size: 20,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  message,
                  style: TextStyle(color: colorScheme.onPrimaryContainer),
                ),
              ),
            ],
          ),
        )
        // Stessa animazione di ingresso del banner d'errore, ma senza lo
        // "shake" (un successo non ha bisogno di richiamare l'attenzione
        // con la stessa urgenza di un errore) — solo una dissolvenza con
        // un leggero movimento verso l'alto.
        .animate()
        .fadeIn(duration: 200.ms)
        .slideY(begin: 0.15, end: 0, curve: Curves.easeOutCubic);
  }
}
