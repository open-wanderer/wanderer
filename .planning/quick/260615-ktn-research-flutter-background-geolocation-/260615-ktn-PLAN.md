---
phase: quick-260615-ktn
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .planning/quick/260615-ktn-research-flutter-background-geolocation-/260615-ktn-FINDINGS.md
autonomous: true
requirements:
  - RESEARCH-BG-GEO-01
must_haves:
  truths:
    - "A findings document exists that a developer can read to decide how to implement background geolocation"
    - "The chosen approach (geolocator 14.x upgrade) is documented with exact steps, assumptions, and open questions"
    - "Platform-specific requirements (Android manifest, iOS plist, Xcode capability) are listed precisely"
    - "Known pitfalls and their mitigations are enumerated"
    - "All unverified assumptions are labelled so the implementer knows what to test first"
  artifacts:
    - path: ".planning/quick/260615-ktn-research-flutter-background-geolocation-/260615-ktn-FINDINGS.md"
      provides: "Implementation-ready spike document for background geolocation"
      contains: "## Decision"
  key_links: []
---

<objective>
Produce a structured implementation-spike findings document from the completed background geolocation research. The document distills option evaluation, the recommended path, exact file-level changes, open assumptions, and a suggested implementation order into a single reference the team can hand to an executor without re-reading the raw research notes.

Purpose: The RESEARCH.md captured raw findings with inline caveats. The FINDINGS.md translates those findings into a decision + ordered action plan ready for `/gsd-quick` or a future phase plan.

Output: `.planning/quick/260615-ktn-research-flutter-background-geolocation-/260615-ktn-FINDINGS.md`
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@/Users/christianbeutel/Documents/svelte/wanderer/.planning/STATE.md
@/Users/christianbeutel/Documents/svelte/wanderer/.planning/quick/260615-ktn-research-flutter-background-geolocation-/260615-ktn-RESEARCH.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Write implementation-spike findings document</name>
  <files>.planning/quick/260615-ktn-research-flutter-background-geolocation-/260615-ktn-FINDINGS.md</files>
  <action>
    Create `260615-ktn-FINDINGS.md` in the quick task directory. The document must cover the following sections in order:

    **## Decision**
    State the chosen approach plainly: upgrade `geolocator` from `^13.0.2` to `^14.0.0` and use `ForegroundNotificationConfig` (Android) + `AppleSettings(allowBackgroundLocationUpdates: true)` (iOS). Include a one-sentence rationale (free license, minimal diff, no new packages). Note the hard prerequisite: Flutter SDK must be >= 3.29.0 — this is the first thing an implementer must verify (`flutter --version`).

    **## Rejected Alternatives**
    Three-row table: Option, Why Rejected. Rows: `flutter_background_geolocation` ($500/app Android production license, over-engineered); `flutter_foreground_task + geolocator` (extra package, no benefit over Option A); `background_fetch` (15-minute minimum fire interval, unsuitable for real-time navigation).

    **## Flutter SDK Gate**
    Describe the fork: if `flutter --version` reports >= 3.29.0, bump `geolocator: ^14.0.0` in `app/pubspec.yaml`. If < 3.29.0, import `AndroidSettings` / `AppleSettings` directly from `geolocator_android` / `geolocator_apple` packages at current 13.x — document the direct import path as a fallback.

    **## Required File Changes**
    A table with three columns: File, Change, Notes. Rows:
    - `app/pubspec.yaml` — bump geolocator to ^14.0.0 (if SDK >= 3.29.0) — run `flutter pub upgrade geolocator` after edit
    - `app/android/app/src/main/AndroidManifest.xml` — add three permissions: `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_LOCATION` (Android 14+), `ACCESS_BACKGROUND_LOCATION` (Android 10+, needed for full lock-screen; Google Play requires justification) — note `ACCESS_BACKGROUND_LOCATION` triggers Play Store review; may omit for v1 if screen-on navigation is acceptable
    - `app/ios/Runner/Info.plist` — add `NSLocationAlwaysAndWhenInUseUsageDescription` string and `UIBackgroundModes` array with `location` value
    - `app/ios/Runner.xcodeproj` (Xcode only) — enable "Location Updates" under Background Modes capability; cannot be done by file edit; requires human action in Xcode
    - `app/lib/routes/navigation_screen.dart` — replace bare `Geolocator.getPositionStream()` with platform-aware `_buildLocationSettings()` helper; provide the full `_buildLocationSettings()` function signature and the updated `initState` call pattern from RESEARCH.md (do not duplicate the full code block; reference the integration pattern section of RESEARCH.md for the exact snippet)
    - `app/lib/util/navigation_launch_util.dart` — iOS "Always" permission upgrade: optional for v1; document the two-step Apple always-permission flow and defer until team decides whether lock-screen navigation without screen-on is required

    **## Pitfalls to Address Before Shipping**
    Numbered list, five items drawn directly from RESEARCH.md Pitfall 1-5:
    1. iOS stream suspended without `allowBackgroundLocationUpdates: true`
    2. `AndroidSettings` not cleanly exposed at geolocator 13.x — upgrade required
    3. Android Doze / OEM battery optimization kills foreground service — `enableWakeLock: true` + user instruction
    4. `showBackgroundLocationIndicator: true` required in `AppleSettings` to pass App Store review
    5. geolocator 14.0.0 requires Flutter >= 3.29.0 — verify SDK before bumping

    **## Open Assumptions (Must Verify During Implementation)**
    A table: ID, Assumption, Risk, How to Verify. Five rows from RESEARCH.md assumptions log (A1–A5). Mark each row with its assumption ID so the implementer can log which assumptions were confirmed.

    **## Suggested Implementation Order**
    Numbered steps the executor should follow:
    1. Run `flutter --version` — confirm SDK >= 3.29.0 (gates the upgrade path)
    2. Bump `geolocator` in pubspec.yaml per SDK gate result; run `flutter pub upgrade geolocator`
    3. Add Android manifest permissions (start without `ACCESS_BACKGROUND_LOCATION`; add only if full lock-screen is required after testing)
    4. Add iOS plist keys
    5. Open Xcode, enable Background Modes > Location Updates capability
    6. Implement `_buildLocationSettings()` in `navigation_screen.dart`
    7. Test on a real Android device with screen-off while stream is active — confirm position events continue
    8. Test on a real iOS device with screen-off — confirm events continue and blue pill indicator appears
    9. Verify `./gradlew mergeDebugManifest` shows correct foreground service type (addresses A3)
    10. Decide on iOS "Always" permission (addresses A4) — if needed, update `navigation_launch_util.dart`

    **## Sources**
    Copy the Sources section from RESEARCH.md verbatim (Primary / Secondary / Tertiary).

    Formatting rules: use markdown headers (##, ###), tables where specified, numbered lists for ordered steps, bullet lists for pitfalls. Do not include fenced code blocks with implementation code — reference the RESEARCH.md integration pattern section by name where code is needed. Keep the document under 200 lines.
  </action>
  <verify>
    Read the created FINDINGS.md and confirm all required sections are present: Decision, Rejected Alternatives, Flutter SDK Gate, Required File Changes, Pitfalls, Open Assumptions, Suggested Implementation Order, Sources.
  </verify>
  <done>
    FINDINGS.md exists at the quick task directory path, contains all eight sections, is under 200 lines, and can be read standalone to produce an implementation plan without referring back to RESEARCH.md for decision rationale.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Research doc → findings doc | Findings summarize RESEARCH.md; no external input, no trust concern |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-ktn-SC | Tampering | No package installs in this task | accept | Documentation-only task; no npm/pub install commands |
</threat_model>

<verification>
Confirm `.planning/quick/260615-ktn-research-flutter-background-geolocation-/260615-ktn-FINDINGS.md` exists and contains the `## Decision` section header.
</verification>

<success_criteria>
FINDINGS.md written with all eight required sections. An executor reading only FINDINGS.md has everything needed to implement background geolocation on the next coding session without re-reading raw research notes.
</success_criteria>

<output>
Create `.planning/quick/260615-ktn-research-flutter-background-geolocation-/260615-ktn-FINDINGS.md` when done. No SUMMARY.md required for this research-only quick task.
</output>
