import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';
import '../../core/widgets/auth_shell.dart';
import '../../core/widgets/error_banner.dart';
import '../../core/widgets/password_visibility_toggle.dart';

/// Apre il link ricevuto via email dopo aver richiesto un reset password
/// (`ForgotPasswordScreen`). Il link ha forma `.../reset-password?token=...`:
/// `router.dart` estrae `token` dalla query string e lo passa qui come
/// [resetToken]. Stessa struttura a 3 stati di `ActivateAccountScreen`
/// (token mancante / form / successo), stesso backend (`ConsumeToken`),
/// solo con purpose `"password_reset"` invece di `"invite"` — vedi
/// docs/adr/0024-recupero-password.md.
class ResetPasswordScreen extends ConsumerStatefulWidget {
  const ResetPasswordScreen({required this.resetToken, super.key});

  /// Può essere `null` se qualcuno apre l'URL senza il parametro `token`.
  final String? resetToken;

  @override
  ConsumerState<ResetPasswordScreen> createState() =>
      _ResetPasswordScreenState();
}

class _ResetPasswordScreenState extends ConsumerState<ResetPasswordScreen> {
  final _formKey = GlobalKey<FormState>();
  final _passwordController = TextEditingController();
  final _confirmPasswordController = TextEditingController();

  bool _obscurePassword = true;
  bool _isSubmitting = false;
  bool _reset = false;
  String? _errorTranslationKey;

  bool get _isPasswordValid => _passwordController.text.length >= 8;

  bool get _canSubmit =>
      !_isSubmitting &&
      _isPasswordValid &&
      _confirmPasswordController.text == _passwordController.text;

  @override
  void dispose() {
    _passwordController.dispose();
    _confirmPasswordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_canSubmit || !_formKey.currentState!.validate()) return;

    setState(() {
      _isSubmitting = true;
      _errorTranslationKey = null;
    });

    try {
      await ref
          .read(authControllerProvider.notifier)
          .confirmPasswordReset(
            token: widget.resetToken!,
            password: _passwordController.text,
          );
      setState(() => _reset = true);
    } on ApiException catch (e) {
      setState(() => _errorTranslationKey = e.translationKey);
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return AuthShell(
      subtitle: 'reset_password.tagline'.tr(),
      child: widget.resetToken == null
          ? _buildMissingToken(context)
          : _reset
          ? _buildSuccess(context)
          : _buildForm(context),
    );
  }

  Widget _buildMissingToken(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          Icons.link_off_rounded,
          color: Theme.of(context).colorScheme.error,
          size: 56,
        ),
        const SizedBox(height: 20),
        Text(
          'reset_password.missing_token'.tr(),
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodyLarge,
        ),
      ],
    ).animate().fadeIn(duration: 300.ms);
  }

  Widget _buildSuccess(BuildContext context) {
    return Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.check_circle_outline_rounded,
              color: Theme.of(context).colorScheme.primary,
              size: 56,
            ),
            const SizedBox(height: 20),
            Text(
              'reset_password.success'.tr(),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 28),
            FilledButton(
              onPressed: () => context.go('/login'),
              child: Text('login.title'.tr()),
            ),
          ],
        )
        .animate()
        .fadeIn(duration: 300.ms)
        .scale(begin: const Offset(0.94, 0.94), curve: Curves.easeOutBack);
  }

  Widget _buildForm(BuildContext context) {
    return Form(
          key: _formKey,
          onChanged: () => setState(() {}),
          autovalidateMode: AutovalidateMode.onUserInteraction,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'reset_password.title'.tr(),
                style: Theme.of(context).textTheme.headlineMedium
                    ?.copyWith(fontWeight: FontWeight.w800),
              ),
              const SizedBox(height: 8),
              Text(
                'reset_password.subtitle'.tr(),
                style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 32),
              if (_errorTranslationKey != null) ...[
                ErrorBanner(message: _errorTranslationKey!.tr()),
                const SizedBox(height: 16),
              ],
              TextFormField(
                controller: _passwordController,
                decoration: InputDecoration(
                  labelText: 'reset_password.password_label'.tr(),
                  prefixIcon: const Icon(Icons.lock_outline_rounded),
                  suffixIcon: PasswordVisibilityToggle(
                    obscured: _obscurePassword,
                    onPressed: () =>
                        setState(() => _obscurePassword = !_obscurePassword),
                  ),
                ),
                obscureText: _obscurePassword,
                autofillHints: const [AutofillHints.newPassword],
                validator: (_) => !_isPasswordValid
                    ? 'reset_password.password_invalid'.tr()
                    : null,
              ),
              const SizedBox(height: 8),
              Text(
                'reset_password.password_requirements'.tr(),
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _confirmPasswordController,
                decoration: InputDecoration(
                  labelText: 'reset_password.confirm_password_label'.tr(),
                  prefixIcon: const Icon(Icons.lock_outline_rounded),
                ),
                obscureText: _obscurePassword,
                autofillHints: const [AutofillHints.newPassword],
                validator: (value) => value != _passwordController.text
                    ? 'reset_password.passwords_dont_match'.tr()
                    : null,
              ),
              const SizedBox(height: 28),
              FilledButton(
                onPressed: _canSubmit ? _submit : null,
                child: _isSubmitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : Text('reset_password.submit'.tr()),
              ),
            ],
          ),
        )
        .animate()
        .fadeIn(delay: 100.ms, duration: 300.ms)
        .slideY(begin: 0.03, end: 0);
  }
}
