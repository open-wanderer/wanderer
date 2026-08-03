import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_button.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/actions/confirm_signout.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;

    final loginState = ref.watch(authProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.settings),
      ),
      body: ListView(
        children: [
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.circleUser, size: 18),
            title: Text(l10n.my_account),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/account'),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.lock, size: 18),
            title: Text(l10n.privacy),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/privacy'),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.globe, size: 18),
            title: Text(l10n.language_and_units(l10n.language, l10n.units)),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/language'),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.bell, size: 18),
            title: Text(l10n.notifications),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/notifications'),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.layerGroup, size: 18),
            title: Text(l10n.categories),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/categories'),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.map, size: 18),
            title: Text(l10n.settings_offline_regions_title),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/regions'),
          ),
          ListTile(
            leading: const FaIcon(FontAwesomeIcons.palette, size: 18),
            title: Text(l10n.appearance),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/appearance'),
          ),
          Divider(),
          SizedBox(height: 16),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 32.0),
            child: WandererButton(
              secondary: true,
              loading: loginState.isLoading,
              child: Text(AppLocalizations.of(context)!.logout),
              onPressed: () async {
                if (!await confirmSignOutWithUnsyncedTrails(context, ref)) {
                  return;
                }
                ref.read(authProvider.notifier).logout();
              },
            ),
          ),
        ],
      ),
    );
  }
}
