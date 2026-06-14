import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

/// Reads the cached [NavigateResponse] for [trailId] from ObjectBox.
///
/// Returns null if:
/// - No [TrailEntity] is found for [trailId]
/// - The entity's [navCacheJson] field is null
/// - The JSON cannot be decoded (treat as cache miss — D-11)
NavigateResponse? _readCachedNav(Store store, String trailId) {
  final box = store.box<TrailEntity>();
  final query = box.query(TrailEntity_.id.equals(trailId)).build();
  final entity = query.findFirst();
  query.close();

  if (entity == null) return null;
  final json = entity.navCacheJson;
  if (json == null) return null;

  try {
    return NavigateResponse.fromJson(jsonDecode(json) as Map<String, dynamic>);
  } catch (_) {
    // Undecodable cache treated as miss (D-11)
    return null;
  }
}

/// Silently re-caches [response] for [trailId] in ObjectBox.
///
/// Reads the existing entity by [trailId] and updates [navCacheJson] with
/// the JSON-encoded [response]. Uses a write transaction (D-12). Swallows
/// all errors — a re-cache failure must never surface to the user (D-13).
/// If the trail entity is not found (trail not downloaded), does nothing.
Future<void> _recacheNav(
    Store store, String trailId, NavigateResponse response) async {
  try {
    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.id.equals(trailId)).build();
    final entity = query.findFirst();
    query.close();

    if (entity == null) return;

    entity.navCacheJson = jsonEncode(response.toJson());
    store.runInTransaction(TxMode.write, () {
      box.put(entity);
    });
  } catch (_) {
    // Swallow errors: cache write is best-effort (D-13)
  }
}


/// Launches turn-by-turn navigation for [trail] from either detail screen.
///
/// Flow:
/// 1. Guards: needs a parsed GPX with ≥2 points (shows error toast + returns on failure).
/// 2. Derives costing from [trail.expand?.category?.name].
/// 3. Builds the waypoint list via [buildNavShape] (downsamples to ≤500,
///    preserving first and last — OFFLINE-01/D-08).
/// 4. POSTs to `/valhalla/navigate` via [apiProvider].
/// 5. Parses [NavigateResponse] from the response body; guards for
///    non-empty maneuvers + shape (V5/T-02-08).
/// 6. Checks `context.mounted` (Pitfall 5 from 02-RESEARCH.md).
/// 7. Pushes `/trail/:id/navigate` with `(response, false)` as extra (D-03).
/// 8. Fires an unawaited background re-cache write of the fresh response (D-12, D-13).
///
/// On [DioException] (network failure): reads cached [NavigateResponse] from
/// ObjectBox. If valid (non-empty maneuvers + shape), pushes navigation with
/// `(cached, true)` — isOffline:true (D-09, OFFLINE-02). Otherwise falls
/// through to the error toast.
///
/// On any error when no usable cache exists, shows an error toast and returns
/// without navigating (D-07). The caller owns the loading state — this
/// function does not manage spinners.
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

  // (2) Derive costing from the trail category via shared helper (Pattern 4,
  //     T-02-09). Shared with downloadTrail so cache and live costing match.
  final costing = costingForCategory(trail.expand?.category?.name);

  // (3) Build shape list via shared helper — downsamples to ≤500, preserves
  //     first+last (OFFLINE-01/D-08). Shared with downloadTrail so the cached
  //     and online shapes are byte-identical.
  final shape = buildNavShape(points);

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

    // (7) Navigate to the navigation screen; isOffline:false for online path (D-03)
    context.push('/trail/${trail.id}/navigate', extra: (response, false));

    // (8) Silently re-cache the fresh response in the background (D-12, D-13)
    final store = ref.read(objectBoxProvider);
    unawaited(_recacheNav(store, trail.id, response));
  } on DioException catch (_) {
    // D-09: network failure — attempt cache fallback (OFFLINE-02)
    // Guard mounted before using context after async gap (Pitfall 5)
    if (!context.mounted) return;
    final store = ref.read(objectBoxProvider);
    final cached = _readCachedNav(store, trail.id);
    if (cached != null &&
        cached.maneuvers.isNotEmpty &&
        cached.shape.isNotEmpty) {
      // Valid cache — push with isOffline:true (D-03, OFFLINE-02)
      context.push('/trail/${trail.id}/navigate', extra: (cached, true));
      return;
    }
    // No usable cache — fall through to show error toast (D-11)
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.triangleExclamation,
            text: l10n.couldnt_start_navigation,
          ),
        );
  } catch (_) {
    // D-07: non-Dio errors (parse failures, etc.) show toast and stay
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
