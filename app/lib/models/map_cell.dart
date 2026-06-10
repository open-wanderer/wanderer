import 'package:freezed_annotation/freezed_annotation.dart';

part 'map_cell.freezed.dart';
part 'map_cell.g.dart';

enum MapCellStatus {
  @JsonValue('new')
  isNew,
  @JsonValue('pending')
  pending,
  @JsonValue('generating')
  generating,
  @JsonValue('ready')
  ready,
  @JsonValue('error')
  error,
}

@freezed
abstract class MapCellInfoList with _$MapCellInfoList {
  const factory MapCellInfoList({required List<MapCellInfo> cells}) =
      _MapCellInfoList;

  factory MapCellInfoList.fromJson(Map<String, dynamic> json) =>
      _$MapCellInfoListFromJson(json);
}

@freezed
abstract class MapCellInfo with _$MapCellInfo {
  const factory MapCellInfo({
    required String key,
    required MapCellStatus status,
    required String url,
    @JsonKey(name: 'size_bytes') int? sizeBytes,
  }) = _MapCellInfo;

  factory MapCellInfo.fromJson(Map<String, dynamic> json) =>
      _$MapCellInfoFromJson(json);
}

@freezed
abstract class MapCellStatusResponse with _$MapCellStatusResponse {
  const factory MapCellStatusResponse({
    required MapCellStatus status,
    @JsonKey(name: 'download_url') String? downloadUrl,
    @JsonKey(name: 'status_url') String? statusUrl,
    @JsonKey(name: 'size_bytes') int? sizeBytes,
    String? error,
  }) = _MapCellStatusResponse;

  factory MapCellStatusResponse.fromJson(Map<String, dynamic> json) =>
      _$MapCellStatusResponseFromJson(json);
}
