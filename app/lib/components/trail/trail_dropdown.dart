import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:wanderer/components/trail/map_app_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_share.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_download_state_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_search_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/util/map_app.dart';

enum TrailAction { open, directions, download, edit, delete }

class TrailDropdown extends ConsumerStatefulWidget {
  final Trail trail;
  final bool availableOffline;
  const TrailDropdown({
    super.key,
    required this.trail,
    this.availableOffline = false,
  });

  @override
  ConsumerState<TrailDropdown> createState() => _TrailDropdownState();
}

class _TrailDropdownState extends ConsumerState<TrailDropdown> {
  @override
  Widget build(BuildContext context) {
    final trail = widget.trail;
    final l18n = AppLocalizations.of(context)!;

    // After this phase `trail.isLocal` is true for BOTH a downloaded trail
    // and an unsynced one (`TrailEntity.toModel()` hardcodes it), so the
    // download and delete actions below branch on `isUnsynced` instead —
    // it, not `isLocal`, is the signal that distinguishes "on the device
    // only, never reached the server" from "downloaded from the server".
    final isUnsynced = isUnsyncedState(trail.syncState);

    final isDownloading = ref
        .watch(downloadingTrailIdsProvider)
        .contains(trail.id);
    final downloadEnabled = !widget.availableOffline && !isDownloading;
    // D-14: delete is refused (not hidden -- the hiker should see it exists
    // and understand why it's momentarily unavailable) while this trail's
    // local id is in the drain provider's in-flight set.
    final isDraining =
        trail.localId != null &&
        ref.watch(trailSyncProvider).contains(trail.localId);
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
        // D-17: hidden entirely (not disabled) for an unsynced trail. This
        // file has no tooltip-on-disabled convention, so a disabled item
        // with no explanation reads as broken -- and offering it at all
        // would fetch from the server with an empty trail id.
        if (!isUnsynced) ...[
          const PopupMenuDivider(),
          PopupMenuItem<TrailAction>(
            value: TrailAction.download,
            onTap: downloadEnabled
                ? () => ref
                      .read(downloadingTrailIdsProvider.notifier)
                      .download(trail)
                : null,
            enabled: downloadEnabled,
            child: ListTile(
              leading: isDownloading
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : FaIcon(
                      widget.availableOffline
                          ? FontAwesomeIcons.circleCheck
                          : FontAwesomeIcons.download,
                      size: 18,
                      color: widget.availableOffline ? Colors.green : null,
                    ),
              title: Text(
                widget.availableOffline ? 'Available offline' : l18n.download,
                style: widget.availableOffline
                    ? const TextStyle(color: Colors.green)
                    : null,
              ),
              contentPadding: EdgeInsets.zero,
            ),
          ),
        ],
        if (_allowDelete(ref)) ...[
          const PopupMenuDivider(),
          PopupMenuItem<TrailAction>(
            value: TrailAction.delete,
            onTap: isDraining ? null : () => _confirmDelete(context, trail),
            enabled: !isDraining,
            child: ListTile(
              leading: FaIcon(
                FontAwesomeIcons.trash,
                color: isDraining ? Colors.grey : Colors.red,
                size: 18,
              ),
              title: Text(
                l18n.delete,
                style: TextStyle(color: isDraining ? Colors.grey : Colors.red),
              ),
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
    if (widget.trail.isLocal) {
      return true;
    }
    final user = ref.watch(authProvider).value;
    if (user == null) return false;

    return widget.trail.author == user.actorId;
  }

  void _confirmDelete(BuildContext context, Trail trail) {
    final l18n = AppLocalizations.of(context)!;

    // D-14: an unsynced trail's own copy is the only copy on earth, so it
    // needs its own l10n key stating the deletion can't be undone. The
    // shared `delete_trail_confirm` string also doubles as the *un-download*
    // confirm in `library_screen.dart`, where "cannot be undone" is false —
    // un-downloading is genuinely undoable — so the two flows must never
    // share copy.
    final confirmCopy = isUnsyncedState(trail.syncState)
        ? l18n.delete_unsynced_trail_confirm
        : l18n.delete_trail_confirm;

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(confirmCopy),
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

  // This method now handles THREE different actions behind one menu item,
  // ordered unsynced -> downloaded -> server. The order is load-bearing: an
  // unsynced trail ALSO reports `trail.isLocal == true` (`TrailEntity.
  // toModel()` hardcodes it), so putting the un-download branch first would
  // route an unsynced trail into `trailLibraryProvider.deleteTrail(trail.id)`
  // with an empty id -- silently no-oping and leaving the hiker's only copy
  // of the trail on the device with no feedback at all. The unsynced branch
  // must be checked, and must return, before the `isLocal` branch runs.
  Future<void> _deleteTrail(BuildContext context, Trail trail) async {
    // Unsynced: this trail has never reached the server, so its ObjectBox
    // row plus local photo copies ARE the only copy that exists. Refused
    // (not silently ignored) while the trail is mid-drain -- see
    // `TrailSync.deleteUnsynced`'s own in-flight guard (D-14); the menu's
    // `isDraining` disable is a UI-only mirror of that guard, not a
    // replacement for it, since a UI-only check would still lose a race.
    if (isUnsyncedState(trail.syncState) && trail.localId != null) {
      Navigator.of(context).pop();
      await ref.read(trailSyncProvider.notifier).deleteUnsynced(trail.localId!);
      return;
    }

    // For a downloaded trail this means "remove the download", and NOTHING
    // else — the `return` is load-bearing. Without it the method fell
    // through to the server DELETE below, so the only available
    // un-download gesture (the download menu item is inert once
    // downloaded) also destroyed the trail on the server.
    //
    // Reachable while perfectly online, too: `TrailNotifier.build()` falls
    // back to the ObjectBox cache on ANY fetch exception, not just offline
    // (`trail_provider.dart`'s `catch (_)`), and a cached model is stamped
    // `isLocal: true`. One timeout while viewing your own downloaded trail
    // was enough to arm it.
    //
    // This is also what makes `_allowDelete`'s unconditional `true` for local
    // trails safe: un-downloading someone else's trail is harmless, whereas
    // falling through offered a server delete with no ownership check.
    if (trail.isLocal) {
      Navigator.of(context).pop();
      ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id);
      return;
    }

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
    if (Platform.isIOS) {
      // iOS has no system app chooser, so detect the installed map apps and
      // let the user pick. A lone result (normally just Apple Maps) skips the
      // sheet.
      final apps = await installedMapApps();
      if (!mounted) return;

      final app = apps.length == 1
          ? apps.first
          : apps.isEmpty
          ? null
          : await showMapAppSheet(context, apps);
      if (app == null) return;

      if (await launchUrl(Uri.parse(app.directionsUrl(lat, lon)))) return;
    } else {
      // Android: a single geo: intent lets the system show its own chooser.
      final geoUrl = Uri.parse('geo:$lat,$lon?q=$lat,$lon');
      if (await canLaunchUrl(geoUrl) && await launchUrl(geoUrl)) return;
    }

    await launchUrl(
      Uri.parse('https://www.openstreetmap.org/directions?to=$lat,$lon'),
      mode: LaunchMode.externalApplication,
    );
  }
}
