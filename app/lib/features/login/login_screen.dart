import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';
import '../../core/widgets/auth_shell.dart';
import '../../core/widgets/error_banner.dart';

/// `ConsumerStatefulWidget` = uno `StatefulWidget` normale di Flutter (serve
/// perché questo widget ha uno stato tutto suo: il testo nei campi, se sta
/// caricando, l'eventuale errore) MA con in più accesso a Riverpod tramite
/// `ref` (per leggere/chiamare `authControllerProvider`). È il widget da
/// usare ogni volta che serve sia stato locale (`setState`) sia stato
/// globale via Riverpod nello stesso widget.
class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  // `GlobalKey<FormState>` è come il widget `Form` si fa "chiamare indietro"
  // da fuori (qui: per chiedergli "i tuoi campi sono validi?" al submit).
  final _formKey = GlobalKey<FormState>();
  final _identifierController = TextEditingController();
  final _passwordController = TextEditingController();

  bool _isSubmitting = false;
  bool _obscurePassword = true;
  String? _errorTranslationKey;

  @override
  void dispose() {
    // I `TextEditingController` tengono risorse native (non solo Dart):
    // vanno sempre chiusi esplicitamente quando il widget viene rimosso,
    // altrimenti si accumula memoria non liberata (memory leak).
    _identifierController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    // `validate()` esegue tutti i `validator:` dei campi sotto e mostra i
    // relativi messaggi di errore sotto ciascuno; torna `false` se almeno
    // uno fallisce, nel qual caso non ha senso proseguire con la chiamata.
    if (!_formKey.currentState!.validate()) return;

    setState(() {
      _isSubmitting = true;
      _errorTranslationKey = null;
    });

    try {
      await ref
          .read(authControllerProvider.notifier)
          .login(
            identifier: _identifierController.text.trim(),
            password: _passwordController.text,
          );
      // Nessuna navigazione esplicita qui: il login riuscito cambia lo
      // stato di `authControllerProvider`, e `router.dart` reindirizza da
      // solo a "/profilo" appena lo osserva (vedi commento in quel file).
    } on ApiException catch (e) {
      setState(() => _errorTranslationKey = e.translationKey);
    } finally {
      if (mounted) setState(() => _isSubmitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    // `flutter_animate` applica animazioni componibili con una sintassi a
    // catena (`.animate().fadeIn()...`): qui ogni gruppo di elementi parte
    // leggermente in ritardo rispetto al precedente ("stagger"), un effetto
    // a cascata molto comune nelle UI curate invece di far comparire tutto
    // insieme di scatto.
    return AuthShell(
      subtitle: 'login.tagline'.tr(),
      child:
          Form(
                key: _formKey,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'login.title'.tr(),
                      style: Theme.of(context).textTheme.headlineMedium
                          ?.copyWith(fontWeight: FontWeight.w800),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      'login.subtitle'.tr(),
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
                        labelText: 'login.identifier_label'.tr(),
                        prefixIcon: const Icon(Icons.alternate_email_rounded),
                      ),
                      autofillHints: const [AutofillHints.username],
                      validator: (value) =>
                          (value == null || value.trim().isEmpty)
                          ? 'login.identifier_required'.tr()
                          : null,
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _passwordController,
                      decoration: InputDecoration(
                        labelText: 'login.password_label'.tr(),
                        prefixIcon: const Icon(Icons.lock_outline_rounded),
                        suffixIcon: IconButton(
                          icon: Icon(
                            _obscurePassword
                                ? Icons.visibility_outlined
                                : Icons.visibility_off_outlined,
                          ),
                          onPressed: () => setState(
                            () => _obscurePassword = !_obscurePassword,
                          ),
                        ),
                      ),
                      obscureText: _obscurePassword,
                      autofillHints: const [AutofillHints.password],
                      // Il login va provato anche premendo Invio nell'ultimo
                      // campo, non solo cliccando il bottone.
                      onFieldSubmitted: (_) => _submit(),
                      validator: (value) => (value == null || value.isEmpty)
                          ? 'login.password_required'.tr()
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
                          : Text('login.submit'.tr()),
                    ),
                  ],
                ),
              )
              .animate()
              .fadeIn(delay: 100.ms, duration: 300.ms)
              .slideY(begin: 0.03, end: 0),
    );
  }
}
