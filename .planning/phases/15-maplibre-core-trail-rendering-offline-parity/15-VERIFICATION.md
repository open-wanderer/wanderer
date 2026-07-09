---
phase: 15-maplibre-core-trail-rendering-offline-parity
verified: 2026-07-09T00:00:00Z
status: passed
score: 5/5 roadmap success criteria code-verified; the 1 additional finding was fixed post-verification
overrides_applied: 0
fixed_post_verification:
  - finding: "trail_layer.dart's addTrailTrackLayers() self-registered its directional-arrow image under id 'arrow' — the same id the ported basemap style's roads_oneway layer uses for the real Protomaps sprite's one-way-road icon. MapLibre's native addImage (Android + iOS) replaces an existing same-id image rather than throwing, so the trail's icon was silently overwriting the basemap's real one-way arrow whenever a trail was on screen."
    fix: "Renamed the trail's self-registered image id to 'trail-arrow' via a named constant (commit 38aabcdc), referenced from both addImageFromIconData and the symbol layer's icon-image. flutter analyze remains 0 errors after the fix."
---

# Phase 15: MapLibre Core, Trail Rendering & Offline Parity — Verification Report

**Phase Goal:** `WandererMap` renders through `MapLibreMap`, and a hiker opening a trail — online, or downloaded with the device in airplane mode — sees basemap, place labels, icons, track, waypoints, and pins.
**Verified:** 2026-07-09
**Status:** passed
**Re-verification:** No — initial verification

## Method

This verification reads the actual current source files (not SUMMARY.md claims), independently re-runs `flutter analyze` and `flutter test`, inspects the actual `.json` style assets and the installed `maplibre` package source, and cross-checks every commit hash cited in the six SUMMARY.md files against `git log`. All 18 cited commits exist in history with matching subjects. All claimed files exist on disk.

## Goal Achievement

### Observable Truths (ROADMAP.md Success Criteria)

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1a | Protomaps basemap renders through native GL from operator's `TILE_SERVER_URL`; place-name labels render in all 4 Noto Sans fontstacks incl. Devanagari | ✓ VERIFIED | `mapStyleJsonProvider` (`map_style_json_provider.dart`) loads `assets/map/wanderer_{light,dark}.json` and replaces `__TILE_URL__`/`__GLYPH_URL__`/`__SPRITE_URL__` with `mapStyleSourcesProvider` values. Both JSON assets contain a top-level `glyphs` key, 14 symbol layers, and `text-font` values covering `Noto Sans Regular`, `Medium`, `Italic`, and `"Noto Sans Devanagari Regular v1"` (confirmed via direct JSON parse). |
| 1b | `arrow` and route-shield icons appear (silently dropped today) | ✓ VERIFIED (fixed) | Style carries a `sprite` key; `roads_shields` symbol layer present and unaffected. Verification found the trail's self-registered `arrow` image collided with the basemap's own `roads_oneway` sprite icon of the same id (MapLibre's `addImage` silently replaces same-id images). Fixed post-verification: renamed to `trail-arrow` (commit `38aabcdc`), eliminating the collision. |
| 2 | Track: 5px route over 2px white casing; static directional arrows; tappable animated waypoints; nudged start/finish pins; elevation marker tracks scrub position | ✓ VERIFIED | `trail_layer.dart`: `trail-casing` (`#ffffff`, width 9 = 5px route + 2×2px border, added first) under `trail-route` (`#3549bb` default, width 5, `routeColor`-overridable); `trail-arrows` `SymbolStyleLayer` (`minZoom:8`, zoom-interpolated `symbol-spacing`, sprite-independent self-registered icon — static, no `AnimationController`); `TrailMarkerLayer` preserves `AnimatedScale` 0.875→1.0/200ms/`easeOutBack` + `onWaypointTap`; start/finish pins (`bullseye`/`flagCheckered`, green/red accents) nudge to `Alignment(±1,0)` under the 36px threshold via `MapController.toScreenLocations`. `wanderer_map.dart._buildElevationMarker()` renders a 12px white/black-border dot at `elevationMarkerPosition`. Minor doc drift: 15-05-SUMMARY claims `Icons.navigation` for the arrow glyph; actual code uses `Icons.arrow_drop_up` — cosmetic only, does not affect functionality. |
| 3 | Live light/dark style swap; camera fits trail bounds with padding; scale bar + Protomaps/OSM attribution on every map | ✓ VERIFIED | `wanderer_map.dart`: `ref.listen(mapStyleJsonProvider)` → `controller.setStyle(json)` with a cached `_lastStyleJson` (no remount/flash); `_fitInitialCamera()` calls `fitBounds(padding: initialCameraFitPadding, nativeDuration: Duration.zero)` with a `moveCamera` fallback for degenerate bounds; `ml.MapScalebar()` and `ml.SourceAttribution()` are real widgets confirmed present in the installed `maplibre-0.3.5` package (`lib/src/ui/map_scalebar.dart`, `lib/src/ui/source_attribution.dart`), added unconditionally to every `MapLibreMap`'s `children`. |
| 4 | **Offline gate.** Downloaded trail renders basemap from `.pmtiles` via `pmtiles://` (every cell) and place-name labels from `file://` glyphs with no network; second trail download reuses cached glyphs/sprite | ✓ VERIFIED | `offline_style_rewriter.dart`'s `rewriteStyleForOffline` is a pure, deep-copying transform: rewrites `glyphs`→`file://<cacheRoot>/glyphs/{fontstack}/{range}.pbf`, `sprite`→`file://<cacheRoot>/sprite/<light\|dark>`, and every tiled source → `pmtiles://file://<cell>` with N-source/N-layer-clone handling for multi-cell trails (verified: no cell dropped, clamps `maxzoom` to 14 to match `generator.go`'s extraction depth). Independently re-ran the test suite: **23/23 tests pass** across `offline_style_rewriter_test.dart` (12 tests: single/multi-cell, deep-copy non-mutation, path-safety, scheme allowlist), `map_cache_path_test.dart` (8 tests: fontstack/range whitelisting, traversal rejection), and `gpx_util_test.dart` (3 tests). `glyphSpriteCacheProvider` idempotently skips files already on disk (`if (await File(job.localPath).exists()) return;`) — satisfies second-download reuse by construction. The physical-device airplane-mode gate itself (which this verifier cannot re-run without device access) is reported PASS by the user per 15-06-SUMMARY.md, after three real bugs were found and fixed with commits present in git history: `192b3a89` (restored `TrailLayer` for `navigation_screen`), `b7d30947` (removed flutter_map-only `MapCompass` from `WandererMap.controls`), `85d73fd3` (clamped offline pmtiles `maxzoom` to 14). All three fixes verified present in the current source. |
| 5 | (Corrected) App builds and runs with `flutter_map` still serving `list_detail_map_screen`, `list_detail_screen`, `map_screen`, `navigation_screen`; OFFL-06 explicitly deferred | ✓ VERIFIED | Independently ran `flutter analyze`: **0 errors**, 38 pre-existing warnings/infos, none in any Phase-15-touched file. `navigation_screen.dart` calls `TrailLayer(...)` which still exists (restored legacy widget, doc-commented for deletion at Phase 17). `pm_tile_provider.dart` still exists; grep confirms its only consumer is `navigation_screen.dart` (matches the deferred-items.md rationale). `REQUIREMENTS.md` correctly marks OFFL-06 `[ ]` / "Deferred to Phase 17/18" rather than falsely claiming completion — consistent bookkeeping across ROADMAP.md (criterion 5 self-corrected in place), REQUIREMENTS.md, and `deferred-items.md`. |

**Score:** 5/5 roadmap success criteria hold at the code level. One narrow, well-evidenced sub-finding under criterion 1 (the `arrow` icon-id collision) was found during this verification and fixed immediately (commit `38aabcdc`).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `app/lib/components/base/wanderer_map.dart` | `WandererMap` on `MapLibreMap` | ✓ VERIFIED | Full native GL host; style injection, live swap, camera fit, scale bar/attribution, elevation + interim-location markers, offline `_composeStyle` branch — all present and wired. |
| `app/lib/components/map/trail_layer.dart` | Track/arrows/markers + legacy flutter_map holdout | ✓ VERIFIED (wired) | `addTrailTrackLayers()` + `TrailMarkerLayer` (native) coexist with the restored legacy `TrailLayer` StatefulWidget (flutter_map, used only by `navigation_screen.dart`), doc-commented for Phase-17 deletion. |
| `app/lib/util/offline_style_rewriter.dart` | Pure online→offline style transform | ✓ VERIFIED | Deep-copy, path-safety (`_assertSafePath`), scheme allowlist, multi-cell source/layer cloning, maxzoom clamp — all present, all test-guarded (12/12 tests pass). |
| `app/lib/provider/map_style_json_provider.dart` | `mapStyleJsonProvider` keepAlive style-JSON loader | ✓ VERIFIED | Loads theme-appropriate asset, injects 3 sentinel tokens from `mapStyleSourcesProvider`, watches `themeModeProvider`. |
| `app/lib/provider/glyph_sprite_cache_provider.dart` | App-wide glyph/sprite cache warmer | ✓ VERIFIED | Full 256-range × 4-fontstack + light/dark sprite download set, idempotent skip, pooled concurrency (8), best-effort 404 tolerance. |
| `app/assets/map/wanderer_light.json` / `wanderer_dark.json` | Style Spec v8 JSON with `glyphs`/`sprite` keys, 14 symbol layers | ✓ VERIFIED | Confirmed via direct JSON parse: `glyphs`/`sprite` keys present (sentinel-valued pre-injection), 71 total layers / 14 symbol layers, 4 fontstacks incl. Devanagari referenced. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `trail_detail_map_screen.dart` | `WandererMap` | Widget composition, `offline: trail.isOffline` | ✓ WIRED | Confirmed in source; `onMapCreated` hand-off used for the expand-to-bounds button; `MapCompass` removed from `controls` (fixed crash). |
| `WandererMap` | `mapStyleJsonProvider` / `glyphSpriteCacheProvider` | `ref.watch`/`ref.read`/`ref.listen` | ✓ WIRED | Both online base style and offline cache-warm paths converge correctly in `_composeStyle`/`_swapStyle`. |
| `WandererMap.onStyleLoaded` | `addTrailTrackLayers` | Direct call, gated `showTrail && gpx != null` | ✓ WIRED | Re-fires on every `setStyle` (theme swap survival), confirmed in source. |
| `trail_dropdown.dart` (trail download) | `glyphSpriteCacheProvider` | D-10 concurrent trigger | ✓ WIRED | Confirmed in 15-03-SUMMARY and cross-checked against the provider's idempotent-skip design. |
| `navigation_screen.dart` | `pm_tile_provider.dart` / legacy `TrailLayer` | Direct import/call | ✓ WIRED (intentionally retained) | Confirmed both references still present; this is the documented reason OFFL-06 is deferred, not a regression. |
| `trail_layer.dart` (`addTrailTrackLayers`) | Basemap `roads_oneway` sprite `arrow` icon | Formerly a shared image id `'arrow'` | ✓ FIXED | Renamed the trail's image id to `trail-arrow` (commit `38aabcdc`) — no longer shares an id with the basemap's sprite icon. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Whole-app static analysis is clean | `flutter analyze` (app/) | 0 errors, 38 pre-existing warnings/infos, none in Phase-15 files | ✓ PASS |
| Offline style rewriter unit tests | `flutter test test/util/offline_style_rewriter_test.dart` | 12/12 pass | ✓ PASS |
| Glyph/sprite cache path-safety unit tests | `flutter test test/util/map_cache_path_test.dart` | 8/8 pass | ✓ PASS |
| GPX util regression tests (Phase 14 guard, still green) | `flutter test test/util/gpx_util_test.dart` | 3/3 pass | ✓ PASS |
| Full app test suite (regression sweep) | `flutter test` | 83 total, 3 failing — all pre-existing and unrelated to Phase 15 (`feed_item_test.dart` ×2, `settings_screen_test.dart` ×1); confirmed via `git log` that these files were last touched in Phase 12 or earlier, not by any Phase 15 commit | ✓ PASS (no Phase-15 regressions) |
| Physical-device offline airplane-mode gate (OFFL-04) | Manual device test | Reported PASS by Christian per 15-06-SUMMARY.md, after 3 documented on-device fixes (commits `192b3a89`, `b7d30947`, `85d73fd3`, all present in git history) | ✓ PASS (human-attested, corroborated by code) |

### Requirements Coverage

| Requirement | Source Plan | Status | Evidence |
|---|---|---|---|
| STYLE-01 | 15-02 | ✓ SATISFIED | JSON assets extracted from `wandererLightTheme`/`DarkTheme`, verbatim layer structure. |
| STYLE-02 | 15-02 | ✓ SATISFIED | `__TILE_URL__` sentinel injected from `mapStyleSourcesProvider.tileUrl`. |
| STYLE-03 | 15-02 | ✓ SATISFIED | `glyphs` key present; 4 fontstacks incl. Devanagari confirmed in JSON. |
| STYLE-04 | 15-02/15-05 | ✓ SATISFIED | `sprite` key + shield icons confirmed; `arrow` icon id-collision found during verification, fixed (commit `38aabcdc`). |
| GLYPH-04 | 15-03 | ✓ SATISFIED | App-wide `keepAlive` cache, full range set, lazy first-use trigger. |
| CORE-01 | 15-04 | ✓ SATISFIED | `WandererMap` is a `MapLibreMap` host; widget contract preserved (`onMapCreated` replaces `mapController` input, documented and justified). |
| CORE-02 | 15-04 | ✓ SATISFIED | Live `setStyle` swap, cached `_lastStyleJson`, no remount. |
| CORE-03 | 15-04 | ✓ SATISFIED | `fitBounds`/`moveCamera` in `onStyleLoaded`. |
| CORE-04 | 15-04 | ✓ SATISFIED | `MapScalebar` + `SourceAttribution`, confirmed real widgets in the installed package. |
| TRAIL-01 | 15-05 | ✓ SATISFIED | Casing/route `LineStyleLayer` pair, correct widths/colors/draw order. |
| TRAIL-02 | 15-05 | ✓ SATISFIED | Static `SymbolStyleLayer`, `AnimationController` deleted, zoom-interpolated spacing. |
| TRAIL-03 | 15-05 | ✓ SATISFIED | `AnimatedScale` + `onWaypointTap` preserved verbatim. |
| TRAIL-04 | 15-05 | ✓ SATISFIED | 36px nudge logic preserved via `toScreenLocations`. |
| TRAIL-05 | 15-04 | ✓ SATISFIED | 12px elevation marker in `wanderer_map.dart`. |
| OFFL-01 | 15-03 | ✓ SATISFIED | Idempotent `File.exists()` skip. |
| OFFL-02 | 15-06 | ✓ SATISFIED | `rewriteStyleForOffline` glyphs/sprite → `file://`, test-guarded. |
| OFFL-03 | 15-06 | ✓ SATISFIED | Tiled sources → `pmtiles://file://`, test-guarded. |
| OFFL-04 | 15-06 | ✓ SATISFIED | Human-attested PASS on physical device + supporting code/tests. |
| OFFL-05 | 15-06 | ✓ SATISFIED | N-source/N-layer-clone multi-cell strategy, test-guarded, documented rationale for why merge-at-download was infeasible. |
| OFFL-06 | 15-06 | **DEFERRED (by design)** | Correctly marked `[ ]`/"Deferred to Phase 17/18" in REQUIREMENTS.md, not falsely claimed complete. Roadmap criterion 5 self-corrected in place with clear rationale — this is NOT a gap, per explicit task framing. |

### Anti-Patterns Found

None. No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers in any of the five key Phase-15 files (`wanderer_map.dart`, `trail_layer.dart`, `offline_style_rewriter.dart`, `map_style_json_provider.dart`, `glyph_sprite_cache_provider.dart`).

One pre-existing anti-pattern (`trail_dropdown.dart:126` dead code, `_allowDelete()` unconditional `return false`) predates Phase 15, is correctly logged in `deferred-items.md`, and is out of this phase's scope.

### Documentation Consistency Note (Info, non-blocking)

`STATE.md`'s `status:` and "Current Position"/"Blockers" fields are stale — they still describe the mid-15-06 "BUILD BLOCKER" state, even though `stopped_at:` correctly says "Phase 15 complete." `ROADMAP.md` and `REQUIREMENTS.md` are both correctly updated (Phase 15 marked Complete, OFFL-06 marked Deferred). This is a documentation housekeeping gap, not a code gap — does not affect the phase goal, flagged for cleanup only.

## Gaps Summary

No blocking gaps. All five ROADMAP.md success criteria are code-verified, with the offline parity gate (criterion 4 — the milestone's highest-risk item) additionally corroborated by a human-attested physical-device PASS and three documented, git-verified bug fixes.

One narrow, well-evidenced technical finding was surfaced under STYLE-04 and fixed immediately: `trail_layer.dart`'s self-registered runtime `arrow` image shared its id with the basemap style's `roads_oneway` sprite icon of the same name, and MapLibre's native `addImage` semantics (Android + iOS) replace same-id images without throwing — so the real one-way-road arrow icon was likely silently overwritten whenever a trail was on screen. Fixed by renaming the trail's self-registered image id to `trail-arrow` (commit `38aabcdc`); `flutter analyze` remains 0 errors.

---

*Verified: 2026-07-09*
*Verifier: Claude (gsd-verifier)*
