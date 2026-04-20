import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';

class WandererTextField extends FormBuilderField<String> {
  final String? label;
  final String? placeholder;
  final FaIconData? icon;
  final bool disabled;
  final bool isPassword;

  WandererTextField({
    super.key,
    required super.name,
    super.validator,
    super.initialValue,
    this.label,
    this.placeholder,
    this.icon,
    this.disabled = false,
    this.isPassword = false,
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
                     child: TextField(
                       controller: TextEditingController(text: field.value)
                         ..selection = TextSelection.fromPosition(
                           TextPosition(offset: (field.value ?? '').length),
                         ),
                       onChanged: (val) => field.didChange(val),
                       enabled: !disabled,
                       obscureText: isPassword,
                       style: TextStyle(
                         color: disabled
                             ? Colors.grey
                             : theme.colorScheme.onSurface,
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
