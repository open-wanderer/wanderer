import 'package:cached_network_image/cached_network_image.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';
import 'package:wanderer/components/base/wanderer_rich_text_editor.dart';
import 'package:wanderer/components/settings/email_change_sheet.dart';
import 'package:wanderer/components/settings/password_change_sheet.dart';
import 'package:wanderer/components/settings/settings_offline_banner.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/profile/profile_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/actions/guard_online.dart';

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
    if (!guardOnline(ref, l10n)) return;

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
      if (mounted) {
        setState(() {
          _avatarLoading = false;
        });
      }
    }
  }

  Future<void> _deleteAccount(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
    String userId,
  ) async {
    if (!guardOnline(ref, l10n)) return;

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

      // Deliberately NOT routed through the shared unsynced-trails sign-out
      // warning gate (see confirm_signout.dart): this sign-out
      // is not a choice the hiker made -- the account was just destroyed
      // server-side via DELETE /user/{id}, so any still-pending trails can
      // never upload regardless of what the hiker does here. Warning them
      // would be pointless; there is nothing to wait for. Do not "fix" this
      // by adding the guard back.
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
    final online = ref.watch(onlineStatusProvider);

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
          const SettingsOfflineBanner(),

          // --- Avatar row ---
          Padding(
            padding: const EdgeInsets.all(16),
            child: GestureDetector(
              onTap: (user == null || !online)
                  ? null
                  : () => _pickAndUploadAvatar(context, ref, l10n, user.id),
              child: Stack(
                alignment: Alignment.center,
                children: [
                  CircleAvatar(
                    radius: 64,
                    backgroundImage: CachedNetworkImageProvider(
                      avatarUrl ?? fallbackUrl,
                    ),
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

          // --- Bio section ---
          _sectionHeader(
            context,
            "${l10n.about} ${user?.username}",
            Theme.of(context).colorScheme.onSurfaceVariant,
          ),
          _BioSection(settings: settings, l10n: l10n),

          // --- Change email ---
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

          // --- Change password ---
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
          // --- Delete account ---
          ListTile(
            leading: Icon(Icons.delete_outline, color: Colors.redAccent),
            title: Text(
              l10n.delete_account,
              style: TextStyle(color: Colors.red.shade400),
            ),
            onTap: (user == null || !online)
                ? null
                : () => _deleteAccount(context, ref, l10n, user.id),
          ),
        ],
      ),
    );
  }
}

/// Stateful bio editing widget wrapping [WandererRichTextEditor], tracking
/// the current HTML so the Save button enables only once it diverges from
/// the persisted value.
class _BioSection extends ConsumerStatefulWidget {
  const _BioSection({required this.settings, required this.l10n});

  final Settings? settings;
  final AppLocalizations l10n;

  @override
  ConsumerState<_BioSection> createState() => _BioSectionState();
}

class _BioSectionState extends ConsumerState<_BioSection> {
  String _persisted = '';
  late String _current;

  // Bumped whenever the persisted value changes from under an unedited
  // editor, forcing WandererRichTextEditor to remount with the fresh
  // initialValue (it only seeds once and never re-reads it otherwise).
  int _editorGeneration = 0;

  @override
  void initState() {
    super.initState();
    _persisted = widget.settings?.bio ?? '';
    _current = _persisted;
  }

  @override
  void didUpdateWidget(_BioSection oldWidget) {
    super.didUpdateWidget(oldWidget);
    final newPersisted = widget.settings?.bio ?? '';
    if (newPersisted != _persisted) {
      // Only remount (resetting in-progress edits) if the user has not
      // started editing yet.
      final unedited = _current == _persisted;
      _persisted = newPersisted;
      if (unedited) {
        _current = newPersisted;
        _editorGeneration++;
      }
    }
  }

  bool get _hasChanged => _current != _persisted;

  Future<void> _save() async {
    final settings = widget.settings;
    if (settings == null) return;
    if (!guardOnline(ref, widget.l10n)) return;
    try {
      await ref
          .read(settingsProvider.notifier)
          .saveToServer(settings.copyWith(bio: _current));
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
    final online = ref.watch(onlineStatusProvider);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          WandererRichTextEditor(
            key: ValueKey(_editorGeneration),
            initialValue: _persisted,
            label: widget.l10n.add_bio,
            onChanged: (html) => setState(() => _current = html),
          ),
          const SizedBox(height: 8),
          ElevatedButton(
            onPressed: (_hasChanged && online) ? _save : null,
            child: Text(widget.l10n.save),
          ),
        ],
      ),
    );
  }
}
