import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// I dati di personalizzazione del comitato che ha fatto il fork di questo
/// progetto: nome, lingua di default, colori del brand. Arrivano da
/// `config/<slug>/app/theme.json` (vedi `docs/adr/0022-frontend-flutter.md`),
/// copiati in `assets/committee/theme.json` da
/// `scripts/sync_committee_config.sh` PRIMA di ogni `flutter run`/`build`
/// (Flutter non legge asset da fuori la cartella del progetto in modo
/// affidabile su tutte le piattaforme).
///
/// **Importante**: qui NON deve mai finire un valore hardcoded specifico del
/// Comitato di Pavullo (vincolo di forkabilità in `CLAUDE.md`) — solo il
/// meccanismo generico per leggere QUALUNQUE tema, quale sia il comitato.
class CommitteeTheme {
  const CommitteeTheme({
    required this.committeeName,
    required this.defaultLocale,
    required this.primary,
    required this.secondary,
    required this.surface,
  });

  final String committeeName;

  /// Lingua con cui parte l'app per QUESTO comitato (es. "it"). Il progetto
  /// di riferimento la imposta a italiano, ma un fork può scegliere
  /// diversamente — per questo non è una costante nel codice, viene dalla
  /// config. Se il file di config non la specifica, [loadCommitteeTheme]
  /// usa "it" come ripiego (l'italiano resta il default del PROGETTO, non
  /// solo di questo comitato).
  final String defaultLocale;

  final Color primary;
  final Color secondary;
  final Color surface;

  /// Costruisce il `ThemeData` di Material 3 a partire da un solo colore
  /// "seme" (il primary del comitato): `ColorScheme.fromSeed` genera da solo
  /// tutte le sfumature necessarie (colori per bottoni, sfondi, testo su
  /// sfondo colorato, ecc.) rispettando le regole di contrasto di Material
  /// Design — non dobbiamo scegliere a mano decine di colori per ogni tema.
  /// [brightness] separa questo asse (chiaro/scuro) dal colore del brand:
  /// stesso primary, ma una palette diversa in dark mode.
  ThemeData toThemeData(Brightness brightness) {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: primary,
      brightness: brightness,
      surface: brightness == Brightness.light ? surface : null,
    );
    return ThemeData(useMaterial3: true, colorScheme: colorScheme);
  }
}

/// Se l'asset manca (es. qualcuno ha dimenticato di lanciare lo script di
/// sync prima di `flutter run`) preferiamo un errore chiaro subito
/// all'avvio, non un crash generico più avanti nell'app.
class CommitteeThemeNotFoundException implements Exception {
  const CommitteeThemeNotFoundException();

  @override
  String toString() =>
      'assets/committee/theme.json non trovato. '
      'Esegui prima `scripts/sync_committee_config.sh` '
      '(vedi app/README.md).';
}

Future<CommitteeTheme> loadCommitteeTheme() async {
  final String rawJson;
  try {
    rawJson = await rootBundle.loadString('assets/committee/theme.json');
  } catch (_) {
    throw const CommitteeThemeNotFoundException();
  }

  final data = jsonDecode(rawJson) as Map<String, dynamic>;
  final colors = data['colors'] as Map<String, dynamic>;

  return CommitteeTheme(
    committeeName: data['committee_name'] as String,
    defaultLocale: data['default_locale'] as String? ?? 'it',
    primary: _colorFromHex(colors['primary'] as String),
    secondary: _colorFromHex(colors['secondary'] as String),
    surface: _colorFromHex(colors['surface'] as String),
  );
}

/// A differenza degli altri provider di questo progetto, [CommitteeTheme]
/// non si carica da solo: va letto UNA VOLTA, in modo asincrono, prima
/// ancora di chiamare `runApp` (vedi `main.dart`), perché serve subito per
/// costruire il primo `MaterialApp`. Questo provider è solo un "segnaposto":
/// `main.dart` lo sovrascrive con `ProviderScope(overrides: [...])` passando
/// il valore già caricato. Se qualcosa lo leggesse PRIMA di quella
/// sovrascrittura sarebbe un errore nel nostro codice di bootstrap, non una
/// situazione normale da gestire silenziosamente — per questo lancia
/// un'eccezione invece di avere un valore di default nascosto.
final committeeThemeProvider = Provider<CommitteeTheme>((ref) {
  throw StateError(
    'committeeThemeProvider non è stato sovrascritto: '
    'loadCommitteeTheme() va chiamato in main() prima di runApp().',
  );
});

/// Converte un colore in formato "#RRGGBB" (quello usato in
/// `config/<slug>/app/theme.json`) in un `Color` di Flutter, che internamente
/// vuole un intero ARGB (Alpha-Rosso-Verde-Blu, 8 bit ciascuno). Il nostro
/// JSON non ha un canale alpha (i colori del tema sono sempre opachi), quindi
/// anteponiamo "FF" (opacità piena) prima di fare il parsing esadecimale.
Color _colorFromHex(String hex) {
  final cleaned = hex.replaceFirst('#', '');
  return Color(int.parse('FF$cleaned', radix: 16));
}
