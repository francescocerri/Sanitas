import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_client.dart';
import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';
import '../../core/widgets/error_banner.dart';
import '../../core/widgets/password_visibility_toggle.dart';
import '../../core/widgets/theme_toggle_button.dart';
import 'user_profile.dart';

/// `FutureProvider` = un provider Riverpod pensato apposta per "un dato che
/// arriva da una chiamata asincrona (qui: una richiesta HTTP) e che la UI
/// deve poter mostrare come caricamento/dato/errore". Usarlo invece di
/// gestire manualmente `isLoading`/`error`/`data` con `setState` evita di
/// riscrivere la stessa logica in ogni schermata che fa una chiamata API.
final userProfileProvider = FutureProvider<UserProfile>((ref) async {
  // Un `FutureProvider` normale calcola il suo valore UNA VOLA SOLA e poi lo
  // tiene in cache per sempre, finché qualcuno non lo invalida esplicitamente
  // — utile per un dato che non cambia mai da solo, ma qui è un problema:
  // senza questa riga, dopo un logout e un nuovo login con un utente diverso
  // questa schermata continuerebbe a mostrare i dati del primo utente
  // caricati all'inizio della sessione dell'app. `ref.watch(...select(...))`
  // osserva SOLO il subject del JWT (l'id utente): cambia solo con un login
  // diverso, non ad ogni rotazione di refresh token, quindi la richiesta
  // riparte solo quando serve davvero, non ad ogni singolo refresh.
  ref.watch(
    authControllerProvider.select((session) => session.claims?.subject),
  );
  final dio = ref.read(apiDioProvider);
  final response = await dio.get<Map<String, dynamic>>('/v1/me');
  return UserProfile.fromJson(response.data!);
});

class ProfileScreen extends ConsumerWidget {
  const ProfileScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(userProfileProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'home.title'.tr(),
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: () => context.go('/home'),
        ),
        title: Text('profile.title'.tr()),
        actions: [
          const ThemeToggleButton(),
          IconButton(
            icon: const Icon(Icons.logout_rounded),
            tooltip: 'common.logout'.tr(),
            onPressed: () => ref.read(authControllerProvider.notifier).logout(),
          ),
          const SizedBox(width: 8),
        ],
      ),
      // `.when(...)` è il modo standard di Riverpod per gestire i 3 stati
      // possibili di un `FutureProvider`: ancora in corso, arrivato con
      // successo, o fallito con un errore — costringe a pensare a tutti e
      // tre, non solo al caso "felice".
      body: profileAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (error, stackTrace) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('errors.unknown'.tr()),
              const SizedBox(height: 8),
              // `ref.invalidate` scarta il risultato/errore memorizzato e fa
              // ripartire da zero il `FutureProvider` — l'equivalente di un
              // "riprova" per una chiamata fallita.
              TextButton(
                onPressed: () => ref.invalidate(userProfileProvider),
                child: Text('common.retry'.tr()),
              ),
            ],
          ),
        ),
        data: (profile) => _ProfileContent(profile: profile),
      ),
    );
  }
}

class _ProfileContent extends StatelessWidget {
  const _ProfileContent({required this.profile});

  final UserProfile profile;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final initial = profile.username.isEmpty
        ? '?'
        : profile.username[0].toUpperCase();

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Intestazione: avatar con l'iniziale dell'utente (non il
              // monogramma del comitato — quello identifica l'app nelle
              // schermate di autenticazione, questo identifica la PERSONA)
              // più nome utente ben in vista.
              Row(
                children: [
                  CircleAvatar(
                    radius: 32,
                    backgroundColor: colorScheme.primaryContainer,
                    child: Text(
                      initial,
                      style: TextStyle(
                        fontSize: 26,
                        fontWeight: FontWeight.w800,
                        color: colorScheme.onPrimaryContainer,
                      ),
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          profile.username,
                          style: Theme.of(context).textTheme.titleLarge,
                        ),
                        Text(
                          profile.email,
                          style: Theme.of(context).textTheme.bodyMedium
                              ?.copyWith(color: colorScheme.onSurfaceVariant),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 28),
              _SectionCard(
                children: [
                  _RolesRow(roles: profile.roles),
                  const Divider(height: 24),
                  _InfoRow(
                    icon: Icons.calendar_today_outlined,
                    label: 'profile.member_since_label'.tr(),
                    value: DateFormat.yMMMd(context.locale.toString())
                        .format(profile.createdAt),
                  ),
                ],
              ),
              const SizedBox(height: 20),
              Text(
                'profile.change_password_title'.tr(),
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 12),
              _SectionCard(children: const [_ChangePasswordForm()]),
            ],
          ),
        ),
      ),
    ).animate().fadeIn(duration: 300.ms);
  }
}

/// Raggruppamento visivo "a card" usato per le sezioni del profilo:
/// sfondo leggermente diverso dallo sfondo della pagina invece di un bordo
/// sottile — più morbido, coerente con lo stile "riempito" dei campi di
/// testo (vedi `committee_theme.dart`).
class _SectionCard extends StatelessWidget {
  const _SectionCard({required this.children});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest
            .withValues(alpha: 0.4),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: children,
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({
    required this.icon,
    required this.label,
    required this.value,
  });

  final IconData icon;
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Row(
      children: [
        Icon(icon, size: 20, color: colorScheme.onSurfaceVariant),
        const SizedBox(width: 12),
        Text(label, style: Theme.of(context).textTheme.labelLarge),
        const Spacer(),
        Text(value, style: Theme.of(context).textTheme.bodyMedium),
      ],
    );
  }
}

/// Riga dedicata ai ruoli: a differenza di [_InfoRow] (un'unica riga
/// etichetta+valore, va bene per un valore corto come una data), qui il
/// valore è una LISTA di lunghezza variabile — con più di un ruolo, un
/// unico `Text` con `join(', ')` andava in overflow orizzontale invece di
/// andare a capo. Ogni ruolo diventa un "chip" arrotondato dentro un `Wrap`,
/// che sposta l'elemento in eccesso sulla riga successiva da solo.
class _RolesRow extends StatelessWidget {
  const _RolesRow({required this.roles});

  final List<String> roles;

  /// I ruoli arrivano da `GET /v1/me` come slug tecnici (es.
  /// "shift_manager", non "Responsabile turni" — quel nome "carino" vive
  /// solo in `config/<slug>/registry/roles.json` lato backend ed è esposto
  /// solo da `GET /v1/roles`, protetto dal permesso `users:manage` che un
  /// volontario qualunque non ha). Qui ci limitiamo a un abbellimento
  /// puramente testuale (via spazi, maiuscole) che non richiede permessi
  /// né chiamate API in più — non è la vera etichetta italiana del ruolo,
  /// ma è comunque molto più leggibile dello slug grezzo.
  static String _prettify(String slug) {
    return slug
        .split('_')
        .where((word) => word.isNotEmpty)
        .map((word) => word[0].toUpperCase() + word.substring(1))
        .join(' ');
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Icon(
              Icons.badge_outlined,
              size: 20,
              color: colorScheme.onSurfaceVariant,
            ),
            const SizedBox(width: 12),
            Text(
              'profile.roles_label'.tr(),
              style: Theme.of(context).textTheme.labelLarge,
            ),
          ],
        ),
        const SizedBox(height: 12),
        if (roles.isEmpty)
          Text('—', style: Theme.of(context).textTheme.bodyMedium)
        else
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: roles
                .map(
                  (role) => Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 12,
                      vertical: 6,
                    ),
                    decoration: BoxDecoration(
                      color: colorScheme.primaryContainer,
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      _prettify(role),
                      style: Theme.of(context).textTheme.labelMedium?.copyWith(
                        color: colorScheme.onPrimaryContainer,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                )
                .toList(),
          ),
      ],
    );
  }
}

/// Isolato in un widget a parte (invece che dentro `ProfileScreen`) perché
/// ha uno stato locale tutto suo (i campi del form, se sta inviando) che non
/// deve rifarsi da capo se `userProfileProvider` viene ricaricato per un
/// altro motivo.
class _ChangePasswordForm extends ConsumerStatefulWidget {
  const _ChangePasswordForm();

  @override
  ConsumerState<_ChangePasswordForm> createState() =>
      _ChangePasswordFormState();
}

class _ChangePasswordFormState extends ConsumerState<_ChangePasswordForm> {
  final _formKey = GlobalKey<FormState>();
  final _oldPasswordController = TextEditingController();
  final _newPasswordController = TextEditingController();

  bool _obscureOld = true;
  bool _obscureNew = true;
  bool _isSubmitting = false;
  String? _errorTranslationKey;
  bool _showSuccess = false;

  @override
  void dispose() {
    _oldPasswordController.dispose();
    _newPasswordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() {
      _isSubmitting = true;
      _errorTranslationKey = null;
      _showSuccess = false;
    });

    try {
      final dio = ref.read(apiDioProvider);
      await dio.post<void>(
        '/v1/password/change',
        data: {
          'old_password': _oldPasswordController.text,
          'new_password': _newPasswordController.text,
        },
      );
      _oldPasswordController.clear();
      _newPasswordController.clear();
      if (mounted) setState(() => _showSuccess = true);
    } on ApiException catch (e) {
      setState(() => _errorTranslationKey = e.translationKey);
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (_errorTranslationKey != null) ...[
            ErrorBanner(message: _errorTranslationKey!.tr()),
            const SizedBox(height: 12),
          ],
          if (_showSuccess) ...[
            SuccessBanner(message: 'profile.change_password_success'.tr()),
            const SizedBox(height: 12),
          ],
          TextFormField(
            controller: _oldPasswordController,
            decoration: InputDecoration(
              labelText: 'profile.old_password_label'.tr(),
              prefixIcon: const Icon(Icons.lock_outline_rounded),
              suffixIcon: PasswordVisibilityToggle(
                obscured: _obscureOld,
                onPressed: () => setState(() => _obscureOld = !_obscureOld),
              ),
            ),
            obscureText: _obscureOld,
            autofillHints: const [AutofillHints.password],
            validator: (value) => (value == null || value.isEmpty)
                ? 'login.password_required'.tr()
                : null,
          ),
          const SizedBox(height: 16),
          TextFormField(
            controller: _newPasswordController,
            decoration: InputDecoration(
              labelText: 'profile.new_password_label'.tr(),
              prefixIcon: const Icon(Icons.lock_reset_rounded),
              suffixIcon: PasswordVisibilityToggle(
                obscured: _obscureNew,
                onPressed: () => setState(() => _obscureNew = !_obscureNew),
              ),
            ),
            obscureText: _obscureNew,
            autofillHints: const [AutofillHints.newPassword],
            validator: (value) => (value == null || value.length < 8)
                ? 'login.password_required'.tr()
                : null,
          ),
          const SizedBox(height: 20),
          FilledButton(
            onPressed: _isSubmitting ? null : _submit,
            child: _isSubmitting
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: Colors.white,
                    ),
                  )
                : Text('profile.change_password_submit'.tr()),
          ),
        ],
      ),
    );
  }
}
