import 'package:freezed_annotation/freezed_annotation.dart';

part 'region_hierarchy_row.freezed.dart';
part 'region_hierarchy_row.g.dart';

/// Discriminates a [RegionHierarchyRow] as a hierarchy-only group node or a
/// downloadable leaf node. Mirrors `db/routes/regions_get.go`'s `kind`
/// string field (`"group"` / `"leaf"`) verbatim.
enum RegionNodeKind {
  @JsonValue('group')
  group,
  @JsonValue('leaf')
  leaf,
}

/// An always-succeeding parse of a single `GET /api/v1/regions` row's
/// hierarchy fields — deliberately distinct from
/// [RegionCatalogEntry], which requires `bbox`/`status` and therefore
/// silently drops every `kind: 'group'` row (by design, see that parser's
/// doc comment). [RegionHierarchyRow.fromJson] never throws for a group row
/// because it never declares those fields at all.
///
/// Field presence mirrors `regions_get.go`'s base `entry` map, which every
/// row (group AND leaf) carries: `id`/`name`/`kind`/`parent`/`path`/`depth`/
/// `sort_order`. `parent` is always a non-null string — `""` for a root,
/// never `null` — matching Go's `r.GetString("parent")`.
@freezed
abstract class RegionHierarchyRow with _$RegionHierarchyRow {
  const factory RegionHierarchyRow({
    required String id,
    required String name,
    required RegionNodeKind kind,

    /// `""` for roots, never null — matches Go's `r.GetString("parent")`.
    required String parent,
    required String path,
    required int depth,

    /// Defaults to `0` (Pitfall 4): deliberate forward-compat guard so a
    /// row fetched before the backend's `sort_order` field ships (or from
    /// an out-of-order rollout) still parses instead of throwing.
    @JsonKey(name: 'sort_order') @Default(0) int sortOrder,
  }) = _RegionHierarchyRow;

  factory RegionHierarchyRow.fromJson(Map<String, dynamic> json) =>
      _$RegionHierarchyRowFromJson(json);
}
