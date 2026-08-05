import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'profile_trail_bounding_box_provider.g.dart';

const _worldViewBoundingBox = TrailBoundingBox(
  maxLat: 0,
  minLat: 0,
  maxLon: 0,
  minLon: 0,
  hasTrails: false,
);

/// Per-profile-handle bounding box, backing the initial camera fit for
/// `ProfileTrailMapScreen` (D-01).
///
/// This is autoDispose, deliberately not keepAlive: a stale bbox for a
/// profile whose trails changed would frame the map wrongly, and one extra
/// request per screen open is cheap relative to the search requests that
/// follow.
///
/// Never surfaces an error to the UI. This is the world-view fallback path
/// D-05 makes first-class, not an edge case: on a mixed-version federation
/// the remote will commonly not support the `handle` param, and the correct
/// behaviour is a silent degrade to the default camera with the bounds
/// search still running. Any `DioException`, timeout, or parse failure
/// resolves to a zeroed [TrailBoundingBox] with `hasTrails: false` rather
/// than propagating an exception.
@riverpod
Future<TrailBoundingBox> profileTrailBoundingBox(Ref ref, String handle) async {
  final api = ref.read(apiProvider);

  try {
    final response = await api.get(
      '/trail/bounding-box',
      queryParameters: {'handle': handle},
      options: Options(receiveTimeout: const Duration(seconds: 6)),
    );

    return TrailBoundingBox.fromJson(response.data as Map<String, dynamic>);
  } catch (e) {
    return _worldViewBoundingBox;
  }
}
