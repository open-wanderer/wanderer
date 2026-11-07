import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:intl/intl.dart';
import 'package:wanderer/i18n/app_localizations.dart';

class WandererDatePicker extends FormBuilderField<DateTime> {
  final String? label;
  final String? placeholder;
  final FaIconData? icon;
  final bool disabled;
  final DateTime? firstDate;
  final DateTime? lastDate;
  final DateFormat? format;

  WandererDatePicker({
    super.key,
    required super.name,
    super.validator,
    super.initialValue,
    this.label,
    this.placeholder,
    this.icon = FontAwesomeIcons.calendar,
    this.disabled = false,
    this.firstDate,
    this.lastDate,
    this.format,
    super.onChanged,
  }) : super(
         builder: (FormFieldState<DateTime?> field) {
           final theme = Theme.of(field.context);
           final isError = field.hasError;
           final String locale = Localizations.localeOf(
             field.context,
           ).toString();
           final displayFormat = format ?? DateFormat.yMMMMd(locale);

           void handleUpdate(DateTime? value) {
             field.didChange(value);
           }

           Future<void> onOpenPicker() async {
             if (disabled) return;

             final pickedDate = await showDatePicker(
               context: field.context,
               initialDate: field.value ?? DateTime.now(),
               firstDate: firstDate ?? DateTime(1900),
               lastDate: lastDate ?? DateTime(2100),
             );

             if (pickedDate != null) {
               handleUpdate(pickedDate);
             }
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
                       readOnly: true,
                       onTap: onOpenPicker,
                       controller: TextEditingController(
                         text: field.value != null
                             ? displayFormat.format(field.value!)
                             : '',
                       ),
                       enabled: !disabled,
                       style: TextStyle(
                         color: disabled
                             ? Colors.grey
                             : theme.colorScheme.onSurface,
                       ),
                       decoration: InputDecoration(
                         hintText:
                             placeholder ??
                             AppLocalizations.of(field.context)!.select_date,
                         hintStyle: TextStyle(color: Colors.grey),
                         filled: true,
                         fillColor: isError
                             ? const Color(0xFFFEF2F2)
                             : theme.inputDecorationTheme.fillColor,
                         contentPadding: const EdgeInsets.all(12),
                         suffixIcon: (field.value != null && !disabled)
                             ? IconButton(
                                 icon: const FaIcon(
                                   FontAwesomeIcons.xmark,
                                   size: 14,
                                 ),
                                 onPressed: () {
                                   handleUpdate(null);
                                 },
                               )
                             : null,
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
