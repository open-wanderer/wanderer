import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';

class WandererRadioOption<T> {
  final String label;
  final T value;
  final FaIconData? icon;

  WandererRadioOption({required this.label, required this.value, this.icon});
}

class WandererRadioGroup<T> extends FormBuilderField<T> {
  final String? label;
  final List<WandererRadioOption<T>> options;
  final bool disabled;
  final ValueChanged<T?>? onValueChanged;
  final Axis orientation;

  WandererRadioGroup({
    super.key,
    required super.name,
    required this.options,
    super.validator,
    super.initialValue,
    this.label,
    this.disabled = false,
    this.onValueChanged,
    this.orientation = Axis.vertical,
  }) : super(
         builder: (FormFieldState<T?> field) {
           final theme = Theme.of(field.context);
           final isError = field.hasError;

           void handleUpdate(T? value) {
             if (disabled) return;
             field.didChange(value);
             onValueChanged?.call(value);
           }

           Widget buildRadioItem(WandererRadioOption<T> option) {
             final isSelected = field.value == option.value;

             return InkWell(
               onTap: () => handleUpdate(option.value),
               borderRadius: BorderRadius.circular(6),
               child: Padding(
                 padding: const EdgeInsets.symmetric(
                   vertical: 8,
                   horizontal: 4,
                 ),
                 child: Row(
                   mainAxisSize: MainAxisSize.min,
                   children: [
                     Container(
                       width: 18,
                       height: 18,
                       decoration: BoxDecoration(
                         shape: BoxShape.circle,
                         border: Border.all(
                           color: theme.colorScheme.onSurface,
                           width: isSelected ? 5 : 1,
                         ),
                         color: theme.colorScheme.surface,
                       ),
                     ),
                     const SizedBox(width: 10),
                     if (option.icon != null) ...[
                       FaIcon(
                         option.icon,
                         size: 14,
                         color: isSelected
                             ? theme.colorScheme.primary
                             : theme.colorScheme.onSurface.withValues(
                                 alpha: 0.7,
                               ),
                       ),
                       const SizedBox(width: 8),
                     ],
                     Text(
                       option.label,
                       style: theme.textTheme.bodyMedium?.copyWith(
                         color: disabled
                             ? Colors.grey
                             : theme.colorScheme.onSurface,
                         fontWeight: isSelected
                             ? FontWeight.w500
                             : FontWeight.normal,
                       ),
                     ),
                   ],
                 ),
               ),
             );
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

               orientation == Axis.vertical
                   ? Column(
                       crossAxisAlignment: CrossAxisAlignment.start,
                       children: options.map(buildRadioItem).toList(),
                     )
                   : Wrap(
                       spacing: 16,
                       children: options.map(buildRadioItem).toList(),
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
