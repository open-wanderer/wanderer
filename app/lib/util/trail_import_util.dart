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

/// Fallback for the last imported [Trail]: go_router can silently null out
/// a non-JSON-serializable `extra` on a same-process router refresh, so
/// `/trail/create/edit`'s route builder falls back to this instead.
Trail? pendingImportedTrail;

/// Uploads a local trail file for conversion, parses the returned GPX, and
/// navigates to the trail create/edit screen with the resulting [Trail].
/// Shared by the Import picker and the OS share-sheet handler.
///
/// Shows an error toast (and does not navigate) on an unsupported extension
/// or a conversion failure.
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
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(path, filename: name),
    });
    final res = await ref
        .read(apiProvider)
        .post('/trail/convert', data: formData);
    // Not persisted yet, so id/created/updated are missing from the server
    // response; supply placeholders (required, non-nullable on the model)
    // for both the trail and any nested waypoints, or parsing throws.
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

    // Parse the inline raw GPX client-side (Gpx isn't serializable) so the
    // route can be drawn on the map.
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
