---
slug: recording-killed-mid-session
status: awaiting_human_verify
trigger: "Issue #1321: Android recording dies mid-session (Samsung Galaxy S21 FE, Android 16, app 0.1.0, battery Unrestricted, location Allow all the time). User Hendrik ran 7 km with RunnerUp in parallel: RunnerUp clean 744 pts, no gaps. Wanderer recorded fine until 10:38:43 (~8 min, ~1.3 km), then nothing for 32.6 min; on reopening, app cold-started and appended 2 points near current position, producing a ~1 km straight-line jump. Hypothesis from user: earlier zig-zag/50 km track (first report) was repeated kill/restart jump segments, not GPS noise."
created: 2026-10-15
updated: 2026-10-15T21:30:00Z
---

# Debug Session: recording-killed-mid-session

## Symptoms

- **Expected behavior:** Background recording keeps logging fixes for the whole session; if the app process dies, reopening fills the gap from tracelet's persisted store (backfillSessionGap).
- **Actual behavior:** Fixes stop after ~8 min (last at 10:38:43); 32.6 min gap; on reopen the app cold-starts and appends 2 live points, producing a ~1 km straight-line jump. RunnerUp on the same phone recorded cleanly the whole time (744 pts, 7.01 km).
- **Error messages:** None visible; the process was killed (cold start on reopen).
- **Timeline:** First reported 2026-10-04 (zig-zag, ~50 km recorded for a ~5 km walk), again 2026-10-11 with a parallel RunnerUp recording. App 0.1.0, tracelet 3.8.8.
- **Reproduction:** Samsung Galaxy S21 FE, Android 16, battery "Unrestricted", location "Allow all the time"; record a walk/run with the screen off. Not reproduced locally.

## Data (untrusted user files, copied to scratchpad)

- Wanderer export: /private/tmp/claude-501/-Users-christianbeutel-Documents-svelte-wanderer/18b4facc-777b-47ff-9a6a-3473dd286d0a/scratchpad/gpx/trail-2026-10-11T13_11_28.082969.gpx
- RunnerUp parallel: /private/tmp/claude-501/-Users-christianbeutel-Documents-svelte-wanderer/18b4facc-777b-47ff-9a6a-3473dd286d0a/scratchpad/gpx/RunnerUp.gpx
- First report (2026-10-04, snapped to roads by the user, so less useful): ~/Downloads/trail-2026-10-04T11_41_57.437892.gpx.txt (copy into its own scratchpad dir before reading; run any Python with -I)

## Initial observations (orchestrator)

- Wanderer fixes every 3-5 s until 10:38:43.
- Three fixes share the timestamp 10:38:35: batched delivery just before the stop.
- The last two points (11:11:18, 11:11:25) have 7+ decimal precision and raw elevation 67.0; all earlier points are rounded to 5 decimals with filtered elevation from 10:37:55 on. So the last two came via a different code path (likely live _onLocation after resume, not backfillSessionGap).
- Key question: did the native tracelet service die too (nothing stored between 10:38:43 and 11:11, so backfill had nothing), or did tracelet keep recording while backfill skipped those fixes (isMoving false, getLocations retention, timestamp parsing, PersistMode)? Also: why are points rounded at all, and does the jump come from appending live fixes after a gap with no segment break?

## Relevant code

- app/lib/services/tracelet_position_source.dart (tracelet 3.8.8, stopOnTerminate:false, fg/bg configs swapped via setConfig, speed motion detection, stationaryPeriodic)
- app/lib/services/session_gap_backfill.dart
- app/lib/main.dart: _maybeResume / _maybeResumeRecording / _pushRecordingResume (isTracking() decides direct resume vs dialog)
- app/lib/routes/navigation_screen.dart (_persistNow, lifecycle handling)
- tracelet sources in the pub cache (tracelet, tracelet_android) for the foreground service, persistence and stopOnTerminate behaviour
- Memory: project-deferred-notification-repost (tracelet notification/heartbeat findings)

## Constraints

- The user builds and installs; never run flutter build or adb install. Hand off after flutter analyze + flutter test.
- No device available showing the bug; reason from code, tracelet source, and the GPX data. Where an on-device check is needed, specify exactly what to capture (e.g. adb logcat filters, dumpsys) for the user to ask Hendrik for.
- Commit messages headline-only, no body, no Co-Authored-By line. Do not push.

## Current Focus

- bug_class: Bohrbug (deterministic ordering defect in gap backfill) sitting behind a Mandelbug (OS-level process death, cause undetermined)
- scope-decision: fix only what is proven. Proven: H1 (backfill reads tracelet's store before ready()). NOT changing: the motion-block / restart-on-lifecycle finding (E2: unproven as cause, changes field battery behaviour) and the chord-across-gap behaviour (same as the existing manual-pause chord; product decision).
- reasoning_checkpoint:
    hypothesis: "backfillSessionGap() returns 0 on every cold start after a real process death, because it calls Tracelet.getLocations() before Tracelet.ready(), and tracelet-sdk 3.8.8 TraceletSdk.getLocations() answers emptyList() while isReady==false (isReady is only set by completeReady() via ready())."
    confirming_evidence:
      - "tracelet-sdk-3.8.8 TraceletSdk.kt:2184 `if (!isReady) return emptyList()`; TraceletSdk.kt:876 sole `isReady = true` inside completeReady(); TraceletSdk.kt:2360 comment: headless boot path 'never calls ready() ... isReady stays false'; HostApiImpl.getLocations (line 655) has no NOT_READY guard so the empty list is a normal success."
      - "app/lib: Tracelet.ready() has one call site (TraceletPositionSource.start, from NavigationScreen post-frame); backfillSessionGap is called from main.dart _pushRecordingResume/_pushNavigationResume before that screen exists."
      - "GPX: every pre-gap point is 5-decimal (rehydrated from the polyline row) and the 2 tail points are full-precision live fixes: zero points were backfilled, over a 1955 s gap during which RunnerUp proves the user was moving."
    falsification_test: "If a cold start called ready() (or anything setting isReady) before backfill, or if getLocations() did not gate on isReady, H1 is wrong. Also wrong if a process-alive Activity re-creation (isReady still true) were the actual scenario - in that case this fix is harmless but not causal for the missing points (then native really recorded nothing)."
    fix_rationale: "Make readiness an explicit precondition of the backfill: backfillSessionGap takes an ensureTraceletReady callback and awaits it before reading the store; main.dart supplies one that runs the same ready(foreground config) the screen would run. This addresses the cause (store unreadable), not the symptom (missing points). Also move the connectivity probe ahead of the backfill so fewer fixes can fall in the unbackfilled window between backfill and the screen's onLocation subscription."
    blind_spots: "(1) Cannot tell from available data whether tracelet's native layer actually recorded during the gap; if the process was killed and the sticky restart was refused/never happened, the store is empty and this fix recovers nothing (the chord remains). (2) Why the process died is undetermined. (3) A sub-second window between getLocations() and the screen subscribing to onLocation can still drop ~1 fix at the seam. (4) A second ready() ahead of the screen's own ready() restarts the native pipeline once more on resume (the screen already does this on every session start)."
    candidate_causes:
      - "code: backfill reads the store before ready() (H1, confirmed)"
      - "environment: OS/OEM killed the app process on Samsung One UI 8 / Android 16 (cause not determinable offline)"
      - "config: fg<->bg profile swap restarts the native pipeline and switches motion to smart/geofences in the background (E2, unproven)"
      - "data: n/a"
    and_gate: "yes - the user-visible loss needs two conditions together: (a) something stops the live stream / kills the process (environment), AND (b) the recovery path is dead (code, H1). Fixing (b) is the only part actionable from the repo; (a) needs on-device capture."
- next_action: awaiting on-device confirmation from the user (see Resolution.verification); then archive. Open: why the process died (needs ApplicationExitInfo from Hendrik's phone).

## Evidence

- timestamp: 2026-10-15T19:30Z
  checked: knowledge base (.planning/debug/knowledge-base.md)
  found: file does not exist; no prior resolved session matches. Related unresolved sessions exist: recording-gps-jitter-still, recording-elevation-gain-jump.
  implication: no known-pattern candidate.

- timestamp: 2026-10-15T19:35Z
  checked: Wanderer GPX (127 pts) via python -I analysis in scratchpad/scripts/analyze.py
  found: pts 0-124 are 5-decimal lat/lon; pts 125-126 are 7+ decimals. Gap 10:38:43 -> 11:11:18 = 1955 s, straight line 1046.9 m. Pts 120-122 share 10:38:35 (dt 0; d=10.7, 23.9, 10.3 m) = Dart received 3 fixes in one burst after a 6 s stall. Cadence before: 3-5 s / ~11 m (=~2.8 m/s running). Elevation: pts 1-108 are float32-quantised (67.0, 66.80000305...), pts 109-124 (from 10:37:55) are continuous float64 (72.237..., i.e. filtered); pts 125-126 both ele=67.0 (raw, same as pt 1).
  implication: (1) 5-decimal rounding = PolylineUtil.encode in _persistNow, so ALL of pts 0-124 were rehydrated from the persisted ActiveNavigationEntity row on cold start => the process really died and NavigationScreen was rebuilt from the row (consistent with report). (2) pts 125-126 are live _onLocation fixes after resume, NOT backfill (backfill would also be polyline-encoded + elevations from tracelet store, and would have produced hundreds of points). (3) Burst at 10:38:35 shows the Dart side was already starved ~8 s before the last point.

- timestamp: 2026-10-15T19:50Z
  checked: RunnerUp.gpx (744 pts, 8 trkseg) via scratchpad/scripts/runnerup.py
  found: user ran continuously 10:31:20 -> 11:10:48, 7005 m, a loop that ends 167 m from start at 11:09:23; no stop, max dt gap 9 s. At 10:38:45 RunnerUp is at 53.115803,9.444542, Wanderer's last point 53.11571,9.44441 (~15 m apart, same place). Wanderer reopened 11:11:18, ~30 s after the run ended.
  implication: during the whole 32.6 min gap the user was moving at ~2.9 m/s. There is no "stationary" explanation for the silence; GPS hardware/provider worked for another app. The 1047 m straight line is a chord across ~5.7 km of real running.

- timestamp: 2026-10-15T20:05Z
  checked: tracelet-sdk 3.8.8 sources (https://repo1.maven.org/maven2/com/ikolvi/tracelet-sdk/3.8.8/tracelet-sdk-3.8.8-sources.jar, extracted to scratchpad/dl-sdk-388/src) - TraceletSdk.kt
  found: getLocations() line 2184: `if (!isReady) return emptyList()`. isReady is set true only in completeReady() (line 876), reached only via ready(). bootstrapForBackground() explicitly never calls ready() (comment at insertLocation line 2360). TraceletAndroidPlugin.onAttachedToEngine only calls sdk.initialize(); TraceletStartupProvider only wires bridges. HostApiImpl.getLocations (line 655) does NOT check isReady, so the empty list comes back as a normal success, no NOT_READY error, nothing for backfill's try/catch to catch.
  implication: on a process that has not yet run ready() (every cold start after a process kill), getLocations() == [] regardless of what is in the SQLite store.

- timestamp: 2026-10-15T20:10Z
  checked: app call sites (grep Tracelet.* in app/lib)
  found: Tracelet.ready() is called in exactly one place: TraceletPositionSource.start(), invoked from NavigationScreen.initState's post-frame callback. backfillSessionGap is called from main.dart _pushNavigationResume/_pushRecordingResume, i.e. BEFORE the screen is pushed. isTracking() (getState) is fine pre-ready: SDK getState() reads persisted StateManager and does not gate on isReady.
  implication: H1 CONFIRMED as a deterministic ordering defect: backfill can only work when the Activity was recreated inside a still-living process (isReady stays true in the SDK singleton). After a real process death it never works. This matches the user's "cold start" observation and the absence of backfilled points (the 2 tail points are live fixes: full precision).

- timestamp: 2026-10-15T20:20Z
  checked: tracelet Config presets + SDK setConfig() + app profiles (tracelet_position_source.dart)
  found: (a) _backgroundConfig() = Config.balanced().copyWith(...) does not override `motion`, so it ships balanced preset motion {motionDetectionMode:2=smart, stationaryTrackingMode:1=geofences, stopTimeout:5} on top of the foreground's speed/periodic block. (b) SDK setConfig() sets needsRestart when any location/motion key changes (desiredAccuracy, distanceFilter, locationUpdateInterval, stationaryRadius, elasticity, motionDetectionMode, stationaryTrackingMode, stopTimeout...), and then does stop(preserveForegroundService=true) + start(isResume=true) = a full native pipeline restart on EVERY foreground<->background transition. The class doc comment "swaps profiles without stopping/restarting the underlying tracking session" is therefore false.
  implication: the pipeline is torn down and rebuilt in the background each time the screen goes off. Not shown to be the cause of this incident (user was moving; SMART needs both inputs to agree on stationary, and the accelerometer detector never starts without ACTIVITY_RECOGNITION), but it is an undocumented behaviour change and a risk; the 10:37:55 change in altitude characteristics (float32-quantised -> continuous) lines up with such a restart ~48 s before the last fix.

- timestamp: 2026-10-15T20:25Z
  checked: tracelet always-on lifecycle log + what the app captures
  found: tracelet 3.8.x writes process/service lifecycle to a persistent log that survives process death ("service: onDestroy - stopRequested=", "service: sticky restart after process death", "boot-tracking: bootstrapping", "foreground-service: ... REFUSED / promoted ... denied location", "location stream stalled", "session: start/stop restart=") readable via Tracelet.getLog()/getLogs(); retention logMaxDays (default 3) and pruned on every ready() (completeReady -> pruneOldLogs). The app never reads getLog/getForegroundServiceHealth/getLogs anywhere.
  implication: the only ground truth for WHY the process/service died is on Hendrik's phone and is probably already pruned (incident 2026-10-11, ready() prunes >3 days). Nothing in the app can surface it for the next incident.

## Eliminated

- hypothesis: tracelet #405 (service record created from background is denied location) caused the silence
  evidence: that failure mode needs a while-in-use-only permission; the user has "Allow all the time" (ACCESS_BACKGROUND_LOCATION), and Android grants a background-started location FGS access in that case. Also battery is "Unrestricted", which exempts the app from the background FGS-start restriction. Not proven absent, but not supported.
  timestamp: 2026-10-15T20:30Z

- hypothesis: backfill ran but skipped the gap because of isMoving / timestamp comparison / PersistMode
  evidence: the user was moving the whole time (RunnerUp), and the ms<=lastMs comparison can only skip fixes already in the breadcrumb; with getLocations() returning [] pre-ready none of those filters is even reached.
  timestamp: 2026-10-15T20:30Z

## Resolution

root_cause: |
  Two conditions, ANDed (the loss needs both):
  (1) ENVIRONMENT, UNDETERMINED: something ended the recording process (or its live stream) at ~10:38:43 while the user kept running. The available evidence cannot say what (see Evidence; the on-device capture that can is listed under verification.open_items).
  (2) CODE, CONFIRMED: the recovery path was dead on every real cold start. backfillSessionGap() called Tracelet.getLocations() before Tracelet.ready(); tracelet-sdk 3.8.8 TraceletSdk.getLocations() returns emptyList() (as a normal success, no error) while isReady==false, and isReady is only set by ready(), which the app calls solely from NavigationScreen's startup, after the backfill. So whatever tracelet's SQLite store held for the 1955 s gap was never read; the resumed breadcrumb got only the two live fixes, joined to the old tail by a 1047 m straight chord.
fix: |
  backfillSessionGap() now takes a required ensureTraceletReady callback and awaits it before reading the store (best-effort: a failure there returns 0 and leaves the row untouched). main.dart supplies _readyTraceletForBackfill, which runs the same foreground-profile ready() the screen runs (new TraceletPositionSource.ensureReady, factored out of start() so the two cannot diverge). main.dart also now runs the connectivity probe BEFORE the backfill instead of after, so fewer fixes fall into the window between the store read and the screen's onLocation subscription. Comment-only: corrected setForeground's doc, which wrongly claimed profile swaps do not restart the native pipeline.
files_changed:
  - app/lib/services/session_gap_backfill.dart
  - app/lib/services/tracelet_position_source.dart
  - app/lib/main.dart
  - app/test/services/session_gap_backfill_test.dart
commit: 7596a362f (feature/app, not pushed)
verification:
  target_test: { result: pass, test: "app/test/services/session_gap_backfill_test.dart (5 tests; 'readies tracelet before it reads the store' models the SDK's isReady gating)" }
  mutation_check: { result: skipped, reason_if_skipped: "Stryker is not applicable to this Dart project; replaced by a manual mutant at the fix site (the `await ensureTraceletReady();` line commented out): 2 of 5 tests failed (call order ['read'] != ['ready','read'], and the failing-ready test spliced 4 points instead of 0) = mutant killed. Original restored and re-verified.", mutant_killed: true }
  no_op_deletion: { result: pass, deletion_justified_by_rca: false, note: "diff is additive (new parameter, new call, new static method, new test); the only removals are the moved ready() lines in start() which now live in ensureReady()" }
  adjacent_tests: { result: pass, suites_run: ["flutter test (full app suite): 1328 passed, 1 skipped, 0 failed"], analyze: "flutter analyze: 13 pre-existing info-level issues, none in touched files" }
  revert_and_reconfirm: { result: pass, bug_returned_on_revert: true, fixed_on_reapply: true, note: "done at unit level via the mutant above. NOT possible for the real symptom: no device, and the process death itself is not reproducible." }
  guardrail_verdict: accepted
  open_items:
    - "ON-DEVICE (user builds/installs): start a recording, move ~2 min, then run `adb shell am crash com.openwanderer.wanderer` (a real process death that keeps Android's sticky-service restart; `am kill` only kills processes without a foreground service, so it does nothing here, and `am force-stop` cancels the sticky restart so tracelet cannot record afterwards). Keep moving ~2 more minutes, then reopen the app. Outcome A: the track is continuous across those 2 min (hundreds of fixes spliced) = the fix works AND tracelet survives process death. Outcome B: still a chord = tracelet did not record after the process died (sticky restart refused/never happened); the fix is then not exercised and the next suspect is tracelet's restart path (capture `adb logcat -d | grep -i -E 'tracelet|LocationService'` right after reopening, and `adb shell dumpsys activity services com.openwanderer.wanderer` while it should be recording)."
    - "WHY did the process die? Not determinable offline. Best single capture: `adb shell dumpsys activity exit-info com.openwanderer.wanderer` (ApplicationExitInfo, Android 11+, survives days/reboots, keeps ~16 entries): the entry near 2026-10-11 10:38-10:39 gives reason (LOW_MEMORY / CRASH_NATIVE / SIGNALED / FREEZER / USER_REQUESTED / OTHER) and description. Needs adb on Hendrik's phone (or `adb bugreport` zip). Second: reproduce on his phone with screen off and capture `adb logcat -b all -v threadtime | grep -E 'wanderer|LocationService|Tracelet|am_proc_died|am_kill|lowmemorykiller|lmkd|FreecessHandler|Freezer'` during the failure."
    - "Not fixed by design, flagged: (a) an unrecoverable gap still joins the old tail and the next fix with a straight chord in the saved GPX, exactly like a manual pause does today; stats (distance/elapsed) do NOT include the chord because the resumed NavigationStatsNotifier re-anchors on the first live fix. (b) fg<->bg profile swap restarts the native pipeline and silently switches motion to smart/geofences in the background (E2) - unproven as cause, would change field battery behaviour, left alone."
