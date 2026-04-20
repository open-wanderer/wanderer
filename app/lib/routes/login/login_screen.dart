import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:wanderer/components/base/wanderer_text_field.dart';
import 'package:wanderer/models/api_error.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import '/i18n/app_localizations.dart';
import 'package:form_builder_validators/form_builder_validators.dart';

class LoginScreen extends ConsumerWidget {
  LoginScreen({super.key});

  final _formKey = GlobalKey<FormBuilderState>();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final loginState = ref.watch(authProvider);

    ref.listen(authProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          String displayMessage = "An unexpected error occurred";

          if (error is DioException) {
            try {
              final apiError = ApiError.fromJson(error.response?.data);

              displayMessage = apiError.message == "Failed to authenticate."
                  ? AppLocalizations.of(context)!.wrong_username_or_password
                  : apiError.message;
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

                WandererTextField(
                  name: 'username',
                  label:
                      "${AppLocalizations.of(context)!.username}/${AppLocalizations.of(context)!.email}",
                  validator: FormBuilderValidators.required(),
                ),
                WandererTextField(
                  name: 'password',
                  label: AppLocalizations.of(context)!.password,
                  isPassword: true,
                  validator: FormBuilderValidators.compose([
                    FormBuilderValidators.required(),
                    FormBuilderValidators.minLength(8),
                  ]),
                ),

                SizedBox(
                  width: double.infinity,
                  child: WandererButton(
                    primary: true,
                    large: true,
                    loading: loginState.isLoading,
                    child: Text(AppLocalizations.of(context)!.login),
                    onPressed: () {
                      if (_formKey.currentState?.saveAndValidate() ?? false) {
                        final data = _formKey.currentState!.value;
                        ref
                            .read(authProvider.notifier)
                            .login(data['username'], data['password']);
                      }
                    },
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
