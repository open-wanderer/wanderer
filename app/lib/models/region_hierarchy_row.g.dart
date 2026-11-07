// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'region_hierarchy_row.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_RegionHierarchyRow _$RegionHierarchyRowFromJson(Map<String, dynamic> json) =>
    _RegionHierarchyRow(
      id: json['id'] as String,
      name: json['name'] as String,
      kind: $enumDecode(_$RegionNodeKindEnumMap, json['kind']),
      parent: json['parent'] as String,
      path: json['path'] as String,
      depth: (json['depth'] as num).toInt(),
      sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
    );

Map<String, dynamic> _$RegionHierarchyRowToJson(_RegionHierarchyRow instance) =>
    <String, dynamic>{
      'id': instance.id,
      'name': instance.name,
      'kind': _$RegionNodeKindEnumMap[instance.kind]!,
      'parent': instance.parent,
      'path': instance.path,
      'depth': instance.depth,
      'sort_order': instance.sortOrder,
    };

const _$RegionNodeKindEnumMap = {
  RegionNodeKind.group: 'group',
  RegionNodeKind.leaf: 'leaf',
};
