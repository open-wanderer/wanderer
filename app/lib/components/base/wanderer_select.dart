import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';

/// A single option in a [WandererSelect], pairing a [value] of type [T] with
/// the [label] shown for it, plus an optional leading [icon] widget.
class SelectItem<T> {
  final T value;
  final String label;
  final Widget? icon;

  const SelectItem({required this.value, required this.label, this.icon});
}

/// A simple single-choice select menu, generic over the option type [T].
///
/// Integrates with `flutter_form_builder`, so the selected value is available
/// via `formKey.currentState!.value['<name>'] as T?`, mirroring
/// [WandererTextField].
class WandererSelect<T> extends FormBuilderField<T> {
  final String? label;
  final String? placeholder;
  final FaIconData? icon;
  final bool disabled;
  final List<SelectItem<T>> items;

  WandererSelect({
    super.key,
    required super.name,
    required this.items,
    super.validator,
    super.initialValue,
    super.onChanged,
    this.label,
    this.placeholder,
    this.icon,
    this.disabled = false,
  }) : super(
         builder: (FormFieldState<T?> field) {
           final theme = Theme.of(field.context);
           final isError = field.hasError;

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

               Row(
                 crossAxisAlignment: CrossAxisAlignment.center,
                 children: [
                   if (icon != null)
                     Padding(
                       padding: const EdgeInsets.only(right: 8),
                       child: SizedBox(
                         width: 24,
                         child: Center(
                           child: FaIcon(
                             icon,
                             size: 16,
                             color: theme.colorScheme.onSurface.withValues(
                               alpha: 0.7,
                             ),
                           ),
                         ),
                       ),
                     ),

                   Expanded(
                     // DropdownButtonFormField is itself a FormField, so
                     // without a Form of its own it registers with the
                     // enclosing FormBuilder's Form and is reset alongside the
                     // WandererSelect wrapping it. Its reset() ends by calling
                     // `onChanged(value)` (material/dropdown.dart), which lands
                     // on the OUTER field's didChange and re-latches its
                     // `_dirty` flag -- unconditionally, value unchanged or
                     // not -- a moment after that field's own reset() cleared
                     // it. `FormBuilderState.isDirty` then stayed true for a
                     // form that had just been reset to its saved values, and
                     // leaving the trail screen asked the user to discard
                     // changes they had already saved. A private Form scope
                     // keeps this inner field out of the outer field set; the
                     // selection still flows in through `initialValue` below,
                     // which FormField.didUpdateWidget applies on rebuild.
                     child: Form(
                       child: DropdownButtonFormField<T>(
                         initialValue: field.value,
                         isExpanded: true,
                         onChanged: disabled
                             ? null
                             : (val) => field.didChange(val),
                         icon: Padding(
                           padding: const EdgeInsets.only(top: 6.0),
                           child: const FaIcon(
                             FontAwesomeIcons.chevronDown,
                             size: 14,
                           ),
                         ),
                         style: TextStyle(
                           color: disabled
                               ? Colors.grey
                               : theme.colorScheme.onSurface,
                         ),
                         hint: placeholder != null
                             ? Text(
                                 placeholder,
                                 style: const TextStyle(color: Colors.grey),
                               )
                             : null,
                         items: items
                             .map(
                               (item) => DropdownMenuItem<T>(
                                 value: item.value,
                                 child: Row(
                                   mainAxisSize: MainAxisSize.min,
                                   children: [
                                     if (item.icon != null) ...[
                                       item.icon!,
                                       const SizedBox(width: 8),
                                     ],
                                     Flexible(
                                       child: Text(
                                         item.label,
                                         overflow: TextOverflow.ellipsis,
                                       ),
                                     ),
                                   ],
                                 ),
                               ),
                             )
                             .toList(),
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
                           focusedBorder: OutlineInputBorder(
                             borderRadius: BorderRadius.circular(6),
                             borderSide: BorderSide(
                               color: isError
                                   ? Colors.red.shade400
                                   : theme.colorScheme.primary,
                               width: 1.5,
                             ),
                           ),
                         ),
                       ),
                     ),
                   ),
                 ],
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
