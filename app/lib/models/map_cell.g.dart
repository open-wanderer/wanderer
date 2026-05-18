// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'map_cell.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_MapCellInfoList _$MapCellInfoListFromJson(Map<String, dynamic> json) =>
    _MapCellInfoList(
      cells: (json['cells'] as List<dynamic>)
          .map((e) => MapCellInfo.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$MapCellInfoListToJson(_MapCellInfoList instance) =>
    <String, dynamic>{'cells': instance.cells};

_MapCellInfo _$MapCellInfoFromJson(Map<String, dynamic> json) => _MapCellInfo(
  key: json['key'] as String,
  status: $enumDecode(_$MapCellStatusEnumMap, json['status']),
  url: json['url'] as String,
  sizeBytes: (json['size_bytes'] as num?)?.toInt(),
);

Map<String, dynamic> _$MapCellInfoToJson(_MapCellInfo instance) =>
    <String, dynamic>{
      'key': instance.key,
      'status': _$MapCellStatusEnumMap[instance.status]!,
      'url': instance.url,
      'size_bytes': instance.sizeBytes,
    };

const _$MapCellStatusEnumMap = {
  MapCellStatus.isNew: 'new',
  MapCellStatus.pending: 'pending',
  MapCellStatus.ready: 'ready',
  MapCellStatus.error: 'error',
};

_MapCellStatusResponse _$MapCellStatusResponseFromJson(
  Map<String, dynamic> json,
) => _MapCellStatusResponse(
  status: $enumDecode(_$MapCellStatusEnumMap, json['status']),
  downloadUrl: json['download_url'] as String?,
  statusUrl: json['status_url'] as String?,
  sizeBytes: (json['size_bytes'] as num?)?.toInt(),
  error: json['error'] as String?,
);

Map<String, dynamic> _$MapCellStatusResponseToJson(
  _MapCellStatusResponse instance,
) => <String, dynamic>{
  'status': _$MapCellStatusEnumMap[instance.status]!,
  'download_url': instance.downloadUrl,
  'status_url': instance.statusUrl,
  'size_bytes': instance.sizeBytes,
  'error': instance.error,
};
