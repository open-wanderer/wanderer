# Pitfalls Research

**Domain:** Interactive draw/edit-on-map feature (multi-waypoint route builder) on a native-GL map (`package:maplibre` 0.3.5, pre-1.0) with Riverpod async state and Valhalla auto-routing
**Researched:** 2026-07-16
**Confidence:** MEDIUM — grounded in this codebase's existing map/provider patterns (verified by reading source) plus MEDIUM-confidence external verification of `package:maplibre` gesture/coordinate APIs (pub.dev changelog + GitHub discussions; official docs site is thin on gesture details). No HIGH-confidence Context7 source was available for `package:maplibre` internals.

## Critical Pitfalls

### Pitfall 1: Marker-drag and map-pan gesture arenas fight for the same pointer

**What goes wrong:**
The Route Planner needs tap-to-add, drag-to-move, and insert-mid-line — all pointer gestures — layered on top of a map that also wants to pan/zoom/rotate on the same pointer. `package:maplibre`'s `Marker`/`WidgetLayer` markers are plain Flutter widgets rendered in an overlay above the native GL surface (this is exactly how `cluster_layer`'s unclustered markers already work in this codebase — a `GestureDetector` inside `ml.Marker`). A `GestureDetector.onPanUpdate` on a marker widget does not automatically suppress the map's own pan recognizer underneath: both recognizers can enter the same gesture arena, and depending on `HitTestBehavior` and widget hit-test size, either the map pans while a marker is "being dragged" (waypoint doesn't move, map moves under the finger), or a tap-to-add fires on top of an existing marker because the marker's hit-test region is smaller than its visual icon.

**Why it happens:**
Flutter's gesture arena resolves per-pointer between whichever recognizers claim it. A `Marker`'s `GestureDetector` wins tap disambiguation reasonably reliably, but pan/drag recognizers are more prone to both firing (or firing in the wrong order) unless the map's own gestures are explicitly disabled for the duration of the drag. This project has no existing precedent for marker-drag (existing markers are tap-only, in `cluster_layer.dart`/`map_screen.dart`), so there is no "copy this pattern" fallback in-repo.

**How to avoid:**
- Use `WidgetLayer`'s `allowInteraction` flag on the waypoint markers so they explicitly claim gesture priority over the underlying map surface (confirmed available in `package:maplibre`, added pre-0.3.5 per pub.dev changelog).
- On `onPanStart` for a waypoint marker, disable the map's own pan/rotate gestures (the package exposes a `MapOptions`/`MapGestures`-style toggle with independent `pan`/`zoom`/`rotate`/`pitch` booleans — set `pan: false`, `rotate: false` for the drag's duration) and restore them on `onPanEnd`/`onPanCancel`. Do this in a `finally`-equivalent (drag-end AND drag-cancel AND widget dispose) so an interrupted drag (app backgrounded mid-drag) can't leave the map permanently un-pannable.
- Give the marker hit-test region generous padding beyond the visual icon (`HitTestBehavior.opaque` on a padded `GestureDetector`, not the bare icon bounds) so tap-to-add on empty map vs. tap-to-select-marker vs. drag-to-move don't misfire near icon edges.
- Route tap-to-add through the map's own `onMapEvent`/`MapEventClick` handler (as `map_screen.dart` already does for cluster taps) rather than a full-map `GestureDetector`, and only treat a click as "add waypoint" when `featuresAtPoint` at that screen point returns no waypoint-layer hits — this mirrors the existing cluster-vs-unclustered-marker disambiguation already proven in this codebase.

**Warning signs:**
- During manual testing, dragging a waypoint marker also visibly pans the map underneath it.
- Tapping near (but not exactly on) an existing waypoint adds a new waypoint instead of selecting/dragging the existing one.
- Drag gesture "sticks" (map stays un-pannable) after backgrounding the app mid-drag or after a drag that ends off-screen.

**Phase to address:**
Waypoint marker/interaction layer phase (the phase that builds tap-to-add/drag/insert-mid-line) — not the routing phase. This is a map-interaction concern, independent of whether auto-routing is on.

---

### Pitfall 2: Re-fetching the routed polyline on every waypoint move without debouncing or coalescing

**What goes wrong:**
If auto-routing calls Valhalla on every `onPanUpdate` frame (or even every `onPanEnd` without debounce) while a user drags a waypoint, the routing provider fires dozens of HTTP requests per drag gesture. Best case this wastes bandwidth/battery and hammers the Valhalla proxy; worst case, responses arrive out of order (see Pitfall 3) and the UI flickers between stale and current route geometry, or redraws the entire `LineStringSource`/`LineStyleLayer` on every intermediate frame, causing visible jank on native GL (each `updateGeoJsonSource` call is a bridge round-trip to the native renderer).

**Why it happens:**
It's natural to wire "waypoint moved → re-route" directly to the drag callback because that's the simplest code path, and drag deltas fire at 60-120Hz. Nothing in the drag gesture API forces you to throttle.

**How to avoid:**
- Update the **straight-line** (unrouted) segment immediately on every drag frame for responsive visual feedback — this is cheap (local geometry only, no network).
- Debounce the **Valhalla re-routing call** specifically (not the visual update) using the same `Timer`-based debounce pattern already established in `map_cluster_search_provider.dart` in this codebase (400ms window, cancel-and-restart on each new move) — but see Pitfall 3, because that exact provider has a latent ordering bug that must NOT be copied verbatim.
- Only trigger re-routing on drag-end (`onPanEnd`) plus a debounce for the case of rapid successive small moves, rather than on every `onPanUpdate` — the fewer distinct trigger points, the less coalescing logic is needed.
- Batch multi-waypoint edits (e.g., insert-mid-line followed immediately by a drag) into a single re-route request rather than one request per intermediate mutation.

**Warning signs:**
- Network tab / Dio logs show a Valhalla request fired for every pixel of drag movement.
- Visible route-line flicker or "double lines" briefly rendering during a drag.
- Battery/network usage spikes noticeably worse than the existing navigation screen's request cadence.

**Phase to address:**
The auto-routing provider phase, specifically. Debounce belongs in the provider layer (Riverpod notifier), not in the widget's gesture callback — this keeps the debounce testable independent of gesture wiring and matches the existing `map_cluster_search_provider.dart` architecture (debounce lives in the provider, the map screen just calls `searchInBounds`/equivalent).

---

### Pitfall 3: Out-of-order Valhalla responses overwrite a newer route with a stale one

**What goes wrong:**
A user drags waypoint A, releases, then immediately drags waypoint B before A's routing response returns. Two Valhalla requests are now in flight. If A's response (for the now-outdated waypoint set) resolves *after* B's response (for the current waypoint set) — plausible, since request/response latency is not guaranteed to preserve send order — the route line snaps back to the stale A-based geometry and silently discards the correct B-based one. This is worse than a generic loading-race because the user has already moved on to editing waypoint B and has no visual cue that the map just regressed.

**Why it happens:**
This project already has exactly this shape of bug latent in `map_cluster_search_provider.dart`: `searchInBounds()` debounces the *start* of a request via a single cancelable `Timer`, but once the timer fires and `_executeSearch` begins its `await api.post(...)`, there is no guard preventing a second `_executeSearch` from starting (from a subsequent `searchInBounds` call after the debounce window) while the first is still in flight — both write to `state` via `AsyncValue.guard`, and whichever HTTP call resolves last wins, regardless of which was issued last. For trail-cluster search this is a low-stakes UX blip (a cluster re-render); for the route planner, applying the same pattern to routing means a user's active edit can be visibly clobbered by a stale response.

**How to avoid:**
- Add a monotonically increasing request generation counter (`int _generation = 0`) in the routing notifier. Increment it on every new routing request; capture the value locally before the `await`; after the `await` resolves, only apply the result to `state` if the captured generation still matches the current `_generation` — otherwise discard silently.
- Alternatively/additionally, use `dio`'s `CancelToken` to actually cancel the in-flight request when a newer one supersedes it (cheaper than letting a doomed request complete) — the codebase does not yet have any `CancelToken` usage, so this would be a new pattern to introduce deliberately, not copy from elsewhere.
- Do not reuse `map_cluster_search_provider.dart`'s debounce-only pattern for the routing provider without adding the generation/cancellation guard — flag this explicitly in code review since it's a direct existing-code precedent that looks correct but isn't race-safe.

**Warning signs:**
- Route line visibly "jumps backward" to an earlier waypoint configuration after a rapid sequence of edits.
- Integration test that fires two overlapping routing requests with reversed resolution order (mock Dio to resolve the second call first) fails without the guard.

**Phase to address:**
The auto-routing provider phase. This is the single highest-value pitfall to prevent explicitly with a unit/widget test (mock Dio with controllable resolution order), since it's silent in normal manual testing (requires specific timing to reproduce) but real in production once users edit routes quickly.

---

### Pitfall 4: Undo/redo stack captures async-in-flight state, replaying a route that doesn't match server truth

**What goes wrong:**
If undo/redo snapshots the waypoint list *and* the currently-displayed route geometry together, and a snapshot is taken while a routing request is still in flight (or was based on a since-superseded response), undo can restore a route line that was never actually validated against the current waypoint set — e.g., undo brings back waypoints from state N but re-displays the (now stale) route geometry from state N+1's in-flight response. Separately, an unbounded undo stack that stores full route geometry (not just waypoint deltas) per step grows memory usage roughly linearly with edit count on what should be a lightweight planning session.

**Why it happens:**
It's tempting to snapshot "whatever is currently on screen" (waypoints + rendered polyline) as one undo unit for simplicity, rather than treating routed geometry as a *derived* value that should never itself be part of undo history.

**How to avoid:**
- Make the undo/redo stack store only the **source of truth**: the ordered waypoint list (and per-waypoint metadata like manual-vs-routed segment flag). Never snapshot the Valhalla-derived polyline itself — always re-derive it (async, debounced, generation-guarded per Pitfall 3) after an undo/redo restores a waypoint state.
- Cap the undo stack depth (e.g., last 50 operations) and/or coalesce rapid-fire drag deltas into a single undo entry per completed gesture (one undo step per drag-end, not per `onPanUpdate` frame) — otherwise a single drag gesture could push dozens of undo entries.
- On undo/redo, treat it exactly like any other waypoint mutation with respect to routing: bump the request generation counter and kick off a fresh (debounced) re-route rather than trying to "restore" a previously cached route response, since that cached response may itself have been superseded or never resolved.

**Warning signs:**
- Undo restores waypoint positions correctly but the displayed route line doesn't match (still shows the routed path for a different waypoint set until the next edit forces a re-route).
- Undo stack size grows unbounded during a long editing session (memory profiling shows steady growth tied to drag-frame count, not edit count).

**Phase to address:**
Undo/redo phase, but it has a hard dependency on the auto-routing provider's generation-guard (Pitfall 3) already existing — sequence undo/redo after (or alongside, sharing the same generation-counter primitive as) the routing provider, not before it.

---

### Pitfall 5: Fast-changing local drag state colliding with async server state in the same Riverpod notifier

**What goes wrong:**
A single "route state" `AsyncNotifier`/`Notifier` that holds both (a) the waypoint list actively being dragged (needs to update on every `onPanUpdate` frame, purely local/synchronous) and (b) the Valhalla-derived route geometry (async, network-backed) tends to force every local drag-frame update through the same `state = AsyncData(...)` / `AsyncValue.guard` machinery meant for the async part. This causes two related problems: (1) UI listening to `.isLoading`/`.hasValue` on the combined state flickers into a loading state on every keystroke-equivalent drag frame if the notifier's `build()`/mutation path isn't carefully separated, and (2) per this project's own documented gotcha, any listener closure over `AsyncValue` extension getters (`isLoading`, `hasValue`, etc.) that isn't explicitly typed can silently resolve to `dynamic` and skip resolution entirely — a known project-specific footgun already called out in `PROJECT.md`/CLAUDE.md context, and one this screen is especially exposed to because it will have far more `ref.watch`/`ref.listen` call sites reacting to fine-grained interaction state than any other screen in the app.

**Why it happens:**
Riverpod's `AsyncNotifier` is designed around "one async operation → one state," but a route planner naturally has two state cadences (60Hz local drag vs. sub-second network routing) that don't belong in the same async container. Modeling them as one provider is the path of least resistance when scaffolding the screen.

**How to avoid:**
- Split into (at least) two providers: a synchronous `Notifier<List<Waypoint>>` (or similar plain, non-async state holder) for the live waypoint list that drag gestures mutate directly and synchronously, and a separate `AsyncNotifier`/`FutureProvider`-family provider for the Valhalla-derived route geometry that `ref.watch`es the waypoint provider (through a debounce boundary, not directly) and only enters loading/error/data states for the actual network call.
- Never let a per-frame drag update pass through `AsyncValue.guard` or any `state = AsyncLoading()` transition — that machinery is for the routing call only.
- Everywhere this screen reads `.isLoading`, `.hasValue`, `.value`, or similar `AsyncValue` extension getters inside a `ref.listen`/callback closure, explicitly type the closure parameter (per this project's existing documented convention) rather than relying on inference, since this screen will have significantly more such call sites (drag start/end, undo/redo, routing toggle, profile change) than existing map screens.
- Reuse the project's existing convention: `.value` (not `.valueOrNull`) for nullable `AsyncValue` access, consistent with the already-documented project preference.

**Warning signs:**
- UI shows a loading spinner/skeleton flicker during marker drag even when auto-routing is toggled off (a sign local drag state is routed through async machinery unnecessarily).
- A `ref.listen` callback that should react to routing completion silently never fires — check for un-typed closure parameters first.
- Frame drops during drag correlate with Riverpod provider rebuild counts (verifiable via Riverpod's observer/logging) rather than actual widget rebuild need.

**Phase to address:**
Provider architecture phase (whichever phase first defines the Route Planner's Riverpod provider set) — this is a foundational state-shape decision that's expensive to retrofit once the UI, undo/redo, and routing phases are all built against a single combined provider.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|-----------------|
| Fire Valhalla request directly on `onPanUpdate` instead of debouncing in the provider | Faster to prototype auto-routing | Request storm, jank, out-of-order overwrites (Pitfalls 2/3) | Never — even a spike/throwaway build should debounce at drag-end |
| Snapshot rendered polyline into undo stack alongside waypoints | Simpler undo implementation initially | Undo/redo can restore mismatched waypoint/route pairs (Pitfall 4) | Never |
| One combined `AsyncNotifier` for waypoints + route | Fewer providers to wire initially | Async machinery leaks into every local drag frame (Pitfall 5) | Only for a disposable prototype spike, never for the shipped screen |
| Skip `CancelToken`/generation guard on first pass, "add it later if it's a problem" | Faster first routing call working end-to-end | Race condition is timing-dependent and easy to miss in manual testing, ships silently broken | Acceptable only if a generation-guard is added before the phase is marked done — do not defer past the routing provider phase |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| Valhalla (via existing SvelteKit proxy) | Re-fetching per drag frame; not distinguishing "manual segment" vs "routed segment" waypoints in the request payload | Debounce + generation-guard (Pitfall 3); build the request from the full ordered waypoint list each time, tagging profile (foot/bike) explicitly per the existing `costingForCategory` convention in `gpx_util.dart` |
| `package:maplibre` `WidgetLayer`/`Marker` | Assuming drag "just works" like a native SDK draggable marker; the package has no built-in marker-drag primitive as of 0.3.5 | Implement drag manually: `allowInteraction: true` + manual `GestureDetector` + explicit map-gesture-disable during drag (Pitfall 1) |
| `package:maplibre` coordinate conversion (`toLngLat`/`toScreenLocation`) | Calling the (async, per the package's 0.3.x API) screen-to-geographic conversion on every drag frame and awaiting it inline, adding latency/jank to drag tracking | Prefer local math from `onPanUpdate`'s `Offset` delta plus a single conversion at drag-start/drag-end where possible; if per-frame conversion is unavoidable, don't block the frame on the `Future` — apply the previous known conversion and reconcile async |
| `GeoJsonSource` updates for the route line | Calling `removeSource`/`addSource` on every route update (this project's own `cluster_layer.dart` comments explicitly warn this causes "id-collision and flicker risk") | Use `updateGeoJsonSource` on an existing source, exactly as `cluster_layer.dart` already does for cluster data |
| `onStyleLoaded` re-registration | Forgetting the route-line source/layer must be re-added inside `onStyleLoaded` (theme toggle rebuilds the style and drops added layers/sources — already documented as a project-wide `maplibre` quirk) | Register the route `GeoJsonSource`/`LineStyleLayer` inside the same `onStyleLoaded` callback pattern used by `addClusterLayers`, guarded against the `onStyleLoaded`-before-`onMapCreated` ordering quirk already known in this project |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Per-frame `updateGeoJsonSource` calls during drag | Visible stutter/jank on native GL surface, especially on lower-end Android devices | Update the straight-line preview via local geometry only during drag; reserve `updateGeoJsonSource` for debounced routing results | Noticeable even at low waypoint counts (5-10) because each call is a Dart↔native bridge round-trip, not a data-size problem |
| Elevation profile recomputed synchronously on every waypoint change | UI thread jank when dragging near a large waypoint count | Debounce elevation recompute the same way as routing; consider computing it only when the elevation view is actually visible (mutually exclusive with waypoint list sheet per this milestone's spec) | Becomes visible past roughly a dozen waypoints or on routed (Valhalla-returned, much denser) polylines vs. straight-line segments |
| Undo stack storing per-frame drag deltas | Memory growth and slow undo/redo traversal in long editing sessions | Coalesce to one undo entry per completed gesture (drag-end), not per frame (Pitfall 4) | Any session with more than a few drag gestures if uncoalesced |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| No visual distinction between a "routing in flight" segment and a settled one | User can't tell if the displayed route reflects their latest edit or a stale/loading one | Show a lightweight in-progress indicator (e.g., dashed/dimmed line) on segments awaiting a routing response, cleared only when the generation-guarded result lands |
| Silent drop of routing failures (Valhalla unreachable) | Route line appears to just stop updating with no explanation, user assumes the app is broken | Fall back to straight-line segments on routing failure (mirrors this project's existing "online-only, drop and fall back" pattern for navigation) with a visible, dismissible notice, not a silent no-op |
| Undo/redo with no visible affordance for how many steps are available | User over- or under-taps undo, landing on an unexpected waypoint state | Standard bounded undo stack with disabled-state buttons at the stack boundaries, consistent with the capped-depth approach in Pitfall 4 |

## "Looks Done But Isn't" Checklist

- [ ] **Marker drag:** Often missing the map-gesture-disable-during-drag step — verify by dragging a waypoint near the screen edge and confirming the map itself doesn't pan.
- [ ] **Auto-routing debounce:** Often only debounces the request trigger, not response ordering — verify with a test that mocks two overlapping requests resolving out of order and asserts the UI shows the result matching the *last-issued* request, not the last-resolved one.
- [ ] **Undo/redo:** Often stores derived route geometry instead of re-deriving it — verify by undoing a state, confirming the route line updates only after a fresh (debounced) routing call, not from a cached snapshot.
- [ ] **Theme toggle mid-edit:** Often forgets to re-register the route-line source/layer inside `onStyleLoaded` — verify by toggling light/dark theme while a route is drawn and confirming the route line survives the style swap.
- [ ] **Handoff to trail_create_screen:** Often forgets to cancel/clear the routing provider's in-flight generation and debounce timer on navigation away — verify no stray Valhalla response arrives and touches state after the Route Planner screen has been popped (use `ref.onDispose` to cancel, same as `map_cluster_search_provider.dart` already does for its debounce `Timer`).

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|-----------------|------------------|
| Gesture conflict shipped (map pans during marker drag) | LOW | Add `allowInteraction` + gesture-disable-during-drag; this is additive and doesn't require restructuring provider state |
| Race condition shipped (stale response overwrites current route) | MEDIUM | Retrofit a generation counter into the existing routing notifier; requires touching every place `state = ...` is assigned after an `await`, but no data model change |
| Undo/redo stores route geometry directly | MEDIUM-HIGH | Requires reworking undo entries to store only waypoint deltas and re-derive route on restore — touches undo/redo data structure, not just the notifier logic |
| Combined async+sync provider shipped | HIGH | Splitting one provider into two after UI/undo-redo/routing all depend on the combined shape requires touching most of the screen's `ref.watch` call sites |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| Gesture-arena conflict (Pitfall 1) | Waypoint marker/interaction phase | Manual test: drag a waypoint to the screen edge, confirm map doesn't pan; tap near (not on) an existing marker, confirm it doesn't add a duplicate waypoint |
| Redraw thrashing without debounce (Pitfall 2) | Auto-routing provider phase | Log/count Valhalla requests during a single sustained drag gesture; assert at most one request per debounce window |
| Out-of-order response race (Pitfall 3) | Auto-routing provider phase | Unit test with mocked Dio resolving two overlapping requests in reverse order; assert final state matches the later-issued request |
| Undo/redo storing derived state (Pitfall 4) | Undo/redo phase (after routing provider's generation-guard exists) | Undo to a prior waypoint state; assert route geometry is re-fetched (debounced) rather than restored from a cached snapshot |
| Combined sync/async provider (Pitfall 5) | Provider architecture phase (first phase that defines Route Planner providers) | Riverpod provider observer/log: confirm drag-frame updates never transition the routing provider through `AsyncLoading` |

## Sources

- Direct source reading of this codebase: `app/lib/routes/map_screen.dart`, `app/lib/components/map/cluster_layer.dart`, `app/lib/provider/trail/map_cluster_search_provider.dart`, `app/lib/util/polyline_util.dart`, `app/lib/util/gpx_util.dart`, `app/pubspec.yaml` — HIGH confidence, these are the project's own existing patterns and latent issues.
- `.planning/PROJECT.md` — project constraints, decisions, and documented `maplibre` 0.3.5 quirks (`onStyleLoaded`/`onMapCreated` ordering, `file://` sprite issue, `Duration(milliseconds: 1)` camera instant-move) — HIGH confidence, project-authoritative.
- [maplibre changelog | Flutter package (pub.dev)](https://pub.dev/packages/maplibre/changelog) — MEDIUM confidence: confirms `WidgetLayer` `allowInteraction`, early "disable some or all input gestures" option, `onStyleLoaded` ordering fixes.
- [Marker - MapLibre Flutter docs](https://flutter-maplibre.pages.dev/docs/annotations/markers/) — LOW-MEDIUM confidence: confirms `MarkerLayer`/`SymbolLayer` distinction but does not document drag gestures explicitly (verified gap, not asserted as fact).
- [Cannot Make gestureRecognizers Work in Maplibre Flutter · Discussion #683](https://github.com/maplibre/flutter-maplibre-gl/discussions/683) and related GitHub/WebSearch results on `MapOptions`/`MapGestures` (`pan`/`zoom`/`rotate`/`pitch` toggles) and `toLngLat`/`toScreenLocation` async conversion methods — MEDIUM confidence, cross-referenced across multiple search results but not confirmed against exact pinned 0.3.5 API surface (flagged as needing validation during implementation spike).
- General Riverpod `AsyncNotifier`/generation-counter/`CancelToken` race-guard pattern — MEDIUM confidence: standard, widely-documented Riverpod community pattern for "cancel stale async work," not verified against a specific official Riverpod doc page for this project's pinned version, but consistent with how `ref.onDispose`/`AsyncValue.guard` are already used elsewhere in this codebase.

---
*Pitfalls research for: Route Planner screen (Flutter mobile app, native-GL map + Riverpod + Valhalla auto-routing)*
*Researched: 2026-07-16*
