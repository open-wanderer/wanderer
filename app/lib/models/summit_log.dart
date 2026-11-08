import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail.dart';

part 'summit_log.freezed.dart';
part 'summit_log.g.dart';

@freezed
abstract class SummitLog with _$SummitLog, RecordFunctions implements IRecord {
  const factory SummitLog({
    required String id,
    required String date,
    required String collectionId,
    required String collectionName,

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
    required DateTime created,
    required DateTime updated,
    SummitLogExpand? expand,
  }) = _SummitLog;

  const SummitLog._();

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
