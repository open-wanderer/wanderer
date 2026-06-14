import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

/// Derives the Valhalla costing string from the trail category name.
///
/// Returns `'bicycle'` when the category name (case-insensitive) contains
/// `'bike'`, `'cycling'`, or `'bicycle'`; otherwise returns `'pedestrian'`.
/// Pattern 4 from 02-RESEARCH.md.
String _costingFor(String? category) {
  final lower = (category ?? '').toLowerCase();
  if (lower.contains('bike') ||
      lower.contains('cycling') ||
      lower.contains('bicycle')) {
    return 'bicycle';
  }
  return 'pedestrian';
}

/// Launches turn-by-turn navigation for [trail] from either detail screen.
///
/// Flow:
/// 1. Guards: needs a parsed GPX with ≥2 points (shows error toast + returns on failure).
/// 2. Derives costing from [trail.expand?.category?.name].
/// 3. Builds the waypoint list `[{'lat': p.latitude, 'lon': p.longitude}]`;
///    downsamples to ≤2000 by taking every Nth point while always keeping
///    the first and last (A4 — the server also caps at 2000, so this is
///    defense-in-depth per T-02-07).
/// 4. POSTs to `/valhalla/navigate` via [apiProvider].
/// 5. Parses [NavigateResponse] from the response body; guards for
///    non-empty maneuvers + shape (V5/T-02-08).
/// 6. Checks `context.mounted` (Pitfall 5 from 02-RESEARCH.md).
/// 7. Pushes `/trail/:id/navigate` with [response] as extra.
///
/// On any error, shows an error toast and returns without navigating (D-07).
/// The caller owns the loading state — this function does not manage spinners.
Future<void> launchNavigation({
  required BuildContext context,
  required WidgetRef ref,
  required Trail trail,
}) async {
  // (0) Guard: location services must be enabled and permission granted.
  final l10n = AppLocalizations.of(context)!;

  void showError(String text) => ref.read(toastProvider.notifier).add(
        ToastMessage(
          type: ToastType.error,
          icon: FontAwesomeIcons.triangleExclamation,
          text: text,
        ),
      );

  if (!await Geolocator.isLocationServiceEnabled()) {
    showError(l10n.location_services_disabled);
    return;
  }

  var permission = await Geolocator.checkPermission();
  if (permission == LocationPermission.deniedForever) {
    showError(l10n.location_permission_permanently_denied);
    return;
  }
  if (permission == LocationPermission.denied) {
    permission = await Geolocator.requestPermission();
    if (permission == LocationPermission.denied ||
        permission == LocationPermission.deniedForever) {
      showError(l10n.location_permission_denied);
      return;
    }
  }

  // (1) Guard: must have a parsed GPX with at least 2 points (V5, T-02-08)
  final gpx = trail.expand?.gpx;
  if (gpx == null) {
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.triangleExclamation,
            text: l10n.couldnt_start_navigation,
          ),
        );
    return;
  }

  final points = gpx.allPoints;
  if (points.length < 2) {
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.triangleExclamation,
            text: l10n.couldnt_start_navigation,
          ),
        );
    return;
  }

  // (2) Derive costing from the trail category (Pattern 4, T-02-09)
  final costing = _costingFor(trail.expand?.category?.name);

  // (3) Build shape list — downsample to ≤500 preserving first+last (A4)
  // trace_route accepts up to ~500 shape points. step = ceil(n/499) guarantees
  // at most 499 regularly-sampled points, so the appended last point can never
  // push the total above 500. Using 499 (not 500) prevents step=1 for n=501.
  List<Map<String, double>> shape;
  if (points.length > 500) {
    final step = (points.length / 499).ceil();
    final sampled = <Map<String, double>>[];
    for (int i = 0; i < points.length; i++) {
      if (i % step == 0) {
        sampled.add({'lat': points[i].latitude, 'lon': points[i].longitude});
      }
    }
    // Always include the last point; deduplicate if already present.
    final lastPoint = {
      'lat': points.last.latitude,
      'lon': points.last.longitude,
    };
    if (sampled.isEmpty || sampled.last != lastPoint) {
      sampled.add(lastPoint);
    }
    shape = sampled;
  } else {
    shape = points
        .map((p) => {'lat': p.latitude, 'lon': p.longitude})
        .toList();
  }

  try {
    // (4) POST to /valhalla/navigate (baseUrl already includes /api/v1)
    final api = ref.read(apiProvider);
    final res = await api.post(
      '/valhalla/navigate',
      data: {'shape': shape, 'costing': costing},
    );

    // (5) Parse response and guard for non-empty content (V5, T-02-08)
    final response = NavigateResponse.fromJson(
      res.data as Map<String, dynamic>,
    );

    if (response.maneuvers.isEmpty || response.shape.isEmpty) {
      // Guard mounted before using context after await (Pitfall 5)
      if (!context.mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.triangleExclamation,
              text: l10n.couldnt_start_navigation,
            ),
          );
      return;
    }

    // (6) Guard against async gap where widget may have been unmounted (Pitfall 5)
    if (!context.mounted) return;

    // (7) Navigate to the navigation screen, passing the response as extra (D-05)
    context.push('/trail/${trail.id}/navigate', extra: response);
  } catch (e) {
    // D-07: on any error (network, parse, etc.) show toast and stay
    // Guard mounted before using context after await (Pitfall 5)
    if (!context.mounted) return;
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.triangleExclamation,
            text: l10n.couldnt_start_navigation,
          ),
        );
  }
}
