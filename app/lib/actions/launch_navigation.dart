import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart';
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/actions/request_background_location.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/services/tracelet_position_source.dart';
import 'package:wanderer/util/route/valhalla.dart';

/// Reads the cached [NavigateResponse] for [trailId] from ObjectBox, or null
/// if not found/decodable. Also used by `main.dart`'s launch-time resume flow.
///
/// Scoped to the signed-in account: trail rows are shared across accounts and
/// survive a logout (see `TrailEntity.savedByUserIds`), so an unfiltered read
/// would let one account navigate a trail only another account had downloaded.
NavigateResponse? readCachedNav(Store store, String trailId) {
  final userId = currentAccountId(store);
  if (userId == null) return null;

  final box = store.box<TrailEntity>();
  final query = box
      .query(
        TrailEntity_.id.equals(trailId) &
            TrailEntity_.savedByUserIds.containsElement(userId),
      )
      .build();
  final entity = query.findFirst();
  query.close();

  if (entity == null) return null;
  final json = entity.navCacheJson;
  if (json == null) return null;

  try {
    return NavigateResponse.fromJson(jsonDecode(json) as Map<String, dynamic>);
  } catch (_) {
    return null;
  }
}

/// Reads the route a `nav` session carried on its own row, or null when the
/// row predates [ActiveNavigationEntity.navResponseJson] or its JSON is
/// unusable.
///
/// Preferred over [readCachedNav] on resume: this copy exists for any trail
/// that was navigated, where the trail cache exists only for one the account
/// downloaded.
NavigateResponse? readSessionNav(ActiveNavigationEntity row) {
  final json = row.navResponseJson;
  if (json == null || json.isEmpty) return null;
  try {
    return NavigateResponse.fromJson(jsonDecode(json) as Map<String, dynamic>);
  } catch (_) {
    return null;
  }
}

/// Silently re-caches [response] for [trailId] in ObjectBox. Swallows all
/// errors — a re-cache failure must never surface to the user. No-ops if
/// the trail isn't downloaded.
Future<void> _recacheNav(
  Store store,
  String trailId,
  NavigateResponse response,
) async {
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
    // Cache write is best-effort.
  }
}

/// Launches turn-by-turn navigation for [trail]. Requests a route from
/// Valhalla; on network failure falls back to a cached [NavigateResponse]
/// from ObjectBox if one exists, otherwise shows an error toast. Does not
/// manage loading spinners — that's the caller's responsibility.
Future<void> launchNavigation({
  required BuildContext context,
  required WidgetRef ref,
  required Trail trail,
}) async {
  final l10n = AppLocalizations.of(context)!;

  void showError(String text) => ref
      .read(toastProvider.notifier)
      .add(
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
  // Background location is what keeps tracking alive when the app is cleared
  // from recents — tracelet stops on task removal without it. Shows Play's
  // required disclosure before the system prompt on Android; navigation
  // proceeds either way if declined.
  if (context.mounted) {
    permission = await requestBackgroundLocation(context, ref, permission);
  }

  // Best-effort: seed the live marker with an already-warm fix from the
  // singleton foreground stream (see `foreground_position_stream_provider`)
  // so it doesn't sit blank through tracelet's own cold GPS acquisition in
  // NavigationScreen. Short timeout and swallowed failure — unlike
  // `_openRecorder`'s blocking fetch, a miss here must never delay or block
  // starting navigation, since the tracelet fix will still arrive shortly.
  final pos = await ref
      .read(foregroundPositionStreamProvider.notifier)
      .currentFix(timeout: const Duration(seconds: 3));
  final Position? seedFix = pos == null ? null : seedPositionFrom(pos);

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

  // Shared with downloadTrail so cache and live costing match.
  final costing = costingForTrail(
    trail,
    subcategories: ref.read(subcategoryProvider),
  );

  // Shared with downloadTrail so cached and online shapes are byte-identical.
  final shape = buildNavShape(points);

  try {
    final api = ref.read(apiProvider);
    final res = await api.post(
      '/valhalla/navigate',
      data: {'shape': shape, 'costing': costing},
    );

    final response = NavigateResponse.fromJson(
      res.data as Map<String, dynamic>,
    );

    if (response.maneuvers.isEmpty || response.shape.isEmpty) {
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

    if (!context.mounted) return;

    // extra: (response, isOffline, resumeSeed, seedFix) — null resumeSeed
    // for a fresh launch.
    context.push(
      '/trail/${trail.id}/navigate',
      extra: (response, false, null, seedFix),
    );

    final store = ref.read(objectBoxProvider);
    unawaited(_recacheNav(store, trail.id, response));
  } on DioException catch (_) {
    if (!context.mounted) return;
    final store = ref.read(objectBoxProvider);
    final cached = readCachedNav(store, trail.id);
    if (cached != null &&
        cached.maneuvers.isNotEmpty &&
        cached.shape.isNotEmpty) {
      context.push(
        '/trail/${trail.id}/navigate',
        extra: (cached, true, null, seedFix),
      );
      return;
    }
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
