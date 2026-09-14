import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api_exception.dart';
import '../../../core/widgets/error_banner.dart';
import '../../manage_users/manage_users_screen.dart' show usersProvider;

/// Bottom sheet di ricerca volontario — stesso identico pattern di ricerca
/// già usato in `manage_users_screen.dart` (`usersProvider`, filtro locale
/// su username/email), qui riusato per scegliere a chi assegnare
/// direttamente una figura (`OccurrenceCard.onAssign`). Ritorna l'id
/// scelto, o null se l'utente chiude lo sheet senza scegliere.
Future<String?> showVolunteerPicker(BuildContext context) {
  return showModalBottomSheet<String>(
    context: context,
    isScrollControlled: true,
    builder: (context) => const _VolunteerPickerSheet(),
  );
}

class _VolunteerPickerSheet extends ConsumerStatefulWidget {
  const _VolunteerPickerSheet();

  @override
  ConsumerState<_VolunteerPickerSheet> createState() =>
      _VolunteerPickerSheetState();
}

class _VolunteerPickerSheetState extends ConsumerState<_VolunteerPickerSheet> {
  String _search = '';

  @override
  Widget build(BuildContext context) {
    final users = ref.watch(usersProvider);
    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.4,
      maxChildSize: 0.9,
      expand: false,
      builder: (context, scrollController) => Padding(
        padding: EdgeInsets.only(
          left: 20,
          right: 20,
          top: 16,
          bottom: MediaQuery.of(context).viewInsets.bottom + 16,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'shifts.assign_title'.tr(),
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 12),
            TextField(
              autofocus: true,
              decoration: InputDecoration(
                hintText: 'manage_users.search_hint'.tr(),
                prefixIcon: const Icon(Icons.search_rounded),
              ),
              onChanged: (value) =>
                  setState(() => _search = value.trim().toLowerCase()),
            ),
            const SizedBox(height: 8),
            Expanded(
              child: users.when(
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (error, _) => Center(
                  child: ErrorBanner(
                    message:
                        (error is ApiException
                                ? error.translationKey
                                : 'errors.unknown')
                            .tr(),
                  ),
                ),
                data: (items) {
                  final filtered = _search.isEmpty
                      ? items
                      : items
                            .where(
                              (u) =>
                                  u.username.toLowerCase().contains(_search) ||
                                  u.email.toLowerCase().contains(_search),
                            )
                            .toList();
                  if (filtered.isEmpty) {
                    return Center(child: Text('manage_users.empty'.tr()));
                  }
                  return ListView.builder(
                    controller: scrollController,
                    itemCount: filtered.length,
                    itemBuilder: (context, index) {
                      final user = filtered[index];
                      return ListTile(
                        title: Text(user.username),
                        subtitle: Text(user.email),
                        onTap: () => Navigator.of(context).pop(user.id),
                      );
                    },
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
  }
}
