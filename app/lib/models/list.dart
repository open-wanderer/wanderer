import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/list_summary.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail.dart';

import 'actor.dart';

part 'list.freezed.dart';
part 'list.g.dart';

enum ListSort {
  @JsonValue('name')
  name,
  @JsonValue('size')
  size,
  @JsonValue('created')
  created,
}

@freezed
abstract class ListExpand with _$ListExpand {
  const factory ListExpand({List<Trail>? trails, Actor? author}) = _ListExpand;

  factory ListExpand.fromJson(Map<String, dynamic> json) =>
      _$ListExpandFromJson(json);
}

@freezed
abstract class WandererList
    with _$WandererList, RecordFunctions
    implements ListSummary {
  const factory WandererList({
    required String id,
    @Default('lists') String collectionId,
    required String name,
    @Default(false) bool public,
    String? description,
    @JsonKey(name: 'elevation_gain') double? elevationGain,
    @JsonKey(name: 'elevation_loss') double? elevationLoss,
    double? distance,
    double? duration,
    String? avatar,
    @Default([]) List<String> trails,
    String? iri,
    ListExpand? expand,
    DateTime? created,
    DateTime? updated,
    @Default('000000000000000') String author,
  }) = _WandererList;

  const WandererList._();

  @override
  int get trailCount => expand?.trails?.length ?? 0;

  factory WandererList.fromJson(Map<String, dynamic> json) =>
      _$WandererListFromJson(json);
}

@freezed
abstract class ListFilter with _$ListFilter {
  const factory ListFilter({
    required String q,
    ListSort? sort,
    String? author,
    bool? public,
    bool? shared,
    String? sortOrder,
  }) = _ListFilter;
}
