import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_catalog_entry.dart';
import 'package:wanderer/models/region_hierarchy_row.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'region_provider.g.dart';

/// Typed error thrown by [RegionRepository.refreshCatalogAndFetchHierarchy]
/// on any network or parse failure (D-03). Callers must not silently swallow
/// a catalog fetch failure the way `subcategory_provider.dart` does — Phase
/// 24's Settings/Regions screen decides how to surface this to the user.
class RegionCatalogException implements Exception {
  const RegionCatalogException(this.message, [this.cause]);

  final String message;
  final Object? cause;

  @override
  String toString() =>
      'RegionCatalogException: $message'
      '${cause != null ? ' (cause: $cause)' : ''}';
}

/// Parses a bare `GET /api/v1/regions` JSON array (`db/routes/regions_get.go`
/// -- no `{items:[...]}` wrapper) into [RegionCatalogEntry] values.
///
/// A non-`List` payload throws [RegionCatalogException] (unexpected shape).
/// Individual malformed elements are caught and skipped rather than aborting
/// the whole parse (T-22-05) -- one bad/hostile array element cannot corrupt
/// or block the rest of the catalog.
List<RegionCatalogEntry> parseRegionCatalog(dynamic data) {
  if (data is! List) {
    throw const RegionCatalogException('unexpected catalog response shape');
  }

  final entries = <RegionCatalogEntry>[];
  for (final element in data) {
    try {
      entries.add(RegionCatalogEntry.fromJson(element as Map<String, dynamic>));
    } catch (_) {
      // Skip this malformed element; do not abort the whole parse.
    }
  }
  return entries;
}

/// Parses a bare `GET /api/v1/regions` JSON array into
/// [RegionHierarchyRow] values (D-05) — a separate path from
/// [parseRegionCatalog], not a replacement for it (Pitfall 2): the existing
/// group-row-drop behavior of [parseRegionCatalog] is by design and stays
/// untouched, since `RegionCatalogEntry`'s ObjectBox-backed pipeline has no
/// use for hierarchy-only group rows.
///
/// A non-`List` payload throws [RegionCatalogException] (unexpected
/// shape). Individual malformed elements are caught and skipped rather than
/// aborting the whole parse (T-31-01, extending the [parseRegionCatalog]/
/// T-22-05 per-element posture) -- one hostile array element cannot corrupt
/// or block the rest of the hierarchy.
List<RegionHierarchyRow> parseRegionHierarchyRows(dynamic data) {
  if (data is! List) {
    throw const RegionCatalogException('unexpected catalog response shape');
  }

  final rows = <RegionHierarchyRow>[];
  for (final element in data) {
    try {
      rows.add(RegionHierarchyRow.fromJson(element as Map<String, dynamic>));
    } catch (_) {
      // Skip this malformed element; do not abort the whole parse.
    }
  }
  return rows;
}

/// The persisted [existing] region paths that are absent from [fetchedPaths]
/// -- i.e. regions dropped from the latest catalog fetch (D-08).
///
/// Compares `path`, never the server record id: the backend re-mints that id
/// on every rebuild (see `RegionEntity.id`), which made every persisted region
/// look dropped and every re-minted one look new.
Set<String> orphanedRegionPaths(
  Iterable<String> fetchedPaths,
  Iterable<RegionEntity> existing,
) {
  final fetchedPathSet = fetchedPaths.toSet();
  return existing
      .map((e) => e.path)
      .where((path) => !fetchedPathSet.contains(path))
      .toSet();
}

/// Fetches and upserts the region catalog into ObjectBox, preserving local
/// download status and `ToOne` package links across refreshes (D-01) --
/// never the destructive `removeAll()`+`putMany()` merge used by
/// `subcategory_provider.dart`/`category_provider.dart`.
class RegionRepository {
  RegionRepository(this._api, this._store);

  final Dio _api;
  final Store _store;

  /// Upserts [entries] into ObjectBox by `path` inside a single write
  /// transaction: an existing row is merged in place via
  /// [RegionEntity.applyCatalogEntry] (preserving `obxId`, both `ToOne`
  /// targets, and `lastDownloadedVersion`); a new row is inserted via
  /// [RegionEntity.fromCatalogEntry]. After upserting, any persisted region
  /// whose path is absent from [entries] is flipped `inCatalog = false` --
  /// its row and any on-disk files are left untouched (D-08).
  ///
  /// Matching on `path` rather than the server record id is what makes a
  /// backend rebuild non-destructive. `ReconcileArchives` deletes and
  /// recreates `region_archives` rows with fresh PocketBase ids (see
  /// `RegionEntity.id`); keyed by id, every re-minted region was inserted as a
  /// brand-new row with no package links while its real row was flipped out of
  /// the catalog still holding the packages and the downloaded archives --
  /// leaving the region reading "not downloaded" while its bytes stayed in the
  /// disk-usage total. Keyed by path, the same rebuild simply updates the row
  /// in place and adopts the new id.
  void upsertCatalog(List<RegionCatalogEntry> entries) {
    _store.runInTransaction(TxMode.write, () {
      final box = _store.box<RegionEntity>();

      for (final entry in entries) {
        try {
          final query = box
              .query(RegionEntity_.path.equals(entry.path))
              .build();
          final existing = query.findFirst();
          query.close();

          if (existing != null) {
            existing.applyCatalogEntry(entry);
            box.put(existing);
          } else {
            box.put(RegionEntity.fromCatalogEntry(entry));
          }
        } on FormatException {
          // Malformed entry (e.g. bad bbox) -- skip it, never fatal.
        }
      }

      final fetchedPaths = entries.map((e) => e.path);
      final orphans = orphanedRegionPaths(fetchedPaths, box.getAll());
      if (orphans.isEmpty) return;

      for (final path in orphans) {
        final query = box.query(RegionEntity_.path.equals(path)).build();
        final entity = query.findFirst();
        query.close();

        if (entity != null) {
          entity.inCatalog = false;
          box.put(entity);
        }
      }
    });
  }

  /// The sole catalog-refresh entry point: GETs `/regions` ONCE, upserts the
  /// catalog into ObjectBox, and returns the parsed group+leaf hierarchy --
  /// both derived from the same response, so the Settings screen no longer
  /// pays for two round trips of the identical payload per open.
  ///
  /// A network failure is wrapped as [RegionCatalogException]; a malformed
  /// (non-`List`) shape throws it out of [parseRegionCatalog] BEFORE any
  /// store write, so a bad response always leaves every persisted row
  /// untouched (D-03).
  Future<List<RegionHierarchyRow>> refreshCatalogAndFetchHierarchy() async {
    final dynamic data;
    try {
      // `enabled=true` asks the backend to prune the catalog to only enabled
      // leaves plus their ancestor groups, so the whole disabled-region catalog
      // never crosses the wire (the client no longer prunes locally).
      data = (await _api.get(
        '/regions',
        queryParameters: {'enabled': 'true'},
      )).data;
    } catch (e) {
      throw RegionCatalogException('failed to fetch region catalog', e);
    }
    upsertCatalog(parseRegionCatalog(data));
    return parseRegionHierarchyRows(data);
  }
}

/// Construction-only provider seam (D-02) -- builds a [RegionRepository]
/// from the existing [apiProvider]/[objectBoxProvider] without performing
/// any fetch on build; the Settings/Regions screen drives
/// [RegionRepository.refreshCatalogAndFetchHierarchy] on open.
@Riverpod(keepAlive: true)
RegionRepository regionRepository(Ref ref) {
  return RegionRepository(ref.watch(apiProvider), ref.watch(objectBoxProvider));
}

/// Synchronous ObjectBox snapshot of every persisted [RegionEntity], sorted
/// alphabetically by [RegionEntity.name] (D-09) -- mirrors
/// `trail_library_provider.dart`'s `TrailLibraryNotifier` structural
/// precedent verbatim. No mutation methods live here; all region mutations
/// flow through `TileRepositoryStatus` (`tile_repository_provider.dart`),
/// which invalidates this provider itself at every terminal point of a
/// download/cancel/delete (RESEARCH.md Pitfall 2 -- ObjectBox `ToOne.target`
/// caches per-instance after first read). That invalidation deliberately
/// lives in the keepAlive notifier, NOT in the calling widget: a widget-side
/// `mounted` guard skipped it whenever the user left the Settings/Regions
/// screen mid-download.
@Riverpod(name: 'regionListNotifierProvider')
class RegionListNotifier extends _$RegionListNotifier {
  @override
  List<RegionEntity> build() {
    final store = ref.watch(objectBoxProvider);
    final regions = store.box<RegionEntity>().getAll();
    regions.sort(
      (a, b) => a.name.toLowerCase().compareTo(b.name.toLowerCase()),
    );
    return regions;
  }
}
