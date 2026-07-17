---
phase: 260717-t7q-add-a-settings-tab-to-the-route-planner-
reviewed: 2026-07-17T00:00:00Z
depth: quick
files_reviewed: 22
files_reviewed_list:
  - app/lib/components/map/route_anchor_layer.dart
  - app/lib/components/route_planner/elevation_tab.dart
  - app/lib/components/route_planner/route_anchor_list_tab.dart
  - app/lib/components/route_planner/route_anchor_sheet.dart
  - app/lib/components/route_planner/settings_tab.dart
  - app/lib/components/route_planner/travel_profile_sheet.dart
  - app/lib/provider/planned_gpx_provider.dart
  - app/lib/provider/planned_gpx_provider.g.dart
  - app/lib/provider/route_anchor_provider.dart
  - app/lib/provider/route_anchor_provider.g.dart
  - app/lib/provider/router_provider.dart
  - app/lib/provider/router_provider.g.dart
  - app/lib/routes/route_planner_screen.dart
  - app/lib/routes/trail_source_select_screen.dart
  - app/lib/util/gpx_util.dart
  - app/lib/util/route_planner_handoff_util.dart
  - app/lib/util/route_travel_bucket.dart
  - app/test/components/route_planner/settings_tab_test.dart
  - app/test/components/route_planner/travel_profile_sheet_test.dart
  - app/test/provider/planned_gpx_provider_test.dart
  - app/test/provider/route_anchor_provider_test.dart
  - app/test/util/gpx_util_test.dart
  - app/test/util/route_travel_bucket_test.dart
findings:
  critical: 2
  warning: 4
  info: 2
  total: 8
status: issues_found
---

# Phase 260717-t7q: Code Review Report

**Reviewed:** 2026-07-17
**Depth:** quick (escalated to targeted deep-read on the 3 areas the requester flagged: provider-refactor state safety, `costing_options` payload construction, and `switchProfile`/`resolveAllSegments` race safety)
**Files Reviewed:** 22
**Status:** issues_found

## Summary

The `costing_options` payload construction in `_resolveSegment` (`route_anchor_provider.dart:141-146`) is correct: it nests the fixed bucket options under the exact `state.travelProfile` key (`'pedestrian'`/`'bicycle'`), matching Valhalla's expected shape and confirmed by the provider tests. The `CancelToken` + generation-counter combo correctly supersedes stale per-segment dispatches for a single `RoutePlannerScreen` session.

However, the single-instance `keepAlive` refactor (Rec B) relies entirely on `resetForSession()` being called exactly once per planner entry, in `RoutePlannerScreen.initState()`. The entry point that reaches that screen (`TrailSourceSelectScreen._openPlanner`) has a re-entrancy hole: its "Design your own route" card is not guarded against its own in-flight loading state, only against the unrelated import flow. A user who taps that card twice during the (up to 4s) geolocation-resolution window can trigger two independent `_openPlanner()` runs, each of which can end up pushing `/route-planner` — and because `routeAnchorsProvider` is a single global instance with no family key, the second instance's `resetForSession()` silently wipes the first instance's in-progress route out from under it. This is the exact "stale/leaked state across repeated Route Planner entries" risk the review was asked to check for, and it is concretely reachable, not just theoretical.

Separately, two `firstWhere` lookups in `route_anchor_provider.dart` (`retrySegment`, `insertAnchorOnSegment`) are not defensive against IDs that no longer exist in `state`, and the native map's segment-tap hit-testing is a real source of such stale IDs given the layer's GeoJSON sync is asynchronous and fire-and-forgotten.

## Critical Issues

### CR-01: "Design your own route" card is not guarded against its own async loading state — reachable double-push of `/route-planner` corrupts the shared keepAlive provider

**File:** `app/lib/routes/trail_source_select_screen.dart:42-62, 136`
**Issue:** `_openPlanner` sets `_plannerLoading = true` only *after* `showTravelProfileSheet` resolves, then awaits `_resolveInitialCenter()` (which can block up to 4 seconds on `foregroundPositionStreamProvider.timeout(Duration(seconds: 4))`) before calling `context.push('/route-planner', ...)`. The card's own `onTap` handler only disables itself for `_importLoading`, never for `_plannerLoading`:

```dart
onTap: _importLoading ? null : () => _openPlanner(l10n),
```

Compare this to the sibling `_importGpx` method, which *does* guard its own re-entrancy at the top (`if (_importLoading) return;`) — `_openPlanner` has no equivalent self-guard. Once the travel-profile sheet is dismissed (closing the modal barrier) and `_resolveInitialCenter()` starts running, the card is fully tappable again; a second tap starts a second `_openPlanner()` call, opens a second `showTravelProfileSheet`, and — if the user makes a second selection before the first center resolution finishes — both branches independently call `context.push('/route-planner', extra: {...})`, pushing two `RoutePlannerScreen` instances onto the navigator stack.

Because `routeAnchorsProvider` (`route_anchor_provider.dart:73-92`) is `@Riverpod(keepAlive: true)` with **no family argument**, both `RoutePlannerScreen` instances share exactly one provider instance. The second instance's `initState()` calls `resetForSession(...)` (`route_planner_screen.dart:102-112`), which unconditionally clears anchors/segments/undo/redo and cancels every in-flight Valhalla request — including the first instance's, even though that screen is still on the navigation stack beneath the second. Popping back to the first screen then shows the second session's (or freshly reset) state, silently discarding whatever the user had already planned in the first session.

**Fix:** Guard the card's own re-entrancy symmetrically with `_importGpx`, e.g.:
```dart
onTap: (_importLoading || _plannerLoading) ? null : () => _openPlanner(l10n),
```
and/or add an early-return guard at the top of `_openPlanner`:
```dart
Future<void> _openPlanner(AppLocalizations l10n) async {
  if (_plannerLoading) return;
  final bucket = await showTravelProfileSheet(context);
  ...
```

---

### CR-02: Unguarded `firstWhere` on stale anchor/segment IDs from asynchronously-synced native map hit-tests can throw uncaught `StateError`

**File:** `app/lib/provider/route_anchor_provider.dart:202-206, 401-411`
**Issue:** `retrySegment` and `insertAnchorOnSegment` both look up anchors/segments via `firstWhere` with no `orElse`:

```dart
Future<void> retrySegment(String beforeAnchorId, String afterAnchorId) {
  final a = state.anchors.firstWhere((x) => x.id == beforeAnchorId);
  final b = state.anchors.firstWhere((x) => x.id == afterAnchorId);
  return _resolveSegment(beforeAnchorId, afterAnchorId, a, b);
}
```
```dart
final targetSegment = state.segments.firstWhere(
  (s) => segmentKey(s.beforeAnchorId, s.afterAnchorId) == key,
);
```

Both are called from `route_planner_screen.dart`'s `onEvent` handler (lines 222-266) using `beforeAnchorId`/`afterAnchorId` read off the native map's hit-tested GeoJSON feature (`_mapController?.featuresAtPoint(...)`). That native layer's source is kept in sync via `_segmentLayer.update(style, next.segments).ignore()` (`route_planner_screen.dart:119-125`) — an async, fire-and-forget call triggered by `ref.listen`. There is a real window between a state mutation (undo, delete, reorder) and the native layer catching up where a tap on the still-stale-rendered segment/marker yields an anchor/segment id pair that no longer exists in `state`. `firstWhere` with no matching element throws a synchronous, uncaught `StateError` directly out of the map's `onEvent` callback (not deferred inside the returned `Future`, since the lookups run before any `await`), which is not wrapped in any try/catch at the call site.

**Fix:** Make both lookups defensive and no-op (rather than crash) on a miss:
```dart
Future<void> retrySegment(String beforeAnchorId, String afterAnchorId) async {
  final a = state.anchors.firstWhereOrNull((x) => x.id == beforeAnchorId);
  final b = state.anchors.firstWhereOrNull((x) => x.id == afterAnchorId);
  if (a == null || b == null) return; // stale hit-test, state has since changed
  return _resolveSegment(beforeAnchorId, afterAnchorId, a, b);
}
```
Apply the same `firstWhereOrNull` + early-return pattern to `insertAnchorOnSegment`'s `targetSegment` lookup.

## Warnings

### WR-01: Re-selecting the already-active travel bucket in Settings still calls `switchProfile`, clearing undo/redo and re-dispatching every segment for no reason

**File:** `app/lib/components/route_planner/settings_tab.dart:82-91`
**Issue:**
```dart
for (final bucket in RouteTravelBucket.values) ...[
  _BucketCard(
    bucket: bucket,
    icon: bucketIcon(bucket, categories),
    selected: bucket == selectedBucket,
    onTap: () =>
        notifier.switchProfile(bucket.costing, bucket.costingOptions),
  ),
```
There is no guard comparing `bucket` to `selectedBucket` before calling `switchProfile`. Per `RouteAnchors.switchProfile`'s own contract (`route_anchor_provider.dart:233-251`), every call is treated as "a FRESH undo baseline" — it unconditionally clears `undoStack`/`redoStack` and re-dispatches a Valhalla resolve for every existing segment via `resolveAllSegments()`. A user who taps the card they're already on (e.g., a mis-tap, or tapping to confirm selection) silently loses their entire undo history and triggers a redundant network round-trip per segment.
**Fix:**
```dart
onTap: () {
  if (bucket == selectedBucket) return;
  notifier.switchProfile(bucket.costing, bucket.costingOptions);
},
```

### WR-02: `_resolveSegment` only catches `DioException` — a shape-decode failure bypasses the `blocked` state and is silently swallowed by `.ignore()`

**File:** `app/lib/provider/route_anchor_provider.dart:130-171`
**Issue:** The success path calls `PolylineUtil.decode(shape, precision: 6)` (line 164) after validating that `shape is String`, but does not validate the *contents* of `shape` are well-formed polyline data. If decoding throws (malformed/truncated shape string from a misbehaving proxy or partial response), the exception is not a `DioException`, so it is not caught by the single `on DioException catch (e)` clause. Most call sites invoke `_resolveSegment(...).ignore()` (`Future.ignore()`), which suppresses the error entirely — the segment is left in whatever state it was in before (not `blocked`, not `routed`), with no user-visible indication that resolution failed, and no retry affordance (blocked-only UI, per `route_planner_screen.dart:129-145`, never triggers for a segment that stays `straight`/`routed`). One call site, `toggleAutoRouting`, awaits `_resolveSegment` directly via `Future.wait` (not `.ignore()`) — there, an uncaught decode error propagates out of `toggleAutoRouting()` to its only caller, `settings_tab.dart:72` (`onChanged: (_) => notifier.toggleAutoRouting()`), which has no error handling, risking an unhandled exception surfacing from the switch's `onChanged` callback.
**Fix:** Widen the catch to cover decode failures and route them through the same `_markBlocked` path:
```dart
} on DioException catch (e) {
  if (e.type == DioExceptionType.cancel) return;
  if (_generation[key] != myGeneration) return;
  _markBlocked(key);
} catch (_) {
  if (_generation[key] != myGeneration) return;
  _markBlocked(key);
}
```

### WR-03: `/route-planner` route silently defaults `travelProfile`/center to placeholder values instead of failing loudly, contradicting the screen's own documented contract

**File:** `app/lib/provider/router_provider.dart:250-268`
**Issue:** `RoutePlannerScreen.initialCenter`'s doc comment (`route_planner_screen.dart:44-48`) explicitly states: *"Required with no fallback default — inventing a hardcoded center here would silently mask a caller bug."* Yet the route builder that constructs this screen does exactly that:
```dart
final profile = extra?['travelProfile'] as String? ?? 'pedestrian';
final costingOptions = extra?['costingOptions'] as Map<String, dynamic>?;
final lat = extra?['lat'] as double?;
final lon = extra?['lon'] as double?;
final center = (lat != null && lon != null)
    ? Geographic(lat: lat, lon: lon)
    : const Geographic(lat: 0, lon: 0);
```
A malformed/incomplete `extra` (e.g. a future regression in `TrailSourceSelectScreen`, or a deep link) silently produces a planner session centered on "null island" (0,0) with a default `pedestrian` profile and no `costingOptions` — exactly the caller-bug-masking scenario the screen's own docstring warns against, just moved one layer up into the router.
**Fix:** Either propagate `null`/an error state when `lat`/`lon` are missing (e.g., redirect back to `/trail/create` instead of rendering with a bogus center), or explicitly document why the router is the intended place for this fallback (if intentional for deep-link resilience) rather than leaving the contradiction implicit.

### WR-04: `switchProfile`/`resolveAllSegments` re-dispatch every segment with no visual "resolving" state while the old polyline/segment state is still shown

**File:** `app/lib/provider/route_anchor_provider.dart:233-284`
**Issue:** `switchProfile` updates `state.travelProfile`/`state.costingOptions` synchronously, then fires `resolveAllSegments()`, which re-dispatches `_resolveSegment` for every segment `.ignore()`d (fire-and-forget). Segment `state` values are left untouched (still `routed`/`straight`/`blocked` from the previous profile) until each individual Valhalla call resolves. There is no intermediate "pending/resolving" `SegmentState`, so the Settings tab can show "Biking / Road" selected while the map still renders polylines that were computed under the previous (e.g. hiking) profile, for however long the batch of Valhalla calls takes — with no loading indicator anywhere in the UI to explain the mismatch. Not a data-corruption bug (final state converges correctly), but a real UX/robustness gap directly touching the race-condition area flagged for review.
**Fix:** Consider adding a `SegmentState.resolving` (or reusing existing loading affordances) set synchronously in `resolveAllSegments`/`switchProfile` before dispatching, so the UI can show all segments as "recomputing" until each settles.

## Info

### IN-01: `RouteAnchorLayer`'s `AnimatedScale` recreates the scale animation identically for every non-selected/non-dragging marker each build

**File:** `app/lib/components/map/route_anchor_layer.dart:103-112`
**Issue:** Minor — not a functional bug, but every marker's `AnimatedScale` is rebuilt from a `for` loop on every `routeAnchorsProvider` emission (e.g. every keystroke of a drag gesture, since `dragAnchor`/`onPanUpdate` both call `setState`). This is a normal Flutter rebuild pattern and out of scope per the "no performance findings" instruction, but worth a note since `onPanUpdate` alone (`setState(() => _dragOffset = ...)`) already rebuilds the entire marker list on every drag frame, not just the dragged marker.
**Fix:** Not required; noted for awareness only (performance is out of v1 review scope).

### IN-02: `bucketIcon` in `settings_tab.dart` and `_bucketIcon` in `travel_profile_sheet.dart` are verbatim duplicates

**File:** `app/lib/components/route_planner/settings_tab.dart:17-35`, `app/lib/components/route_planner/travel_profile_sheet.dart:16-34`
**Issue:** The two functions are identical in logic (category-to-icon resolution for a `RouteTravelBucket`), differing only in the leading `_` (one is public/exported, one private). Both files' own doc comments acknowledge this ("Mirrors `travel_profile_sheet.dart`'s identical resolution"). This is intentional per the plan notes, but is a maintenance risk: any future fix to one must be manually mirrored to the other.
**Fix:** Consider hoisting the shared logic into `route_travel_bucket.dart` (or a small shared util) as a single function imported by both call sites, since `bucketIcon` in `settings_tab.dart` is already public and could serve both.

---

_Reviewed: 2026-07-17_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
