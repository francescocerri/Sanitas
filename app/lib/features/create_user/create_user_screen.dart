import 'package:dio/dio.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_client.dart';
import '../../core/api_exception.dart';
import '../../core/auth/auth_controller.dart';
import '../../core/widgets/error_banner.dart';
import '../../core/widgets/theme_toggle_button.dart';

final canManageUsersProvider = Provider<bool>(
  (ref) => ref.watch(
    authControllerProvider.select(
      (session) => session.claims?.hasPermission('users:manage') ?? false,
    ),
  ),
);

class AvailableRole {
  const AvailableRole({required this.slug, required this.displayName});
  final String slug;
  final String displayName;
}

ApiException _userError(DioException error) => ApiException.fromDioException(
  error,
  statusToKey: const {
    400: 'errors.invalid_payload',
    403: 'create_user.forbidden',
    409: 'create_user.duplicate',
  },
);

// Il catalogo ruoli è di sola lettura per chiunque sia autenticato (vedi
// GET /v1/roles in services/registry): nessun controllo su
// canManageUsersProvider qui, riusato anche da manage_users_screen.dart
// per mostrare il nome dei ruoli a chi non ha users:manage. Chi crea
// utenti resta comunque protetto altrove: la rotta /users/new è
// raggiungibile solo con quel permesso (vedi router.dart) e
// POST /v1/users lo richiede anche lato backend.
final availableRolesProvider = FutureProvider.autoDispose<List<AvailableRole>>((
  ref,
) async {
  ref.watch(
    authControllerProvider.select((session) => session.claims?.subject),
  );
  try {
    final response = await ref
        .watch(apiDioProvider)
        .get<List<dynamic>>('/v1/roles');
    return response.data!
        .map(
          (entry) => AvailableRole(
            slug: entry['slug'] as String,
            displayName: entry['display_name'] as String,
          ),
        )
        .toList();
  } on DioException catch (error) {
    throw _userError(error);
  }
});

class CreateUserScreen extends ConsumerStatefulWidget {
  const CreateUserScreen({super.key});

  @override
  ConsumerState<CreateUserScreen> createState() => _CreateUserScreenState();
}

class _CreateUserScreenState extends ConsumerState<CreateUserScreen> {
  final _formKey = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _username = TextEditingController();
  final _roles = <String>{};
  bool _submitting = false;
  String? _error;
  String? _inviteUrl;
  bool _emailSent = false;
  String _createdUsername = '';
  String _createdEmail = '';

  bool get _validEmail =>
      RegExp(r'^[^\s@]+@[^\s@]+\.[^\s@]+$').hasMatch(_email.text.trim());
  bool get _valid => _validEmail && _username.text.trim().isNotEmpty;

  @override
  void dispose() {
    _email.dispose();
    _username.dispose();
    super.dispose();
  }

  void _reset() {
    _formKey.currentState?.reset();
    _email.clear();
    _username.clear();
    setState(() {
      _roles.clear();
      _error = null;
      _inviteUrl = null;
      _emailSent = false;
    });
  }

  Future<void> _submit() async {
    if (_submitting ||
        !_valid ||
        !ref.read(canManageUsersProvider) ||
        !ref.read(availableRolesProvider).hasValue ||
        !_formKey.currentState!.validate()) {
      return;
    }
    final username = _username.text.trim();
    final email = _email.text.trim();
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final response = await ref
          .read(apiDioProvider)
          .post<Map<String, dynamic>>(
            '/v1/users',
            data: {
              'email': email,
              'username': username,
              'roles': _roles.toList(),
            },
          );
      if (!mounted) return;
      setState(() {
        _createdUsername = username;
        _createdEmail = email;
        _inviteUrl = response.data!['invite_url'] as String;
        // Assente in risposte più vecchie della funzionalità email? Mai:
        // il campo è sempre presente lato backend, ma `as bool? ?? false`
        // costa nulla e non fa mai crashare la UI per un contratto API
        // che comunque non dovrebbe mai divergere.
        _emailSent = response.data!['email_sent'] as bool? ?? false;
      });
    } on DioException catch (error) {
      if (mounted) setState(() => _error = _userError(error).translationKey);
    } on ApiException catch (error) {
      if (mounted) setState(() => _error = error.translationKey);
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  Future<void> _copyInvite() async {
    try {
      await Clipboard.setData(ClipboardData(text: _inviteUrl!));
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('create_user.copied'.tr())));
    } on PlatformException {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('create_user.copy_failed'.tr())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final allowed = ref.watch(canManageUsersProvider);
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'home.title'.tr(),
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: _submitting ? null : () => context.go('/home'),
        ),
        title: Text('create_user.title'.tr()),
        actions: const [ThemeToggleButton(), SizedBox(width: 8)],
      ),
      body: !allowed
          ? Center(child: Text('create_user.forbidden'.tr()))
          : SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 900),
                  child: _inviteUrl != null
                      ? _success(context)
                      : _form(context),
                ),
              ),
            ),
    );
  }

  Widget _secondaryButton(
    BuildContext context, {
    required VoidCallback? onPressed,
    required String label,
  }) {
    final theme = Theme.of(context);
    return OutlinedButton(
      style: OutlinedButton.styleFrom(
        minimumSize: const Size(0, 56),
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        foregroundColor: theme.colorScheme.onSurface,
        backgroundColor: theme.colorScheme.surfaceContainerLow,
        side: BorderSide(color: theme.colorScheme.outline),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        textStyle: theme.textTheme.titleMedium?.copyWith(
          fontWeight: FontWeight.w700,
        ),
      ),
      onPressed: onPressed,
      child: Text(label, textAlign: TextAlign.center),
    );
  }

  Widget _panel(BuildContext context, Widget child) => Container(
    padding: const EdgeInsets.all(24),
    decoration: BoxDecoration(
      color: Theme.of(context).colorScheme.surfaceContainerHighest
          .withValues(alpha: 0.4),
      borderRadius: BorderRadius.circular(20),
    ),
    child: child,
  );

  Widget _form(BuildContext context) {
    final theme = Theme.of(context);
    final roles = ref.watch(availableRolesProvider);
    final fields = _panel(
      context,
      Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'create_user.access_title'.tr(),
            style: theme.textTheme.titleMedium,
          ),
          const SizedBox(height: 6),
          Text(
            'create_user.access_hint'.tr(),
            style: theme.textTheme.bodyMedium,
          ),
          const SizedBox(height: 24),
          TextFormField(
            controller: _email,
            enabled: !_submitting,
            keyboardType: TextInputType.emailAddress,
            autocorrect: false,
            decoration: InputDecoration(
              labelText: 'profile.email_label'.tr(),
              prefixIcon: const Icon(Icons.alternate_email_rounded),
            ),
            validator: (_) =>
                _validEmail ? null : 'create_user.email_invalid'.tr(),
          ),
          const SizedBox(height: 18),
          TextFormField(
            controller: _username,
            enabled: !_submitting,
            autocorrect: false,
            decoration: InputDecoration(
              labelText: 'profile.username_label'.tr(),
              prefixIcon: const Icon(Icons.person_outline_rounded),
            ),
            validator: (_) => _username.text.trim().isEmpty
                ? 'create_user.username_required'.tr()
                : null,
          ),
          const SizedBox(height: 28),
          Text(
            'create_user.roles_title'.tr(),
            style: theme.textTheme.titleMedium,
          ),
          const SizedBox(height: 6),
          Text('create_user.roles_hint'.tr()),
          const SizedBox(height: 16),
          roles.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (error, _) => Column(
              children: [
                ErrorBanner(
                  message:
                      (error is ApiException
                              ? error.translationKey
                              : 'errors.unknown')
                          .tr(),
                ),
                TextButton(
                  onPressed: () => ref.invalidate(availableRolesProvider),
                  child: Text('common.retry'.tr()),
                ),
              ],
            ),
            data: (items) => items.isEmpty
                ? Text('create_user.no_roles'.tr())
                : Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: items
                        .map(
                          (role) => FilterChip(
                            label: Text(role.displayName),
                            selected: _roles.contains(role.slug),
                            onSelected: _submitting
                                ? null
                                : (selected) => setState(() {
                                    if (selected) {
                                      _roles.add(role.slug);
                                    } else {
                                      _roles.remove(role.slug);
                                    }
                                  }),
                          ),
                        )
                        .toList(),
                  ),
          ),
          const SizedBox(height: 12),
          Text(
            'create_user.selected_roles'.tr(
              namedArgs: {'count': '${_roles.length}'},
            ),
            style: theme.textTheme.bodySmall,
          ),
        ],
      ),
    );
    return Form(
      key: _formKey,
      onChanged: () => setState(() {}),
      autovalidateMode: AutovalidateMode.onUserInteraction,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            'create_user.eyebrow'.tr(),
            style: theme.textTheme.labelLarge?.copyWith(
              color: theme.colorScheme.primary,
              letterSpacing: 1.5,
            ),
          ),
          const SizedBox(height: 10),
          Text(
            'create_user.headline'.tr(),
            style: theme.textTheme.headlineLarge?.copyWith(
              fontWeight: FontWeight.w800,
            ),
          ),
          const SizedBox(height: 12),
          Text(
            'create_user.subtitle'.tr(),
            style: theme.textTheme.bodyLarge?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 28),
          if (_error != null) ...[
            ErrorBanner(message: _error!.tr()),
            const SizedBox(height: 20),
          ],
          LayoutBuilder(
            builder: (context, constraints) => constraints.maxWidth >= 720
                ? Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Expanded(flex: 3, child: fields),
                      const SizedBox(width: 28),
                      Expanded(flex: 2, child: _inviteHelp(context)),
                    ],
                  )
                : Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      fields,
                      const SizedBox(height: 24),
                      _inviteHelp(context),
                    ],
                  ),
          ),
          const SizedBox(height: 24),
          LayoutBuilder(
            builder: (context, constraints) {
              final secondary = _secondaryButton(
                context,
                onPressed: _submitting ? null : _reset,
                label: 'create_user.reset'.tr(),
              );
              final primary = FilledButton(
                style: FilledButton.styleFrom(minimumSize: const Size(0, 56)),
                onPressed: !_submitting && _valid && roles.hasValue
                    ? _submit
                    : null,
                child: _submitting
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : Text('create_user.submit'.tr()),
              );
              if (constraints.maxWidth < 600) {
                return Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [secondary, const SizedBox(height: 12), primary],
                );
              }
              return Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  Flexible(child: secondary),
                  const SizedBox(width: 12),
                  Flexible(child: primary),
                ],
              );
            },
          ),
        ],
      ),
    ).animate().fadeIn(duration: 300.ms);
  }

  Widget _inviteHelp(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Container(
          padding: const EdgeInsets.all(22),
          decoration: BoxDecoration(
            gradient: LinearGradient(
              colors: [
                theme.colorScheme.primaryContainer,
                theme.colorScheme.surfaceContainerHighest,
              ],
            ),
            borderRadius: BorderRadius.circular(20),
          ),
          child: Icon(
            Icons.mark_email_unread_outlined,
            size: 58,
            color: theme.colorScheme.onPrimaryContainer,
          ),
        ),
        const SizedBox(height: 24),
        Text(
          'create_user.welcome_title'.tr(),
          style: theme.textTheme.titleMedium,
        ),
        for (var i = 1; i <= 3; i++) ...[
          const SizedBox(height: 22),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              CircleAvatar(
                radius: 14,
                backgroundColor: theme.colorScheme.primaryContainer,
                child: Text(
                  '$i',
                  style: TextStyle(color: theme.colorScheme.onPrimaryContainer),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'create_user.step_${i}_title'.tr(),
                      style: theme.textTheme.labelLarge,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      'create_user.step_${i}_body'.tr(),
                      style: theme.textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ],
          ),
        ],
        const SizedBox(height: 20),
        Text(
          'create_user.manual_email'.tr(),
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }

  Widget _success(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: _panel(
          context,
          Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Icon(
                _emailSent
                    ? Icons.mark_email_read_outlined
                    : Icons.check_circle_outline_rounded,
                size: 64,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(height: 20),
              Text(
                (_emailSent
                        ? 'create_user.email_sent_title'
                        : 'create_user.success_title')
                    .tr(),
                textAlign: TextAlign.center,
                style: theme.textTheme.headlineMedium,
              ),
              const SizedBox(height: 12),
              Text(
                _emailSent
                    ? 'create_user.email_sent_body'.tr(
                        namedArgs: {'email': _createdEmail},
                      )
                    : 'create_user.success_body'.tr(
                        namedArgs: {'username': _createdUsername},
                      ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 24),
              if (_emailSent)
                // L'email è già partita: il link resta disponibile solo
                // come ripiego silenzioso (spam, indirizzo sbagliato...),
                // non più in primo piano come quando l'invio non è
                // configurato — vedi docs/adr/0023.
                Center(
                  child: TextButton.icon(
                    onPressed: _copyInvite,
                    icon: const Icon(Icons.copy_rounded, size: 18),
                    label: Text('create_user.copy_fallback'.tr()),
                  ),
                )
              else ...[
                SelectableText(_inviteUrl!),
                const SizedBox(height: 20),
                FilledButton.icon(
                  onPressed: _copyInvite,
                  icon: const Icon(Icons.copy_rounded),
                  label: Text('create_user.copy'.tr()),
                ),
                const SizedBox(height: 16),
                Text('create_user.expiry'.tr(), textAlign: TextAlign.center),
              ],
              const SizedBox(height: 20),
              _secondaryButton(
                context,
                onPressed: _reset,
                label: 'create_user.another'.tr(),
              ),
            ],
          ),
        ),
      ),
    ).animate().fadeIn(duration: 300.ms);
  }
}
