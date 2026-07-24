# Phase 26: Trail Download Guard - Research

**Researched:** 2026-07-24
**Domain:** Flutter/Dart mobile app — client-side geometric coverage check + Riverpod state + bottom-sheet UI, no new packages, no backend changes
**Confidence:** HIGH

## Summary

This phase inserts a pure, local, synchronous coverage check in front of the existing single shared trail-download entry point (`DownloadingTrailIds.download(trail)`), and — only when coverage is missing — shows a bottom modal sheet that lets the user opt into downloading missing region packages alongside the trail. Every data shape and downstream API needed already exists in the codebase exactly as `26-CONTEXT.md` and `26-UI-SPEC.md` describe (verified by direct file reads, not assumed): `Trail`/`TrailEntity` bbox fields (`minLat`/`minLon`/`maxLat`/`maxLon`), `RegionEntity`'s four discrete bbox fields plus its computed `status` getter (`RegionStatus.notDownloaded/downloading/downloaded/updateAvailable/error`), `TileRepositoryStatus.downloadVector`/`downloadDem`, `RegionListNotifier` (`regionListNotifierProvider`) as the synchronous local-catalog read, and `formatBytes()` for size display. No new pubspec dependency is needed — the "geo library" question in the task brief is moot: coverage is plain float rectangle-overlap arithmetic (D-03: raw bbox-vs-bbox intersection, no polyline/turf-style geometry).

The single architecturally interesting question — where the guard check + `BuildContext`-requiring modal sheet lives, given `DownloadingTrailIds.download(trail)` is a Riverpod `Notifier` method with no widget context of its own — has a **verified, existing precedent** in this codebase: `app/lib/main.dart` already uses a global `navigatorKey.currentContext` (declared in `router_provider.dart`) to show dialogs from non-widget code (the resume-navigation prompt). The guard can and should reuse this exact pattern, letting the coverage check + sheet live inside `DownloadingTrailIds.download()` itself, so the "single shared entry point, no duplication across the two call sites" goal from `26-CONTEXT.md`'s discretion note is met literally, not just structurally.

One real implementation gap was found during research (not present in CONTEXT.md/UI-SPEC.md): `DownloadNotificationService.showProgress(trailName, done, total)` hardcodes trail-specific title/body copy (`'Downloading trail... {pct}%'`) with no parameter for the aggregated "Downloading offline content" / "{done}/{total} · {N} item(s)" copy the UI-SPEC's D-10 unified notification requires. This service needs a small signature extension (or a new method) — see Pitfall 1 and Code Examples below.

**Primary recommendation:** Add a single pure top-level function `List<RegionEntity> missingCoverageRegions(Trail trail, List<RegionEntity> catalog)` (or equivalent), call it synchronously inside `DownloadingTrailIds.download()` before any download starts, and branch: fully covered → existing behavior untouched (GUARD-01); regions missing → show the bottom sheet via `navigatorKey.currentContext`, dismiss on Download, then fire `download(trail)`'s existing body plus `downloadVector`/`downloadDem` for each checked region — all backgrounded, none awaited by the sheet.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Trail/region bbox overlap computation | Frontend Server / Client (Flutter app logic) | — | Pure in-memory arithmetic over already-fetched local data; no network, no backend involvement (D-11 local-only) |
| Missing-region bottom sheet UI | Browser / Client (Flutter widget tree) | — | Presentational; reads Riverpod state, no business logic beyond selection bookkeeping |
| Region catalog storage/freshness | Client (ObjectBox local store) | API/Backend (catalog origin, Phase 21.5/22) | Guard reads the already-synced local catalog only; catalog refresh itself is Phase 24's concern, untouched here |
| Trail + region parallel downloads | Client (Dio-based download engines) | — | `trailDownloadService`/`TileRepositoryManager` already own this; guard is purely a trigger, not a new engine |
| Unified download notification | Client (`flutter_local_notifications` via `DownloadNotificationService`) | — | Single fixed-id (42) notification extended with new aggregate copy; no new notification channel needed |

## Standard Stack

### Core
No new libraries. This phase is 100% composition of existing, already-verified dependencies.

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `flutter_riverpod` / `riverpod_annotation` | pinned (existing) | State (coverage result caching not needed — computed inline; sheet's own checkbox `setState`) | Already the app's exclusive state-management approach |
| `objectbox` | ^5.3.1 (existing) | `RegionEntity` read via `regionListNotifierProvider` | Already the local persistence layer for regions |
| `font_awesome_flutter` | existing | Sheet/row icons | House convention (UI-SPEC, Design System table) |

### Supporting
Not applicable — no supporting libraries beyond what Phase 22–24 already installed.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Raw bbox-vs-bbox intersection (plain `if` arithmetic on 4 doubles) | `turf`-equivalent Dart geo package (e.g. `dart_jts`, `geobase`) | Rejected by CONTEXT.md D-03 explicitly — polyline-aware/threshold overlap adds geometry-processing dependency weight and complexity with no functional benefit at bbox granularity; this project's web side uses `@turf/*` for exactly the kind of polygon math this phase deliberately avoids |
| `navigatorKey.currentContext` for the sheet | Threading `BuildContext` as a parameter through `download(trail, context)` | Both are viable; `navigatorKey` is **already established in this codebase** (`main.dart`) for showing UI from non-widget/notifier code, so it is lower-risk and requires no signature change to the two existing call sites |

**Installation:** None required — zero new packages.

**Version verification:** Not applicable (no new packages). Existing pins verified by direct read of `app/pubspec.yaml`: `objectbox: ^5.3.1`, `objectbox_flutter_libs: ^5.3.1`, `font_awesome_flutter` (already imported throughout), `flutter_local_notifications` (imported in `download_notification_service.dart`).

## Package Legitimacy Audit

Not applicable — this phase introduces zero new external packages. `slopcheck`/registry verification steps were skipped because there is nothing to check; UI-SPEC.md's own Registry Safety section reaches the identical conclusion ("This phase introduces zero new packages").

**Packages removed due to slopcheck [SLOP] verdict:** none (N/A — no packages evaluated)
**Packages flagged as suspicious [SUS]:** none (N/A — no packages evaluated)

## Architecture Patterns

### System Architecture Diagram

```text
User taps "Download" (trail_detail_screen.dart / trail_dropdown.dart)
        │
        ▼
ref.read(downloadingTrailIdsProvider.notifier).download(trail)   ← unchanged call site, both places
        │
        ▼
┌────────────────────────────────────────────────────────────────┐
│ DownloadingTrailIds.download(trail)  [Notifier method]          │
│                                                                  │
│  1. Guard entry (NEW):                                          │
│     regions = ref.read(regionListNotifierProvider)  (local-only,│
│                synchronous, D-11 — no network fetch)             │
│     missing = missingCoverageRegions(trail, regions)  (pure fn) │
│                                                                  │
│     ┌─ missing.isEmpty? ──────────────┐                         │
│     │ YES → skip sheet entirely       │  NO → show bottom sheet  │
│     │  (GUARD-01: unchanged behavior) │   via navigatorKey       │
│     └─────────────────────────────────┘   .currentContext       │
│                                              │                   │
│                                              ▼                   │
│                              MissingCoverageSheet (NEW widget)  │
│                              - per-region Vector/DEM checkboxes │
│                              - combined-size summary (setState) │
│                              - Download button (always enabled) │
│                                              │                   │
│                              dismiss (swipe/tap-outside)         │
│                              → returns null → download() ABORTS │
│                              (no trail download starts at all)  │
│                                              │                   │
│                              tap Download → returns selection  │
│                              → sheet pops immediately (D-09)     │
│                                              │                   │
│  2. Existing download body (UNCHANGED):     │                   │
│     toast → showProgress → trailDownloadService.downloadTrail   │
│     (+ NEW: fire-and-forget downloadVector/downloadDem per      │
│      selected region, via tileRepositoryStatusProvider.notifier)│
│                                              │                   │
│  3. Unified notification (EXTENDED):        │                   │
│     downloadNotificationServiceProvider aggregates trail's      │
│     onProgress(done,total) + each selected package's             │
│     RegionDownloadState progress into one combined bar          │
└────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
app/lib/
├── util/
│   └── trail_coverage_util.dart        # NEW — pure missingCoverageRegions() + bbox overlap helper
├── components/
│   └── trail/
│       └── missing_coverage_sheet.dart # NEW — the bottom modal sheet widget
├── provider/
│   └── trail/
│       └── trail_download_state_provider.dart  # MODIFIED — guard entry point inside download()
├── provider/
│   └── router_provider.dart            # UNCHANGED — reuse existing `navigatorKey`
├── services/
│   └── download_notification_service.dart      # MODIFIED — parameterize title/body for D-10 aggregation
```

### Pattern 1: Pure bbox-overlap coverage check
**What:** A pure, synchronous, unit-testable function that takes the trail's bbox and the local region catalog snapshot and returns exactly the set of regions overlapping the trail that are NOT `downloaded`/`updateAvailable`.
**When to use:** Called once, synchronously, at the top of `DownloadingTrailIds.download()` — no async gap before this check, so no stale-widget risk.
**Example:**
```dart
// Source: derived from RegionEntity (app/lib/entities/region_entity.dart) and
// Trail (app/lib/models/trail.dart) bbox field shapes verified in this session.

/// True iff [a] and [b] (each [minLon, minLat, maxLon, maxLat]-shaped)
/// overlap at all — a plain axis-aligned rectangle intersection test.
/// D-03: no polyline awareness, no minimum-overlap-area threshold.
bool bboxesOverlap({
  required double aMinLon, required double aMinLat,
  required double aMaxLon, required double aMaxLat,
  required double bMinLon, required double bMinLat,
  required double bMaxLon, required double bMaxLat,
}) {
  return aMinLon <= bMaxLon &&
      aMaxLon >= bMinLon &&
      aMinLat <= bMaxLat &&
      aMaxLat >= bMinLat;
}

/// D-01/D-02: every catalog region whose bbox overlaps [trail]'s bbox AND
/// whose RegionEntity.status is neither downloaded nor updateAvailable.
/// D-11: [catalog] must be the already-loaded local snapshot
/// (`ref.read(regionListNotifierProvider)`), never a fresh network fetch.
List<RegionEntity> missingCoverageRegions(Trail trail, List<RegionEntity> catalog) {
  return catalog.where((region) {
    final overlaps = bboxesOverlap(
      aMinLon: trail.minLon, aMinLat: trail.minLat,
      aMaxLon: trail.maxLon, aMaxLat: trail.maxLat,
      bMinLon: region.minLon, bMinLat: region.minLat,
      bMaxLon: region.maxLon, bMaxLat: region.maxLat,
    );
    if (!overlaps) return false;
    return region.status != RegionStatus.downloaded &&
        region.status != RegionStatus.updateAvailable;
  }).toList();
}
```

### Pattern 2: Non-widget UI trigger via the app's existing `navigatorKey`
**What:** `DownloadingTrailIds.download()` is a `Notifier` method, not a widget — it has no `BuildContext`. This codebase already solves the identical problem for the resume-navigation dialog.
**When to use:** Any time a Riverpod notifier/service (not a widget) needs to show a dialog/sheet.
**Example:**
```dart
// Source: app/lib/main.dart lines 202-214 (verified in this session) — the
// EXISTING resume-navigation-dialog pattern this phase should reuse verbatim.
import 'package:wanderer/provider/router_provider.dart' show navigatorKey;

// Inside DownloadingTrailIds.download(trail), before starting any download:
final ctx = navigatorKey.currentContext;
if (ctx != null && missing.isNotEmpty) {
  final selection = await showMissingCoverageSheet(ctx, trail, missing);
  if (selection == null) return; // sheet dismissed → abort, no download at all
  // selection.checkedRegionIds -> fire downloadVector/downloadDem per D-08/D-09
}
```
`navigatorKey` is declared in `app/lib/provider/router_provider.dart:56` (`final navigatorKey = GlobalKey<NavigatorState>();`) and wired into the app's `GoRouter` — it is already app-lifetime-scoped and safe to read from anywhere, matching the existing `main.dart` precedent exactly (no new global state introduced).

### Pattern 3: Sheet shape — mirror `track_save_options_sheet.dart` verbatim
**What:** `showModalBottomSheet` with a drag handle, bold title, scrollable body, bottom-padded full-width `FilledButton`, exact shape (`BorderRadius.vertical(top: Radius.circular(20))`) already used twice in this codebase.
**When to use:** This is the UI-SPEC's own mandated precedent (Component & Pattern Inventory section) — do not design a new sheet shape.
**Example:**
```dart
// Source: app/lib/components/navigation/track_save_options_sheet.dart
// (verified in this session, shape reused near-verbatim)
Future<MissingCoverageSelection?> showMissingCoverageSheet(
  BuildContext context,
  Trail trail,
  List<RegionEntity> missingRegions,
) {
  return showModalBottomSheet<MissingCoverageSelection>(
    context: context,
    isDismissible: true,
    enableDrag: true,
    isScrollControlled: true, // NEW vs. the precedent — this sheet's content
                              // list can be longer (N regions) than a fixed
                              // 2-toggle sheet; needed so it can grow past
                              // the default 50% max height on a long list.
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (ctx) => _MissingCoverageSheetContent(
      trail: trail,
      missingRegions: missingRegions,
    ),
  );
}
```

### Anti-Patterns to Avoid
- **Storing `BuildContext` on the Notifier as a field:** Only read `navigatorKey.currentContext` transiently, at the point of use, inside `download()` — never assign it to a notifier instance field (classic dangling-context bug class).
- **Calling `RegionRepository.refreshCatalog()` from the guard:** D-11 is explicit — the guard must read the already-synced local snapshot only (`ref.read(regionListNotifierProvider)`), never trigger a network fetch on the download tap.
- **Awaiting the region downloads before starting the trail download:** D-08/D-09 require all selected downloads to start in parallel and the sheet to dismiss immediately; do not `await` `downloadVector`/`downloadDem` before calling the trail download's existing body.
- **Reusing `RegionRepository` for the local read:** `RegionRepository` (in `region_provider.dart`) only exposes `fetchCatalog()`/`upsertCatalog()`/`refreshCatalog()` — it has **no** synchronous "get all local regions" method. The correct local-read provider is `regionListNotifierProvider` (`RegionListNotifier`), which already does exactly this (`store.box<RegionEntity>().getAll()`, sorted). `26-CONTEXT.md`'s canonical-refs section names `RegionRepository` as "the locally-stored catalog the guard reads for D-11" — this is imprecise; the actual correct read path is `regionListNotifierProvider`, not `RegionRepository`. (Flagged for the planner — see Assumptions Log below is not needed since this is a verified correction, not an assumption.)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bbox overlap math | A geo/turf-style Dart package | Four-comparison inline arithmetic (Pattern 1 above) | D-03 explicitly rejects polyline-aware/threshold geometry; a rectangle intersection test is 4 comparisons, not a library problem |
| Byte-size formatting | A new formatter for the sheet's per-package/combined-size text | `formatBytes()` from `app/lib/util/byte_format_util.dart` (verified, already used by Phase 24's Settings screen) | Exact convention match (1 decimal, KB/MB/GB steps) the UI-SPEC's Copywriting Contract requires verbatim |
| Showing UI from a Notifier | A new "dialog service" abstraction | The existing `navigatorKey` (`router_provider.dart`) + `showModalBottomSheet`, exactly as `main.dart`'s resume-navigation dialog already does | Codebase already has one sanctioned mechanism for this; a second one would be an unjustified parallel pattern |
| Progress aggregation | A new generic multi-source progress aggregator class | Sum `done`/`total` byte-equivalents across the trail's `onProgress(done,total)` callback and each selected `RegionDownloadState.vectorProgress`/`demProgress` fraction × that package's known catalog `vectorSize`/`demSize`, fed into the existing `downloadNotificationServiceProvider.showProgress` | D-10's own math is simple enough (weighted sum of fractions) that a dedicated aggregator class would be over-engineering for a single notification bar |

**Key insight:** Every "hard part" of this phase (bbox math, size formatting, sheet shape, download triggering, progress) already has an exact, verified precedent somewhere in this codebase from Phases 22–25. The actual net-new work is small: one pure function, one new widget, and a signature extension on one existing service method.

## Common Pitfalls

### Pitfall 1: `DownloadNotificationService.showProgress` cannot express the D-10 aggregate copy today
**What goes wrong:** `showProgress(trailName, done, total)` (verified in `app/lib/services/download_notification_service.dart`) hardcodes the title to `trailName` and the body to `'Downloading trail... {pct}%'` / `'Preparing download...'`. The UI-SPEC's D-10 copy ("Downloading offline content" title, "{done}/{total} · {N} item(s)" body) has no way to reach the notification through this signature.
**Why it happens:** The service was built in Phase 23/trail-download-era for a single-download-source case; it was never designed to aggregate multiple sources.
**How to avoid:** Extend `showProgress` to accept optional `String? title` / `String? body` overrides (defaulting to today's exact strings when null, so the zero-region "GUARD-01 unchanged" path is byte-for-byte identical to current behavior), or add a small new method (e.g. `showAggregateProgress(title, body, done, total)`) that the guard calls only when `checkedRegions.isNotEmpty`.
**Warning signs:** If a plan calls `showProgress(trail.name, ...)` unmodified expecting different copy to appear, it will silently keep showing the old trail-only text.

### Pitfall 2: `RegionEntity.status` computed getter reads a `ToOne.target` — cache staleness after a background download completes
**What goes wrong:** `RegionEntity.status` derives from `vectorPackage.target?.status`. ObjectBox `ToOne.target` caches per-entity-instance after first read (documented in this codebase's own `region_provider.dart` comment: "RESEARCH.md Pitfall 2 — ObjectBox ToOne.target caches per-instance after first read"). If the guard computed `missing` once and held onto stale `RegionEntity` instances across a long-running trail download, a region that finishes downloading mid-flight (via the sheet's own parallel `downloadVector` call) would not be reflected if re-checked against the same stale instances.
**Why it happens:** This is a pre-existing, already-documented ObjectBox gotcha in this exact codebase (Phase 24 hit it too).
**How to avoid:** The guard's coverage check only needs to run ONCE, synchronously, at tap-time — it does not re-check mid-download (GUARD-03/D-09 explicitly say the guard doesn't own progress or re-verification). As long as the check itself always reads a **fresh** `ref.read(regionListNotifierProvider)` snapshot (not a cached field), this pitfall does not manifest for this phase's actual scope. Flag it only so a future phase extending the guard to live-update the sheet mid-session knows to re-invalidate.
**Warning signs:** A region shows `notDownloaded` in the guard sheet even though Settings shows it `downloaded` — would indicate a stale snapshot was read instead of a fresh `ref.read`.

### Pitfall 3: `RegionStatus.error` code value differs from what `26-CONTEXT.md` states
**What goes wrong:** `26-CONTEXT.md`'s canonical-refs table states `RegionStatus` values as `notDownloaded(0)/downloading(1)/downloaded(2)/updateAvailable(3)/error(2 code)` — the "error(2 code)" fragment is unclear/likely a documentation slip. The verified source (`app/lib/models/region_status.dart`) shows `error(5)`, not `2` (which is `downloaded`'s code) — with an explicit doc comment explaining `error` was appended AFTER `updateAvailable` with a new code, and `paused(4)` was removed and deliberately left unused/unreassigned.
**Why it happens:** Likely a transcription slip in CONTEXT.md's own summary of the enum.
**How to avoid:** Never hand-transcribe the enum's codes into a plan or task description — reference `RegionStatus.error`/`RegionStatus.downloaded` by name, or re-read `region_status.dart` directly. This phase's own coverage logic (Pattern 1 above) compares by enum value (`region.status != RegionStatus.downloaded`), never by raw `.code` int, so this discrepancy has zero functional impact on the guard itself — it only matters if a plan or task text quotes the numeric codes verbatim.
**Warning signs:** A task description or code comment citing `RegionStatus.error`'s code as `2` — that value collides with `downloaded`.

### Pitfall 4: Zero-region catalog / all-regions-not-in-catalog trail (D-04's "no region at all" case)
**What goes wrong:** If `missingCoverageRegions` returns empty NOT because everything is covered but because **no catalog region overlaps the trail at all** (a genuine no-region gap, D-04), the guard must distinguish "fully covered" from "nothing to check against" — both produce an empty `missing` list, but only the first should proceed silently; the second must additionally surface the non-blocking orange toast ("Part of this trail isn't covered by any offered region").
**Why it happens:** A naive `missing.isEmpty` check alone cannot tell these two cases apart — you also need to know whether ANY catalog region overlapped the trail's bbox at all (regardless of status).
**How to avoid:** Compute two things, not one: (a) `overlappingRegions` = all catalog regions whose bbox overlaps the trail (any status), and (b) `missing` = the subset of (a) not downloaded/updateAvailable. If `overlappingRegions.isEmpty`, show the D-04 warning toast (regardless of `missing` being trivially empty too) and proceed with the download exactly as GUARD-01 describes. If `overlappingRegions.isNotEmpty` but `missing.isEmpty`, proceed silently (fully covered). Only show the bottom sheet when `missing.isNotEmpty`.
**Warning signs:** A trail entirely outside every configured region silently downloads with zero user feedback — violates D-04's "never silent, never generic dead-end" requirement.

### Pitfall 5: Sheet's "Download" button must remain always-enabled, including when zero checkboxes are checked
**What goes wrong:** A natural Flutter instinct is to disable a primary CTA when "nothing is selected." Here that's explicitly wrong (D-08): 0 selected + Download = valid trail-only path (the GUARD-03 escape hatch), not an invalid state.
**Why it happens:** Standard form-validation muscle memory doesn't apply to this specific button.
**How to avoid:** Never gate the `FilledButton`'s `onPressed` on `selectedRegionIds.isNotEmpty` — only gate it on the sheet having finished building (i.e., always enabled once rendered), per UI-SPEC's Copywriting Contract row for the Primary CTA.
**Warning signs:** A plan/task acceptance criterion phrased as "Download button disabled until a region is selected" — this directly contradicts D-08 and must be rejected during plan review.

## Code Examples

### Reading the local region catalog synchronously (D-11)
```dart
// Source: app/lib/provider/region/region_provider.dart (verified this session)
final regions = ref.read(regionListNotifierProvider); // List<RegionEntity>, sorted by name, no I/O
```

### Existing region download trigger (reused as-is by the sheet's Download button)
```dart
// Source: app/lib/provider/region/tile_repository_provider.dart (verified this session)
// Fire-and-forget per D-09 — do not await inside the sheet's onPressed.
ref.read(tileRepositoryStatusProvider.notifier).downloadVector(region.id);
if (demChecked) {
  ref.read(tileRepositoryStatusProvider.notifier).downloadDem(region.id);
}
```

### Existing trail bbox shape (both `Trail` and `TrailEntity`)
```dart
// Source: app/lib/models/trail.dart (verified this session) — freezed fields
// @JsonKey(name: 'max_lat') @Default(0) double maxLat,
// @JsonKey(name: 'max_lon') @Default(0) double maxLon,
// @JsonKey(name: 'min_lat') @Default(0) double minLat,
// @JsonKey(name: 'min_lon') @Default(0) double minLon,
// Trail also exposes trail.bounds -> LngLatBounds (maplibre type) — NOT used
// for the coverage check (that needs raw doubles against RegionEntity's own
// raw doubles), but confirms the same four-field shape is already canonical
// for this model.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Trail-scoped PMTiles download only, no region concept | App-wide region tile repository (this milestone, Phase 22-25.1) | v1.6, Phases 22-25.1 (all Complete) | This phase (26) is the first to connect the two systems — trail downloads now check region coverage before proceeding |
| N/A — no prior "coverage guard" existed | This phase introduces the concept for the first time | Phase 26 (this phase) | Net-new UX; no legacy behavior to migrate away from |

**Deprecated/outdated:** None relevant — this phase builds entirely on Phase 22-25.1's freshly-completed, non-deprecated APIs.

## Assumptions Log

No claims in this research are tagged `[ASSUMED]` — every factual claim about existing code (`RegionEntity`, `Trail`, `TileRepositoryStatus`, `DownloadNotificationService`, `navigatorKey`, `formatBytes`, l10n keys, `track_save_options_sheet.dart`, `settings_offline_regions_screen.dart`) was confirmed via direct `Read`/`grep` of the actual files in this session — tagged `[VERIFIED: codebase read]` throughout. `26-CONTEXT.md` and `26-UI-SPEC.md` (both pre-existing, user-approved documents) are treated as locked decisions per the `<upstream_input>` contract, not as claims requiring re-verification, except where this research found and flagged a concrete discrepancy (Pitfall 3, Anti-Patterns section's `RegionRepository` correction).

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| — | (none) | — | — |

**This table is empty:** All claims in this research were verified directly against the codebase — no user confirmation needed before planning.

## Open Questions (RESOLVED)

1. **Should the D-10 unified notification's title/body extension be a new method or an optional-parameter extension of `showProgress`?**
   - What we know: `showProgress(trailName, done, total)` is called from exactly one place today (`DownloadingTrailIds.download`), so changing its signature is low-risk (single call site) — verified via `grep` showing no other callers.
   - What's unclear: Whether the planner prefers a clean second method (`showAggregateProgress`) to keep `showProgress`'s existing contract frozen, vs. adding nullable `title`/`body` params to `showProgress` itself (fewer methods, slightly busier signature).
   - Recommendation: Either is safe; a new method is marginally cleaner since `showProgress`'s existing 0-region call path must remain byte-for-byte unchanged (GUARD-01), and a separate method makes that invariant visually obvious in a diff.
   - RESOLVED: Planner adopted the new-method approach — `showAggregateProgress` implemented as a sibling method in `26-02-PLAN.md`, `showProgress` left untouched.

2. **Exact combined-size math when a region's DEM checkbox is checked but the region has no `demUrl`**
   - What we know: `settings_offline_regions_screen.dart`'s precedent already gates the DEM row's existence entirely on `region.demUrl != null` (a region with no DEM archive simply has no DEM row/checkbox at all) — verified in this session.
   - What's unclear: Nothing outstanding — this fully resolves the question the task brief raised about DEM-checkbox-without-DEM-archive; the row simply doesn't render (matches UI-SPEC's Component Inventory: "only if the region has a DEM archive").
   - Recommendation: Reuse the identical `region.demUrl != null` gate verbatim in the new sheet; no new decision needed.
   - RESOLVED: `26-02-PLAN.md` reuses the `region.demUrl != null` gate verbatim; no new decision required.

## Environment Availability

Skipped — this phase has no new external tool/service/runtime dependency. All work happens inside the existing Flutter app against already-persisted local ObjectBox data; no network calls are added (D-11 is explicitly local-only), no new native plugin, no new backend endpoint.

## Validation Architecture

Skipped — `.planning/config.json`'s `workflow.nyquist_validation` is explicitly `false`.

For the planner's reference regardless (since `flutter test` conventions still apply for whatever test coverage the plan chooses to include): this codebase uses plain `flutter_test` (no mocking framework beyond what's built in) with a `test/` tree mirroring `lib/`'s structure (`test/util/`, `test/provider/`, `test/models/`, `test/services/`). Pure functions (like the recommended `missingCoverageRegions`/`bboxesOverlap`) are the easiest unit-test target and match the existing `test/util/region_tile_status_util_test.dart` style (table-driven `group`/`test` blocks, verified in this session) — recommended location: `test/util/trail_coverage_util_test.dart`.

## Security Domain

`security_enforcement` is `true` (ASVS level 1, block on `high`) per `.planning/config.json`.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | Phase adds no auth surface — reuses existing authenticated Dio client for region/trail downloads (unchanged) |
| V3 Session Management | No | No session state introduced |
| V4 Access Control | No | No new access-control boundary — coverage check reads only locally-persisted data already scoped to the signed-in device/user via existing ObjectBox store |
| V5 Input Validation | Marginal | Trail/region bbox values are already validated at their respective ingestion points (`RegionEntity.fromCatalogEntry`/`applyCatalogEntry` throw `FormatException` on `bbox.length != 4`, verified this session; Trail bbox arrives via the existing, already-hardened trail API). This phase's own bbox-overlap function should defensively treat any of the 8 doubles as already-trusted local data — no new external input path is introduced, so no new validation surface is required |
| V6 Cryptography | No | No crypto/secrets involved |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Malformed/degenerate bbox (e.g. `minLon > maxLon` from a corrupted local row) causing an incorrect coverage verdict | Tampering (of local data, low severity — device-local, not network-attacker-controlled) | The overlap function should not crash on a malformed bbox; a defensively-written `bboxesOverlap` (Pattern 1) naturally returns `false` for a degenerate `min > max` row rather than throwing, so a corrupted region row is worst-case treated as "does not overlap" (safe default — the guard just won't count that broken region as either covered or missing, and it silently drops out of both lists) |
| Guard becoming a false sense of "fully downloaded"/offline-safe signal | Information disclosure of false confidence, not a security vuln per se | Out of ASVS scope — this is a UX-correctness concern (D-04's explicit non-blocking warning), already addressed by design, not a security control |

No blocking (`high`) security findings — this phase's threat surface is minimal (local-only computation, no new network/auth/crypto path).

## Sources

### Primary (HIGH confidence — direct codebase reads, this session)
- `app/lib/provider/trail/trail_download_state_provider.dart` — `DownloadingTrailIds.download(trail)` full implementation
- `app/lib/entities/region_entity.dart` — `RegionEntity` bbox fields, `status` getter, package `ToOne` relations
- `app/lib/models/region_status.dart` — `CatalogStatus`/`RegionStatus`/`PackageStatus` enums and exact `.code` values
- `app/lib/models/trail.dart` — `Trail` bbox fields (`minLat`/`minLon`/`maxLat`/`maxLon`), `bounds` getter
- `app/lib/entities/trail_entity.dart` — mirrors the same bbox field shape (grep-verified)
- `app/lib/services/trail_download_service.dart` — `downloadTrail(...)` full signature incl. `onProgress(done,total)`
- `app/lib/provider/download_notification_provider.dart` + `app/lib/services/download_notification_service.dart` — full `showProgress`/`showGenerating`/`showSuccess`/`showError` implementation, notification id 42
- `app/lib/provider/region/tile_repository_provider.dart` — `TileRepositoryStatus.downloadVector`/`downloadDem`/`cancelVector`/`cancelDem`/`delete`/`deleteDemPackage`
- `app/lib/provider/region/region_provider.dart` — `RegionRepository` (fetch/upsert/refresh only), `RegionListNotifier`/`regionListNotifierProvider` (the actual local-snapshot read)
- `app/lib/models/region_download_state.dart` — `RegionDownloadState.vectorProgress`/`demProgress`
- `app/lib/models/region_catalog_entry.dart` — `RegionCatalogEntry` field shapes incl. `demStatus`/`demUrl`/`demSize`
- `app/lib/entities/downloaded_tile_package_entity.dart` — `sizeBytesOnDisk`, `status`
- `app/lib/util/byte_format_util.dart` — `formatBytes()` exact implementation
- `app/lib/routes/settings_offline_regions_screen.dart` — full region-row rendering precedent (Vector/DEM tiles, DEM gating on `demUrl != null`, size display logic)
- `app/lib/components/navigation/track_save_options_sheet.dart` — bottom-sheet shape precedent (`showModalBottomSheet` config, drag handle, `_ToggleCard`, full-width `FilledButton`)
- `app/lib/main.dart` (lines 140-240) — `navigatorKey.currentContext` non-widget dialog-trigger precedent (resume-navigation dialog)
- `app/lib/provider/router_provider.dart` — `navigatorKey` declaration (`GlobalKey<NavigatorState>`)
- `app/lib/routes/trail_detail_screen.dart` (~line 130) / `app/lib/components/trail/trail_dropdown.dart` (~line 107) — both existing call sites, confirmed identical `.download(trail)` invocation shape, no `BuildContext` passed today
- `app/lib/i18n/app_en.arb` — confirmed exact l10n keys `regions_vector_tile_title`, `regions_dem_tile_title`, `regions_download_failed`, `regions_update_available` match UI-SPEC's copy table verbatim
- `app/pubspec.yaml` — confirmed no geo/turf-equivalent package present or needed; `objectbox: ^5.3.1` pin
- `.planning/config.json` — `workflow.nyquist_validation: false`, `workflow.security_enforcement: true` (ASVS level 1, block on `high`)
- `app/test/util/region_tile_status_util_test.dart` — existing pure-function unit-test style precedent

### Secondary (MEDIUM confidence)
None used — this phase required no external/web research; every question was resolvable directly against the local codebase.

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new packages, all reuse of already-pinned, already-verified dependencies
- Architecture: HIGH — every pattern cited (sheet shape, navigatorKey trigger, provider reads) was read directly from working, already-shipped code in this exact codebase, not inferred from documentation
- Pitfalls: HIGH — Pitfall 1 (notification signature gap) and the `RegionRepository`-vs-`regionListNotifierProvider` correction were discovered by direct code inspection during this research session, not carried over from CONTEXT.md/UI-SPEC.md assumptions

**Research date:** 2026-07-24
**Valid until:** 30 days (stable, internal-only codebase dependencies; no external API/library version drift risk since zero new packages are introduced)

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| GUARD-01 | On trail download tap, check trail bbox coverage against downloaded/updateAvailable regions before proceeding | Pattern 1 (`missingCoverageRegions`) + Pitfall 4 (distinguishing "fully covered" from "no region at all") give the exact algorithm; verified `RegionEntity.status`/`Trail` bbox field shapes make this a direct, no-guesswork implementation |
| GUARD-02 | If coverage is missing, a dialog names specific missing region(s) + size with a direct in-dialog "Download region" CTA per region | Pattern 3 (sheet shape) + Don't-Hand-Roll's `formatBytes()` entry + verified `settings_offline_regions_screen.dart` row precedent give the exact UI composition; UI-SPEC.md already locks the copy |
| GUARD-03 | Multi-region trail lists all missing regions with individual + combined size, lets user download any subset, never forces full coverage | Pitfall 5 (Download button always enabled) + Pattern 1's per-region list output directly satisfy this; combined-size math is a simple sum over selected regions' `vectorSize`/`demSize` |
| GUARD-04 | `updateAvailable` regions satisfy coverage identically to `downloaded` | Pattern 1's `missingCoverageRegions` filter explicitly checks both `RegionStatus.downloaded` and `RegionStatus.updateAvailable` as satisfying; verified against the real `RegionStatus` enum (Pitfall 3 catches and corrects a related documentation slip in CONTEXT.md's own enum-code transcription) |
</phase_requirements>
