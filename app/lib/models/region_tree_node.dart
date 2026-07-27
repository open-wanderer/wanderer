import 'package:wanderer/models/region_hierarchy_row.dart';

/// Ephemeral, in-memory tree node assembled by `buildRegionTree` (D-03) from
/// a flat `List<RegionHierarchyRow>`.
///
/// Deliberate exception to this codebase's Freezed-everything model
/// convention (mirrors `region_entity.dart`'s own documented deliberate
/// exceptions, e.g. its explicit `.code`-shadow properties): this tree is
/// rebuilt from scratch on every fetch, is never value-compared (only
/// looked up by [id] in a `Map`), and its [children] list is populated via
/// in-place mutation while the tree is assembled — none of which plays well
/// with Freezed's copyWith/equality machinery, so a plain mutable class is
/// the simpler, correct shape here.
class RegionTreeNode {
  RegionTreeNode({
    required this.id,
    required this.name,
    required this.kind,
    required this.parentId,
    required this.path,
    required this.depth,
    required this.sortOrder,
    required this.enabled,
  });

  final String id;
  final String name;
  final RegionNodeKind kind;
  final String parentId;
  final String path;
  final int depth;
  final int sortOrder;

  /// Leaf-only downloadable flag copied from `RegionHierarchyRow.enabled`;
  /// `null` for group rows. Consumed by `pruneToDownloadable`.
  final bool? enabled;

  /// Populated by `buildRegionTree` as it attaches each node to its parent.
  /// Growable and mutable by design — see the class doc comment.
  final List<RegionTreeNode> children = [];
}
