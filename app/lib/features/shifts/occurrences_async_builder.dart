import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_exception.dart';
import '../../core/widgets/error_banner.dart';
import 'shift_models.dart';
import 'shifts_filter.dart';
import 'shifts_providers.dart';

/// Wrapper condiviso dalle 4 viste: guarda `occurrencesProvider(range)`,
/// gestisce loading/errore con lo stesso pattern di `manage_users_screen.dart`
/// (`ErrorBanner` + "Riprova"), applica [filter] alla lista scaricata e
/// passa il risultato a [builder] — evita di ripetere lo stesso
/// `AsyncValue.when` in ogni vista.
class OccurrencesAsyncBuilder extends ConsumerWidget {
  const OccurrencesAsyncBuilder({
    super.key,
    required this.range,
    required this.filter,
    required this.builder,
  });

  final OccurrenceRange range;
  final ShiftsFilter filter;
  final Widget Function(BuildContext context, List<ShiftOccurrence> items)
  builder;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(occurrencesProvider(range));
    return async.when(
      loading: () => const Padding(
        padding: EdgeInsets.all(24),
        child: Center(child: CircularProgressIndicator()),
      ),
      error: (error, _) => Padding(
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
              onPressed: () => ref.invalidate(occurrencesProvider(range)),
              child: Text('common.retry'.tr()),
            ),
          ],
        ),
      ),
      data: (items) => builder(context, filterOccurrences(items, filter)),
    );
  }
}
