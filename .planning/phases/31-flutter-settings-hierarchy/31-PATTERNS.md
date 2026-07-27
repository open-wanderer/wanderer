# Phase 31: Flutter Settings Hierarchy - Pattern Map

**Mapped:** 2026-07-27
**Files analyzed:** 8 (5 new/modified Dart files, 1 modified Go file, 2 pure-logic support pieces)
**Analogs found:** 8 / 8 (this phase is a within-repo port; the "analog" for most new files is the exact JS it replaces, or a sibling Dart file of the same shape)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `app/lib/models/region_hierarchy_row.dart` (NEW) | model | transform (JSON→object, always-succeeds) | `app/lib/models/region_catalog_entry.dart` | exact (same Freezed API-response-model shape, sibling file) |
| `app/lib/models/region_tree_node.dart` (NEW) | model | transform (in-memory tree) | `db/routes/regions_ext/regions_ui.html:515-533` (`buildTree`'s row-object shape) | role-match (JS object → Dart mutable class, deliberate non-Freezed exception) |
| `app/lib/util/region_tree_util.dart` (NEW) | utility | transform (pure tree algorithm) | `db/routes/regions_ext/regions_ui.html:515-615` (`buildTree`/`computeDefaultExpanded`/`flattenVisible`/`computeFilterMatches`) | exact (1:1 port target, same repo) |
| `app/lib/provider/region/region_provider.dart` (MODIFIED — add parse/fetch fns) | service/provider | request-response + CRUD | same file's existing `parseRegionCatalog`/`fetchRegionCatalog`/`RegionRepository` | exact (add a sibling function pair in the same file, same conventions) |
| `app/lib/routes/settings_offline_regions_screen.dart` (MODIFIED) | component (screen) | request-response + event-driven (expand/collapse, filter) | same file's existing `build()`/`_buildRegionRow`/`_buildEmptyState` | exact (modifying itself; per-row render methods stay untouched) |
| `db/routes/regions_get.go` (MODIFIED — add `sort_order` field) | controller (Go HTTP handler) | request-response | same file's existing `entry := map[string]any{...}` block | exact (one-line addition to an existing handler) |
| `app/test/util/region_tree_util_test.dart` (NEW) | test | transform (pure function unit tests) | `app/test/util/region_disk_usage_util_test.dart` | role-match (pure-util test file, same `group()`/`test()` shape) |
| `app/lib/i18n/app_en.arb` (MODIFIED — new keys) | config | — | existing `regions_*` ARB keys in the same file | exact |

## Pattern Assignments

### `app/lib/models/region_hierarchy_row.dart` (model, transform)

**Analog:** `app/lib/models/region_catalog_entry.dart`

**Full existing analog** (`region_catalog_entry.dart:1-38`):
```dart
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/region_status.dart';

part 'region_catalog_entry.freezed.dart';
part 'region_catalog_entry.g.dart';

@freezed
abstract class RegionCatalogEntry with _$RegionCatalogEntry {
  const factory RegionCatalogEntry({
    required String id,
    required String name,
    required List<double> bbox,
    required CatalogStatus status,
    String? version,
    @JsonKey(name: 'vector_url') String? vectorUrl,
    @JsonKey(name: 'vector_size') int? vectorSize,
    @JsonKey(name: 'dem_status') CatalogStatus? demStatus,
    @JsonKey(name: 'dem_url') String? demUrl,
    @JsonKey(name: 'dem_size') int? demSize,
    String? error,
  }) = _RegionCatalogEntry;

  factory RegionCatalogEntry.fromJson(Map<String, dynamic> json) =>
      _$RegionCatalogEntryFromJson(json);
}
```

**Copy this shape exactly** for `RegionHierarchyRow`, but with ALL fields required (per D-05/Pattern 3 in RESEARCH.md) except `sortOrder` which defaults to `0` for forward-compat with D-07 sequencing:
```dart
part 'region_hierarchy_row.freezed.dart';
part 'region_hierarchy_row.g.dart';

enum RegionNodeKind {
  @JsonValue('group') group,
  @JsonValue('leaf') leaf,
}

@freezed
abstract class RegionHierarchyRow with _$RegionHierarchyRow {
  const factory RegionHierarchyRow({
    required String id,
    required String name,
    required RegionNodeKind kind,
    required String parent,   // "" for roots, never null (Go's r.GetString)
    required String path,
    required int depth,
    @JsonKey(name: 'sort_order') @Default(0) int sortOrder,
  }) = _RegionHierarchyRow;

  factory RegionHierarchyRow.fromJson(Map<String, dynamic> json) =>
      _$RegionHierarchyRowFromJson(json);
}
```
Notice `region_catalog_entry.dart` uses `@JsonKey(name: 'vector_url')` for snake_case→camelCase mapping — the same convention applies to `sort_order` → `sortOrder`.

---

### `app/lib/util/region_tree_util.dart` (utility, transform)

**Analog:** `db/routes/regions_ext/regions_ui.html:515-615` (JS, same repo, Phase 30, already shipped)

**buildTree pattern** (`regions_ui.html:515-533`):
```javascript
function buildTree(rows) {
  var byId = new Map();
  rows.forEach(function (r) {
    byId.set(r.id, Object.assign({}, r, { children: [] }));
  });
  var roots = [];
  byId.forEach(function (node) {
    if (node.parent && byId.has(node.parent)) {
      byId.get(node.parent).children.push(node);
    } else {
      roots.push(node);
    }
  });
  byId.forEach(function (node) {
    node.children.sort(function (a, b) { return a.sort_order - b.sort_order; });
  });
  roots.sort(function (a, b) { return a.sort_order - b.sort_order; });
  return roots;
}
```
**Dart port gotcha (Pitfall 3 in RESEARCH.md):** JS's `if (node.parent && ...)` relies on `""` being falsy. Dart has no implicit string truthiness — port as `node.parentId.isNotEmpty ? byId[node.parentId] : null`, NOT `node.parentId != null` (parent is always a non-null `""` for roots per `regions_get.go`'s `r.GetString("parent")`).

**computeDefaultExpanded pattern** (`regions_ui.html:547-557`) — port `hasEnabledDescendant`'s shape (`node.enabled` → in Flutter this becomes the "has a download or in-progress" predicate per D-02, not `enabled`):
```javascript
function computeDefaultExpanded(roots) {
  var expanded = new Set();
  function hasEnabledDescendant(node) {
    if (node.kind === 'leaf') return !!node.enabled;
    var any = node.children.some(hasEnabledDescendant);
    if (any) expanded.add(node.id);
    return any;
  }
  roots.forEach(hasEnabledDescendant);
  return expanded;
}
```

**flattenVisible pattern** (`regions_ui.html:559-581`):
```javascript
function flattenVisible(roots, expandedSet, filterMatchSet) {
  var out = [];
  (function walk(nodes, depth) {
    nodes.forEach(function (n) {
      if (filterMatchSet && !filterMatchSet.has(n.id)) return;
      n.depth = depth;
      out.push(n);
      if (n.kind === 'group') {
        var shouldDescend = filterMatchSet ? true : expandedSet.has(n.id);
        if (shouldDescend) walk(n.children, depth + 1);
      }
    });
  })(roots, 0);
  return out;
}
```
Dart return type: use a record `List<({RegionTreeNode node, int depth})>` instead of mutating `n.depth` in place (avoids mutating the canonical tree node — cleaner than the JS original). Record-return precedent already exists in this codebase: `app/lib/services/tile_repository_manager.dart:54` —
```dart
({List<String> vectorPaths, List<String> demPaths}) splitRegionTilePaths(
```
Follow that exact record-type declaration style.

**computeFilterMatches pattern** (`regions_ui.html:583-...`, continues past line 594 — same file, read further if needed): matches self-name OR any descendant, pulls in full ancestor chain and, when self matches, the whole subtree. Port 1:1 with `Set<String>` instead of JS `Set`.

---

### `app/lib/models/region_tree_node.dart` (model, transform)

**Analog:** the row-object shape built by `buildTree` above (`Object.assign({}, r, {children: []})`), NOT a Freezed model — deliberate exception (see RESEARCH.md "Alternatives Considered"). Plain mutable class:
```dart
class RegionTreeNode {
  RegionTreeNode({
    required this.id,
    required this.name,
    required this.kind,
    required this.parentId,
    required this.path,
    required this.depth,
    required this.sortOrder,
  });

  final String id;
  final String name;
  final RegionNodeKind kind;
  final String parentId;
  final String path;
  final int depth;
  final int sortOrder;
  final List<RegionTreeNode> children = [];
}
```
Document the Freezed-exception rationale inline as a comment, mirroring how `region_entity.dart` documents its own deliberate exceptions (e.g. lines 54-66's explicit `.code`-shadow comment block) — this codebase's convention is to leave a code comment explaining any deliberate deviation from the dominant pattern, not just note it in planning docs.

---

### `app/lib/provider/region/region_provider.dart` (service/provider, request-response + CRUD)

**Analog:** same file's existing `parseRegionCatalog`/`fetchRegionCatalog` (lines 34-67)

**Imports pattern** (lines 1-7):
```dart
import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_catalog_entry.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
```
Add `import 'package:wanderer/models/region_hierarchy_row.dart';`.

**Parse pattern to copy verbatim in shape** (lines 34-48) — same per-element try/catch-and-skip posture (Pitfall 2 in RESEARCH.md says do NOT touch this one; add a sibling):
```dart
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
```
New sibling `parseRegionHierarchyRows(dynamic data)` follows the identical shape but targets `RegionHierarchyRow.fromJson` — same `data is! List` guard, same `RegionCatalogException`, same per-element try/catch/skip (per RESEARCH.md's Known Threat Pattern: "extend the same posture to `parseRegionHierarchyRows`").

**Fetch pattern to copy verbatim in shape** (lines 58-67):
```dart
Future<List<RegionCatalogEntry>> fetchRegionCatalog(Dio api) async {
  try {
    final response = await api.get('/regions');
    return parseRegionCatalog(response.data);
  } on RegionCatalogException {
    rethrow;
  } catch (e) {
    throw RegionCatalogException('failed to fetch region catalog', e);
  }
}
```
New sibling `fetchRegionHierarchyRows(Dio api)` re-GETs the SAME `/regions` endpoint (per RESEARCH.md Assumption A3 — reuse `_api`, don't add a second HTTP client) and calls `parseRegionHierarchyRows`.

**Error type reuse** (lines 15-25) — reuse `RegionCatalogException` unchanged, do not add a new exception type:
```dart
class RegionCatalogException implements Exception {
  const RegionCatalogException(this.message, [this.cause]);
  final String message;
  final Object? cause;
  @override
  String toString() => 'RegionCatalogException: $message'
      '${cause != null ? ' (cause: $cause)' : ''}';
}
```

**Provider pattern** (lines 152-155) — `@Riverpod(keepAlive: true)` construction-only seam; the new `fetchRegionHierarchy` capability should be exposed as a method on the existing `RegionRepository` class (lines 86-146), not a new provider:
```dart
@Riverpod(keepAlive: true)
RegionRepository regionRepository(Ref ref) {
  return RegionRepository(ref.watch(apiProvider), ref.watch(objectBoxProvider));
}
```

---

### `app/lib/routes/settings_offline_regions_screen.dart` (component/screen, request-response + event-driven)

**Analog:** itself — the existing flat-list render pipeline is the analog for the new hierarchical one.

**State-field pattern to extend** (lines 35-47) — add tree-shape/expand fields alongside the existing `_searchQuery`/`_diskUsageRegions` fields, same style (plain `State` fields, not new providers):
```dart
class _SettingsOfflineRegionsScreenState
    extends ConsumerState<SettingsOfflineRegionsScreen> {
  String _searchQuery = '';
  List<RegionEntity>? _diskUsageRegions;
  Future<int>? _diskUsageFuture;
  Object? _freshInstallError;
  StackTrace? _freshInstallStackTrace;
  // NEW: List<RegionTreeNode>? _treeRoots; Set<String> _expandedIds = {};
  // NEW: bool _expandedSeeded = false;
```

**Fetch-and-invalidate pattern to copy for the new hierarchy fetch** (lines 64-90, `_refreshCatalog`) — same try/catch/toast-only-if-cached-data-exists shape; the hierarchy-row fetch should be added INSIDE this same method (see RESEARCH.md Pattern 2's sketch), not a new lifecycle method:
```dart
Future<void> _refreshCatalog() async {
  try {
    await ref.read(regionRepositoryProvider).refreshCatalog();
    if (!mounted) return;
    ref.invalidate(regionListNotifierProvider);
  } catch (e, st) {
    if (!mounted) return;
    final cached = ref.read(regionListNotifierProvider);
    if (cached.isNotEmpty) {
      final l10n = AppLocalizations.of(context)!;
      ref.read(toastProvider.notifier).add(
        ToastMessage(
          type: ToastType.error,
          icon: FontAwesomeIcons.circleExclamation,
          text: l10n.error_saving_settings,
        ),
      );
      return;
    }
    setState(() {
      _freshInstallError = e;
      _freshInstallStackTrace = st;
    });
  }
}
```

**Row-dispatch pattern to copy for the new group/leaf dispatch** (lines 255-269, `_buildRegionRow`) — same precedence-gate shape (check a status/kind, dispatch to the right builder), applied to `row.node.kind == leaf ? _buildRegionRow(entity) : _buildGroupRow(node, depth)`:
```dart
Widget _buildRegionRow(RegionEntity region) {
  if (region.catalogStatus == CatalogStatus.building) {
    return _buildDisabledRow(region, caption: ...);
  }
  if (region.catalogStatus == CatalogStatus.error) {
    return _buildDisabledRow(region, caption: ...);
  }
  return _buildActiveRow(region);
}
```

**List rendering pattern to replace** (lines 158-165) — `ListView.separated` over `filtered` becomes `ListView.builder` over `visibleRows` (from `flattenVisible`); KEEP the same `padding`/empty-state guard structure:
```dart
ListView.separated(
  padding: const EdgeInsets.only(bottom: 16),
  itemCount: filtered.length,
  separatorBuilder: (_, _) => const SizedBox(height: 16),
  itemBuilder: (context, index) => _buildRegionRow(filtered[index]),
),
```
Per RESEARCH.md Pitfall 7, the new `itemBuilder` MUST key each row explicitly: `key: ValueKey(row.node.id)` (not `ValueKey(i)`), since row widget *type* now varies (group vs. leaf) at a given index across rebuilds.

**Untouched per-row methods (DO NOT modify; call as-is from the new nested renderer):** `_buildActiveRow`, `_buildVectorTile`, `_buildDemTile`, `_buildVectorTrailing`, `_buildDemTrailing`, `_tileLeadingIcon`, `_tileSubtitle`, `_buildDiskUsageSummary`, `_hasAnyDiskUsage`, `_save`, `_onDownloadVector`, `_onCancelVector`, `_onDownloadDem`, `_onCancelDem`, `_onDeleteDemPackage`, `_onDeleteRegion` (lines 300-722) — all take a `RegionEntity`/nothing tree-shaped, so they compose directly under indented rows.

**Empty-state pattern to reuse for D-04's offline empty state** (lines 226-248, `_buildEmptyState`) — reuse this exact method, just with new copy keys (`regions_offline_unavailable_title`/`_body`):
```dart
Widget _buildEmptyState({required String title, required String body}) {
  return Center(
    child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(title, style: Theme.of(context).textTheme.bodyLarge, textAlign: TextAlign.center),
          const SizedBox(height: 8),
          Text(body, style: Theme.of(context).textTheme.bodySmall, textAlign: TextAlign.center),
        ],
      ),
    ),
  );
}
```

---

### `db/routes/regions_get.go` (controller, request-response)

**Analog:** itself — one-line addition to the existing entry map.

**Current entry-map pattern** (lines 45-52):
```go
entry := map[string]any{
    "id":     r.Id,
    "name":   r.GetString("name"),
    "kind":   r.GetString("kind"),
    "parent": r.GetString("parent"), // relation field's raw value = parent record id, "" for roots (A3)
    "path":   r.GetString("path"),
    "depth":  r.GetInt("depth"),
}
```
**Add** (D-07): `"sort_order": r.GetInt("sort_order"),` — applies to every row (group and leaf), so it belongs in this same base map, above the `if r.GetString("kind") != "leaf"` branch at line 54, not inside either branch.

**Error-handling convention already in file** (lines 39-41, 71): use `e.App.FindAllRecords`/`e.InternalServerError` style consistently — no change needed here since `sort_order` reads from the already-loaded `r` record, no new query.

---

### `app/test/util/region_tree_util_test.dart` (test, transform)

**Analog:** `app/test/util/region_disk_usage_util_test.dart`

**Structure to copy** (full file pattern, lines 1-40+): top-of-file comment block explaining WHY this test exists (tie back to a decision letter), `group()`/`test()` nesting, a small `buildX()` helper factory for constructing fixture objects inline:
```dart
import 'package:flutter_test/flutter_test.dart';
// ... model imports

// ---------------------------------------------------------------------------
// Tests for region_tree_util's buildRegionTree/computeDefaultExpanded/
// flattenVisible/computeFilterMatches — pure functions, no ObjectBox/network
// needed (mirrors region_disk_usage_util_test.dart's precedent for testing
// pure utils directly, per RESEARCH.md Pitfall 5).
// ---------------------------------------------------------------------------

void main() {
  RegionHierarchyRow buildRow({
    required String id,
    String parent = '',
    RegionNodeKind kind = RegionNodeKind.leaf,
    int sortOrder = 0,
  }) => RegionHierarchyRow(
    id: id, name: id, kind: kind, parent: parent, path: id, depth: 0, sortOrder: sortOrder,
  );

  group('buildRegionTree', () {
    test('roots have parent == "" and are sorted by sortOrder', () {
      // ...
    });
  });
}
```
Per RESEARCH.md Pitfall 5, this is the highest-value, cheapest test coverage in the phase — write it before/alongside any widget-level test.

---

## Shared Patterns

### Provider-invalidation-in-finally (mutation actions)
**Source:** `settings_offline_regions_screen.dart:623-641` (`_save`)
**Apply to:** No new mutation actions are added in this phase (D-01/D-03 keep download/cancel/delete untouched) — this pattern is called out only so any new code does NOT accidentally trigger a tree rebuild through this same invalidation path (see Pitfall 1 below).
```dart
Future<void> _save(Future<void> Function() op) async {
  try {
    await op();
  } catch (_) {
    if (!mounted) return;
    // ... error toast
  } finally {
    if (mounted) ref.invalidate(regionListNotifierProvider);
  }
}
```

### Per-element try/catch/skip on hostile API data
**Source:** `region_provider.dart:40-46` (`parseRegionCatalog`), `regions_get.go`'s own group-row-drop comment (lines 54-62)
**Apply to:** `parseRegionHierarchyRows` (new) — same posture, never abort the whole parse for one bad element.

### Freezed API-response model convention
**Source:** `region_catalog_entry.dart` (full file)
**Apply to:** `region_hierarchy_row.dart` — `@freezed` + `part '..._freezed.dart'`/`part '..._g.dart'` + `factory .fromJson`, `@JsonKey(name: 'snake_case')` for camelCase mapping.

### Dart record for a render-time tuple type
**Source:** `app/lib/services/tile_repository_manager.dart:54` (`splitRegionTilePaths`'s `({List<String> vectorPaths, List<String> demPaths})` return type)
**Apply to:** `flattenVisible`'s return type `List<({RegionTreeNode node, int depth})>` — zero-codegen record, not a new Freezed class, matching this codebase's existing precedent for purely-render-time tuples.

## Critical Cross-Cutting Insight (from RESEARCH.md, load-bearing for the plan)

**Tree shape and live leaf data must be rebuilt on different cadences.** `regionListNotifierProvider` is invalidated on every download/cancel/delete action (`_save`'s `finally`, `_onCancelVector`, etc.) — far more often than the catalog is genuinely re-fetched. Tree shape (`_treeRoots`) and expand-state seeding must be cached in `State` and reseeded ONLY inside `_refreshCatalog()`, never derived inside `build()`. `flattenVisible` (pure, cheap) is the only tree function that should run on every `build()`. See RESEARCH.md Pattern 2 and Pitfall 1 for the full rationale and a complete illustrative sketch.

## No Analog Found

None — every file in this phase has a strong same-repo analog (either the JS it replaces, a sibling Dart file of the identical shape, or itself).

## Metadata

**Analog search scope:** `app/lib/models/`, `app/lib/util/`, `app/lib/provider/region/`, `app/lib/entities/`, `app/lib/routes/`, `app/test/util/`, `db/routes/`, `db/routes/regions_ext/`
**Files scanned:** `region_catalog_entry.dart`, `region_provider.dart`, `region_entity.dart`, `region_disk_usage_util.dart`, `settings_offline_regions_screen.dart`, `regions_get.go`, `regions_ui.html` (lines 500-600), `tile_repository_manager.dart`, `region_disk_usage_util_test.dart`
**Pattern extraction date:** 2026-07-27
