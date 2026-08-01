---
status: diagnosed
phase: 34-dart-conversion-port
source: [34-VERIFICATION.md, 34-06-PLAN.md, 34-07-PLAN.md]
started: 2026-08-01T13:20:00Z
updated: 2026-08-01T14:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Offline flows complete with no dead end
steps: |
  Put the device in airplane mode, then:
  (a) record a short track and save it;
  (b) plan a 3-anchor route and tap Finish;
  (c) import a .gpx file from the share sheet.
expected: |
  (a) the save-options sheet does NOT appear; the app lands on
  trail_create_screen with the track drawn.
  (b) no error toast, no stall; the app lands on trail_create_screen with all
  three anchors intact. (Before this phase the planner showed an error toast
  and stranded you here — you could not finish a route offline at all.)
  (c) no options sheet; a trail is created with correct distance/elevation.
result: issue
reported: |
  a) pass
  b) partial. Why does the save options sheet appear in route planner when
  online? It has no purpose. The route is following roads and using valhalla
  elevation anyways
  c) partial. The save options sheet has the wrong title ("Save recordings")
  and needs more bottom padding because of the bottom navigation bar
severity: minor
note: |
  Offline behaviour (the D-15/D-16 fix under test) passed for all three flows.
  Both reported problems are about the ONLINE save-options sheet.

### 2. Anchor structure survives the online round trip
steps: |
  Online: plan a 3-anchor route, tap Finish, enable BOTH toggles (recalculate
  heights + follow roads), confirm. Then re-open the resulting trail in the
  route planner.
expected: |
  It still shows three anchors, not a collapsed start/end pair. This exercises
  the leg-boundary anchor re-pin, which the planner flagged as the phase's
  highest-risk new code — snapped legs must be forced back to the original
  anchor coordinates, since anchors are located by exact coordinate match.
result: skipped
reason: |
  User: "See my answer in Test 1. This is unnecessary." The test's setup step
  (enable both toggles on the route planner's save-options sheet) is exactly
  the sheet the user says should not appear for the route planner at all —
  the route already follows roads and already carries Valhalla elevation.
  If gap 1 is closed by removing the sheet from the planner flow, this
  round trip becomes unreachable and the leg-boundary anchor re-pin becomes
  dead code on that path. Diagnosis must confirm whether the re-pin is still
  reachable from any other flow before it is dropped or left untested.
severity_note: not a code defect — superseded by gap 1

### 3. Transcode round trip and published API docs
steps: |
  With the app pointed at a server running this change:
  (a) import a .kml and a .fit file while online;
  (b) open /docs/api and find the POST /api/v1/trail/convert entry;
  (c) on the web trail-edit page, import a file with the client-side picker.
expected: |
  (a) each produces a trail whose distance/elevation match what the same track
  reports after saving — proving the app measured the server-transcoded GPX
  itself rather than trusting a server-computed value (PORT-05).
  (b) the entry describes a GPX response, not a Trail.
  (c) still works unchanged — the web computes client-side and only uses the
  endpoint for transcoding (D-08).
result: issue
reported: |
  a) pass. However: the same trail uploaded on the web vs. the app produces
  different lengths:
  ~/Downloads/19440058502_ACTIVITY.fit
  On web: 10.51km | On app: 10.97km
  Elevation matches: 344m up, 351 down
  b) pass. Check if the json content type is still needed. If not remove it
  alongside the associated tests
  c) pass
severity: major
note: |
  All three sub-checks passed as written. The mismatch is a cross-client
  disagreement the test did not cover: same .fit file, same transcoded GPX,
  same elevation totals (344 up / 351 down) but 10.51 km on web vs 10.97 km
  on the app — a ~4.4% distance-only divergence.

## Summary

total: 3
passed: 0
issues: 2
pending: 0
skipped: 1
blocked: 0

## Adjacent Findings

Surfaced during diagnosis, outside the four gaps. Each needs its own decision.

- **Latent data bug (planner).** `_applySegment` (`app/lib/provider/route_anchor_provider.dart:301-320`)
  uses Freezed `copyWith` without clearing `elevationProfile`/`elevations`, so they survive a
  polyline replacement until the fire-and-forget `_resolveElevation` lands. Because
  `buildFinalPlannedGpx` *prefers* `elevationProfile` over `polyline`, tapping Finish inside that
  window saves the leg's **pre-edit geometry**. `recalcHeights` does not fix it (it refetches over
  the same stale profile). Deserves its own debug session.
- **Sibling sheets carry the same bottom-inset defect** as gap 2:
  `app/lib/components/trail/missing_coverage_sheet.dart:153` and
  `app/lib/components/base/wanderer_icon_picker.dart:166`.
- **Mirror-image legacy tolerance (client side).** `app/lib/util/trail_import_util.dart:220-230`
  still tolerates a legacy JSON *response* Map (`data['expand']['gpx_data']`), pinned by
  `app/test/util/trail_import_util_test.dart:231`. Kept deliberately (T-34-37, D-05, "self-hosters
  control their own server/app pairing") as new-app/old-server skew tolerance — but since the
  endpoint never shipped upstream, no such old server exists. Same cleanup family as gap 4,
  separate decision. The accepted-and-ignored `name` field still advertised in `requestBody`
  belongs to the same family.

## Working Tree Note

At diagnosis time the tree carried uncommitted hand-edits by the user (mtimes 13:58–14:55,
all predating the 15:01:12 UAT commit and the agent spawn) to:
`app/lib/components/navigation/track_save_options_sheet.dart`,
`app/lib/routes/trail_source_select_screen.dart`,
`app/lib/util/track_save_options_util.dart`,
`web/src/lib/util/gpx_util.ts`,
`web/src/routes/api/v1/trail/convert/+server.ts`.

These are a partial workaround for gap 2 (`title: "Adjust track"` + hardcoded
`kBottomNavigationBarHeight + 48`). Gap 2's diagnosis rejects the padding half of that shape.
The `+server.ts` edit also introduced a stray backtick that would ship to `/docs/api`.
Reconcile before applying fixes.

## Gaps

- truth: "Online route planner Finish should not offer save options that cannot change the result"
  status: failed
  reason: "User reported: b) partial. Why does the save options sheet appear in route planner when online? It has no purpose. The route is following roads and using valhalla elevation anyways"
  severity: minor
  test: 1
  root_cause: "resolveTrackSaveOptions (app/lib/util/track_save_options_util.dart:29) is a source-blind gate: it takes only (WidgetRef, BuildContext), has no per-source parameter, and shows the sheet unconditionally whenever onlineStatusProvider is true. Plan 34-06 routed all three capture sources through it to get D-15's single-code-path offline handling, which handed the route planner a sheet written for the recording flow."
  user_claim_verdict: |
    PARTIALLY correct.
    - "recalculate heights" IS inert, provably: _resolveElevation stores
      buildNavShape(polyline) as elevationProfile; refetchAllHeights then
      re-POSTs that identical shape to the identical deterministic DEM
      endpoint. Cannot change output in any planner state.
    - "follow roads" is NOT inert. A planner route is not always
      Valhalla-routed. Three sources of straight-line legs:
      (1) user-facing auto-routing OFF switch in the planner Settings tab
          (components/route_planner/settings_tab.dart:40-46);
      (2) SegmentState.blocked while fully ONLINE — _resolveSegment marks
          blocked on non-2xx / missing trip.legs[0].shape / out-of-range
          coords, and badResponse deliberately never flips
          onlineStatusProvider, so an unroutable anchor pair yields a
          straight leg with no offline transition at all;
      (3) offline-planned then saved online.
      For fully-routed legs it is still redundant AND lossy: it re-vertexes
      geometry from a decimation of an already-decimated shape and discards
      good elevations for a refetch.
  artifacts:
    - path: "app/lib/util/track_save_options_util.dart"
      issue: "source-blind gate; online branch unconditional (line 29-43)"
    - path: "app/lib/routes/route_planner_screen.dart"
      issue: "line 529 calls the gate unconditionally; line 523-527 edit mode already skips it — per-flow precedent exists in the same method"
    - path: "app/lib/util/route_planner_handoff_util.dart"
      issue: "lines 459-499 snap + leg-boundary re-pin block goes dead if followRoads is hardcoded off"
    - path: "app/lib/provider/route_anchor_provider.dart"
      issue: "SegmentState transitions that make followRoads occasionally meaningful"
  missing:
    - "Drop the 'recalculate heights' toggle for the planner flow (provably inert)"
    - "Show the sheet from _onFinish only when state.segments.any((s) => s.state != SegmentState.routed), with only the follow-roads toggle"
    - "If instead removing the sheet outright: delete finishPlanning's followRoads/recalcHeights params and buildFinalPlannedGpx's whole snap block rather than leaving unreachable code with passing tests"
  debug_session: ".planning/debug/planner-save-options-sheet-pointless.md"
  supersedes_test: 2
  anchor_repin_reachability: |
    BECOMES DEAD CODE if the sheet is removed outright. The sole production
    snapCosting assignment is route_planner_handoff_util.dart:616, fed only
    by this sheet via finishPlanning (only caller: route_planner_screen.dart
    :537). The re-pin block sits inside `if (costing != null)`. No alternative
    entry point reaches it — import-then-edit uses snapShapeToRoads on a flat
    track then opens the planner in EDIT mode (snapCosting: null); recording
    uses snapShapeToRoads on a flat breadcrumb with no per-leg trkseg. It
    would survive only as unit-test-referenced code
    (app/test/util/route_planner_handoff_util_test.dart:763, 813).
  substitute_for_test_2: |
    The round-trip invariant the re-pin protects is still live without it:
    plan a 3-anchor route online, Finish (no sheet), save, reopen in the
    planner. This exercises anchorsFromTrack/segmentPolylinesFromTrack over
    buildFinalPlannedGpx's trkseg-per-leg output — the actual risk. Only the
    snap-specific drift path disappears.

- truth: "The save-options sheet is correctly titled and laid out for the flow that opened it"
  status: failed
  reason: "User reported: c) partial. The save options sheet has the wrong title (\"Save recordings\") and needs more bottom padding because of the bottom navigation bar"
  severity: cosmetic
  test: 1
  root_cause: |
    TITLE: showTrackSaveOptionsSheet took only (BuildContext) and rendered
    l10n.save_recording_options unconditionally; the shared gate
    resolveTrackSaveOptions is called identically by all three capture
    sources with no source discriminator, so the recording title leaks into
    import and planner. (Actual en string is "Save recording"; the report
    paraphrased it.)
    PADDING: the sheet's only bottom inset was 24 + viewInsets.bottom.
    viewInsets is the KEYBOARD inset and is 0 here. The sheet never read
    MediaQuery.padding.bottom, and showModalBottomSheet's useSafeArea
    defaults to false — so padding.bottom is in scope and simply unread.
  obstruction_mechanism: |
    showModalBottomSheet also defaults to useRootNavigator: false. go_router's
    ShellRoute builds a real nested Navigator handed to WandererLayout as its
    Scaffold body, and WandererLayout sets extendBody: true with a
    BottomAppBar(kBottomNavigationBarHeight) plus a centerDocked FAB. A sheet
    opened from a shell child therefore extends to the physical screen bottom
    and the Scaffold paints the BottomAppBar (56px) + FAB overhang (~28px)
    on top of it, covering the Save button.
  host_matrix: |
    | Flow                  | Route            | Navigator | Title wrong | Obscured by            |
    |-----------------------|------------------|-----------|-------------|------------------------|
    | Recording save        | /record          | root      | no          | nothing (no nav bar)   |
    | Planner Finish        | /route-planner   | root      | yes         | nothing (see gap 1)    |
    | Import, in-app picker | /trail/create    | SHELL     | yes         | BottomAppBar + FAB     |
    | Import, OS share      | navigatorKey ctx | root      | yes         | system gesture bar     |
  artifacts:
    - path: "app/lib/components/navigation/track_save_options_sheet.dart"
      issue: "hardcoded title (line 74 @HEAD); bottom inset reads only viewInsets (line 48-52 @HEAD)"
    - path: "app/lib/util/track_save_options_util.dart"
      issue: "shared gate with no source parameter; single funnel for all three flows"
    - path: "app/lib/components/base/wanderer_layout.dart"
      issue: "extendBody: true + BottomAppBar + docked FAB — the obstruction (lines 52-71)"
  missing:
    - "PADDING: resolve the inset from the presentation context INSIDE the sheet — add MediaQuery.of(context).padding.bottom to the existing viewInsets.bottom term (or pass useSafeArea: true), plus a small allowance for the docked-FAB overhang applied only when that padding is non-zero. Self-adjusts across all four hosts; no caller changes."
    - "REJECT the current working-tree workaround's shape: a hardcoded kBottomNavigationBarHeight + 48 inside the shared gate adds ~104px of dead space to /record, /route-planner and the share-intent path, none of which have a nav bar."
    - "TITLE: give showTrackSaveOptionsSheet a source enum (recording/import/planner) rather than a raw String?, and let the gate own the l10n lookup. The three call sites already know their own source."
    - "i18n: new keys go in app/lib/i18n/app_en.arb only (it is the template; the other 13 locales lack this key family and fall back to English), then flutter gen-l10n and commit regenerated app_localizations*.dart. The string must not remain a Dart literal — the workaround's `// TODO: make this l10n` is the only non-localized string in the widget."
    - "Fix the stray backtick typo introduced in the working tree at web/src/routes/api/v1/trail/convert/+server.ts:49 (`...clients compute trail metrics themselves.`.`) before it ships to /docs/api."
  existing_convention: |
    SafeArea wrapper — app/lib/routes/library_screen.dart:119 (a shell child,
    the direct precedent) and app/lib/components/trail/map_app_sheet.dart:23.
    Counter-example NOT to copy: travel_profile_sheet.dart:34 hardcodes
    24 + kBottomNavigationBarHeight + 56 — it is the sheet this one was
    "styled after", but it is opened only from a shell screen.
  sibling_sheets_with_same_latent_defect:
    - "app/lib/components/trail/missing_coverage_sheet.dart:153"
    - "app/lib/components/base/wanderer_icon_picker.dart:166"
  debug_session: ".planning/debug/save-options-sheet-title-and-padding.md"

- truth: "The same track file yields the same distance whether imported on web or in the app"
  status: failed
  reason: "User reported: the same trail uploaded on the web vs. the app produces different lengths for ~/Downloads/19440058502_ACTIVITY.fit — web 10.51km, app 10.97km. Elevation matches exactly (344m up, 351m down)."
  severity: major
  test: 3
  evidence:
    file: "~/Downloads/19440058502_ACTIVITY.fit"
    web_distance_km: 10.51
    app_distance_km: 10.97
    elevation_gain_m: 344
    elevation_loss_m: 351
    note: "Elevation agreeing while distance diverges points at the distance formula/filtering, not at the transcode or the point set."
  root_cause: |
    DISPLAY BUG, not a computation bug. The app's elevation-chart stats
    header renders the chart's RAW cumulative-distance plotting axis as if
    it were the trail's length. elevation_profile.dart:143 does
    formatDistance(maxDist), where maxDist = _points.last.distanceM (line 80)
    and _points comes from buildElevationTrackPoints, whose accumulator is
    unfiltered. Every other distance surface on BOTH clients uses the
    5m-threshold smoothed accumulator GpxMetricsComputation
    .totalDistanceSmoothed. The same header's up/down arrows read
    trail.elevationGain/Loss, which ARE smoothed — hence one stats row
    showing raw distance beside smoothed elevation, and hence elevation
    matching exactly while distance did not.
  which_side_is_wrong: |
    The APP, and only on that one widget. Nothing is computed or persisted
    wrongly. The Dart port is bit-for-bit parity with the TS original.
  measurements: |
    Measured on the actual UAT file:
      web  trail.distance                     = 10553.4607247122 m
      Dart trailFromGpx(...).distance         = 10553.4607247166 m
      -> agree to 5e-9 m over 11 km; form_data_util.dart:25 persists this.
      web  raw                                = 10971.376198346872 m
      Dart raw                                = 10971.376198347361 m
      Dart buildElevationTrackPoints(...).last.distanceM
                                              = 10971.376198347361 m
      -> formatDistance -> "10.97 km", plus 344.2/351.2 -> "344 m"/"351 m".
      Exact match to the user's report.
    Byte-identical GPX confirmed: the convert endpoint calls the SAME
    fromFile() the web page calls client-side; both parse 2569 points,
    1 <trk>, 1 <trkseg>, identical first coords to the last decimal.
    Identical earth model: TS haversineDistance R=6371km; Dart
    haversineMeters delegates to geobase SphericalGreatCircle (6371000.0 m).
    Mean point spacing 10971/2568 = 4.27 m against a 5 m gate, so raw
    exceeds smoothed by 3.96% — the reported ~4.4% gap.
  artifacts:
    - path: "app/lib/components/trail/elevation_profile.dart"
      issue: "line 143 (and line 108, the scrub stat pt.distanceM) display raw plotting coordinates as trail distance; maxDist at line 80"
    - path: "app/lib/routes/trail_create_screen.dart"
      issue: "line 746 renders that widget — the screen the import flow lands on, where the user read 10.97 km"
  missing:
    - "Feed the header's ruler stat from the same source as its elevation stats: widget.trail?.distance, falling back to computeTrailMetrics(widget.gpx).distance when there is no trail, mirroring the existing gain/loss branch at lines 125-133."
    - "Do NOT change the accumulator behind _points — the comment block at elevation_profile.dart:661-686 documents with measurements why the x-axis must stay raw, and suggests scaling the axis by smoothed/raw if the axis maximum must equal the reported distance."
    - "Decide the scrub stat at line 108 in the same change — same class of mismatch."
  open_question: |
    User reported web as 10.51 km but every web display path resolves to
    10553.46 m -> "10.55 km" (42 m / 0.4% residual). Most likely a
    transcription slip in the UAT note; does not affect the diagnosis since
    the ~4.4% gap is fully accounted for. Worth confirming with the user.
  debug_session: ".planning/debug/web-app-distance-mismatch.md"

- truth: "The convert endpoint carries no content type it no longer serves"
  status: failed
  reason: "User reported: Check if the json content type is still needed. If not remove it alongside the associated tests"
  severity: minor
  test: 3
  root_cause: |
    Two premises in the original framing were wrong and are corrected here:
    (1) the handler is NOT Go — it is a SvelteKit route at
        web/src/routes/api/v1/trail/convert/+server.ts;
    (2) there is NO JSON RESPONSE branch — Phase 34-07 (df61d581) already
        deleted it; json() was dropped from the imports and the single
        success path is hardcoded to application/gpx+xml, with the D-06
        regression guard already asserting the 200 body does not parse
        as JSON.
    What remains is an application/json REQUEST content-type branch at
    +server.ts:82/99-102. It has never had a caller — not "no longer used"
    but never used. Introduced in ca063023 (2026-07-19) in the same commit
    that first made the app call the endpoint, whose only entry point was
    convertGpxToTrail(WidgetRef, FormData) — multipart by construction.
    Speculative generality from day one.
  live_consumers: "none"
  consumer_audit: |
    Flutter: trail_import_util.dart:211-217 transcodeToGpx() sends
      FormData.fromMap({'file': MultipartFile...}) -> multipart. A PORT-03
      gate test (app/test/util/trail_import_util_test.dart:414) structurally
      pins the app to exactly one trail/convert occurrence, so no hidden
      second call site can exist.
    Web: zero callers. grep for trail/convert across web/src/ excluding the
      route's own directory exits 1. The trail-edit page transcodes
      client-side (+page.svelte:480,485 -> fromFile + gpx2trail). D-08 holds.
    Go / scripts / migrations / compose: zero.
  breaking_change_risk: |
    NONE, for two independent reasons.
    (1) The entire endpoint does not exist on origin/main
        (open-wanderer/wanderer, v0.20.0). It lives only on the unmerged
        feature/app branch and has never shipped, so there are no
        third-party consumers and there cannot be.
    (2) Even ignoring that: the generated OpenAPI requestBody.content
        declares ONLY multipart/form-data. application/json was never a
        declared request media type — it appears solely in the
        human-readable description prose, which is what was seen on
        /docs/api. Both spec files are gitignored build artifacts.
  associated_tests: |
    All in web/src/routes/api/v1/trail/convert/convert.test.ts (16 tests,
    currently green). Exactly 3 touch the JSON branch:
      :100 content validation - 400s on a non-GPX body via the JSON branch
      :114 JSON branch - accepts the `gpx` key
      :128 JSON branch - accepts the `gpxData` key
  artifacts:
    - path: "web/src/routes/api/v1/trail/convert/+server.ts"
      issue: "lines 99-102 application/json request branch has never had a caller; description prose at 47-49 still advertises it"
    - path: "web/src/routes/api/v1/trail/convert/convert.test.ts"
      issue: "3 tests cover the dead branch; the one at :100 is a CR-04 security guard that must be preserved, not deleted"
  missing:
    - "Delete +server.ts:99-102 (4 lines). application/json requests then fall into the raw-text else, assertParsableGpx rejects, -> 400 'Invalid GPX content' — already proven by the passing test at :83. One edge case improves: Content-Type: application/json carrying a bare GPX string currently 500s (request.json() SyntaxError); after removal it 200s."
    - "Reword the swagger description at +server.ts:47-49 to describe multipart + raw-text only. requestBody needs no change. Regenerate via npm run openapi:generate (web) + npm run sync-openapi (docs); nothing to commit, both artifacts are gitignored."
    - "Drop the 2-test 'JSON branch' suite, but REWRITE (do not delete) the test at :100 — it is a CR-04 security regression guard proving the endpoint never echoes an attacker-controlled body. Send the same non-GPX payload with an application/json header through the raw-text path so the guard survives."
  scope_note: |
    Phase 34 did not overlook this. 34-07-PLAN.md D-06 states verbatim: "the
    three input branches (multipart, JSON, raw text) and the empty-body 400
    guard are unchanged." It was a deliberate minimal-diff choice — change
    the output, leave the inputs alone. This is scoped-out follow-up, not a
    Phase 34 defect.
  debug_session: ".planning/debug/convert-endpoint-json-content-type.md"
