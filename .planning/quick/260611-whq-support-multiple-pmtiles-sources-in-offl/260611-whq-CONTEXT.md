# Quick Task 260611-whq: Support multiple pmtiles sources in offline map rendering - Context

**Gathered:** 2026-06-11
**Status:** Ready for planning

<domain>
## Task Boundary

Fix the offline map rendering in `WandererMap` to load and use all pmtiles archives stored in `trail.pmTiles` (a `List<String>`), rather than only the first entry (`pmTiles[0]`). The download service splits the trail bounding box into multiple cells and saves each as a separate `.pmtiles` file — the map component must query all of them to render correctly.

</domain>

<decisions>
## Implementation Decisions

### Multi-archive tile lookup strategy
- Sequential lookup: try each `PmTilesArchive` in order; return the first tile found; throw `ProviderException` only if no archive contains the tile.

### Scope
- Only `wanderer_map.dart` is in scope. `wanderer_offline_map.dart` has the same bug but is explicitly excluded from this task.

### Zoom range with multiple archives
- Union strategy: `minimumZoom` = min across all archives; `maximumZoom` = max across all archives.

### Claude's Discretion
- Where to add the multi-archive provider class (new file vs. extending `pm_tile_provider.dart`).
- Error handling details beyond the `ProviderException` convention already established.

</decisions>

<specifics>
## Specific Ideas

- A new `MultiPmTilesVectorTileProvider` class that holds a `List<PmTilesArchive>` and implements `provide(TileIdentity tile)` by iterating archives sequentially until one succeeds.
- `_initOffline()` in `_WandererMapState` should open all archives in `widget.trail.pmTiles` in parallel (`Future.wait`) and construct the multi-provider.
- The state field `_offlineTileProvider` may need to change type from `PmTilesVectorTileProvider?` to the new multi-provider type (or a shared base type).

</specifics>

<canonical_refs>
## Canonical References

- `app/lib/vendor/vector_map_tiles/pm_tile_provider.dart` — existing single-archive provider; new class should live nearby or extend the same file.
- `app/lib/components/base/wanderer_map.dart` — only file to be modified in the map component layer.
- `app/lib/services/trail_download_service.dart` — produces `List<String> cellPaths`; read-only reference, no changes needed.

</canonical_refs>
