import 'package:flutter/material.dart';

/// Icona mostra/nascondi password riusata in login, attivazione account e
/// cambio password. `AnimatedSwitcher` fa dissolvere l'icona vecchia mentre
/// quella nuova compare, invece di scambiarle di scatto — un tocco leggero
/// che rende più "vivo" un controllo altrimenti banale.
class PasswordVisibilityToggle extends StatelessWidget {
  const PasswordVisibilityToggle({
    required this.obscured,
    required this.onPressed,
    super.key,
  });

  final bool obscured;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      icon: AnimatedSwitcher(
        duration: const Duration(milliseconds: 150),
        // `KeyedSubtree` con una chiave diversa per ogni stato è come si
        // dice ad `AnimatedSwitcher` "questi due sono widget DIVERSI, anima
        // il passaggio dall'uno all'altro" — senza una chiave che cambia,
        // vedrebbe sempre "un'icona" e non saprebbe che è cambiata.
        child: Icon(
          obscured ? Icons.visibility_outlined : Icons.visibility_off_outlined,
          key: ValueKey(obscured),
        ),
      ),
      onPressed: onPressed,
    );
  }
}
