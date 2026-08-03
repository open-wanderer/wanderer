import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/list_summary.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail.dart';

import 'actor.dart';

part 'list.freezed.dart';
part 'list.g.dart';

enum ListFilterSort {
  @JsonValue('name')
  name,
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

  @override
  double? get elevationGain =>
      expand?.trails?.fold<double>(0.0, (s, t) => s + t.elevationGain);

  @override
  double? get elevationLoss =>
      expand?.trails?.fold<double>(0.0, (s, t) => s + t.elevationLoss);

  @override
  double? get distance =>
      expand?.trails?.fold<double>(0.0, (s, t) => s + t.distance);

  @override
  double? get duration =>
      expand?.trails?.fold<double>(0.0, (s, t) => s + t.duration);

  @override
  String get summaryAuthorName => expand?.author?.username ?? "Unknown";

  @override
  String get summaryAuthorAvatar => expand?.author?.icon ?? "";

  @override
  String? get summaryAuthorActorId =>
      expand?.author?.id ??
      (author.isNotEmpty && author != "000000000000000" ? author : null);

  factory WandererList.fromJson(Map<String, dynamic> json) =>
      _$WandererListFromJson(json);
}

@freezed
abstract class ListFilter with _$ListFilter {
  const factory ListFilter({
    required String q,
    String? author,
    bool? public,
    bool? shared,
    required ListFilterSort sort,
    required SortOrder sortOrder,
  }) = _ListFilter;

  // ignore: unused_element
  const ListFilter._();

  String toFilterText({String? actorId}) {
    String filterText = '';

    if (author != null && author!.isNotEmpty) {
      filterText += 'author = $author';
    }

    if (public != null || shared != null) {
      if (filterText.isNotEmpty) {
        filterText += ' AND ';
      }
      filterText += '(';
      if (public != null) {
        filterText += '(public = $public';
        if (author == null || author!.isEmpty || author == actorId) {
          filterText += ' OR author = $actorId';
        }
        filterText += ')';
      }
      if (shared != null) {
        if (shared == true) {
          filterText += ' OR shares = $actorId';
        } else {
          filterText += ' AND NOT shares = $actorId';
        }
      }
      filterText += ')';
    }

    return filterText;
  }
}
