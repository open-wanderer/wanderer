import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/components/trail/missing_coverage_sheet.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/download_notification_provider.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/region/region_provider.dart';
import 'package:wanderer/provider/region/tile_repository_provider.dart';
import 'package:wanderer/provider/router_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_download_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/util/trail_coverage_util.dart';

part 'trail_download_state_provider.g.dart';

/// Trail ids currently being downloaded, shared across every download entry
/// point (detail screen button, dropdown menu item) so starting a download
/// from one immediately disables the others instead of allowing a duplicate,
/// racing download of the same trail. `keepAlive` so the in-progress state
/// survives whichever widget happens to rebuild or unmount mid-download.
@Riverpod(keepAlive: true)
class DownloadingTrailIds extends _$DownloadingTrailIds {
  @override
  Set<String> build() => {};

  Future<void> download(Trail trail) async {
    if (state.contains(trail.id)) return;
    state = {...state, trail.id};

    // CR-01: hoisted above the outer try/finally below so the post-try
    // region-futures wait, aggregate-subscription close, and glyph-cache
    // await (all of which must run AFTER the trail download settles, exactly
    // as before) can still reference them. Populated inside the try; the
    // only ways to reach the code below the try are (a) normal completion,
    // where all three are assigned, since (b) the dismiss-abort `return` and
    // any exception both exit via the outer `finally` first.
    final regionFutures = <Future<void>>[];
    ProviderSubscription? aggregateSub;
    Future<void>? glyphCacheWarm;
    // Gap 1 (26-05, Part B): set once the trail's own download resolves so
    // the deferred aggregate success below (fired after all region futures
    // settle) knows whether the trail side actually succeeded.
    var trailSucceeded = false;

    try {
      // Coverage guard (GUARD-01/02/03/04): a local-only, synchronous check
      // against the region catalog snapshot, run BEFORE any download starts.
      // D-11: read the already-persisted local snapshot only -- never trigger
      // a network catalog fetch on the download tap.
      final regions = ref.read(regionListNotifierProvider);
      // Usable coverage = overlapping regions that either already cover offline
      // or are downloadable from the server right now. A region that overlaps
      // but is still building/errored or was removed from the server is NOT
      // usable coverage, so a trail covered only by such regions correctly
      // warns "not covered by any offered region" instead of silently
      // downloading with no offline map — and never gets offered a Download
      // action that can only fail.
      final usable = usableCoverageRegions(trail, regions);
      final missing = missingCoverageRegions(trail, regions);

      MissingCoverageSelection? selection;

      if (usable.isEmpty) {
        // D-04: the trail's bbox falls inside no usable catalog region at all
        // -- a genuine no-region gap. Non-blocking warning; the download still
        // proceeds below, unchanged, no sheet.
        ref
            .read(toastProvider.notifier)
            .add(
              ToastMessage(
                type: ToastType.info,
                icon: FontAwesomeIcons.triangleExclamation,
                text: "Part of this trail isn't covered by any offered region.",
              ),
            );
      } else if (missing.isNotEmpty) {
        // One or more overlapping regions aren't downloaded/updateAvailable --
        // surface the missing-coverage sheet (GUARD-02/GUARD-03).
        final ctx = navigatorKey.currentContext;
        if (ctx != null) {
          selection = await showMissingCoverageSheet(ctx, trail, missing);
          if (selection == null) {
            // Dismissed (swipe/tap-outside) -- abort the whole download.
            // Nothing starts at all, matching the RESEARCH architecture
            // diagram's "dismiss -> download() ABORTS". The outer `finally`
            // below clears trail.id from state (CR-01).
            return;
          }
        }
        // ctx == null: never strand the user -- fall through to a trail-only
        // download exactly as the fully-covered path below.
      }
      // else: overlapping.isNotEmpty && missing.isEmpty -- fully covered
      // (GUARD-01): fall straight through to the existing download body below,
      // no sheet, no extra toast, byte-for-byte unchanged from today.

      final trailDownloadService = ref.read(trailDownloadServiceProvider);
      final notificationService = ref.read(downloadNotificationServiceProvider);
      final toastNotifier = ref.read(toastProvider.notifier);

      // Trail download is a second, independent trigger for the shared
      // app-wide glyph/sprite cache warm. Fire it concurrently with the trail
      // download and await it separately (below) so a glyph-cache failure never
      // fails or corrupts the trail entity write. Idempotent + keepAlive → a
      // no-op if the map was already opened first.
      glyphCacheWarm = ref.read(glyphSpriteCacheProvider.future);

      toastNotifier.add(
        ToastMessage(
          type: ToastType.info,
          icon: FontAwesomeIcons.download,
          text: 'Downloading ${trail.name}...',
        ),
      );

      // D-08/D-09: start every checked region package alongside the trail, all
      // fire-and-forget -- never awaited before the trail download starts.
      // downloadVector/downloadDem are already idempotent/re-entry-guarded, so
      // no second guard is added here.
      final vectorRegions = selection?.vectorRegions ?? const <RegionEntity>[];
      final demRegions = selection?.demRegions ?? const <RegionEntity>[];
      final hasSelectedPackages =
          vectorRegions.isNotEmpty || demRegions.isNotEmpty;
      final tileRepoNotifier = ref.read(tileRepositoryStatusProvider.notifier);

      // D-10: unified id-42 notification aggregating the trail's onProgress
      // callback with every selected package's live progress -- only when at
      // least one package was selected. The 0-region path below keeps calling
      // showProgress(trail.name, ...) completely unchanged (GUARD-01).
      var lastTrailFraction = 0.0;
      final itemCount = 1 + vectorRegions.length + demRegions.length;

      // Gap 1 (26-05, Part A): per-package monotonic completion latches.
      // tileRepositoryStatusProvider's vectorProgress/demProgress fields are
      // an EPHEMERAL "currently downloading" signal that
      // tile_repository_provider.dart deliberately clears back to null the
      // instant a package finishes (its documented per-region Settings/
      // Regions "downloading" signal -- untouched, NOT modified here). A raw
      // `?? 0.0` read of that field cannot distinguish "not started" from
      // "just finished" -- both read null -- so a finished package's
      // contribution would snap from 1.0 back to 0.0 the moment it
      // completes. These latch maps fix that: live progress only ever RAISES
      // a package's latched contribution (see updateAggregate below), and
      // each package's own future forces its latch to a full 1.0 on
      // completion (see the whenComplete callbacks below), so the
      // contribution can never move backwards once set. Two separate maps
      // (never one keyed solely by region.id) so a region checked for BOTH
      // Vector and DEM doesn't collide on a single key.
      final vectorLatched = {for (final region in vectorRegions) region.id: 0.0};
      final demLatched = {for (final region in demRegions) region.id: 0.0};

      void updateAggregate() {
        final packageStates = ref.read(tileRepositoryStatusProvider);
        var sum = lastTrailFraction;
        for (final region in vectorRegions) {
          final live = packageStates[region.id]?.vectorProgress;
          if (live != null && live > vectorLatched[region.id]!) {
            vectorLatched[region.id] = live;
          }
          sum += vectorLatched[region.id]!;
        }
        for (final region in demRegions) {
          final live = packageStates[region.id]?.demProgress;
          if (live != null && live > demLatched[region.id]!) {
            demLatched[region.id] = live;
          }
          sum += demLatched[region.id]!;
        }
        final combined = (sum / itemCount).clamp(0.0, 1.0);
        notificationService
            .showAggregateProgress(
              'Downloading offline content',
              'Downloading… ${(combined * 100).round()}% · $itemCount items',
              (combined * 100).round(),
              100,
            )
            .catchError((_) {});
      }

      regionFutures.addAll([
        for (final region in vectorRegions)
          tileRepoNotifier.downloadVector(region.id).whenComplete(() {
            vectorLatched[region.id] = 1.0;
            updateAggregate();
          }),
        for (final region in demRegions)
          tileRepoNotifier.downloadDem(region.id).whenComplete(() {
            demLatched[region.id] = 1.0;
            updateAggregate();
          }),
      ]);

      // `Ref.listenManual` (WidgetRef-only) isn't available on a Notifier's
      // plain `Ref`; `ref.container.listen` is the equivalent manually-closed
      // subscription API for provider/notifier code (same underlying
      // ProviderContainer.listen that WidgetRef.listenManual wraps).
      aggregateSub = hasSelectedPackages
          ? ref.container.listen(
              tileRepositoryStatusProvider,
              (_, _) => updateAggregate(),
            )
          : null;

      if (hasSelectedPackages) {
        updateAggregate();
      } else {
        await notificationService.showProgress(trail.name, 0, 0);
      }

      try {
        await trailDownloadService.downloadTrail(
          trail,
          onProgress: (done, total) {
            if (hasSelectedPackages) {
              lastTrailFraction = total > 0 ? (done / total).clamp(0, 1) : 0;
              updateAggregate();
            } else {
              notificationService
                  .showProgress(trail.name, done, total)
                  .catchError((_) {});
            }
          },
        );
        // Gap 1 (26-05, Part B): the trail's own contribution is driven to a
        // full 1.0 the instant downloadTrail() resolves -- guarantees its
        // term reaches full in the aggregate even if the final onProgress
        // tick never fired (e.g. the download ended in the generating
        // phase) -- and trailSucceeded records that the trail side is done,
        // consumed by the deferred aggregate-success call after the region
        // futures below settle.
        lastTrailFraction = 1.0;
        trailSucceeded = true;
        // Gap 1 (26-05, Part B): only fire the immediate id-42 success on
        // the trail-only path (no packages selected). When packages ARE
        // selected, the id-42 notification must keep advancing until they
        // settle too -- see the deferred success call below, after
        // aggregateSub?.close().
        if (regionFutures.isEmpty) {
          await notificationService.showSuccess(trail.name);
        }
        ref.invalidate(trailLibraryProvider);
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
      }
    } finally {
      // CR-01: the single, sole cleanup of trail.id -- always runs, whether
      // the method above completed normally, dismissed via the missing-
      // coverage sheet, or threw anywhere in between. The download button is
      // never permanently stranded/disabled. Unlock timing is unchanged: this
      // still fires the instant the trail download settles, NOT delayed until
      // the region packages below finish.
      state = {...state}..remove(trail.id);
    }

    // Best-effort: a region package failure must never fail or error the
    // trail download's own success/error path -- it surfaces through the
    // region engine's own Settings/Regions status instead.
    if (regionFutures.isNotEmpty) {
      try {
        await Future.wait(regionFutures);
      } catch (_) {
        // Isolated from the trail download above; nothing to do here.
      } finally {
        // CR-02/GUARD-04: once the guard's own region packages settle (success
        // or failure), invalidate regionListNotifierProvider so a still-alive
        // RegionListNotifier (e.g. Settings/Offline Regions mounted) re-reads
        // the now-current status from ObjectBox instead of a stale cached
        // ToOne.target -- mirrors settings_offline_regions_screen.dart's
        // _save/_onCancelVector pattern. Only fires when a region mutation
        // actually happened (inside the isNotEmpty guard).
        ref.invalidate(regionListNotifierProvider);
      }
    }
    aggregateSub?.close();

    // Gap 1 (26-05, Part B): the deferred id-42 success. By the time
    // Future.wait(regionFutures) above has returned, every selected
    // package's whenComplete has already latched its contribution to 1.0
    // (Part A), so the final updateAggregate() call shows 100% -- only THEN
    // does this call replace it with the success state, instead of firing
    // the instant the trail alone resolved. Only fires when packages were
    // selected AND the trail side actually succeeded (trailSucceeded) --
    // matches the trail-only-path guard above, so a failed trail download
    // never gets a false success notification here.
    if (regionFutures.isNotEmpty && trailSucceeded) {
      await ref
          .read(downloadNotificationServiceProvider)
          .showSuccess(trail.name)
          .catchError((_) {});
    }

    // Await the shared cache warm separately: its failure is isolated from
    // the trail download's success/failure above so a glyph/sprite miss
    // never surfaces as a trail-download error.
    try {
      await glyphCacheWarm;
    } catch (_) {
      // Best-effort: glyph/sprite cache warm failure must not block or fail
      // the offline trail download.
    }
  }
}
