import 'package:gpx/gpx.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'trail_provider.g.dart';

@riverpod
class TrailNotifier extends _$TrailNotifier {
  @override
  FutureOr<Trail> build(String id) async {
    final api = ref.watch(apiProvider);

    final response = await api.get(
      "/trail/$id",
      queryParameters: {
        "expand":
            "category,waypoints_via_trail,summit_logs_via_trail,summit_logs_via_trail.author,trail_share_via_trail.actor,trail_like_via_trail,tags,author",
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
        expand: (trail.expand ?? const TrailExpand()).copyWith(gpx: parsedGpx),
      );
    }

    return trail;
  }
}
