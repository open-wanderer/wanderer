import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/download_notification_provider.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_download_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';

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
