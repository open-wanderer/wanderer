import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_hierarchy_row.dart';
import 'package:wanderer/models/region_status.dart';
import 'package:wanderer/provider/region/region_provider.dart';

Map<String, dynamic> _fullReadyJson({String id = 'de-nrw', String? path}) => {
  'id': id,
  'path': path ?? id,
  'name': 'North Rhine-Westphalia',
  'bbox': [5.9, 50.3, 9.5, 52.5],
  'status': 'ready',
  'version': '2026-07-01',
  'vector_url': '/api/v1/regions/${path ?? id}/download',
  'vector_size': 123456,
  'dem_status': 'ready',
  'dem_url': '/api/v1/regions/${path ?? id}/download-dem',
  'dem_size': 654321,
};

Map<String, dynamic> _minimalBuildingJson({
  String id = 'de-bay',
  String? path,
}) => {
  'id': id,
  'path': path ?? id,
  'name': 'Bavaria',
  'bbox': [8.9, 47.2, 13.9, 50.6],
  'status': 'building',
};

void main() {
  group('parseRegionCatalog', () {
    test('a full-ready + minimal-building fixture yields 2 entries', () {
      final entries = parseRegionCatalog([
        _fullReadyJson(),
        _minimalBuildingJson(),
      ]);

      expect(entries, hasLength(2));
      expect(entries[0].id, 'de-nrw');
      expect(entries[0].status, CatalogStatus.ready);
      expect(entries[1].id, 'de-bay');
      expect(entries[1].status, CatalogStatus.building);
    });

    test('a valid + malformed element drops the malformed one', () {
      final entries = parseRegionCatalog([
        _fullReadyJson(),
        {'id': 'de-bad'}, // missing required name/bbox/status
      ]);

      expect(entries, hasLength(1));
      expect(entries.single.id, 'de-nrw');
    });

    test('an empty array yields an empty list', () {
      expect(parseRegionCatalog([]), isEmpty);
    });

    test('a non-List payload throws RegionCatalogException', () {
      expect(
        () => parseRegionCatalog({'items': []}),
        throwsA(isA<RegionCatalogException>()),
      );
    });
  });

  group('parseRegionHierarchyRows', () {
    Map<String, dynamic> groupJson({String id = 'ca-ab'}) => {
      'id': id,
      'name': 'Alberta',
      'kind': 'group',
      'parent': '',
      'path': id,
      'depth': 0,
      'sort_order': 1,
    };

    Map<String, dynamic> leafJson({String id = 'ca-ab-banff'}) => {
      'id': id,
      'name': 'Banff',
      'kind': 'leaf',
      'parent': 'ca-ab',
      'path': id,
      'depth': 1,
      'sort_order': 0,
    };

    test(
      'a group-kind fixture (no bbox/status) parses successfully -- the row the leaf parser drops',
      () {
        final rows = parseRegionHierarchyRows([groupJson()]);

        expect(rows, hasLength(1));
        expect(rows.single.kind, RegionNodeKind.group);
        expect(rows.single.id, 'ca-ab');
      },
    );

    test('a leaf-kind fixture also parses', () {
      final rows = parseRegionHierarchyRows([leafJson()]);

      expect(rows, hasLength(1));
      expect(rows.single.kind, RegionNodeKind.leaf);
      expect(rows.single.parent, 'ca-ab');
    });

    test('a malformed element (missing required id) is skipped, not fatal', () {
      final rows = parseRegionHierarchyRows([
        groupJson(),
        {'name': 'no id here'},
      ]);

      expect(rows, hasLength(1));
      expect(rows.single.id, 'ca-ab');
    });

    test(
      'sort_order maps from the sort_order JSON key, defaulting to 0 when absent (Pitfall 4)',
      () {
        final withSortOrder = parseRegionHierarchyRows([groupJson()]);
        final withoutSortOrder = parseRegionHierarchyRows([
          {
            'id': 'ca-bc',
            'name': 'British Columbia',
            'kind': 'group',
            'parent': '',
            'path': 'ca-bc',
            'depth': 0,
          },
        ]);

        expect(withSortOrder.single.sortOrder, 1);
        expect(withoutSortOrder.single.sortOrder, 0);
      },
    );

    test('a non-List payload throws RegionCatalogException', () {
      expect(
        () => parseRegionHierarchyRows({'items': []}),
        throwsA(isA<RegionCatalogException>()),
      );
    });

    // The repository's `refreshCatalogAndFetchHierarchy` round trip is not
    // unit-tested here: its constructor requires an ObjectBox Store, and this
    // suite has no established pattern for opening one in a plain
    // `flutter test` (the only Store-using files under test/ are on-device
    // spike harnesses, not _test.dart suites). Its behavior is exercised by
    // proxy through the pure parse functions it delegates to
    // (parseRegionCatalog/parseRegionHierarchyRows, both covered above).
  });

  group('orphanedRegionPaths', () {
    // `id` is deliberately unrelated to `path` here: the backend re-mints
    // record ids, so orphan detection must key on `path` alone.
    RegionEntity entity(String path) =>
        RegionEntity(path: path, id: 'rec-$path', name: path);

    test('returns persisted paths absent from the fetched set', () {
      final result = orphanedRegionPaths(
        {'a', 'b'},
        [entity('a'), entity('c')],
      );

      expect(result, {'c'});
    });

    test('returns an empty set when every persisted path is still fetched', () {
      final result = orphanedRegionPaths(
        {'a', 'b', 'c'},
        [entity('a'), entity('b'), entity('c')],
      );

      expect(result, isEmpty);
    });

    test('returns an empty set when there are no persisted entities', () {
      expect(orphanedRegionPaths({'a'}, []), isEmpty);
    });

    test('a re-minted record id does not make a region look orphaned', () {
      // Same path, different id than the persisted row -- the pre-fix
      // id-keyed comparison reported this as orphaned and flipped the row
      // (and its downloaded archive) out of the catalog.
      final persisted = RegionEntity(
        path: 'canada.canada_alberta.canada_alberta_south',
        id: '1ani4n8myc8rh2m',
        name: 'South',
      );

      expect(
        orphanedRegionPaths(
          {'canada.canada_alberta.canada_alberta_south'},
          [persisted],
        ),
        isEmpty,
      );
    });
  });
}
