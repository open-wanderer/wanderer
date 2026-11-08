import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';

class WandererTextField extends FormBuilderField<String> {
  final String? label;
  final String? placeholder;
  final FaIconData? icon;
  final bool disabled;
  final bool isPassword;

  /// Platform autofill hints (e.g. [AutofillHints.username]).
  ///
  /// This is what makes a password manager — Bitwarden, 1Password, the
  /// system keychain — recognise the field. It only works when the field
  /// also sits inside an [AutofillGroup]; see login_screen.dart.
  final Iterable<String>? autofillHints;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final ValueChanged<String>? onSubmitted;

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
    this.autofillHints,
    this.keyboardType,
    this.textInputAction,
    this.onSubmitted,
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
                     child: _WandererTextFieldInput(
                       field: field,
                       placeholder: placeholder,
                       disabled: disabled,
                       isPassword: isPassword,
                       isError: isError,
                       autofillHints: autofillHints,
                       keyboardType: keyboardType,
                       textInputAction: textInputAction,
                       onSubmitted: onSubmitted,
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

/// The [TextField] itself, split out so it can own a [TextEditingController]
/// that outlives a rebuild.
///
/// The controller has to be stable for autofill to work: a password manager
/// fills every field in the [AutofillGroup] at once, including ones that are
/// not focused, by writing straight into their controllers. A controller
/// rebuilt on every frame would drop that write (and, before this, was also
/// never disposed and reset the caret on every keystroke).
class _WandererTextFieldInput extends StatefulWidget {
  const _WandererTextFieldInput({
    required this.field,
    required this.disabled,
    required this.isPassword,
    required this.isError,
    this.placeholder,
    this.autofillHints,
    this.keyboardType,
    this.textInputAction,
    this.onSubmitted,
  });

  final FormFieldState<String?> field;
  final bool disabled;
  final bool isPassword;
  final bool isError;
  final String? placeholder;
  final Iterable<String>? autofillHints;
  final TextInputType? keyboardType;
  final TextInputAction? textInputAction;
  final ValueChanged<String>? onSubmitted;

  @override
  State<_WandererTextFieldInput> createState() =>
      _WandererTextFieldInputState();
}

class _WandererTextFieldInputState extends State<_WandererTextFieldInput> {
  late final TextEditingController _controller = TextEditingController(
    text: widget.field.value ?? '',
  );

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant _WandererTextFieldInput oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Pick up value changes that did not come from typing here — a form
    // reset, or a programmatic setValue. Typing already leaves the two in
    // sync, so this never fights the caret.
    final value = widget.field.value ?? '';
    if (value != _controller.text) {
      _controller.value = TextEditingValue(
        text: value,
        selection: TextSelection.collapsed(offset: value.length),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return TextField(
      controller: _controller,
      onChanged: widget.field.didChange,
      onSubmitted: widget.onSubmitted,
      enabled: !widget.disabled,
      obscureText: widget.isPassword,
      autocorrect: !widget.isPassword,
      enableSuggestions: !widget.isPassword,
      autofillHints: widget.disabled ? null : widget.autofillHints,
      keyboardType: widget.keyboardType,
      textInputAction: widget.textInputAction,
      style: TextStyle(
        color: widget.disabled ? Colors.grey : theme.colorScheme.onSurface,
      ),
      decoration: InputDecoration(
        hintText: widget.placeholder,
        filled: true,
        fillColor: widget.isError
            ? const Color(0xFFFEF2F2)
            : theme.inputDecorationTheme.fillColor,
        contentPadding: const EdgeInsets.all(12),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(6),
          borderSide: BorderSide(
            color: widget.isError
                ? Colors.red.shade400
                : theme.colorScheme.outline,
          ),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(6),
          borderSide: BorderSide(
            color: widget.isError
                ? Colors.red.shade400
                : theme.colorScheme.primary,
            width: 1.5,
          ),
        ),
      ),
    );
  }
}
