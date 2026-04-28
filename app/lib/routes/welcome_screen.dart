import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/components/welcome/server_selctor.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';

class WelcomeScreen extends ConsumerWidget {
  const WelcomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final serverSelection = ref.watch(serverSelectionProvider);

    final router = ref.watch(routerProvider);

    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 32.0),
          child: Column(
            children: [
              const Spacer(flex: 1),

              SvgPicture.asset(
                "assets/svgs/logo_text_dark.svg",
                semanticsLabel: 'wanderer logo',
                height: 80,
              ),
              const SizedBox(height: 16),
              Text(
                "${AppLocalizations.of(context)!.welcome_to} wanderer",
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                  letterSpacing: -0.5,
                ),
              ),

              const Spacer(flex: 1),
              const Spacer(flex: 2),

              ServerSelector(),
              const SizedBox(height: 24),
              Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  WandererButton(
                    onPressed: () => {router.push('/login')},
                    primary: true,
                    disabled: serverSelection.value?.selectedServer == null,
                    child: Text(AppLocalizations.of(context)!.login),
                  ),
                  const SizedBox(height: 12),
                  WandererButton(
                    onPressed: () => {router.push('/register')},
                    secondary: true,
                    disabled: serverSelection.value?.selectedServer == null,
                    child: Text(AppLocalizations.of(context)!.register),
                  ),
                ],
              ),
              const SizedBox(height: 32),
            ],
          ),
        ),
      ),
    );
  }
}
