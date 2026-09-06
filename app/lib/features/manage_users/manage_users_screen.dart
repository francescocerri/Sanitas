import 'package:dio/dio.dart';
import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_client.dart';
import '../../core/api_exception.dart';
import '../../core/widgets/error_banner.dart';
import '../../core/widgets/theme_toggle_button.dart';
import '../create_user/create_user_screen.dart'
    show AvailableRole, availableRolesProvider, canManageUsersProvider;

/// Un utente così come lo restituisce `GET /v1/users` — solo i campi che
/// servono a questa schermata (niente `permissions`, non ci serve qui).
class ManagedUser {
  const ManagedUser({
    required this.id,
    required this.username,
    required this.email,
    required this.roles,
  });

  final String id;
  final String username;
  final String email;
  final List<String> roles;
}

ApiException _manageUsersError(DioException error) =>
    ApiException.fromDioException(
      error,
      statusToKey: const {
        400: 'errors.invalid_payload',
        403: 'create_user.forbidden',
        404: 'manage_users.user_not_found',
      },
    );

/// Elenco di tutti gli utenti, con i loro ruoli già popolati dal backend
/// (vedi `internal/user.Repository.ListUsers` in `services/registry`).
/// Stesso schema di `availableRolesProvider`: richiede `users:manage`,
/// si invalida da sé quando cambia l'utente loggato.
final usersProvider = FutureProvider.autoDispose<List<ManagedUser>>((
  ref,
) async {
  if (!ref.watch(canManageUsersProvider)) {
    throw const ApiException('create_user.forbidden');
  }
  try {
    final response = await ref
        .watch(apiDioProvider)
        .get<List<dynamic>>('/v1/users');
    return response.data!
        .map(
          (entry) => ManagedUser(
            id: entry['id'] as String,
            username: entry['username'] as String,
            email: entry['email'] as String,
            roles: (entry['roles'] as List<dynamic>).cast<String>(),
          ),
        )
        .toList();
  } on DioException catch (error) {
    throw _manageUsersError(error);
  }
});

/// Lista utenti + modifica ruoli, riservata a chi ha `users:manage`. Niente
/// schermata/rotta separata per la modifica: si tocca una riga per
/// espanderla sul posto (mostra i chip dei ruoli disponibili) invece di
/// aprire un secondo schermo — più semplice da mantenere, un solo file.
class ManageUsersScreen extends ConsumerStatefulWidget {
  const ManageUsersScreen({super.key});

  @override
  ConsumerState<ManageUsersScreen> createState() => _ManageUsersScreenState();
}

class _ManageUsersScreenState extends ConsumerState<ManageUsersScreen> {
  final _searchController = TextEditingController();
  String _search = '';

  // Solo una riga alla volta può essere espansa/in modifica: espandendone
  // un'altra si scartano le eventuali modifiche non salvate della
  // precedente — stesso compromesso di semplicità già scelto altrove in
  // questa sessione invece di un'inutile conferma "modifiche non salvate"
  // per un caso così raro.
  //
  // Due stati distinti apposta: toccare la riga mostra solo le info in
  // sola lettura (email, ruoli attuali come badge statici); solo la
  // matitina entra in modifica (chip selezionabili + Salva) — così non
  // capita di modificare qualcosa per sbaglio solo aprendo una riga per
  // curiosità.
  String? _expandedUserId;
  String? _editingUserId;
  Set<String> _pendingRoles = {};
  bool _saving = false;
  String? _errorTranslationKey;

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  void _toggleExpanded(ManagedUser user) {
    setState(() {
      if (_expandedUserId == user.id) {
        _expandedUserId = null;
        _editingUserId = null;
      } else {
        _expandedUserId = user.id;
        _editingUserId = null;
        _errorTranslationKey = null;
      }
    });
  }

  void _toggleEditing(ManagedUser user) {
    setState(() {
      if (_editingUserId == user.id) {
        // Annulla la modifica: torna alla sola visualizzazione, scartando
        // le selezioni non salvate.
        _editingUserId = null;
      } else {
        _expandedUserId = user.id;
        _editingUserId = user.id;
        _pendingRoles = user.roles.toSet();
        _errorTranslationKey = null;
      }
    });
  }

  Future<void> _save(String userId) async {
    setState(() {
      _saving = true;
      _errorTranslationKey = null;
    });
    try {
      await ref
          .read(apiDioProvider)
          .patch<void>(
            '/v1/users/$userId/roles',
            data: {'roles': _pendingRoles.toList()},
          );
      ref.invalidate(usersProvider);
      if (!mounted) return;
      setState(() {
        _expandedUserId = null;
        _editingUserId = null;
      });
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('manage_users.saved'.tr())));
    } on DioException catch (error) {
      if (mounted) {
        setState(
          () => _errorTranslationKey = _manageUsersError(error).translationKey,
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final allowed = ref.watch(canManageUsersProvider);
    final users = ref.watch(usersProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'home.title'.tr(),
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: () => context.go('/home'),
        ),
        title: Text('manage_users.title'.tr()),
        actions: const [ThemeToggleButton(), SizedBox(width: 8)],
      ),
      body: !allowed
          ? Center(child: Text('create_user.forbidden'.tr()))
          : SafeArea(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 720),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      Padding(
                        padding: const EdgeInsets.fromLTRB(20, 16, 20, 8),
                        child: TextField(
                          controller: _searchController,
                          decoration: InputDecoration(
                            hintText: 'manage_users.search_hint'.tr(),
                            prefixIcon: const Icon(Icons.search_rounded),
                          ),
                          onChanged: (value) => setState(
                            () => _search = value.trim().toLowerCase(),
                          ),
                        ),
                      ),
                      Expanded(
                        child: users.when(
                          loading: () =>
                              const Center(child: CircularProgressIndicator()),
                          error: (error, _) => Center(
                            child: Padding(
                              padding: const EdgeInsets.all(24),
                              child: Column(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  ErrorBanner(
                                    message:
                                        (error is ApiException
                                                ? error.translationKey
                                                : 'errors.unknown')
                                            .tr(),
                                  ),
                                  const SizedBox(height: 12),
                                  TextButton(
                                    onPressed: () =>
                                        ref.invalidate(usersProvider),
                                    child: Text('common.retry'.tr()),
                                  ),
                                ],
                              ),
                            ),
                          ),
                          data: (items) {
                            final filtered = _search.isEmpty
                                ? items
                                : items
                                      .where(
                                        (u) =>
                                            u.username.toLowerCase().contains(
                                              _search,
                                            ) ||
                                            u.email.toLowerCase().contains(
                                              _search,
                                            ),
                                      )
                                      .toList();
                            if (filtered.isEmpty) {
                              return Center(
                                child: Text('manage_users.empty'.tr()),
                              );
                            }
                            // Un utente con più ruoli compare in più
                            // sezioni: qui la vista è organizzata "per
                            // ruolo, chi ce l'ha", non una partizione degli
                            // utenti — un utente può comparire zero, una o
                            // più volte a seconda di quanti ruoli ha.
                            return ref
                                .watch(availableRolesProvider)
                                .when(
                                  loading: () => const Center(
                                    child: CircularProgressIndicator(),
                                  ),
                                  error: (error, _) => Center(
                                    child: ErrorBanner(
                                      message:
                                          (error is ApiException
                                                  ? error.translationKey
                                                  : 'errors.unknown')
                                              .tr(),
                                    ),
                                  ),
                                  data: (availableRoles) => _buildSections(
                                    context,
                                    availableRoles,
                                    filtered,
                                  ),
                                );
                          },
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
    );
  }

  /// Costruisce una sezione per ogni ruolo del catalogo che ha almeno un
  /// utente assegnato (niente sezioni vuote), più una sezione finale per
  /// chi non ha ancora nessun ruolo — solo se non vuota.
  Widget _buildSections(
    BuildContext context,
    List<AvailableRole> availableRoles,
    List<ManagedUser> users,
  ) {
    final theme = Theme.of(context);
    final sections = <(String title, List<ManagedUser> members)>[];
    for (final role in availableRoles) {
      final members = users.where((u) => u.roles.contains(role.slug)).toList();
      if (members.isNotEmpty) {
        sections.add((role.displayName, members));
      }
    }
    final withoutRole = users.where((u) => u.roles.isEmpty).toList();
    if (withoutRole.isNotEmpty) {
      sections.add(('manage_users.no_role_section'.tr(), withoutRole));
    }

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        for (final section in sections) ...[
          Padding(
            padding: const EdgeInsets.fromLTRB(4, 12, 4, 8),
            child: Text(
              'manage_users.section_header'.tr(
                namedArgs: {
                  'role': section.$1.toUpperCase(),
                  'count': '${section.$2.length}',
                },
              ),
              style: theme.textTheme.labelLarge?.copyWith(
                color: theme.colorScheme.primary,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.4,
              ),
            ),
          ),
          for (final user in section.$2) ...[
            _UserRow(
              user: user,
              expanded: _expandedUserId == user.id,
              editing: _editingUserId == user.id,
              pendingRoles: _pendingRoles,
              saving: _saving && _editingUserId == user.id,
              errorTranslationKey: _editingUserId == user.id
                  ? _errorTranslationKey
                  : null,
              onTap: () => _toggleExpanded(user),
              onEditToggle: () => _toggleEditing(user),
              onRoleToggled: (slug, selected) => setState(() {
                if (selected) {
                  _pendingRoles.add(slug);
                } else {
                  _pendingRoles.remove(slug);
                }
              }),
              onSave: () => _save(user.id),
            ),
            const SizedBox(height: 8),
          ],
        ],
      ],
    );
  }
}

class _UserRow extends ConsumerWidget {
  const _UserRow({
    required this.user,
    required this.expanded,
    required this.editing,
    required this.pendingRoles,
    required this.saving,
    required this.errorTranslationKey,
    required this.onTap,
    required this.onEditToggle,
    required this.onRoleToggled,
    required this.onSave,
  });

  final ManagedUser user;
  final bool expanded;
  final bool editing;
  final Set<String> pendingRoles;
  final bool saving;
  final String? errorTranslationKey;
  final VoidCallback onTap;
  final VoidCallback onEditToggle;
  final void Function(String slug, bool selected) onRoleToggled;
  final VoidCallback onSave;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final roles = ref.watch(availableRolesProvider);

    return Material(
      color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          user.username,
                          style: theme.textTheme.titleSmall?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        if (expanded) ...[
                          const SizedBox(height: 2),
                          Text(
                            user.email,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ] else if (user.roles.isEmpty)
                          Text(
                            'manage_users.no_roles'.tr(),
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: 'manage_users.edit_roles'.tr(),
                    icon: Icon(
                      editing ? Icons.edit_rounded : Icons.edit_outlined,
                      color: editing
                          ? theme.colorScheme.primary
                          : theme.colorScheme.onSurfaceVariant,
                    ),
                    onPressed: onEditToggle,
                  ),
                  Icon(
                    expanded
                        ? Icons.keyboard_arrow_up_rounded
                        : Icons.keyboard_arrow_down_rounded,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ],
              ),
              AnimatedSize(
                duration: const Duration(milliseconds: 180),
                child: !expanded
                    ? const SizedBox(width: double.infinity)
                    : Padding(
                        padding: const EdgeInsets.only(top: 12),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            if (errorTranslationKey != null) ...[
                              ErrorBanner(message: errorTranslationKey!.tr()),
                              const SizedBox(height: 12),
                            ],
                            if (!editing)
                              // Sola visualizzazione: badge statici, non
                              // toccabili — per modificarli serve la
                              // matitina, non basta aprire la riga.
                              roles.when(
                                loading: () => const Center(
                                  child: CircularProgressIndicator(),
                                ),
                                error: (_, _) => Text('errors.unknown'.tr()),
                                data: (items) {
                                  final byslug = {
                                    for (final role in items)
                                      role.slug: role.displayName,
                                  };
                                  if (user.roles.isEmpty) {
                                    return Text(
                                      'manage_users.no_roles'.tr(),
                                      style: theme.textTheme.bodySmall
                                          ?.copyWith(
                                            color: theme
                                                .colorScheme
                                                .onSurfaceVariant,
                                          ),
                                    );
                                  }
                                  return Wrap(
                                    spacing: 8,
                                    runSpacing: 8,
                                    children: user.roles
                                        .map(
                                          (slug) => Chip(
                                            label: Text(byslug[slug] ?? slug),
                                          ),
                                        )
                                        .toList(),
                                  );
                                },
                              )
                            else ...[
                              roles.when(
                                loading: () => const Center(
                                  child: CircularProgressIndicator(),
                                ),
                                error: (_, _) => Text('errors.unknown'.tr()),
                                data: (items) => Wrap(
                                  spacing: 8,
                                  runSpacing: 8,
                                  children: items
                                      .map(
                                        (role) => FilterChip(
                                          label: Text(role.displayName),
                                          selected: pendingRoles.contains(
                                            role.slug,
                                          ),
                                          onSelected: saving
                                              ? null
                                              : (selected) => onRoleToggled(
                                                  role.slug,
                                                  selected,
                                                ),
                                        ),
                                      )
                                      .toList(),
                                ),
                              ),
                              const SizedBox(height: 12),
                              Align(
                                alignment: Alignment.centerRight,
                                child: FilledButton(
                                  onPressed: saving ? null : onSave,
                                  child: saving
                                      ? const SizedBox(
                                          width: 18,
                                          height: 18,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                            color: Colors.white,
                                          ),
                                        )
                                      : Text('manage_users.save'.tr()),
                                ),
                              ),
                            ],
                          ],
                        ),
                      ),
              ),
            ],
          ),
        ),
      ),
    ).animate().fadeIn(duration: 200.ms);
  }
}
