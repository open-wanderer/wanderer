import 'package:wanderer/models/region_hierarchy_row.dart';
import 'package:wanderer/models/region_tree_node.dart';

/// Assembles a flat `List<RegionHierarchyRow>` into a group/leaf tree
/// (D-01), ported 1:1 from `regions_ui.html:515-533`'s `buildTree`.
///
/// A row whose `parent` is `""` becomes a root. A row whose `parentId` is
/// non-empty but whose parent id is absent from the map ALSO falls back to
/// a root (mirrors the JS `byId.has(node.parent)` guard). Every node's
/// `children` list, and the `roots` list itself, are sorted ascending by
/// `sortOrder` (Pitfall 4 — sibling ordering must not depend on input
/// array order).
List<RegionTreeNode> buildRegionTree(List<RegionHierarchyRow> rows) {
  final byId = <String, RegionTreeNode>{
    for (final row in rows)
      row.id: RegionTreeNode(
        id: row.id,
        name: row.name,
        kind: row.kind,
        parentId: row.parent,
        path: row.path,
        depth: row.depth,
        sortOrder: row.sortOrder,
        enabled: row.enabled,
      ),
  };

  final roots = <RegionTreeNode>[];
  for (final node in byId.values) {
    // Pitfall 3: parent is always a non-null string ("" for roots), so the
    // JS `if (node.parent && byId.has(node.parent))` truthiness check ports
    // to `.isNotEmpty`, NOT `!= null`.
    final parent = node.parentId.isNotEmpty ? byId[node.parentId] : null;
    if (parent != null) {
      parent.children.add(node);
    } else {
      roots.add(node);
    }
  }

  for (final node in byId.values) {
    node.children.sort((a, b) => a.sortOrder.compareTo(b.sortOrder));
  }
  roots.sort((a, b) => a.sortOrder.compareTo(b.sortOrder));

  return roots;
}

/// Prunes [roots] down to only downloadable leaves plus the ancestor group
/// chain needed to reach them (31-UAT Test 1 / APPUI-01 gap closure).
///
/// A leaf survives iff `node.enabled == true` — a `false` or `null` (group
/// row shape / missing flag / legacy row) leaf is dropped. A group survives
/// iff it still has at least one surviving child after its own descendants
/// are pruned (post-order); a group left with zero children is dropped
/// entirely, including any group whose only leaves were all disabled.
///
/// Per the resolved product decision, pruning inspects `enabled` ONLY —
/// never any build/status field, so an `enabled == true` leaf that is
/// still `status == "building"` continues to survive and render (matching
/// the screen's existing disabled-row treatment for a still-building
/// region).
///
/// Mutates each surviving group's `children` list in place via
/// `retainWhere`. This is safe because `RegionTreeNode` is an ephemeral
/// tree rebuilt from scratch on every fetch (D-03) — pruning never touches
/// a cached or shared structure, only the freshly-built graph about to be
/// rendered once.
List<RegionTreeNode> pruneToDownloadable(List<RegionTreeNode> roots) {
  bool keep(RegionTreeNode node) {
    if (node.kind == RegionNodeKind.leaf) {
      return node.enabled == true;
    }
    node.children.retainWhere(keep);
    return node.children.isNotEmpty;
  }

  return roots.where(keep).toList();
}

/// Given a predicate over leaf ids, returns the set of group node ids that
/// should be expanded by default (D-02) — a group is expanded iff ANY
/// descendant leaf satisfies [hasDownloadOrInProgress]. Ported 1:1 from
/// `regions_ui.html:599-609`'s `computeDefaultExpanded`/
/// `hasEnabledDescendant`, generalized from a hardcoded `node.enabled` check
/// to an injected predicate so it can be recomputed against a fresh
/// `RegionEntity` snapshot without rebuilding tree shape.
Set<String> computeDefaultExpanded(
  List<RegionTreeNode> roots,
  bool Function(String leafId) hasDownloadOrInProgress,
) {
  final expanded = <String>{};

  bool walk(RegionTreeNode node) {
    if (node.kind == RegionNodeKind.leaf) {
      return hasDownloadOrInProgress(node.id);
    }
    final any = node.children.map(walk).toList().any((v) => v);
    if (any) expanded.add(node.id);
    return any;
  }

  for (final root in roots) {
    walk(root);
  }

  return expanded;
}

/// Flattens [roots] into a depth-annotated, render-ready list (D-06),
/// ported 1:1 from `regions_ui.html:611-633`'s `flattenVisible`. Returns a
/// record type (`({RegionTreeNode node, int depth})`, matching
/// `tile_repository_manager.dart`'s `splitRegionTilePaths` precedent)
/// rather than mutating `node.depth` in place, so the canonical tree node
/// is never touched by a render pass.
///
/// A node absent from a non-null [filterMatches] is skipped entirely. A
/// group descends into its children when [filterMatches] is non-null
/// (matched groups are treated as expanded while filtering) OR the group's
/// id is present in [expanded].
List<({RegionTreeNode node, int depth})> flattenVisible(
  List<RegionTreeNode> roots,
  Set<String> expanded,
  Set<String>? filterMatches,
) {
  final out = <({RegionTreeNode node, int depth})>[];

  void walk(List<RegionTreeNode> nodes, int depth) {
    for (final node in nodes) {
      if (filterMatches != null && !filterMatches.contains(node.id)) continue;
      out.add((node: node, depth: depth));
      if (node.kind == RegionNodeKind.group) {
        final shouldDescend = filterMatches != null || expanded.contains(node.id);
        if (shouldDescend) walk(node.children, depth + 1);
      }
    }
  }

  walk(roots, 0);
  return out;
}

/// Returns the set of node ids that match [query] (case-insensitive
/// substring of `name`) plus every matched node's full ancestor chain, or
/// `null` for an empty query (D-06). Ported 1:1 from
/// `regions_ui.html:635-667`'s `computeFilterMatches`.
///
/// A node whose own name matches pulls in its WHOLE subtree (so searching
/// "Alberta" surfaces every leaf under it). A node that matches only via a
/// descendant stays narrow: itself, its ancestor chain, and whichever
/// descendant(s) matched — not its other non-matching children.
Set<String>? computeFilterMatches(List<RegionTreeNode> roots, String query) {
  if (query.isEmpty) return null;

  final q = query.toLowerCase();
  final matchSet = <String>{};

  void addSubtree(RegionTreeNode node) {
    matchSet.add(node.id);
    for (final child in node.children) {
      addSubtree(child);
    }
  }

  bool walk(RegionTreeNode node, List<String> ancestors) {
    final selfMatch = node.name.toLowerCase().contains(q);
    if (selfMatch) {
      addSubtree(node);
      matchSet.addAll(ancestors);
      return true;
    }

    var childMatch = false;
    for (final child in node.children) {
      if (walk(child, [...ancestors, node.id])) childMatch = true;
    }
    if (childMatch) {
      matchSet.add(node.id);
      matchSet.addAll(ancestors);
      return true;
    }
    return false;
  }

  for (final root in roots) {
    walk(root, const []);
  }

  return matchSet;
}
