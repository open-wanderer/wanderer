---
status: awaiting_human_verify
trigger: "The panning/zooming on the route planner's map becomes more and more sluggish/laggy the more route anchors a user creates"
created: 2026-10-27
updated: 2026-10-27
---

## Symptoms

- **Platform:** Flutter app (app/lib), branch feature/app
- **Expected:** Panning and zooming the route planner map stays smooth regardless of how many route anchors exist
- **Actual:** Panning/zooming gets progressively more sluggish/laggy as the user adds route anchors
- **Errors:** None seen
- **Timeline:** Always been this way (not a regression)
- **Reproduction:** Open the route planner in the Flutter app, add anchors; lag is noticeable at roughly 10-20 anchors and grows with more

## Context

- Known (memory): map panning alone already burns ~2 cores on device; stacked maps are not a leak
- User builds and installs; do not run flutter build or adb install. Hand off after analyze + test.
- Prefer verifying on device (ADB) over reasoning from symptoms when device state is needed.

## Current Focus

SESSION 3 (2026-10-27): R1-R4 device readings are IN and interpreted (Evidence "SESSION 3"). (T) is confirmed and narrowed: the cost the user feels is the creation burst, dominated by whole-route GeoJSON uploads (4 per creation, 2 byte-identical), whose time is mostly JNI/native, not Dart encode. FIXED in two commits, tests first: (F1) skip uploads that would draw nothing new, (F2) marker memo keyed on (id, lat, lon, position). Not done, with reasons in the checkpoint: compute(), per-segment JSON cache, list-tab stats select equality, coalescing/throttling.
status_now: awaiting_human_verify. Fixes committed locally on feature/app (1f692f90f, 44800e490), analyze clean, 238 targeted tests green. PERF-AB probe REMOVED from the tree and saved (with an improved classifier and an encode/JNI split) under the scratchpad perf-ab-probe/ directory for the on-device re-measure.
bug_class: Bohrbug-like, performance; transient-after-state-change (T) confirmed for the reported symptom; steady-state per-frame cost fixed earlier by c2187ab29 (R2 settled 1% over budget at N=20)
known_pattern_candidate: none (knowledge base has no matching entry)

reasoning_checkpoint:
  hypothesis: "(T) The lag the user feels is the anchor-creation burst, not steady-state pan. Its dominant removable component is the NUMBER of whole-route GeoJSON uploads: 4 per creation (harness: constant for k = 2..30), 2 of which carry byte-identical GeoJSON because they only attach elevations to unchanged geometry. Each upload costs 2 ms (route at rest) to 27 ms (inside a burst) of synchronous UI-thread time at ~570 points, of which only ~0.7 ms is Dart encode (AOT bench) - the rest is the JNI/native update, which cannot leave the main isolate. A second, smaller removable component is the whole-marker-set rebuild after the reverse-geocode result, which only changes anchor.location (no marker reads it)."
  confirming_evidence:
    - "R1 (20 adds, no pan, profile build, HEAD): burst class n=208, build 1.9/30.2/43.2 ms, 112/208 = 54% over budget; settled class (same session) 2% over budget. Verified against the pasted line."
    - "R2 (pan at rest, N=20): settled n=300, build 1.6/2.2/2.9, total 6.1/8.0/11.5, 4/300 = 1% over budget; the (i) falsification threshold (>=10%) did not trigger. Same 578-point route, same device: uploads cost 1.9-2.4 ms here vs 7.7 / 27.1 ms (578pt) and 4.5 / 25.1 ms (551pt) in R1 -> the cost is a property of the burst, not of route size or accumulated state."
    - "R1 uploads line: the probe ring is capped at 64 entries (planner_perf_probe.dart: `if (_uploads.length > 64) removeAt(0)`), so 'uploads=64' is the cap, not a count. 30 of the last 64 were byte-identical (47%), matching the harness 2-of-4. Newest-first print order => oldest to newest: 549pt 7.6, 551pt 4.5, 551pt 4.5*, 551pt 25.1*, 578pt 27.1, 578pt 7.7*: the redundant uploads alone account for 37.3 of 76.5 ms (49%) of that creation's synchronous upload time."
    - "AOT bench (scratchpad aot_bench/bench.dart, dart compile exe, Intel laptop): Dart-only encode of the GeoJSON = 0.70 ms at 570 points (23 KB), 4.19 ms at 3800 points, 0.027 ms at 30 points. At R1/R2 size Dart encode is <= 0.7 ms of a 1.9-27 ms call, i.e. 30% of the cheapest and 3% of the slowest."
    - "Empty route upload (R3, N=1, 0pt): 0.2 ms -> fixed cost of the JNI call itself."
    - "harness (real screen, 30 creations): uploads 4 per creation of which 2 byte-identical, constant; marker widget rebuilds 2k per creation (two anchors-identity changes: append, then geocode)."
    - "R4 (3 add/undo cycles at N=0, weak): clean settled windows 2-4% over budget after each cycle; the 12-17% settled windows are the ones that began right after an add and contain late network emissions (emission class is void, see below)."
  falsification_test: "After F1: the real-screen harness must show 2 uploads per creation with auto-routing on (1 with it off) and 0 byte-identical ones. On device, repeating R1 with the scratchpad probe: redundant=0 and uploads per creation 2; burst over-budget share must fall clearly below 54% and build p90 below 30.2 ms. If uploads per creation drop to 2 with none redundant and the burst over-budget share stays >= ~40%, upload COUNT was not the driver: the remaining cost is bytes per upload / native contention (next levers: per-segment sources or time-based throttle) or something else in the burst (sheet, geocode rebuild)."
  fix_rationale: "F1 removes the largest measured component directly: 2 of 4 uploads and their Dart encode disappear (tier 1: compare ids/state/polyline identity before encoding; tier 2: string equality after encoding as a safety net), keyed on the style instance so a new style is never skipped. F2 removes one whole marker-set rebuild per creation by keying the memo on what the markers actually render (id, lat, lon, position)."
  rejected_with_reasons:
    - "compute() for the encode: Dart encode is <= 0.7 ms of 1.9-27 ms at the measured size; the JNI call cannot leave the main isolate; the segments would have to be deep-copied into the isolate at a cost comparable to the encode itself."
    - "per-segment JSON cache: saves <= ~0.7 ms per upload at the measured size (4 ms at 3800 points). Not justified by the device numbers; becomes justified if the re-measure with the encode/JNI split shows encode as a large share on a long route."
    - "list-tab stats select equality: cumulativeStatsByAnchorId costs 0.41 ms (JIT) at 20 anchors per emission, rebuilds a few visible rows; < 4% of a frame budget, an order of magnitude below the uploads."
    - "coalescing / throttling uploads: the emissions of one creation arrive at network latencies hundreds of ms apart, so same-frame coalescing merges almost nothing; a time throttle delays visible route feedback and its interval is a guess without a device. Follow-up candidate: fan-out paths (reverseRoute, switchProfile, resolveAllSegments) emit 2 x segments whole-route uploads."
    - "dropping the straight-line upload that Valhalla replaces within ~hundreds of ms: changes visible behaviour, out of scope."
  blind_spots: "(a) The JNI/native share is derived by subtraction from a laptop AOT bench; the Pixel's Dart share may be larger. (b) Why a same-size upload is 10x slower inside a burst is unexplained (candidates: ART allocation GC for the 46 KB UTF-16 String, native converter/tile re-index overlap from the previous upload, 24px hit layer); only fewer/smaller uploads address any of them. (c) R4 was 3 cycles at N=0, so native-side accumulation is untested - but R2's 2 ms at-rest upload after 20 creations argues against it. (d) The user never said which build the earlier 'low numbers at 20+' came from. (e) maplibre_android updateGeoJsonSource leaves the id String, data String and source object unreleased until Dart GC (finalizable handle external size 0, jni dartjni.c:351-357): an upstream issue candidate, not changed here."
  candidate_causes:
    - "[code] 4 whole-route uploads per creation, 2 redundant (fixed by F1)"
    - "[code] whole-marker-set rebuild on a location-only anchor change (fixed by F2)"
    - "[environment] native converter / ART GC contention inside bursts (not addressable from Dart beyond fewer and smaller uploads)"
    - "[data] route length scales upload bytes; Valhalla/geocode latency decides burst overlap with the user's pan"
    - "[measurement] probe 'emission' class void on device: it compared FrameTiming.buildStart (engine Now() at BeginFrame) with SchedulerBinding.currentSystemFrameTimeStamp, which the framework documents as 'a more or less arbitrary value' on most platforms; my earlier note that they are documented equal was wrong"
  and_gate: "YES for the felt symptom: the burst (uploads) AND a pan or tap overlapping it. Pan at rest is cheap (R2 1%), a lone creation is a few frames; stacked uploads from rapid creation or a pan during the burst make frames drop."

reasoning_checkpoint_session2 (kept for the record; superseded by the block above):
  hypothesis: "(T) The metric the user sees rises on every anchor creation because one creation is a burst of ~7 provider emissions on the UI thread, each of which re-encodes and uploads the WHOLE route as GeoJSON (4 per creation, 2 of them byte-identical to the previous one), rebuilds ALL N marker widgets twice (the geocode result only changes anchor.location, which no marker reads), recomputes the whole-route stats for the list on every emission, and rebuilds the screen/list; every one of those costs scales with route size (vertices, N), and the probe's 300-frame ring (never cleared on add, and read non-destructively) kept those frames in the 'pan' reading. Steady-state pan cost does not depend on the number of earlier creations."
  confirming_evidence:
    - "user: 'amount of markers does not seem to make a difference ... low numbers with 20+ markers ... number goes up whenever I create a new marker' (build unknown)"
    - "real-screen harness, 30 creations through the real onEvent path: emissions 7, uploads 4 (2 byte-identical), screen/layer/flow rebuilds 2 each - constant per creation; marker widget rebuilds 2k and upload size 4k KB - linear in route size; per camera frame after each creation: 5 projections, screen 0, layer 1, flow 1, markers 0, rows 0 - identical for k = 8..30"
    - "probe code: ring fed from mount, snapshot() non-destructive, creation never clears it => the only reading taken after 19 creations (all@N=19, 20%) is the only one that could contain creation frames; every other reading came from a window nextVariant() had just cleared"
    - "JIT bench of the Dart part of one upload: 3.9 ms at 20 anchors x 200-point legs, 5.3 ms at 30, linear in vertices; x4 per creation, on the UI thread, against an 11.1 ms budget"
  falsification_test: "On the HEAD build + new probe, variant 'all', N~20: (i) at-rest pan with no taps for 10 s after the route has settled -> 'settled' class over budget should be <= ~5% and total p90 <= ~10 ms; if it is >= 10%, (T) is not the whole story (steady-state cost is still dear at N~20). (ii) After 10 add+undo cycles at constant N, the same at-rest reading must not rise by more than ~3 points; if it does, (A) accumulation exists and is NOT Dart-side. (iii) The 'emission' class must show build p90 well above the 'settled' build p90 and the uploads line must show several ms of synchronous UI time per upload; if emission frames are as cheap as settled frames, the burst is not what the user sees."
  fix_rationale: "None applied. If (i)-(iii) confirm (T), the candidate fixes are ranked by measured saving (harness): skip an upload when only elevations changed (4 -> 2 uploads per creation, test: byte-identical upload count == 0), key RouteAnchorLayer's marker memo on (id, lat, lon) instead of list identity (2 -> 1 whole-marker-set rebuilds per creation), memoise per-segment JSON so an upload is a concat instead of an O(vertices) re-encode, and make the list tab's stats select compare equal when nothing it shows changed. These attack burst SIZE; (A) would need a different, native-side investigation."
  blind_spots: "(a) Nothing in (T) is measured on device: the sync cost of an upload and the over-budget share of emission frames are predictions from a JIT bench. (b) The user's build for 'low numbers at 20+' is unknown. (c) Native-side per-creation accumulation (mbgl GeoJSON worker state, JNI global refs from unreleased JStrings in updateGeoJsonSource) cannot be seen from Dart; only the add+undo cycle reading can reveal it. (d) The classifier assumes FrameTiming.buildStart and SchedulerBinding.currentSystemFrameTimeStamp share a clock (documented as identical); if 'emission n=0' appears with adds>0 the assumption failed. (e) The felt lag may not be what the probe's total-span metric measures (e.g. input latency, thermal throttling)."
  candidate_causes:
    - "[code] creation burst: whole-route GeoJSON re-encode+upload x4, whole-marker-set rebuild x2, whole-route stats x7 per creation (T)"
    - "[environment] native: GeoJSON source re-parse/tile invalidation on every upload (including 2 redundant ones) competing with rendering; Java-heap pressure from unreleased 0.2 MB JStrings (A or T, unmeasurable from Dart)"
    - "[data] route size (vertices) scales every term; Valhalla latency decides how the 7 emissions interleave with the user's pan"
    - "[measurement] probe window contamination made a burst look like steady-state cost (already fixed in the probe)"
  and_gate: "Possible: burst cost AND the user panning during the burst. Pan alone at rest is expected cheap; a state change alone is a few frames. The user-visible lag needs both. This is why the new reading splits emission / burst / settled."

reasoning_checkpoint_c2187ab29 (the session-1 checkpoint behind the committed fix; its claim 'necessary cause, AND-gate' is WITHDRAWN, see Evidence PROBE WINDOW CONFOUND; the fix itself stays on its measured per-frame merits):
  hypothesis: "RouteAnchorLayer (stock ml.WidgetLayer) is the necessary cause: on EVERY camera event it makes N synchronous JNI projections, rebuilds N Positioned+Transform, relayouts N children and repaints N blurred-shadow markers inside the page-level repaint boundary - which also drags the docked sheet's (and appbar/buttons') paint and raster into every camera frame. The native segment layers then amplify the tail of that UI-thread work (shared CPU/thread contention with mbgl), which is why segments and sheet each look like they 'cause' the lag only when markers are present (multiplicative N x load, not additive)."
  confirming_evidence:
    - "no-markers: build p90 2.0 / p99 3.5, over budget 4% == all@N=0 baseline (1.3/1.8, 4%) and ~bare map (1.5/3.4, 2%); all@N=19: 8.6/21.0, 20%. Removing the one widget restores baseline."
    - "all@N=0 (RouteAnchorLayer mounted, EMPTY marker list, still camera-subscribed every frame) is baseline-cheap -> the cost is proportional to marker COUNT, not to the per-frame subscription/rebuild"
    - "raster flat across all six readings (p50 2.3-3.2, p90 4.2-5.2, p99 5.5-8.4; all@N=19 raster p99 7.1 is not above bare map) -> not a raster/compositor cost (C2 eliminated as the driver)"
    - "scratch widget test on the CURRENT code (MaterialApp page, 19 anchors, camera ticks): 19.0 native toScreenLocation calls per frame, and a sibling stand-in for the sheet is REPAINTED 1.0x per camera frame (marker repaint dirties the page-level RepaintBoundary) - direct observation of both N-proportional mechanisms"
    - "additivity check on over-budget%: bare 2, so factor excess g+s=2 (no markers), m+s=7 (no segments), m+g=5 (no sheet) -> additive model predicts m+g+s=7, observed 18 -> strongly superadditive (AND-gate / multiplicative), markers are the factor present in all three high readings"
  falsification_test: "On device, same chip, variant 'all', ~19-20 anchors, comparable pan: if build p90 stays > ~4 ms or over-budget stays > ~8% after the fix, then the N-proportional marker work (JNI count, widget churn, page-level repaint) was NOT the cost driver. Next lever would then be per-thread CPU (planner_thread_cpu.py, mbgl threads) and segment layer cost (C3: 24px hit layer, geometry simplification)."
  fix_rationale: "Make the marker layer's per-frame cost independent of N and invisible to the rest of the page: (1) project with 4 native reference points + exact bilinear fit over the markers' mercator bounding box, self-validated on the 4th corner (<0.5 px) with fallback to per-marker native projection (pitch, antimeridian, degenerate); (2) position with Flow (paint-time translation, no per-frame relayout, no per-frame Positioned/Transform allocation); (3) per-marker RepaintBoundary so marker pictures (blurred shadow, border, text) are recorded once and only re-parented per frame; (4) one RepaintBoundary around the whole layer so a camera frame no longer repaints the page layer (sheet, appbar, buttons); (5) cull markers outside the viewport. This targets the necessary factor at every level where the evidence shows N-proportional per-frame work, and it breaks the AND-gate at its marker end (the cheapest end to fix, and the only one with a code-level, device-free regression test)."
  blind_spots: "(a) Why segments amplify is NOT proven; native CPU contention / lock stalls is a hypothesis only. (b) One reading per variant, pan vigor uncontrolled, and the N=0 and N=19 'all' snapshots shared one rolling window (snapshot() does not clear) - run-to-run noise unknown; the fix check should repeat 'all'. (c) Per-call JNI stall cost is unmeasured (estimated 20-50 us, can't explain 7 ms alone; tail is the issue). (d) Affine fit relies on runtime validation for pitch != 0 and for iOS parity (untestable here). (e) Per-marker RepaintBoundary effect on Android hybrid-composition overlay surfaces is not measurable off-device. (f) Dart-side segment data is identical in all variants, so GC/heap from polylines does not explain the no-segments gap."
  candidate_causes:
    - "[code] N unbatched JNI projections + N widget/layout/paint per camera frame in WidgetLayer (necessary factor)"
    - "[code] marker repaint dirties the page-level repaint boundary, re-recording/re-rasterising the docked sheet every camera frame (sheet amplifier, verified in widget test)"
    - "[environment] native CPU/thread contention from mbgl rendering 5 segment line layers (incl. 24px hit layer) amplifies the tail of the above (segments amplifier, UNVERIFIED)"
    - "[data] N anchors and total polyline vertex count scale all of the above"
  and_gate: "YES - markers are necessary (removal restores baseline); segments and sheet are amplifiers (each removal cuts over-budget by more than half but does not restore baseline); observed excess 18 pts vs 7 pts additive. Fix breaks the gate at the marker end; segments left untouched until a post-fix reading says otherwise."
next_action: AWAITING USER ON-DEVICE VERIFICATION (see Resolution, SESSION 3, on-device). Build a profile build of HEAD (44800e490), feel the creation burst (tap ~20 anchors quickly, pan during/after), and optionally re-apply the scratchpad probe (README.txt there) for R1/R2 numbers. If the burst is still long with 2 real uploads per creation and none identical, the next levers are bytes per upload (per-segment sources / encode split from the probe's "(enc ..)") and a time throttle; if the burst is gone, archive.

decision_table (session 1, kept for the record; its row-1 verdict rested on the contaminated all@N=19 reading):
  - no markers + BUILD up -> C1  <== HIT
  - no markers + RASTER up -> C2   (raster flat: not hit)
  - no segments -> C3              (partial only: 20% -> 9%, not baseline: not hit)
  - no sheet -> C4                 (partial only: 20% -> 7%, not baseline: not hit)
  - only bare map recovers -> AND-gate fallback (not hit: no-markers alone recovers)
  - bare map still laggy -> reopen (not hit: bare 2% vs N=0 4%)

PERF-AB probe cleanup list (DONE in session 3, after the fix commits):
  - deleted app/lib/components/route_planner/planner_perf_probe.dart, app/test/components/route_planner/planner_perf_probe_test.dart, app/test/tmp_planner_creation_audit_test.dart (assertions promoted to app/test/routes/route_planner_screen_creation_test.dart), app/test/tmp_burst_cost_bench_test.dart
  - reverted the PERF-AB hooks in app/lib/routes/route_planner_screen.dart (git checkout; the file is back at HEAD)
  - nothing PERF-AB was ever staged or committed
  - restorable copy: /private/tmp/claude-501/-Users-christianbeutel-Documents-svelte-wanderer/7e97f07d-aed4-4338-bea8-450907c73db2/scratchpad/perf-ab-probe/ (README.txt, route_planner_screen.perf-ab.patch, app/lib/components/route_planner/planner_perf_probe.dart, app/test/components/route_planner/planner_perf_probe_test.dart, app/test/tmp_planner_creation_audit_test.dart, app/test/tmp_burst_cost_bench_test.dart); `git apply --check` of the patch passes at HEAD 44800e490

## Evidence

- timestamp: 2026-10-27
  checked: maplibre 0.3.5 widget_layer.dart + maplibre_android map_state.dart
  found: WidgetLayer.build runs on every camera event (MapLibreInheritedModel always notifies); it calls controller.toScreenLocations(points), which on Android is `lngLats.map(toScreenLocation)` = one LatLng construct + Projection.toScreenLocation + PointF field reads + 2 global-ref releases per marker, per frame. A batched native Projection.toScreenLocations(double[],double[]) exists in jni.g.dart but is unused. It also rebuilds a Stack of N Positioned+Transform each frame.
  implication: per-frame Dart/JNI cost is linear in marker count; matches "gets worse with more anchors"

- timestamp: 2026-10-27
  checked: route_anchor_layer.dart
  found: Marker widgets are memoized (identical list reused while anchors unchanged) so element subtrees are skipped, but the WidgetLayer per-frame projection/Stack work is unavoidable and uncapped (no viewport culling, no cap). Each marker child = MapMarkerGestures(RawGestureDetector + 2 recognizer factories) > AnimatedScale > Container w/ blurred BoxShadow + Border + Text.
  implication: memoization already removed the subtree rebuild; what remains per marker per frame is projection + Positioned layout + paint of blurred shadow

- timestamp: 2026-10-27
  checked: .planning/PERFORMANCE-PLAN.md (item 2.2, P4) + memory project-app-thermal-profile-findings
  found: Prior profiling already identified "WidgetLayer's own unbatched per-marker toScreenLocation" as a remaining upstream cost and capped trail markers at 60 for that reason. Planner anchors were never capped/culled. Dart CPU profiler is blind to native side; per-thread /proc sampling was the instrument that worked.
  implication: same mechanism, known and unaddressed for the planner

- timestamp: 2026-10-27
  checked: adb devices / adb mdns services
  found: no device attached, no wireless device advertised
  implication: cannot capture on-device per-thread CPU in this run; any fix must be handed off for device verification

- timestamp: 2026-10-27
  checked: route_anchor_provider.dart state mutation sites, route_planner_screen.dart build/listen, route_anchor_sheet.dart, route_anchor_list_tab.dart
  found: provider state only mutates on user actions or network completion (segment route, elevation, reverse-geocode); screen/sheet/list do NOT subscribe to camera and have no AnimationController/Skeletonizer/shimmer. The only camera-subscribed widgets in the planner are RouteAnchorLayer(WidgetLayer) and WandererMapCompass (already cached). Marker cache in RouteAnchorLayer is valid during pan (anchors list identity stable).
  implication: sheet/list/provider are not per-frame costs; remaining per-frame suspects are markers (C1/C2) and native segment layers (C3)

- timestamp: 2026-10-27
  checked: back-of-envelope magnitude of C1
  found: ~5-10us per toScreenLocation (LatLng ctor + call + PointF field reads + 2 global-ref releases) => 20 anchors ~0.1-0.2 ms/frame on the UI thread. Prior P4 work removed per-frame Dart work and moved pan main-thread only 33.7%->31.1%; plan notes HWUI RenderThread / GPU submission dominates.
  implication: C1 alone is unlikely to explain visible lag at 10-20 anchors; C2 (raster/compositor, merged with platform thread under hc) and C3 (native geometry) are at least as likely. A "WidgetLayer batching" fix would be a guess.

- timestamp: 2026-10-27
  checked: Flutter docs on Android Hybrid Composition (web)
  found: Flutter content drawn above a platform view goes to an extra overlay FlutterImageView; hc merges raster and platform threads. maplibre iOS/Android both publish MapCamera per camera event.
  implication: widget markers over the map are a real per-frame raster/overlay cost on the platform thread that also services MapLibre gestures; supports C2 as a first-class candidate

- timestamp: 2026-10-27
  checked: PERF-AB probe implementation
  found: flutter analyze clean on route_planner_screen.dart + planner_perf_probe.dart; test/components/route_planner/planner_perf_probe_test.dart passes (variant gating is single-factor, cycling resets window, percentiles correct for injected FrameTimings, empty-window path safe); scratchpad planner_thread_cpu.py parser unit-checked on /proc stat lines incl. comm with spaces/parens. Existing test/components/route_planner + test/components/map still green (16 tests). Nothing committed.
  implication: instrument is ready; the only missing input is an on-device run

- timestamp: 2026-10-27
  checked: USER MEASUREMENTS (profile build, 90 Hz so budget 11.1 ms, ~20 anchors, N=19)
  found: |
    variant      frames  build p50/p90/p99   raster p50/p90/p99  total p50/p90/p99   over budget
    all N=0      300     1.0 / 1.3 / 1.8     2.9 / 5.2 / 8.4     6.7 /  9.5 / 14.1   4%
    all N=19     300     2.3 / 8.6 / 21.0    3.2 / 4.6 / 7.1     7.8 / 14.3 / 29.6   20%
    no markers   300     1.3 / 2.0 / 3.5     2.7 / 4.7 / 6.8     6.2 /  8.7 / 14.4   4%
    no segments  300     2.0 / 3.5 / 7.5     3.0 / 4.7 / 8.4     7.1 / 10.4 / 20.3   9%
    no sheet     205     1.7 / 2.8 / 7.0     2.8 / 4.6 / 6.6     6.4 / 10.2 / 19.5   7%
    bare map     176     1.0 / 1.5 / 3.4     2.3 / 4.2 / 5.5     5.6 /  7.9 / 13.7   2%
  implication: decision-table row 1 hit (see Current Focus). Raster is flat; BUILD (UI thread) carries the whole regression and mostly in the tail (p50 only +1.3 ms, p90 +7.3, p99 +19). Markers are necessary; segments and sheet are amplifiers (superadditive: additive model predicts 7% over budget, observed 20%).

- timestamp: 2026-10-27
  checked: orchestrator observations vs code and numbers
  found: |
    (1) "Raster flat" - VERIFIED numerically (see table). Caveat: FrameTiming raster covers the raster thread only; Android HWUI RenderThread / FlutterImageView overlay drawing on the platform thread are invisible to it.
    (2) "Regression is in BUILD" - VERIFIED (p90 8.6 vs 1.3, p99 21.0 vs 1.8).
    (3) "segments / sheet removal also cuts build p90 sharply" - VERIFIED numerically (3.5 / 2.8), but each is a single run with uncontrolled pan vigor, and the N=0 and N=19 'all' snapshots share one rolling window because snapshot() does not clear it. Treat the size of the gap as indicative, the direction (superadditive) as likely.
    (4) "frames=205/176 vs 300" - NOT a signal about frame rate: nextVariant() clears the window and the window caps at 300, so the count is just how many Flutter frames the pan produced since the last switch; no-markers (no per-frame marker work at all) shows 300 while no-sheet (markers present) shows 205. p99 spikes (21 ms vs p50 2.3) are sporadic stalls, not a steady per-frame cost: p50 only moved +1.3 ms, which matches the steady Dart cost of 19 markers.
    (5) "variants are single-factor" - VERIFIED in route_planner_screen.dart: 'no sheet' only drops RouteAnchorSheet (nothing sets MapLibre content insets/padding from the sheet; the 164 px sheet inset is used only as a one-off fitBounds padding), 'no segments' only removes the native source + 5 layers (provider/segment data stay in Dart, so heap/GC from polylines is identical in every variant), 'no markers' only drops RouteAnchorLayer (compass is the only other camera-subscribed widget). Two caveats: 'all N=0' is NOT 'all' (no sheet - it mounts at >=1 anchor - no segments, empty marker list), so N=0 vs N=19 conflates N with sheet+segments mounting; and collection-if in MapLibreMap.children shifts child indices at a variant switch (one-off remount of the compass, not a pan cost).
  implication: the numbers support 'markers necessary, segments+sheet amplify'; they do not by themselves identify HOW segments amplify (native-side only, since Dart-side data is identical across variants).

- timestamp: 2026-10-27
  checked: page-level repaint coupling, widget test on CURRENT code (scratch app/test/tmp_old_layer_probe_test.dart: MaterialApp page, RouteAnchorLayer with 19 seeded anchors inside a MapLibreInheritedModel, fake MapController counting toScreenLocation, a CustomPaint counting paints as a stand-in for the docked sheet with no RepaintBoundary of its own, 10 camera ticks)
  found: "OLD: frames=10 projections/frame=19.0 siblingPaints/frame=1.0"
  implication: direct observation of both N-proportional mechanisms. (a) N native projections per frame. (b) WidgetLayer's Stack repaint propagates to the nearest repaint boundary, which is the route's page-level RepaintBoundary, so every camera frame re-records (and re-rasterises) everything else on the page, including the docked sheet with its blurred 10 px top shadow. This is the code-level reason the sheet only costs anything while markers are mounted (AND-gate sheet end). The segments end is native and is not explained by any Dart-side finding.

- timestamp: 2026-10-27
  checked: device state (adb devices) - a device (19131FDF6001Y8) is now attached, app com.openwanderer.wanderer installed, MainActivity focused app with NotificationShade focus (user is holding it)
  found: no pan in progress, so per-thread CPU sampling would only record idle; did not inject input into the user's device
  implication: segments-amplifier mechanism stays an open hypothesis; planner_thread_cpu.py is the ready follow-up if the post-fix reading still shows a tail

- timestamp: 2026-10-27
  checked: Flutter 3.44.2 sources while implementing the fix (flow.dart, object.dart)
  found: RenderFlow.isRepaintBoundary is true, and the Flow widget wraps every child in a RepaintBoundary (keyed ValueKey<Key>(child.key) or by index) unless built with Flow.unwrapped. First draft wrapped twice; mutation checks showed the 'outer boundary' and 'per-marker boundary' tests could not fail for that reason. Now: Flow.unwrapped + one cached RepaintBoundary per marker (no per-frame wrapper allocation), no outer boundary.
  implication: the page-level isolation comes for free from Flow; the regression tests guard it by behaviour (sheet stand-in repaint count, marker repaint count), not by structure.

- timestamp: 2026-10-27
  checked: fix verification off-device (flutter analyze whole app; flutter test test/components/map test/components/route_planner test/util/map test/util/route test/provider = 377 passed)
  found: analyze shows only pre-existing infos (vendor/tiptap_flutter, objectbox/actor/category entities, navigation_stats_provider, local_photo_store, util/local/id.dart) - none in touched files. New tests: batched_projection_test (11), screen_marker_layer_test (11), route_anchor_layer_test (4).
  mutation checks (each deliberately breaks one property, the named test fails, restored afterwards): centre probe removed -> 'tall, narrow set under tilt' fails with 9.2 px error; Stack+Positioned instead of Flow (the old design) -> 'camera frames do not repaint the rest of the page' fails; per-marker boundaries removed -> 'markers are neither rebuilt nor repainted when only moved' fails; never batch -> 'native projections stay constant' fails; culling removed -> 'not painted until they enter' fails; wrapper keys removed -> 'per-marker state follows the key' fails. Same RouteAnchorLayer integration test against the OLD ml.WidgetLayer: projections/frame 19.0 vs 5 (fails), tap / drag / numbering tests pass unchanged (behaviour parity).
  implication: the device-free properties of the fix are pinned. NOT verified: the actual frame times on device, iOS projection parity, and whether the native segment amplifier needs its own fix.

- timestamp: 2026-10-27 (session 2)
  checked: USER EVIDENCE 2 (verbatim, human-verify checkpoint response)
  found: |
    "New result: the amount of markers does not seem to make a difference. I can get low over budget numbers with 20+ markers. The number goes up whenever I create a new marker"
    Not stated: which build (probe build before c2187ab29, or a build containing it). No new [planner-perf] lines. OPEN QUESTION, not assumed either way.
  implication: |
    (1) Per-frame cost proportional to marker COUNT (C1/C2-by-count) is contradicted: low numbers are reachable at N>=20. This holds on either build: on the fixed build it is what c2187ab29 should produce; on the old build it contradicts the all@N=19 20% reading (see PROBE WINDOW CONFOUND for why that reading is suspect).
    (2) The metric moves on each anchor CREATION event, independent of N. Two readings: transient burst recorded by the probe window (T) or real per-add accumulation (A).

- timestamp: 2026-10-27 (session 2)
  checked: planner_perf_probe.dart (window semantics) against the measurement protocol in Evidence "USER MEASUREMENTS"
  found: |
    PROBE WINDOW CONFOUND. PlannerPerfProbe.start() is registered in initState and collects every FrameTiming from screen mount into a 300-frame ring. Only nextVariant() clears it; snapshot() does not and creating an anchor does not. Consequences:
    - Every variant after the first was read from a window that nextVariant() had just cleared, i.e. pure pan frames.
    - The first variant 'all' was read twice from ONE ring: at N=0 and again at N=19, and the N=19 reading was taken after 19 creations. Unless the user panned for >=300 frames (3.3 s of continuous frames at 90 Hz) after the last add, that window still contained creation-burst frames.
    So the keystone number (all@N=19: build p90 8.6 / p99 21.0, over budget 20%) is the only reading that may include creation bursts, and it is the one the whole 'markers are necessary, superadditive' argument rests on. The clean-window readings with markers present (no segments 9%, no sheet 7%) vs no markers 4% / bare 2% still show a markers effect on the OLD layer (+3..5 points), but single runs with uncontrolled pan vigor, and the AND-gate/superadditivity claim (additive 7% vs observed 18..20%) is now unsupported.
  implication: |
    The earlier conclusion 'markers necessary, segments and sheet amplify' is weakened to 'markers cost something on the old layer in clean windows'. c2187ab29 is NOT reverted: its mechanisms were observed directly in a scratch test on the old code (19 native projections per camera frame, sheet stand-in repainted 1.0x per camera frame) and are now 5 and 0; those are real per-frame savings that do not depend on the contaminated reading. What is withdrawn is the claim that it is THE cause of the reported lag.

- timestamp: 2026-10-27 (session 2)
  checked: static audit of one anchor creation (route_planner_screen.dart onEvent -> RouteAnchors.appendAnchor -> state emissions)
  found: |
    One tap = featuresAtPoint (sync native query on the UI thread) then appendAnchor, which emits state: _pushUndo (1), anchors (2), segments (3); then network completions: reverse geocode writes anchors (4), straight-segment height writes segments (5), Valhalla route writes segments (6), routed height writes segments (7). Seven emissions; each one rebuilds RoutePlannerScreen (ref.watch of the whole state), RouteAnchorLayer (ref.watch) and the sheet list tab (select on cumulativeStatsByAnchorId, a getter returning a fresh Map every time, so it never compares equal).
    ref.listen in build: segments identity changed on emissions 3,5,6,7 => RouteSegmentLayer.update => buildSegmentsGeoJson(ALL segments, ALL polyline vertices) + JNI upload, 4 times per add. Emissions 5 and 7 only attach elevation arrays; the GeoJSON content (ids, state, polyline) is byte-identical to the previous upload, so 2 of the 4 uploads are pure waste and force mbgl to invalidate and re-parse the whole GeoJSON source.
    Emissions 2 and 4 change the anchors list identity; RouteAnchorLayer memoises on identical(anchors), so ALL N marker widgets are rebuilt twice per add; emission 4 only changes anchor.location, which no marker reads.
    No accumulating structure found in: route_anchor_provider (undo stack capped at 20; _inFlight/_generation maps keyed per segment/anchor, bounded by route size), RouteSegmentLayer (one source + 5 layers, _added flag), RouteAnchorSheet (one TabController, no ticker beyond it), MapMarkerGestures (global pointer route installed once; recognizer listener removed in didStopTrackingLastPointer/dispose), ScreenMarkerFlow (derived lists replaced, not appended), maplibre_android map_state (listeners added once, removed in dispose).
    Riverpod ref.listen inside build: flutter_riverpod 3.3.1 consumer.dart closes the previous build's listeners at line ~440 before re-running build => does NOT accumulate (source-verified; to be confirmed empirically in the harness).
  implication: |
    Static audit supports (T) and finds no (A). The burst scales with total route size (whole-route re-encode x4, N-marker rebuild x2, N-row list rebuild x7), which would also explain 'gets worse the more anchors there are' without any steady-state per-frame cost.

- timestamp: 2026-10-27 (session 2)
  checked: REAL-SCREEN HARNESS (scratch app/test/tmp_planner_creation_audit_test.dart): the real RoutePlannerScreen, real RouteAnchors/RouteSegmentLayer/RouteAnchorLayer/ScreenMarkerLayer/sheet, a fake MapLibrePlatform (MapLibreMapState subclass that fires onMapCreated/onStyleLoaded/MapEventClick/MapEventMoveCamera and counts toScreenLocation) and a fake Api (geocode, Valhalla route with 200-point legs, height). 30 consecutive creations through the real onEvent click path; counters per creation and per camera frame (debugOnRebuildDirtyWidget, fake StyleController call counts, container listener).
  found: |
    per creation, k = 2..30 (k=1 has no segment): provider emissions 7 (constant); GeoJSON uploads 4 (constant) of which 2 BYTE-IDENTICAL to the previous upload (elevation results) (constant); RoutePlannerScreen rebuilds 2, RouteAnchorLayer 2, ScreenMarkerFlow 2, list tab 2 (constant); MapMarkerGestures (marker subtree) rebuilds = 2 x k (LINEAR in route size: every geocode result re-creates all marker widgets); upload size 4 KB x k (LINEAR: whole route re-encoded each time; 119 KB at k=30 with 200-point legs).
    per camera frame after each creation: projections = k for k<8 (below minBatchSize), 5 for k>=8, constant; RoutePlannerScreen 0, RouteAnchorLayer 1, ScreenMarkerFlow 1, MapMarkerGestures 0, ListTile 0, list tab 0 - IDENTICAL for k=8..30.
    transientCallbackCount after 10 s idle = 1 = Riverpod UncontrolledProviderScope's own one-shot frame callback (stack points at provider_scope.dart:364), not a Ticker/animation.
  implication: |
    NO accumulation on the Dart side: emissions, uploads, rebuilds per creation and per camera frame do not grow with the number of earlier creations (it also confirms empirically that ref.listen-in-build does not stack listeners: if it did, uploads per creation would exceed 4). Per-camera-frame cost is flat in k. What does grow with k is the SIZE of each creation burst (marker rebuilds 2k, upload bytes 4k KB, list/stat computation) - consistent with (T) and with the user's 'number goes up whenever I create a marker'.

- timestamp: 2026-10-27 (session 2)
  checked: scratch app/test/tmp_burst_cost_bench_test.dart - wall time of the pure-Dart parts of one burst, JIT + debug asserts on a laptop (ratios and growth are meaningful, absolutes are not; a profile build on the Pixel is the real unit)
  found: |
    anchors | buildSegmentsGeoJson ms (KB) | cumulativeStatsByAnchorId ms
        5   | 0.92 (22)                    | 0.10
       10   | 1.85 (48)                    | 0.19
       20   | 3.85 (101)                   | 0.41
       30   | 5.31 (154)                   | 0.63
    linear in total polyline vertices. x4 uploads per creation; the list tab recomputes cumulativeStatsByAnchorId on every emission (its select() getter returns a fresh Map, never equal).
  implication: one creation spends ~4 x (3..8 ms) in whole-route JSON encode on the UI thread at N=20 on top of N-marker rebuilds, i.e. on the order of the 11.1 ms frame budget per emission frame. Plausible as the source of over-budget frames after each creation; NOT measured on device.

- timestamp: 2026-10-27 (session 2)
  checked: maplibre_android 0.3.5 JNI path used per creation (style_controller.dart updateGeoJsonSource, map_state.dart featuresAtPoint/_nativeQueryToRenderedFeatures)
  found: updateGeoJsonSource does `source?.geoJson$3 = data.toJString()` with no releasedBy(arena): one Java String global ref (UTF-16 copy of the whole GeoJSON, ~0.2 MB at N=20) per upload stays alive until the Dart wrapper's finalizer runs. featuresAtPoint allocates a small unreleased JArray of layer-id JStrings and runs Gson over each hit feature's properties. All bounded by Dart GC finalization, not by the number of creations.
  implication: a possible Java-heap pressure contributor (4 uploads per creation) but it is memory, not per-frame pan cost; unproven as accumulation; noted only.

- timestamp: 2026-10-27 (session 2)
  checked: PROBE TRUSTWORTHINESS (is the instrument itself accumulating or distorting?)
  found: |
    No accumulation in the probe: addTimingsCallback is registered once in initState (start()), removed once in dispose; rings capped at 300; no per-variant or per-add registration; PlannerPerfChips rebuilds on variant/snapshot changes and on screen rebuilds (emissions), never per camera frame, and does not depend on the camera.
    Distortion found and fixed: (a) the window was fed continuously from mount and creations never cleared it; (b) snapshot() did not clear, so consecutive readings overlapped. Both made a reading mean 'the last 300 frames, whatever they were'.
    Rebuilt (uncommitted): STATS now reads AND clears; every frame is classified emission (the frame that built a provider emission, identified by its engine timestamp via SchedulerBinding.currentSystemFrameTimeStamp in a post-frame callback; documented equal to FrameTiming buildStart), burst (starts within 2.5 s after an emission frame) or settled (the rest); the readout prints n, build/raster/total p50/p90/p99 and 'over budget k/n' per class, N, adds, emissions, and the last six GeoJSON uploads as 'points, synchronous UI ms (encode + native call)' with '*' on uploads that carried byte-identical GeoJSON. Chip shows N, adds, emissions.
    Verified: 6 unit/widget tests (classification boundaries, tolerance, clearing, percentiles, noteEmission frame identification); on the REAL screen in the harness 3 creations give adds=3, emissions=17 (3+7+7), uploads=8, redundant=4 (the 2-of-4 pattern) and the first frame of every creation classifies as 'emission'. NOT verified: that the clocks match on a device (documented, not measured) - if a device readout shows 'emission n=0' with adds>0 the assumption failed and the class split is void (the uploads line still holds).
  implication: the next device reading is interpretable per frame class instead of being a rolling mixture.

- timestamp: 2026-10-27 (session 2)
  checked: decision on commit c2187ab29 (keep or revert)
  found: |
    KEEP. Its value does not depend on the contaminated reading: on the old code the scratch test observed 19 native projections per camera frame and a sheet stand-in repainted 1.0x per camera frame; on the fixed layer the real-screen harness observes 5 projections (k >= 8; k below the batch threshold of 8 projects per point, so k) and 0 sheet-level rebuilds per camera frame, flat for k = 8..30. The clean-window readings with markers present on the OLD layer (no segments 9%, no sheet 7%) vs without markers (4% / bare 2%) also show markers cost something per pan frame there. Nothing in session 2 indicates harm (377 tests green in session 1; behaviour parity tests pass against both layers).
    What it is NOT: evidence that this was the cause of the report. The report is explained at least as well by (T).
  implication: leave it committed; do not revert without a device reading showing a regression.

- timestamp: 2026-10-27 (session 3)
  checked: SESSION 3 DEVICE READINGS R1-R4 (profile build = HEAD c2187ab29 + extended uncommitted probe, variant "all", 90 Hz so budget 11.1 ms), verbatim in the checkpoint response; summary
  found: |
    R1 creation burst (20 taps, no pan): adds=20 emissions=134 frames=300, over budget 38%. burst n=208 build 1.9/30.2/43.2 raster 5.3/8.1/11.5 total 12.7/40.8/57.0 over 112/208=54%; settled n=92 build 1.0/1.5/2.9 total 6.0/7.8/12.7 over 2/92=2%; emission n=0 (VOID). uploads line: see below.
    R2 pan at rest N=20: settled n=300 build 1.6/2.2/2.9 raster 2.6/3.7/5.2 total 6.1/8.0/11.5 over 4/300=1%; burst n=0; uploads=4 (578pt 2.4* 2.1* 1.9, 577pt 2.3). emissions rose 134 -> 141 with adds unchanged: the trailing network results of the last creation.
    R3 (taken at N=1 after the route was cleared, adds=21; NOT comparable to R2): burst n=93 total 6.4/10.4/14.5 over 8%; settled n=169 total 6.4/9.9/13.9 over 6%; uploads=1, 0pt, 0.2 ms.
    R4 (3 add/undo cycles at N=0, not 10 at N~20): paired readings (burst/settled) over budget 7%/13%, 0/2%, 7%/12%, 0/4%, 10%/17%, 0/3%. The pair after each cycle alternates: the window that contains the add reads 12-17% in 'settled' (late network emissions fall outside the 2.5 s burst window), the following pan-only window reads 2-4%.
    Step 6 (earlier 'low numbers at 20+': before or after c2187ab29) was not answered.
  implication: |
    (1) VERIFIED coordinator reading: pan at rest on HEAD is fine (R2 1%, falsification (i) did not trigger); the burst is the cost (R1 54% over budget in burst frames vs 2% settled in the same session).
    (2) CORRECTED: 'uploads=64 for 20 adds (3.2 per add)' is wrong. The probe keeps the last 64 uploads (ring cap), so 64 is a cap. 19 creations with a segment x 4 = 76 uploads were issued; the harness and the code both say exactly 4 per creation. 30 of the last 64 (47%) were byte-identical.
    (3) The 'emission' class is void because the probe compared two different stamps: FrameTiming.buildStart against SchedulerBinding.currentSystemFrameTimeStamp (documented as 'a more or less arbitrary value' on most platforms). They matched in tests only because the test binding fakes both. burst/settled still work as 'within 2.5 s of a state change' vs 'quiet' (next entry); the uploads line involves no clock at all.
    (4) R4 does not show accumulation (weak: 3 cycles, N=0), consistent with the harness finding no Dart-side accumulation and with R2: an at-rest 578-point upload after 20 creations costs 1.9-2.4 ms, the same size costs 7.7-27 ms inside the burst, so the slowness is transient contention, not state that accumulated.

- timestamp: 2026-10-27 (session 3)
  checked: why 'emission n=0' in every reading although burst n=208 in R1
  found: |
    classify() calls a frame 'emission' when |frame.buildStart - recorded| <= 1 ms and 'burst' when 0 < d <= 2.5 s after ANY recorded emission. The recorded value is SchedulerBinding.currentSystemFrameTimeStamp, i.e. the argument of PlatformDispatcher.onBeginFrame, which Flutter documents as 'a more or less arbitrary value' on most platforms (scheduler/binding.dart:1139-1150) - NOT as equal to FrameTiming.buildStart, as my session-2 note claimed. burst n=208 in R1 proves both are in the same time base on this device (otherwise d would be huge and nothing would classify as burst); the miss of the 1 ms test is therefore an OFFSET between the two stamps (most likely the vsync target time handed to the framework vs the actual BeginFrame Now() the timing records; not verified on device). Consequence: an emission frame has d of about -(1 frame interval), so it is neither 'emission' nor 'burst' for itself, but the frames after it - and, with 7 emissions per creation and continuous tapping, almost every frame in the tap window - are 'burst'. R2: burst n=0 is consistent (no recorded emission within 2.5 s before its frames; the 7 trailing emissions came before the user started panning).
  implication: |
    The burst/settled split is usable as 'within ~2.5 s after a state change' vs 'quiet', with the emission frames themselves (the expensive ones) folded into whichever class they fall into (burst in R1, since other emissions precede them). It does NOT isolate the emission frame, which is what the void class was for. So claim: during the 20-tap window 38% of ALL frames were over budget against 1% in the R2 pan-at-rest window of the same session, 54% of the frames within 2.5 s of a state change vs 2% of the rest; the uploads line is measured with a Stopwatch and involves no clock matching. The saved probe copy gets the classification fixed (Timeline.now, same clock as FrameTiming).

- timestamp: 2026-10-27 (session 3)
  checked: ENCODE vs JNI split (the orchestrator's question)
  found: |
    Dart-only part, AOT (dart compile exe, scratch /private/tmp/.../scratchpad/aot_bench/bench.dart, same shape as buildSegmentsGeoJson, Intel laptop): 570 pts / 23 KB = 0.704 ms; 3800 pts / 138 KB = 4.194 ms; 30 pts = 0.027 ms (median of 400 after 200 warm-ups). Device uploads at 570-578 pts: 1.9-2.4 ms at rest, 4.5 / 7.6 / 7.7 ms and 25.1 / 27.1 ms in the burst; 0.2 ms for an empty route.
    => at the measured size the Dart encode is about 0.7 ms: 30-37% of an at-rest upload, 16% of a 4.5 ms burst upload, ~3% of the 25-27 ms ones; the rest (about 1.3 ms at rest, up to ~26 ms in a burst) is JNI + native + whatever stalls them. Caveats: Intel-laptop AOT, not a Pixel; the bench coordinates carry 17 significant digits (40 B per point) whereas polyline6-decoded Valhalla points serialise shorter, so the Dart share is an upper bound for routed data.
    JNI path (maplibre_android 0.3.5 style_controller.dart:258-264): `_jStyle.getSourceAs<GeoJsonSource>(id.toJString())` then `source?.geoJson$3 = data.toJString()`: three global refs (id String, source, data String) never released explicitly. jni 1.0.0 dartjni.c:351-357 registers the finalizable handle with external size 0, so Dart's GC is not driven by the Java heap those strings pin; they live until the Dart wrapper is collected.
  implication: |
    compute() cannot help: it could move at most the 0.7 ms Dart part (and needs the segment objects deep-copied into the isolate), while the JNI call stays on the main isolate. The levers are the NUMBER of uploads and the bytes per upload. F1 removes the redundant half of the count. Whether bytes per upload (per-segment sources) are worth it is decided by the post-fix re-measure, not now.

- timestamp: 2026-10-27 (session 3)
  checked: stacking question (does a new add during in-flight work cancel or re-upload?)
  found: |
    By code: no cancellation across segments and no coalescing. _resolveSegment/_resolveElevation/_resolveAnchorLocation key their CancelToken and generation per segment key / anchor id, so only a re-edit of the SAME segment/anchor supersedes. A new add creates a new segment key and a new anchor id, so N rapid creations are N independent chains of (geocode, straight-height, route, routed-height), and every completion emits a state change that re-uploads the WHOLE route (all segments, including the ones other chains are still resolving). Counts are therefore additive, not multiplicative: 4 uploads per creation regardless of overlap (harness with settle between taps shows 4; the overlap test added in session 3 confirms with taps in quick succession). What stacks is the timing: with fast tapping the uploads of several creations interleave, so the burst window has more uploads per second and each one lands while the native side may still be converting the previous string (the 25-27 ms uploads in R1 are the 3rd/4th upload of a creation within about a second).
    Fan-out paths are worse: resolveAllSegments (switchProfile, reverseRoute) fires S independent Valhalla chains, i.e. 2S whole-route uploads of O(S) bytes each. F1 halves those (the S height results become no-ops).
  implication: no re-upload multiplication bug; the fix is to stop paying for the redundant half, and to keep the fan-out cases in mind as a follow-up if the re-measure still shows long bursts.

- timestamp: 2026-10-27 (session 3)
  checked: fixes, tests-first, with the real-screen harness promoted to app/test/routes/route_planner_screen_creation_test.dart
  found: |
    BEFORE (production files at HEAD c2187ab29, new tests): RED. 12 creations: uploads per creation 4 (expected 2) with 2 byte-identical (expected 0); overlapping taps (6 taps, then settle): 20 uploads (expected 10) = exactly 4 per creation, i.e. overlap changes the timing, not the count (answers the stacking question: additive, no multiplication); auto-routing off: 8 uploads for 4 segments (expected 4); marker widget rebuilds per creation 2k.
    AFTER (F1 + F2): GREEN. Per creation: uploads 2 (0 identical) for k = 2..30 (printed table of the scratch harness: `k | emis 7 | uploads(ident) 2(0) | marker rebuilds k`), overlap test 10 uploads, auto-routing off 4 uploads for 4 segments, marker rebuilds k (was 2k), screen/layer/flow rebuilds per creation unchanged and constant (2 each), per-camera-frame counters unchanged (5 projections, 0 screen / marker / list rebuilds). The final upload always equals buildSegmentsGeoJson(final segments) in every scenario, i.e. skipping never leaves the map behind.
    F1 = RouteSegmentLayer remembers the style, the segments list and the GeoJSON it last gave the native source; an update whose ids/state/polyline identity match (segmentsRenderIdentically, new in util/route/segment.dart) is dropped BEFORE encoding, one that encodes to an equal string is dropped before the JNI call, a different StyleController is never dropped, a failed upload clears the memo, remove() clears it.
    F2 = RouteAnchorLayer's marker memo is valid when ids, lat, lon and positions match (the cache holds the newest list afterwards so the per-camera-frame check is still an identity test).
    Mutation checks by hand (Stryker is JS-only): F1 4 of 4 killed (polyline ignored in the cheap check; style identity ignored; no forget after a failed upload; second check always skips), F2 4 of 4 killed (id ignored; length ignored; coordinates ignored; always rebuild = old behaviour). Two extra tests were added for F2 because the first round left mutants alive (appended anchor; a different anchor at the same coordinates).
  implication: the redundant half of the creation burst's uploads (and their Dart encode) and one whole-marker-set rebuild per creation are gone; what remains per creation is two real uploads, whose cost is JNI/native and route-size dependent.

- timestamp: 2026-10-27 (session 3)
  checked: verification runs
  found: flutter analyze (whole app): 13 infos, all pre-existing, none in touched files. flutter test test/components/map test/components/route_planner test/routes/route_planner_screen_creation_test.dart test/util/route test/util/map test/provider/route_anchor_provider_test.dart test/provider/planned_gpx_provider_test.dart = 238 passed (run before the probe copies were removed from the count; after removal the map/route/creation suites = 168 passed, analyze of the touched dirs clean). No flutter build / adb install was run.
  implication: off-device verification is complete; the remaining check is the user's.

- timestamp: 2026-10-27 (session 3, after the debugger returned)
  checked: user answer to R-step 6 ("which build gave the earlier 'low numbers at 20+ markers'"), relayed by the session manager's coordinator, treated as data
  found: |
    User: "It was with the same build that we did the initial tests". So the "low over-budget numbers with 20+ markers" came from the FIRST probe build, i.e. BEFORE c2187ab29 (stock ml.WidgetLayer in RouteAnchorLayer).
  implication: |
    On the old layer, at-rest pan with 20+ markers could already read low; the per-frame marker cost (N projections, sheet repaint) was therefore not shown to be necessary for the REPORTED symptom, which fits the burst explanation (the one 20% reading was the only one whose 300-frame window held creation frames). c2187ab29's projection/sheet changes are real and directly observed (19 -> 5 native projections per frame, sheet repaints 1.0 -> 0 per camera frame, device-free), but they are not shown to be the fix for the reported lag. The lag fix is 44800e490 + 1f692f90f.

- timestamp: 2026-10-27 (session 3, after the debugger returned)
  checked: user decision on whether c2187ab29 stays, relayed by the session manager's coordinator, treated as data
  found: |
    User: "It feels even smoother now. So I'd like to keep it". Subjective on-device feel, NOT measured (no probe numbers were taken for it).
  implication: |
    DECISION: c2187ab29 is KEPT. Reasoning: (1) it cuts a measured, deterministic per-frame cost (N native JNI projections and a sheet repaint per camera frame) that the old layer paid on every pan and zoom, (2) it is covered by tests that fail on the old layer, (3) no regression was observed, (4) the user prefers it after trying it. It is NOT claimed to be the root cause of the reported lag, and its commit message makes no such claim beyond "keep pan smooth as anchors grow". No history was rewritten: c2187ab29, 1f692f90f and 44800e490 stay as local commits on feature/app, not pushed. The burst fixes (1f692f90f, 44800e490) remain unverified on device; the user's "smoother" comment predates them.

DEVICE PROTOCOL handed to the user (session 2): build = working tree HEAD (c2187ab29 plus the uncommitted extended probe), profile build, variant chip stays on 'all' (default). Reading R1: tap STATS once and ignore (clears), tap the map ~20 times (any pace, no panning), wait 10 s, tap STATS = creation cost (emission class). R2: pan+pinch continuously ~10 s, no taps, tap STATS = at-rest pan at N~20. R3: tap the map once and start panning immediately for ~6 s, tap STATS = pan overlapping one creation. R4: 10 times [tap the map to add an anchor, wait 5 s, tap the Undo button], wait 10 s, tap STATS to clear, pan ~10 s, tap STATS = at-rest pan after 10 extra creations at constant N. Optional control: tap the PERF-AB chip 4 times to 'bare map', pan ~10 s, STATS, tap the chip once to return to 'all'. Also: which build gave the earlier 'low numbers at 20+ markers'.

## Eliminated

- hypothesis: per-frame cost proportional to the CURRENT marker count is what the user sees (C1/C2-by-count)
  evidence: user 2026-10-27 session 2: low numbers reachable with 20+ markers, count does not matter; the one reading that supported it (all@N=19 20%) is the only one whose window may contain creation bursts. Weakened to 'markers cost something per frame on the old layer' (clean-window 9% / 7% vs 4% / 2%); not eliminated for the old build, eliminated as the explanation of the report. The fix stays (see decision above).
  timestamp: 2026-10-27 (session 2)
- hypothesis: Dart-side accumulation per anchor creation (stacked ref.listen listeners, growing rebuild/upload counts, leaking tickers, growing per-camera-frame cost)
  evidence: real-screen harness, 30 creations: emissions 7, uploads 4, screen/layer/flow rebuilds 2 per creation for every k; per-camera-frame counters identical for k = 8..30; only transient callback left after 10 s idle is Riverpod's own one-shot; flutter_riverpod 3.3.1 closes the previous build's listeners (consumer.dart ~440). Scratch guard test passes.
  timestamp: 2026-10-27 (session 2)
- hypothesis: the probe itself accumulates across resets/variant changes (duplicate FrameTiming callbacks, uncleared window)
  evidence: single addTimingsCallback in start(), removed in dispose; rings capped; the only defect was un-cleared windows (fixed), not accumulation.
  timestamp: 2026-10-27 (session 2)

- hypothesis: lag comes from RoutePlannerScreen/sheet/list rebuilding per camera frame
  evidence: none of them read MapCamera/MapController; provider emits no state during pan; no always-running animation in route_planner/*.dart (grep for AnimationController|Ticker|Skeletonizer|Shimmer|Progress found only the sheet's TabController ticker)
  timestamp: 2026-10-27
- hypothesis: RouteAnchorLayer marker memoization is silently broken (cache miss every frame)
  evidence: cache key is identical(anchors) + selectedId + brightness, all stable during pan (static analysis); drag path already rebuilds only the dragged marker. Not runtime-verified.
  timestamp: 2026-10-27
- hypothesis: C2 raster/compositor cost of the blurred marker shadows (and sheet shadow) is the driver
  evidence: raster p50/p90/p99 is flat across all variants (2.3-3.2 / 4.2-5.2 / 5.5-8.4); all@N=19 raster p99 7.1 is below all@N=0 raster p99 8.4. Regression is entirely in BUILD.
  timestamp: 2026-10-27
- hypothesis: native segment geometry (C3) alone is the driver
  evidence: removing segments leaves markers+sheet at 9% over budget (build p90 3.5), not baseline; removing markers with segments+sheet present gives 4% (baseline). Segments only amplify.
  timestamp: 2026-10-27
- hypothesis: the sheet (C4) alone is the driver
  evidence: removing the sheet leaves markers+segments at 7% (build p90 2.8); no-markers with the sheet present is baseline (4%). Sheet only amplifies, and via the page-level repaint boundary shared with the marker Stack (scratch test: sibling repainted every camera frame).
  timestamp: 2026-10-27
- hypothesis: Dart heap/GC pressure from large segment polylines explains the no-segments gap
  evidence: the 'no segments' variant removes only native style layers; the Dart-side segments stay in provider state in every variant, so heap size is identical across variants.
  timestamp: 2026-10-27

## Resolution

SESSION 3 RESOLUTION (supersedes the STATUS paragraph below where they differ):

root_cause: |
  The lag the user feels is the anchor-creation burst, not steady-state panning. c2187ab29 already removed the per-frame cost proportional to the marker count (R2: pan at rest at N=20 is 1% over budget on HEAD). What remained: each creation emits seven provider states, four of which re-uploaded the WHOLE route as a GeoJSON string over JNI (2 ms at rest, 4.5-27 ms inside a burst at ~570 points on a Pixel; about 0.7 ms of that is Dart encode, the rest JNI/native), two of those byte-identical because they only attach elevations, and two of which rebuilt every marker widget (the second only for a reverse-geocoded `location` no marker reads). Uploads from overlapping creations add up (4 each) and the 25-27 ms ones are the 3rd/4th within about a second, so a pan or tap landing in the burst drops frames.
  Contributing, NOT addressed: why a same-size upload is 10x slower inside a burst than at rest (native converter / ART allocation GC / tile re-index overlap are candidates; only fewer and smaller uploads reduce them); bytes per upload grow with route length; fan-out edits (reverseRoute, switchProfile) emit 2 x segments whole-route uploads.
fix: |
  Two commits on feature/app (local, not pushed):
  44800e490 fix(app): skip route planner segment uploads that would draw nothing new - RouteSegmentLayer skips an update whose ids/state/polyline identity match what the source already shows (before encoding) or whose GeoJSON is string-equal (before the JNI call); keyed on the style instance; memo cleared on failure and remove(). New helper segmentsRenderIdentically in util/route/segment.dart.
  1f692f90f fix(app): stop rebuilding every route anchor marker for a location-only change - RouteAnchorLayer's marker memo is keyed on (id, lat, lon, position) instead of list identity.
verification:
  guardrail (off-device, DONE):
    target_test: { result: pass, test: "test/routes/route_planner_screen_creation_test.dart (creation uploads 2, none identical, markers <= k; overlap 10 uploads; auto-routing off 1 per segment; pan after creations) + test/components/map/route_segment_layer_test.dart + test/components/map/route_anchor_layer_test.dart + test/util/route/segment_test.dart", note: "agent-authored; the creation tests fail on the pre-fix production files (4 vs 2 uploads, 20 vs 10, 8 vs 4, 2k marker rebuilds) and pass now" }
    mutation_check: { result: pass, reason_if_skipped: "Stryker is JS-only, no Dart mutation tool is configured - done by hand", mutant_killed: "8 of 8 hand-seeded mutants killed (F1: polyline ignored, style identity ignored, no forget after failed upload, second check always skips; F2: id ignored, length ignored, coordinates ignored, always rebuild). F2 first pass left 2 mutants alive; two tests were added and all were then killed." }
    no_op_deletion: { result: pass, deletion_justified_by_rca: true, note: "F1 is an early return, so the detector applies. Justified by the RCA: the skipped call carries GeoJSON the map already shows (cheap check compares ids, state and polyline identity; the safety net compares the encoded strings), and every scenario test asserts that the last upload equals the GeoJSON of the final route state. A different style, a failed upload and remove() are never skipped (tests). F2 narrows a cache key, it removes no behaviour." }
    adjacent_tests: { result: pass, suites_run: [test/components/map, test/components/route_planner, test/routes/route_planner_screen_creation_test.dart, test/util/route, test/util/map, test/provider/route_anchor_provider_test.dart, test/provider/planned_gpx_provider_test.dart], count: 238 }
    revert_and_reconfirm: { result: pass, bug_returned_on_revert: true, fixed_on_reapply: true, note: "device-free level: the two production files were restored to HEAD and the creation tests failed (4 vs 2, 20 vs 10, 8 vs 4, 2k markers), then the fixes were restored and they pass. DEVICE-level revert-and-reconfirm is the pending human check below." }
    guardrail_verdict: accepted
  on-device (PENDING, human-verify checkpoint): |
    Build a profile build of HEAD 44800e490 yourself (no probe in it).
    Feel test, no tooling: tap ~20 anchors quickly on a route with realistic legs, then pan/zoom during and right after; compare with how it felt before. Also worth trying: drag an anchor in a 20-anchor route and add anchors after switching the travel profile.
    Numbers (optional): re-apply the scratchpad probe (README.txt in /private/tmp/claude-501/-Users-christianbeutel-Documents-svelte-wanderer/7e97f07d-aed4-4338-bea8-450907c73db2/scratchpad/perf-ab-probe/), profile build, protocol R1/R2 in that README.
    BEFORE numbers (HEAD c2187ab29, same chip): R1 20 taps: over budget 38% of 300 frames, frames within 2.5 s of a state change 54% over budget, build p50/p90/p99 1.9/30.2/43.2 ms; 4 update calls per creation (the ring showed 64 = its cap, 30 byte-identical); synchronous cost per call 4.5-27.1 ms at 551-578 points. R2 pan at rest N=20: 1% over budget, build p90 2.2 ms (this one was already fine).
    Pass: R1 shows 2 working calls per creation (the other two ~0.0 ms and starred), no byte-identical uploads, and the burst over-budget share clearly below 38% overall / 54% burst, with build p90 well below 30.2 ms. If the calls that remain are still 4-27 ms and the share stays >= ~40%, upload count was not the whole story: read the "(enc ..)" values (Dart vs JNI), then consider per-segment sources or a time throttle.
files_changed:
  - app/lib/components/map/route_segment_layer.dart
  - app/lib/util/route/segment.dart
  - app/lib/components/map/route_anchor_layer.dart
  - app/test/components/map/route_segment_layer_test.dart (new)
  - app/test/components/map/route_anchor_layer_test.dart
  - app/test/util/route/segment_test.dart
  - app/test/routes/route_planner_screen_creation_test.dart (new)
commits: [44800e490, 1f692f90f] (feature/app, local, not pushed) on top of c2187ab29
oracle_type: derived + metamorphic (the segment layer's contract is "the map is told exactly the GeoJSON of the current route, never more often than it changes", asserted against buildSegmentsGeoJson of the final state; metamorphic: an elevation-only change must not alter what is uploaded)
follow_ups_not_done: |
  (1) bytes per upload / per-segment sources, decided by the re-measure's "(enc ..)" split; (2) a per-segment GeoJSON cache (saves <= ~0.7 ms per upload at 570 points, ~4 ms at 3800); (3) cumulativeStatsByAnchorId select equality in the list tab (0.4 ms JIT at 20 anchors); (4) fan-out paths (reverseRoute, switchProfile, resolveAllSegments) still emit 2 x segments whole-route uploads, halved by F1 only for the height results; (5) upstream: maplibre_android 0.3.5 updateGeoJsonSource leaves the id String, data String and source object unreleased until the Dart wrapper is collected (jni finalizable handle external size 0) - candidate issue for maplibre-gl-native-flutter, not worked around; (6) the earlier-session note that c2187ab29's layer is a drop-in for the other ml.WidgetLayer users (trail/location markers) still stands.

SESSION 2 STATUS (kept for the record):  root cause of the REPORTED symptom is NOT confirmed. What is established: (1) the pre-fix layer did O(N) native projections and a sheet repaint per camera frame, and c2187ab29 removed both (directly observed, device-free); (2) the earlier 'necessary cause / AND-gate' conclusion rested on a reading whose window may have contained creation bursts, so it is withdrawn; (3) per creation the app does a burst of redundant, route-size-proportional UI-thread work (harness-verified), which is the leading explanation of 'the number goes up whenever I create a marker' and is awaiting a device reading; (4) no Dart-side accumulation exists. The text below is the session-1 resolution, kept for the record.

root_cause (session 1, partly withdrawn - see STATUS): |
  Necessary cause (confirmed by the no-markers variant restoring baseline and by a widget test on the pre-fix code): the stock ml.WidgetLayer behind RouteAnchorLayer does O(N) work on EVERY native camera event - N synchronous JNI toScreenLocation round trips, N Positioned+Transform allocations and relayouts, and a repaint of N blurred-shadow markers inside the page-level repaint boundary, which also re-records and re-rasterises the docked sheet (scratch test on old code: 19.0 projections/frame, sibling repainted 1.0x/frame). Build (UI thread) p90/p99 went 1.3/1.8 ms at N=0 to 8.6/21.0 ms at N=19, raster flat.
  Amplifiers: the docked sheet (shares the repaint boundary with the marker Stack - code-level mechanism verified) and the native segment layers (mechanism NOT proven; native CPU/thread contention is the working hypothesis; Dart-side segment data is identical in every variant). Over-budget frames: 20% observed vs 7% predicted by an additive model, i.e. superadditive.
fix: |
  commit c2187ab29 (feature/app, local only): RouteAnchorLayer now renders through a new ScreenMarkerLayer instead of ml.WidgetLayer. (1) positions via projectBatched: 5 native projections per frame regardless of N (bbox-corner affine fit in Web-Mercator space, validated on the 4th corner and the centre within 0.5 px, per-point fallback otherwise), gated off when MapCamera.pitch > 0.05; (2) markers placed by Flow (paint-time translation, no per-frame Positioned/Transform/relayout), Flow is its own repaint boundary so camera frames stop repainting the sheet; (3) one cached RepaintBoundary per marker so marker pictures are recorded once; (4) markers outside the viewport (+16 px) are not painted or hit-testable; (5) marker slots keyed by anchor id (AnimatedScale state follows the anchor). Segment layers and the sheet were deliberately NOT touched: no evidence they are the necessary factor, and removing the necessary factor breaks the AND-gate.
verification:
  guardrail (off-device, DONE):
    target_test: { result: pass, test: "test/components/map/route_anchor_layer_test.dart: a camera frame costs a constant number of native projections and leaves the rest of the page alone", note: "agent-authored; fails on the pre-fix layer with 19.0 vs 5 projections/frame, passes now" }
    mutation_check: { result: pass, reason_if_skipped: "Stryker is JS-only, no Dart mutation tool is configured - done by hand instead", mutant_killed: "6 of 6 hand-seeded mutants killed (centre probe removed, Stack+Positioned instead of Flow, per-marker boundaries removed, batching disabled, culling removed, wrapper keys removed). First pass left 3 survivors (outer boundary redundant with RenderFlow, per-marker boundary double-wrapped by Flow, keyed-state test vacuous because a re-created State also has the right id); the tests and the code were corrected, then all 6 were killed." }
    no_op_deletion: { result: pass, deletion_justified_by_rca: n/a, note: "diff is +1272/-7, additive; it replaces a stock widget with a purpose-built one, it does not remove behaviour. Tap / drag / numbering tests pass unchanged against both the old and new layer." }
    adjacent_tests: { result: pass, suites_run: [test/components/map, test/components/route_planner, test/util/map, test/util/route, test/provider] , count: 377 }
    revert_and_reconfirm: { result: pass, bug_returned_on_revert: true, fixed_on_reapply: true, note: "device-free properties only: swapping ScreenMarkerLayer back to ml.WidgetLayer in route_anchor_layer.dart makes the driving test fail (19.0 vs 5); restoring makes it pass. A git revert -n would also delete the tests, so the layer was swapped in place. The DEVICE-level revert-and-reconfirm is the pending human check below: the before numbers are the user's, the after numbers are not yet in." }
    guardrail_verdict: accepted
  on-device (PENDING, human-verify checkpoint): |
    Same profile build + PERF-AB chip, variant 'all', ~19-20 anchors, ~10 s pan+pinch, tap STATS.
    Before: build p50/p90/p99 2.3 / 8.6 / 21.0, raster 3.2 / 4.6 / 7.1, total 7.8 / 14.3 / 29.6, over budget 20%.
    Pass = build p90 <= 3.5 ms and over budget <= 9% (the best single-removal variants). Strong pass = near the no-markers numbers (build p90 ~2.0, p99 ~3.5, over budget ~4%).
    Fail = build p90 >= 6 ms or over budget >= 15% -> marker work was not the driver; reopen with planner_thread_cpu.py and C3 (segment layers).
    Control: cycle the chip once to 'no markers' in the same session and read it, so pan vigor / thermal state is controlled; read 'all' twice.
files_changed:
  - app/lib/components/map/route_anchor_layer.dart
  - app/lib/components/map/screen_marker_layer.dart (new)
  - app/lib/util/map/batched_projection.dart (new)
  - app/test/components/map/map_test_harness.dart (new)
  - app/test/components/map/screen_marker_layer_test.dart (new)
  - app/test/components/map/route_anchor_layer_test.dart (new)
  - app/test/util/map/batched_projection_test.dart (new)
commit: c2187ab29 fix(app): keep route planner pan smooth as anchors grow (local, not pushed)
oracle_type: derived (the fix's contract is a model: affine == per-point projection within 0.5 px; plus metamorphic: camera pan must not change marker/sheet paint counts)
temporary_files_CLEANUP_DONE (session 3): the PERF-AB probe, its test, the two tmp_* tests and the hooks in app/lib/routes/route_planner_screen.dart were removed from the tree and never committed. A restorable copy (README.txt, route_planner_screen.perf-ab.patch, the probe and the tests) is kept in /private/tmp/claude-501/-Users-christianbeutel-Documents-svelte-wanderer/7e97f07d-aed4-4338-bea8-450907c73db2/scratchpad/perf-ab-probe/ for an on-device re-measure.
future_note: TrailMarkerLayer (components/map/trail_layer.dart), LocationMarkerLayer, list/map screens still use ml.WidgetLayer with the same per-frame O(N) cost; ScreenMarkerLayer is a drop-in (non-rotating, non-flat markers). Not changed here - out of scope for the planner bug.
