import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../theme/committee_theme.dart';
import 'brand_mark.dart';
import 'theme_toggle_button.dart';

/// Piccola scritta "SANITAS" sopra il monogramma del comitato: il comitato
/// (es. "CRI Pavullo") resta il protagonista visivo — è la sua app, non
/// quella di Sanitas — ma il nome della piattaforma resta comunque leggibile
/// da qualche parte, così chi la usa sa cosa sta usando. "Sanitas" non è un
/// dato di configurazione del comitato (non viola il contratto di
/// forkabilità in `CLAUDE.md`): è il nome del progetto software stesso,
/// scelto apposta neutro rispetto al brand di un singolo comitato (vedi
/// ADR-0001) — resta lo stesso per ogni fork, non è personalizzabile.
class _SanitasWordmark extends StatelessWidget {
  const _SanitasWordmark({required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) {
    return Text(
      'app.title'.tr().toUpperCase(),
      style: Theme.of(context).textTheme.labelMedium?.copyWith(
        color: color,
        fontWeight: FontWeight.w700,
        letterSpacing: 3,
      ),
    );
  }
}

/// Involucro condiviso dalle schermate di autenticazione (login, attivazione
/// account): su schermi larghi (web/desktop) mostra un pannello brandizzato
/// a sinistra — un pattern comune nelle app gestionali/SaaS moderne, che dà
/// identità visiva subito riconoscibile invece di un form anonimo in mezzo
/// a una pagina bianca — e la card del form a destra. Su schermi stretti
/// (mobile) il pannello sparirebbe via schermo utile: resta solo il
/// monogramma sopra il form.
class AuthShell extends ConsumerWidget {
  const AuthShell({required this.child, this.subtitle, super.key});

  final Widget child;

  /// Frase breve mostrata sotto il nome del comitato nel pannello largo
  /// (es. "Accedi per gestire i tuoi turni"). Facoltativa.
  final String? subtitle;

  /// Sopra questa larghezza (px logici) si passa al layout a due colonne.
  /// 840 è la soglia classica per "tablet in orizzontale / desktop" usata
  /// da molte guide di Material Design.
  static const _wideBreakpoint = 840.0;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final committeeTheme = ref.watch(committeeThemeProvider);
    final isWide = MediaQuery.sizeOf(context).width >= _wideBreakpoint;

    final formCard =
        Center(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(32),
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 420),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (!isWide) ...[
                        Center(
                          child: _SanitasWordmark(
                            color: Theme.of(context)
                                .colorScheme
                                .onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 12),
                        const Center(child: BrandMark(size: 72)),
                        const SizedBox(height: 32),
                      ],
                      child,
                    ],
                  ),
                ),
              ),
              // Una piccola animazione di ingresso (dissolvenza + leggero
              // spostamento verso l'alto) rende la comparsa della schermata meno
              // "di scatto" — `flutter_animate` la applica con una riga sola,
              // senza dover scrivere un `AnimationController` a mano.
            )
            .animate()
            .fadeIn(duration: 350.ms)
            .slideY(begin: 0.04, end: 0, curve: Curves.easeOutCubic);

    if (!isWide) {
      return Scaffold(
        body: Stack(
          children: [
            formCard,
            const SafeArea(
              child: Align(
                alignment: Alignment.topRight,
                child: ThemeToggleButton(),
              ),
            ),
          ],
        ),
      );
    }

    return Scaffold(
      body: Stack(
        children: [
          Row(
            children: [
              Expanded(
                child: Container(
                  decoration: BoxDecoration(
                    gradient: LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [
                        committeeTheme.primary,
                        committeeTheme.secondary,
                      ],
                    ),
                  ),
                  alignment: Alignment.center,
                  padding: const EdgeInsets.all(48),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const _SanitasWordmark(color: Colors.white70),
                      const SizedBox(height: 16),
                      const BrandMark(size: 96, inverse: true),
                      const SizedBox(height: 28),
                      Text(
                        committeeTheme.committeeName,
                        style: Theme.of(context).textTheme.headlineMedium
                            ?.copyWith(
                              color: Colors.white,
                              fontWeight: FontWeight.w800,
                            ),
                        textAlign: TextAlign.center,
                      ),
                      if (subtitle != null) ...[
                        const SizedBox(height: 12),
                        Text(
                          subtitle!,
                          style: Theme.of(context).textTheme.bodyLarge
                              ?.copyWith(color: Colors.white70),
                          textAlign: TextAlign.center,
                        ),
                      ],
                    ],
                  ),
                ),
              ),
              Expanded(child: formCard),
            ],
          ),
          const SafeArea(
            child: Align(
              alignment: Alignment.topRight,
              child: ThemeToggleButton(),
            ),
          ),
        ],
      ),
    );
  }
}
