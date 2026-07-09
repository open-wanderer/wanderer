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
  - "VERDICT (Android, physical device, 2026-07-09): A1 PASS — file:// glyph template resolves both online and offline, label renders correctly. A2 FAIL — file:// sprite does NOT resolve despite valid, correctly-sized cached files (sprite.json 3549B, sprite.png 16174B, sprite@2x.png 28852B) at the exact path the style references; icon fails to render both online and offline (the style always points sprite at file://, so this isn't a network-vs-offline distinction — it's file:// sprite resolution itself). Native resolves file:// for glyphs but apparently not for the sprite atlas — a narrower, distinct code path."

patterns-established:
  - "Throwaway spike files carry a `// SPIKE 15-01 — THROWAWAY` header and a debug-only kDebugMode entry point"

requirements-completed: []  # OFFL-04 progressing — glyph half proven; sprite half needs 15-06 follow-up

# Metrics
duration: ~15min
completed: 2026-07-09
---

# Phase 15 Plan 01: file:// Glyph Risk-Gate Spike Summary

**COMPLETE — PARTIAL PASS.** Task 1 (build) done. Task 2 (physical-device airplane-mode verification) ran: **A1 (glyphs) PASSED**, **A2 (sprite) FAILED**.

A throwaway MapLibreMap spike screen renders a place-name label (A1) and an `arrow` sprite icon (A2) from a hand-built minimal style whose `glyphs` template and `sprite` base are `file://` URLs under `<app-docs>/spike_glyphs`, pre-seeded online by `seedSpikeGlyphCache()`.

**The milestone's primary risk gate is resolved: MapLibre GL Native DOES resolve `file://` glyph URL templates**, both online and in airplane mode on a physical Android device — the "Wanderer Spike" label renders correctly in both states. This unblocks Phase 15 waves 2-5 (15-02..15-05), none of which touch offline sprite resolution.

**A secondary, narrower gap was found: the `arrow` sprite icon never rendered, online or offline**, despite `seedSpikeGlyphCache()` successfully caching valid, correctly-sized files at exactly the `file://<cache>/sprite.{json,png}` path the style references (confirmed via `adb shell run-as ... ls -la`: `sprite.json` 3549B, `sprite.png` 16174B, `sprite@2x.png` 28852B — not corrupted, not empty). Because the style always points `sprite` at `file://` (there's no live-network variant in this spike), the online/offline distinction doesn't apply to A2 the way it does to A1 — this is a **file:// sprite resolution failure**, not a connectivity issue. Since production online rendering uses `https://` sprite URLs (not `file://`), STYLE-04/online icon rendering is unaffected — this only threatens the OFFL-02 offline rewrite's sprite half. **Plan 15-06 (OFFL-02/03/04/05) must investigate this before closing OFFL-02** — see "Next Phase Readiness" below.

## Status

- ✅ **Task 1 — build the spike screen + seed helper + debug entry** — DONE, committed `d713456b`.
- ✅ **Task 2 — physical-device airplane-mode verification (Christian ran it, Android)** — A1 PASS, A2 FAIL. See verdict above.

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
- **Human gate (Task 2): RUN on a physical Android device, airplane mode.** A1 (glyph label) PASS. A2 (sprite icon) FAIL — see verdict above. A build/wiring bug was found and fixed during verification (see Deviations).

## Decisions Made

- Source + layers embedded directly in the raw style JSON string (not Dart-side `GeoJsonSource`/`SymbolStyleLayer`), because the gate is specifically about MapLibre GL Native resolving `file://` URL fields inside a raw style document.
- Chose the real `arrow` icon (confirmed in `sprites/v4/light.json`) for the A2 `icon-image` test to avoid a misleading A2 FAIL from a guessed icon name.
- Sprite seeding is best-effort (`<spriteUrl>/light.*`) so any sprite miss cannot abort the primary A1 glyph gate.

## Deviations from Plan

- A real bug was found and fixed during verification, outside the original plan scope: the debug-only FAB in `main.dart` had a `tooltip:` property that crashed on first frame with "No Overlay widget found" — the FAB lives in `MaterialApp.router`'s `builder`, which sits above the router's internal `Navigator`/`Overlay`, and `RawTooltip`'s `build()` unconditionally asserts an `Overlay` ancestor. Fixed by removing the `tooltip:` property (commit `d0d3920f`). This was necessary for Christian to reach the spike screen at all; without it the app crashed before the spike could even be tested.
- A2 (sprite `file://` resolution) did not pass, contrary to the "ideally sprite" aspiration in RESEARCH.md's Spike Design section. This is a real, documented finding, not a plan deviation — see verdict above and "Next Phase Readiness."

## Issues Encountered

- `main.dart`'s debug FAB tooltip crash (see Deviations) — fixed inline before the human gate could run.
- A2 sprite `file://` resolution failure — files verified valid and present via `adb shell run-as ... ls -la` (not a download/corruption issue); root cause not yet diagnosed (native log not yet captured — no error was thrown, the icon layer just silently renders nothing). Deferred to 15-06.

## Next Phase Readiness

**UNBLOCKED for waves 2-5.** The primary risk gate (A1, glyph labels via `file://`) passed — plans 15-02 through 15-05 (style extraction, glyph cache, `WandererMap` core rewrite, trail/marker rendering) do not depend on offline sprite resolution and are clear to execute.

**Plan 15-06 (OFFL-02/03/04/05) has an added investigation item:** before closing OFFL-02 (the offline style rewriter that points `sprite` at `file://`), 15-06 must either (a) find and fix the root cause of the A2 sprite `file://` failure — candidates to investigate: MapLibre Native's sprite loader may require a different scheme/format than the glyph loader (e.g. an internal HTTP(S)-only fast path for the combined `sprite.json`+`sprite.png` fetch, a pixel-ratio-driven `@Nx` suffix mismatch, or a native-side sprite-cache validator that rejects `file://`), by capturing native logs (`adb logcat` filtered for `mbgl`/sprite/resource-load errors) during a repeat of this spike with verbose logging enabled — or (b) if unresolvable in-phase, land OFFL-02/OFFL-03/OFFL-04 (basemap + labels offline) without offline sprite icons, and explicitly scope out offline `arrow`/route-shield icon rendering as a known, documented gap (icons still render fine online, this only affects downloaded/airplane-mode trails) — bring that scope call back for confirmation rather than silently descoping, per the same D-03 spirit (no silently-assumed fallback on a failed sub-gate).
- Delete the three throwaway spike artifacts (`spike_glyph_file_screen.dart`, `spike_glyph_seed.dart`, the `main.dart` FAB) as originally planned once 15-06 lands the real offline cache — this is unaffected by the A2 finding.

## Self-Check: PASSED

- FOUND: app/lib/routes/spike_glyph_file_screen.dart
- FOUND: app/lib/util/spike_glyph_seed.dart
- FOUND: .planning/phases/15-maplibre-core-trail-rendering-offline-parity/15-01-SUMMARY.md
- FOUND: commit d713456b (Task 1)
- FOUND: commit d0d3920f (tooltip/Overlay crash fix, found during Task 2 verification)

---
*Phase: 15-maplibre-core-trail-rendering-offline-parity*
*Status: COMPLETE — A1 PASS, A2 FAIL (documented follow-up for 15-06)*
