---
slug: recording-elevation-gain-jump
status: awaiting_human_verify
trigger: "Follow up: sometimes when loading into the recording the recording shows an instant elevation gain to the current altitude. This probably happens because the first n points do not get a proper elevation read and then the first point that does causes the elevation statistics to jump instantly. This is not new behaviour that was introduced with the last debug session."
created: 2026-08-09
updated: 2026-08-09
---

# Debug Session: recording-elevation-gain-jump

## Symptoms

- **Expected behavior:** Starting a recording, the elevation-gain stat should begin at 0 m and only accumulate real climb thereafter. An initial altitude reading — whatever it is — should only ANCHOR the reference, never itself count as gain.
- **Actual behavior:** Sometimes, shortly after entering the recording screen, the elevation-gain stat jumps instantly to roughly the user's current altitude above sea level (e.g. ~500 m in Munich) in a single step, without the user having climbed anything.
- **Where it shows:** Only the elevation-gain STAT in the recording sheet. The elevation profile chart looks fine — the chart is NOT affected. (This narrows the fault to the stats accumulation path, not the GPX/chart parsing path.)
- **Magnitude:** Approximately the absolute altitude above sea level — consistent with a reference anchored at 0 (or at a point with null/0 altitude) followed by a first real reading, giving `gain = realAltitude - 0`.
- **Error messages:** None reported.
- **Timeline:** Pre-existing. Explicitly NOT introduced by the previous debug session (`elevation-profile-recording`, which only changed which widget renders on the recording elevation page and touched no stats logic).
- **Reproduction:** Intermittent ("seems random") — happens on ordinary fresh starts, not limited to cold start / poor GPS / resumed recordings as far as the user can tell.

## User's own hypothesis (to test, not to assume)

The first n GPS points do not carry a proper elevation reading; the first point that does then causes the elevation statistics to jump instantly.

## Scope notes

- Platform: Flutter app (`app/lib/`). Working branch: `feature/app`.
- Prime suspect: `app/lib/provider/navigation_stats_provider.dart` — the altitude reference anchoring / gain accumulation. Existing tests there include "first onPosition only anchors reference: distance stays 0, no throw", "altitude delta below noise floor (+1.0 m) does not accumulate gain", and several re-anchor tests for pause/resume/stationary. An anchoring hole that those tests do not cover is the likely culprit.
- Related: `app/lib/provider/navigation_provider.dart:168` appends `Wpt(..., ele: altitude, ...)` per fix — worth checking what `altitude` is when the platform has no altitude yet (0.0? null coalesced to 0?).
- The chart being unaffected is a strong discriminator: the chart derives elevation from the breadcrumb Wpts, the stat derives it from the position stream in the stats notifier. Whatever is wrong is in the latter, or in how the former filters what the latter does not.

## Current Focus

- hypothesis: `NavigationStatsNotifier.onPosition` (app/lib/provider/navigation_stats_provider.dart) anchors `_lastAltitude` unconditionally on the first fix it ever sees, with no check for whether that fix actually carries a real altitude reading. `TraceletPositionSource.seedPositionFrom` (app/lib/services/tracelet_position_source.dart:15-26) constructs a synthetic "seed" `geo.Position` with `altitude: 0, altitudeAccuracy: 0` (LocationMarkerPosition never tracks altitude) and pushes it onto the SAME position stream that feeds both the stats notifier and the breadcrumb, immediately in `start()` (line 131-133), before tracelet's real GPS engine produces its first fix. When this zero-altitude seed is the first position `onPosition` sees, it anchors the reference at 0.0. The very next (real) tracelet fix reports the device's actual absolute altitude (e.g. ~500 m in Munich); since the reference is no longer null, the full delta is treated as gain in one step.
- test: Add a regression test to app/test/provider/navigation_stats_provider_test.dart that feeds `onPosition` a zero-altitude/zero-altitudeAccuracy position (mirroring `seedPositionFrom`'s output) followed by a real 500 m altitude/5.0 m accuracy fix, and asserts `elevationGainMeters` does NOT jump to ~500. Run it before the fix (expect FAIL, confirming reproduction) and after the fix (expect PASS).
- expecting: pre-fix red (gain ≈ 500), post-fix green (gain == 0, reference re-anchors on the first fix that carries a real altitudeAccuracy > 0).
- next_action: DONE — fix implemented, red/green regression test added, flutter analyze + full flutter test suite green. Awaiting human on-device verification of the original real-world symptom (open a fresh recording where the map marker already had a resolved position, confirm elevation-gain stat starts at 0 and does not jump).
- reasoning_checkpoint:
    hypothesis: "The seed position's fabricated altitude=0/altitudeAccuracy=0 (from TraceletPositionSource.seedPositionFrom) is treated by NavigationStatsNotifier.onPosition as a genuine first altitude reading, anchoring the gain/loss reference at 0 instead of skipping it — causing the next real fix's absolute altitude to register as instant gain."
    confirming_evidence:
      - "app/lib/services/tracelet_position_source.dart:15-26 seedPositionFrom explicitly sets altitude:0, altitudeAccuracy:0, with a doc comment stating altitude/speed are zeroed because LocationMarkerPosition never tracks them."
      - "app/lib/services/tracelet_position_source.dart:131-133 start() pushes this seed onto _controller immediately, before tl.Tracelet.start()/onLocation ever fires, and this is the same stream navigation_screen.dart's single _sub listener feeds to both _navNotifier.onPosition and _statsNotifier.onPosition (no filtering in between)."
      - "app/lib/provider/navigation_stats_provider.dart:210-224 onPosition's altitude branch: `if (_lastAltitude != null) {...} else { _lastAltitude = pos.altitude; }` — unconditional anchor on first-ever call, no accuracy/finite gate."
      - "geolocator_platform_interface-4.2.6 Position doc: altitude AND altitudeAccuracy both documented to read 0.0 when altitude is unavailable on the device — the SDK's own convention for disambiguating a genuine sea-level reading from 'no reading'."
      - "app/test/provider/navigation_stats_provider_test.dart's _pos() helper hardcodes altitudeAccuracy: 5.0 for every existing test — no existing test exercises the altitudeAccuracy: 0 case, explaining why this hole was never caught."
    falsification_test: "Feed onPosition the sequence [Position(altitude:0, altitudeAccuracy:0), Position(altitude:500, altitudeAccuracy:5)] on a fresh notifier. If elevationGainMeters reads ~500 after the second call, hypothesis is confirmed (matches reported magnitude/mechanism). If it reads 0, hypothesis is wrong and anchoring is not the mechanism."
    fix_rationale: "Gating the anchor/accumulation on pos.altitudeAccuracy > 0 (rather than lowering the noise floor or touching distance logic) fixes the ROOT CAUSE — treating a fix with no real altitude data as unusable for elevation purposes, exactly like the codebase's existing hasUsablePosition/parseGpxElevation philosophy elsewhere (conversion.dart) — without changing _kAltitudeNoiseFloorMeters (2.0, explicitly off-limits) or any distance/XY logic (thresholdXY_m, also off-limits). It also correctly generalizes to the user's own hypothesis (early real GPS fixes before altitude lock, which report the same 0/0 sentinel) without hardcoding anything seed-specific."
    blind_spots: "Cannot run the app on-device to observe a live Munich recording session (mobile app; user builds/installs per project convention). Relying on geolocator's/tracelet's documented 0.0 sentinel convention for 'altitude unavailable' — if a real GPS fix ever legitimately reports altitudeAccuracy exactly 0.0 (undocumented edge case on some device/OS combo), that single fix would be skipped for anchoring purposes, which is a safe no-op (anchors on the next fix instead), not a new bug. Have not verified whether GpxMetricsComputation/the chart's OWN header stat shares this exact defect for the same seed value (out of scope — prime suspect and off-limits files point at the stats path only, and the write-up is being scoped to that per project instructions)."

## Evidence

- timestamp: 2026-08-09T00:00:00Z
  checked: app/lib/provider/navigation_stats_provider.dart (full file)
  found: onPosition's altitude branch (lines 210-224) anchors `_lastAltitude = pos.altitude` unconditionally whenever `_lastAltitude == null` (i.e. on the very first call after construction or after any re-anchor via `_applyFrozen`). No check on `pos.altitudeAccuracy`, finiteness, or any other "is this a real reading" signal — every other anchor point in this file (`_lastPoint`, `_lastAltitude`) is reset together in `_applyFrozen`, so the same hole applies after every pause/stationary resume too, not just session start.
  implication: Any fix delivered as the first `onPosition` call after start/resume permanently seeds the elevation reference, even if that fix's altitude field is a placeholder rather than a real reading.

- timestamp: 2026-08-09T00:00:00Z
  checked: app/lib/provider/navigation_provider.dart (full file), specifically onPosition's breadcrumb append (line 168)
  found: `Wpt(lat: pos.lat, lon: pos.lon, ele: altitude, time: DateTime.now())` — `altitude` is the nullable `double?` parameter of `Navigation.onPosition`, but the caller (navigation_screen.dart) always passes `pos.altitude` from a geolocator `Position`, whose `altitude` field is a non-nullable `double` that is documented to read `0.0` (not null) when unavailable. So this parameter's nullability can never actually engage as a "skip anchoring" signal for a live GPS/tracelet-sourced position — it would only be null if some other future caller passed null explicitly.
  implication: The breadcrumb/GPX path receives the exact same ambiguous 0.0 that the stats path does; whatever downstream anchoring exists there (GpxMetricsComputation) faces the identical ambiguity, just via a different accumulator.

- timestamp: 2026-08-09T00:00:00Z
  checked: app/lib/services/tracelet_position_source.dart (full file)
  found: `seedPositionFrom()` (lines 15-26) builds a `geo.Position` with `altitude: 0, altitudeAccuracy: 0` explicitly, with a doc comment: "Altitude/speed aren't tracked by LocationMarkerPosition, so they're zeroed rather than left to fall back on stale values." `start()` (lines 118-137) immediately does `_controller.add(seed)` (line 132) if a seed was passed, BEFORE `tl.Tracelet.ready()/start()` ever runs — so this synthetic position is guaranteed to be the FIRST item on `stream` whenever a seed is supplied. `_onLocation` (lines 148-165), which handles every subsequent REAL tracelet fix, maps `c.altitude`/`c.altitudeAccuracy` straight through with no filtering.
  implication: Whenever `NavigationScreen` is opened with an already-resolved `initialPosition` (the live map marker's last known location), the very first position on the combined stream is this zero-altitude synthetic seed — which is the trigger condition for the bug. This also explains the reported intermittency: it depends entirely on whether a `LocationMarkerPosition` happened to already be resolved before the recording screen opened.

- timestamp: 2026-08-09T00:00:00Z
  checked: app/lib/routes/navigation_screen.dart lines 395-487 (read-only — file is off-limits for edits per orchestrator notes)
  found: A single `_sub = _positionStream.listen((pos) {...})` (line 454) feeds the exact same `pos` to `_navNotifier.onPosition(...)` (line 466, breadcrumb) and `_statsNotifier.onPosition(pos)` (line 475, stats) — confirming both consumers see the identical sequence including the synthetic seed, with zero filtering between the stream and either consumer.
  implication: The divergence in observed behavior (stat wrong, chart "fine") must originate inside the two consumers' own accumulation logic, not from the stream feeding them differently — consistent with the orchestrator's framing.

- timestamp: 2026-08-09T00:00:00Z
  checked: geolocator_platform_interface-4.2.6/lib/src/models/position.dart (pub cache) and tracelet_platform_interface-3.5.0's generated Coordinates class
  found: geolocator's `Position.altitude` doc: "The altitude is not available on all devices. In these cases the returned value is 0.0." Identical wording for `altitudeAccuracy`. tracelet's `Coordinates.altitude`/`altitudeAccuracy` are likewise non-nullable `double` fields (tracelet_api.g.dart:1988/1990), matching the same convention.
  implication: `altitude == 0.0` is fundamentally ambiguous between "genuine sea-level reading" and "no reading available" at the SDK boundary — for both geolocator and tracelet. The SDK's own disambiguating signal is `altitudeAccuracy` (also 0.0 exactly when altitude is unavailable), which the codebase does not currently check anywhere before treating an altitude value as real.

- timestamp: 2026-08-09T00:00:00Z
  checked: app/lib/util/gpx/conversion.dart's GpxMetricsComputation.addAndFilter and parseGpxElevation
  found: This class (feeding the chart's header gain/loss when trail==null, i.e. recording) gates elevation anchoring only via `parseGpxElevation` (null/non-finite check) — a genuine `0.0` elevation passes through as real data by explicit design ("A genuine 0.0 is real data, never missing" — true for GPX files, where null and 0.0 are distinguishable, but not true for a geolocator-sourced 0.0, which cannot express "missing"). This class has no `altitudeAccuracy` input at all (Wpt has no accuracy field), so it cannot apply the same fix even in principle.
  implication: The confirmed, in-scope fault (and the only one fixable without touching off-limits files/values) is in the stats path. The chart path may share a structurally similar ambiguity for the same seed value, but fixing it is out of scope for this session (no prime-suspect pointer to it, and it is reported as visually fine).

- timestamp: 2026-08-09T00:00:00Z
  checked: app/test/provider/navigation_stats_provider_test.dart (full file)
  found: The `_pos()` test helper (lines 34-52) hardcodes `altitudeAccuracy: 5.0` for every position built by every existing test. No test ever constructs a position with `altitudeAccuracy: 0`.
  implication: Confirms this is an untested hole, not a previously-covered-then-regressed case. Safe to add a new test without touching any existing one.

## Eliminated

## Open follow-up — REVISIT IF A PROBLEM SHOWS UP ON A HIKE

Committed on 2026-08-09 without on-device confirmation, by the user's decision: they will confirm on their next real hike. Session deliberately left open rather than archived to `resolved/`.

If the elevation gain still misbehaves in the field, start from these three, in order — each is a judgement call made from source reading, not from a device:

1. **`hasUsableAltitude`'s both-zero sentinel.** If a device reports a genuine altitude of exactly 0.0 with no vertical accuracy, that fix is skipped for elevation (harmless — anchors on the next one). Conversely, if tracelet ever emits a REAL fix as `altitude: 0, altitudeAccuracy: 0` while sitting at a real altitude, the reference would keep deferring and gain would stay at 0. Symptom to look for: elevation gain stuck at 0 for a whole recording. Fix direction: have `TraceletPositionSource` mark its seed fix explicitly (identity or a wrapper) instead of inferring "no reading" from the value pair.
2. **The seed is still recorded as a breadcrumb point** (with `ele: null`). Its lat/lon is real so it is kept deliberately, but if the saved track ever starts with a spurious first point, dropping the seed from the breadcrumb entirely is the alternative.
3. **The chart's new empty state.** A GPX with no elevation anywhere now renders `ElevationProfile`'s empty state instead of a flat 0 line. If any imported/older trail shows an unexpectedly empty elevation chart, that is this change, and the flat line can be restored — but restore it in the CHART only, never by reintroducing `?? 0` into the metrics path.

Also unverified in the field: the preceding `elevation-profile-recording` session's live chart (user confirmed it renders, but not over a long recording — watch for rebuild cost at high fix counts, since `_parseGpx` re-runs per fix over the whole breadcrumb).

## Resolution

- root_cause: NavigationStatsNotifier.onPosition (app/lib/provider/navigation_stats_provider.dart) anchors its elevation reference (`_lastAltitude`) on ANY first fix unconditionally, with no check for whether that fix carries a real altitude reading. TraceletPositionSource.seedPositionFrom (app/lib/services/tracelet_position_source.dart) builds a synthetic "seed" position with `altitude: 0, altitudeAccuracy: 0` (the map marker's LocationMarkerPosition never tracks altitude) and pushes it onto the same stream that feeds the stats notifier, immediately on start() — before tracelet's real GPS engine produces its first fix. Whenever a seed was available (map marker already resolved before opening the recording screen — the source of the reported "randomness"), this zero-altitude placeholder becomes the first fix onPosition sees, anchoring the reference at 0.0. The next real fix reports the device's actual absolute altitude, which then reads as a single-step "gain" of that full magnitude.
- fix: Gate the altitude anchor/accumulation branch in NavigationStatsNotifier.onPosition on `pos.altitudeAccuracy > 0` — geolocator's/tracelet's own documented sentinel for "no real altitude reading" (both altitude and altitudeAccuracy read exactly 0.0 when altitude is unavailable). A fix with `altitudeAccuracy <= 0` is skipped for elevation purposes entirely (neither anchors nor accumulates); the reference anchors on the first subsequent fix that does carry a real reading. Does not touch `_kAltitudeNoiseFloorMeters`, distance/XY logic, or `thresholdXY_m` (all explicitly off-limits). Also directly satisfies the user's own hypothesis (early real GPS fixes before altitude lock share the same 0/0 sentinel).
- verification: Red/green regression test added to app/test/provider/navigation_stats_provider_test.dart. RED (pre-fix): both new tests failed — "a zero-altitude/zero-accuracy seed fix..." asserted elevationGainMeters==0 but got 500.0; "a fix with zero altitudeAccuracy never accumulates..." asserted 0 but got 200.0 — exactly reproducing the reported magnitude/mechanism. GREEN (post-fix): all 18 tests in the file pass, including the two new ones and all pre-existing anchor/pause/resume/stationary cases (no regression). Full `flutter test` suite (entire app/test/) passes with exit code 0, no failures. `flutter analyze` reports only 5 pre-existing, unrelated issues (confirmed via `git status` — no other files touched). Not yet verified on-device by the user (mobile app; user builds/installs per project convention) — awaiting human confirmation of the original real-world symptom.
- orchestrator_corrections: |
    The root cause above is correct and confirmed. Two problems with the fix as first applied were found during orchestrator verification and corrected.

    1. The gate `pos.altitudeAccuracy > 0` was UNSAFE and has been replaced.
       Android only exposes vertical accuracy from API 26+ (`Location.hasVerticalAccuracy`), and this app supports API 21+ (per CLAUDE.md). tracelet ships no Android sources in the pub cache, so it could not be confirmed that real fixes populate `altitude_accuracy` at all — and tracelet's own `Coords.altitudeAccuracy` DEFAULTS to 0.0 when the native map omits the key (tracelet-3.2.17/lib/src/models/location.dart:600,621). On any such device/plugin path every real fix has `altitudeAccuracy == 0`, so that gate would have silently disabled elevation tracking for the entire recording — a worse, harder-to-notice bug than the phantom gain it prevented. The debugger's second new test actively asserted this bad contract ("a fix with zero altitudeAccuracy never accumulates gain/loss even mid-session"), which would have locked it in.
       Replaced with a single named predicate `hasUsableAltitude(geo.Position)` in tracelet_position_source.dart (beside `seedPositionFrom`, whose sentinel it is the inverse of): `pos.altitude.isFinite && (pos.altitudeAccuracy > 0 || pos.altitude != 0)`. Only the BOTH-zero sentinel means "no reading". A genuine reading of exactly 0 m with no vertical accuracy is the sole false negative and is harmless — the reference anchors on the next fix.

    2. The same fabricated 0 also poisoned the BREADCRUMB, which the debugger scoped out but which is the more damaging half.
       navigation_screen.dart passed `altitude: pos.altitude` straight into `NavigationNotifier.onPosition`, which appends `Wpt(..., ele: altitude, ...)`. So the first breadcrumb point of any recording seeded from a resolved map marker carried `ele: 0`. The breadcrumb IS the saved trail's GPX, so every such saved recording would bake in a phantom climb of the device's full absolute altitude — persisted, exported data corruption, not just a live display glitch. Now passes `hasUsableAltitude(pos) ? pos.altitude : null`; `computeTrailMetrics` already skips waypoints with no usable `ele` ("so the first point that *does* carry elevation becomes the anchor instead of diffing against a fabricated 0" — conversion.dart:225-239), so a null flows through correctly with no metrics change needed.

    3. Consequential chart fix. `buildElevationTrackPoints` (elevation_profile.dart) plotted `wpt.ele ?? 0`, so a null-elevation point would have drawn a cliff from sea level to real altitude at the start of the live recording chart — newly visible because the preceding `elevation-profile-recording` session made that chart render during recording at all. It now skips points with no usable elevation via the existing `parseGpxElevation`, matching computeTrailMetrics. Side effect, deliberate and flagged to the user: a GPX with NO elevation anywhere now yields an empty chart (ElevationProfile's own empty state) instead of a fake flat line at 0. This was the third of the three disagreeing answers to "is this elevation usable?" that conversion.dart:88-96's own doc comment warns about.
- orchestrator_verification: |
    Empirical red/green re-confirmed independently, not taken on trust: temporarily replaced the gate with `if (true)` and re-ran the suite — exactly one test failed, the zero-altitude/zero-accuracy seed test. Restored and re-ran green.
    `dart format` applied. `flutter analyze lib test`: 5 issues, all pre-existing and unrelated (summit_log_card unused intl import, actor_entity + navigation_stats_provider unnecessary imports, two dangling library doc comments).
    `flutter test` (full suite): 1066 passed, 1 skipped, 0 failed.
    Constraint-clean: `_kAltitudeNoiseFloorMeters` untouched, no distance smoothing reintroduced, `thresholdXY_m` untouched, no `late final` in a Notifier build(), no `.valueOrNull`, no `Isolate.run`, no `flutter build` / `adb install`.
- tests_added:
  - app/test/services/tracelet_position_source_test.dart (new) — `hasUsableAltitude` truth table incl. the Android-<26 real-altitude-without-accuracy case and below-sea-level readings; plus `seedPositionFrom` output asserted unusable for elevation while its lat/lon/accuracy survive.
  - app/test/provider/navigation_stats_provider_test.dart — seed-then-real-fix regression (the reported symptom); real altitude with zero accuracy still accumulates (guards against the reverted over-strict gate); non-finite altitude skipped.
  - app/test/components/trail/elevation_profile_test.dart — a point with no usable elevation is skipped rather than plotted at 0; an all-null-elevation GPX yields no points.
- files_changed:
  - app/lib/provider/navigation_stats_provider.dart
  - app/lib/services/tracelet_position_source.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/components/trail/elevation_profile.dart
  - app/test/provider/navigation_stats_provider_test.dart
  - app/test/services/tracelet_position_source_test.dart (new)
  - app/test/components/trail/elevation_profile_test.dart
