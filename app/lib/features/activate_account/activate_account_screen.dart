import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';

/// Apre il link di attivazione che un amministratore inoltra a mano al
/// volontario dopo avergli creato l'account (oggi non c'è ancora invio
/// email automatico, vedi `docs/funzionale/registry.md`). Il link ha forma
/// `.../attiva-account?token=<token-di-invito>`: `router.dart` estrae
/// `token` dalla query string e lo passa qui come [inviteToken].
class ActivateAccountScreen extends ConsumerStatefulWidget {
  const ActivateAccountScreen({required this.inviteToken, super.key});

  /// Può essere `null` se qualcuno apre l'URL senza il parametro `token`
  /// (link copiato male, ecc.) — in quel caso mostriamo un errore invece
  /// di un form che non potrebbe comunque funzionare.
  final String? inviteToken;

  @override
  ConsumerState<ActivateAccountScreen> createState() => _ActivateAccountScreenState();
}

class _ActivateAccountScreenState extends ConsumerState<ActivateAccountScreen> {
  final _formKey = GlobalKey<FormState>();
  final _passwordController = TextEditingController();
  final _confirmPasswordController = TextEditingController();

  bool _isSubmitting = false;
  bool _activated = false;
  String? _errorTranslationKey;

  @override
  void dispose() {
    _passwordController.dispose();
    _confirmPasswordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() {
      _isSubmitting = true;
      _errorTranslationKey = null;
    });

    try {
      await ref.read(authControllerProvider.notifier).activateAccount(
            token: widget.inviteToken!,
            password: _passwordController.text,
          );
      setState(() => _activated = true);
    } on ApiException catch (e) {
      setState(() => _errorTranslationKey = e.translationKey);
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 400),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: widget.inviteToken == null
                ? _buildMissingToken(context)
                : _activated
                    ? _buildSuccess(context)
                    : _buildForm(context),
          ),
        ),
      ),
    );
  }

  Widget _buildMissingToken(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(Icons.error_outline, color: Theme.of(context).colorScheme.error, size: 48),
        const SizedBox(height: 16),
        Text('activate_account.missing_token'.tr(), textAlign: TextAlign.center),
      ],
    );
  }

  Widget _buildSuccess(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(Icons.check_circle_outline, color: Theme.of(context).colorScheme.primary, size: 48),
        const SizedBox(height: 16),
        Text('activate_account.success'.tr(), textAlign: TextAlign.center),
        const SizedBox(height: 24),
        FilledButton(
          onPressed: () => context.go('/login'),
          child: Text('login.title'.tr()),
        ),
      ],
    );
  }

  Widget _buildForm(BuildContext context) {
    return Form(
      key: _formKey,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'activate_account.title'.tr(),
            style: Theme.of(context).textTheme.headlineMedium,
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 24),
          if (_errorTranslationKey != null) ...[
            Text(
              _errorTranslationKey!.tr(),
              style: TextStyle(color: Theme.of(context).colorScheme.error),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
          ],
          TextFormField(
            controller: _passwordController,
            decoration: InputDecoration(labelText: 'activate_account.password_label'.tr()),
            obscureText: true,
            autofillHints: const [AutofillHints.newPassword],
            validator: (value) =>
                (value == null || value.length < 8) ? 'login.password_required'.tr() : null,
          ),
          const SizedBox(height: 16),
          TextFormField(
            controller: _confirmPasswordController,
            decoration: InputDecoration(labelText: 'activate_account.confirm_password_label'.tr()),
            obscureText: true,
            autofillHints: const [AutofillHints.newPassword],
            // Il controllo "le due password coincidono" si fa qui: al
            // backend arriva solo la password definitiva, non ha motivo di
            // sapere che ne è stata digitata due volte per conferma.
            validator: (value) =>
                value != _passwordController.text ? 'activate_account.passwords_dont_match'.tr() : null,
          ),
          const SizedBox(height: 24),
          FilledButton(
            onPressed: _isSubmitting ? null : _submit,
            child: _isSubmitting
                ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                : Text('activate_account.submit'.tr()),
          ),
        ],
      ),
    );
  }
}
