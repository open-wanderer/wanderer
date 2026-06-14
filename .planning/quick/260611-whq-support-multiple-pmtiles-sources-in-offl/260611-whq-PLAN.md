---
phase: quick-260611-whq
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/vendor/vector_map_tiles/pm_tile_provider.dart
  - app/lib/components/base/wanderer_map.dart
autonomous: true
requirements: [WHQ-01]
must_haves:
  truths:
    - "Offline map renders tiles from every archive in trail.pmTiles, not just pmTiles[0]"
    - "A tile present in any archive is returned; only when no archive has it is a 404 ProviderException thrown"
    - "Offline map still loads when trail.pmTiles has exactly one entry (no regression)"
  artifacts:
    - path: "app/lib/vendor/vector_map_tiles/pm_tile_provider.dart"
      provides: "MultiPmTilesVectorTileProvider that fans tile requests across a List<PmTilesArchive>"
      contains: "class MultiPmTilesVectorTileProvider"
    - path: "app/lib/components/base/wanderer_map.dart"
      provides: "_initOffline opens all pmTiles archives in parallel and builds the multi-provider"
      contains: "Future.wait"
  key_links:
    - from: "app/lib/components/base/wanderer_map.dart"
      to: "MultiPmTilesVectorTileProvider"
      via: "_offlineTileProvider field + _buildTileLayer tileProviders"
      pattern: "MultiPmTilesVectorTileProvider"
---

<objective>
Fix offline map rendering in `WandererMap` so it loads and queries ALL pmtiles archives in `trail.pmTiles` (a `List<String>`), instead of only `pmTiles[0]`. The download service splits a trail's bounding box into multiple cells, saving each as a separate `.pmtiles` file; the map currently renders only the first cell, leaving the rest of the trail blank offline.

Purpose: Offline maps must cover the whole downloaded trail, matching what `trail_download_service.dart` actually produces.
Output: A new `MultiPmTilesVectorTileProvider` class and an updated `_WandererMapState` that opens every archive in parallel and renders from all of them.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/quick/260611-whq-support-multiple-pmtiles-sources-in-offl/260611-whq-CONTEXT.md

# Files to modify / reference
@app/lib/vendor/vector_map_tiles/pm_tile_provider.dart
@app/lib/components/base/wanderer_map.dart

# Read-only reference: produces List<String> cellPaths saved into trail.pmTiles
@app/lib/services/trail_download_service.dart
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add MultiPmTilesVectorTileProvider</name>
  <files>app/lib/vendor/vector_map_tiles/pm_tile_provider.dart</files>
  <action>
In `pm_tile_provider.dart`, add a new class `MultiPmTilesVectorTileProvider extends VectorTileProvider` alongside the existing `PmTilesVectorTileProvider` (per CONTEXT "Claude's Discretion": extending the same file). It holds a `final List<PmTilesArchive> archives` set via a primary constructor `MultiPmTilesVectorTileProvider.fromArchives(this.archives, {this.type = TileProviderType.vector})`.

Implement the required `VectorTileProvider` abstract members:
- `type` — the `TileProviderType` field (default `TileProviderType.vector`).
- `tileOffset` — return `TileOffset.DEFAULT`, matching `PmTilesVectorTileProvider`.
- `maximumZoom` — Union strategy per CONTEXT: `max` of `archive.maxZoom` across all `archives`. Use a reduce/fold over the list.
- `minimumZoom` — Union strategy per CONTEXT: `min` of `archive.minZoom` across all `archives`.
- `provide(TileIdentity tile)` — Sequential lookup per CONTEXT: compute `tileId` once via `ZXY(tile.z, tile.x, tile.y).toTileId()`, then iterate `archives` in order; for each, `await archive.tile(tileId)` inside a `try` and on success return `Uint8List.fromList(data.bytes())`; on `TileNotFoundException` continue to the next archive. After the loop (no archive had the tile), throw `ProviderException(message: 'Tile not found: $tile', retryable: Retryable.none, statusCode: 404)` — identical convention to the single-archive provider.

Also add a static async helper `static Future<MultiPmTilesVectorTileProvider> fromSources(List<String> sources, {TileProviderType type = TileProviderType.vector})` that opens every source in parallel via `Future.wait(sources.map((s) => PmTilesArchive.from(s)))` and returns `MultiPmTilesVectorTileProvider.fromArchives(archives, type: type)`. Guard against an empty `sources` list by throwing an `ArgumentError` so callers fail loudly rather than constructing a provider that always 404s.

Do not modify the existing `PmTilesVectorTileProvider` class. Keep imports unchanged (`dart:typed_data`, `pmtiles`, `vector_map_tiles`) — they already cover everything needed.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && dart analyze lib/vendor/vector_map_tiles/pm_tile_provider.dart</automated>
  </verify>
  <done>`MultiPmTilesVectorTileProvider` exists, implements all abstract `VectorTileProvider` members, does sequential lookup with union zoom range, has a `fromSources` parallel-open helper, and `dart analyze` reports no errors for the file.</done>
</task>

<task type="auto">
  <name>Task 2: Wire WandererMap to use all archives</name>
  <files>app/lib/components/base/wanderer_map.dart</files>
  <action>
Update `_WandererMapState` to render from every archive in `widget.trail.pmTiles` instead of `pmTiles[0]`.

1. Change the state field type from `PmTilesVectorTileProvider? _offlineTileProvider;` to `MultiPmTilesVectorTileProvider? _offlineTileProvider;` (the import for the provider file is already present).

2. In `_initOffline()`, replace the single `PmTilesVectorTileProvider.fromSource(widget.trail.pmTiles[0])` call with `MultiPmTilesVectorTileProvider.fromSources(widget.trail.pmTiles)` (this opens all archives in parallel via `Future.wait` per CONTEXT). Keep the existing `try/catch` that sets `_error` on failure and the `if (mounted) setState(...)` guards. The single-archive case still works because `fromSources` handles a list of length 1.

3. Leave `_buildTileLayer()` as-is — it already passes `_offlineTileProvider!` into `TileProviders({'protomaps': _offlineTileProvider!})`, and the new type satisfies the same `VectorTileProvider` contract. The `_ready` getter (`_offlineTileProvider != null`) also needs no change.

Do NOT touch `wanderer_offline_map.dart` — CONTEXT explicitly excludes it from this task even though it has the same bug.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && dart analyze lib/components/base/wanderer_map.dart && grep -q "MultiPmTilesVectorTileProvider.fromSources(widget.trail.pmTiles)" lib/components/base/wanderer_map.dart && ! grep -q "pmTiles\[0\]" lib/components/base/wanderer_map.dart</automated>
  </verify>
  <done>`_offlineTileProvider` is typed `MultiPmTilesVectorTileProvider?`, `_initOffline` calls `fromSources(widget.trail.pmTiles)`, no `pmTiles[0]` reference remains, `wanderer_offline_map.dart` is untouched, and `dart analyze` reports no errors.</done>
</task>

</tasks>

<verification>
- `dart analyze lib/vendor/vector_map_tiles/pm_tile_provider.dart lib/components/base/wanderer_map.dart` reports no errors.
- `grep -c "pmTiles\[0\]" app/lib/components/base/wanderer_map.dart` returns 0 (the single-archive shortcut is gone).
- `git diff --name-only` shows only `pm_tile_provider.dart` and `wanderer_map.dart` changed (no edits to `wanderer_offline_map.dart`).
</verification>

<success_criteria>
- Offline `WandererMap` builds a `MultiPmTilesVectorTileProvider` from every entry in `trail.pmTiles`.
- Tile requests are tried against each archive sequentially; first hit wins, all-miss yields a 404 `ProviderException`.
- Zoom range is the union (min of mins, max of maxes) across archives.
- A single-archive trail still renders (no regression).
- `wanderer_offline_map.dart` is unchanged.
</success_criteria>

<output>
Create `.planning/quick/260611-whq-support-multiple-pmtiles-sources-in-offl/260611-whq-SUMMARY.md` when done
</output>
