# Phase 31: Flutter Settings Hierarchy - Research

**Researched:** 2026-07-27
**Domain:** Flutter/Dart mobile UI — client-side tree flattening, Riverpod state shape, Freezed model design
**Confidence:** HIGH (this phase is >90% an internal port of an already-shipped algorithm within the same repo; verified by direct reading of both the source (`regions_ui.html`) and every target file, not external libraries)

## Summary

This phase ports a JS tree-flattening algorithm (`buildTree`/`computeDefaultExpanded`/`flattenVisible`/`computeFilterMatches`, already shipped in Phase 30's admin page) into Dart, and layers it *above* the existing, unmodified `regionListNotifierProvider` → `RegionEntity` → `_buildActiveRow` rendering pipeline in `settings_offline_regions_screen.dart`. No new external packages are required — every recommendation below uses Flutter SDK, Dart 3 language features, and packages already in `app/pubspec.yaml` (`freezed_annotation`, `flutter_riverpod`, `font_awesome_flutter`).

The single most important architectural decision this research surfaces — **not spelled out in CONTEXT.md** — is that tree *shape* (group/leaf structure, sort order) and tree *content* (live download status per leaf) must be rebuilt on different cadences. `regionListNotifierProvider` is invalidated after **every** download/cancel/delete action (`_save`'s `finally`, `_onCancelVector`, etc.) — far more often than the catalog is actually re-fetched. If expand/collapse state or the tree structure itself were recomputed every time that provider changes, a user's manual expand/collapse choices would be silently wiped out every time they tap a Download button. The recommended design (see Pattern 1/2 below) keeps hierarchy *shape* in local `State` (rebuilt only on a genuine catalog fetch) and joins it against the *live* `RegionEntity` list at render time via a cheap `Map<String, RegionEntity>` lookup, so `ListView.builder` rows stay live without the tree ever needing to be reconstructed on a progress tick.

A second finding: `RegionCatalogEntry.fromJson` (per D-05) does not just need "handling" for group rows — it must stay completely untouched, and a **new**, separate, always-succeeding model (`RegionHierarchyRow`, all fields required per `regions_get.go`'s unconditional entry-map keys) must be parsed from the *same* HTTP response to recover group rows and the parent/depth/sort_order shape without touching the existing ObjectBox upsert pipeline at all.

**Primary recommendation:** Add one new lightweight Freezed model (`RegionHierarchyRow`), one new plain (non-Freezed) mutable tree-node class (`RegionTreeNode`) with four pure top-level functions ported 1:1 from the admin's JS, hold expand/filter state as plain `State` fields (not a new Riverpod provider), and render via `ListView.builder` over a `List<({RegionTreeNode node, int depth})>` (a Dart record — a pattern this codebase already uses, see `splitRegionTilePaths` in `tile_repository_manager.dart:54`).

## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Port the admin page's flatten-list algorithm (`buildTree` / `computeDefaultExpanded` / `flattenVisible` from `db/routes/regions_ext/regions_ui.html:500-582`) to Dart, rather than nesting Flutter's built-in `ExpansionTile`. Build the tree once from the flat API response, compute a flat depth-tagged array of currently-visible rows based on expand state, and render it via `ListView.builder` with depth-based indentation. Leaf rows keep their existing `_buildRegionRow`/`_buildVectorTile`/`_buildDemTile` rendering, just indented — no rewrite of the download-action UI.
- **D-02:** Default expand state mirrors the admin's rule (its D-07): a branch auto-expands only if it contains a leaf the user has already downloaded (Vector and/or DEM present) or mid-download; everything else starts collapsed.
- **D-03:** Group rows (`kind != "leaf"`) are NOT persisted to ObjectBox. They exist only as an ephemeral client-side tree structure, rebuilt from the API's flat list on each fetch, used purely to organize/nest the existing `RegionEntity` leaf rows for rendering. No new ObjectBox entity, no group-specific local state to keep in sync.
- **D-04 (deliberate scoped exception to APPUI-02):** When offline (no fresh catalog fetch succeeds and no cached hierarchy shape exists), the screen shows an **empty state**, not a fallback flat list. Previously-downloaded regions become unmanageable (no cancel/delete, no disk-usage view) while offline — a deliberate, scoped regression, not an oversight.
- **D-05:** Because `RegionCatalogEntry.fromJson` currently requires `bbox`/`status` and silently drops group rows, the parsing layer needs a lightweight in-memory group-node representation (separate from `RegionCatalogEntry`, which stays leaf/download-focused) so group rows survive parsing long enough to build the tree.
- **D-06:** Filter behavior mirrors the admin page's `computeFilterMatches`: typing narrows the tree to matching leaves/groups plus their full ancestor chain, auto-expanded, so matches are visible without extra taps. Not a "flatten to matches only" simplification.
- **D-07:** Add `sort_order` to `GET /api/v1/regions`'s response (`db/routes/regions_get.go`'s entry map) — the DB column already exists but the handler doesn't surface it yet. The Flutter tree sorts siblings within a group by `sort_order`.

### Claude's Discretion

- Exact debounce timing on the filter input, if any.
- Visual treatment of the empty state shown offline (D-04) — icon, copy, whether it explains why.
- Whether the depth-indentation uses fixed padding per level or some other visual grouping.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| APPUI-01 | Settings → Offline Maps/Regions presents downloadable regions as a collapsible hierarchy matching the admin-defined tree, instead of a flat list | Pattern 1 (ported tree algorithm), Pattern 2 (render-time join), Code Examples section give a directly implementable Dart port; UI-SPEC's Interaction Contract table gives exact tap-target/chevron/indentation rules |
| APPUI-02 | Existing per-region download/cancel/delete actions and disk-usage summary continue working unchanged within the new hierarchical presentation — no download-UX regression | Pattern 2 explicitly preserves `_buildRegionRow`/`_buildActiveRow`/`_buildVectorTile`/`_buildDemTile`/`totalRegionDiskUsageBytes` untouched; Pitfall 1 (expand-state churn) and Pitfall 2 (tree-shape rebuild cadence) are the two concrete failure modes that would otherwise regress this requirement |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Region hierarchy shape (group/leaf tree, parent/depth/sort_order) | Browser/Client (Flutter, in-memory) | API/Backend (`GET /api/v1/regions` is the source data) | D-03: tree is ephemeral, client-only, rebuilt from the existing flat API response — no new persistence tier |
| `sort_order` field exposure | API/Backend | — | D-07: a genuine gap in `db/routes/regions_get.go`'s entry map; DB column already exists, one-line Go fix |
| Leaf download/cancel/delete actions, progress | Browser/Client (Flutter) | Database/Storage (ObjectBox `RegionEntity`/`DownloadedTilePackageEntity`) | Unchanged — Phase 22-27 already established this; this phase does not touch it |
| Disk-usage summary | Browser/Client (Flutter, direct filesystem stat) | — | Unchanged — `region_disk_usage_util.dart` stats the filesystem directly, no tier change |
| Expand/collapse UI state | Browser/Client (Flutter, page-local `State`) | — | Purely a rendering concern; no reason to elevate to a Riverpod provider or persist across screen disposal (see Pattern 2) |

## Standard Stack

### Core

No new dependencies. This phase is a client-side algorithm port and a one-field Go addition.

| Library | Version (installed) | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `freezed_annotation` / `freezed` | 3.1.0 / 3.2.5 [VERIFIED: `app/pubspec.yaml`] | New `RegionHierarchyRow` model | Matches every other API-response model in this codebase (`RegionCatalogEntry`, `RegionDownloadState`) |
| `json_annotation` / `json_serializable` | 4.11.0 / 6.13.0 [VERIFIED: `app/pubspec.yaml`] | `RegionHierarchyRow.fromJson` | Same convention as `RegionCatalogEntry.fromJson` |
| `flutter_riverpod` / `riverpod_annotation` | 3.3.1 / 4.0.2 [VERIFIED: `app/pubspec.yaml`] | No new providers needed for tree state (see Pattern 2) — existing `regionListNotifierProvider`/`tileRepositoryStatusProvider` are read as-is | Established provider pattern already covers all data this phase touches |
| `font_awesome_flutter` | 11.0.0 [VERIFIED: `app/pubspec.yaml`] | Chevron icons (`angleRight`/`angleDown`) per UI-SPEC | Existing icon convention on this exact screen |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Dart 3 records (language feature, not a package) | Dart 3.12.2 [VERIFIED: `dart --version` in this environment] | The flattened-row type (`({RegionTreeNode node, int depth})`) | Already an established pattern in this exact codebase — `tile_repository_manager.dart:54`'s `splitRegionTilePaths` returns `({List<String> vectorPaths, List<String> demPaths})`. Prefer a record over a new Freezed class for this purely-render-time tuple — zero codegen, zero equality/copyWith overhead for a value that's rebuilt every frame. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Plain mutable `RegionTreeNode` class | Freezed immutable tree node with `@Default(const [])` children | Freezed's `copyWith`-based immutability fights a single-pass, parent-then-attach-children imperative build (`buildTree`'s exact shape) — you'd need a two-pass "build then freeze" or a mutable builder class anyway. Since `RegionTreeNode` is truly ephemeral (D-03: rebuilt on every fetch, never diffed/compared by value, looked up by `id` in a `Map`), the deliberate exception to this codebase's "everything is Freezed" convention is justified. Flag this explicitly in the plan/verification so it isn't flagged as an inconsistency later. |
| A single Riverpod provider for `(tree, expanded, filterMatches)` combined | Keep expand/filter state as plain `State` fields (current `_searchQuery` precedent) | A combined provider adds indirection with no payoff — nothing outside this one screen needs tree/expand state, and Riverpod's `keepAlive` semantics would need extra care to avoid exactly the same expand-state-churn bug this research flags for the naive approach. Page-local `State` is simpler and mirrors the screen's own existing `_searchQuery` field. |
| Fetching hierarchy rows via a second HTTP request | Parse the *same* `GET /api/v1/regions` response twice (once via existing `parseRegionCatalog`, once via new `parseRegionHierarchyRows`) | A second request doubles network traffic for identical data and reintroduces a race between the two fetches (which could disagree if the catalog changes between requests). Parsing the same response body twice is free. |

**Installation:** None — no new pubspec entries.

**Version verification:** All versions above were read directly from `app/pubspec.yaml` and the local `flutter --version`/`dart --version` output in this environment — no registry lookup needed since nothing new is being installed.

## Package Legitimacy Audit

Not applicable — this phase introduces zero new external packages (Node.js, Python, or Dart/pub). Every model/pattern recommended above uses packages already present in `app/pubspec.yaml`. `slopcheck`/registry verification was not run because there is nothing to install.

## Architecture Patterns

### System Architecture Diagram

```
GET /api/v1/regions (Go: regions_get.go, unchanged shape + new sort_order field)
        |
        v
  Dio response.data  (raw List<dynamic>, both group and leaf rows)
        |
        +----------------------------------------+
        |                                         |
        v                                         v
parseRegionCatalog(data)                 parseRegionHierarchyRows(data)
  (EXISTING, untouched)                    (NEW — every row succeeds,
  -> List<RegionCatalogEntry>               only 6 always-present fields:
     (leaf rows only; group rows            id/name/kind/parent/path/depth
      silently dropped by fromJson's         + sort_order once D-07 lands)
      required bbox/status)                 -> List<RegionHierarchyRow>
        |                                         |
        v                                         v
RegionRepository.upsertCatalog()          buildRegionTree(hierarchyRows)
  -> ObjectBox RegionEntity rows            (pure, in-memory, cached in
     (UNCHANGED — Phase 22-27 code)          _SettingsOfflineRegionsScreenState
        |                                    as _treeRoots; rebuilt ONLY on a
        |                                    genuine fetch, never on a
        |                                    download/progress-driven
        |                                    provider invalidation)
        v                                         |
regionListNotifierProvider                        |
  (ObjectBox snapshot, watched                     |
   every build() as today)                         |
        |                                          |
        +------------------+-----------------------+
                           |
                           v
              render-time JOIN (cheap, every build()):
              Map<String, RegionEntity> byId = {for r in regions r.id: r}
                           |
                           v
     flattenVisible(_treeRoots, _expandedIds, _filterMatchIds)
        -> List<({RegionTreeNode node, int depth})>   (pure, re-run every
                                                         build — O(n) walk,
                                                         cheap even at 1306
                                                         nodes)
                           |
                           v
              ListView.builder(itemBuilder: (i) =>
                row.node.kind == leaf
                  ? _buildRegionRow(byId[row.node.id])   // UNCHANGED method
                  : _buildGroupRow(row.node, row.depth))  // NEW, per UI-SPEC
```

### Recommended Project Structure

```
app/lib/
├── models/
│   ├── region_catalog_entry.dart      # UNCHANGED (D-05)
│   ├── region_hierarchy_row.dart      # NEW — Freezed, all-fields-required, group+leaf
│   └── region_tree_node.dart          # NEW — plain mutable class (not Freezed, see Alternatives)
├── util/
│   └── region_tree_util.dart          # NEW — pure functions: buildRegionTree,
│                                       #   computeDefaultExpanded, flattenVisible,
│                                       #   computeFilterMatches (ported from regions_ui.html)
├── entities/
│   └── region_entity.dart             # UNCHANGED (D-03)
├── provider/region/
│   └── region_provider.dart           # ADD: parseRegionHierarchyRows() + a
│                                       #   fetchRegionHierarchy(Dio) sibling to
│                                       #   fetchRegionCatalog(); everything else UNCHANGED
└── routes/
    └── settings_offline_regions_screen.dart  # MODIFIED per Pattern 1/2 below;
                                                 all _build*Tile/*Row/*Trailing methods UNCHANGED
```

### Pattern 1: Ported pure tree functions (D-01/D-02/D-06)

**What:** Four pure, top-level Dart functions mirroring `regions_ui.html:515-615` exactly, differing only in Dart idiom (no `Object.assign`, use `Map`/`Set`/records).

**When to use:** Called once per genuine catalog fetch (`buildRegionTree`, `computeDefaultExpanded`) and once per `build()` call (`flattenVisible`, `computeFilterMatches` when a filter is active).

**Example:**
```dart
// Source: ported from db/routes/regions_ext/regions_ui.html:515-582
// New file: app/lib/util/region_tree_util.dart

import 'package:wanderer/models/region_hierarchy_row.dart';
import 'package:wanderer/models/region_tree_node.dart';

/// Builds the group/leaf tree from every hierarchy row (group AND leaf).
/// Mirrors buildTree(rows) — single pass, byId map, parent-then-attach.
/// Siblings sorted by sortOrder (D-07); roots sorted the same way.
List<RegionTreeNode> buildRegionTree(List<RegionHierarchyRow> rows) {
  final byId = <String, RegionTreeNode>{
    for (final r in rows)
      r.id: RegionTreeNode(
        id: r.id,
        name: r.name,
        kind: r.kind,
        parentId: r.parent,
        path: r.path,
        depth: r.depth,
        sortOrder: r.sortOrder,
      ),
  };

  final roots = <RegionTreeNode>[];
  for (final node in byId.values) {
    // Dart has no JS-style truthy-string check -- explicit isNotEmpty guard
    // (regions_get.go emits "" for roots, per its own A3 comment).
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

/// A branch auto-expands only if it contains a leaf with an existing
/// download (vector and/or DEM present) or a mid-flight download (D-02).
/// [statusOf] looks up the CURRENT RegionEntity for a leaf id -- passed in
/// rather than embedded in the tree, so this can be recomputed against a
/// fresh `regions` snapshot without ever rebuilding tree shape (Pattern 2).
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

  for (final r in roots) {
    walk(r);
  }
  return expanded;
}

/// Flattens the tree into visible, depth-tagged rows given the current
/// expand/filter state. Pure -- safe to call on every build().
List<({RegionTreeNode node, int depth})> flattenVisible(
  List<RegionTreeNode> roots,
  Set<String> expanded,
  Set<String>? filterMatches,
) {
  final out = <({RegionTreeNode node, int depth})>[];
  void walk(List<RegionTreeNode> nodes, int depth) {
    for (final n in nodes) {
      if (filterMatches != null && !filterMatches.contains(n.id)) continue;
      out.add((node: n, depth: depth));
      if (n.kind == RegionNodeKind.group) {
        // D-06: while filtering, matched groups are treated as expanded
        // (non-matching children already excluded above); otherwise fall
        // back to the explicit expandedSet (D-02).
        final shouldDescend = filterMatches != null || expanded.contains(n.id);
        if (shouldDescend) walk(n.children, depth + 1);
      }
    }
  }

  walk(roots, 0);
  return out;
}

/// A node whose own name matches pulls in its whole subtree; a node that
/// only matches via a descendant stays narrow (ancestors + that path).
/// Mirrors computeFilterMatches(roots, query) exactly.
Set<String>? computeFilterMatches(List<RegionTreeNode> roots, String query) {
  if (query.isEmpty) return null;
  final q = query.toLowerCase();
  final matchSet = <String>{};

  void addSubtree(RegionTreeNode node) {
    matchSet.add(node.id);
    for (final c in node.children) {
      addSubtree(c);
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

  for (final r in roots) {
    walk(r, const []);
  }
  return matchSet;
}
```

### Pattern 2: Separate tree-shape rebuild cadence from live leaf data (the key architectural insight)

**What:** Tree *shape* (`_treeRoots: List<RegionTreeNode>`) and expand-state seeding are cached in `State` and rebuilt only when `_refreshCatalog()` actually fetches new data — never as a side effect of `regionListNotifierProvider`'s frequent post-mutation invalidations. Live leaf data (`RegionEntity`, download progress) is joined in at render time via an O(1) map lookup, so `_buildRegionRow`/`_buildActiveRow` keep receiving a fresh `RegionEntity` on every build without the tree itself ever being torn down and rebuilt.

**When to use:** Any time hierarchical UI state (expand/collapse) is layered over a provider that changes far more often than the hierarchy shape itself does.

**Example:**
```dart
// settings_offline_regions_screen.dart -- illustrative sketch, not final code

class _SettingsOfflineRegionsScreenState
    extends ConsumerState<SettingsOfflineRegionsScreen> {
  String _searchQuery = ''; // unchanged

  // NEW state -- tree SHAPE, rebuilt only on a genuine fetch:
  List<RegionTreeNode>? _treeRoots;
  Set<String> _expandedIds = {};
  bool _expandedSeeded = false;

  Future<void> _refreshCatalog() async {
    try {
      await ref.read(regionRepositoryProvider).refreshCatalog(); // unchanged
      final hierarchyRows =
          await ref.read(regionRepositoryProvider).fetchHierarchyRows(); // NEW
      if (!mounted) return;
      ref.invalidate(regionListNotifierProvider);
      setState(() {
        _treeRoots = buildRegionTree(hierarchyRows);
        _expandedSeeded = false; // reseed defaults against the NEW shape
      });
    } catch (e, st) {
      // ... existing D-04 empty-state / toast logic, unchanged shape
    }
  }

  @override
  Widget build(BuildContext context) {
    final regions = ref.watch(regionListNotifierProvider); // unchanged watch
    final byId = {for (final r in regions) r.id: r}; // cheap, every build

    if (_treeRoots != null && !_expandedSeeded) {
      _expandedIds = computeDefaultExpanded(
        _treeRoots!,
        (leafId) {
          final r = byId[leafId];
          if (r == null) return false;
          return r.status == RegionStatus.downloaded ||
              r.status == RegionStatus.updateAvailable ||
              r.status == RegionStatus.downloading ||
              (r.demPackage.target?.status == PackageStatus.downloaded) ||
              (r.demPackage.target?.status == PackageStatus.downloading);
        },
      );
      _expandedSeeded = true;
    }

    final filterMatches = _searchQuery.trim().isEmpty || _treeRoots == null
        ? null
        : computeFilterMatches(_treeRoots!, _searchQuery.trim());

    final visibleRows = _treeRoots == null
        ? const <({RegionTreeNode node, int depth})>[]
        : flattenVisible(_treeRoots!, _expandedIds, filterMatches);

    // ... ListView.builder(itemCount: visibleRows.length, itemBuilder: (_, i) {
    //   final row = visibleRows[i];
    //   if (row.node.kind == RegionNodeKind.leaf) {
    //     final entity = byId[row.node.id];
    //     if (entity == null) return const SizedBox.shrink(); // Pitfall 6
    //     return Padding(
    //       padding: EdgeInsets.only(left: _indentFor(row.depth)),
    //       child: _buildRegionRow(entity),
    //     );
    //   }
    //   return _buildGroupRow(row.node, row.depth); // NEW method, per UI-SPEC
    // })
  }
}
```

### Pattern 3: `RegionHierarchyRow` — the always-succeeding sibling model (D-05)

**What:** A new Freezed model containing ONLY the six fields `regions_get.go`'s `RegionsList` unconditionally sets on every row (group or leaf): `id`, `name`, `kind`, `parent`, `path`, `depth`, plus `sort_order` once D-07 lands. Crucially, this model NEVER requires `bbox`/`status`, so `fromJson` never throws for a group row.

**Example:**
```dart
// Source: db/routes/regions_get.go lines 45-52 (fields ALWAYS present, both
// kind=group and kind=leaf) + sort_order once D-07's Go change lands.
// New file: app/lib/models/region_hierarchy_row.dart

import 'package:freezed_annotation/freezed_annotation.dart';

part 'region_hierarchy_row.freezed.dart';
part 'region_hierarchy_row.g.dart';

enum RegionNodeKind {
  @JsonValue('group')
  group,
  @JsonValue('leaf')
  leaf,
}

@freezed
abstract class RegionHierarchyRow with _$RegionHierarchyRow {
  const factory RegionHierarchyRow({
    required String id,
    required String name,
    required RegionNodeKind kind,
    /// Raw relation id; `""` for roots -- see regions_get.go's own A3 comment.
    required String parent,
    required String path,
    required int depth,
    /// Defaults to 0 if the backend hasn't shipped D-07's sort_order field
    /// yet (defensive against task-ordering slop within this same phase);
    /// once D-07 lands this is always present and meaningful.
    @JsonKey(name: 'sort_order') @Default(0) int sortOrder,
  }) = _RegionHierarchyRow;

  factory RegionHierarchyRow.fromJson(Map<String, dynamic> json) =>
      _$RegionHierarchyRowFromJson(json);
}
```

### Pattern 4: The Go `sort_order` addition (D-07)

**What:** A one-line addition to `db/routes/regions_get.go`'s entry map, applying to every row (group and leaf) since sort order matters for both.

**Example:**
```go
// Source: db/routes/regions_get.go lines 45-52
entry := map[string]any{
    "id":         r.Id,
    "name":       r.GetString("name"),
    "kind":       r.GetString("kind"),
    "parent":     r.GetString("parent"),
    "path":       r.GetString("path"),
    "depth":      r.GetInt("depth"),
    "sort_order": r.GetInt("sort_order"), // NEW -- D-07; column already exists
                                            // (1785000000_create_regions_collection.go:36,67)
}
```

### Anti-Patterns to Avoid

- **Rebuilding `_treeRoots` inside `build()` or on every `regionListNotifierProvider` change:** This is the single biggest risk in this phase. It silently resets user-driven expand/collapse state every time a download completes, a cancel fires, or a progress tick arrives, directly regressing the "no download-UX regression" spirit of APPUI-02 even though no single line looks wrong in isolation.
- **Embedding a `RegionEntity` reference inside `RegionTreeNode`:** Couples tree shape to live data identity, forcing a full tree rebuild whenever `RegionEntity` instances change (which ObjectBox does on every query). Keep the join at render time via a flat `Map<String, RegionEntity>` instead (Pattern 2).
- **Using Flutter's `ExpansionTile`:** Explicitly rejected by D-01. `ExpansionTile` nests naturally for a shallow tree but does not give you a single flat, depth-indexed row list — which UI-SPEC's 44px-minimum-touch-target and 80px-capped-indent rules are specified against — and would require re-deriving `flattenVisible`'s semantics from `ExpansionTile`'s own internal state anyway.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Filter-input debounce | A new `Timer`-based debounce inside the tree filter logic | Nothing — `WandererSearchBar` (`app/lib/components/base/wanderer_searchbar.dart:20-32`) already wraps its `onChanged` callback in a 500ms `Timer`-based debounce, unconditionally, for every caller. The filter recompute itself (`computeFilterMatches`/`flattenVisible`) is pure, synchronous, and cheap even at ~1300 nodes, so no *additional* debounce is needed on top of the widget's existing one. | Avoids a double-debounce (500ms widget-level + a second custom one) that would make the UI feel laggier than today's flat-list filter, contradicting UI-SPEC's explicit "No debounce... same immediacy as today" instruction. |
| Generic tree/hierarchy widget | A pub.dev tree-view package (e.g. a generic `flutter_treeview`-style dependency) | The ported pure-function approach (Pattern 1) | D-01 explicitly rejects both `ExpansionTile` and (implicitly) any third-party tree widget — the admin page's algorithm is the specified behavioral reference, and a generic package would need custom row-builder shims anyway to reuse `_buildRegionRow`/`_buildVectorTile`/`_buildDemTile` verbatim. |

**Key insight:** This phase's "don't hand-roll" risk is inverted from the usual case — the risk here is *rebuilding* something (the tree, on every provider tick) that should be memoized, not under-building something that needs a library. The existing per-row download UI is already correctly factored (plain methods taking a `RegionEntity`) and needs zero new abstraction.

## Common Pitfalls

### Pitfall 1: Expand-state churn from provider invalidation frequency
**What goes wrong:** Every download/cancel/delete action calls `ref.invalidate(regionListNotifierProvider)`. If tree shape + expand defaults are recomputed inside `build()` on every rebuild (rather than cached in `State` and only reseeded on a genuine fetch), a user's manual expand/collapse taps are silently reverted mid-session.
**Why it happens:** It's the "obvious" naive implementation — `regions = ref.watch(...)` already triggers `build()`, so it's tempting to derive everything (including the tree and expand defaults) from that single watch.
**How to avoid:** Cache `_treeRoots`/`_expandedIds` in `State`, seeded only in `_refreshCatalog()` (Pattern 2). `flattenVisible` (cheap, pure) is the only tree function that should run on every `build()`.
**Warning signs:** Manually expanding a group, then tapping Download on any visible leaf, causes the group to collapse again.

### Pitfall 2: `RegionCatalogEntry.fromJson` per-element try/catch masking group rows
**What goes wrong:** `parseRegionCatalog`'s existing per-element `try { ... } catch (_) { }` (region_provider.dart:40-46) already silently swallows every group row today (missing `bbox`/`status` throws inside `fromJson`, caught, skipped). It's easy to assume this catch needs "fixing" — it does not; D-05 explicitly keeps this behavior and adds a *separate* parse path instead.
**Why it happens:** The catch block's comment ("Skip this malformed element") reads as a bug for group rows specifically, when it's actually working exactly as designed for the *pre-Phase-31* app (confirmed by `regions_get.go`'s own comment: "the existing Flutter parser... drops entries missing its required bbox/status fields, so group rows are ignored by the pre-Phase-31 app rather than crashing it").
**How to avoid:** Do not touch `RegionCatalogEntry`/`parseRegionCatalog`. Add `RegionHierarchyRow`/`parseRegionHierarchyRows` as a fully independent parse over the same raw response (Pattern 3).
**Warning signs:** A diff that touches `region_catalog_entry.dart`'s required fields.

### Pitfall 3: JS truthy-string root check doesn't port directly
**What goes wrong:** `buildTree`'s JS check `if (node.parent && byId.has(node.parent))` relies on `""` being falsy. Dart has no implicit string-to-bool coercion (won't even compile), forcing an explicit check — but a careless port might write `node.parentId != null` instead of `node.parentId.isNotEmpty`, which is wrong since `regions_get.go` always emits `parent: ""` (never `null`) for roots (its own A3 comment).
**Why it happens:** `parent` is a required `String` field (never nullable) per the Go handler's `r.GetString("parent")`, which returns `""` not a missing key.
**How to avoid:** Use `node.parentId.isNotEmpty ? byId[node.parentId] : null`, matching Pattern 1's code example exactly.
**Warning signs:** Every region renders as a root (flat list again) or a runtime null-check failure on `byId['']`.

### Pitfall 4: `sort_order` absent until D-07's Go change lands
**What goes wrong:** If the Flutter model/parsing task is executed before the Go task in the same phase (or the Go task is skipped/rolled back), `RegionHierarchyRow.sortOrder` silently defaults to `0` for every row, and sibling ordering falls back to array-insertion order (`FindAllRecords`'s implicit order, not guaranteed to match `sort_order`).
**Why it happens:** These are two independently-executable pieces of work (a Go handler change and a Dart model), and CONTEXT.md frames D-07 as "small, surgical" — easy to sequence last or skip under time pressure.
**How to avoid:** Sequence the Go change before or in the same task as the Dart parsing work; verify by asserting a specific known-order pair of siblings (e.g., two named sub-regions with distinct `sort_order`) renders in the expected order in a widget test.
**Warning signs:** Region ordering visually differs between the admin's tree (which explicitly `sort=sort_order`s) and the app's tree.

### Pitfall 5: No existing widget test coverage for this screen
**What goes wrong:** `settings_offline_regions_screen.dart` has zero widget tests today (only `region_entity_test.dart`, `region_catalog_entry_test.dart`, `region_provider_test.dart`, `region_disk_usage_util_test.dart`, `region_tile_status_util_test.dart`, `region_download_state_test.dart` exist — all model/util/provider level, not widget level). There is no baseline regression test proving "the flat list renders download actions correctly today," so APPUI-02's "no regression" claim cannot be automatically verified against a pre-existing test; new widget tests must be written from scratch as part of this phase, not merely extended.
**Why it happens:** Testing this ConsumerStatefulWidget requires wiring a `ProviderScope` with overridden `regionListNotifierProvider`/`tileRepositoryStatusProvider` — a nontrivial setup absent from this codebase's Settings screens historically.
**How to avoid:** Plan should include new widget tests for: (1) leaf rows still show Download/Cancel/Delete actions correctly nested under a group in the new hierarchy render, (2) disk-usage summary total is unchanged whether regions are flat or nested, (3) `computeDefaultExpanded`/`flattenVisible`/`computeFilterMatches` pure-function unit tests (cheapest, highest-value coverage — mirrors the model/util-level test pattern already used for `region_disk_usage_util_test.dart`).
**Warning signs:** Plan Tasks list widget verification purely as manual/on-device, when the pure tree functions are trivially unit-testable without a live ObjectBox store (same precedent as `region_disk_usage_util_test.dart`, `region_file_path_test.dart`).

### Pitfall 6: Leaf tree node with no matching `RegionEntity`
**What goes wrong:** `RegionRepository.upsertCatalog` (region_provider.dart:107-122) already silently skips a leaf entry whose `bbox` fails `RegionEntity.fromCatalogEntry`'s 4-element guard (`FormatException`, caught, skipped). If a hierarchy row's `id` has no corresponding entry in `byId` (the render-time `Map<String, RegionEntity>`), the render must not crash.
**Why it happens:** `RegionHierarchyRow` parsing (always succeeds for the 6 base fields) and `RegionCatalogEntry`/ObjectBox upsert (can silently drop a malformed leaf) run over the same response through two independent, differently-strict pipelines — they can disagree.
**How to avoid:** Guard the leaf-row branch of the `ListView.builder` itemBuilder: if `byId[row.node.id]` is `null`, render `const SizedBox.shrink()` (or omit the row entirely from `visibleRows` up front) rather than force-unwrapping.
**Warning signs:** A crash or `!`-null-assertion failure only reproducible with a specific malformed catalog entry (rare in practice, but a hostile/misconfigured backend should not crash the app — T-22-05 precedent).

### Pitfall 7: `ListView.builder` item identity across mixed group/leaf rows
**What goes wrong:** Without an explicit `key`, Flutter's element-reuse heuristics for a `ListView.builder` can occasionally misattribute state (e.g. a `ListTile`'s internal `InkWell` splash animation) across scroll when row *types* alternate (group row widget vs. leaf row's bordered-card widget) at the same list index on successive builds (e.g., right after an expand/collapse changes which indices hold which row kind).
**Why it happens:** The existing flat list (`ListView.separated`) never had this problem because every item was structurally identical (`_buildRegionRow` for every index). The new tree has two structurally different widget shapes sharing one `itemBuilder`.
**How to avoid:** Give each rendered row an explicit `key: ValueKey(row.node.id)` (not `ValueKey(i)`) so Flutter tracks each row's identity by region/group id across rebuilds, not by list position.
**Warning signs:** Rare, hard-to-reproduce visual glitches (wrong row briefly shows another row's press-highlight) after expanding/collapsing near the top of a long scrolled list.

## Code Examples

### Group row widget (new, per UI-SPEC's Interaction Contract)

```dart
// Source: 31-UI-SPEC.md's Interaction Contract + Color/Typography sections
Widget _buildGroupRow(RegionTreeNode node, int depth) {
  final l10n = AppLocalizations.of(context)!;
  final isExpanded = _expandedIds.contains(node.id);
  final indent = _indentFor(depth); // capped at depth 4 -> max 80px, per UI-SPEC

  return Semantics(
    label: isExpanded
        ? l10n.regions_group_collapse_label(node.name)
        : l10n.regions_group_expand_label(node.name),
    button: true,
    child: InkWell(
      onTap: () => setState(() {
        if (isExpanded) {
          _expandedIds = {..._expandedIds}..remove(node.id);
        } else {
          _expandedIds = {..._expandedIds, node.id};
        }
      }),
      child: Container(
        constraints: const BoxConstraints(minHeight: 44), // UI-SPEC touch target
        padding: EdgeInsets.only(left: indent, right: 16),
        alignment: Alignment.centerLeft,
        child: Row(
          children: [
            FaIcon(
              isExpanded
                  ? FontAwesomeIcons.angleDown
                  : FontAwesomeIcons.angleRight,
              size: 16,
              color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                node.name,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.bodyLarge!
                    .copyWith(fontWeight: FontWeight.bold),
              ),
            ),
          ],
        ),
      ),
    ),
  );
}

double _indentFor(int depth) => 16.0 + (depth.clamp(0, 4) * 16.0); // base 16, +16/level, cap 80
```

### New ARB keys needed (per UI-SPEC's Copywriting Contract)

```jsonc
// app/lib/i18n/app_en.arb -- add alongside the existing "regions_*" keys
"regions_offline_unavailable_title": "Can't load regions",
"regions_offline_unavailable_body": "Connect to the internet to browse and manage downloadable regions.",
"regions_group_expand_label": "Expand {name}",
"@regions_group_expand_label": { "placeholders": { "name": { "type": "String" } } },
"regions_group_collapse_label": "Collapse {name}",
"@regions_group_collapse_label": { "placeholders": { "name": { "type": "String" } } }
```
Note: this project's `l10n.yaml` points `arb-dir` at `lib/i18n` (not the Flutter-default `lib/l10n`) — confirmed by direct read of `app/l10n.yaml`. Every translated locale file (`app_de.arb`, `app_cs.arb`, etc., ~15+ files under `app/lib/i18n/`) needs the same new keys added (English fallback is acceptable for non-English locales at merge time per this codebase's established practice — see prior phases' l10n additions).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Hierarchy shape (`_treeRoots`) is cached in-memory only, for the current app session — not persisted to disk across a cold app restart. D-04's "no cached hierarchy shape exists" is interpreted as "no successful fetch has happened yet this session," not "no disk-persisted shape from a prior session." | Pattern 2, D-04 interpretation | If the intended behavior is closer to a small on-disk cache (e.g., SharedPreferences-stored last-known tree JSON) so a cold offline restart can still show the last-known hierarchy, this assumption under-delivers on APPUI-02's spirit. Low risk given D-03 explicitly frames the tree as ephemeral/rebuilt-per-fetch, and D-04 is described as a "deliberate, accepted regression," but the planner should confirm this reading with the user if there's any doubt. |
| A2 | `RegionTreeNode` should be a plain mutable class, not a Freezed model, as a deliberate exception to this codebase's otherwise-universal Freezed convention. | Standard Stack > Alternatives Considered, Architecture Patterns > Pattern 1 | Low risk technically (the code works either way), but a reviewer unfamiliar with the rationale could flag it as an inconsistency during code review; the plan should carry this rationale forward into a code comment (mirroring how other deliberate codebase exceptions are documented, e.g. `region_entity.dart`'s explicit `.code` shadow-property comments). |
| A3 | `RegionHierarchyRow` and `parseRegionHierarchyRows`/`fetchRegionHierarchy` are added directly to `region_provider.dart` (or a small new sibling file), reusing the existing `apiProvider`/Dio client rather than a second, separate HTTP client instance. | Architecture Patterns > Recommended Project Structure | Very low risk — this mirrors `fetchRegionCatalog`'s existing shape exactly; only the file/function name is a judgment call, not the underlying design. |

## Open Questions

1. **Should the offline empty state (D-04) distinguish "never fetched successfully this session" from "fetched successfully before, but the app was cold-restarted offline"?**
   - What we know: D-04's copy ("Can't load regions... Connect to the internet") reads as a single, undifferentiated state.
   - What's unclear: Whether a disk-persisted hierarchy cache (so a previously-seen hierarchy survives a cold offline restart) is in scope, or whether "cached hierarchy shape" purely means "already fetched this session."
   - Recommendation: Treat as in-memory-only per Assumption A1 (matches D-03's "ephemeral... rebuilt from the API's flat list on each fetch" framing most literally) unless the user says otherwise during plan review.

2. **Exact chevron/group-row visual nesting when a leaf's existing bordered card sits directly under a group row with no border, per UI-SPEC's Color section ("tree structure itself... has no card/border").**
   - What we know: UI-SPEC is explicit that leaf cards keep their border/card treatment and group rows have none.
   - What's unclear: Whether there should be any subtle visual separator (e.g., a thin divider) between a group row and its first child leaf card, purely for scanability at deep nesting (depth 3-4) — UI-SPEC doesn't rule this out but also doesn't specify it, leaving it to "Claude's Discretion" per CONTEXT.md.
   - Recommendation: Default to no separator (simplest, matches UI-SPEC literally); a plan/verification step can add one later if on-device testing shows deep nesting is hard to scan.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK | All Dart/Flutter work in this phase | ✓ [VERIFIED: `flutter --version` in this environment] | 3.44.2 | — |
| Dart SDK (records language feature) | Pattern 1's `({RegionTreeNode node, int depth})` record type | ✓ [VERIFIED: `dart --version`] | 3.12.2 (records available since Dart 3.0) | — |
| `build_runner` (freezed/riverpod codegen) | `RegionHierarchyRow`'s generated `.freezed.dart`/`.g.dart` | ✓ [VERIFIED: `freezed`/`json_serializable` present in `app/pubspec.yaml` dev_dependencies] | freezed 3.2.5, json_serializable 6.13.0 | — |
| Go toolchain (`db/` module) | D-07's one-line `regions_get.go` change | Not independently probed in this session — assumed available given this repo's existing Go backend and prior phases (28-30) already compiled/migrated Go code successfully | — | — |

**Missing dependencies with no fallback:** None identified.

## Security Domain

`security_enforcement` is `true` in `.planning/config.json` (ASVS Level 1, block on `high`). This phase's surface area is almost entirely client-side rendering plus one read-only field addition to an already-authenticated endpoint.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No new auth surface — `GET /api/v1/regions` is already gated by `apis.RequireAuth()` at the route-group level (confirmed: `db/routes/regions_get.go`'s own doc comment, "Auth is enforced at the route-group level in main.go"); this phase adds one field to the existing response, not a new endpoint. |
| V3 Session Management | No | Unchanged — same session/cookie flow as every other authenticated Flutter API call via `apiProvider`. |
| V4 Access Control | No | `sort_order` is non-sensitive metadata (an integer ordering hint), already readable indirectly via the admin's PocketBase `/api/collections/regions/records` endpoint; no new access-control decision is introduced. |
| V5 Input Validation | Yes | The filter/search text field (`WandererSearchBar`) already exists and its input is used only for a client-side, non-executed substring match (`name.toLowerCase().contains(q)`) — no SQL/shell/template injection surface. No new validation is needed beyond what's already in place. |
| V6 Cryptography | No | Not applicable — no cryptographic material touched by this phase. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed/hostile catalog element crashing the client (e.g. a `regions` API row with an unexpected `kind` value or missing `parent`) | Denial of Service (client-side) | Already-established pattern in this codebase: per-element `try/catch`, skip-and-continue rather than abort-the-whole-parse (T-22-05 precedent) — extend the same posture to `parseRegionHierarchyRows`. |
| A region id used to construct a filesystem path (existing `region_file_path.dart` builders) | Tampering (path traversal) | Unchanged — this phase does not touch the download/delete path-building code at all; `assertValidRegionId`'s existing regex guard remains the sole gate, untouched by this phase's scope. |

## Sources

### Primary (HIGH confidence — direct file reads in this repository)
- `db/routes/regions_ext/regions_ui.html:500-767` — the exact `buildTree`/`computeDefaultExpanded`/`flattenVisible`/`computeFilterMatches` implementation and its Alpine.js wiring (fetch pagination, `$watch` filter debounce, `toggleExpand`), read in full.
- `db/routes/regions_get.go` (full file) — confirms `sort_order` is genuinely absent from the entry map (D-07 gap verified directly, not assumed) and that group rows carry only `id/name/kind/parent/path/depth`.
- `db/migrations/1785000000_create_regions_collection.go` (full file) — confirms the `sort_order` DB column exists (`NumberField`, line ~67) and the exact seed-time parent-linking algorithm (two-pass `Save()`, path-keyed).
- `app/lib/models/region_catalog_entry.dart`, `app/lib/entities/region_entity.dart`, `app/lib/provider/region/region_provider.dart`, `app/lib/util/region_disk_usage_util.dart`, `app/lib/routes/settings_offline_regions_screen.dart`, `app/lib/models/region_status.dart`, `app/lib/provider/region/tile_repository_provider.dart`, `app/lib/services/tile_repository_manager.dart` (relevant sections) — every file this phase modifies or must not regress, read directly.
- `app/lib/components/base/wanderer_searchbar.dart` — confirms the existing 500ms debounce, informing the Don't-Hand-Roll section.
- `app/lib/i18n/app_en.arb`, `app/l10n.yaml` — confirms exact ARB key set and non-standard `arb-dir` location.
- `app/pubspec.yaml` — confirms exact installed package versions (no registry lookup needed, nothing new installed).
- Local environment: `flutter --version` / `dart --version` — confirms Flutter 3.44.2 / Dart 3.12.2 available in this session.

### Secondary (MEDIUM confidence)
- None — this research required no external web lookups; the entire task is an internal-repo algorithm port plus a one-line backend change, and every claim above was verified by direct file inspection rather than training-data recall.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; every version number read directly from `pubspec.yaml`/local toolchain, not training data.
- Architecture: HIGH — the ported algorithm is read verbatim from the shipped admin page; the render-time-join pattern (Pattern 2) is a direct consequence of tracing `regionListNotifierProvider`'s actual invalidation call sites in the existing screen code, not a hypothesis.
- Pitfalls: HIGH for Pitfalls 1-6 (each traced to a specific, cited line of existing code); MEDIUM for Pitfall 7 (a general Flutter `ListView.builder` identity concern, well-established Flutter knowledge but not verified against this specific screen's behavior on-device).

**Research date:** 2026-07-27
**Valid until:** No external time pressure — this research is anchored entirely to code already in this repository (stable until the port itself is implemented); re-verify only if `regions_get.go`/`regions_ui.html`/the Flutter region files change again before this phase is planned.
