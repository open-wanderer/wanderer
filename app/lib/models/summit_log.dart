import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/trail.dart';

part 'summit_log.freezed.dart';
part 'summit_log.g.dart';

@freezed
abstract class SummitLog with _$SummitLog {
  const factory SummitLog({
    String? id,
    required String date,
    @Default("") String text,
    String? gpx,
    @Default([]) List<String> photos,
    double? distance,
    @JsonKey(name: 'elevation_gain') double? elevationGain,
    @JsonKey(name: 'elevation_loss') double? elevationLoss,
    double? duration,
    @Default("000000000000000") String author,
    String? trail,
    String? iri,
    String? created,
    SummitLogExpand? expand,
  }) = _SummitLog;

  factory SummitLog.fromJson(Map<String, dynamic> json) =>
      _$SummitLogFromJson(json);
}

@freezed
abstract class SummitLogExpand with _$SummitLogExpand {
  const factory SummitLogExpand({
    @JsonKey(name: 'gpx_data') String? gpxData,
    Trail? trail,
    Actor? author,
  }) = _SummitLogExpand;

  factory SummitLogExpand.fromJson(Map<String, dynamic> json) =>
      _$SummitLogExpandFromJson(json);
}
