import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';

/// A multi-line plain-text input, mirroring [WandererTextField]'s chrome and
/// `flutter_form_builder` integration for fields that don't need rich text
/// (e.g. `Waypoint.description`, which is stored as plain text).
class WandererTextArea extends FormBuilderField<String> {
  final String? label;
  final String? placeholder;
  final bool disabled;
  final int minLines;
  final int? maxLines;

  WandererTextArea({
    super.key,
    required super.name,
    super.validator,
    super.initialValue,
    super.onChanged,
    this.label,
    this.placeholder,
    this.disabled = false,
    this.minLines = 3,
    this.maxLines,
  }) : super(
         builder: (FormFieldState<String?> field) {
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
               TextField(
                 controller: TextEditingController(text: field.value)
                   ..selection = TextSelection.fromPosition(
                     TextPosition(offset: (field.value ?? '').length),
                   ),
                 onChanged: (val) => field.didChange(val),
                 enabled: !disabled,
                 minLines: minLines,
                 maxLines: maxLines,
                 keyboardType: TextInputType.multiline,
                 textAlignVertical: TextAlignVertical.top,
                 style: TextStyle(
                   color: disabled ? Colors.grey : theme.colorScheme.onSurface,
                 ),
                 decoration: InputDecoration(
                   hintText: placeholder,
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
