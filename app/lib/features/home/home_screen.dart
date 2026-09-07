import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/auth/auth_controller.dart';
import '../../core/widgets/theme_toggle_button.dart';
import '../create_user/create_user_screen.dart' show canManageUsersProvider;

/// Schermata su cui si atterra dopo il login (vedi `router.dart`): un
/// punto di partenza unico da cui raggiungere le varie aree dell'app,
/// invece di andare dritti al profilo. Per ora ci sono solo "Il mio
/// profilo" e "Nuovo utente" (quest'ultima solo per chi ha il permesso
/// `users:manage`) — man mano che nasceranno le altre aree previste dal
/// progetto (turni, mezzi e magazzino, servizi ed emergenze) diventeranno
/// altrettante card qui, al posto di quelle "in arrivo" disattivate.
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  /// Sopra questa larghezza la griglia passa a 3 colonne, sopra questa
  /// (ma sotto la precedente) a 2, altrimenti resta a 1 — le stesse
  /// soglie usate in tutta l'app per "stretto/medio/largo".
  static const _threeColumnBreakpoint = 900.0;
  static const _twoColumnBreakpoint = 560.0;

  /// Altezza fissa di OGNI card, indipendentemente da quante colonne ci
  /// sono o da quanto testo contiene: è quello che garantisce che tutte
  /// le card sembrino della stessa dimensione, il problema lamentato con
  /// la versione precedente (un `Wrap` di card a larghezza fissa, alte
  /// quanto il loro contenuto — con o senza sottotitolo/badge venivano
  /// fuori altezze diverse).
  static const _cardHeight = 176.0;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final username = ref.watch(authControllerProvider).claims?.username ?? '';
    final canManageUsers = ref.watch(canManageUsersProvider);

    final activeCards = [
      _HomeCard(
        icon: Icons.person_outline_rounded,
        title: 'home.profile_title'.tr(),
        subtitle: 'home.profile_subtitle'.tr(),
        onTap: () => context.go('/profilo'),
      ),
      if (canManageUsers)
        _HomeCard(
          icon: Icons.person_add_alt_1_rounded,
          title: 'home.create_user_title'.tr(),
          subtitle: 'home.create_user_subtitle'.tr(),
          badge: 'home.admin_badge'.tr(),
          onTap: () => context.go('/users/new'),
        ),
      // Aperta a chiunque: l'elenco utenti è una rubrica di sola lettura
      // per chi non ha `users:manage` (che invece vede anche la matitina
      // per modificare i ruoli) — vedi manage_users_screen.dart.
      _HomeCard(
        icon: Icons.manage_accounts_outlined,
        title: 'home.manage_users_title'.tr(),
        subtitle: 'home.manage_users_subtitle'.tr(),
        onTap: () => context.go('/users'),
      ),
    ];

    final comingSoonCards = [
      _HomeCard(
        icon: Icons.calendar_month_outlined,
        title: 'home.shifts_title'.tr(),
        subtitle: 'home.shifts_subtitle'.tr(),
        badge: 'home.coming_soon_badge'.tr(),
      ),
      _HomeCard(
        icon: Icons.local_shipping_outlined,
        title: 'home.fleet_title'.tr(),
        subtitle: 'home.fleet_subtitle'.tr(),
        badge: 'home.coming_soon_badge'.tr(),
      ),
      _HomeCard(
        icon: Icons.emergency_outlined,
        title: 'home.emergencies_title'.tr(),
        subtitle: 'home.emergencies_subtitle'.tr(),
        badge: 'home.coming_soon_badge'.tr(),
      ),
    ];

    return Scaffold(
      appBar: AppBar(
        actions: [
          IconButton(
            icon: const Icon(Icons.person_outline_rounded),
            tooltip: 'profile.title'.tr(),
            onPressed: () => context.go('/profilo'),
          ),
          const ThemeToggleButton(),
          IconButton(
            icon: const Icon(Icons.logout_rounded),
            tooltip: 'common.logout'.tr(),
            onPressed: () => ref.read(authControllerProvider.notifier).logout(),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 960),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'app.title'.tr().toUpperCase(),
                  style: theme.textTheme.labelLarge?.copyWith(
                    color: colorScheme.primary,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 1.5,
                  ),
                ),
                const SizedBox(height: 10),
                Text(
                  'home.greeting'.tr(namedArgs: {'username': username}),
                  style: theme.textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.w800,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'home.subtitle'.tr(),
                  style: theme.textTheme.bodyLarge?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 28),
                _HomeCardGrid(cards: activeCards, cardHeight: _cardHeight),
                const SizedBox(height: 32),
                Text(
                  'home.coming_soon_label'.tr(),
                  style: theme.textTheme.labelLarge?.copyWith(
                    color: colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 12),
                _HomeCardGrid(cards: comingSoonCards, cardHeight: _cardHeight),
              ],
            ),
          ),
        ),
      ).animate().fadeIn(duration: 300.ms).slideY(begin: 0.03, end: 0),
    );
  }
}

/// Griglia responsiva usata due volte in [HomeScreen] (card attive, card
/// "in arrivo"): sceglie il numero di colonne in base allo spazio
/// disponibile e forza ogni cella alla stessa [cardHeight] fissa,
/// qualunque sia il numero di colonne — così le card riempiono sempre
/// tutta la larghezza a disposizione (niente spazio vuoto a destra su
/// schermi larghi, niente card più strette del previsto su mobile) e
/// sono sempre tutte della stessa dimensione.
class _HomeCardGrid extends StatelessWidget {
  const _HomeCardGrid({required this.cards, required this.cardHeight});

  final List<Widget> cards;
  final double cardHeight;

  static const _spacing = 16.0;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final columns =
            constraints.maxWidth >= HomeScreen._threeColumnBreakpoint
            ? 3
            : constraints.maxWidth >= HomeScreen._twoColumnBreakpoint
            ? 2
            : 1;
        final columnWidth =
            (constraints.maxWidth - _spacing * (columns - 1)) / columns;

        return GridView.count(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          crossAxisCount: columns,
          crossAxisSpacing: _spacing,
          mainAxisSpacing: _spacing,
          childAspectRatio: columnWidth / cardHeight,
          children: cards,
        );
      },
    );
  }
}

/// Una card della home: attiva (con `onTap`, icona/testo a piena
/// opacità) o "in arrivo" (senza `onTap`, sbiadita, badge "Prossimamente").
/// Stessa struttura interna per tutte — icona, titolo, sottotitolo,
/// sempre presenti — e il badge (se c'è) ancorato in fondo con uno
/// `Spacer`: così, anche se il testo cambia lunghezza, il contenuto resta
/// allineato allo stesso modo in ogni card.
class _HomeCard extends StatelessWidget {
  const _HomeCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    this.badge,
    this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final String? badge;
  final VoidCallback? onTap;

  bool get _disabled => onTap == null;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;

    final content = Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerHighest.withValues(
          alpha: _disabled ? 0.2 : 0.4,
        ),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            icon,
            size: 26,
            color: _disabled
                ? colorScheme.onSurfaceVariant.withValues(alpha: 0.5)
                : colorScheme.primary,
          ),
          const SizedBox(height: 14),
          Text(
            title,
            style: textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.w700,
              color: _disabled
                  ? colorScheme.onSurfaceVariant.withValues(alpha: 0.6)
                  : null,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            subtitle,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: textTheme.bodySmall?.copyWith(
              color: colorScheme.onSurfaceVariant,
            ),
          ),
          const Spacer(),
          if (badge != null)
            Align(
              alignment: Alignment.centerLeft,
              child: Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 4,
                ),
                decoration: BoxDecoration(
                  color: _disabled
                      ? colorScheme.surfaceContainerHighest
                      : colorScheme.primaryContainer,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  badge!,
                  style: textTheme.labelSmall?.copyWith(
                    color: _disabled
                        ? colorScheme.onSurfaceVariant
                        : colorScheme.onPrimaryContainer,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ),
        ],
      ),
    );

    if (_disabled) return content;

    // `Material` + `InkWell` invece di `GestureDetector`: dà il feedback a
    // "increspatura" (ripple) standard di Material Design al tocco, che ci
    // si aspetta da un elemento cliccabile — `GestureDetector` da solo non
    // lo fornisce.
    return Material(
      color: Colors.transparent,
      borderRadius: BorderRadius.circular(20),
      child: InkWell(
        borderRadius: BorderRadius.circular(20),
        onTap: onTap,
        child: content,
      ),
    );
  }
}
