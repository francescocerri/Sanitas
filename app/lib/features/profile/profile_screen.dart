import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';
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
  ref.watch(authControllerProvider.select((session) => session.claims?.subject));
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
        title: Text('profile.title'.tr()),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: 'common.logout'.tr(),
            onPressed: () => ref.read(authControllerProvider.notifier).logout(),
          ),
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
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 480),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _InfoRow(label: 'profile.email_label'.tr(), value: profile.email),
            _InfoRow(label: 'profile.username_label'.tr(), value: profile.username),
            _InfoRow(
              label: 'profile.roles_label'.tr(),
              value: profile.roles.isEmpty ? '—' : profile.roles.join(', '),
            ),
            const SizedBox(height: 8),
            Text(
              'profile.member_since'.tr(
                namedArgs: {'date': DateFormat.yMMMd(context.locale.toString()).format(profile.createdAt)},
              ),
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 32),
            const Divider(),
            const SizedBox(height: 16),
            Text('profile.change_password_title'.tr(), style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 16),
            const _ChangePasswordForm(),
          ],
        ),
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(width: 120, child: Text(label, style: Theme.of(context).textTheme.labelLarge)),
          Expanded(child: Text(value)),
        ],
      ),
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
  ConsumerState<_ChangePasswordForm> createState() => _ChangePasswordFormState();
}

class _ChangePasswordFormState extends ConsumerState<_ChangePasswordForm> {
  final _formKey = GlobalKey<FormState>();
  final _oldPasswordController = TextEditingController();
  final _newPasswordController = TextEditingController();

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
      await dio.post<void>('/v1/password/change', data: {
        'old_password': _oldPasswordController.text,
        'new_password': _newPasswordController.text,
      });
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
            Text(
              _errorTranslationKey!.tr(),
              style: TextStyle(color: Theme.of(context).colorScheme.error),
            ),
            const SizedBox(height: 8),
          ],
          if (_showSuccess) ...[
            Text(
              'profile.change_password_success'.tr(),
              style: TextStyle(color: Theme.of(context).colorScheme.primary),
            ),
            const SizedBox(height: 8),
          ],
          TextFormField(
            controller: _oldPasswordController,
            decoration: InputDecoration(labelText: 'profile.old_password_label'.tr()),
            obscureText: true,
            autofillHints: const [AutofillHints.password],
            validator: (value) =>
                (value == null || value.isEmpty) ? 'login.password_required'.tr() : null,
          ),
          const SizedBox(height: 16),
          TextFormField(
            controller: _newPasswordController,
            decoration: InputDecoration(labelText: 'profile.new_password_label'.tr()),
            obscureText: true,
            autofillHints: const [AutofillHints.newPassword],
            validator: (value) =>
                (value == null || value.length < 8) ? 'login.password_required'.tr() : null,
          ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: _isSubmitting ? null : _submit,
            child: _isSubmitting
                ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                : Text('profile.change_password_submit'.tr()),
          ),
        ],
      ),
    );
  }
}
