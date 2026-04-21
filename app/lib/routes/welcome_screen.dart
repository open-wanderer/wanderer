import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/current_server_provider.dart';
import 'package:wanderer/provider/router_provider.dart';

class WelcomeScreen extends ConsumerWidget {
  const WelcomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final serverURL = ref.watch(currentServerProvider);

    final router = ref.watch(routerProvider);
    final api = ref.watch(apiProvider);

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

              _buildServerSelector(context, router, serverURL),
              const SizedBox(height: 24),
              Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  WandererButton(
                    onPressed: () => {router.push('/login')},
                    primary: true,
                    disabled: serverURL == null,
                    child: Text(AppLocalizations.of(context)!.login),
                  ),
                  const SizedBox(height: 12),
                  WandererButton(
                    onPressed: () => {}, // router.push('/register')
                    secondary: true,
                    disabled: serverURL == null,
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

  Widget _buildServerSelector(
    BuildContext context,
    GoRouter router,
    String? serverUrl,
  ) {
    final bool hasServer = serverUrl != null && serverUrl.isNotEmpty;

    final theme = Theme.of(context);

    return Material(
      type: MaterialType.card,
      color: theme.colorScheme.surfaceContainerLow,
      borderRadius: BorderRadius.circular(16),
      clipBehavior: Clip.antiAlias,
      child: ListTile(
        onTap: () {
          router.push('/select-server');
        },
        leading: CircleAvatar(
          backgroundColor: theme.colorScheme.primaryContainer,
          child: FaIcon(
            FontAwesomeIcons.fediverse,
            color: theme.colorScheme.onPrimaryContainer,
            size: 20,
          ),
        ),
        title: Text(
          hasServer
              ? serverUrl.replaceFirst(RegExp(r'https?://'), '')
              : "Select a Server",
          style: theme.textTheme.bodyMedium?.copyWith(
            fontWeight: FontWeight.bold,
            color: hasServer ? null : theme.colorScheme.onSurfaceVariant,
          ),
        ),
        subtitle: Text(
          hasServer
              ? "Tap to change instance"
              : "Required to login or register",
          style: theme.textTheme.bodySmall,
        ),
        trailing: const Icon(Icons.arrow_forward_ios_rounded, size: 14),

        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      ),
    );
  }
}
