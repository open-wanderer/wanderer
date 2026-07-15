import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_share.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/download_notification_provider.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_download_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_search_provider.dart';

enum TrailAction { open, directions, download, edit, delete }

class TrailDropdown extends ConsumerStatefulWidget {
  final Trail trail;
  const TrailDropdown({super.key, required this.trail});

  @override
  ConsumerState<TrailDropdown> createState() => _TrailDropdownState();
}

class _TrailDropdownState extends ConsumerState<TrailDropdown> {
  bool _isDownloading = false;

  @override
  Widget build(BuildContext context) {
    final trail = widget.trail;
    final l18n = AppLocalizations.of(context)!;
    final isOffline = ref
        .watch(trailLibraryProvider)
        .any((t) => t.id == trail.id);
    final downloadEnabled = !isOffline && !_isDownloading;
    return PopupMenuButton<TrailAction>(
      offset: const Offset(0, 48),
      borderRadius: BorderRadius.all(Radius.circular(56)),
      icon: FaIcon(
        FontAwesomeIcons.ellipsisVertical,
        color: Theme.of(context).colorScheme.onSurface,
        size: 18,
      ),
      style: IconButton.styleFrom(
        backgroundColor: Theme.of(context).colorScheme.surface,
      ),
      itemBuilder: (BuildContext context) => <PopupMenuEntry<TrailAction>>[
        PopupMenuItem<TrailAction>(
          value: TrailAction.open,
          onTap: () => context.push('/trail/${trail.id}/map'),
          child: ListTile(
            leading: FaIcon(FontAwesomeIcons.map, size: 18),
            title: Text(l18n.show_on_map),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        PopupMenuItem<TrailAction>(
          value: TrailAction.directions,
          onTap: trail.lat != null && trail.lon != null
              ? () => _openDirections(trail.lat!, trail.lon!)
              : null,
          child: ListTile(
            leading: FaIcon(
              FontAwesomeIcons.car,
              size: 18,
              color: trail.lat == null ? Colors.grey : null,
            ),
            title: Text(
              l18n.directions,
              style: trail.lat == null
                  ? const TextStyle(color: Colors.grey)
                  : null,
            ),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        if (_canEditTrail(ref)) ...[
          const PopupMenuDivider(),
          PopupMenuItem<TrailAction>(
            value: TrailAction.edit,
            onTap: () async {
              await context.push('/trail/create/edit', extra: trail);
              ref.invalidate(trailProvider(trail.id));
            },
            child: ListTile(
              leading: FaIcon(FontAwesomeIcons.pen, size: 18),
              title: Text(l18n.edit),
              contentPadding: EdgeInsets.zero,
            ),
          ),
        ],
        const PopupMenuDivider(),
        PopupMenuItem<TrailAction>(
          value: TrailAction.download,
          onTap: downloadEnabled
              ? () => _downloadTrail(context, ref, trail)
              : null,
          enabled: downloadEnabled,
          child: ListTile(
            leading: _isDownloading
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : FaIcon(
                    isOffline
                        ? FontAwesomeIcons.circleCheck
                        : FontAwesomeIcons.download,
                    size: 18,
                    color: isOffline ? Colors.green : null,
                  ),
            title: Text(
              isOffline ? 'Available offline' : l18n.download,
              style: isOffline ? const TextStyle(color: Colors.green) : null,
            ),
            contentPadding: EdgeInsets.zero,
          ),
        ),
        if (_allowDelete(ref)) ...[
          const PopupMenuDivider(),
          PopupMenuItem<TrailAction>(
            value: TrailAction.delete,
            onTap: () => _confirmDelete(context, trail),
            child: ListTile(
              leading: FaIcon(
                FontAwesomeIcons.trash,
                color: Colors.red,
                size: 18,
              ),
              title: Text(l18n.delete, style: TextStyle(color: Colors.red)),
              contentPadding: EdgeInsets.zero,
            ),
          ),
        ],
      ],
    );
  }

  bool _canEditTrail(WidgetRef ref) {
    final user = ref.watch(authProvider).value;
    if (user == null) return false;

    final trail = widget.trail;
    return trail.author == user.actorId ||
        (trail.expand?.trailShareViaTrail?.any(
              (s) =>
                  s.permission == TrailPermission.edit &&
                  s.actor == user.actorId,
            ) ??
            false);
  }

  bool _allowDelete(WidgetRef ref) {
    final user = ref.watch(authProvider).value;
    if (user == null) return false;

    return widget.trail.author == user.actorId;
  }

  void _confirmDelete(BuildContext context, Trail trail) {
    final l18n = AppLocalizations.of(context)!;

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(l18n.delete_trail_confirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: Text(l18n.cancel),
          ),
          TextButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              _deleteTrail(context, trail);
            },
            child: Text(l18n.delete, style: const TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );
  }

  Future<void> _deleteTrail(BuildContext context, Trail trail) async {
    final router = GoRouter.of(context);

    try {
      await ref.read(trailSaveProvider.notifier).deleteTrail(trail);
      ref.invalidate(trailLibraryProvider);
      ref.invalidate(trailSearchProvider);
      final handle = ref.read(authProvider).value?.preferredUsername;
      if (handle != null) {
        ref.invalidate(profileTrailsProvider('@$handle'));
      }
      if (router.canPop()) router.pop();
    } catch (e) {
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.xmark,
              text: 'Error deleting trail',
            ),
          );
    }
  }

  Future<void> _openDirections(double lat, double lon) async {
    final nativeUrl = Uri.parse(
      Platform.isIOS ? 'maps://?daddr=$lat,$lon' : 'geo:$lat,$lon?q=$lat,$lon',
    );
    final webUrl = Uri.parse(
      'https://www.openstreetmap.org/directions?to=$lat,$lon',
    );
    if (await canLaunchUrl(nativeUrl)) {
      await launchUrl(nativeUrl);
    } else {
      await launchUrl(webUrl, mode: LaunchMode.externalApplication);
    }
  }

  void _downloadTrail(BuildContext context, WidgetRef ref, Trail trail) async {
    setState(() => _isDownloading = true);

    final trailDownloadService = ref.read(trailDownloadServiceProvider);
    final notificationService = ref.read(downloadNotificationServiceProvider);
    // Captured up front: this notifier is keepAlive, so it outlives the
    // widget. Calling it later via the captured reference (instead of
    // `ref.read` again) avoids touching WidgetRef after the widget is
    // unmounted, which throws if the screen is closed mid-download.
    final toastNotifier = ref.read(toastProvider.notifier);

    // Trail download is a second, independent trigger for the shared
    // app-wide glyph/sprite cache warm. Fire it concurrently with the trail
    // download and await it separately (below) so a glyph-cache failure never
    // fails or corrupts the trail entity write. Idempotent + keepAlive → a
    // no-op if the map was already opened first.
    final glyphCacheWarm = ref.read(glyphSpriteCacheProvider.future);

    toastNotifier.add(
      ToastMessage(
        type: ToastType.info,
        icon: FontAwesomeIcons.download,
        text: 'Downloading ${trail.name}...',
      ),
    );
    await notificationService.showProgress(trail.name, 0, 0);

    try {
      await trailDownloadService.downloadTrail(
        trail,
        onProgress: (done, total) =>
            notificationService.showProgress(trail.name, done, total),
      );
      await notificationService.showSuccess(trail.name);
      toastNotifier.add(
        ToastMessage(
          type: ToastType.success,
          icon: FontAwesomeIcons.circleCheck,
          text: 'Trail saved for offline use',
        ),
      );
    } catch (e) {
      await notificationService.showError(trail.name);
      toastNotifier.add(
        ToastMessage(
          type: ToastType.error,
          icon: FontAwesomeIcons.xmark,
          text: 'Error saving trail',
        ),
      );
    } finally {
      if (mounted) setState(() => _isDownloading = false);
    }

    // Await the shared cache warm separately: its failure
    // is isolated from the trail download's success/failure above so a
    // glyph/sprite miss never surfaces as a trail-download error.
    try {
      await glyphCacheWarm;
    } catch (_) {
      // Best-effort: glyph/sprite cache warm failure must not block or fail the
      // offline trail download.
    }
  }
}
