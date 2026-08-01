import 'dart:io';

import 'package:collection/collection.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:path/path.dart' as p;
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/gpx_conversion_util.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/reverse_geocode_util.dart';
import 'package:wanderer/util/route_planner_handoff_util.dart';
import 'package:wanderer/util/track_save_options_util.dart';

/// Supported trail file extensions for both the in-app picker and inbound
/// share intents. `allowedExtensions` on the file picker is only a hint, so
/// this is also used to re-validate before uploading.
const trailImportExtensions = ['gpx', 'kml', 'kmz', 'tcx', 'fit'];

/// Fallback for the last imported [Trail]: go_router can silently null out
/// a non-JSON-serializable `extra` on a same-process router refresh, so
/// `/trail/create/edit`'s route builder falls back to this instead.
Trail? pendingImportedTrail;

/// Uploads a local trail file for conversion, parses the returned GPX, and
/// navigates to the trail create/edit screen with the resulting [Trail].
/// Shared by the Import picker and the OS share-sheet handler.
///
/// PORT-03: a `.gpx` file is read and converted entirely on-device — no
/// network call for the conversion itself. Every other supported extension
/// is transcoded server-side via [transcodeToGpx] (PORT-05) and then
/// measured locally, same as a native `.gpx`.
///
/// After parsing, gates the same two post-capture toggles the recording and
/// route-planner paths offer (D-15, see `track_save_options_util.dart`):
/// shown only when online, skipped entirely offline (both toggles treated
/// as declined, so the parsed track is used unchanged) — this is what makes
/// an offline import complete end to end with no network call. A cancelled
/// sheet aborts the import with no toast (not an error).
///
/// Shows an error toast (and does not navigate) on an unsupported extension
/// or a genuine conversion failure.
Future<void> importTrailFile({
  required WidgetRef ref,
  required String path,
  required String name,
  required BuildContext navContext,
  required AppLocalizations l10n,
}) async {
  void showError() {
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.circleExclamation,
            text: l10n.trail_source_import_error,
          ),
        );
  }

  // `allowedExtensions` is only a hint, so re-validate before uploading.
  final ext = p.extension(name).replaceFirst('.', '').toLowerCase();
  if (ext.isEmpty || !trailImportExtensions.contains(ext)) {
    showError();
    return;
  }

  try {
    final gpxXml = ext == 'gpx'
        ? await File(path).readAsString()
        : await transcodeToGpx(ref, path, name);

    final gpx = parseGpxSafely(gpxXml);

    if (!navContext.mounted) return;
    final options = await resolveTrackSaveOptions(ref, navContext);
    if (options == null) return;
    if (!navContext.mounted) return;

    // (false, false) covers BOTH the offline path and "the user declined
    // both toggles online" (D-15) — the single most important branch below:
    // it is what makes an offline import work end to end, since neither
    // conditional block runs and no network call is ever attempted.
    final (recalcHeights, followRoads) = options;

    var finalGpx = gpx;
    var finalGpxData = gpxXml;

    if (recalcHeights || followRoads) {
      // Mirror `_saveRecordedTrack`'s pipeline (navigation_screen.dart):
      // full-resolution shape, snap before heights so elevation reflects
      // the final (possibly snapped) shape, original first/last times
      // preserved so the GPX-derived duration survives the transform.
      final points = gpx.allPoints;
      var workingShape = [
        for (final point in points) {'lat': point.lat, 'lon': point.lon},
      ];

      if (followRoads && workingShape.length >= 2) {
        // An imported file has no category yet (the user picks one on
        // trail_create_screen) — 'pedestrian' is the same fallback
        // costingForTrail itself uses for an unresolved category.
        workingShape = await snapShapeToRoads(
          ref,
          buildNavShape(points),
          'pedestrian',
        );
      }

      var heights = const <num>[];
      if (recalcHeights && workingShape.length >= 2) {
        heights = await fetchHeightsForShape(ref, workingShape);
      }

      // Both transform helpers already fall back silently on failure
      // (a failed snap returns its input shape, a failed height fetch
      // returns an empty list), so a mid-import network drop degrades to
      // the untransformed track rather than failing the import outright.
      final originalWaypoints = gpx.allWaypoints;
      finalGpx = mergeHeightsIntoGpx(
        workingShape,
        heights,
        startTime: originalWaypoints.firstOrNull?.time,
        endTime: originalWaypoints.lastOrNull?.time,
      );
      // Re-serialise so the saved track matches its own computed metrics —
      // passing the untransformed original text here would save a track
      // that disagrees with the trail built from finalGpx below.
      finalGpxData = GpxWriter().asString(finalGpx);
    }

    final trail = await buildLocalTrail(
      ref,
      finalGpx,
      fallbackName: name,
      gpxData: finalGpxData,
    );

    pendingImportedTrail = trail;
    if (!navContext.mounted) return;
    navContext.push('/trail/create/edit', extra: trail);
  } catch (e) {
    showError();
  }
}

/// The sole remaining caller of the convert endpoint's `POST` route in the
/// app (PORT-03). Reached only for a non-GPX extension (kml/kmz/tcx/fit)
/// that the app cannot parse itself — the server transcodes it to GPX,
/// which the app then measures on its own via [buildLocalTrail] (PORT-05:
/// server transcodes, app computes).
///
/// Tolerates BOTH response shapes the endpoint may return: a raw
/// `application/gpx+xml` body (a [String] `res.data` — what plan 34-07 makes
/// the endpoint return) and the legacy JSON body with
/// `expand.gpx_data` (a [Map] `res.data` — what it returns today, before
/// 34-07 lands). This is deliberate CLIENT tolerance for server/app version
/// skew in self-hosted setups (D-05's rationale: "self-hosters control their
/// own server/app pairing"), not a server-versioning scheme, and it keeps no
/// trail-computing behaviour alive on the server.
Future<String> transcodeToGpx(WidgetRef ref, String path, String name) async {
  final formData = FormData.fromMap({
    'file': await MultipartFile.fromFile(path, filename: name),
  });
  final res = await ref
      .read(apiProvider)
      .post('/trail/convert', data: formData);

  final data = res.data;
  if (data is String && data.isNotEmpty) {
    return data;
  }
  if (data is Map) {
    final expand = data['expand'] as Map<String, dynamic>?;
    final gpxData = expand?['gpx_data'] as String?;
    if (gpxData != null && gpxData.isNotEmpty) {
      return gpxData;
    }
  }
  throw StateError(
    'transcodeToGpx: convert endpoint response contained no usable GPX data',
  );
}

/// Builds a complete, unsaved [Trail] from a parsed [gpx] entirely on-device
/// (PORT-01/PORT-03) via [trailFromGpx], then performs D-07's optional,
/// best-effort reverse-geocode fill for `location`.
///
/// The geocode step runs only when [ref]'s [onlineStatusProvider] is true
/// AND the trail has a start coordinate; any failure (offline flip mid-call,
/// no address for the point, network error) is swallowed and the trail is
/// still returned with `location` left null — offline, the field is simply
/// empty for the user to type (D-07). `includeRoad: false` and `fullLabel`
/// reproduce the old server behaviour this replaces (the convert endpoint's
/// `+server.ts:87-99`, which called `searchLocationReverse` with an empty
/// options object — i.e. `includeRoad` falsy — and used `fullLabel`,
/// warning-and-continuing on failure).
Future<Trail> buildLocalTrail(
  WidgetRef ref,
  Gpx gpx, {
  String? fallbackName,
  Duration? movingDuration,
  String? gpxData,
}) async {
  final trail = trailFromGpx(
    gpx,
    fallbackName: fallbackName,
    movingDuration: movingDuration,
    gpxData: gpxData,
  );

  final lat = trail.lat;
  final lon = trail.lon;
  if (ref.read(onlineStatusProvider) && lat != null && lon != null) {
    try {
      final result = await searchLocationReverseStructured(
        ref.read(apiProvider),
        lat,
        lon,
        includeRoad: false,
      );
      if (result != null && result.fullLabel.isNotEmpty) {
        return trail.copyWith(location: result.fullLabel);
      }
    } catch (_) {
      // Best-effort only — never blocks the save (D-07).
    }
  }

  return trail;
}
