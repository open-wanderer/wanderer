# Phase 24: Settings — Offline Maps/Regions UI - Research

**Researched:** 2026-07-22
**Domain:** Flutter/Dart mobile UI (Riverpod state management, ObjectBox local persistence, Material 3) consuming an already-built download engine
**Confidence:** HIGH

## Summary

Phase 24 builds exactly one new screen and wires it to infrastructure that already exists and is fully functional: Phase 22's `RegionEntity`/`RegionRepository` (catalog + local persistence) and Phase 23's `TileRepositoryManager`/`TileRepositoryStatus` (resumable download engine + ephemeral progress state). No new packages, no new backend endpoints, no new ObjectBox entities. The work is almost entirely "read two existing sibling screens (`settings_categories_screen.dart`, `settings_subcategories_screen.dart`) and reproduce their established list/dialog/toast/AsyncLoader conventions against a new data source."

Three things are **not** yet built anywhere in the codebase and must be created fresh by this phase's plan: (1) a synchronous ObjectBox-reading Riverpod provider that exposes the region list to the UI (`TrailLibraryNotifier` in `trail_library_provider.dart` is the exact structural precedent — no such provider exists yet for `RegionEntity`), (2) a disk-usage aggregation utility that reads actual on-disk byte counts rather than trusting the persisted `sizeBytesOnDisk` field (which Phase 23's engine only ever populates on successful completion, never for an in-progress or paused partial `.part` file — a genuine gap against D-06's stated requirement), and (3) the `deleteDemPackage` method D-01 requires on both `TileRepositoryManager` and `TileRepositoryStatus` (currently absent from both files, confirmed by direct inspection).

**Primary recommendation:** Mirror `settings_categories_screen.dart` structurally (ConsumerStatefulWidget, `_save` error-toast wrapper, confirm-dialog pattern) but back it with a new synchronous `regionListProvider` (mirrors `trail_library_provider.dart`, not the async `categoryPreferenceProvider` pattern) so the list renders instantly from local ObjectBox data and never blocks on network. Explicitly re-fetch (`ref.invalidate`) that provider after every `TileRepositoryStatus` mutation and after catalog refresh — ObjectBox's `ToOne.target` is cached per-instance after first read, so a stale `RegionEntity` held across a rebuild will not reflect a sibling code path's write.

## Architectural Responsibility Map

This is a mobile app; tiers are Flutter-specific, not the web browser/SSR/API/CDN split.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Region list content (name, catalog status, sizes) | ObjectBox Local Store | Flutter UI (render) | Already persisted by Phase 22; this phase adds the read-provider, not new storage |
| Search/filter by name (D-08) | Flutter UI | — | Pure client-side substring match over an already-loaded in-memory list; no new fetch |
| Download / pause / resume / delete actions | TileRepositoryManager (download engine) | Riverpod State (`TileRepositoryStatus`) | Phase 23 already owns start/pause/resume/delete; this phase's screen is a caller, not an owner |
| DEM-only delete (D-01, net-new) | TileRepositoryManager (new method) | Riverpod State (new notifier method) | Still engine-tier work per CONTEXT.md's own framing — "additive to the existing engine, not new scope" |
| Per-row combined progress (D-07) | Riverpod State (`TileRepositoryStatus`/`RegionDownloadState`) | Flutter UI | `vectorProgress`/`demProgress` already tracked; combining them into one bar is a pure UI computation |
| Disk usage summary (D-06) | Filesystem (file stat) + ObjectBox Local Store | Flutter UI | New pure utility — persisted `sizeBytesOnDisk` alone is insufficient (see Pitfall 1) |
| Catalog refresh trigger | Backend API (`RegionRepository.fetchCatalog`) | ObjectBox Local Store (`upsertCatalog`) | Already built (Phase 22); this phase only decides the call site and the failure UX |
| `updateAvailable` badge (D-05) | ObjectBox Local Store (`RegionEntity.status` computed getter) | Flutter UI | Status resolution already exists; UI only renders the persistent banner |

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SETUI-01 | Flat, searchable region list (no tree) | `RegionEntity` box has no hierarchy; new `regionListProvider` returns a flat `List<RegionEntity>`; `WandererSearchBar` (existing, debounced 500ms) provides the search input — filter client-side on `.name` per D-08 |
| SETUI-02 | Name, 4/6-state status, size breakdown before download | `RegionEntity.status` getter (6 values, D-04) + `vectorSize`/`demSize` (already populated from catalog fetch, no download needed) |
| SETUI-03 | Download/pause/resume/delete per region, visible progress | `TileRepositoryStatus.downloadVector/downloadDem/pause/resume/delete` — all four already exist and are called directly; progress via `RegionDownloadState.vectorProgress/demProgress` |
| SETUI-04 | Independent per-region DEM toggle | `region.demPackage` is an independent `ToOne`, already decoupled from `vectorPackage` per Phase 22/23; D-01's new `deleteDemPackage` is the only missing piece (toggle-on already works via `downloadDem`) |
| SETUI-05 | Total disk usage summary | Requires a **new** utility — see Pitfall 1; do not sum `sizeBytesOnDisk` naively |
| SETUI-06 | `updateAvailable` non-blocking badge + optional update action | `RegionStatus.updateAvailable` already resolved by `RegionEntity.status`; "update" action is just re-invoking `downloadVector` (same call as initial download — Phase 23's resumable/overwrite path handles it) |

## Standard Stack

### Core

This phase adds **zero new dependencies**. Everything is already in `app/pubspec.yaml` (verified 2026-07-22, `flutter --version` → 3.44.2 / Dart 3.12.2 installed and current):

| Library | Version (pubspec) | Purpose | Why Standard (already used identically elsewhere) |
|---------|--------------------|---------|-----------------------------------------------------|
| `flutter_riverpod` | ^3.3.1 | State management | Every existing Settings screen uses `ConsumerStatefulWidget`/`ConsumerWidget` |
| `riverpod_annotation` | ^4.0.2 | Code-gen providers | `tile_repository_provider.dart`, `region_provider.dart` already use `@Riverpod(keepAlive: true)` / `@riverpod` |
| `go_router` | ^17.2.1 | Navigation | New route nests under existing `/settings` `GoRoute` exactly like `categories`/`account`/etc. |
| `font_awesome_flutter` | ^11.0.0 | Icons | `FontAwesomeIcons.download/pause/play/trash/circleCheck/circleExclamation/map` all already used elsewhere in this exact semantic role |
| `objectbox` | ^5.3.1 | Local persistence | `RegionEntity`/`DownloadedTilePackageEntity` boxes already exist (Phase 22/23) |
| `skeletonizer` | ^2.1.3 | Loading skeleton | Via `AsyncLoader` — only needed if this phase wraps the *initial* catalog fetch in an `AsyncValue` (see Architecture Patterns) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled `formatBytes()` util | `filesize` / `human_bytes` pub packages | Not in pubspec.yaml; UI-SPEC (already approved) explicitly specifies the exact format ("one decimal place, unit steps at KB/MB/GB") — a ~10-line pure function is simpler than adding a dependency for one call site, and avoids a fresh legitimacy-audit package addition for zero benefit |
| Synchronous `regionListProvider` (recommended) | Reactive ObjectBox `Stream<Query<RegionEntity>>` (live auto-updating query) | ObjectBox-Dart supports `box.query(...).watch(triggerImmediately: true)` for a live stream, but **no existing provider in this codebase uses that pattern** (grepped `app/lib/provider/**` — zero matches for box-stream watching). Every existing box-backed provider (`trail_library_provider.dart`) uses the read-once-then-manually-invalidate pattern instead. Introducing the stream pattern here would be the first of its kind and a bigger deviation from established conventions than the phase's decisions warrant — recommend matching `TrailLibraryNotifier` instead. |

**Installation:** None required.

## Package Legitimacy Audit

**Not applicable.** This phase introduces zero new third-party packages (confirmed against `app/pubspec.yaml` and the UI-SPEC's own "zero new packages" statement). The `slopcheck`/registry-verification gate is skipped; nothing to audit.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ SettingsScreen                                                  │
│   ListTile("Offline Maps/Regions") ──push──▶ /settings/regions  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ OfflineRegionsScreen (ConsumerStatefulWidget, NEW)               │
│                                                                    │
│  initState/build:                                                 │
│    1. ref.watch(regionListProvider)      ── sync snapshot ──┐    │
│    2. fire-and-forget refreshCatalog()   ── async, non-block│    │
│    3. ref.watch(tileRepositoryStatusProvider) ── ephemeral  │    │
│                                                               │    │
│  ┌────────────────────────────────────────────────────────┐│    │
│  │ Disk usage summary card (D-06)                          ││    │
│  │  = regionDiskUsageBytes(store, regions)  [NEW utility]  ││    │
│  └────────────────────────────────────────────────────────┘│    │
│  ┌────────────────────────────────────────────────────────┐│    │
│  │ WandererSearchBar (existing, debounced 500ms)            ││    │
│  │  → filters by RegionEntity.name only (D-08)              ││    │
│  └────────────────────────────────────────────────────────┘│    │
│  ┌────────────────────────────────────────────────────────┐│    │
│  │ Region row (per RegionEntity, sorted A-Z, D-09)          ││    │
│  │  catalogStatus != ready?                                 ││    │
│  │    → disabled row, label, NO status/action rendering ────┼─── PRECEDENCE
│  │  else → render RegionStatus (6 states, D-04) + actions   ││    │  RULE
│  │        + combined progress (D-07, from ephemeral state)  ││    │  (Pitfall 3)
│  │        + DEM Switch (independent, D-04 SETUI)            ││    │
│  └────────────────────────────────────────────────────────┘│    │
└──────────────────────────────┬────────────────────────────────┘
                                │ action calls
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ TileRepositoryStatus (Riverpod, EXISTING, keepAlive)              │
│  downloadVector / downloadDem / pause / resume / delete           │
│  + deleteDemPackage  [NEW method, D-01]                           │
└──────────────────────────────┬────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│ TileRepositoryManager (EXISTING service)                          │
│  writes DownloadedTilePackageEntity + RegionEntity.ToOne targets   │
│  + deleteDemPackage  [NEW method, D-01]                           │
└──────────────────────────────┬────────────────────────────────┘
                                │ ObjectBox write ── triggers ──▶ ref.invalidate(regionListProvider)
                                ▼                                    (Pitfall 1: ToOne caching)
┌─────────────────────────────────────────────────────────────────┐
│ ObjectBox: RegionEntity box, DownloadedTilePackageEntity box       │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

No new top-level folders — follows existing flat conventions:

```
app/lib/
├── routes/
│   └── settings_offline_regions_screen.dart   # NEW — the screen itself
├── provider/region/
│   ├── region_provider.dart                    # EXISTING — add regionListProvider here (same file, same tier)
│   └── tile_repository_provider.dart           # EXISTING — add deleteDemPackage notifier method
├── services/
│   └── tile_repository_manager.dart            # EXISTING — add deleteDemPackage method
└── util/
    ├── byte_format_util.dart                   # NEW — formatBytes(int) → "45 MB" / "2.4 GB"
    └── region_disk_usage_util.dart             # NEW — sums actual on-disk bytes (see Pitfall 1)
```

### Pattern 1: Synchronous ObjectBox Snapshot Provider (not AsyncValue)

**What:** A plain `@riverpod` class whose `build()` reads `store.box<RegionEntity>().getAll()` once and returns a `List<RegionEntity>`, with mutation call sites (or an external `ref.invalidate`) refreshing it.
**When to use:** Any screen reading a local ObjectBox box as its primary list source, when the data doesn't need to be an `AsyncValue` (no network round-trip on read).
**Example:**
```dart
// Source: app/lib/provider/trail/trail_library_provider.dart (existing, verbatim structural precedent)
@riverpod
class TrailLibraryNotifier extends _$TrailLibraryNotifier {
  @override
  List<Trail> build() {
    final store = ref.watch(objectBoxProvider);
    final box = store.box<TrailEntity>();
    final trails = box.getAll().map((t) => t.toModel()).toList();
    trails.sort((a, b) => b.created.compareTo(a.created));
    return trails;
  }
  // mutation methods update `state` directly or call ref.invalidateSelf()
}
```
Apply this exact shape for `regionListProvider`, sorting alphabetically by `name` (D-09) instead of by `created`.

### Pattern 2: Row-Level Precedence — `catalogStatus` Gates Before `RegionStatus`

**What:** Two independent enums drive a region row's rendering: `RegionEntity.catalogStatus` (backend build state: `building`/`ready`/`error`) and the computed `RegionEntity.status` (local download lifecycle: 6 values, D-04). D-09 requires `building`/`error` catalog rows to render as disabled with a specific caption and **no download action at all** — regardless of what the local `status` getter would otherwise compute (which is always `notDownloaded` in that case, since no package row exists yet per Phase 22's D-09).
**When to use:** Every region row's build method.
**Example (pseudocode, not yet written anywhere):**
```dart
Widget _buildRow(RegionEntity region) {
  if (region.catalogStatus == CatalogStatus.building) {
    return _disabledRow(region, caption: l10n.regions_not_yet_available);
  }
  if (region.catalogStatus == CatalogStatus.error) {
    return _disabledRow(region, caption: l10n.regions_build_failed);
  }
  // catalogStatus == ready: fall through to the 6-state RegionStatus.status switch
  return _activeRow(region, region.status);
}
```
This precedence rule is not written down anywhere in CONTEXT.md/UI-SPEC as an explicit "check this first" instruction — it's a synthesis of D-04 + D-09 that the planner must encode explicitly or a `building`/`error` region will incorrectly render an active "Download" button.

### Pattern 3: DEM Toggle Visibility Gated on `demUrl` Presence

**What:** Per Phase 22's D-09/API contract, `region.demUrl` is only ever non-null when `demStatus == CatalogStatus.ready` AND the backend file exists. There is no scenario where a DEM toggle should render enabled-but-empty.
**When to use:** Rendering the DEM `Switch` (SETUI-04) — render it only when `region.demUrl != null`; omit entirely (not disabled/greyed) when `null`, mirroring the "no ghost UI for unavailable data" convention already used elsewhere (e.g. `settings_categories_screen.dart` only renders subcategory chips `if (subs.isNotEmpty)`).

### Anti-Patterns to Avoid

- **Reusing `deleteRegion()` for the DEM toggle-off path:** D-01 is explicit that `deleteRegion()` (Phase 23) is the wrong granularity — it deletes both packages together. The DEM-only path needs the new `deleteDemPackage()` method.
- **Trusting `region.vectorPackage.target?.status` from a `RegionEntity` instance held across an `await`:** see Pitfall 1 below.
- **Gating the whole screen behind a blocking `AsyncLoader<List<RegionEntity>>`-style error state on every catalog refresh:** see Pitfall 4 below — this would hide already-downloaded, fully-usable-offline regions the moment the device has no network.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Resumable download, disk pre-check, PMTiles validation | A new download loop for the DEM-delete or retry paths | `TileRepositoryManager.startVectorDownload`/`startDemDownload` (unchanged, called again for retry per D-03) | Already handles Range resume, disk-space fail-closed check, and archive validation — re-invoking the existing method IS the correct "retry from scratch" per D-03's own wording |
| Region id / path safety | String-concatenating a region id into a file path anywhere in the new UI code | `assertValidRegionId`/`regionVectorPath`/`regionDemPath` (`region_file_path.dart`) | Already the sanctioned path-builder; the new disk-usage utility must route through these too, not re-derive paths |
| Confirm-before-destructive-action dialog | A bespoke dialog widget for region delete | `AlertDialog` + `showDialog<bool>` exactly as `settings_categories_screen.dart`'s `_onToggleOff` and `settings_account_screen.dart`'s `_deleteAccount` already do | Byte-for-byte reusable pattern already proven in two sibling screens |
| Error-only toast surfacing | A new toast/snackbar mechanism | `toastProvider` (`ToastMessage(type: ToastType.error, icon: FontAwesomeIcons.circleExclamation, ...)`) | Established app-wide convention, zero reason to deviate |

**Key insight:** Every piece of *download engine* logic this phase needs already exists and is fully tested (Phase 23's `tile_repository_manager_test.dart`, `region_entity_test.dart`, etc.). The only genuinely new code is: one engine method (D-01), one UI screen, one snapshot provider, and two small utility functions (byte formatting, disk usage aggregation).

## Common Pitfalls

### Pitfall 1: `sizeBytesOnDisk` Does Not Reflect In-Progress or Paused Downloads

**What goes wrong:** D-06 requires the disk-usage summary to include partial `.part` file bytes for `downloading` and `paused` packages, "because partial files genuinely occupy disk space." But `TileRepositoryManager._updatePackageStatus` (the only writer of `sizeBytesOnDisk`) only ever passes a non-null `sizeBytesOnDisk` on the successful-completion path (`startVectorDownload`/`startDemDownload`'s happy path, after `PmTilesArchive` validation and rename to the final path). On the `CancelToken.isCancel` (pause) branch, `sizeBytesOnDisk` is never touched — it stays at whatever it was before (typically `null`, since a package only reaches `paused` mid-download, before ever reaching `downloaded`). Naively summing `sizeBytesOnDisk` across all packages will silently under-report disk usage for every downloading/paused region, directly contradicting D-06.
**Why it happens:** The field was designed (Phase 23) purely to record the final archive size for completed downloads — it was never a "live progress" field, and Phase 24 is the first consumer to need current-moment disk usage across all states.
**How to avoid:** Build a new utility that, for every persisted region, checks the on-disk byte length directly via `File(regionVectorPath(root, id)).lengthSync()` (final file, if `status == downloaded`) OR `File('${regionVectorPath(root, id)}.part').lengthSync()` (partial file, if `status == downloading` or `paused`), falling back to 0 if neither exists. Do the same for the DEM path. This keeps the disk-usage number accurate without any change to Phase 23's engine (which CONTEXT.md scopes as untouched except for D-01).
**Warning signs:** Disk usage total looks correct only when no downloads are in-progress/paused; a manual on-device test with a paused download will expose an under-count if this isn't handled.

### Pitfall 2: ObjectBox `ToOne.target` Caches After First Read

**What goes wrong:** ObjectBox-Dart's `ToOne` (confirmed via the installed package source, `~/.pub-cache/hosted/pub.dev/objectbox-5.3.2/lib/src/relations/to_one.dart`): *"[ToOne] uses lazy initialization, so on first access this will read the target object from the database"* — implying the read is cached on that specific `ToOne` instance afterward. A `RegionEntity` instance held in a provider's state across a download action's `await` will keep returning its **stale** cached `vectorPackage.target?.status` even after `TileRepositoryManager` writes a fresh status to the same underlying row via a different `RegionEntity`/`ToOne` instance.
**Why it happens:** `regionListProvider`'s `build()` produces one snapshot of `RegionEntity` objects; `TileRepositoryManager` internally re-queries `_regionById()` for its own working copy and mutates that separate instance's box row. The two instances share no in-memory identity.
**How to avoid:** After every `TileRepositoryStatus` action (`downloadVector`/`downloadDem`/`pause`/`resume`/`delete`/new `deleteDemPackage`) completes, explicitly `ref.invalidate(regionListProviderProvider)` (or the screen re-reads via `ref.refresh`) so a **fresh** `box.getAll()` produces new `RegionEntity`/`ToOne` instances with unresolved (thus freshly-queried) targets. Do not rely on Riverpod's own dependency graph to do this automatically — `regionListProvider` and `tileRepositoryStatusProvider` are not `ref.watch`-linked to each other today.
**Warning signs:** A region's row appears stuck on "Downloading…" or "Not downloaded" after a download actually completes, until the screen is popped and re-pushed (which forces a fresh `build()`).

### Pitfall 3: `catalogStatus` vs. Computed `RegionStatus` — Two Independent Axes

See Architecture Pattern 2 above. **Warning sign:** a region whose backend archive build failed (`catalogStatus == error`) shows an active, tappable "Download" button that then throws `StateError('Region $id has no vector archive available yet')` from `TileRepositoryManager.startVectorDownload` (which explicitly checks `region.vectorUrl == null` and throws) — because a `building`/`error` catalog region's local `RegionStatus` genuinely does compute to `notDownloaded`.

### Pitfall 4: A Blocking Catalog-Refresh Error Would Hide Already-Downloaded, Usable-Offline Regions

**What goes wrong:** The approved UI-SPEC (written before this RESEARCH.md existed, which it explicitly permits RESEARCH.md to correct on implementation mechanics) says to "reuse `AsyncLoader` + `WandererError` verbatim" for "Error state — initial catalog fetch fails." If implemented as a screen-wide `AsyncLoader<List<RegionEntity>>` gating the *entire* list on the network catalog fetch's `AsyncValue`, then a user who is offline (or whose instance's `/regions` endpoint is briefly down) would see a full-screen error and **lose access to managing regions they already downloaded and can already use offline** — actively contradicting the offline-first purpose of this whole milestone.
**Why it happens:** `RegionRepository.refreshCatalog()` (Phase 22, D-03) is explicitly designed so "a fetch failure always happens before any store write" — i.e., failure is safe and inert for already-persisted data — but the UI-SPEC's error-state prescription doesn't distinguish "first-ever launch, zero cached regions, fetch fails: nothing to show" from "regions already cached locally, fetch fails: stale-but-usable list."
**How to avoid:** Render the region list from the synchronous `regionListProvider` snapshot **unconditionally** (it always has *something*, even if empty). Trigger `refreshCatalog()` as a fire-and-forget call on screen open. On failure: if `regionListProvider`'s snapshot is non-empty, surface an error **toast** only (matches `RegionCatalogException`'s own doc comment: "Phase 24's Settings/Regions screen decides how to surface this to the user" — a toast is a valid decision, not mandated to be a full-screen error). Reserve the full-screen `AsyncLoader`/`WandererError` treatment for the one genuine edge case the UI-SPEC's copy already anticipates separately ("Empty state — catalog itself is empty… REGN-01") combined with a fetch failure on that same first load. This does not change any UI-SPEC *copy* (still "Something went wrong" generic text) — only *when* it's shown.
**Warning signs:** Manual offline test: open the Regions screen with airplane mode on and at least one previously-downloaded region — if the screen goes blank/errors instead of showing that region, this pitfall was not addressed.

### Pitfall 5: `TileRepositoryManager` Is Never Disposed (Pre-Existing, Not This Phase's Bug — But Relevant)

**What goes wrong (informational only):** `tileRepositoryManagerProvider` is `@Riverpod(keepAlive: true)` and its `TileRepositoryManager.dispose()` method (which cancels the `AppLifecycleListener` and any in-flight `CancelToken`s) is never called from `main.dart` or anywhere else in the app today (grepped, zero call sites). This is a Phase 23 concern, not something Phase 24 needs to fix — flagging only so the planner doesn't assume disposal wiring exists and try to hook into it.
**How to avoid:** Not in scope for this phase. Do not add a `dispose()` call as part of this phase's screen (the manager is app-lifetime `keepAlive`, and the screen unmounting should not tear down a repository other screens/Phase 25's map pipeline will also depend on).

## Code Examples

### Combined Progress Bar (D-07)

```dart
// Pattern only — RegionDownloadState already has both fields
// (app/lib/models/region_download_state.dart, existing)
double? combinedProgress(RegionDownloadState? s) {
  if (s == null) return null;
  final parts = [s.vectorProgress, s.demProgress].whereType<double>();
  if (parts.isEmpty) return null;
  return parts.reduce((a, b) => a + b) / parts.length;
}
```

### Byte Formatting (new utility, per UI-SPEC's stated convention)

```dart
// NEW file: app/lib/util/byte_format_util.dart
// Convention per 24-UI-SPEC.md: one decimal place, unit steps at KB/MB/GB.
String formatBytes(int bytes) {
  const kb = 1024;
  const mb = kb * 1024;
  const gb = mb * 1024;
  if (bytes >= gb) return '${(bytes / gb).toStringAsFixed(1)} GB';
  if (bytes >= mb) return '${(bytes / mb).toStringAsFixed(1)} MB';
  if (bytes >= kb) return '${(bytes / kb).toStringAsFixed(1)} KB';
  return '$bytes B';
}
```

### Retry Action (D-03) — Re-Invoke the Existing Download Call

```dart
// Source: derived from tile_repository_provider.dart's existing methods —
// D-03 says retry "re-invokes downloadVector()/downloadDem() from scratch
// for whichever package failed." No new engine method needed for this.
void _onRetry(RegionEntity region) {
  if (region.vectorPackage.target?.status == PackageStatus.error) {
    ref.read(tileRepositoryStatusProvider.notifier).downloadVector(region.id);
  }
  if (region.demPackage.target?.status == PackageStatus.error) {
    ref.read(tileRepositoryStatusProvider.notifier).downloadDem(region.id);
  }
}
```

## State of the Art

Not applicable in the "library deprecation" sense — this is internal-only integration work against code written in the same milestone (Phase 22/23, days prior). There is no external ecosystem drift to account for.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Orphaned regions (`inCatalog == false`, dropped from a catalog refresh) should still appear in the Settings list if previously downloaded, so the user can still see/delete them. Neither `24-CONTEXT.md` nor `24-UI-SPEC.md` explicitly decides this — it falls outside all 6 documented decisions. | Open Questions (below), `regionListProvider` filtering | If wrong, either a downloaded-but-orphaned region silently disappears from the UI while still consuming disk space (bad — user can't find it to delete), or an admin-removed region stays visible forever with no visual distinction (confusing). Low-medium risk; recommend surfacing to discuss-phase or plan-checker before finalizing the filter predicate. |
| A2 | The "update" action for `updateAvailable` (SETUI-06) is implemented as a plain re-call to `downloadVector(regionId)` (same call as a first download) rather than a distinct code path. Based on reading `TileRepositoryManager.startVectorDownload`'s unconditional behavior (always re-downloads to `finalPath`, updates `lastDownloadedVersion` on success) — there is no separate "update" method anywhere in Phase 23's engine. | Code Examples / SETUI-06 support | Low risk — if wrong, the only consequence is a slightly different call site than assumed; the underlying download mechanics are unaffected either way. |

## Open Questions

1. **Are orphaned (`inCatalog == false`) but still-downloaded regions shown in the list?**
   - What we know: Phase 22's D-08 explicitly preserves the row and on-disk files when a region drops out of the catalog ("Downloaded files/packages are left on disk untouched"). Phase 24's CONTEXT.md never revisits this for the UI layer.
   - What's unclear: Whether `regionListProvider` should filter to `inCatalog == true` only, or show all persisted rows regardless, possibly with a distinct visual treatment for orphaned-but-downloaded regions.
   - Recommendation: Default to showing all persisted `RegionEntity` rows regardless of `inCatalog` (safest — never hides disk-consuming data from the disk-usage-owning screen), and flag this as a one-line clarifying question for `/gsd-plan-phase` or the plan-checker rather than blocking on it. If a region is `inCatalog == false` AND `notDownloaded` (never downloaded, then vanished from catalog before ever being fetched-and-shown), it's reasonable to exclude it since the user never knew it existed.

2. **Does the "update" action need a distinct spinner/label state from a first-time download, given the row already shows `updateAvailable`'s persistent banner (D-05) throughout?**
   - What we know: D-05 says the region "continues to appear/behave as downloaded while the badge is shown" — so during the update's re-download, is the row still showing the update banner, or does it flip to the normal `downloading` row treatment?
   - What's unclear: CONTEXT.md doesn't specify the transient state during an update-triggered re-download.
   - Recommendation: Treat it identically to the normal `downloading` state (D-04's row treatment) once the update button is tapped — the banner's whole purpose (per D-05) is pre-tap visibility; mid-download, `RegionStatus.downloading` already takes over via the same `TileRepositoryStatus` ephemeral state used for a first download. No special-casing needed.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK | Entire app build/run | ✓ | 3.44.2 (stable) | — |
| Dart SDK | Entire app build/run | ✓ | 3.12.2 | — |
| Existing backend `/api/v1/regions` endpoint | Catalog refresh (already built, Phase 21.5) | Not re-verified this session (no new endpoint introduced by this phase) | — | — |

No new external dependencies are introduced by this phase; nothing to gate.

## Security Domain

`security_enforcement` is enabled (ASVS Level 1, block on `high`). This phase's attack surface is minimal — it is a local Settings screen consuming already-authenticated, already-validated infrastructure; no new network endpoints, no new user-input-to-query paths beyond a client-side substring filter.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | Reuses the existing cookie-authenticated `apiProvider` Dio client (Phase 22's `RegionRepository`); no new auth surface |
| V3 Session Management | No | Not touched by this phase |
| V4 Access Control | No | Region catalog/downloads are instance-scoped, not per-user-permissioned; no new authorization decision introduced |
| V5 Input Validation | Marginal — already handled upstream | The only user input this phase adds is the search box's free-text string, used exclusively as an in-memory `String.contains`/`toLowerCase` substring match against already-loaded `RegionEntity.name` values — never interpolated into a file path, query, or request. Region ids (the actual security-sensitive string in this domain) are validated by `assertValidRegionId` (Phase 23, unchanged) before ever reaching a filesystem path; this phase's new disk-usage utility must route through that same validator/path-builder rather than re-deriving paths, to inherit the existing protection (see Don't Hand-Roll table) |
| V6 Cryptography | No | Not touched |

### Known Threat Patterns for this Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Path traversal via a malformed/hostile region id reaching a filesystem path | Tampering | Already mitigated by `region_file_path.dart`'s `assertValidRegionId` (regex allow-list, throws `ArgumentError`) — this phase's new disk-usage utility MUST reuse `regionVectorPath`/`regionDemPath`, never construct paths independently |
| Unbounded disk fill from a malicious/misconfigured region declaring a tiny `vectorSize`/`demSize` but streaming an unbounded response | Denial of Service | Already mitigated upstream by `hasEnoughSpace`'s pre-check (Phase 23, unchanged) — outside this phase's scope to re-verify, but worth noting no new bypass is introduced by adding UI call sites to the same download methods |

## Sources

### Primary (HIGH confidence — direct codebase inspection, 2026-07-22)

- `app/lib/entities/region_entity.dart` — `RegionEntity.status` computed getter, catalog vs. local field split
- `app/lib/entities/downloaded_tile_package_entity.dart` — `sizeBytesOnDisk` write sites (confirmed only set on completion)
- `app/lib/models/region_status.dart` — `CatalogStatus`/`RegionStatus`/`PackageStatus` enum definitions
- `app/lib/models/region_download_state.dart` — ephemeral progress state shape
- `app/lib/services/tile_repository_manager.dart` — full download/pause/resume/delete implementation, confirmed `deleteDemPackage` absent
- `app/lib/provider/region/tile_repository_provider.dart` — `TileRepositoryStatus` notifier, confirmed `deleteDemPackage` absent
- `app/lib/provider/region/region_provider.dart` — `RegionRepository`/`RegionCatalogException`/`refreshCatalog`, confirmed no existing call site
- `app/lib/routes/settings_categories_screen.dart`, `settings_subcategories_screen.dart`, `settings_screen.dart`, `settings_account_screen.dart` — structural precedent for list/dialog/toast/routing conventions
- `app/lib/provider/trail/trail_library_provider.dart` — the synchronous ObjectBox-snapshot provider pattern this phase should mirror
- `app/lib/components/base/wanderer_searchbar.dart`, `app/lib/components/async_loader.dart`, `app/lib/provider/toast_provider.dart` — reusable components confirmed present and matching UI-SPEC's claims
- `app/lib/provider/router_provider.dart` (lines ~192-234) — exact `GoRoute` nesting pattern under `/settings`
- `~/.pub-cache/hosted/pub.dev/objectbox-5.3.2/lib/src/relations/to_one.dart` — `ToOne` lazy-caching doc comment (source of Pitfall 2)
- `app/lib/i18n/app_en.arb` — confirmed no existing l10n keys for any of this phase's copy (grepped for region/offline/DEM-related strings)
- `app/pubspec.yaml` — confirmed no new packages needed
- Direct `flutter --version`/`dart --version` shell probe — 2026-07-22, Flutter 3.44.2 / Dart 3.12.2

### Secondary (MEDIUM confidence)

None — this phase's entire technical surface was directly verifiable in the local codebase; no external ecosystem research was needed.

### Tertiary (LOW confidence)

None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, every convention directly read from existing sibling files
- Architecture: HIGH — the two-tier (catalogStatus/RegionStatus) precedence rule and the ToOne-caching pitfall were derived from direct source reading, not inference
- Pitfalls: HIGH — all five pitfalls trace to specific lines of already-existing code or installed-package doc comments, not speculation

**Research date:** 2026-07-22
**Valid until:** 2026-08-21 (30 days — internal-only integration research against code from the same milestone; low drift risk, but re-verify if Phase 23's engine is modified before Phase 24 executes)
