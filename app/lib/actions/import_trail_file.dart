import 'dart:io';

import 'package:collection/collection.dart';
import 'package:flutter/foundation.dart' show compute, debugPrint;
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
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/geo/reverse_geocode.dart';
import 'package:wanderer/util/route/planner_handoff.dart';
import 'package:wanderer/actions/resolve_track_save_options.dart';

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
/// A `.gpx` file is read and converted entirely on-device — no
/// network call for the conversion itself. Every other supported extension
/// is transcoded server-side via [transcodeToGpx] and then
/// measured locally, same as a native `.gpx`.
///
/// After parsing, gates the same two post-capture toggles the recording and
/// route-planner paths offer (see `resolve_track_save_options.dart`):
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
  void showError(String text) {
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.circleExclamation,
            text: text,
          ),
        );
  }

  final isOffline = await ref
      .read(onlineStatusProvider.notifier)
      .refresh()
      .then((online) => !online);

  // `allowedExtensions` is only a hint, so re-validate before uploading.
  final ext = p.extension(name).replaceFirst('.', '').toLowerCase();
  if (ext.isEmpty || !trailImportExtensions.contains(ext)) {
    showError(l10n.trail_source_import_error);
    return;
  }
  if (ext != 'gpx' && isOffline) {
    showError(l10n.trail_source_offline_import_error);
    return;
  }

  // The `try` deliberately ENDS after `buildLocalTrail`. It used to
  // span the navigation push too, so a throw from `navContext.push` — or from
  // anything after `pendingImportedTrail` was assigned — showed the user
  // "import failed" for an import that had in fact succeeded, while leaving a
  // stale non-null `pendingImportedTrail` global behind.
  final Trail trail;
  try {
    final gpxXml = ext == 'gpx'
        ? await File(path).readAsString()
        : await transcodeToGpx(ref, path, name);

    // Off the UI thread: a file import is user-initiated with a spinner up,
    // and a big GPX parse used to jank the whole frame right at the tap.
    // compute (not a raw Isolate.run closure): a closure captures its
    // enclosing scope frame — here that includes a BuildContext and the
    // widget binding, which are unsendable; compute sends only the string.
    final gpx = await compute(parseGpxSafely, gpxXml);

    final hasUsablePoint =
        gpx.allWaypoints.isNotEmpty ||
        gpx.rtes.any(
          (r) => r.rtepts.any((p) => p.lat != null && p.lon != null),
        );
    if (!hasUsablePoint) {
      debugPrint(
        'importTrailFile: "$name" contains no track point with both lat '
        'and lon; refusing to import an empty trail',
      );
      showError(l10n.trail_source_import_error);
      return;
    }

    if (!navContext.mounted) return;
    final options = await resolveTrackSaveOptions(
      ref,
      navContext,
      TrackSaveOptionsSource.import,
    );
    if (options == null) return;
    if (!navContext.mounted) return;

    // (false, false) covers BOTH the offline path and "the user declined
    // both toggles online" — the single most important branch below:
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
        // `fallbackShape` is the full-resolution geometry, NOT the ≤500-point
        // request hint: a failed/rejected snap must leave the imported track
        // at its own resolution rather than persisting a decimation of it
        workingShape = await snapShapeToRoads(
          ref,
          buildNavShape(points),
          'pedestrian',
          fallbackShape: workingShape,
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
      // `source: gpx` is load-bearing, not defensive: only the track
      // GEOMETRY is being replaced here, so the source document's own
      // metadata name/description, its `<wpt>` markers, its routes and its
      // track name must survive the transform. Without it the re-serialised
      // `finalGpxData` below — which is what gets uploaded and persisted —
      // would permanently drop every imported waypoint and fall back to
      // naming the trail after the file.
      finalGpx = mergeHeightsIntoGpx(
        workingShape,
        heights,
        startTime: originalWaypoints.firstOrNull?.time,
        endTime: originalWaypoints.lastOrNull?.time,
        source: gpx,
      );
      // Re-serialise so the saved track matches its own computed metrics —
      // passing the untransformed original text here would save a track
      // that disagrees with the trail built from finalGpx below.
      finalGpxData = await compute(serializeGpxToXml, finalGpx);
    }

    trail = await buildLocalTrail(
      ref,
      finalGpx,
      fallbackName: name,
      gpxData: finalGpxData,
    );
  } catch (e, st) {
    // Every distinct failure mode in the block above — an unreadable file,
    // transcodeToGpx's StateError, a FormatException from an unsanitised
    // tag, a StateError from a <trkpt> missing lat/lon — collapses into one
    // untyped toast. Logging the exception is what makes the import path
    // diagnosable in the field at all; it was previously bound and
    // dropped on the floor.
    debugPrint('importTrailFile failed for "$name": $e\n$st');
    showError(l10n.trail_source_import_error);
    return;
  }

  pendingImportedTrail = trail;
  if (!navContext.mounted) return;
  navContext.push('/trail/create/edit', extra: trail);
}

/// The sole remaining caller of the convert endpoint's `POST` route in the
/// app. Reached only for a non-GPX extension (kml/kmz/tcx/fit)
/// that the app cannot parse itself — the server transcodes it to GPX,
/// which the app then measures on its own via [buildLocalTrail]: the server
/// transcodes, the app computes.
///
/// Tolerates BOTH response shapes the endpoint may return: a raw
/// `application/gpx+xml` body (a [String] `res.data`) and the legacy JSON
/// body with `expand.gpx_data` (a [Map] `res.data` — what older servers
/// still return). This is deliberate CLIENT tolerance for server/app version
/// skew in self-hosted setups (the rationale: "self-hosters control their
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
/// via [trailFromGpx], then performs the optional,
/// best-effort reverse-geocode fill for `location`.
///
/// The geocode step runs only when [ref]'s [onlineStatusProvider] is true
/// AND the trail has a start coordinate; any failure (offline flip mid-call,
/// no address for the point, network error) is swallowed and the trail is
/// still returned with `location` left null — offline, the field is simply
/// empty for the user to type. `includeRoad: false` and `fullLabel`
/// reproduce the old server behaviour this replaces (the convert endpoint
/// called `searchLocationReverse` with an empty
/// options object — i.e. `includeRoad` falsy — and used `fullLabel`,
/// warning-and-continuing on failure).
Future<Trail> buildLocalTrail(
  WidgetRef ref,
  Gpx gpx, {
  String? fallbackName,
  Duration? movingDuration,
  String? gpxData,
}) async {
  // compute(): trailFromGpx runs the full ported metrics computation over
  // every trackpoint (distance/elevation accumulation) — pure CPU that
  // otherwise lands on the UI thread at exactly the moments that must stay
  // responsive (recording Stop tap, planner finish, file import). The
  // computation itself is byte-identical — only the thread changes. compute
  // with an explicit args record, never a capturing closure: the enclosing
  // scope holds a WidgetRef, which is unsendable.
  final trail = await compute(trailFromGpxIsolate, (
    gpx: gpx,
    fallbackName: fallbackName,
    movingDuration: movingDuration,
    gpxData: gpxData,
  ));

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
      // Best-effort only — never blocks the save.
    }
  }

  return trail;
}
