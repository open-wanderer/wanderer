# App Performance & Battery Improvement Plan

**Status:** Audit complete, no code changed. Findings from a 4-way parallel codebase audit
(GPS/sensors, network/timers, map rendering, state management), cross-referenced against the
on-device profiling from 2026-08-05/06 (Pixel 6, profile build, batterystats + per-thread /proc
sampling).

**Measured baseline (already banked):** the `androidTextureMode: false` + `AndroidPlatformViewMode.hc`
fix cut pan CPU 168.9% → 63.9%. Confirmed intact at all 4 `MapOptions(` call sites. Remaining
measured costs: MapLibre platform render thread ~34% during pan, ~1.4 MB/s network while panning,
GPS receiver lifecycle, 1.28 GB peak PSS.

Battery priority order for a hiking app: **radio (network) > GPS/sensors > CPU wakeups > GPU > memory**.
The cell/WiFi radio is the most expensive component after the screen; multi-hour recording sessions
are the core use case, so per-fix costs compound harder than per-frame costs.

---

## Tier 0 — Radio & network (biggest battery lever)

### 0.1 Hillshade raster-DEM tiles are the ~1.4 MB/s ★ ROOT CAUSE FOUND
The mystery 260–300 MB per 8 min of panning is **not** vector tiles. Both style assets hardcode a
third-party raster-DEM source:
- `app/assets/map/wanderer_light.json:12-16` (identical in `wanderer_dark.json`): `https://tiles.mapterhorn.com/tilejson.json`, `tileSize: 512`, `maxzoom: 12`
- Layer at `:477-481` has **no minzoom/maxzoom gate** — renders at every zoom.
- Live-measured: one DEM tile = **384,292 bytes**. ~4 tiles/s during pan ≈ 1.5 MB/s — matches the measured 1.4 MB/s exactly.
- The `tilejson.json` response has **no cache-control/etag/last-modified** → re-fetched every style load; `MainActivity.kt:21` `MapLibre.setConnected(true)` (the offline-proxy workaround) forces always-online revalidation semantics, compounding this.

**Decision (2026-08-08):** a bare layer `minzoom` was already tried by the user and rejected — the
sudden appearance of hillshading is jarring. Approved approach instead:

a. **Zoom-interpolated fade, not a hard gate.** Ramp `hillshade-exaggeration` (or
   `hillshade-shadow-color` alpha) with an `interpolate` expression from 0 at ~z7.5 to full at
   ~z9.5, so hillshade fades in imperceptibly instead of popping.
b. **Source-side `minzoom` at the zoom where the ramp is still ~0** (e.g. 7). Below that, no DEM
   tiles are fetched at all — and since exaggeration is 0 there, nothing visually appears or
   disappears. This kills all low/overview-zoom DEM traffic with zero visual pop.
c. **Raise the ambient cache (0.2)** so mid-zoom browsing (z8–12, where hillshade is visible and
   fetches are unavoidable) stops thrashing — revisited areas serve from cache.
d. *(Optional follow-up, not in scope for the first pass)*: serve a lighter DEM — either self-host
   a downsampled/recompressed terrarium set, or add a `HILLSHADE_URL` to the style-sources API so
   instance admins can supply their own, mirroring the existing `TILE_SERVER_URL` pattern.

**Verification:** repeat the 90 s pan capture at an overview zoom (expect near-zero DEM traffic)
and at z10–11 (expect traffic on first visit, near-zero on revisit after c). Visually confirm the
fade-in is smooth on device.

### 0.2 Ambient tile cache too small for DEM tiles
`app/lib/main.dart:73-97` sets 256 MB. At 384 KB/DEM-tile that holds ~680 tiles — cache thrash is
structural. After 0.1 lands, re-evaluate; if hillshade stays, raise the cache (512 MB–1 GB) so
revisited areas stop refetching.

### 0.3 No disk cache for any image
22 raw `Image.network`/`NetworkImage` sites (trail photos, avatars, collages — e.g.
`app/lib/components/trail/photo_collage.dart:196`, `trail_card.dart:62`, `actor_avatar.dart:149`).
Flutter's ImageCache is memory-only → **every avatar and trail photo re-downloads on every cold
start** and on in-session eviction. Only the signed-in user's own avatar is hand-cached
(`store/avatar_cache.dart`).
**Fix:** adopt `cached_network_image` (or a thin dio-based disk cache) and migrate all 22 sites.

### 0.4 Glyph/sprite warm fires ~1000 requests per session
`app/lib/provider/glyph_sprite_cache_provider.dart:83-158`: 4 fontstacks × 256 ranges + 8 sprites;
404 ranges are deleted and **re-requested every session forever**.
**Fix:** persist a manifest of known-404 ranges (or fetch the font's coverage once) so steady-state
sessions make ~0 requests.

### 0.5 Riverpod default retry = 10 attempts / ~45 s on ~29 keepAlive providers
Only `trail_filter_provider.dart:54-59` overrides it. One offline window (common on trail) = a burst
of 10 radio wakeups per failing provider.
**Fix:** set a global conservative retry (e.g. 2 attempts, exponential) on the root
`ProviderContainer`, keep per-provider overrides where warranted.

### 0.6 `/health` GET on every app resume
`app/lib/main.dart:230-232` → `trail_sync_provider.dart:180` probes unconditionally on every
foreground resume, even with an empty sync queue.
**Fix:** only probe when the sync queue is non-empty (or debounce to once per N minutes).

### 0.7 Loopback offline proxy: no cache headers + full table scan per tile
`app/lib/services/tile_proxy_server.dart`: responses carry no Cache-Control (`:170-172`) so MapLibre
can't ambient-cache offline tiles, and `RegionEntity` `getAll()` runs **per tile request** (`:123`).
CPU/disk drain rather than network.
**Fix:** add `Cache-Control: max-age` to proxy responses; cache the region list in memory and
invalidate on region add/remove.

---

## Tier 1 — Recording/navigation session (multi-hour hot path)

### 1.1 Breadcrumb is O(n²) over a hike ★ biggest per-fix CPU
Three compounding costs, all growing with track length (4 h @ 1 Hz ≈ 14,400 points):
- `provider/navigation_provider.dart:131-138`: `[...state.breadcrumb, Wpt(...)]` — full list copy per fix.
- `routes/navigation_screen.dart:1093-1113`: full-track `jsonEncode` + full `updateGeoJsonSource('breadcrumb')` platform-channel replace **per fix**. The guard `prev?.breadcrumb == next.breadcrumb` (`:1103`) is identity-compare and can never be true with the copy pattern.
- `_persistNow` (`:628-681`) every 10 s: three full O(n) passes (polyline encode, elevations, timestamps) + full ObjectBox blob rewrite → O(n²) total write volume.

**Fix:**
- Split the breadcrumb GeoJSON into a frozen source (chunks of e.g. 500 points, pushed once) + a small active-tail source updated per fix — bounds the per-fix serialization to the tail.
- Persist deltas: append new points to the stored blob (or chunked rows) instead of rewriting.
- Keep an efficient grow structure internally; publish a lightweight change signal (count/revision) instead of a copied list.

### 1.2 NavigationScreen rebuilds the entire 1,968-line screen ≥1 Hz
`routes/navigation_screen.dart:1115-1122` watches `navigationProvider` (per fix) and
`navigationStatsProvider` (1 s `Timer.periodic`, `navigation_stats_provider.dart:145`) with **no
`.select`** — rebuilding the whole Stack including `MapLibreMap`, `TrailMarkerLayer`, scalebar,
attribution, compass, sheet, `ElevationProfile`. Only ~6 scalar fields are actually consumed.
Multipliers riding on each rebuild:
- 4 unconditional JNI option setters, because `MapOptions` lacks `==` and is constructed fresh in `build()` (`maplibre_platform_interface .../map_options.dart:13-33`; `map_state.dart:342-359`).
- `_composeStyle` per build (`:1145`) — offline mode does full style jsonDecode/encode round-trips (`util/region/offline_style_rewriter.dart:76,198,207,280`).
- `shapeAsGeographic` twice per build (`:1177-1179`) — O(n) route-polyline allocation each call.
- `TrailMarkerLayer` build → `gpx.allPoints` — two full trackpoint list allocations (`components/map/trail_layer.dart:377`, `util/gpx/gpx.dart:66-79`).
- `ElevationProfile.didUpdateWidget` deep-compares `Gpx` (no `identical` short-circuit in gpx 2.3.0) — O(trackpoints) per rebuild.

**Fix:** hoist stats/nav consumption into small Consumer widgets with `.select`; cache the composed
style, the `MapOptions` instance, `shapeAsGeographic`, and `allPoints`; add `identical()` guards
before deep Gpx comparisons.

### 1.3 Provider family key deep-hashes the full route per lookup
`NavigateResponse` freezed `==`/`hashCode` use `DeepCollectionEquality` over the entire `shape`
polyline (`models/navigate_response.freezed.dart:518-524`); the nav provider family keys on it.
~8–10 provider lookups per fix (`navigation_screen.dart:387-417,1095-1123`) = ~10 full-polyline
hashes per second.
**Fix:** cache provider instances in fields at initState, and/or key the family on a cheap stable id.

### 1.4 Per-frame camera pushes (two independent sources)
- `_bearingFollowTicker` (`navigation_screen.dart:229,477-511`): one native `animateCamera` (300 ms duration!) **per display frame** (60–120 Hz) in heading-up follow mode.
- `_positionAnimController` (`:176-183`): 200 ms tween → `moveCamera` per frame after each fix — near-continuous with `distanceFilter: 0`.

**Fix:** throttle bearing pushes to the sensor rate and skip when delta < ~0.5°; let one animation
source own the camera at a time.

### 1.5 Heading sensor runs at ~60 Hz, comments claim 15 Hz
`navigation_screen.dart:435-458`: `SensorInterval.uiInterval` = 16.7 ms. Each event: low-pass →
marker publish → rebuild. ~60 callbacks+rebuilds/s.
**Fix:** explicit ~100–200 ms sampling period + suppress publishes under a degree threshold.

### 1.6 Background behavior during recording
Correctly stopped on background: heading sub, bearing ticker, tracelet foreground→background config
swap (`navigation_screen.dart:592-608`). **Not stopped:** per-fix `updateGeoJsonSource` onto an
invisible map, the 1 s stats ticker, the 10 s persist timer, per-fix full-screen rebuild machinery.
Also `inactive` (notification shade, calls) triggers a full tracelet config round-trip.
**Fix:** gate UI-side per-fix work (GeoJSON push, marker tween) on lifecycle; keep only stats
accumulation + persistence; debounce the `inactive` transition (~2 s).

### 1.7 Geolocator receiver runs while backgrounded / on covered screens
`provider/foreground_position_stream_provider.dart` now has proper settings (high accuracy, 10 m
filter, 2 s interval) and refcounting — but **no lifecycle awareness**. `LocationMarkerLayer` on
`map_screen.dart:587` (under ShellRoute, stays mounted beneath pushed screens) holds the GPS
receiver while the user reads a trail detail or backgrounds the app.
**Fix:** pause the receiver on app background (unless a tracelet session is active) and consider
route-visibility gating for the map tab. Also iOS: `AppleSettings` lacks
`pauseLocationUpdatesAutomatically`/`activityType: fitness` — set them.

### 1.8 Misc recording items
- `tracelet` `stopOnTerminate: false` (`services/tracelet_position_source.dart:70-107`): **confirmed intentional (2026-08-08)** — recording must survive app termination. Do NOT flip the flag. Instead add relaunch-time reconciliation: on app start, if the native service is running but no `ActiveNavigationEntity` session is persisted, stop the orphaned service; if a session IS persisted, reattach/restore the recording UI.
- `active_navigation_store.dart:30` `read()` = `getAll().first` — use a query limit 1.
- Stats 1 s ticker keeps firing while frozen/paused (`navigation_stats_provider.dart`) — stop the timer when frozen.

---

## Tier 2 — Map pan CPU (interaction-driven; attacks the remaining ~34% platform + ~64% total)

Root mechanism: MapLibre 0.3.5's `MapLibreInheritedModel.updateShouldNotify` returns `true`
unconditionally → every child touching `MapController/MapCamera.maybeOf` rebuilds **every native
camera frame** during pan.

### 2.1 `TrailMarkerLayer` has no memoization (RouteAnchorLayer's fix was never ported)
`components/map/trail_layer.dart:296-419`: every waypoint marker subtree (gestures, AnimatedScale,
BoxShadow, FaIcon) rebuilt from scratch per camera frame + 2 JNI projections just for the
start/finish nudge. Mounted on trail maps and NavigationScreen.
**Fix:** port the `route_anchor_layer.dart:78-84` widget-cache pattern (cache keyed on waypoint list
identity + selection + brightness).

### 2.2 `WidgetLayer` does N unbatched JNI projections per frame
`maplibre widget_layer.dart:58-59` → per-marker `toScreenLocation` JNI round-trip + arena alloc.
Up to 100 markers on `map_screen`/`profile_trail_map_screen` (`hitsPerPage: 100`).
**Fix (app-side):** cap visible unclustered markers harder / cluster more aggressively at low zoom.
**Fix (upstream):** batch `toScreenLocations` in one JNI call — worth a patch/PR to maplibre-flutter.

### 2.3 `WandererAttribution` per-frame work + leaked recognizers
`components/base/wanderer_attribution.dart:36-45`: `getAttributionsSync()` per camera frame; when
expanded, re-parses HTML and allocates undisposed `TapGestureRecognizer`s per attribution per frame
(`:93-137`). Mounted on 5 maps.
**Fix:** fetch attributions on style load only; parse HTML once; dispose recognizers.

### 2.4 Per-frame JNI in stock overlays
`MapCompass` (full `getCamera()` JNI per frame, 7 mounts), `MapScalebar` (2 JNI calls + painter
alloc per frame, 5 mounts).
**Fix:** wrap or fork with throttling (e.g. rebuild at ≤10 Hz or on-idle); low individual cost but
they ride on every pan frame on every map.

### 2.5 `RouteAnchorLayer` drag path bypasses its own cache
`components/map/route_anchor_layer.dart:76,151-156`: during a drag, all N markers rebuild per frame
(+ `toLngLat` JNI per frame), not just the dragged one.
**Fix:** keep static markers from cache; rebuild only the dragged marker.

### 2.6 No push-under gating for stacked maps
Up to 4 live MapLibre SurfaceViews (map → detail panel → detail map → navigation);
`PlatformViewPopGuard` only covers pop animations, and only on 2 of the hosts. Prior measurement
showed memory fully releases on pop and stacking is **not** the heat source — so this is deliberate
scope-down: covered maps keep render threads but idle cost was measured at ~9%. Do last, verify
with per-thread sampling before investing.

### 2.7 Scroll/scrub-driven full-subtree rebuilds
- `trail_detail_screen.dart:53-69`: appbar-opacity `setState` per scroll frame → `TrailPanel.build` → **`computeTrailMetrics` full-GPX pass per scroll frame** (`components/trail/trail_panel.dart:56-58`).
- Elevation-chart scrub: `trail_detail_map_screen.dart:226-228` / `trail_create_screen.dart:1297-1299` `setState` per touch sample rebuilds the whole map host (+ 4 JNI option setters each, per 1.2).

**Fix:** ValueNotifier for opacity/marker position (pattern already used correctly in
`map_screen.dart:86-87`); memoize `computeTrailMetrics` per GPX identity.

---

## Tier 3 — Main-isolate jank & memory (1.28 GB peak PSS)

### 3.1 All GPX parsing is synchronous on the UI isolate
Zero `compute()`/`Isolate.run` in the codebase. Worst path: opening the Library parses **every
downloaded trail's full GPX XML** in one synchronous pass (`provider/trail/trail_library_provider.dart:41-42`
→ `entities/trail_entity.dart:377` → `parseGpxSafely`), then sorts. Also trail-detail open and
recording-finish (`GpxWriter().asString` + `trailFromGpx` inline at Stop tap,
`navigation_screen.dart:870` → `util/route/planner_handoff.dart:342`).
**Fix:** move parse/serialize to `Isolate.run`; for the library list, store precomputed metrics on
the entity so list rendering doesn't need parsed GPX at all (lazy-parse on detail open).

### 3.2 Double retention: parsed `Gpx` + raw XML per trail
`Trail.expand` carries both the full `Wpt` object graph and the raw `gpxData` string; the library
holds every downloaded trail simultaneously. 50 trails × 10k points ≈ 500k live `Wpt`s + 50 XML
strings. Directly feeds the 1.28 GB peak.
**Fix:** drop `gpxData` from the in-memory model (re-read from store on demand); with 3.1's
metrics-on-entity, don't attach parsed Gpx to list items.

### 3.3 Unbounded session growth in keepAlive providers
- `trail_search_provider.dart:42`: append-only paged results, never trimmed.
- `route_anchor_provider.dart:196-203`: undo/redo stacks duplicate all segment polylines + elevation profiles per entry.
- `mapClusterSearchProvider` / `mapTrailSearchProvider`: per-family raw GeoJSON kept for the session.

**Fix:** cap pages retained, cap undo depth (e.g. 20), autoDispose or explicit trim on screen exit.

### 3.4 Welcome-screen topography painter burns ~29k `sin()` per frame, forever
`components/welcome/topography_background.dart:55-139`: 1-day `repeat()` controller drives a
full-screen CustomPaint (24 contours × ~100 steps × 12 hash-sin calls) on welcome/login/register/
profile_share — and keeps painting **under pushed routes** (maintainState). RepaintBoundary exists;
no visibility/lifecycle gating.
**Fix:** gate on route visibility (`ModalRoute.isCurrent`/`TickerMode`) + app lifecycle; optionally
precompute contour paths and animate only a phase offset.

### 3.5 Small cleanups (batch into one pass)
- `DateFormat` constructed per list-item build (`trail_card.dart:181`, `trail_list_item.dart:125`) — hoist/memoize per locale.
- `file.existsSync()` inside list-item `build()` during scroll (`trail_card.dart:50`, `trail_list_item.dart:40`) — resolve async once.
- Library filter+sort runs per keystroke over full parsed-trail list (`library_screen.dart:35-55`) — debounce + precomputed lowercase fields.
- Bottom-sheet result lists use non-builder `ListView` with accumulated pages (`map_screen.dart:760`, `profile_trail_map_screen.dart:798`, `global_search_screen.dart:216`) — switch to `.builder` + keys.
- Missing keys on `TrailCard` in `.builder` lists (`library_screen.dart:122-131` et al.).
- `toast_provider.dart:35` uncancellable 4 s `Future.delayed` per toast.
- `plannedElevationGpx` returns a fresh `Gpx` each change → Riverpod deep-compares whole tracks (`planned_gpx_provider.dart:60-99`) — return identical instance when unchanged.
- `addPostFrameCallback` in `build()` at `trail_detail_screen.dart:86` re-pushes per frame until route changes — guard once.

---

## Sequencing & measurement protocol

Each tier gets an on-device before/after capture (Pixel 6, profile build, wireless adb — device
genuinely unplugged or batterystats reads zero; subtract ~9% adb sampling overhead; per-thread
`/proc/<pid>/task/*/stat` sampling, NOT the Dart CPU profiler — it was blind to the last big win).

| Phase | Contents | Expected win | Risk |
|---|---|---|---|
| P1 | 0.1 hillshade gate + 0.2 cache size — **DONE, VERIFIED ON DEVICE 2026-08-08.** Source+layer `minzoom: 7`, exaggeration ramp z7.5→z9.5 (full value 0.5 = MapLibre default, so z≥9.5 looks unchanged) in both style assets; ambient cache 256→512 MB in `main.dart`. Measured (Pixel 6, per-UID netstats, 90 s pans; old baseline ≈126 MB/90 s): overview zoom **255 KB** (~500×), fresh mid-zoom z10–11 **18.2 MB** (~7×), revisit same area **4.8 MB** (cache-dominated; residue ≈ sweep-overlap margin). Fade-in confirmed visually smooth. Reserve knob if ever needed: source `maxzoom` 12→11/10. | ~1.4 MB/s → ~0.05 MB/s pan traffic; big radio win | Low (style JSON edit) |
| P2 | 1.1 breadcrumb O(n²) + 1.2 nav rebuild scoping + 1.3 hash keys, **plus 1.4/1.5/1.6 pulled forward** (all navigation_screen work) — **DONE 2026-08-08: analyze+1043 tests green, manual on-device behavior review passed (user, 2026-08-08).** Details: breadcrumb grows in place behind a stable `UnmodifiableListView` + `breadcrumbLength` change signal; GeoJSON split into frozen `breadcrumb` + per-fix `breadcrumb-tail` source (rolls every 250 pts); per-fix handler does zero provider lookups (cached notifiers, `onPosition` returns advance flag, listenManual frozen mirror); build() watches only `select(currentManeuverIndex)` with stats scoped into sheet/FAB Consumers; MapOptions + offline-compose memoized; heading sensor 60→15 Hz with 0.3° publish threshold; bearing ticker skips JNI push when converged (<0.02°/frame); `inactive` lifecycle no-op; backgrounded map gets no GeoJSON pushes (caught up on resume). NOT done here: `_persistNow` still O(n)/10 s (measured trivial — deliberate); TrailMarkerLayer memoization (P4); ElevationProfile got `identical()` guards. | Recording-session CPU flat instead of growing; fewer jank spikes late in hikes | Medium (core recording path — needs field test) |
| P3 | 1.7 geolocator background gating + 1.8 misc — **IMPLEMENTED 2026-08-08, analyze+1043 tests green, awaiting device check.** Details: `ForegroundPositionStream` is now a `WidgetsBindingObserver` — receiver cancelled on paused/hidden/detached, restarted on resume for surviving acquires; `inactive` ignored; iOS `AppleSettings` gains `activityType: fitness` + `pauseLocationUpdatesAutomatically`. Tracelet orphan reconciliation: `stopOrphanedTracking()` called from every resume-drop path in `main.dart` (no row at launch, unresolvable row, user declines) so a declined/dropped session no longer leaves the native service tracking forever. Stats 1 Hz ticker now stops during frozen intervals and paused resumes; `active_nav.read` uses `query().findFirst()`. **Deferred:** route-visibility gating for the map tab's marker (receiver still runs while a detail screen covers /map in foreground — needs RouteObserver plumbing across nested navigators; foreground-only, bounded cost). | GPS duty cycle down outside recording | Medium (lifecycle edge cases) |
| P4 | 2.1–2.5 pan-path memoization — **DONE, VERIFIED 2026-08-08** (measured: screen-on idle 9%→2.9%, locked/backgrounded 0.1%, pan main thread 33.7%→31.1%, no busy loops; pan-total not comparable to the old 90 s protocol — HWUI RenderThread GPU submission dominates and scales with pan vigor; visual review passed). Details: TrailMarkerLayer got the RouteAnchorLayer widget-cache (waypoint subtrees keyed on list identity/selection/primary color; start-finish pins built once; nudge now pure camera math — was 2 JNI projections/frame); RouteAnchorLayer drags rebuild only the dragged marker (cache no longer invalidated at drag start); WandererAttribution caches attributions per style identity, parses HTML once per string, and reuses/disposes tap recognizers (was per-frame parse + leaked recognizers when expanded); new `map_ui_controls.dart` WandererMapCompass/WandererMapScalebar replace the package widgets at all 12 mounts — compass reads Flutter-side MapCamera (was a JNI getCamera/frame), scalebar re-derives meters-per-pixel only past zoom/lat epsilons (was 2 JNI/frame), both return cached subtrees otherwise; unclustered trail markers capped at 60 (was 100; each costs one JNI projection/frame in WidgetLayer). Remaining upstream: WidgetLayer's own unbatched per-marker `toScreenLocation` — a maplibre-flutter patch/PR candidate. | Chip at the remaining ~64% pan CPU (Dart share) | Low-medium |
| P5 | 0.3–0.7 network hygiene — **DONE, VERIFIED ON DEVICE 2026-08-08.** Details: all 22 image sites migrated to `cached_network_image` 3.4.1 (`NetworkImage`→`CachedNetworkImageProvider`, `Image.network`→`Image(image: ...)`; the signed-in user's hand-rolled avatar_cache path kept; `debugAvatarImageProviderFactory` test seam added since the cache manager's real I/O never completes under FakeAsync); glyph warm memoizes 404s to `map_cache/glyphs/.missing-urls.txt` keyed by full URL (transient failures NOT memoized); global `ProviderScope(retry:)` = 2 attempts 400/800 ms (was 10 over ~45 s for every keepAlive provider); `/health` probe on resume now fires only when the sync queue has due candidates; loopback proxy adds `Cache-Control: max-age=86400` to tile responses and memoizes the region table with a 5 s TTL (was a full `getAll()` per tile request). | Cold-start and flaky-network radio use | Low |
| P6 | 3.1–3.3 isolates + memory | Library jank gone; peak PSS well under 1 GB | Medium (async refactor) |
| P7 | 3.4–3.5 cleanups | Minor | Low |

Field-test gate for P2/P3: one real multi-hour recorded hike comparing battery drain %/hour and
verifying track integrity (breadcrumb, stats, persisted recovery) — distance/elevation outputs must
stay bit-identical (see memory: distance is raw, elevation gate is deliberate).

## Decisions (resolved 2026-08-08)

1. **Hillshade (0.1):** hard `minzoom` rejected (jarring pop-in, tried by user). Approved: zoom-interpolated exaggeration fade + source minzoom under the ramp + bigger ambient cache. Lighter self-hosted/instance-configurable DEM is an optional follow-up.
2. **Background recording is a first-class priority.** All lifecycle gating (1.6, 1.7, P3) must strip only UI-side work; tracelet recording, stats accumulation, and persistence stay fully alive in background.
3. **`stopOnTerminate: false` is intentional** — recording survives app termination. Keep the flag; add relaunch reconciliation (see 1.8).
4. **Marker cap (2.2): approved** — Meilisearch caps results at 100 anyway, so tightening the rendered set at low zooms loses nothing.
5. **Measurement workflow:** Claude prepares code and hands off after analyze+test; the user builds and installs (never Claude). Captures then run over wireless adb against the Pixel 6, genuinely unplugged. P2/P3 additionally want a real recorded hike before/after for %/hour drain comparison.
