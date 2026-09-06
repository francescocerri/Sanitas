import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../theme/committee_theme.dart';

/// Il "logo" dell'app quando il comitato che ha fatto il fork non fornisce
/// un file immagine (caso comune: non tutti hanno un logo già digitalizzato
/// pronto): un cerchio con l'iniziale del nome del comitato — lo stesso
/// pattern di "monogramma colorato" che usano Slack, Notion, Google per gli
/// avatar quando manca una vera immagine. Un solo punto da cambiare in
/// futuro se un comitato vorrà caricare un logo vero.
class BrandMark extends ConsumerWidget {
  const BrandMark({this.size = 64, this.inverse = false, super.key});

  final double size;

  /// `false` (default): cerchio con gradiente nei colori del brand e testo
  /// bianco — usato su sfondi neutri (la card del form, lo splash).
  /// `true`: cerchio bianco semi-trasparente con testo colorato — usato
  /// quando il monogramma sta SOPRA uno sfondo già colorato col gradiente
  /// del brand (il pannello laterale su schermi larghi, vedi `auth_shell.dart`),
  /// dove la versione a colori pieni si "perderebbe" nello sfondo.
  final bool inverse;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final committeeTheme = ref.watch(committeeThemeProvider);

    return Container(
      width: size,
      height: size,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        gradient: inverse
            ? null
            : LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [committeeTheme.primary, committeeTheme.secondary],
              ),
        color: inverse ? Colors.white.withValues(alpha: 0.16) : null,
        border: inverse
            ? Border.all(color: Colors.white.withValues(alpha: 0.4), width: 1.5)
            : null,
        boxShadow: inverse
            ? null
            : [
                BoxShadow(
                  color: committeeTheme.primary.withValues(alpha: 0.35),
                  blurRadius: 24,
                  offset: const Offset(0, 8),
                ),
              ],
      ),
      alignment: Alignment.center,
      child: Text(
        committeeTheme.monogram,
        style: TextStyle(
          fontSize: size * 0.42,
          fontWeight: FontWeight.w800,
          color: Colors.white,
        ),
      ),
    );
  }
}
