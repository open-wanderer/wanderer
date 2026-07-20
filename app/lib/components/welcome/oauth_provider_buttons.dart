import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/api_error.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/oauth_providers_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

/// Renders an "or" divider followed by one button per OAuth2 provider
/// configured on [serverUrl] — meant to sit below the Login/Register buttons
/// on the welcome screen. Renders nothing while loading, on fetch error, or
/// when the server has no providers configured, so it never disturbs the
/// existing layout on servers without OAuth.
class OAuthProviderButtons extends ConsumerStatefulWidget {
  final String serverUrl;

  const OAuthProviderButtons({super.key, required this.serverUrl});

  @override
  ConsumerState<OAuthProviderButtons> createState() =>
      _OAuthProviderButtonsState();
}

class _OAuthProviderButtonsState extends ConsumerState<OAuthProviderButtons> {
  // Tracks which provider button triggered the in-flight login, so only that
  // button shows a spinner while the others are simply disabled.
  String? _pendingProviderName;

  @override
  Widget build(BuildContext context) {
    final providers = ref.watch(oauthProvidersProvider(widget.serverUrl));
    final authState = ref.watch(authProvider);

    ref.listen(authProvider, (previous, next) {
      next.whenOrNull(
        error: (error, _) {
          if (_pendingProviderName == null) return;
          setState(() => _pendingProviderName = null);

          String displayMessage = "An unexpected error occurred";
          if (error is DioException) {
            try {
              displayMessage = ApiError.fromJson(error.response?.data).message;
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
        data: (_) => _pendingProviderName = null,
      );
    });

    return providers.when(
      data: (list) {
        if (list.isEmpty) return const SizedBox.shrink();
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const SizedBox(height: 24),
            Row(
              children: [
                const Expanded(child: Divider()),
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                  child: Text(
                    AppLocalizations.of(context)!.or,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ),
                const Expanded(child: Divider()),
              ],
            ),
            const SizedBox(height: 24),
            for (final provider in list) ...[
              WandererButton(
                secondary: true,
                loading:
                    authState.isLoading &&
                    _pendingProviderName == provider.name,
                disabled:
                    authState.isLoading &&
                    _pendingProviderName != provider.name,
                onPressed: () {
                  setState(() => _pendingProviderName = provider.name);
                  ref.read(authProvider.notifier).loginWithOAuth(provider);
                },
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (provider.img != null) ...[
                      _ProviderIcon(provider.img!),
                      const SizedBox(width: 8),
                    ],
                    Text(provider.displayName),
                  ],
                ),
              ),
              const SizedBox(height: 12),
            ],
          ],
        );
      },
      loading: () => const SizedBox.shrink(),
      error: (_, _) => const SizedBox.shrink(),
    );
  }
}

/// Decodes an OAuth provider's base64 data-URL logo (SVG or raster) for
/// display. PocketBase serves provider icons as SVGs, so [SvgPicture.memory]
/// is the common path; raster falls back to [Image.memory].
class _ProviderIcon extends StatelessWidget {
  final String dataUrl;

  const _ProviderIcon(this.dataUrl);

  static final _dataUrlPattern = RegExp(r'^data:([^;]+);base64,(.*)$');

  @override
  Widget build(BuildContext context) {
    final match = _dataUrlPattern.firstMatch(dataUrl);
    if (match == null) return const SizedBox(width: 20, height: 20);

    final mimeType = match.group(1)!;
    final bytes = base64Decode(match.group(2)!);

    if (mimeType.contains('svg')) {
      return SvgPicture.memory(bytes, width: 20, height: 20);
    }
    return Image.memory(bytes, width: 20, height: 20);
  }
}
