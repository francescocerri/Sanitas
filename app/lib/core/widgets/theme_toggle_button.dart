import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../theme/theme_mode_controller.dart';

/// Bottone che fa scorrere `ThemeMode` fra `system` → `light` → `dark` → di
/// nuovo `system` a ogni tocco, con un'icona diversa per far capire subito
/// in che stato ci si trova. Il toggle chiaro/scuro è una delle richieste
/// più comuni nelle UI di autenticazione moderne (vedi ricerca citata nella
/// conversazione) — qui è visibile fin dalla schermata di login, non
/// nascosto in un menu impostazioni.
class ThemeToggleButton extends ConsumerWidget {
  const ThemeToggleButton({this.color, super.key});

  /// Colore dell'icona: utile per renderlo leggibile sopra sfondi scuri
  /// (es. il pannello brandizzato di `auth_shell.dart`), dove il colore di
  /// default del tema non basterebbe da solo.
  final Color? color;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeControllerProvider);

    final (icon, tooltip) = switch (themeMode) {
      ThemeMode.system => (Icons.brightness_auto_outlined, 'Tema: automatico'),
      ThemeMode.light => (Icons.light_mode_outlined, 'Tema: chiaro'),
      ThemeMode.dark => (Icons.dark_mode_outlined, 'Tema: scuro'),
    };

    return IconButton(
      icon: Icon(icon, color: color),
      tooltip: tooltip,
      onPressed: () {
        final next = switch (themeMode) {
          ThemeMode.system => ThemeMode.light,
          ThemeMode.light => ThemeMode.dark,
          ThemeMode.dark => ThemeMode.system,
        };
        ref.read(themeModeControllerProvider.notifier).setThemeMode(next);
      },
    );
  }
}
