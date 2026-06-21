import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/base/wanderer_text_field.dart';
import 'package:dio/dio.dart';
import 'package:wanderer/models/api_error.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

import '/i18n/app_localizations.dart';

class PasswordChangeSheet extends ConsumerStatefulWidget {
  const PasswordChangeSheet({super.key, required this.userId});

  final String userId;

  @override
  ConsumerState<PasswordChangeSheet> createState() =>
      _PasswordChangeSheetState();
}

class _PasswordChangeSheetState extends ConsumerState<PasswordChangeSheet> {
  final _formKey = GlobalKey<FormBuilderState>();
  bool _isLoading = false;

  Future<void> _submit() async {
    if (!(_formKey.currentState?.saveAndValidate() ?? false)) return;

    final v = _formKey.currentState!.value;

    setState(() => _isLoading = true);

    try {
      await ref.read(apiProvider).post(
        '/user/${widget.userId}',
        data: {
          'oldPassword': v['currentPassword'],
          'password': v['newPassword'],
          'passwordConfirm': v['confirmNewPassword'],
        },
      );

      if (!mounted) return;

      ref.read(toastProvider.notifier).add(
        ToastMessage(
          type: ToastType.success,
          icon: FontAwesomeIcons.check,
          text: AppLocalizations.of(context)!.new_password_success,
        ),
      );

      Navigator.of(context).pop();
    } on DioException catch (error) {
      String message;
      try {
        final apiError = ApiError.fromJson(error.response?.data);
        message = apiError.message;
      } catch (_) {
        message = error.message ?? AppLocalizations.of(context)!.error_updating_password;
      }
      if (!mounted) return;
      ref.read(toastProvider.notifier).add(
        ToastMessage(
          type: ToastType.error,
          icon: FontAwesomeIcons.circleExclamation,
          text: message,
        ),
      );
    } catch (_) {
      if (!mounted) return;
      ref.read(toastProvider.notifier).add(
        ToastMessage(
          type: ToastType.error,
          icon: FontAwesomeIcons.circleExclamation,
          text: AppLocalizations.of(context)!.error_updating_password,
        ),
      );
    } finally {
      if (mounted) {
        setState(() => _isLoading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: EdgeInsets.fromLTRB(
        16,
        24,
        16,
        16 + MediaQuery.of(context).viewInsets.bottom,
      ),
      child: FormBuilder(
        key: _formKey,
        autovalidateMode: AutovalidateMode.onUnfocus,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          spacing: 16,
          children: [
            WandererTextField(
              name: 'currentPassword',
              label: l10n.current_password,
              isPassword: true,
              validator: FormBuilderValidators.required(),
            ),
            WandererTextField(
              name: 'newPassword',
              label: l10n.new_password,
              isPassword: true,
              validator: FormBuilderValidators.required(),
            ),
            WandererTextField(
              name: 'confirmNewPassword',
              label: l10n.password_confirm,
              isPassword: true,
              validator: FormBuilderValidators.compose([
                FormBuilderValidators.required(),
                (val) {
                  final pw = _formKey.currentState?.fields['newPassword']?.value
                      as String?;
                  return (val != pw) ? l10n.passwords_must_match : null;
                },
              ]),
            ),
            WandererButton(
              primary: true,
              large: true,
              loading: _isLoading,
              onPressed: _submit,
              child: Text(l10n.change_password),
            ),
          ],
        ),
      ),
    );
  }
}
