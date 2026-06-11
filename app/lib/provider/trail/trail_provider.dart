import 'package:gpx/gpx.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

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

        final parsedGpx = GpxReader().fromString(gpxResponse.data);

        trail = trail.copyWith(
          expand: (trail.expand ?? const TrailExpand()).copyWith(
            gpx: parsedGpx,
            gpxData: gpxResponse.data,
          ),
        );
      }

      return trail;
    } catch (_) {
      final store = ref.read(objectBoxProvider);
      final box = store.box<TrailEntity>();
      final query = box.query(TrailEntity_.id.equals(id)).build();
      final entity = query.findFirst();
      query.close();

      if (entity != null) return entity.toModel();
      rethrow;
    }
  }
}
