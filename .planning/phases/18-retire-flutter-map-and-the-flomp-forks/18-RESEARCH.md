# Phase 18: Retire flutter_map and the flomp Forks - Research

**Researched:** 2026-07-10
**Domain:** Flutter/Dart dependency cleanup (pubspec.yaml package removal, dependency_overrides removal, version pinning) + dead-code excavation + manual on-device regression
**Confidence:** HIGH

## Summary

Phase 18 is pure subtraction, not new construction: no CONTEXT.md exists for this phase (discuss-phase has not run), there is no new stack to select, and the Package Legitimacy Gate does not apply because **zero new packages are introduced**. The work is (1) delete four `flutter_map` family packages, (2) delete two `vector_map_tiles`/`vector_tile_renderer` packages and their `flomp/*` git overrides, (3) pin `maplibre` to an exact version, and (4) manually walk every map surface on a physical device.

The investigation found the codebase is **almost** ready for a clean removal, but not quite — three genuine blockers exist that a naive "just delete the pubspec lines" plan would miss entirely:

1. **`app/lib/provider/map_style_provider.dart` is dead code that still compiles** — its `mapStyleProvider` Riverpod provider (built on `vector_map_tiles`'s `Style` type and the flomp `vector_tile_renderer` fork's theme functions) has zero real consumers left (confirmed: the only remaining reference to the string `mapStyleProvider` anywhere in `lib/` is inside a code comment in `map_screen.dart` describing what it used to be replaced by). However, the file also exports `effectiveBrightness()`, a small pure function that three live files (`map_style_json_provider.dart`, `wanderer_map.dart`, `navigation_screen.dart`) still import from it via `show effectiveBrightness`. Deleting `map_style_provider.dart` outright breaks three importers unless `effectiveBrightness` is relocated first.
2. **`app/tool/extract_map_styles.dart` directly imports the flomp `vector_tile_renderer` fork's internal (non-barrel) source files** (`package:vector_tile_renderer/src/themes/wanderer/wanderer_light_theme.dart`). This is a `tool/` CLI, not `lib/` or `test/`, so none of Phase 15-17's `lib/`-scoped grep/analyze gates ever touched it — but `analysis_options.yaml` has no path exclusions, so a repo-wide `flutter analyze` (no path argument) **will** fail on this file's now-unresolvable import the moment `vector_tile_renderer` leaves `pubspec.yaml`. Phase 15's own research (`15-RESEARCH.md` line 109) already flagged this file as "deleted in Phase 18 (CLEAN-02)" — this is a confirmed, pre-planned deletion, not a new discovery, but it is easy to miss if a plan only grep-checks `lib/`.
3. **`flutter_map_location_marker` cannot be dropped by simply deleting the pubspec line** — three live files (`foreground_position_stream_provider.dart`, `wanderer_map.dart`, `map_screen.dart`) import it *only* for two of its plain data classes, `LocationMarkerPosition` (a 3-field `{latitude, longitude, accuracy}` value class with no flutter_map rendering dependency) and `ServiceDisabledException`. These are used as a bespoke Stream<T> payload type for `foregroundPositionStreamProvider`, unrelated to the actual `flutter_map` widget the package also ships. Removing the package requires first replacing these two types with tiny local equivalents.

By contrast, `flutter_map`, `flutter_map_animations`, and `flutter_map_marker_cluster` (plus its transitive `flutter_map_marker_popup`) are **fully dead** — a repo-wide grep confirms zero remaining source references to any of `AnimatedMapController`, `CurrentLocationLayer`, `package:flutter_map_marker_cluster`, or `package:flutter_map_animations`, and `app/lib/util/map_coordinate_adapter.dart` (the only file still importing bare `package:flutter_map`) has **zero importers anywhere in the tree** — it is leftover dead code from the Phase 14 boundary-adapter strategy and can be deleted outright.

**Primary recommendation:** Sequence the plan as (a) relocate `effectiveBrightness` out of `map_style_provider.dart` into `map_style_json_provider.dart` and update its two other importers' import lines → (b) delete `map_style_provider.dart` + its generated `.g.dart`, `tool/extract_map_styles.dart`, and `map_coordinate_adapter.dart` → (c) replace `LocationMarkerPosition`/`ServiceDisabledException` usage with local classes in `foreground_position_stream_provider.dart` → (d) remove the six packages and two `flomp/*` overrides from `pubspec.yaml`, pin `maplibre: 0.3.5` (exact) → (e) `flutter pub get` → (f) `flutter analyze` (whole package, not just `lib/`) and `flutter test` → (g) physical-device walk of all six map surfaces online and in airplane mode.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Map rendering (basemap, track, markers, clusters, navigation puck) | Browser/Client (Flutter mobile app) | — | All rendering already migrated to native `maplibre` GL bindings in Phases 15-17; this phase only removes the now-unused rendering libraries, it does not move any capability between tiers |
| Style JSON composition (`mapStyleJsonProvider`) | Client (Flutter) | — | Already the sole live style path; unaffected by this phase except for the `effectiveBrightness` relocation |
| Dependency manifest / version pinning | Build tooling (`pubspec.yaml`, not a runtime tier) | — | Static declaration, resolved at `pub get` time, not a request-time concern |
| On-device regression verification | Client (Flutter, manual QA) | — | No backend or SvelteKit involvement; this is entirely an app-side, manual, physical-device concern per success criterion 4 |

This phase touches exactly one tier (Flutter mobile client) and one non-runtime artifact (the pub dependency manifest). There is no web/SvelteKit, Go/PocketBase, or database surface in scope — `CLAUDE.md`'s constraint that v1.4 is "app-only apart from the new glyph/sprite endpoint" (already delivered in Phase 13) is honored by definition here.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CLEAN-01 | `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, and `flutter_map_marker_cluster` removed from `pubspec.yaml`; `flutter pub deps` shows none in the tree | Confirmed exact current resolved versions via live `flutter pub deps --style=compact` run (see Package Legitimacy Audit / Current State table). Confirmed `flutter_map_marker_cluster`, `flutter_map_animations`, and bare `flutter_map` have zero remaining source references — safe to delete outright. Confirmed `flutter_map_location_marker` requires a small refactor first (`LocationMarkerPosition`/`ServiceDisabledException` extraction) — see Common Pitfalls #3 and Code Examples. |
| CLEAN-02 | `vector_map_tiles`/`vector_tile_renderer` removed from `pubspec.yaml`; `dependency_overrides` no longer names either `flomp/*` fork | Confirmed both packages' only two source-level consumers: `map_style_provider.dart` (dead `mapStyleProvider` provider — safe to delete after relocating `effectiveBrightness`) and `tool/extract_map_styles.dart` (one-off asset-generation CLI, already pre-flagged for Phase-18 deletion in `15-RESEARCH.md:109`). Confirmed the `meta: ^1.18.0` override is unrelated to the flomp forks and must stay. |
| CLEAN-03 | `maplibre` pinned to an exact version, not a caret range | Confirmed via pub.dev API: `0.3.5` is both the currently-resolved version (`pubspec.lock`) and the latest published version (published 2026-04-11). Confirmed via official CHANGELOG that 0.3.4→0.3.5 and 0.3.3+2→0.3.4 changes are bug fixes / additive features only (no breaking API changes affecting this app's usage: `MapCompass`, `enableLocation`/`trackLocation`, `animateCamera`/`fitBounds`, `GeoJsonSource`/`LineStyleLayer`, `pmtiles://`/`file://` source handling). Safe to pin `maplibre: 0.3.5` exactly with no code changes required. |
</phase_requirements>

## Package Legitimacy Audit

**Not applicable in the standard sense — this phase installs zero new packages.** Every action is a *removal* of already-vetted, already-running dependencies (four of the six were vetted and running since before v1.4; `vector_map_tiles`/`vector_tile_renderer` forks were vetted in `15-RESEARCH.md`'s own Package Legitimacy Audit, which explicitly pre-flagged the flomp fork "read only by the one-off extraction script... slated for deletion in Phase 18"). The slopcheck/registry-verification gate exists to catch hallucinated or malicious *new* installs; it has no meaningful signal to add when the action is `flutter pub remove`. No `slopcheck` run was performed for this reason.

**Current state (verified via live `flutter pub deps --style=compact` run against this exact repo, 2026-07-10):**

| Package | Constraint in pubspec.yaml | Resolved version (pubspec.lock) | Latest on pub.dev | Live source imports remaining | Disposition |
|---------|---------------------------|----------------------------------|--------------------|-------------------------------|--------------|
| `flutter_map` | `^8.3.0` | 8.3.0 | 8.3.1 `[VERIFIED: pub.dev API]` | `lib/util/map_coordinate_adapter.dart` only — **zero importers of that file anywhere** | REMOVE (delete package + delete the now-orphaned adapter file) |
| `flutter_map_animations` | `^0.10.0` | 0.10.0 | 0.10.0 `[VERIFIED: pub.dev API]` | none | REMOVE (already fully dead) |
| `flutter_map_location_marker` | `^10.0.2` | 10.2.0 | 10.3.0 `[VERIFIED: pub.dev API]` | `foreground_position_stream_provider.dart`, `wanderer_map.dart` (`show LocationMarkerPosition`), `map_screen.dart` (`show LocationMarkerPosition`) — for two plain data classes only, not the map widget | REMOVE **after** relocating `LocationMarkerPosition`/`ServiceDisabledException` to local classes |
| `flutter_map_marker_cluster` | `^8.2.2` | 8.2.2 | 8.2.2 `[VERIFIED: pub.dev API]` | none (repo-wide grep for `flutter_map_marker_cluster`/`AnimatedMapController`/`CurrentLocationLayer` returns zero hits) | REMOVE (already fully dead) |
| `flutter_map_marker_popup` (transitive, via `flutter_map_marker_cluster`) | not direct | 8.1.1 | — | none | Falls away automatically once `flutter_map_marker_cluster` is removed |
| `vector_map_tiles` | `^10.0.0-beta.2` (git override → `flomp/flutter-vector-map-tiles`) | 10.0.0-beta.2 @ `2ad23ae2` | n/a (fork, not published independently) | `map_style_provider.dart` only (dead `mapStyleProvider` provider) | REMOVE **after** relocating `effectiveBrightness` and deleting `map_style_provider.dart` |
| `vector_tile_renderer` | `^7.0.0-beta.1` (git override → `flomp/dart-vector-tile-renderer`) | 7.0.0-beta.2 @ `d52dd7da` | n/a (fork) | `map_style_provider.dart` AND `tool/extract_map_styles.dart` (imports internal theme source files directly, bypassing the public barrel) | REMOVE **after** deleting both files |
| `maplibre` | `^0.3.3+2` | 0.3.5 | 0.3.5, published 2026-04-11 `[VERIFIED: pub.dev API]` | All six map surfaces (native GL, current stack) | PIN exact `0.3.5` (no caret) |
| `flutter_rotation_sensor` (transitive, via `flutter_map_location_marker`) | not direct | 0.2.0 | — | none direct | Falls away once `flutter_map_location_marker` is removed |

**Packages removed:** `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, `flutter_map_marker_cluster`, `vector_map_tiles`, `vector_tile_renderer` (six total, plus their two-entry `flomp/*` `dependency_overrides` block and the transitive `flutter_map_marker_popup`/`flutter_rotation_sensor`).
**Packages pinned (not removed):** `maplibre` — caret range `^0.3.3+2` becomes exact `0.3.5`.
**`dependency_overrides` entries that STAY:** `meta: ^1.18.0` — unrelated to the flomp forks (resolves an unrelated transitive `meta` version conflict); do not remove this entry.

## Architecture Patterns

### Deletion Order (avoids a broken mid-cleanup build)

The safe order is **usages before manifest, manifest before verification, verification before device**:

```
1. Grep-audit (read-only, no edits): confirm every source-level importer of
   the six packages, cross-check against this RESEARCH.md's table.
   grep -rn "package:flutter_map\|package:vector_map_tiles\|package:vector_tile_renderer" \
     app/lib app/test app/tool

2. Code changes FIRST (while packages are still in pubspec.yaml, so
   `flutter analyze` can validate each step incrementally):
   a. Relocate `effectiveBrightness()` from map_style_provider.dart into
      map_style_json_provider.dart. Update the two other importers
      (wanderer_map.dart, navigation_screen.dart) to drop their
      `import '.../map_style_provider.dart' show effectiveBrightness;`
      line entirely (both already import map_style_json_provider.dart).
   b. Delete app/lib/provider/map_style_provider.dart and
      app/lib/provider/map_style_provider.g.dart.
   c. Delete app/tool/extract_map_styles.dart (pre-flagged in
      15-RESEARCH.md:109 — its only purpose was to seed the now-committed
      assets/map/wanderer_{light,dark}.json files, which stay).
   d. Delete app/lib/util/map_coordinate_adapter.dart (zero importers).
   e. In foreground_position_stream_provider.dart: replace the
      `LocationMarkerPosition`/`ServiceDisabledException` import with two
      small local classes (see Code Examples). Update wanderer_map.dart's
      and map_screen.dart's `show LocationMarkerPosition` imports to point
      at the new local type's location instead.
   f. `flutter analyze` (whole package — catches any stray reference
      the grep missed) + `flutter test` — MUST be clean before touching
      pubspec.yaml. The packages are still present at this point, so any
      failure here is a code-logic bug, not a missing-dependency error.

3. pubspec.yaml edits (only after step 2 passes clean):
   a. Delete the four flutter_map family lines + vector_map_tiles +
      vector_tile_renderer from `dependencies:`.
   b. Delete the vector_tile_renderer and vector_map_tiles entries from
      `dependency_overrides:` (keep the `meta:` entry).
   c. Change `maplibre: ^0.3.3+2` to `maplibre: 0.3.5` (no caret).

4. `flutter pub get` — resolves the new, smaller dependency graph.
   If this fails, a leftover reference from step 2 exists; re-audit
   before proceeding (do not retry pub get blindly).

5. Verification (whole-package, not lib/-scoped):
   flutter analyze                      # no path arg — catches tool/, test/
   flutter pub deps --style=compact | grep -iE \
     "flutter_map|vector_map_tiles|vector_tile_renderer"   # expect empty
   flutter test

6. Physical-device walk (success criterion 4) — see the six-surface
   checklist below, online AND in airplane mode for each.
```

**Why code-before-manifest, not manifest-before-code:** Deleting the pubspec lines first and running `flutter pub get` immediately would make `map_style_provider.dart`, `tool/extract_map_styles.dart`, and the location-marker imports fail to *resolve* rather than fail a lint — `flutter analyze` reports these as import-resolution errors indistinguishable from "did I break something new," making it harder to isolate the refactor from the removal. Doing the code changes first, with the packages still installed, means every `flutter analyze` run in step 2 is diagnosing actual code correctness, and the only thing that can go wrong in step 3-4 is "did I delete the right manifest lines."

### The Six Map Surfaces (success criterion 4), mapped to files

| Surface (as named in ROADMAP) | Screen/file | Map host widget |
|---|---|---|
| Trail detail | `app/lib/routes/trail_detail_screen.dart` | No map of its own — hosts a tab/route to "Trail map" below; verify navigating into it still works |
| Trail map | `app/lib/routes/trail_detail_map_screen.dart` | `WandererMap` (`app/lib/components/base/wanderer_map.dart`) |
| List | `app/lib/routes/list_detail_screen.dart` | `SearchMap` (`app/lib/components/base/search_map.dart`) |
| List map | `app/lib/routes/list_detail_map_screen.dart` | `SearchMap` |
| Map screen | `app/lib/routes/map_screen.dart` | `SearchMap` |
| Navigation | `app/lib/routes/navigation_screen.dart` | `ml.MapLibreMap` directly (native, no wrapper widget) |

`trail_detail_screen.dart` itself contains zero map-related code (confirmed via grep) — it is a route-shell around `TrailDetailMapScreen`, wired through `app/lib/provider/router_provider.dart` (`/trail/:id` → `TrailDetailScreen`, nested `/trail/:id/map` → `TrailDetailMapScreen`). The on-device checklist should verify the tab/navigation transition itself still works, not expect an inline map on that screen.

### Anti-Patterns to Avoid

- **Deleting pubspec lines before auditing source usages:** produces a `flutter pub get` or `flutter analyze` failure with no clear signal of *which* file needs fixing, especially for `tool/` files that a `lib/`-scoped mental model overlooks.
- **Trusting a `lib/`-only grep as proof of zero remaining references:** `analysis_options.yaml` has no `analyzer: exclude:` block, so `tool/` and any other top-level Dart directory is analyzed too. Any verification grep/analyze command in the plan should cover `app/lib`, `app/test`, AND `app/tool`.
- **Assuming `dependency_overrides:` removal means deleting the whole block:** the `meta: ^1.18.0` entry is unrelated to the flomp forks and must be preserved.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Verifying zero remaining dependency-tree references | A custom recursive pubspec-lock parser | `flutter pub deps --style=compact \| grep -iE "flutter_map\|vector_map_tiles\|vector_tile_renderer"` | Official, already-verified-safe read-only command (see Environment Availability) — reflects Dart's actual resolved graph, including transitive deps a source grep cannot see |
| Confirming zero source-level imports remain | A hand-rolled Dart AST import scanner | `grep -rn "package:flutter_map\|package:vector_map_tiles\|package:vector_tile_renderer" app/lib app/test app/tool` | Simple, fast, and — critically — must include `app/tool`, which prior phases' `lib/`-scoped grep gates did not cover |
| Replacing `LocationMarkerPosition` | A generic "location" abstraction layer or new package | A private, file-local plain data class with the same 3 fields (`latitude`, `longitude`, `accuracy`) | The type is consumed only as a `Stream<T>` payload inside this app's own provider; no external API contract depends on the exact class identity, just its three numeric fields |

**Key insight:** every "don't hand-roll" temptation here is really "don't build tooling to verify the removal" — the correct verification tools (`flutter pub deps`, `flutter analyze`, `grep`) already exist and are safe to run read-only.

## Common Pitfalls

### Pitfall 1: Treating `map_style_provider.dart` as fully dead and deleting it in one step
**What goes wrong:** `flutter analyze` immediately reports three broken imports in `map_style_json_provider.dart`, `wanderer_map.dart`, and `navigation_screen.dart` (all `show effectiveBrightness`).
**Why it happens:** The file's *primary* export (`mapStyleProvider`, the Riverpod provider) is genuinely dead, which makes it easy to conclude the whole file is dead without checking its secondary export.
**How to avoid:** Relocate `effectiveBrightness()` into `map_style_json_provider.dart` first (both other consumers already import that file), update the two downstream imports to drop the now-redundant `map_style_provider.dart` line, THEN delete `map_style_provider.dart` + `.g.dart`.
**Warning signs:** `flutter analyze` reporting "Target of URI doesn't exist: 'package:wanderer/provider/map_style_provider.dart'" in exactly those three files.

### Pitfall 2: Deleting `vector_tile_renderer`/`vector_map_tiles` from pubspec.yaml without deleting `tool/extract_map_styles.dart` first
**What goes wrong:** A full-repo `flutter analyze` (no path argument) fails on `tool/extract_map_styles.dart`'s two imports of internal flomp-fork theme source files. This is easy to miss because Phase 15-17's own verification gates were explicitly scoped to `lib/` (per their SUMMARY.md files: "flutter analyze over the whole lib/ tree"), and `tool/` was never in scope for any prior phase's grep gate.
**Why it happens:** `analysis_options.yaml` has no exclusion for `tool/`, and the file was never touched by Phases 15-17 (it ran once, at Phase 15 extraction time, to produce the committed `assets/map/wanderer_{light,dark}.json` files).
**How to avoid:** Delete `tool/extract_map_styles.dart` as part of this phase's plan — it has already served its one-off purpose (the generated assets are committed and stay); `15-RESEARCH.md:109` pre-flagged exactly this deletion.
**Warning signs:** `flutter analyze` (whole-package) reporting unresolved `package:vector_tile_renderer/src/themes/...` imports specifically in `tool/`.

### Pitfall 3: Assuming `flutter_map_location_marker` has zero remaining usages because no `flutter_map` widget from it is rendered
**What goes wrong:** Deleting the package breaks three files that import `LocationMarkerPosition`/`ServiceDisabledException` as plain data types, unrelated to any rendering widget.
**Why it happens:** Phase 17's SUMMARY.md explicitly states the *rendering* half of this package (`CurrentLocationLayer`) was fully retired — but the package also ships small data classes that got repurposed as a convenient `Stream<T>` payload shape and were never migrated off, because they never depended on `flutter_map` itself.
**How to avoid:** Grep specifically for `LocationMarkerPosition` and `ServiceDisabledException` (not just `flutter_map_location_marker` import lines) before concluding the package is droppable; replace both with local classes (see Code Examples) before removing the pubspec entry.
**Warning signs:** `flutter analyze` reporting unresolved `LocationMarkerPosition`/`ServiceDisabledException` symbols in `foreground_position_stream_provider.dart`, `wanderer_map.dart`, `map_screen.dart`.

### Pitfall 4: Running `flutter analyze lib/` (or similar path-scoped commands) as the final verification gate
**What goes wrong:** A scoped analyze can pass green while `tool/` (or any other non-`lib/` Dart file) is silently broken, giving false confidence that CLEAN-02 is complete.
**Why it happens:** This exact scoping pattern is precedented in this project — Phase 17-02's SUMMARY.md explicitly describes running "flutter analyze over the whole lib/ tree," which was correct for that phase's scope (it only touched files under `lib/`) but is insufficient once `tool/` also has a package-removal-sensitive file.
**How to avoid:** For this phase specifically, run bare `flutter analyze` (repository default scope, no path argument) as the gate, not a `lib/`-scoped invocation.
**Warning signs:** A previously-scoped analyze command silently omits `tool/extract_map_styles.dart` from its output entirely.

### Pitfall 5: Removing the whole `dependency_overrides:` block instead of the two flomp entries
**What goes wrong:** Deleting `meta: ^1.18.0` alongside the flomp overrides can reintroduce an unrelated transitive version conflict that has nothing to do with this phase's packages.
**Why it happens:** All three overrides live in the same YAML block, making a careless "delete the block" edit look tempting.
**How to avoid:** Edit only the `vector_tile_renderer:` and `vector_map_tiles:` map entries under `dependency_overrides:`; leave `meta: ^1.18.0` untouched.
**Warning signs:** `flutter pub get` reporting a new, unrelated version-solving conflict involving `meta` after the edit.

## Code Examples

### Relocating `effectiveBrightness` (Pitfall 1 fix)

```dart
// app/lib/provider/map_style_json_provider.dart — ADD this function
// (moved from the soon-to-be-deleted map_style_provider.dart; behavior unchanged)

Brightness effectiveBrightness(ThemeMode mode) {
  if (mode == ThemeMode.dark) return Brightness.dark;
  if (mode == ThemeMode.light) return Brightness.light;
  return WidgetsBinding.instance.platformDispatcher.platformBrightness;
}
```

```dart
// app/lib/components/base/wanderer_map.dart — BEFORE
import 'package:wanderer/provider/map_style_provider.dart'
    show effectiveBrightness;
// (map_style_json_provider.dart already imported elsewhere in this file)

// AFTER — delete the map_style_provider.dart import line entirely;
// effectiveBrightness is now available from the already-imported
// map_style_json_provider.dart with no import change needed.
```

Apply the identical AFTER change to `app/lib/routes/navigation_screen.dart` (it has the same before/after shape — both files already import `map_style_json_provider.dart` separately).

### Replacing `LocationMarkerPosition`/`ServiceDisabledException` (Pitfall 3 fix)

```dart
// app/lib/provider/foreground_position_stream_provider.dart — local replacements
// for the two flutter_map_location_marker data types this file actually uses.
// Field shape matches the original exactly (latitude, longitude, accuracy;
// a no-arg marker exception), so no call-site logic changes.

class LocationMarkerPosition {
  final double latitude;
  final double longitude;
  final double accuracy;

  const LocationMarkerPosition({
    required this.latitude,
    required this.longitude,
    required this.accuracy,
  });
}

class ServiceDisabledException implements Exception {
  const ServiceDisabledException();
}
```

Remove `import 'package:flutter_map_location_marker/flutter_map_location_marker.dart';` from this file, and change the two `show LocationMarkerPosition` imports in `wanderer_map.dart`/`map_screen.dart` to import this file's new local class instead (e.g. `import 'package:wanderer/provider/foreground_position_stream_provider.dart' show LocationMarkerPosition;` — that file is already imported by both for `foregroundPositionStreamProvider` itself, so this is a same-import addition, not a new import line).

### Pinning maplibre (CLEAN-03)

```yaml
# app/pubspec.yaml — dependencies: section
# BEFORE
  maplibre: ^0.3.3+2
# AFTER
  maplibre: 0.3.5
```

No source changes required — 0.3.5 is already the resolved version running in production today per `pubspec.lock`; this edit only removes future caret-range upgrade risk.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `trail_detail_screen.dart` has no inline map and success criterion 4's "trail detail" surface refers to navigating into `trail_detail_map_screen.dart`, not a map widget on that screen itself | Architecture Patterns — Six Map Surfaces table | Low — verified directly via grep (zero map-related tokens in that file) and `router_provider.dart`'s route nesting; this is `[VERIFIED: codebase grep]`, not assumed, but flagged here since the on-device checklist should confirm the *navigation flow* rather than expect a rendered map on that specific screen |

**No other assumptions were required** — every claim in this document was verified directly against the live repository (grep, `flutter pub deps`, `pubspec.lock`) or the official pub.dev registry API. This is unusually low-uncertainty research because the phase is subtractive against an already-fully-implemented, already-verified stack (Phases 15-17), not exploratory greenfield work.

## Open Questions

1. **Should `app/lib/vendor/vector_map_tiles/` (the now-empty directory left by Phase 17-02's deletion of `pm_tile_provider.dart`) be removed?**
   - What we know: The directory contains zero files (confirmed via `find`). Git does not track empty directories, so it isn't actually present in a fresh clone — it only exists in this working tree because a file was deleted from it without `rmdir`.
   - What's unclear: Whether removing it locally has any observable effect at all, since git tracking is a non-issue.
   - Recommendation: Optional cleanup (`rmdir app/lib/vendor/vector_map_tiles app/lib/vendor` if `vendor/` is also then empty) — cosmetic only, not required by any success criterion. Low priority; safe to skip.

2. **Does the on-device walk need a fresh install (uninstall + reinstall) to catch a stale native location-permission or GPS-service dialog interaction from the removed `flutter_map_location_marker` plugin?**
   - What we know: The replacement `LocationMarkerPosition`/`ServiceDisabledException` classes are pure Dart with no platform channel of their own — the actual GPS interaction is via `Geolocator`, unaffected by this refactor.
   - What's unclear: Whether `flutter_map_location_marker`'s Android/iOS plugin registration (if any) leaves any native-side artifact after removal that a hot-reload/hot-restart wouldn't clear.
   - Recommendation: A full `flutter clean && flutter pub get` before the on-device build, plus a normal (not hot-reloaded) fresh app launch, is sufficient — this is standard practice whenever a plugin with native platform code (`flutter_map_location_marker` ships Android/iOS/web platform interfaces per its own pubspec `flutter_rotation_sensor` dependency) is removed, and no evidence suggests special uninstall/reinstall handling is needed beyond that.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK | Building/analyzing the app, running `flutter pub deps` | Yes | 3.44.2 (stable channel) | — |
| Dart SDK | `dart analyze`, code generation | Yes | 3.12.2 | — |
| Network access to pub.dev | Verifying `maplibre` latest version, package legitimacy checks | Yes (confirmed via live `curl` to `pub.dev/api/packages/maplibre` during this research session) | — | — |
| Physical Android/iOS device | Success criterion 4's on-device walk (online + airplane mode) | Not verified in this sandbox — same constraint noted in every Phase 15-17 RESEARCH.md; this project's established pattern defers physical-device checks to a dedicated checkpoint plan | — | Follow the same on-device-checkpoint pattern established in 15-06 (three physical-device bugs found) and 17-03 (7/7 checks passed) |

**Missing dependencies with no fallback:** None blocking plan creation — the physical-device requirement is a known, already-established pattern in this project (every prior map phase ended with an on-device checkpoint plan), not a new gap.

**Verification commands confirmed safe to run (read-only, already exercised during this research session):**
```bash
flutter pub deps --style=compact   # confirmed working, ~10s runtime, no side effects
flutter analyze                    # confirmed available; run with no path argument for whole-package scope
grep -rn "package:flutter_map\|package:vector_map_tiles\|package:vector_tile_renderer" app/lib app/test app/tool
```

## Validation Architecture

Skipped — `.planning/config.json` sets `workflow.nyquist_validation: false` explicitly.

## Security Domain

`security_enforcement: true`, `security_asvs_level: 1` per `.planning/config.json` — section included per policy, though this phase has essentially no traditional ASVS surface (no new auth, session, input-validation, or cryptography code; no new endpoints; no user-facing data flows change).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Not touched — no auth code in this phase |
| V3 Session Management | No | Not touched |
| V4 Access Control | No | Not touched |
| V5 Input Validation | No | No new user input surfaces introduced |
| V6 Cryptography | No | Not touched |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Supply-chain risk from unofficial git-hosted fork dependencies (`dependency_overrides` pointing at `flomp/*` repos, not a vetted registry release) | Tampering | **This phase's actual security payoff:** removing the two `flomp/*` git-fork overrides eliminates the only two dependencies in this app that resolve from an arbitrary GitHub `ref: main` (a moving target, not an immutable, checksum-verified registry release) rather than pub.dev's `sha256`-pinned hosted source. After CLEAN-02, every dependency in `pubspec.lock` resolves from `hosted` (pub.dev, sha256-verified) or the Flutter SDK itself — closing the only non-registry trust boundary in the app's dependency graph. |
| Stale/abandoned dependency accumulation (unmaintained `flutter_map` plugin family left installed after functional migration) | Tampering / Elevation of Privilege (transitively, via unpatched native code) | Deleting unused-but-installed packages reduces the app's total native-code attack surface (both `flutter_map_location_marker` and `flutter_map_marker_cluster` ship platform-channel code) even though no code path currently invokes them — an installed-but-unused plugin is still linked into the compiled binary and still receives whatever runtime permissions its manifest declares. |

## Sources

### Primary (HIGH confidence)
- Live repository state: `app/pubspec.yaml`, `app/pubspec.lock` (read directly, 2026-07-10)
- Live `flutter pub deps --style=compact` execution against this exact repository (2026-07-10) — full resolved dependency graph including transitive packages
- Live `curl https://pub.dev/api/packages/maplibre` — confirms 0.3.5 is both currently-resolved and latest-published (2026-04-11)
- Live `curl` fetch of `maplibre`'s official CHANGELOG.md from `github.com/josxha/flutter-maplibre` — confirms no breaking changes between 0.3.3+2 and 0.3.5
- Live `curl https://pub.dev/api/packages/{flutter_map,flutter_map_animations,flutter_map_location_marker,flutter_map_marker_cluster}` — confirms latest published versions for the audit table
- `.planning/phases/15-maplibre-core-trail-rendering-offline-parity/15-RESEARCH.md` (this project's own prior research) — pre-flagged `tool/extract_map_styles.dart` and the `vector_tile_renderer` flomp fork for Phase-18 deletion, corroborating this session's independent grep-based discovery
- `.planning/phases/17-navigation-on-maplibre/17-01-SUMMARY.md`, `17-02-SUMMARY.md`, `17-03-SUMMARY.md` — confirm CORE-05/06/07/OFFL-06 completion state and the exact files touched/deleted
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md` — phase scope, requirement text, and decision history

### Secondary (MEDIUM confidence)
- None — every claim in this document traces to a primary source above (live command execution, official pub.dev API, or this project's own prior-phase artifacts).

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: N/A (removal-only phase, no new stack) — all package facts HIGH confidence (verified via live pub.dev API + live pubspec.lock)
- Architecture: HIGH — deletion ordering and dead-code findings verified via direct grep/read of this exact repository, not general Flutter knowledge
- Pitfalls: HIGH — all five pitfalls are concrete, file-and-line-verified findings in this repository, not generic "common mistakes" from training data

**Research date:** 2026-07-10
**Valid until:** Effectively indefinite for the removal mechanics (this is a static snapshot of an already-completed codebase state); 30 days for the `maplibre` "latest version" claim specifically, since upstream could publish a new release before planning executes.
