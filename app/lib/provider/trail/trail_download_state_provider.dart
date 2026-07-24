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

    // Coverage guard (GUARD-01/02/03/04): a local-only, synchronous check
    // against the region catalog snapshot, run BEFORE any download starts.
    // D-11: read the already-persisted local snapshot only -- never trigger
    // a network catalog fetch on the download tap.
    final regions = ref.read(regionListNotifierProvider);
    final overlapping = overlappingRegions(trail, regions);
    final missing = missingCoverageRegions(trail, regions);

    MissingCoverageSelection? selection;

    if (overlapping.isEmpty) {
      // D-04: the trail's bbox falls inside no catalog region at all -- a
      // genuine no-region gap. Non-blocking warning; the download still
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
          // diagram's "dismiss -> download() ABORTS".
          state = {...state}..remove(trail.id);
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
    final glyphCacheWarm = ref.read(glyphSpriteCacheProvider.future);

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
    final regionFutures = <Future<void>>[
      for (final region in vectorRegions)
        tileRepoNotifier.downloadVector(region.id),
      for (final region in demRegions) tileRepoNotifier.downloadDem(region.id),
    ];

    // D-10: unified id-42 notification aggregating the trail's onProgress
    // callback with every selected package's live progress -- only when at
    // least one package was selected. The 0-region path below keeps calling
    // showProgress(trail.name, ...) completely unchanged (GUARD-01).
    var lastTrailFraction = 0.0;
    final itemCount = 1 + vectorRegions.length + demRegions.length;

    void updateAggregate() {
      final packageStates = ref.read(tileRepositoryStatusProvider);
      var sum = lastTrailFraction;
      for (final region in vectorRegions) {
        sum += packageStates[region.id]?.vectorProgress ?? 0.0;
      }
      for (final region in demRegions) {
        sum += packageStates[region.id]?.demProgress ?? 0.0;
      }
      final combined = (sum / itemCount).clamp(0.0, 1.0);
      notificationService.showAggregateProgress(
        'Downloading offline content',
        'Downloading… ${(combined * 100).round()}% · $itemCount items',
        (combined * 100).round(),
        100,
      );
    }

    // `Ref.listenManual` (WidgetRef-only) isn't available on a Notifier's
    // plain `Ref`; `ref.container.listen` is the equivalent manually-closed
    // subscription API for provider/notifier code (same underlying
    // ProviderContainer.listen that WidgetRef.listenManual wraps).
    final aggregateSub = hasSelectedPackages
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
        onGeneratingChanged: (isGenerating) {
          if (isGenerating) notificationService.showGenerating(trail.name);
        },
        onProgress: (done, total) {
          if (hasSelectedPackages) {
            lastTrailFraction = total > 0 ? (done / total).clamp(0, 1) : 0;
            updateAggregate();
          } else {
            notificationService.showProgress(trail.name, done, total);
          }
        },
      );
      await notificationService.showSuccess(trail.name);
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
    } finally {
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
      }
    }
    aggregateSub?.close();

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
