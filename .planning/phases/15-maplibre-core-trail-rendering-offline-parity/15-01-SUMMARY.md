---
phase: 15-maplibre-core-trail-rendering-offline-parity
plan: 01
subsystem: ui
tags: [maplibre, flutter, offline, glyphs, sprite, file-uri, spike, risk-gate]

# Dependency graph
requires:
  - phase: 13-glyph-sprite-endpoint
    provides: "mapStyleSourcesProvider {tileUrl, glyphUrl, spriteUrl} — glyph {fontstack}/{range}.pbf template + sprite base"
  - phase: 14-coordinate-type-migration
    provides: "Geographic/LngLatBounds coordinate vocabulary used by MapOptions.initCenter"
provides:
  - "Throwaway SpikeGlyphFileScreen — minimal MapLibreMap over a hand-built file:// glyph + sprite style"
  - "seedSpikeGlyphCache() — one-off online pre-seed of glyph ranges + sprite files into <app-docs>/spike_glyphs"
  - "Debug-only FAB entry point (kDebugMode) in main.dart to reach the spike"
  - "PENDING: physical-device airplane-mode PASS/FAIL verdict for file:// glyph (A1) + sprite (A2) resolution"
affects: [15-02, 15-03, 15-04, 15-05, 15-06, OFFL-02, OFFL-04, GLYPH-04]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hand-built minimal MapLibre Style Spec v8 JSON passed as a raw string to MapOptions.initStyle"
    - "file://<app-docs>/{fontstack}/{range}.pbf glyph template with literal tokens for native runtime substitution"
    - "Idempotent app-docs pre-seed via existing dio api client, mirroring trail_download_service.dart"

key-files:
  created:
    - app/lib/routes/spike_glyph_file_screen.dart
    - app/lib/util/spike_glyph_seed.dart
  modified:
    - app/lib/main.dart

key-decisions:
  - "Embed source + layers directly in the style JSON string (not Dart-side GeoJsonSource/SymbolStyleLayer) — the gate tests native resolution of a raw style doc"
  - "Use the real Protomaps v4 'arrow' sprite icon for the A2 icon-image test (verified present in sprites/v4/light.json) rather than guessing a name"
  - "Seed the light sprite variant (spriteUrl base + '/light') best-effort so an A2 sprite miss never aborts the primary A1 glyph gate"

patterns-established:
  - "Throwaway spike files carry a `// SPIKE 15-01 — THROWAWAY` header and a debug-only kDebugMode entry point"

requirements-completed: []  # OFFL-04 NOT complete — gated on Task 2 physical-device verdict

# Metrics
duration: ~15min
completed: PENDING (Task 2 awaiting physical-device verification)
---

# Phase 15 Plan 01: file:// Glyph Risk-Gate Spike Summary

**INCOMPLETE / PENDING VERIFICATION — Task 1 (build) done; Task 2 (physical-device airplane-mode PASS/FAIL) is a blocking human gate that has NOT run.**

A throwaway MapLibreMap spike screen renders a place-name label (A1) and an `arrow` sprite icon (A2) from a hand-built minimal style whose `glyphs` template and `sprite` base are `file://` URLs under `<app-docs>/spike_glyphs`, pre-seeded online by `seedSpikeGlyphCache()`. Whether MapLibre GL Native actually resolves those `file://` URLs offline on a real device is the milestone's highest-risk unknown (OFFL-04) and is **not yet known**.

## Status

- ✅ **Task 1 — build the spike screen + seed helper + debug entry** — DONE, committed `d713456b`.
- ⏸️ **Task 2 — physical-device airplane-mode verification (Christian runs)** — BLOCKING HUMAN GATE, not executed. Claude cannot access a physical device (D-01/D-02). The plan is **not complete** and no downstream Phase 15 plan (15-02..15-06) is cleared to run until this returns PASS.

## Performance

- **Duration:** ~15 min (Task 1 only)
- **Tasks:** 1 of 2 complete (Task 2 is a human-only gate)
- **Files modified:** 3

## Accomplishments (Task 1)

- `seedSpikeGlyphCache()` resolves `getApplicationDocumentsDirectory()`, creates `<app-docs>/spike_glyphs/Noto Sans Regular/`, reads `mapStyleSourcesProvider` for `glyphUrl`/`spriteUrl`, and (while online) fetches glyph ranges `0-255.pbf` + `256-511.pbf` plus `sprite.json`/`sprite.png`/`sprite@2x.png` via the existing dio client. Idempotent — skips files already on disk (satisfies the OFFL-01-style reuse contract by construction).
- `SpikeGlyphFileScreen` (ConsumerStatefulWidget) seeds on `initState`, then hand-builds a MINIMAL style JSON: one GeoJSON point with a `name`, `"glyphs": "file://<docs>/spike_glyphs/{fontstack}/{range}.pbf"`, `"sprite": "file://<docs>/spike_glyphs/sprite"`, and two `symbol` layers — a `text-field` label using `text-font: ["Noto Sans Regular"]` (A1) and an `icon-image: "arrow"` layer (A2). An on-screen panel shows the seed log + cache path so Christian can confirm the files landed before enabling airplane mode.
- Debug-only FAB in `main.dart` (behind `kDebugMode`) pushes the spike screen via `navigatorKey` — reachable without a real trail.
- Every spike file carries the `// SPIKE 15-01 — THROWAWAY` marker.

## Task Commits

1. **Task 1: build the throwaway file:// glyph + sprite spike screen and seed helper** — `d713456b` (feat)

**Plan metadata:** pending (this SUMMARY + STATE update committed separately; ROADMAP plan-progress deliberately NOT marked complete).

## Files Created/Modified

- `app/lib/routes/spike_glyph_file_screen.dart` — minimal MapLibreMap spike over a file:// glyph/sprite style with label + icon layers.
- `app/lib/util/spike_glyph_seed.dart` — one-off online pre-seed of glyph ranges + sprite files into `<app-docs>/spike_glyphs`.
- `app/lib/main.dart` — debug-only `kDebugMode` FAB entry point to the spike screen.

## Verification

- `flutter analyze lib/routes/spike_glyph_file_screen.dart lib/util/spike_glyph_seed.dart lib/main.dart` → **No issues found.**
- Style JSON contains a top-level `glyphs` key with a `file://` value and a `symbol` layer with `text-font: ["Noto Sans Regular"]`. ✅
- `seedSpikeGlyphCache()` fetches `Noto Sans Regular/0-255.pbf` (+ `256-511.pbf`) and the sprite files into `<app-docs>/spike_glyphs/`, idempotent. ✅
- Debug-only path to open `SpikeGlyphFileScreen` exists. ✅
- **Human gate (Task 2): NOT RUN** — requires a physical device in airplane mode (D-01/D-02). Verdict unknown.

## Decisions Made

- Source + layers embedded directly in the raw style JSON string (not Dart-side `GeoJsonSource`/`SymbolStyleLayer`), because the gate is specifically about MapLibre GL Native resolving `file://` URL fields inside a raw style document.
- Chose the real `arrow` icon (confirmed in `sprites/v4/light.json`) for the A2 `icon-image` test to avoid a misleading A2 FAIL from a guessed icon name.
- Sprite seeding is best-effort (`<spriteUrl>/light.*`) so any sprite miss cannot abort the primary A1 glyph gate.

## Deviations from Plan

None — Task 1 executed exactly as written. Task 2 is intentionally halted per its `gate="blocking-human"` disposition (D-01/D-02/D-03).

## Issues Encountered

None during Task 1.

## Next Phase Readiness

**BLOCKED on the human gate.** Nothing downstream is cleared:

- **On PASS** (label + arrow render offline on a physical device in airplane mode): plans 15-02..15-06 are unblocked; OFFL-02 (`file://` rewrite) and OFFL-04 are safe to build. Mark OFFL-04 progressing and delete the three spike artifacts as later plans land the real cache.
- **On FAIL** (label blank / tofu / style-load error / only worked because it silently reached the network): capture the native resource-load error (Android `adb logcat` — glyph/resource 404 or "unsupported scheme" — or the iOS device console) and record it here verbatim. Per **D-03**, do NOT pick a fallback direction; the phase STOPS and the offline-label strategy is re-decided (roadmap revision per ROADMAP.md "Risk gate") before any further Phase 15 execution.

## Awaiting Resume Signal

Christian to reply **"PASS"** (glyphs + sprite render offline) or **"FAIL: &lt;captured native log&gt;"** after running the debug build on a PHYSICAL device in airplane mode per the plan's `<how-to-verify>` steps.

## Self-Check: PASSED

- FOUND: app/lib/routes/spike_glyph_file_screen.dart
- FOUND: app/lib/util/spike_glyph_seed.dart
- FOUND: .planning/phases/15-maplibre-core-trail-rendering-offline-parity/15-01-SUMMARY.md
- FOUND: commit d713456b (Task 1)

_Note: this self-check confirms Task 1 artifacts only. Task 2 (physical-device airplane-mode PASS/FAIL) remains an open human gate — the plan is intentionally NOT marked complete._

---
*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Status: Task 1 complete, Task 2 pending physical-device verification*
