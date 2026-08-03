import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_like.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/gpx/conversion.dart';

part 'trail_provider.g.dart';

@riverpod
class TrailNotifier extends _$TrailNotifier {
  @override
  FutureOr<Trail> build(String id) async {
    final api = ref.watch(apiProvider);

    try {
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
        // and friends) — and the broad `catch (_)` below would swallow it and
        // silently degrade to the offline cache, showing a stale trail with no
        // indication why.
        final parsedGpx = parseGpxSafely(gpxResponse.data as String);

        trail = trail.copyWith(
          expand: (trail.expand ?? const TrailExpand()).copyWith(
            gpx: parsedGpx,
            gpxData: gpxResponse.data,
          ),
        );
      }

      return trail;
    } catch (_) {
      // Offline fallback, scoped to the signed-in account: trail rows are
      // shared and survive a logout (see `TrailEntity.savedByUserIds`), so an
      // unfiltered read here would serve one account the cached copy of a
      // private trail only another account had downloaded.
      final store = ref.read(objectBoxProvider);
      final userId = currentAccountId(store);
      if (userId == null) rethrow;

      final box = store.box<TrailEntity>();
      final query = box
          .query(
            TrailEntity_.id.equals(id) &
                TrailEntity_.savedByUserIds.containsElement(userId),
          )
          .build();
      final entity = query.findFirst();
      query.close();

      // Guarded for the same reason as TrailLibraryNotifier.build(): toModel()
      // parses the cached GPX and can throw. Letting it escape from inside
      // this catch block would replace the ORIGINAL failure (why we fell back
      // to the cache at all) with an unrelated parse error. A corrupt cache
      // entry means "no usable cache", so fall through and surface the real
      // cause.
      if (entity != null) {
        try {
          return entity.toModel();
        } catch (e, st) {
          debugPrint(
            'TrailNotifier: cached trail "$id" failed to parse, falling '
            'through to the original error: $e\n$st',
          );
        }
      }
      rethrow;
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
