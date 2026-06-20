import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
import 'package:wanderer/components/settings/email_change_sheet.dart';
import 'package:wanderer/components/settings/password_change_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/profile/profile_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

class SettingsAccountScreen extends ConsumerStatefulWidget {
  const SettingsAccountScreen({super.key});

  @override
  ConsumerState<SettingsAccountScreen> createState() =>
      _SettingsAccountScreenState();
}

class _SettingsAccountScreenState extends ConsumerState<SettingsAccountScreen> {
  bool _avatarLoading = false;

  Future<void> _pickAndUploadAvatar(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    String userId,
  ) async {
    final picked = await ImagePicker().pickImage(source: ImageSource.gallery);
    if (picked == null) return;

    try {
      final formData = FormData.fromMap({
        'avatar': await MultipartFile.fromFile(picked.path),
      });
      setState(() {
        _avatarLoading = true;
      });
      await ref.read(apiProvider).post('/user/$userId/file', data: formData);

      if (!context.mounted) return;

      await ref.read(authProvider.notifier).refresh();
      ref.invalidate(ownProfileProvider);
    } catch (_) {
      if (!context.mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_settings,
            ),
          );
    } finally {
      setState(() {
        _avatarLoading = false;
      });
    }
  }

  Future<void> _deleteAccount(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    String userId,
  ) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.confirm_deletion),
        content: Text(l10n.account_delete_confirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(
              l10n.delete_account,
              style: const TextStyle(color: Colors.red),
            ),
          ),
        ],
      ),
    );

    if (confirmed != true) return;

    if (!context.mounted) return;

    try {
      await ref.read(apiProvider).delete('/user/$userId');

      if (!context.mounted) return;

      await ref.read(authProvider.notifier).logout();
    } catch (_) {
      if (!context.mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_settings,
            ),
          );
    }
  }

  Widget _sectionHeader(BuildContext context, String label, Color color) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Text(
        label,
        style: Theme.of(context).textTheme.titleSmall?.copyWith(color: color),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final user = ref.watch(authProvider).value;
    final settings = ref.watch(settingsProvider);

    final avatarUrl = user?.getFileUrl(user.serverUrl, user.avatar);
    final fallbackUrl =
        'https://api.dicebear.com/7.x/initials/png?seed=${user?.preferredUsername}&backgroundType=gradientLinear';

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.account),
      ),
      body: ListView(
        children: [
          // --- Avatar row (ACCT-01) ---
          Padding(
            padding: const EdgeInsets.all(16),
            child: GestureDetector(
              onTap: user == null
                  ? null
                  : () => _pickAndUploadAvatar(context, ref, l10n, user.id),
              child: Stack(
                alignment: Alignment.center,
                children: [
                  CircleAvatar(
                    radius: 64,
                    backgroundImage: NetworkImage(avatarUrl ?? fallbackUrl),
                    onBackgroundImageError: (e, _) {},
                  ),
                  Positioned.fill(
                    child: Container(
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: Colors.black.withValues(
                          alpha: 0.4,
                        ), // Adjust alpha (0.0 to 1.0) for darkness
                      ),
                    ),
                  ),
                  if (_avatarLoading) ...{
                    CircularProgressIndicator(color: Colors.white),
                  } else ...{
                    FaIcon(FontAwesomeIcons.pen, color: Colors.white),
                  },
                ],
              ),
            ),
          ),

          // --- Bio section (ACCT-02) ---
          _sectionHeader(
            context,
            "${l10n.about} ${user?.username}",
            Theme.of(context).colorScheme.onSurfaceVariant,
          ),
          _BioSection(settings: settings, l10n: l10n),

          // --- Change email (ACCT-03) ---
          ListTile(
            leading: const Icon(Icons.email_outlined),
            title: Text(l10n.change_email),
            onTap: user == null
                ? null
                : () {
                    showModalBottomSheet<void>(
                      context: context,
                      isScrollControlled: true,
                      builder: (_) => EmailChangeSheet(userId: user.id),
                    );
                  },
          ),

          // --- Change password (ACCT-04) ---
          ListTile(
            leading: const Icon(Icons.lock_outline),
            title: Text(l10n.change_password),
            onTap: user == null
                ? null
                : () {
                    showModalBottomSheet<void>(
                      context: context,
                      isScrollControlled: true,
                      builder: (_) => PasswordChangeSheet(userId: user.id),
                    );
                  },
          ),
          Divider(),
          _sectionHeader(context, l10n.danger_zone, Colors.redAccent),
          // --- Delete account (ACCT-05) ---
          ListTile(
            leading: Icon(Icons.delete_outline, color: Colors.redAccent),
            title: Text(
              l10n.delete_account,
              style: TextStyle(color: Colors.red.shade400),
            ),
            onTap: user == null
                ? null
                : () => _deleteAccount(context, ref, l10n, user.id),
          ),
        ],
      ),
    );
  }
}

/// Stateful bio editing widget that holds a [TextEditingController] and
/// re-evaluates the Save button enable condition on every keystroke.
class _BioSection extends ConsumerStatefulWidget {
  const _BioSection({required this.settings, required this.l10n});

  final Settings? settings;
  final AppLocalizations l10n;

  @override
  ConsumerState<_BioSection> createState() => _BioSectionState();
}

class _BioSectionState extends ConsumerState<_BioSection> {
  late TextEditingController _controller;
  String _persisted = '';

  @override
  void initState() {
    super.initState();
    _persisted = widget.settings?.bio ?? '';
    _controller = TextEditingController(text: _persisted);
  }

  @override
  void didUpdateWidget(_BioSection oldWidget) {
    super.didUpdateWidget(oldWidget);
    final newPersisted = widget.settings?.bio ?? '';
    if (newPersisted != _persisted) {
      _persisted = newPersisted;
      // Only update the controller if the text is still equal to the old
      // persisted value (i.e. the user has not started editing).
      if (_controller.text == _persisted) {
        _controller.text = newPersisted;
      }
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  bool get _hasChanged => _controller.text != _persisted;

  Future<void> _save() async {
    final settings = widget.settings;
    if (settings == null) return;
    try {
      await ref
          .read(settingsProvider.notifier)
          .saveToServer(settings.copyWith(bio: _controller.text));
    } catch (_) {
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: widget.l10n.error_saving_settings,
            ),
          );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _controller,
            maxLines: 4,
            onChanged: (_) => setState(() {}),
            decoration: InputDecoration(hintText: widget.l10n.add_bio),
          ),
          const SizedBox(height: 8),
          ElevatedButton(
            onPressed: _hasChanged ? _save : null,
            child: Text(widget.l10n.save),
          ),
        ],
      ),
    );
  }
}
