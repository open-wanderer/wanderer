import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/base/wanderer_text_field.dart';
import 'package:wanderer/models/api_error.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/offline_guard_util.dart';

import '/i18n/app_localizations.dart';

class EmailChangeSheet extends ConsumerStatefulWidget {
  const EmailChangeSheet({super.key, required this.userId});

  final String userId;

  @override
  ConsumerState<EmailChangeSheet> createState() => _EmailChangeSheetState();
}

class _EmailChangeSheetState extends ConsumerState<EmailChangeSheet> {
  final _formKey = GlobalKey<FormBuilderState>();
  bool _isLoading = false;

  Future<void> _submit() async {
    if (!guardOnline(ref, AppLocalizations.of(context)!)) return;

    if (!(_formKey.currentState?.saveAndValidate() ?? false)) return;

    final v = _formKey.currentState!.value;

    setState(() => _isLoading = true);

    try {
      await ref
          .read(apiProvider)
          .post(
            '/user/${widget.userId}/email',
            data: {
              'email': v['email'],
              'currentPassword': v['currentPassword'],
            },
          );

      if (!mounted) return;

      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.success,
              icon: FontAwesomeIcons.check,
              text: AppLocalizations.of(context)!.email_updated,
            ),
          );

      Navigator.of(context).pop();

      await ref.read(authProvider.notifier).refresh();
    } on DioException catch (error) {
      String message;
      try {
        final apiError = ApiError.fromJson(error.response?.data);
        message = apiError.message;
      } catch (_) {
        message = error.message ?? 'An unexpected error occurred';
      }

      if (!mounted) return;

      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: message,
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
              name: 'email',
              label: l10n.email,
              validator: FormBuilderValidators.compose([
                FormBuilderValidators.required(),
                FormBuilderValidators.email(),
              ]),
            ),
            WandererTextField(
              name: 'currentPassword',
              label: l10n.current_password,
              isPassword: true,
              validator: FormBuilderValidators.required(),
            ),
            WandererButton(
              primary: true,
              large: true,
              loading: _isLoading,
              onPressed: _submit,
              child: Text(l10n.change_email),
            ),
          ],
        ),
      ),
    );
  }
}
