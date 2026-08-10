import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_like.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/util/gpx/conversion.dart';

part 'trail_provider.g.dart';

/// Fetches trail [id] from the server, network-only.
///
/// Extracted out of `TrailNotifier.build` so the edit flow has a seam that
/// CANNOT serve a cached model: this function never reads
/// ObjectBox and never falls back to the stored copy, unlike `build`, which
/// wraps this call in its own `try`/`catch` and falls back to the offline
/// cache on failure. Throws on any failure -- the caller decides what
/// "no server trail" means for its own flow.
Future<Trail> fetchServerTrail(Dio api, String id) async {
  final response = await api.get(
    "/trail/$id",
    queryParameters: {
      "expand":
          "category,waypoints_via_trail,trail_share_via_trail.actor,trail_like_via_trail,tags,author",
    },
  );

  if (response.data == null) {
    throw Exception("No trail data received from server");
  }

  var trail = Trail.fromJson(response.data);

  if (trail.gpx != null && trail.gpx!.isNotEmpty) {
    final gpxResponse = await api.get("/files/trails/$id/${trail.gpx}");

    if (gpxResponse.data == null) {
      throw Exception("No gpx data received from server");
    }

    // parseGpxSafely, not a bare GpxReader: this GPX came off the server
    // and was authored by any user on the instance (or federated in), so
    // it needs the full sanitize chain. Parsing it directly used to throw
    // FormatException on tags package:gpx parses with the throwing
    // double.parse/int.parse (<hdop></hdop>, <sat></sat>, <pdop>N/A</pdop>
    // and friends) — and a broad `catch` around this call would swallow it
    // and silently degrade to the offline cache, showing a stale trail with
    // no indication why.
    //
    // compute(): a long trail's XML parse is tens of milliseconds of
    // synchronous CPU, and this runs on every trail-detail open — off the
    // UI thread it stops eating the open animation's frames. compute (not a
    // raw Isolate.run closure): a closure captures its enclosing scope
    // frame, which here can drag non-sendable objects into the isolate
    // message; compute sends exactly the string. Throws propagate across
    // the isolate boundary unchanged, so the error semantics above are
    // preserved.
    final parsedGpx = await compute(parseGpxSafely, gpxResponse.data as String);

    trail = trail.copyWith(
      expand: (trail.expand ?? const TrailExpand()).copyWith(
        gpx: parsedGpx,
        gpxData: gpxResponse.data,
      ),
    );
  }

  return trail;
}

@riverpod
class TrailNotifier extends _$TrailNotifier {
  /// Online always fetches; the ObjectBox row is the offline fallback
  /// only (see the `catch` block below). Do not reintroduce a family-key flag
  /// for display source; that broke every consumer.
  @override
  FutureOr<Trail> build(String id) async {
    final api = ref.watch(apiProvider);

    try {
      final trail = await fetchServerTrail(api, id);

      // The record and the GPX are both already in hand at this
      // point, so writing them into a downloaded row costs zero additional
      // network traffic. Photos are deliberately not refreshed here --
      // that stays the explicit *Update* action's job.
      _refreshLibraryRow(trail);

      return trail;
    } catch (_) {
      final cached = _readCached(id);
      if (cached != null) return cached;
      rethrow;
    }
  }

  /// The trigger: opportunistically refreshes [trail]'s downloaded
  /// library row, if this account has one.
  ///
  /// Its own `try`/`catch` is not optional defensiveness -- this call sits
  /// INSIDE `build`'s `try`, so an escaping exception would be caught by the
  /// `catch (_)` above and silently downgrade a perfectly good server
  /// response to the stale cached copy, the exact inversion this phase
  /// exists to eliminate.
  void _refreshLibraryRow(Trail trail) {
    try {
      final store = ref.read(objectBoxProvider);
      final accountId = currentAccountId(store);
      if (accountId == null) return;
      applyServerTrailToLibraryRow(store, accountId: accountId, trail: trail);
    } catch (e) {
      debugPrint(
        'TrailNotifier: _refreshLibraryRow failed for "${trail.id}": $e',
      );
    }
  }

  /// The downloaded copy of [id], or null when there is no usable one.
  ///
  /// Scoped to the signed-in account: trail rows are shared and survive a
  /// logout (see `TrailEntity.savedByUserIds`), so an unfiltered read here
  /// would serve one account the cached copy of a private trail only another
  /// account had downloaded.
  Trail? _readCached(String id) {
    final store = ref.read(objectBoxProvider);
    final userId = currentAccountId(store);
    if (userId == null) return null;

    final box = store.box<TrailEntity>();
    final query = box
        .query(
          TrailEntity_.id.equals(id) &
              TrailEntity_.savedByUserIds.containsElement(userId),
        )
        .build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return null;

    // Guarded for the same reason as TrailLibraryNotifier.build(): toModel()
    // parses the cached GPX and can throw. Letting it escape when this is
    // called from the fetch failure's catch block would replace the ORIGINAL
    // failure (why we fell back to the cache at all) with an unrelated parse
    // error. A corrupt cache entry means "no usable cache", so report none and
    // let the caller surface the real cause.
    try {
      return entity.toModel();
    } catch (e, st) {
      debugPrint('TrailNotifier: cached trail "$id" failed to parse: $e\n$st');
      return null;
    }
  }

  Future<void> like(String actorId) async {
    final trail = state.value;
    if (trail == null) return;

    final api = ref.read(apiProvider);

    try {
      await api.put("/trail-like", data: {"actor": actorId, "trail": trail.id});

      final newLike = TrailLike(actor: actorId, trail: trail.id);

      state = AsyncData(
        trail.copyWith(
          likeCount: trail.likeCount + 1,
          expand: (trail.expand ?? const TrailExpand()).copyWith(
            trailLikeViaTrail: [
              ...(trail.expand?.trailLikeViaTrail ?? []),
              newLike,
            ],
          ),
        ),
      );
    } catch (_) {
      // Leave state unchanged on failure (graceful degradation).
    }
  }

  Future<void> unlike(String actorId) async {
    final trail = state.value;
    if (trail == null) return;

    final api = ref.read(apiProvider);

    try {
      await api.post(
        "/trail-like/delete",
        data: {"actor": actorId, "trail": trail.id},
      );

      final remaining = (trail.expand?.trailLikeViaTrail ?? [])
          .where((l) => l.actor != actorId)
          .toList();

      state = AsyncData(
        trail.copyWith(
          likeCount: trail.likeCount > 0 ? trail.likeCount - 1 : 0,
          expand: (trail.expand ?? const TrailExpand()).copyWith(
            trailLikeViaTrail: remaining,
          ),
        ),
      );
    } catch (_) {
      // Leave state unchanged on failure (graceful degradation).
    }
  }
}
