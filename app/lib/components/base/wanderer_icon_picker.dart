import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/theme/icons.dart';

/// A searchable icon picker over the FontAwesome catalogue.
///
/// Integrates with `flutter_form_builder`, storing the chosen [FaIconData] so it
/// is available via `formKey.currentState!.value['<name>'] as FaIconData?`,
/// mirroring [WandererTextField] / [WandererSelect]. Tapping the field opens a
/// modal sheet with a search box and a scrollable grid of icons drawn from
/// [fontAwesomeIconsMap].
class WandererIconPicker extends FormBuilderField<FaIconData> {
  final String? label;

  WandererIconPicker({
    super.key,
    required super.name,
    super.validator,
    super.initialValue,
    super.onChanged,
    this.label,
  }) : super(
         builder: (FormFieldState<FaIconData?> field) {
           final theme = Theme.of(field.context);
           final isError = field.hasError;
           final icon = field.value;
           final iconName = icon != null
               ? fontAwesomeIconsMapReversed[icon]
               : null;

           Future<void> openPicker() async {
             final selected = await showModalBottomSheet<FaIconData>(
               context: field.context,
               isScrollControlled: true,
               showDragHandle: true,
               builder: (_) => _IconPickerSheet(selected: icon),
             );
             if (selected != null) field.didChange(selected);
           }

           return Column(
             crossAxisAlignment: CrossAxisAlignment.start,
             children: [
               if (label != null && label.isNotEmpty)
                 Padding(
                   padding: const EdgeInsets.only(bottom: 4),
                   child: Text(
                     label,
                     style: theme.textTheme.bodySmall?.copyWith(
                       fontWeight: FontWeight.w600,
                       color: theme.colorScheme.onSurface,
                     ),
                   ),
                 ),
               InkWell(
                 onTap: openPicker,
                 borderRadius: BorderRadius.circular(6),
                 child: InputDecorator(
                   decoration: InputDecoration(
                     filled: true,
                     fillColor: isError
                         ? const Color(0xFFFEF2F2)
                         : theme.inputDecorationTheme.fillColor,
                     contentPadding: const EdgeInsets.all(12),
                     enabledBorder: OutlineInputBorder(
                       borderRadius: BorderRadius.circular(6),
                       borderSide: BorderSide(
                         color: isError
                             ? Colors.red.shade400
                             : theme.colorScheme.outline,
                       ),
                     ),
                     border: OutlineInputBorder(
                       borderRadius: BorderRadius.circular(6),
                       borderSide: BorderSide(
                         color: isError
                             ? Colors.red.shade400
                             : theme.colorScheme.outline,
                       ),
                     ),
                   ),
                   child: Row(
                     children: [
                       SizedBox(
                         width: 24,
                         child: Center(
                           child: FaIcon(
                             icon ?? FontAwesomeIcons.circle,
                             size: 16,
                             color: theme.colorScheme.onSurface,
                           ),
                         ),
                       ),
                       const SizedBox(width: 8),
                       Expanded(
                         child: Text(
                           iconName ?? '',
                           overflow: TextOverflow.ellipsis,
                           style: TextStyle(color: theme.colorScheme.onSurface),
                         ),
                       ),
                       FaIcon(
                         FontAwesomeIcons.chevronDown,
                         size: 14,
                         color: theme.colorScheme.onSurface.withValues(
                           alpha: 0.7,
                         ),
                       ),
                     ],
                   ),
                 ),
               ),
               if (isError)
                 Padding(
                   padding: const EdgeInsets.only(top: 4),
                   child: Text(
                     field.errorText!,
                     style: TextStyle(color: Colors.red.shade400, fontSize: 12),
                   ),
                 ),
             ],
           );
         },
       );
}

/// The modal sheet body: a search box plus a grid of icons. Kept stateful so the
/// query filters locally without rebuilding the host form.
class _IconPickerSheet extends StatefulWidget {
  final FaIconData? selected;

  const _IconPickerSheet({required this.selected});

  @override
  State<_IconPickerSheet> createState() => _IconPickerSheetState();
}

class _IconPickerSheetState extends State<_IconPickerSheet> {
  static final List<MapEntry<String, FaIconData>> _allIcons =
      fontAwesomeIconsMap.entries.toList();

  String _query = '';

  List<MapEntry<String, FaIconData>> get _filtered {
    final q = _query.trim().toLowerCase();
    if (q.isEmpty) return _allIcons;
    // Match on the icon name with dashes treated as spaces, mirroring the web
    // combobox (`icon_util.ts`).
    return _allIcons
        .where((e) => e.key.replaceAll('-', ' ').contains(q))
        .toList();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final filtered = _filtered;

    return Padding(
      padding: EdgeInsets.only(
        left: 16,
        right: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.7,
        child: Column(
          children: [
            TextField(
              autofocus: true,
              onChanged: (v) => setState(() => _query = v),
              decoration: InputDecoration(
                hintText: l10n.search,
                prefixIcon: const Padding(
                  padding: EdgeInsets.all(12),
                  child: FaIcon(FontAwesomeIcons.magnifyingGlass, size: 16),
                ),
                filled: true,
                fillColor: theme.inputDecorationTheme.fillColor,
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(6),
                  borderSide: BorderSide(color: theme.colorScheme.outline),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(6),
                  borderSide: BorderSide(
                    color: theme.colorScheme.primary,
                    width: 1.5,
                  ),
                ),
              ),
            ),
            const SizedBox(height: 12),
            Expanded(
              child: GridView.builder(
                gridDelegate: const SliverGridDelegateWithMaxCrossAxisExtent(
                  maxCrossAxisExtent: 72,
                  mainAxisSpacing: 8,
                  crossAxisSpacing: 8,
                ),
                itemCount: filtered.length,
                itemBuilder: (context, index) {
                  final entry = filtered[index];
                  final isSelected = entry.value == widget.selected;
                  return InkWell(
                    onTap: () => Navigator.of(context).pop(entry.value),
                    borderRadius: BorderRadius.circular(6),
                    child: Container(
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(6),
                        border: Border.all(
                          color: isSelected
                              ? theme.colorScheme.primary
                              : theme.colorScheme.outline,
                          width: isSelected ? 1.5 : 1,
                        ),
                        color: isSelected
                            ? theme.colorScheme.primary.withValues(alpha: 0.1)
                            : null,
                      ),
                      child: Center(
                        child: FaIcon(
                          entry.value,
                          size: 28,
                          color: theme.colorScheme.onSurface,
                        ),
                      ),
                    ),
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
