import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';

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
      await ref.read(authControllerProvider.notifier).login(
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
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          // Limita la larghezza del form su schermi larghi (web/desktop):
          // senza, un `TextField` si allargherebbe fino a riempire tutta la
          // finestra, illeggibile su un monitor largo.
          constraints: const BoxConstraints(maxWidth: 400),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    'login.title'.tr(),
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
                    controller: _identifierController,
                    decoration: InputDecoration(labelText: 'login.identifier_label'.tr()),
                    autofillHints: const [AutofillHints.username],
                    validator: (value) =>
                        (value == null || value.trim().isEmpty) ? 'login.identifier_required'.tr() : null,
                  ),
                  const SizedBox(height: 16),
                  TextFormField(
                    controller: _passwordController,
                    decoration: InputDecoration(labelText: 'login.password_label'.tr()),
                    obscureText: true,
                    autofillHints: const [AutofillHints.password],
                    // Il login va provato anche premendo Invio nell'ultimo
                    // campo, non solo cliccando il bottone.
                    onFieldSubmitted: (_) => _submit(),
                    validator: (value) =>
                        (value == null || value.isEmpty) ? 'login.password_required'.tr() : null,
                  ),
                  const SizedBox(height: 24),
                  FilledButton(
                    onPressed: _isSubmitting ? null : _submit,
                    child: _isSubmitting
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Text('login.submit'.tr()),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
