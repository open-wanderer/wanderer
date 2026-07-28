import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/region_hierarchy_row.dart';
import 'package:wanderer/util/region_tree_util.dart';

// ---------------------------------------------------------------------------
// Tests for region_tree_util's buildRegionTree/computeDefaultExpanded/
// flattenVisible/computeFilterMatches — pure functions, no ObjectBox/network
// needed (mirrors region_disk_usage_util_test.dart's precedent for testing
// pure utils directly, per 31-RESEARCH.md Pitfall 5 -- this screen has zero
// widget-test baseline, so these pure-function tests are the cheapest
// highest-value coverage). Ported 1:1 from regions_ui.html:515-615
// (D-01/D-02/D-06); Pitfall 3 covers the ""-root gotcha, Pitfall 4 the
// sort_order ordering gotcha.
// ---------------------------------------------------------------------------

void main() {
  RegionHierarchyRow buildRow({
    required String id,
    String parent = '',
    RegionNodeKind kind = RegionNodeKind.leaf,
    int sortOrder = 0,
    String? name,
  }) => RegionHierarchyRow(
    id: id,
    name: name ?? id,
    kind: kind,
    parent: parent,
    path: id,
    depth: 0,
    sortOrder: sortOrder,
  );

  group('buildRegionTree', () {
    test('a row with parent == "" becomes a root', () {
      final roots = buildRegionTree([buildRow(id: 'a')]);

      expect(roots, hasLength(1));
      expect(roots.single.id, 'a');
    });

    test('a row whose parent matches a known id is attached as a child', () {
      final roots = buildRegionTree([
        buildRow(id: 'group-a', kind: RegionNodeKind.group),
        buildRow(id: 'leaf-a', parent: 'group-a'),
      ]);

      expect(roots, hasLength(1));
      expect(roots.single.children, hasLength(1));
      expect(roots.single.children.single.id, 'leaf-a');
    });

    test(
      'a row whose parentId is non-empty but absent from the map falls back to a root',
      () {
        final roots = buildRegionTree([buildRow(id: 'orphan', parent: 'missing-parent')]);

        expect(roots, hasLength(1));
        expect(roots.single.id, 'orphan');
      },
    );

    test(
      'roots are sorted ascending by sortOrder even when supplied out of order (Pitfall 4)',
      () {
        final roots = buildRegionTree([
          buildRow(id: 'c', sortOrder: 2),
          buildRow(id: 'a', sortOrder: 0),
          buildRow(id: 'b', sortOrder: 1),
        ]);

        expect(roots.map((n) => n.id), ['a', 'b', 'c']);
      },
    );

    test(
      'sibling leaves under a group are sorted ascending by sortOrder regardless of input order (Pitfall 4)',
      () {
        final roots = buildRegionTree([
          buildRow(id: 'group-a', kind: RegionNodeKind.group),
          buildRow(id: 'leaf-2', parent: 'group-a', sortOrder: 2),
          buildRow(id: 'leaf-0', parent: 'group-a', sortOrder: 0),
          buildRow(id: 'leaf-1', parent: 'group-a', sortOrder: 1),
        ]);

        expect(
          roots.single.children.map((n) => n.id),
          ['leaf-0', 'leaf-1', 'leaf-2'],
        );
      },
    );
  });

  group('computeDefaultExpanded', () {
    test('a leaf returns the predicate result directly', () {
      final roots = buildRegionTree([buildRow(id: 'leaf-a')]);

      final expandedTrue = computeDefaultExpanded(roots, (id) => true);
      final expandedFalse = computeDefaultExpanded(roots, (id) => false);

      // Leaves are never added to the expanded set themselves (only groups
      // are); this just confirms the predicate result doesn't spuriously
      // mark anything expanded for a leaf-only tree.
      expect(expandedTrue, isEmpty);
      expect(expandedFalse, isEmpty);
    });

    test(
      'a group is added to the expanded set iff a descendant leaf satisfies the predicate',
      () {
        final roots = buildRegionTree([
          buildRow(id: 'group-a', kind: RegionNodeKind.group),
          buildRow(id: 'leaf-a', parent: 'group-a'),
          buildRow(id: 'group-b', kind: RegionNodeKind.group),
          buildRow(id: 'leaf-b', parent: 'group-b'),
        ]);

        final expanded = computeDefaultExpanded(roots, (id) => id == 'leaf-a');

        expect(expanded, {'group-a'});
      },
    );

    test('nested groups: a deeply nested matching leaf expands every ancestor group', () {
      final roots = buildRegionTree([
        buildRow(id: 'top', kind: RegionNodeKind.group),
        buildRow(id: 'mid', parent: 'top', kind: RegionNodeKind.group),
        buildRow(id: 'leaf-deep', parent: 'mid'),
      ]);

      final expanded = computeDefaultExpanded(roots, (id) => id == 'leaf-deep');

      expect(expanded, {'top', 'mid'});
    });
  });

  group('flattenVisible', () {
    test('a collapsed group is included but its children are skipped', () {
      final roots = buildRegionTree([
        buildRow(id: 'group-a', kind: RegionNodeKind.group),
        buildRow(id: 'leaf-a', parent: 'group-a'),
      ]);

      final visible = flattenVisible(roots, <String>{}, null);

      expect(visible.map((r) => r.node.id), ['group-a']);
    });

    test('an expanded group includes its children at depth + 1', () {
      final roots = buildRegionTree([
        buildRow(id: 'group-a', kind: RegionNodeKind.group),
        buildRow(id: 'leaf-a', parent: 'group-a'),
      ]);

      final visible = flattenVisible(roots, {'group-a'}, null);

      expect(visible.map((r) => r.node.id), ['group-a', 'leaf-a']);
      expect(visible.firstWhere((r) => r.node.id == 'group-a').depth, 0);
      expect(visible.firstWhere((r) => r.node.id == 'leaf-a').depth, 1);
    });

    test('a filtered-out node (absent from filterMatches) is skipped', () {
      final roots = buildRegionTree([
        buildRow(id: 'group-a', kind: RegionNodeKind.group),
        buildRow(id: 'leaf-a', parent: 'group-a'),
        buildRow(id: 'leaf-b', parent: 'group-a'),
      ]);

      final visible = flattenVisible(roots, <String>{}, {'group-a', 'leaf-a'});

      expect(visible.map((r) => r.node.id), ['group-a', 'leaf-a']);
    });

    test('a group descends when filterMatches is non-null, even if not in expanded', () {
      final roots = buildRegionTree([
        buildRow(id: 'group-a', kind: RegionNodeKind.group),
        buildRow(id: 'leaf-a', parent: 'group-a'),
      ]);

      final visible = flattenVisible(roots, <String>{}, {'group-a', 'leaf-a'});

      expect(visible.map((r) => r.node.id), ['group-a', 'leaf-a']);
    });
  });

  group('computeFilterMatches', () {
    test('an empty query returns null', () {
      final roots = buildRegionTree([buildRow(id: 'leaf-a')]);

      expect(computeFilterMatches(roots, ''), isNull);
    });

    test('a self-name match pulls the whole subtree plus the ancestor chain', () {
      final roots = buildRegionTree([
        buildRow(id: 'top', kind: RegionNodeKind.group, name: 'Alberta'),
        buildRow(id: 'mid', parent: 'top', kind: RegionNodeKind.group, name: 'Rockies'),
        buildRow(id: 'leaf-deep', parent: 'mid', name: 'Banff'),
      ]);

      final matches = computeFilterMatches(roots, 'alberta');

      expect(matches, {'top', 'mid', 'leaf-deep'});
    });

    test(
      'a descendant-only match pulls the node and ancestor chain but NOT the whole subtree',
      () {
        final roots = buildRegionTree([
          buildRow(id: 'top', kind: RegionNodeKind.group, name: 'Alberta'),
          buildRow(id: 'mid', parent: 'top', kind: RegionNodeKind.group, name: 'Rockies'),
          buildRow(id: 'leaf-match', parent: 'mid', name: 'Banff'),
          buildRow(id: 'leaf-nomatch', parent: 'mid', name: 'Jasper'),
        ]);

        final matches = computeFilterMatches(roots, 'banff');

        expect(matches, {'top', 'mid', 'leaf-match'});
        expect(matches, isNot(contains('leaf-nomatch')));
      },
    );
  });

}
