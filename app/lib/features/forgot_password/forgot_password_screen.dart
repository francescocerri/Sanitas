import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';
import '../../core/widgets/auth_shell.dart';
import '../../core/widgets/error_banner.dart';

/// Punto d'ingresso per chi ha dimenticato la password: un solo campo
/// (email o username, come al login), poi un messaggio di conferma sempre
/// uguale — mai un errore "account non trovato". Il backend
/// (`POST /v1/password/reset/request`) risponde sempre 204 per lo stesso
/// motivo: rivelare se un identifier esiste sarebbe un modo per un
/// attaccante di scoprire chi ha un account, vedi
/// docs/adr/0024-recupero-password.md.
class ForgotPasswordScreen extends ConsumerStatefulWidget {
  const ForgotPasswordScreen({super.key});

  @override
  ConsumerState<ForgotPasswordScreen> createState() =>
      _ForgotPasswordScreenState();
}

class _ForgotPasswordScreenState extends ConsumerState<ForgotPasswordScreen> {
  final _formKey = GlobalKey<FormState>();
  final _identifierController = TextEditingController();

  bool _isSubmitting = false;
  bool _submitted = false;
  String? _errorTranslationKey;

  @override
  void dispose() {
    _identifierController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_formKey.currentState!.validate()) return;

    setState(() {
      _isSubmitting = true;
      _errorTranslationKey = null;
    });

    try {
      await ref
          .read(authControllerProvider.notifier)
          .requestPasswordReset(identifier: _identifierController.text.trim());
      setState(() => _submitted = true);
    } on ApiException catch (e) {
      setState(() => _errorTranslationKey = e.translationKey);
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return AuthShell(
      subtitle: 'forgot_password.tagline'.tr(),
      child: _submitted ? _buildSuccess(context) : _buildForm(context),
    );
  }

  Widget _buildSuccess(BuildContext context) {
    return Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.mark_email_read_outlined,
              color: Theme.of(context).colorScheme.primary,
              size: 56,
            ),
            const SizedBox(height: 20),
            Text(
              'forgot_password.success'.tr(),
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
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'forgot_password.title'.tr(),
                style: Theme.of(context).textTheme.headlineMedium
                    ?.copyWith(fontWeight: FontWeight.w800),
              ),
              const SizedBox(height: 8),
              Text(
                'forgot_password.subtitle'.tr(),
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
                controller: _identifierController,
                decoration: InputDecoration(
                  labelText: 'forgot_password.identifier_label'.tr(),
                  prefixIcon: const Icon(Icons.alternate_email_rounded),
                ),
                autofillHints: const [AutofillHints.username],
                onFieldSubmitted: (_) => _submit(),
                validator: (value) => (value == null || value.trim().isEmpty)
                    ? 'forgot_password.identifier_required'.tr()
                    : null,
              ),
              const SizedBox(height: 28),
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
                    : Text('forgot_password.submit'.tr()),
              ),
              const SizedBox(height: 16),
              TextButton(
                onPressed: () => context.go('/login'),
                child: Text('forgot_password.back_to_login'.tr()),
              ),
            ],
          ),
        )
        .animate()
        .fadeIn(delay: 100.ms, duration: 300.ms)
        .slideY(begin: 0.03, end: 0);
  }
}
