import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Chiaro/scuro è una preferenza PERSONALE di chi usa l'app, separata dal
/// colore del brand del comitato (quello viene da `committee_theme.dart` ed
/// è deciso dal fork, non dall'utente) — vedi `docs/adr/0022-frontend-flutter.md`.
/// Persistita con `shared_preferences` (semplice storage chiave-valore su
/// disco, non cifrato: va benissimo, non è un dato sensibile come il
/// refresh token, che invece usa `flutter_secure_storage`).
class ThemeModeController extends Notifier<ThemeMode> {
  static const _prefsKey = 'sanitas_theme_mode';

  @override
  ThemeMode build() {
    // Come per `AuthController`, `build()` non può essere `async`: partiamo
    // dal default di sistema e, se l'utente aveva scelto qualcosa in
    // passato, lo carichiamo subito dopo in background.
    Future.microtask(_loadSaved);
    return ThemeMode.system;
  }

  Future<void> _loadSaved() async {
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString(_prefsKey);
    if (saved != null) {
      state = ThemeMode.values.byName(saved);
    }
  }

  Future<void> setThemeMode(ThemeMode mode) async {
    state = mode;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_prefsKey, mode.name);
  }
}

final themeModeControllerProvider = NotifierProvider<ThemeModeController, ThemeMode>(
  ThemeModeController.new,
);
