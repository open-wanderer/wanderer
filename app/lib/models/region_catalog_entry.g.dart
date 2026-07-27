// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'region_catalog_entry.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_RegionCatalogEntry _$RegionCatalogEntryFromJson(Map<String, dynamic> json) =>
    _RegionCatalogEntry(
      id: json['id'] as String,
      path: json['path'] as String,
      name: json['name'] as String,
      bbox: (json['bbox'] as List<dynamic>)
          .map((e) => (e as num).toDouble())
          .toList(),
      status: $enumDecode(_$CatalogStatusEnumMap, json['status']),
      version: json['version'] as String?,
      vectorUrl: json['vector_url'] as String?,
      vectorSize: (json['vector_size'] as num?)?.toInt(),
      demStatus: $enumDecodeNullable(
        _$CatalogStatusEnumMap,
        json['dem_status'],
      ),
      demUrl: json['dem_url'] as String?,
      demSize: (json['dem_size'] as num?)?.toInt(),
      error: json['error'] as String?,
    );

Map<String, dynamic> _$RegionCatalogEntryToJson(_RegionCatalogEntry instance) =>
    <String, dynamic>{
      'id': instance.id,
      'path': instance.path,
      'name': instance.name,
      'bbox': instance.bbox,
      'status': _$CatalogStatusEnumMap[instance.status]!,
      'version': instance.version,
      'vector_url': instance.vectorUrl,
      'vector_size': instance.vectorSize,
      'dem_status': _$CatalogStatusEnumMap[instance.demStatus],
      'dem_url': instance.demUrl,
      'dem_size': instance.demSize,
      'error': instance.error,
    };

const _$CatalogStatusEnumMap = {
  CatalogStatus.building: 'building',
  CatalogStatus.ready: 'ready',
  CatalogStatus.error: 'error',
  CatalogStatus.absent: 'absent',
};
