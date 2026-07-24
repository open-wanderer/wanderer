import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/components/trail/missing_coverage_sheet.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/download_notification_provider.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/region/region_provider.dart';
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

    if (overlapping.isEmpty) {
      // D-04: the trail's bbox falls inside no catalog region at all -- a
      // genuine no-region gap. Non-blocking warning; the download still
      // proceeds below, unchanged, no sheet.
      ref.read(toastProvider.notifier).add(
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
        final selection = await showMissingCoverageSheet(ctx, trail, missing);
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
    await notificationService.showProgress(trail.name, 0, 0);

    try {
      await trailDownloadService.downloadTrail(
        trail,
        onGeneratingChanged: (isGenerating) {
          if (isGenerating) notificationService.showGenerating(trail.name);
        },
        onProgress: (done, total) =>
            notificationService.showProgress(trail.name, done, total),
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
