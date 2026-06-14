---
phase: 05-cache-write-fallback-ui
verified: 2026-06-14T17:30:00Z
status: human_needed
score: 4/4 must-haves verified
overrides_applied: 0
re_verification: false
human_verification:
  - test: "Download a trail while connected. Then disable all network access on the device and tap Navigate on that trail. Confirm that navigation launches without an error toast and the wifi-off icon appears in the maneuver banner."
    expected: "Navigation screen opens, shows step-by-step maneuvers from the cached ObjectBox data, and displays the wifi_off icon at the trailing edge of the active banner row."
    why_human: "Requires a real device, a network toggle (airplane mode), a previously-downloaded trail with GPX data, and a live Valhalla service to have run at download time. Cannot simulate DioException fallback path with grep."
  - test: "Launch navigation while connected. Confirm the wifi-off icon is NOT visible during normal online navigation."
    expected: "NavigationScreen shows maneuver instructions with no wifi-off icon. Cache is silently refreshed in the background (no user-visible change)."
    why_human: "Visual absence of icon requires live render. Background re-cache (unawaited) cannot be observed without ObjectBox inspection."
  - test: "While connected, download a trail. Disconnect network. Tap Navigate. Confirm toast says 'couldnt_start_navigation' when the trail has no downloaded GPX (no navCacheJson in ObjectBox)."
    expected: "Error toast appears, not a crash or blank screen."
    why_human: "Requires a trail entity with null navCacheJson (no GPX at download time or Valhalla outage during download) to exercise the fallback error-toast path."
---

# Phase 5: Cache Write + Fallback + UI Verification Report

**Phase Goal:** Hikers can follow downloaded trails step-by-step without a network connection, and the app silently keeps the cache current after each online session
**Verified:** 2026-06-14T17:30:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | After a trail is downloaded, navigation can be launched without a network connection and the maneuver list is served from ObjectBox (OFFLINE-01, OFFLINE-02) | VERIFIED | `trail_download_service.dart` lines 80-119: best-effort try/catch POSTs to `/valhalla/navigate`, sets `entity.navCacheJson = jsonEncode(response.toJson())` before `box.put`. `navigation_launch_util.dart` `on DioException` block reads `_readCachedNav(store, trail.id)` and pushes `extra: (cached, true)`. Both code paths exist and are wired. |
| 2 | When the network call succeeds, navigation launches normally with no user-visible change; the local cache is silently updated for future offline use (OFFLINE-03) | VERIFIED | `navigation_launch_util.dart` line 210: `context.push(..., extra: (response, false))`. Line 214: `unawaited(_recacheNav(store, trail.id, response))`. Re-cache is fire-and-forget; `_recacheNav` swallows all errors (lines 52-66). |
| 3 | When navigation falls back to the cache, a distinct icon appears in the NavigationScreen AppBar indicating offline mode (OFFLINE-04) | VERIFIED | `navigation_screen.dart` line 28: `final bool isOffline`. Lines 359-366: `if (widget.isOffline) ...[const SizedBox(width: 8), Icon(Icons.wifi_off, color: colorScheme.onSurface, size: 20)]` appended to `_buildActiveBannerContent` Row. Router at `router_provider.dart` lines 202-209 unpacks `(NavigateResponse, bool)` record and passes `isOffline` to constructor. |
| 4 | A Valhalla outage during trail download does not block or error the download — the cache step is best-effort and silent (OFFLINE-01) | VERIFIED | `trail_download_service.dart` lines 80-119: the entire cache block is inside `try { ... } catch (_) { }` with bare swallowing catch (no rethrow). The tile-download rethrow block (lines 57-67) is separate and unaffected. `box.put(entity)` at lines 121-123 is outside the cache try/catch. |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/gpx_util.dart` | `buildNavShape(List<LatLng>)` top-level helper, ≤500 downsampling with first/last preserved | VERIFIED | Lines 19-45: function exists as public top-level, implements `points.length > 500` branch with `(points.length / 499).ceil()`, coordinate-value dedup fix. Doc comment references OFFLINE-01/D-08. |
| `app/test/util/gpx_util_test.dart` | Unit tests proving downsampling cap, first/last preservation, and 501-point dedup | VERIFIED | File exists, imports `package:wanderer/util/gpx_util.dart`, contains `group('buildNavShape', ...)` with 4 `test(` calls covering 2-point, 500-point, 1000-point, and 501-point cases. |
| `app/lib/routes/navigation_screen.dart` | `isOffline` constructor param + conditional `Icons.wifi_off` in active banner row | VERIFIED | Line 28: `final bool isOffline`. Line 34: `this.isOffline = false`. Lines 359-366: conditional spread with `SizedBox(width: 8)` + `Icon(Icons.wifi_off, colorScheme.onSurface, size: 20)`. No placeholder. |
| `app/lib/provider/router_provider.dart` | Navigate route unpacks `(NavigateResponse, bool)` record | VERIFIED | Lines 202-209: `if (extra is! (NavigateResponse, bool))` fallback to `TrailDetailScreen`, then `final (response, isOffline) = extra;`, then `NavigationScreen(id: trailId, response: response, isOffline: isOffline)`. |
| `app/lib/util/navigation_launch_util.dart` | DioException catch → ObjectBox cache read → push `isOffline:true`; success → push `isOffline:false` + unawaited re-cache | VERIFIED | Lines 178-251: try block pushes `extra: (response, false)` then `unawaited(_recacheNav(...))`. `on DioException catch (_)` reads cache, validates, pushes `extra: (cached, true)` or shows toast. Generic `catch (_)` shows toast. Private helpers `_readCachedNav` and `_recacheNav` present at lines 26-66. |
| `app/lib/services/trail_download_service.dart` | Best-effort Valhalla cache write before `box.put` inside `downloadTrail` | VERIFIED | Lines 80-119: `buildNavShape(points)` called, POST to `/valhalla/navigate`, `entity.navCacheJson = jsonEncode(response.toJson())` guarded by `maneuvers.isNotEmpty && shape.isNotEmpty`, entire block in swallowing `catch (_)`. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `navigation_launch_util.dart` | `buildNavShape` | `gpx_util.dart` import + direct call | WIRED | Line 176: `final shape = buildNavShape(points);`. Import `package:wanderer/util/gpx_util.dart` at line 18. Inline `(points.length / 499).ceil()` block confirmed absent. |
| `navigation_launch_util.dart` | `objectBoxProvider` | `ref.read(objectBoxProvider)` | WIRED | Line 213: `final store = ref.read(objectBoxProvider);` (success path). Line 219: same in DioException catch. Import `objectbox_store_provider.dart` at line 16. |
| `navigation_launch_util.dart` | `NavigationScreen` | `context.push` with `(NavigateResponse, bool)` record | WIRED | Line 210: `extra: (response, false)`. Line 225: `extra: (cached, true)`. Router at `router_provider.dart` consumes both. |
| `trail_download_service.dart` | `buildNavShape` | `gpx_util.dart` import + direct call | WIRED | Line 87: `final shape = buildNavShape(points);`. Import `package:wanderer/util/gpx_util.dart` at line 13. |
| `trail_download_service.dart` | `POST /valhalla/navigate` | `_api.post('/valhalla/navigate', ...)` inside best-effort try/catch | WIRED | Lines 101-104: `_api.post('/valhalla/navigate', data: {'shape': shape, 'costing': costing}, cancelToken: cancelToken)`. |
| `trail_download_service.dart` | `entity.navCacheJson` | `jsonEncode(response.toJson())` assigned before `box.put` | WIRED | Line 113: `entity.navCacheJson = jsonEncode(response.toJson())`. This assignment occurs before `_store.runInTransaction` at line 121. |
| `router_provider.dart` | `NavigationScreen` | Constructor call with `isOffline` | WIRED | Line 208-209: `NavigationScreen(id: trailId, response: response, isOffline: isOffline)`. Pattern `NavigationScreen(.*isOffline` confirmed. |
| `navigation_screen.dart` | `Icons.wifi_off` | Conditional Icon in `_buildActiveBannerContent` | WIRED | Lines 359-366: `if (widget.isOffline) ...[const SizedBox(width: 8), Icon(Icons.wifi_off, ...)]`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| `navigation_screen.dart` | `widget.isOffline` | Router passes bool unpacked from `(NavigateResponse, bool)` record extra | Yes — set to `true` in DioException branch, `false` on success, both in `navigation_launch_util.dart` | FLOWING |
| `navigation_screen.dart` | `widget.response` | NavigateResponse from API or ObjectBox cache | Yes — either live API response or `NavigateResponse.fromJson(jsonDecode(entity.navCacheJson))` | FLOWING |
| `trail_download_service.dart` | `entity.navCacheJson` | NavigateResponse from Valhalla POST encoded to JSON | Yes — `jsonEncode(response.toJson())`, guarded for non-empty maneuvers; persisted via `runInTransaction` | FLOWING |
| `navigation_launch_util.dart` | `cached` | ObjectBox read via `_readCachedNav` | Yes — queries `store.box<TrailEntity>()` by trail id, decodes `navCacheJson`; null on miss | FLOWING |

### Behavioral Spot-Checks

Step 7b: SKIPPED — no server running. The code is structured for compile-time verification; behavioral checks require a device with GPS and network toggle. See Human Verification items.

Flutter analyze was run at commit time per each SUMMARY (no errors on all modified files). Cannot re-run analyze during verification without a working Flutter SDK environment.

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist in this phase or declared in PLAN files. Step 7c: SKIPPED — no probes declared or conventional.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|------------|------------|-------------|--------|---------|
| OFFLINE-01 | 05-01-PLAN, 05-04-PLAN | Trail download caches Valhalla navigation instructions to ObjectBox | SATISFIED | `trail_download_service.dart` performs best-effort Valhalla POST and stores `navCacheJson`; `buildNavShape` shared helper in `gpx_util.dart` ensures identical shape payload |
| OFFLINE-02 | 05-01-PLAN, 05-03-PLAN | `launchNavigation` falls back to cached instructions on DioException | SATISFIED | `navigation_launch_util.dart`: `on DioException catch (_)` reads cache, validates `maneuvers.isNotEmpty && shape.isNotEmpty`, pushes `extra: (cached, true)` |
| OFFLINE-03 | 05-03-PLAN | After successful online fetch, local cache silently updated | SATISFIED | `navigation_launch_util.dart` line 214: `unawaited(_recacheNav(store, trail.id, response))` after push; `_recacheNav` swallows all errors |
| OFFLINE-04 | 05-02-PLAN | NavigationScreen shows offline indicator icon when operating from cache | SATISFIED | `navigation_screen.dart`: `isOffline` field, `Icons.wifi_off` with `colorScheme.onSurface` tint and size 20 conditionally rendered in `_buildActiveBannerContent` |

All four OFFLINE requirements listed in REQUIREMENTS.md for Phase 5 are accounted for. No orphaned requirements.

**Deviation from plan — OFFLINE-04 icon:** Plan 05-02 specified `FaIcon(FontAwesomeIcons.wifiSlash)`. Executor correctly identified that `wifiSlash` does not exist in `font_awesome_flutter` 11.0.0 and substituted `Icon(Icons.wifi_off)` (Material). This is functionally and visually equivalent — same symbol, same color contract (`colorScheme.onSurface`), same size (20). The deviation is documented in 05-02-SUMMARY.md and does not alter the requirement intent.

### Anti-Patterns Found

No debt markers (TBD, FIXME, XXX) found in any file modified by this phase. No placeholder implementations detected. No empty return stubs found. The `catch (_) {}` bare swallowing catches in `trail_download_service.dart` and `_recacheNav` are intentional best-effort patterns per the phase design (D-06, D-13) and are documented with comments. The `print()` calls in `_downloadPhotos` are pre-existing (not introduced by this phase).

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | — | — | — | — |

### Human Verification Required

#### 1. Offline Navigation from Cache

**Test:** Download a trail while connected to the network. Disable all network (airplane mode). Tap Navigate on that trail from the trail detail screen.
**Expected:** Navigation screen launches showing step-by-step maneuvers (not an error toast). The wifi_off icon is visible at the trailing edge of the active maneuver banner row.
**Why human:** Requires real device with GPS, network toggle capability, a pre-downloaded trail with GPX data, and a Valhalla service that was reachable at download time to populate `navCacheJson`. Cannot simulate `DioException` → ObjectBox read path without live conditions.

#### 2. Online Navigation — No Offline Icon, Silent Re-cache

**Test:** While connected to the network, tap Navigate on any trail with GPX data.
**Expected:** Navigation screen opens normally. The wifi_off icon is NOT visible in the maneuver banner. The cache is refreshed in the background (no spinner, no toast, no user-visible change).
**Why human:** Visual absence of the icon requires a live render. Background re-cache (`unawaited`) cannot be observed without ObjectBox inspection tooling.

#### 3. No-Cache Error Toast Path

**Test:** Navigate to a trail that was NOT downloaded (or was downloaded when Valhalla was unavailable so `navCacheJson` is null). Disable all network. Tap Navigate.
**Expected:** An error toast ("Couldn't start navigation" or equivalent) appears. No crash, no blank screen.
**Why human:** Requires a trail entity with null `navCacheJson` to exercise the `cached == null` fallback-error-toast path inside the `on DioException` branch.

### Gaps Summary

No gaps found. All four roadmap success criteria have verified code implementations with full data-flow traces. All four OFFLINE requirement IDs from REQUIREMENTS.md are satisfied by concrete code. The only pending items are the three human verification tests which require a real device and network conditions.

---

_Verified: 2026-06-14T17:30:00Z_
_Verifier: Claude (gsd-verifier)_
