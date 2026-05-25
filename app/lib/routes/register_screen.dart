import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:form_builder_validators/form_builder_validators.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/base/wanderer_text_field.dart';
import 'package:wanderer/components/welcome/server_selctor.dart';
import 'package:wanderer/models/api_error.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

import '/i18n/app_localizations.dart';

class RegisterScreen extends ConsumerWidget {
  RegisterScreen({super.key});

  final _formKey = GlobalKey<FormBuilderState>();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final registerState = ref.watch(authProvider);

    ref.listen(authProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          String displayMessage = "An unexpected error occurred";

          if (error is DioException) {
            try {
              final apiError = ApiError.fromJson(error.response?.data);

              if (apiError.detail?.data?.email?.code ==
                  "validation_not_unique") {
                displayMessage = AppLocalizations.of(context)!.email_not_unique;
              } else if (apiError.detail?.data?.username?.code ==
                  "validation_not_unique") {
                displayMessage = AppLocalizations.of(
                  context,
                )!.username_not_unique;
              } else {
                displayMessage = apiError.message;
              }
            } catch (_) {
              displayMessage = error.message ?? "Network connection issue";
            }
          } else {
            displayMessage = error.toString();
          }
          ref
              .read(toastProvider.notifier)
              .add(
                ToastMessage(
                  type: ToastType.error,
                  icon: FontAwesomeIcons.circleExclamation,
                  text: displayMessage,
                ),
              );
        },
      );
    });

    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(32.0),
          child: FormBuilder(
            key: _formKey,
            autovalidateMode: AutovalidateMode.onUnfocus,
            child: AutofillGroup(
              child: SingleChildScrollView(
                child: Column(
                  spacing: 12,
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.center,
                  children: [
                    SvgPicture.asset(
                      "assets/svgs/logo_text_twoline_dark.svg",
                      semanticsLabel: 'wanderer logo with text',
                    ),
                    Text(
                      AppLocalizations.of(context)!.slogan,
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    SizedBox(height: 12),

                    ServerSelector(icon: FontAwesomeIcons.pencil),
                    WandererTextField(
                      name: 'username',
                      label: AppLocalizations.of(context)!.username,
                      validator: FormBuilderValidators.required(),
                      autofillHints: const [AutofillHints.newUsername],
                    ),
                    WandererTextField(
                      name: 'email',
                      label: AppLocalizations.of(context)!.email,
                      validator: FormBuilderValidators.compose([
                        FormBuilderValidators.required(),
                        FormBuilderValidators.email(),
                      ]),
                      autofillHints: const [AutofillHints.email],
                      keyboardType: TextInputType.emailAddress,
                    ),
                    WandererTextField(
                      name: 'password',
                      label: AppLocalizations.of(context)!.password,
                      isPassword: true,
                      validator: FormBuilderValidators.compose([
                        FormBuilderValidators.required(),
                        FormBuilderValidators.minLength(8),
                      ]),
                      autofillHints: const [AutofillHints.newPassword],
                    ),

                    SizedBox(
                      width: double.infinity,
                      child: WandererButton(
                        primary: true,
                        large: true,
                        loading: registerState.isLoading,
                        child: Text(AppLocalizations.of(context)!.register),
                        onPressed: () async {
                          if (_formKey.currentState?.saveAndValidate() ??
                              false) {
                            final data = _formKey.currentState!.value;
                            TextInput.finishAutofillContext();
                            ref
                                .read(authProvider.notifier)
                                .register(
                                  data['username'],
                                  data['email'],
                                  data['password'],
                                );
                          }
                        },
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
