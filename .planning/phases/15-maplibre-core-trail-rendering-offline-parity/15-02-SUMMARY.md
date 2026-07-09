---
phase: 15-maplibre-core-trail-rendering-offline-parity
plan: 02
subsystem: ui
tags: [flutter, maplibre, riverpod, map-style, vector_tile_renderer, json-assets]

# Dependency graph
requires:
  - phase: 15-01
    provides: "Glyph file:// spike verdict (A1 PASS); mapStyleSourcesProvider + MapStyleSources model context"
  - phase: 13
    provides: "/map/style-sources endpoint returning operator tile/glyph/sprite URLs"
provides:
  - "assets/map/wanderer_{light,dark}.json — checked-in MapLibre Style Spec v8 assets with __TILE_URL__/__GLYPH_URL__/__SPRITE_URL__ sentinel tokens plus glyphs + sprite keys"
  - "tool/extract_map_styles.dart — one-off regeneration CLI"
  - "mapStyleJsonProvider — keepAlive Future<String> provider injecting operator URLs into the raw style String, re-running on theme change (enables CORE-02 in 15-04)"
affects: [15-04, 15-06, CORE-02, OFFL-02]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CLI dump-and-augment: import vector_tile_renderer theme src files directly (bypass FFI barrel) and overwrite tile/glyph/sprite endpoints with sentinel tokens"
    - "Runtime style String: rootBundle.loadString + lossless String.replaceAll of 3 unique sentinels; no jsonDecode/re-encode"

key-files:
  created:
    - app/tool/extract_map_styles.dart
    - app/assets/map/wanderer_light.json
    - app/assets/map/wanderer_dark.json
    - app/lib/provider/map_style_json_provider.dart
    - app/lib/provider/map_style_json_provider.g.dart
  modified:
    - app/pubspec.yaml

key-decisions:
  - "ADD mapStyleJsonProvider rather than change mapStyleProvider's return type, so the 4 flutter_map screens still on vtr.Style keep compiling (phase success criterion #5)"
  - "Import vector_tile_renderer theme src/ files directly in the CLI to avoid the FFI compile crash under plain `dart run`"
  - "Style assets are build inputs — regenerate via `dart run tool/extract_map_styles.dart` from app/ only if the flomp fork's wanderer themes change"

patterns-established:
  - "Sentinel-token style injection: assets carry __TILE_URL__/__GLYPH_URL__/__SPRITE_URL__ placeholders; runtime provider replaceAll-injects operator URLs from mapStyleSourcesProvider"

requirements-completed: [STYLE-01, STYLE-02, STYLE-03, STYLE-04]

# Metrics
duration: 12min
completed: 2026-07-09
---

# Phase 15 Plan 02: JSON Style Assets + mapStyleJsonProvider Summary

**Extracted the wanderer light/dark Dart themes into checked-in MapLibre Style Spec v8 JSON assets with glyphs/sprite keys and sentinel tile/glyph/sprite tokens, plus a keepAlive mapStyleJsonProvider that injects operator URLs into a raw style String and re-runs on theme change.**

## Performance

- **Duration:** ~12 min
- **Completed:** 2026-07-09
- **Tasks:** 2
- **Files modified:** 6 (5 created, 1 modified)

## Accomplishments
- `tool/extract_map_styles.dart` one-off CLI dumps both themes to JSON and overwrites the tile/glyph/sprite endpoints with sentinel placeholder tokens.
- Generated + committed `assets/map/wanderer_light.json` and `wanderer_dark.json` (version 8, 14 symbol layers each, all 4 Noto fontstacks preserved), registered under `flutter/assets`.
- `mapStyleJsonProvider` (`@Riverpod(keepAlive: true) Future<String>`) loads the theme-appropriate asset and losslessly injects `tileUrl`/`glyphUrl`/`spriteUrl` from `mapStyleSourcesProvider`, watching `themeModeProvider` so it re-runs on theme change.
- Legacy `mapStyleProvider` (returning `vtr.Style`) left untouched — the 4 not-yet-migrated flutter_map screens keep compiling.

## Task Commits

1. **Task 1: Extraction CLI + generated style assets** - `16a6b9a1` (feat)
2. **Task 2: mapStyleJsonProvider injecting tile/glyph/sprite URLs** - `fc343987` (feat)

## Files Created/Modified
- `app/tool/extract_map_styles.dart` - One-off CLI dumping themes to JSON with sentinel tokens + glyphs/sprite keys
- `app/assets/map/wanderer_light.json` - Generated light style asset (placeholders + glyphs + sprite)
- `app/assets/map/wanderer_dark.json` - Generated dark style asset
- `app/lib/provider/map_style_json_provider.dart` - keepAlive Future<String> provider injecting operator URLs
- `app/lib/provider/map_style_json_provider.g.dart` - Generated Riverpod part file
- `app/pubspec.yaml` - Registered `assets/map/` under `flutter/assets`

## Decisions Made
- **ADD, don't mutate:** New `mapStyleJsonProvider` sits alongside `mapStyleProvider`; the legacy `vtr.Style` path is preserved for `list_detail_map_screen.dart`, `list_detail_screen.dart`, `map_screen.dart`, `navigation_screen.dart`.
- **Lossless injection:** Plain `String.replaceAll` on the 3 unique sentinel tokens — no jsonDecode/re-encode, so layer structure is untouched.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Import theme src files directly to avoid FFI compile crash**
- **Found during:** Task 1 (running the extraction CLI)
- **Issue:** `dart run tool/extract_map_styles.dart` crashed the pure-Dart VM compiler (`type 'InvalidType' is not a subtype of type 'FunctionType'`) because importing the public `vector_tile_renderer.dart` barrel transitively pulls in the package's FFI/Flutter-engine code, which cannot compile without a Flutter engine.
- **Fix:** Import the two theme source files directly (`package:vector_tile_renderer/src/themes/wanderer/wanderer_{light,dark}_theme.dart`). These are self-contained `Map<String, dynamic>` literals with zero imports, so the CLI runs cleanly under `dart run`.
- **Files modified:** app/tool/extract_map_styles.dart
- **Verification:** CLI runs without error; both assets generate and parse.
- **Committed in:** `16a6b9a1` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to make the CLI runnable; no scope creep, no change to the generated output structure.

## Issues Encountered
- **Plan assumption drift:** The plan stated the Dart themes have NO `glyphs`/`sprite` keys, but the current flomp `vector_tile_renderer` fork already includes both (with real Protomaps URLs). The CLI's `map['glyphs'] = ...` / `map['sprite'] = ...` OVERWRITES these with the sentinel tokens rather than adding fresh keys, so the acceptance criteria (keys present with placeholder values) are satisfied either way. No code change was needed beyond noting the overwrite semantics in the CLI comment.

## Notes
- **Regeneration command:** from `app/`, run `dart run tool/extract_map_styles.dart`. Regenerate only if the flomp fork's wanderer themes change (build input, not runtime-generated).
- The offline branch (rewriting injected https URLs to `file://`/`pmtiles://`) is intentionally NOT built here — that is 15-06 (OFFL-02).

## Next Phase Readiness
- `mapStyleJsonProvider` is ready for 15-04's `WandererMap` rewrite (CORE-02) to consume as a raw style String.
- No blockers introduced. `flutter analyze` reports no errors (38 pre-existing info-level warnings in unrelated files remain out of scope).

## Self-Check: PASSED

- All 5 created files exist on disk.
- Both task commits (`16a6b9a1`, `fc343987`) present in git history.

---
*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Completed: 2026-07-09*
