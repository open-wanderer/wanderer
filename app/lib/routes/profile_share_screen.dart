import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:share_plus/share_plus.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

class ProfileShareScreen extends ConsumerWidget {
  const ProfileShareScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final userEntity = ref.watch(authProvider).value;

    if (userEntity == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    final serverUrl = userEntity.serverUrl.endsWith('/')
        ? userEntity.serverUrl.substring(0, userEntity.serverUrl.length - 1)
        : userEntity.serverUrl;
    final profileUrl = '$serverUrl/profile/${userEntity.preferredUsername}';

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const FaIcon(FontAwesomeIcons.xmark, size: 16),
          onPressed: () => context.pop(),
        ),
        title: Text(AppLocalizations.of(context)!.share_profile),
      ),
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Card(
                  elevation: 0,
                  shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(16),
                    side: BorderSide(
                      color: Theme.of(context).colorScheme.outline,
                    ),
                  ),
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: QrImageView(
                      data: profileUrl,
                      version: QrVersions.auto,
                      size: 220,
                      backgroundColor: Colors.white,
                    ),
                  ),
                ),
                const SizedBox(height: 24),
                Text(
                  profileUrl,
                  textAlign: TextAlign.center,
                  style: TextStyle(fontSize: 13, color: Colors.grey[600]),
                ),
                const SizedBox(height: 32),
                Row(
                  children: [
                    Expanded(
                      child: WandererButton(
                        secondary: true,
                        onPressed: () async {
                          await Clipboard.setData(
                            ClipboardData(text: profileUrl),
                          );
                          ref
                              .read(toastProvider.notifier)
                              .add(
                                ToastMessage(
                                  type: ToastType.success,
                                  icon: FontAwesomeIcons.copy,
                                  text: 'Link copied',
                                ),
                              );
                        },
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            FaIcon(FontAwesomeIcons.copy, size: 14),
                            SizedBox(width: 8),
                            Text(AppLocalizations.of(context)!.copy_link),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: WandererButton(
                        primary: true,
                        onPressed: () {
                          SharePlus.instance.share(
                            ShareParams(text: profileUrl),
                          );
                        },
                        child: Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: [
                            FaIcon(FontAwesomeIcons.shareNodes, size: 14),
                            SizedBox(width: 8),
                            Text(AppLocalizations.of(context)!.share),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
