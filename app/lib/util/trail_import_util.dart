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
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

/// Supported trail file extensions for both the in-app picker and inbound
/// share intents. `allowedExtensions` on the file picker is only a hint, so
/// this is also used to re-validate before uploading.
const trailImportExtensions = ['gpx', 'kml', 'kmz', 'tcx', 'fit'];

/// Fallback for the last imported [Trail] when GoRouter's `extra` doesn't
/// survive a same-process router refresh. go_router encodes the current
/// route's `extra` via `json.encoder.convert` whenever it reports state back
/// to the platform (RouteMatchListCodec); a non-JSON-serializable `Trail`
/// fails that encode and is silently replaced with `null` (go_router logs a
/// warning, doesn't throw). A refresh can land seconds after navigating —
/// e.g. authProvider settling while a share-triggered import is already
/// showing the create/edit screen — so `/trail/create/edit`'s route builder
/// reads this as a fallback instead of bouncing back to the source selector.
Trail? pendingImportedTrail;

/// Uploads a local trail file to the server for conversion, parses the returned
/// GPX client-side, and navigates to the trail create/edit screen with the
/// resulting (unsaved) [Trail]. Shared by the New Trail → Import picker and the
/// OS share-sheet handler so both reach `/trail/create/edit` identically.
///
/// Shows an error toast (and does not navigate) on an unsupported extension or
/// a conversion failure. Returns after navigation is scheduled.
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

  // `allowedExtensions` is only a hint — several platforms (and shared files)
  // can carry any type, so reject unsupported types before uploading.
  final ext = p.extension(name).replaceFirst('.', '').toLowerCase();
  if (ext.isEmpty || !trailImportExtensions.contains(ext)) {
    showError();
    return;
  }

  try {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(path, filename: name),
    });
    final res = await ref
        .read(apiProvider)
        .post('/trail/convert', data: formData);
    // The trail is not persisted yet, so the server omits id/created/updated.
    // Supply the same placeholders Trail.empty() uses; the spread lets any
    // server-provided value win. The record is timestamped on save. Waypoint
    // has the same required (non-nullable) id/created/updated fields, so any
    // nested waypoints need the same placeholders or parsing will throw.
    final data = res.data as Map<String, dynamic>;
    final now = DateTime.now().toIso8601String();
    final expand = data['expand'] as Map<String, dynamic>?;
    final waypoints = expand?['waypoints_via_trail'] as List<dynamic>?;
    if (waypoints != null) {
      for (var i = 0; i < waypoints.length; i++) {
        waypoints[i] = {
          'id': '',
          'created': now,
          'updated': now,
          ...waypoints[i] as Map<String, dynamic>,
        };
      }
    }
    var trail = Trail.fromJson({
      'id': '',
      'created': now,
      'updated': now,
      ...data,
    });

    // The unsaved trail carries its raw GPX inline (no file to fetch yet).
    // Parse it into expand.gpx so the route can be drawn on the map. The
    // Gpx object isn't serializable, so this step stays client-side; the
    // bounding box already arrives on the record from gpx2trail.
    final gpxData = trail.expand?.gpxData;
    if (gpxData != null && gpxData.isNotEmpty) {
      final parsedGpx = GpxReader().fromString(sanitizeGpxEmail(gpxData));
      trail = trail.copyWith(
        expand: (trail.expand ?? const TrailExpand()).copyWith(gpx: parsedGpx),
      );
    }

    pendingImportedTrail = trail;
    if (!navContext.mounted) return;
    navContext.push('/trail/create/edit', extra: trail);
  } catch (e) {
    showError();
  }
}
