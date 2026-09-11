import 'package:easy_localization/easy_localization.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api_exception.dart';
import '../../../core/widgets/error_banner.dart';
import '../shifts_providers.dart';

/// `HH:MM`, stesso pattern imposto lato backend (`timeOfDayPattern` in
/// `internal/httpapi/templates.go`) — validato anche qui per dare un
/// errore immediato invece di aspettare la risposta 400.
final _timeOfDayPattern = RegExp(r'^([01]\d|2[0-3]):[0-5]\d$');

/// Tab "Turni-template" del gestore (`shifts:configure`): elenco dei
/// turni-template (righe inattive dimmerate), tocco per aprire il form di
/// modifica sul posto, "+" per crearne uno nuovo — stesso genere di
/// espansione-riga già visto in `manage_users_screen.dart`, qui più
/// semplice (niente doppio stato sola-lettura/modifica: un template ha
/// pochi campi, si apre già in modifica).
class ShiftTemplatesTab extends ConsumerStatefulWidget {
  const ShiftTemplatesTab({super.key});

  @override
  ConsumerState<ShiftTemplatesTab> createState() => _ShiftTemplatesTabState();
}

class _ShiftTemplatesTabState extends ConsumerState<ShiftTemplatesTab> {
  String? _expandedId;

  @override
  Widget build(BuildContext context) {
    final templates = ref.watch(templatesProvider);
    final canConfigure = ref.watch(canConfigureShiftTemplatesProvider);

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 720),
        child: templates.when(
          loading: () => const Center(child: CircularProgressIndicator()),
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
                    onPressed: () => ref.invalidate(templatesProvider),
                    child: Text('common.retry'.tr()),
                  ),
                ],
              ),
            ),
          ),
          data: (items) => ListView(
            padding: EdgeInsets.fromLTRB(16, 16, 16, canConfigure ? 96 : 16),
            children: [
              for (final tpl in items) ...[
                _TemplateRow(
                  template: tpl,
                  editable: canConfigure,
                  expanded: _expandedId == tpl.id,
                  onTap: () => setState(
                    () => _expandedId = _expandedId == tpl.id ? null : tpl.id,
                  ),
                  onSaved: () => setState(() => _expandedId = null),
                ),
                const SizedBox(height: 8),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

class _TemplateRow extends ConsumerStatefulWidget {
  const _TemplateRow({
    required this.template,
    required this.editable,
    required this.expanded,
    required this.onTap,
    required this.onSaved,
  });

  final ShiftTemplateSummary template;
  final bool editable;
  final bool expanded;
  final VoidCallback onTap;
  final VoidCallback onSaved;

  @override
  ConsumerState<_TemplateRow> createState() => _TemplateRowState();
}

class _TemplateRowState extends ConsumerState<_TemplateRow> {
  final _formKey = GlobalKey<FormState>();
  late int _weekday;
  // Non `late final`: `_resetFromTemplate` viene richiamato ogni volta che
  // la riga si richiude dopo una modifica (vedi `didUpdateWidget` sotto),
  // e un `final` non si può riassegnare una seconda volta — creati una sola
  // volta qui, `_resetFromTemplate` si limita a riscriverne il `.text`.
  final _startTime = TextEditingController();
  final _endTime = TextEditingController();
  final _label = TextEditingController();
  late bool _active;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _resetFromTemplate();
  }

  @override
  void didUpdateWidget(covariant _TemplateRow old) {
    super.didUpdateWidget(old);
    if (!widget.expanded && old.expanded) _resetFromTemplate();
  }

  void _resetFromTemplate() {
    _weekday = widget.template.weekday;
    _startTime.text = widget.template.startTime;
    _endTime.text = widget.template.endTime;
    _label.text = widget.template.label;
    _active = widget.template.active;
  }

  @override
  void dispose() {
    _startTime.dispose();
    _endTime.dispose();
    _label.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await updateTemplate(
        ref,
        widget.template.id,
        weekday: _weekday,
        startTime: _startTime.text.trim(),
        endTime: _endTime.text.trim(),
        label: _label.text.trim(),
        active: _active,
      );
      ref.invalidate(templatesProvider);
      ref.invalidate(occurrencesProvider);
      if (!mounted) return;
      widget.onSaved();
    } on ApiException catch (error) {
      if (mounted) setState(() => _error = error.translationKey);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final tpl = widget.template;

    return Material(
      color: theme.colorScheme.surfaceContainerHighest.withValues(
        alpha: tpl.active ? 0.4 : 0.2,
      ),
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: widget.editable ? widget.onTap : null,
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
                          tpl.label,
                          style: theme.textTheme.titleSmall?.copyWith(
                            fontWeight: FontWeight.w700,
                            color: tpl.active
                                ? null
                                : theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          '${'shifts.weekday_${tpl.weekday}'.tr()} · '
                          '${tpl.startTime}–${tpl.endTime}',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),
                  if (!tpl.active)
                    Padding(
                      padding: const EdgeInsets.only(right: 8),
                      child: Text(
                        'shifts.template_inactive'.tr(),
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ),
                  if (widget.editable)
                    Icon(
                      widget.expanded
                          ? Icons.keyboard_arrow_up_rounded
                          : Icons.keyboard_arrow_down_rounded,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                ],
              ),
              AnimatedSize(
                duration: const Duration(milliseconds: 180),
                child: !widget.expanded
                    ? const SizedBox(width: double.infinity)
                    : Padding(
                        padding: const EdgeInsets.only(top: 12),
                        child: Form(
                          key: _formKey,
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              if (_error != null) ...[
                                ErrorBanner(message: _error!.tr()),
                                const SizedBox(height: 12),
                              ],
                              DropdownButtonFormField<int>(
                                initialValue: _weekday,
                                decoration: InputDecoration(
                                  labelText: 'shifts.template_weekday'.tr(),
                                ),
                                items: [
                                  for (var w = 0; w <= 6; w++)
                                    DropdownMenuItem(
                                      value: w,
                                      child: Text('shifts.weekday_$w'.tr()),
                                    ),
                                ],
                                onChanged: _saving
                                    ? null
                                    : (value) =>
                                          setState(() => _weekday = value!),
                              ),
                              const SizedBox(height: 12),
                              Row(
                                children: [
                                  Expanded(
                                    child: TextFormField(
                                      controller: _startTime,
                                      enabled: !_saving,
                                      decoration: InputDecoration(
                                        labelText: 'shifts.template_start_time'
                                            .tr(),
                                        hintText: 'HH:MM',
                                      ),
                                      validator: (value) =>
                                          _timeOfDayPattern.hasMatch(
                                            value?.trim() ?? '',
                                          )
                                          ? null
                                          : 'shifts.template_time_invalid'.tr(),
                                    ),
                                  ),
                                  const SizedBox(width: 12),
                                  Expanded(
                                    child: TextFormField(
                                      controller: _endTime,
                                      enabled: !_saving,
                                      decoration: InputDecoration(
                                        labelText: 'shifts.template_end_time'
                                            .tr(),
                                        hintText: 'HH:MM',
                                      ),
                                      validator: (value) =>
                                          _timeOfDayPattern.hasMatch(
                                            value?.trim() ?? '',
                                          )
                                          ? null
                                          : 'shifts.template_time_invalid'.tr(),
                                    ),
                                  ),
                                ],
                              ),
                              const SizedBox(height: 12),
                              TextFormField(
                                controller: _label,
                                enabled: !_saving,
                                decoration: InputDecoration(
                                  labelText: 'shifts.template_label'.tr(),
                                ),
                                validator: (value) =>
                                    (value?.trim().isEmpty ?? true)
                                    ? 'shifts.template_label_required'.tr()
                                    : null,
                              ),
                              SwitchListTile(
                                contentPadding: EdgeInsets.zero,
                                title: Text('shifts.template_active'.tr()),
                                value: _active,
                                onChanged: _saving
                                    ? null
                                    : (value) =>
                                          setState(() => _active = value),
                              ),
                              const SizedBox(height: 8),
                              Align(
                                alignment: Alignment.centerRight,
                                child: FilledButton(
                                  onPressed: _saving ? null : _save,
                                  child: _saving
                                      ? const SizedBox(
                                          width: 18,
                                          height: 18,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                            color: Colors.white,
                                          ),
                                        )
                                      : Text('shifts.template_save'.tr()),
                                ),
                              ),
                            ],
                          ),
                        ),
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// FAB "+" per creare un nuovo turno-template — dialog separato (non
/// un'altra riga espandibile: non c'è ancora un `ShiftTemplateSummary` da
/// mostrare finché non è stato creato).
class NewTemplateFab extends ConsumerStatefulWidget {
  const NewTemplateFab({super.key});

  @override
  ConsumerState<NewTemplateFab> createState() => _NewTemplateFabState();
}

class _NewTemplateFabState extends ConsumerState<NewTemplateFab> {
  @override
  Widget build(BuildContext context) {
    return FloatingActionButton.extended(
      onPressed: () => showDialog<void>(
        context: context,
        builder: (context) => _NewTemplateDialog(ref: ref),
      ),
      icon: const Icon(Icons.add_rounded),
      label: Text('shifts.template_new'.tr()),
    );
  }
}

class _NewTemplateDialog extends StatefulWidget {
  const _NewTemplateDialog({required this.ref});

  final WidgetRef ref;

  @override
  State<_NewTemplateDialog> createState() => _NewTemplateDialogState();
}

class _NewTemplateDialogState extends State<_NewTemplateDialog> {
  final _formKey = GlobalKey<FormState>();
  int _weekday = DateTime.monday % 7;
  final _startTime = TextEditingController();
  final _endTime = TextEditingController();
  final _label = TextEditingController();
  bool _saving = false;
  String? _error;

  @override
  void dispose() {
    _startTime.dispose();
    _endTime.dispose();
    _label.dispose();
    super.dispose();
  }

  Future<void> _create() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await createTemplate(
        widget.ref,
        weekday: _weekday,
        startTime: _startTime.text.trim(),
        endTime: _endTime.text.trim(),
        label: _label.text.trim(),
      );
      widget.ref.invalidate(templatesProvider);
      widget.ref.invalidate(occurrencesProvider);
      if (!mounted) return;
      Navigator.of(context).pop();
    } on ApiException catch (error) {
      if (mounted) setState(() => _error = error.translationKey);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('shifts.template_new'.tr()),
      content: Form(
        key: _formKey,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              if (_error != null) ...[
                ErrorBanner(message: _error!.tr()),
                const SizedBox(height: 12),
              ],
              DropdownButtonFormField<int>(
                initialValue: _weekday,
                decoration: InputDecoration(
                  labelText: 'shifts.template_weekday'.tr(),
                ),
                items: [
                  for (var w = 0; w <= 6; w++)
                    DropdownMenuItem(
                      value: w,
                      child: Text('shifts.weekday_$w'.tr()),
                    ),
                ],
                onChanged: _saving
                    ? null
                    : (value) => setState(() => _weekday = value!),
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: TextFormField(
                      controller: _startTime,
                      enabled: !_saving,
                      decoration: InputDecoration(
                        labelText: 'shifts.template_start_time'.tr(),
                        hintText: 'HH:MM',
                      ),
                      validator: (value) =>
                          _timeOfDayPattern.hasMatch(value?.trim() ?? '')
                          ? null
                          : 'shifts.template_time_invalid'.tr(),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: TextFormField(
                      controller: _endTime,
                      enabled: !_saving,
                      decoration: InputDecoration(
                        labelText: 'shifts.template_end_time'.tr(),
                        hintText: 'HH:MM',
                      ),
                      validator: (value) =>
                          _timeOfDayPattern.hasMatch(value?.trim() ?? '')
                          ? null
                          : 'shifts.template_time_invalid'.tr(),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _label,
                enabled: !_saving,
                decoration: InputDecoration(
                  labelText: 'shifts.template_label'.tr(),
                ),
                validator: (value) => (value?.trim().isEmpty ?? true)
                    ? 'shifts.template_label_required'.tr()
                    : null,
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: _saving ? null : () => Navigator.of(context).pop(),
          child: Text('common.cancel'.tr()),
        ),
        FilledButton(
          onPressed: _saving ? null : _create,
          child: _saving
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text('shifts.template_save'.tr()),
        ),
      ],
    );
  }
}
