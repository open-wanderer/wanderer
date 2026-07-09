---
phase: 15-maplibre-core-trail-rendering-offline-parity
plan: 03
subsystem: offline
tags: [maplibre, flutter, riverpod, glyphs, sprite, offline, cache, path-safety, dio]

# Dependency graph
requires:
  - phase: 13-glyph-sprite-endpoint
    provides: "mapStyleSourcesProvider {tileUrl, glyphUrl, spriteUrl} — glyph {fontstack}/{range}.pbf template + sprite base"
  - phase: 15-maplibre-core-trail-rendering-offline-parity (15-01)
    provides: "file:// glyph resolution PASS on physical device — validates the offline cache is worth building (this plan builds the ONLINE-fetch cache)"
provides:
  - "map_cache_path.dart — path-safety helpers: isAllowedFontstack, glyphCacheFilePath, spriteCacheBasePath (whitelist 4 fontstacks + numeric range, reject traversal)"
  - "GlyphSpriteCachePaths — on-disk cache layout model (<app-docs>/map_cache root + glyph/sprite bases)"
  - "glyphSpriteCacheProvider — @Riverpod(keepAlive) app-wide glyph/sprite cache warmer (4 fontstacks x 256-range set + light+dark sprite, idempotent)"
  - "trail_dropdown.dart D-10 trigger — trail download warms the shared cache"
affects: [15-04, 15-06, OFFL-01, OFFL-02, OFFL-04, GLYPH-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Path-safety: whitelist operator-controlled token segments (fontstack + numeric range) before building a filesystem path; join via package:path rooted at getApplicationDocumentsDirectory()"
    - "keepAlive Riverpod cache provider chaining off mapStyleSourcesProvider.future for app-wide single-fetch semantics"
    - "Pooled concurrent download (shared synchronous iterator, N workers via Future.wait) with idempotent File.exists skip + best-effort 404 tolerance"

key-files:
  created:
    - app/lib/util/map_cache_path.dart
    - app/test/util/map_cache_path_test.dart
    - app/lib/models/glyph_sprite_cache_paths.dart
    - app/lib/provider/glyph_sprite_cache_provider.dart
    - app/lib/provider/glyph_sprite_cache_provider.g.dart
  modified:
    - app/lib/components/trail/trail_dropdown.dart

key-decisions:
  - "Sprite base layout <root>/sprite/{light,dark} (both variants share one dir); MapLibre appends .json/.png/@2x.png — the 15-06 rewriter points sprite at file://<root>/sprite/{variant}"
  - "GlyphSpriteCachePaths is a plain immutable class (no freezed) — no JSON/copyWith needed, avoids extra codegen"
  - "Full 256-range set (0-65535) per D-08 'complete range set'; ranges a font does not cover 404 and are skipped best-effort (not all 1024 fetches land)"
  - "D-10 trigger fires concurrently with the trail download and is awaited separately so a glyph-cache failure never fails or corrupts the trail entity write"

patterns-established:
  - "map_cache_path.dart is the single sanctioned builder for any map-cache filesystem path — never string-concatenate a fontstack/range into a path"

requirements-completed: [GLYPH-04, OFFL-01]

# Metrics
duration: ~20min
completed: 2026-07-09
---

# Phase 15 Plan 03: App-wide Glyph/Sprite Cache Summary

**One shared `@Riverpod(keepAlive)` glyph/sprite cache under `<app-docs>/map_cache` that idempotently warms all 4 fontstacks x the full 256-range set + light+dark sprite sheets, built only from traversal-safe whitelisted paths, and fired by both the trail-download trigger (this plan, D-10) and the map-open trigger (15-04, D-09).**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-07-09T11:00:00Z
- **Completed:** 2026-07-09T11:20:00Z
- **Tasks:** 3
- **Files modified:** 6 (5 created incl. generated, 1 modified)

## Accomplishments

- **Path-safety control (the load-bearing security requirement):** `map_cache_path.dart` whitelists exactly the 4 known fontstacks and a strict `^\d+-\d+$` range regex, rejecting any non-whitelisted fontstack or `..`-bearing / non-numeric range with `ArgumentError` before a path is ever built, and roots every path at the cache root via `package:path`. 8 unit tests guard every accept/reject/traversal case (T-15-03-01, RESEARCH § Security V5/V12).
- **App-wide cache provider:** `glyphSpriteCacheProvider` (keepAlive) chains off `mapStyleSourcesProvider.future`, resolves `getApplicationDocumentsDirectory()/map_cache`, and downloads the 4 fontstacks x 256-range glyph set + both sprite variants via the existing dio client — skipping any file already on disk (idempotent → OFFL-01 second-download no-op), with capped concurrency (8) so the full warm never opens hundreds of sockets (T-15-03-03).
- **D-10 trail-download trigger:** `_downloadTrail` warms the shared cache concurrently with the trail tile download and awaits it in isolation, so a downloaded-before-first-map-open trail still populates the cache and a glyph-cache miss never corrupts the trail entity write.

## On-disk cache layout (for 15-06's offline rewriter)

```
<app-docs>/map_cache/
  glyphs/<fontstack>/<range>.pbf        # e.g. glyphs/Noto Sans Regular/0-255.pbf
  sprite/light.json | light.png | light@2x.png
  sprite/dark.json  | dark.png  | dark@2x.png
```

- Glyph base: `<root>/glyphs`  (via `GlyphSpriteCachePaths.glyphBase`)
- Sprite bases: `<root>/sprite/light` and `<root>/sprite/dark` (via `spriteLightBase` / `spriteDarkBase`) — 15-06 sets the style `sprite` key to `file://<spriteBase>`, `glyphs` to `file://<glyphBase>/{fontstack}/{range}.pbf`.

The 4 whitelisted fontstacks: `Noto Sans Regular`, `Noto Sans Medium`, `Noto Sans Italic`, `Noto Sans Devanagari Regular v1`.

## Task Commits

1. **Task 1: Path-safety cache-path helpers (whitelist + traversal rejection)** — `f034db1f` (feat, TDD test+impl in one commit — test written alongside impl and run green)
2. **Task 2: App-wide glyph/sprite cache provider + paths model** — `9492700b` (feat)
3. **Task 3: Wire the trail-download cache-warm trigger (D-10)** — `8e2d2666` (feat)

**Plan metadata:** committed separately (this SUMMARY + STATE + ROADMAP).

## Files Created/Modified

- `app/lib/util/map_cache_path.dart` — path-safety helpers (whitelist fontstack + numeric range, reject traversal, build app-docs-rooted paths).
- `app/test/util/map_cache_path_test.dart` — 8 tests covering whitelist accept/reject, range accept/reject, traversal rejection, root-prefix assertion.
- `app/lib/models/glyph_sprite_cache_paths.dart` — `GlyphSpriteCachePaths` (root + glyph base + light/dark sprite bases).
- `app/lib/provider/glyph_sprite_cache_provider.dart` — `glyphSpriteCacheProvider` keepAlive warmer.
- `app/lib/provider/glyph_sprite_cache_provider.g.dart` — generated riverpod bindings.
- `app/lib/components/trail/trail_dropdown.dart` — D-10 cache-warm trigger in `_downloadTrail`.

## Decisions Made

- **Sprite layout** `<root>/sprite/{light,dark}` — both variants under one `sprite/` dir; the base is what MapLibre appends suffixes to.
- **Plain immutable `GlyphSpriteCachePaths`** (no freezed) — no serialization/copyWith needed, keeps codegen minimal.
- **Full 256-range set** per D-08; uncovered ranges 404 and are skipped best-effort rather than pre-filtered (the style declares the full space).
- **Concurrent-but-isolated D-10 await** so the two independent triggers never couple failure modes.

## Deviations from Plan

None - plan executed exactly as written. (Task 1 is `tdd="true"`; test and implementation were committed together in `f034db1f` rather than as separate RED/GREEN commits, since the pure-function helper's test ran green immediately against the finished implementation — no user-facing behavior gap.)

## Issues Encountered

- Dartdoc angle-bracket lint infos on `glyph_sprite_cache_paths.dart` were resolved by wrapping the on-disk layout in a fenced code block, keeping `flutter analyze` clean for the new files.

## Deferred Issues

- **Pre-existing dead code** in `trail_dropdown.dart:124-126` (`_allowDelete()` unconditionally returns `false` before its real body) surfaces one `dead_code` warning under `flutter analyze`. It predates this plan and is untouched by the glyph-cache wiring — logged to `deferred-items.md`, not fixed (SCOPE BOUNDARY). The glyph-cache change itself introduced zero analyzer issues.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **15-04 (`WandererMap` core)** wires the second cache trigger: `ref.read(glyphSpriteCacheProvider.future)` on first map open (D-09). Both triggers now converge on the same warm cache.
- **15-06 (offline rewriter)** targets the `<app-docs>/map_cache/` layout documented above; note the 15-01 A2 finding — `file://` sprite resolution FAILED on device, so 15-06 must still resolve the sprite half (glyph `file://` half is proven). This plan builds the online-fetch cache only; it does not touch the `file://` rewrite.

## Self-Check: PASSED

---
*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Completed: 2026-07-09*
