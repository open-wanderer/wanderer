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
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_deletion_provider.dart';
import 'package:wanderer/provider/trail/trail_download_state_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_search_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/provider/trail/local_trail_provider.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/util/route/map_app.dart';
import 'package:wanderer/util/trail/route_location.dart';

enum TrailAction {
  open,
  directions,
  download,
  update,
  removeDownload,
  edit,
  delete,
}

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

    // Destructive-action AVAILABILITY derives from
    // library membership (`widget.availableOffline`) and authorship, never
    // from a cached-provenance flag on the model -- that flag drifts with
    // network conditions (see `_allowDelete`). Destructive-action SCOPING,
    // one layer down, derives from the identity the action actually
    // destroys: for local capture state that is `owner`/account, resolved
    // by `ownLiveCaptureProvider` against the ROW, never by `syncState` read
    // off the shared cache row (`TrailNotifier._readCached` is scoped only
    // by `savedByUserIds`, so this model can carry another account's
    // `localId` and `syncState`).
    //
    // Unsynced-ness and downloaded-ness are NOT mutually exclusive. A row
    // can be both a live capture and a library member indefinitely: the
    // drain parks a repeatedly-failing upload in `failed` state with no
    // scheduled retry, so the overlap window is unbounded, not a brief
    // race.
    //
    // The watch below is unconditional: `ownLiveCaptureProvider` accepts a
    // null/absent local id and returns `false` for it, so every build reads
    // it, not just unsynced ones.
    final isOwnLiveCapture = ref.watch(ownLiveCaptureProvider(trail.localId));

    // The download family (Download / Update / Remove
    // download) is shown to any library member EXCEPT the account whose own
    // live capture this row is -- for your own not-yet-uploaded capture
    // there is nothing meaningful to download or un-download, and unlike
    // Edit, waiting for connectivity never makes it available. This is what
    // avoids hiding the safe action (Remove download) from a legitimate
    // library member in the overlap state while still offering the
    // destructive one (Delete).
    //
    // `trailHasServerId(trail.id)` preserves the original, still-valid
    // reason for hiding the family: offering it for a row with a blank id
    // (a local-sentinel id is blanked) would fetch from the server with an
    // empty trail id. Without this term, a null `currentAccountId` making
    // `isOwnLiveCapture` resolve to `false` could surface a Download item
    // pointing at nothing, and the `PopupMenuDivider` would otherwise render
    // with no items behind it.
    //
    // The term guards the WHOLE family, not just the not-offline
    // branch. `Update` (the `availableOffline` branch below) calls the same
    // `downloadingTrailIdsProvider.notifier.download(trail)` the Download
    // item does, so an empty `trail.id` reaching it is the same "fetch with
    // an empty trail id" this term exists to prevent. Today no library
    // member can have a blank id -- the overlap only begins after
    // `writeServerTrailId` has run -- but that premise lives in
    // another file and nothing asserted it, so the guard was asymmetric
    // with its own stated rationale.
    //
    // Accepted consequence (not worked around): a hiker who downloaded
    // their OWN trail while its upload was still in flight does not see
    // *Remove download* for it, because `isOwnLiveCapture` is true for them.
    // That follows directly from account scoping and costs nothing -- the
    // store keeps that row on removal anyway, and the trail remains in
    // their own-trails list.
    final showDownloadFamily = !isOwnLiveCapture && trailHasServerId(trail.id);

    final isDownloading = ref
        .watch(downloadingTrailIdsProvider)
        .contains(trail.id);
    // This is the single re-entry guard shared by all three download-
    // family actions (Download / Update / Remove download) -- which of the
    // three renders is decided by `widget.availableOffline` below, not by
    // this flag.
    final downloadEnabled = !isDownloading;
    // Delete is refused (not hidden -- the hiker should see it exists
    // and understand why it's momentarily unavailable) while this trail's
    // local id is in the drain provider's in-flight set.
    final isDraining =
        trail.localId != null &&
        ref.watch(trailSyncProvider).contains(trail.localId);
    // A local-sentinel id is blanked, so '/trail/${trail.id}/map' is
    // '/trail//map' for a not-yet-uploaded trail -- go_router
    // canonicalizes that to '/trail/map', which matches no route.
    final String? mapLocation = trailMapLocation(trail);
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
          onTap: mapLocation == null ? null : () => context.push(mapLocation),
          enabled: mapLocation != null,
          child: ListTile(
            leading: FaIcon(
              FontAwesomeIcons.map,
              size: 18,
              color: mapLocation == null ? Colors.grey : null,
            ),
            title: Text(
              l18n.show_on_map,
              style: mapLocation == null
                  ? const TextStyle(color: Colors.grey)
                  : null,
            ),
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
              // Unsynced: no server copy exists to fetch, and
              // `trailProvider('')` would be a meaningless family instance,
              // so the local model is the only one there is. This is a
              // ROUTING decision (which model to hand the editor), not a
              // destructive-action gate, so it stays read directly off
              // `trail.syncState` -- the scoping rule governs destructive
              // actions, not this.
              if (isUnsyncedState(trail.syncState)) {
                await context.push('/trail/create/edit', extra: trail);
                final localId = trail.localId;
                if (localId != null) {
                  ref.invalidate(localTrailProvider(localId));
                }
                return;
              }

              // `TrailDownloadService` overwrites a downloaded
              // row's `photos` with LOCAL FILE PATHS, so the editor's photo
              // picker (seeded from `trail.localPhotos`/`trail.photos`) was
              // resending those paths under the append-only `photos+` key on
              // every save, doubling the server photo set, and computing
              // `_removedServerPhotos` from an always-empty list made photo
              // removal a silent no-op. Fetching the server copy here, and
              // handing THAT to the editor instead of `trail`, makes the
              // editor structurally unable to receive a cached model, which
              // is what actually stops both bugs.
              final api = ref.read(apiProvider);
              final Trail fetched;
              try {
                fetched = await fetchServerTrail(api, trail.id);
              } catch (e) {
                debugPrint(
                  'trail_dropdown: fetchServerTrail("${trail.id}") failed '
                  'for edit: $e',
                );
                if (!mounted) return;
                ref
                    .read(toastProvider.notifier)
                    .add(
                      ToastMessage(
                        type: ToastType.warning,
                        icon: FontAwesomeIcons.circleExclamation,
                        text: l18n.edit_needs_connection,
                      ),
                    );
                return;
              }

              if (!context.mounted) return;
              // The edit screen invalidates the LIST providers itself;
              // refreshing the single-trail provider stays a caller
              // responsibility.
              await context.push('/trail/create/edit', extra: fetched);
              ref.invalidate(trailProvider(trail.id));
            },
            child: ListTile(
              leading: FaIcon(FontAwesomeIcons.pen, size: 18),
              title: Text(l18n.edit),
              contentPadding: EdgeInsets.zero,
            ),
          ),
        ],
        // Hidden entirely (not disabled) when
        // `showDownloadFamily` is false -- see the comment above where it is
        // computed. This file has no tooltip-on-disabled convention, so a
        // disabled item with no explanation reads as broken.
        if (showDownloadFamily) ...[
          const PopupMenuDivider(),
          if (widget.availableOffline) ...[
            // The old single inert "Available offline" item becomes
            // two flat items -- flat `PopupMenuItem` + `ListTile`, no
            // sub-sheet; this app has no menu-item-opens-a-sub-sheet pattern
            // anywhere and a sheet would bury both actions two taps deep.
            PopupMenuItem<TrailAction>(
              value: TrailAction.update,
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
                    : const FaIcon(FontAwesomeIcons.arrowsRotate, size: 18),
                // Reuses the already-translated `regions_update_action`
                // key ("Update") rather than minting a trail-scoped
                // duplicate -- it is translated in all 14 locales, including
                // a real German "Aktualisieren". Do not "fix" this into a
                // new key.
                title: Text(l18n.regions_update_action),
                // The status the old inert item carried survives here,
                // now translated -- it was previously a hardcoded English
                // literal.
                subtitle: Text(
                  l18n.available_offline,
                  style: const TextStyle(color: Colors.green),
                ),
                contentPadding: EdgeInsets.zero,
              ),
            ),
            PopupMenuItem<TrailAction>(
              value: TrailAction.removeDownload,
              onTap: downloadEnabled
                  ? () => _confirmRemoveDownload(context, trail)
                  : null,
              enabled: downloadEnabled,
              child: ListTile(
                // Not red -- a removal is not a deletion; the red lives on
                // the confirm dialog's own confirm action.
                leading: FaIcon(
                  FontAwesomeIcons.circleMinus,
                  size: 18,
                  color: downloadEnabled ? null : Colors.grey,
                ),
                title: Text(
                  l18n.remove,
                  style: downloadEnabled
                      ? null
                      : const TextStyle(color: Colors.grey),
                ),
                contentPadding: EdgeInsets.zero,
              ),
            ),
          ] else ...[
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
                    : const FaIcon(FontAwesomeIcons.download, size: 18),
                title: Text(l18n.download),
                contentPadding: EdgeInsets.zero,
              ),
            ),
          ],
        ],
        if (_allowDelete(ref, isOwnLiveCapture: isOwnLiveCapture)) ...[
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

  /// Mirrors `settings_offline_regions_screen.dart`'s `_onDeleteRegion`
  /// -- the awaited `showDialog<bool>` + `if (confirmed != true) return;`
  /// form, not this file's own fire-and-forget `_confirmDelete`.
  Future<void> _confirmRemoveDownload(BuildContext context, Trail trail) async {
    // Resolved before the first await -- `context` is a parameter here, not
    // `State.context`, so a post-await `mounted` check does not license
    // reading from it. This file documents the same discipline at its other
    // async call sites below.
    final l18n = AppLocalizations.of(context)!;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(trail.name),
        // One dialog, no connectivity branching, no extra offline-only
        // warning line. The body already states re-downloading is needed,
        // which is the honest cost either way, and refusing removal while
        // offline would be paternalistic -- freeing space is legitimate
        // precisely in the field.
        content: Text(l18n.remove_download_confirm_body),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l18n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(l18n.remove, style: const TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );

    if (confirmed != true) return;
    if (!context.mounted) return;

    // Local-only: under the single-object model the trail still exists on
    // the server, so this must never issue a network delete. It also must
    // NOT pop the route -- the screen stays, this provider's state updates,
    // `availableOffline` flips to false and the menu offers Download again.
    //
    // Awaited: discarding this future dropped any error into
    // the zone as an unhandled async error instead of surfacing it, and
    // skipped `deleteTrail`'s own `state` update.
    await ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id);
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

  /// The escape hatch is gated on THIS account owning the
  /// local row, resolved through `ownLiveCaptureProvider` -> `build()`'s
  /// `isOwnLiveCapture` -> an owner-scoped ObjectBox query
  /// (`isOwnLiveCapture` in `local_trail_store.dart`) that consults the
  /// row's own `owner` and `syncState` and never calls `toModel()`.
  ///
  /// Without that gate, `TrailNotifier._readCached` -- scoped only by
  /// `savedByUserIds` -- could hand account B a model carrying account A's
  /// `localId` and `failed` `syncState`, and a bare `isUnsyncedState` branch
  /// would arm Delete for them with no authorship check at all.
  bool _allowDelete(WidgetRef ref, {required bool isOwnLiveCapture}) {
    // Load-bearing escape hatch, checked first: an unsynced capture is the
    // only copy on earth, and its `author` can be the `Trail.author`
    // placeholder rather than a real actor id, so an authorship-only gate
    // would silently strip the hiker's ability to delete their own
    // recording. The hatch is kept, but scoped to THIS account's own row
    // via `isOwnLiveCapture` -- it can never fire for a row this account
    // merely downloaded.
    if (isOwnLiveCapture) {
      return true;
    }

    final user = ref.watch(authProvider).value;
    if (user == null) return false;

    // Independent of `widget.availableOffline` -- a trail the hiker
    // authored AND downloaded now shows both Remove download and Delete
    // trail. Someone else's downloaded trail shows Remove download only,
    // which is correct: it was never theirs to delete.
    return widget.trail.author == user.actorId;
  }

  void _confirmDelete(BuildContext context, Trail trail) {
    final l18n = AppLocalizations.of(context)!;

    // An unsynced trail's own copy is the only copy on earth, so it
    // needs its own l10n key stating the deletion can't be undone.
    //
    // `delete_trail_confirm` is the SERVER-delete confirm only.
    // The un-download confirm is `remove_download_confirm_body` -- used by
    // `_confirmRemoveDownload` here and by `library_screen.dart` -- and the
    // two must never be merged: un-downloading is genuinely undoable, a
    // server delete is not. `library_screen_remove_guard_test.dart` asserts
    // `delete_trail_confirm` is ABSENT from `library_screen.dart` for
    // exactly this reason.
    //
    // Decided on `trail.id`, NOT `trail.syncState`: `writeServerTrailId`
    // stamps a real server id the instant `PUT /trail/form` is accepted, well
    // before the drain's waypoint loop finishes or the row is retired. A
    // `failed` row -- syncState-wise indistinguishable from a trail that
    // never left the device -- can already be live on the server. Claiming
    // "it hasn't been uploaded yet, so this can't be undone" is false for
    // that row, and it must not get the copy that implies a purely local
    // delete. `trailHasServerId` is exactly the right signal: empty
    // means genuinely device-only, non-empty means the server already has
    // (at least) a partial copy.
    final confirmCopy = trailHasServerId(trail.id)
        ? l18n.delete_trail_confirm
        : l18n.delete_unsynced_trail_confirm;

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

  // Delete means delete-on-the-server, gated on authorship (`_allowDelete`).
  // This method has two branches: unsynced local delete (the trail has never
  // reached the server, so its only copy lives on the device), or fall
  // through to `_deleteOnServer`. The unsynced branch must be checked, and
  // must return, before the fall-through -- an unsynced trail's copy is
  // device-only, and reaching `_deleteOnServer` for it would be meaningless.
  //
  // A third branch used to live here -- un-downloading, gated on a
  // provenance flag that `TrailEntity.toModel()` hardcodes `true` for every
  // cached row and that `TrailNotifier.build()`'s any-exception cache
  // fallback could attach to a server-authored trail after nothing more than
  // a timeout. It made "remove the download" and "delete on the server" the
  // same gesture, decided by a signal that drifts with network conditions.
  // It is gone: un-downloading is now its own labelled menu item
  // (`TrailAction.removeDownload`) with its own confirm
  // (`_confirmRemoveDownload`), which calls only the library-membership
  // provider and never reaches the server.
  Future<void> _deleteTrail(BuildContext context, Trail trail) async {
    // Unsynced: this trail has never reached the server (or, for a null
    // localId below, has no local handle at all), so the ONLY question is
    // where its copy actually lives.
    if (isUnsyncedState(trail.syncState)) {
      final localId = trail.localId;

      // A null localId is NOT a device-only copy -- it is the shape
      // `retireUploadedLocalTrail`'s demote branch and `TrailDownloadService`'s
      // carry-forward can both produce for a row that already has a server
      // id. It must be routed to a real server delete when one exists,
      // matching what `_confirmDelete` already promised via
      // `trailHasServerId` -- never silently treated as having nothing to
      // delete.
      if (localId == null) {
        if (trailHasServerId(trail.id)) {
          return _deleteOnServer(context, trail);
        }
        if (!mounted) return;
        final l18n = AppLocalizations.of(context)!;
        ref
            .read(toastProvider.notifier)
            .add(
              ToastMessage(
                type: ToastType.error,
                icon: FontAwesomeIcons.xmark,
                text: l18n.error_deleting_trail,
              ),
            );
        return;
      }

      // The refusal is only "not silently ignored" if someone reads it. Popping
      // first navigated the user away and THEN discarded the boolean, so a
      // refused delete looked exactly like a completed one: no toast, no
      // explanation, trail still there when they navigated back. The menu's
      // `isDraining` disable does not cover this -- the drain can start between
      // the menu build and this confirm-dialog tap.
      // Resolved before the await: `context` is a parameter here, not
      // `State.context`, so a post-await `mounted` check does not license
      // reading from it.
      final l18n = AppLocalizations.of(context)!;

      // `TrailSync.deleteUnsynced` now issues a real `DELETE /trail/{id}`
      // first when the row already carries a real server id -- a
      // `failed`/`pending`/`uploading` row can, since `writeServerTrailId`
      // stamps that id well before the row is retired. It no longer throws:
      // every outcome (including a failed or refused server DELETE) is
      // classified via `UnsyncedDeleteResult` instead, so the local
      // row and photos are left untouched unless the outcome is `deleted`.
      final result = await ref
          .read(trailSyncProvider.notifier)
          .deleteUnsynced(localId);
      if (!mounted) return;

      switch (result) {
        case UnsyncedDeleteResult.deleted:
          // No-ops for a device-only row (`announce` ignores an empty id),
          // but `deleteUnsynced` issues a real server DELETE first when the
          // row already carries a server id — and such a row can be
          // indexed, so it can be sitting in the map's results.
          ref.read(trailDeletionsProvider.notifier).announce(trail.id);
          if (context.mounted && Navigator.of(context).canPop()) {
            Navigator.of(context).pop();
          }
          return;
        case UnsyncedDeleteResult.blockedInFlight:
          ref
              .read(toastProvider.notifier)
              .add(
                ToastMessage(
                  type: ToastType.warning,
                  icon: FontAwesomeIcons.circleExclamation,
                  text: l18n.delete_blocked_while_uploading,
                ),
              );
          return;
        case UnsyncedDeleteResult.needsConnection:
          ref
              .read(toastProvider.notifier)
              .add(
                ToastMessage(
                  type: ToastType.warning,
                  icon: FontAwesomeIcons.circleExclamation,
                  text: l18n.delete_needs_connection,
                ),
              );
          return;
        case UnsyncedDeleteResult.failed:
          {
            // The drain can retire this row between the
            // screen's last read and this tap, leaving `trail.id` on the
            // blank local sentinel while the trail is alive on the server.
            // `deleteUnsynced` then finds no row, issues no DELETE (the
            // owner-scoped `readLocalTrailServerId` returns null) and
            // reports `failed` -- so the hiker could not delete the trail
            // from this screen at all, under a toast blaming a failure that
            // never happened.
            //
            // `trail_create_screen.dart`'s `resolveNetworkSaveTarget`
            // already handles exactly this shape on the save path. Mirror it
            // here rather than inventing a second recovery route.
            //
            // Safe across accounts: `serverIdForRetired` is itself
            // account-guarded -- a memo minted under another account is
            // refused and returns null -- so this can only ever recover a
            // trail this account actually retired. When it yields nothing
            // we fall through to the original toast unchanged.
            final retiredId = ref
                .read(trailSyncProvider.notifier)
                .serverIdForRetired(localId);
            if (retiredId != null && context.mounted) {
              await _deleteOnServer(context, trail.copyWith(id: retiredId));
              return;
            }
            ref
                .read(toastProvider.notifier)
                .add(
                  ToastMessage(
                    type: ToastType.error,
                    icon: FontAwesomeIcons.xmark,
                    text: l18n.error_deleting_trail,
                  ),
                );
            return;
          }
      }
    }

    if (!context.mounted) return;
    await _deleteOnServer(context, trail);
  }

  /// The real `DELETE /trail/{id}` path, extracted so the unsynced branch's
  /// null-`localId` fall-through can route here directly instead of
  /// falling through the rest of `_deleteTrail`.
  Future<void> _deleteOnServer(BuildContext context, Trail trail) async {
    // Resolved before any await, matching the unsynced branch's own
    // discipline -- `context` is a parameter here, not `State.context`, so a
    // post-await `mounted` check does not license reading from it.
    final l18n = AppLocalizations.of(context)!;
    final router = GoRouter.of(context);

    try {
      await ref.read(trailSaveProvider.notifier).deleteTrail(trail);
      ref.invalidate(trailLibraryProvider);
      ref.invalidate(trailSearchProvider);
      // The map providers are announced to rather than invalidated — they
      // hold their last bounds in instance fields, so invalidating them
      // empties the map instead of refreshing it. See trail_deletion_provider.
      ref.read(trailDeletionsProvider.notifier).announce(trail.id);
      final handle = ref.read(authProvider).value?.preferredUsername;
      if (handle != null) {
        ref.invalidate(profileTrailsProvider('@$handle'));
      }
      if (router.canPop()) router.pop();
    } catch (e) {
      debugPrint('trail_dropdown: _deleteOnServer("${trail.id}") failed: $e');
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.xmark,
              text: l18n.error_deleting_trail,
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
