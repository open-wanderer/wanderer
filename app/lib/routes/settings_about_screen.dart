import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:package_info_plus/package_info_plus.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:wanderer/i18n/app_localizations.dart';

const _githubUrl = 'https://github.com/open-wanderer/wanderer';
const _discordUrl = 'https://discord.gg/USSEBY98CP';
const _privacyPolicyUrl = 'https://openwanderer.com/privacy';
const _changelogUrl = 'https://wanderer.to/changelog/';
const _licenseUrl = '$_githubUrl/blob/main/LICENSE';

class SettingsAboutScreen extends StatefulWidget {
  const SettingsAboutScreen({super.key});

  @override
  State<SettingsAboutScreen> createState() => _SettingsAboutScreenState();
}

class _SettingsAboutScreenState extends State<SettingsAboutScreen> {
  // Held in state so a rebuild (theme switch) doesn't re-query the platform.
  final Future<PackageInfo> _packageInfo = PackageInfo.fromPlatform();

  /// Formats the version the way the app bug report template asks for it,
  /// e.g. "0.1.0 (10)", so users can copy it straight into an issue.
  String _formatVersion(PackageInfo info) => info.buildNumber.isEmpty
      ? info.version
      : '${info.version} (${info.buildNumber})';

  /// Opens the app bug report form with the app version already filled in.
  Uri _bugReportUri(String? version) =>
      Uri.parse('$_githubUrl/issues/new').replace(
        queryParameters: {
          'template': 'app_bug_report.yml',
          'app-version': ?version,
        },
      );

  Future<void> _open(Uri uri) =>
      launchUrl(uri, mode: LaunchMode.externalApplication);

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const BackButtonIcon(),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.about),
      ),
      body: FutureBuilder<PackageInfo>(
        future: _packageInfo,
        builder: (context, snapshot) {
          final version = snapshot.hasData
              ? _formatVersion(snapshot.data!)
              : null;
          return ListView(
            children: [
              const SizedBox(height: 32),
              Center(
                child: SvgPicture.asset(
                  "assets/svgs/logo_text_twoline_${theme.brightness.name}.svg",
                  semanticsLabel: 'wanderer logo with text',
                ),
              ),
              const SizedBox(height: 16),
              Center(
                child: SelectableText(
                  version == null ? '' : l10n.about_version(version),
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
              const SizedBox(height: 32),
              const Divider(),
              ListTile(
                leading: const FaIcon(FontAwesomeIcons.github, size: 18),
                title: const Text('GitHub'),
                subtitle: Text(l10n.about_github_subtitle),
                trailing: const Icon(Icons.open_in_new, size: 18),
                onTap: () => _open(Uri.parse(_githubUrl)),
              ),
              ListTile(
                leading: const FaIcon(FontAwesomeIcons.discord, size: 18),
                title: const Text('Discord'),
                subtitle: Text(l10n.about_discord_subtitle),
                trailing: const Icon(Icons.open_in_new, size: 18),
                onTap: () => _open(Uri.parse(_discordUrl)),
              ),
              ListTile(
                leading: const FaIcon(FontAwesomeIcons.bug, size: 18),
                title: Text(l10n.about_report_bug),
                trailing: const Icon(Icons.open_in_new, size: 18),
                onTap: () => _open(_bugReportUri(version)),
              ),
              ListTile(
                leading: const FaIcon(
                  FontAwesomeIcons.clockRotateLeft,
                  size: 18,
                ),
                title: Text(l10n.about_changelog),
                trailing: const Icon(Icons.open_in_new, size: 18),
                onTap: () => _open(Uri.parse(_changelogUrl)),
              ),
              const Divider(),
              ListTile(
                leading: const FaIcon(FontAwesomeIcons.shieldHalved, size: 18),
                title: Text(l10n.about_privacy_policy),
                trailing: const Icon(Icons.open_in_new, size: 18),
                onTap: () => _open(Uri.parse(_privacyPolicyUrl)),
              ),
              ListTile(
                leading: const FaIcon(FontAwesomeIcons.scaleBalanced, size: 18),
                title: Text(l10n.about_license),
                subtitle: const Text('GNU AGPL v3.0'),
                trailing: const Icon(Icons.open_in_new, size: 18),
                onTap: () => _open(Uri.parse(_licenseUrl)),
              ),
              ListTile(
                leading: const FaIcon(FontAwesomeIcons.fileLines, size: 18),
                title: Text(l10n.about_licenses),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => showLicensePage(
                  context: context,
                  applicationName: 'wanderer',
                  applicationVersion: version,
                ),
              ),
            ],
          );
        },
      ),
    );
  }
}
