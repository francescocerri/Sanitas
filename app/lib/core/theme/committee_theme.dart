import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_fonts/google_fonts.dart';

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

  /// La prima lettera (maiuscola) del nome del comitato, es. "C" per "CRI
  /// Pavullo": usata come "monogramma" al posto di un vero logo — nessun
  /// comitato è tenuto a fornirci un file immagine, un'iniziale colorata è
  /// un placeholder di marca pulito e comunissimo (Slack, Notion, Google
  /// fanno lo stesso per gli avatar). Vedi `lib/core/widgets/brand_mark.dart`.
  String get monogram => committeeName.trim().isEmpty
      ? '?'
      : committeeName.trim()[0].toUpperCase();

  /// Costruisce il `ThemeData` di Material 3 a partire da un solo colore
  /// "seme" (il primary del comitato): `ColorScheme.fromSeed` genera da solo
  /// tutte le sfumature necessarie (colori per bottoni, sfondi, testo su
  /// sfondo colorato, ecc.) rispettando le regole di contrasto di Material
  /// Design — non dobbiamo scegliere a mano decine di colori per ogni tema.
  /// [brightness] separa questo asse (chiaro/scuro) dal colore del brand:
  /// stesso primary, ma una palette diversa in dark mode.
  ///
  /// Oltre ai colori, qui si personalizzano anche tipografia, forma dei
  /// campi di testo e dei bottoni: è la differenza fra un'app che "sembra
  /// Material di default" e una con un'identità propria, pur restando
  /// dentro le regole di Material 3 (niente reinventato da zero).
  ThemeData toThemeData(Brightness brightness) {
    final colorScheme = ColorScheme.fromSeed(
      seedColor: primary,
      brightness: brightness,
      surface: brightness == Brightness.light ? surface : null,
    );

    // "Plus Jakarta Sans" al posto del font di sistema: da solo cambia
    // moltissimo la percezione di cura visiva, per zero lavoro di design.
    // `GoogleFonts.xTextTheme` prende il text theme di Material di default
    // e ne sostituisce solo il font, mantenendo tutte le dimensioni/pesi
    // già bilanciati da Material Design.
    final textTheme = GoogleFonts.plusJakartaSansTextTheme(
      brightness == Brightness.light
          ? ThemeData.light().textTheme
          : ThemeData.dark().textTheme,
    );

    const fieldRadius = 16.0;
    const buttonRadius = 16.0;

    return ThemeData(
      useMaterial3: true,
      colorScheme: colorScheme,
      textTheme: textTheme,
      scaffoldBackgroundColor: colorScheme.surface,
      appBarTheme: AppBarTheme(
        backgroundColor: colorScheme.surface,
        foregroundColor: colorScheme.onSurface,
        elevation: 0,
        scrolledUnderElevation: 0,
        titleTextStyle: textTheme.titleLarge?.copyWith(
          fontWeight: FontWeight.w700,
        ),
      ),
      // Campi di testo "riempiti" e arrotondati invece del sottile
      // sottolineato di default: più moderni, più facili da toccare su
      // mobile, e il colore di riempimento crea contrasto col resto della
      // pagina senza bisogno di un bordo marcato.
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 20,
          vertical: 18,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(fieldRadius),
          borderSide: BorderSide.none,
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(fieldRadius),
          borderSide: BorderSide.none,
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(fieldRadius),
          borderSide: BorderSide(color: colorScheme.primary, width: 2),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(fieldRadius),
          borderSide: BorderSide(color: colorScheme.error, width: 1.5),
        ),
        labelStyle: TextStyle(color: colorScheme.onSurfaceVariant),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(56),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(buttonRadius),
          ),
          textStyle: textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(buttonRadius),
          ),
        ),
      ),
    );
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
