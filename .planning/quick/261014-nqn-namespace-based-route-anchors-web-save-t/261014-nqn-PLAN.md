---
phase: quick-261014-nqn
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - /Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web/src/lib/util/gpx_segment_util.ts
  - /Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web/src/routes/trail/edit/[id]/+page.svelte
  - /Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web/src/lib/util/gpx_util.test.ts
autonomous: true
requirements: [ANCHOR-01, ANCHOR-02, ANCHOR-03, ANCHOR-04]

estimate:
  tokens: 40000
  raw_tokens: 40000
  tasks: 2
  confidence: low

must_haves:
  truths:
    - "ANCHOR-01 (D-01, D-02): Saving a drawn multi-leg route in the web route editor uploads a GPX in which consecutive legs that meet exactly are ONE trkseg, each junction point carries <wanderer:anchor/>, and the root declares xmlns:wanderer=\"https://wanderer.to/xmlns/gpx/1\""
    - "ANCHOR-02 (D-01): Saving never mutates valhallaStore.route. After a save the editor still holds one trkseg per leg, and its anchors and undo/redo stack are unchanged, so editing can continue and undo still works"
    - "ANCHOR-03 (D-01): Legs whose touching endpoints are not numerically equal stay separate segments in the stored file (no tolerance-based or unconditional joining). Reloading the saved trail in the editor (existing split on load) restores the same legs and anchors, with the marks stripped"
    - "ANCHOR-03: An empty route (nothing drawn) still serializes at save without throwing"
    - "ANCHOR-04 (D-03, D-04): The export-time join in trail2gpx and the split on both editor load paths are kept unchanged. #1370 leaves no dead code; nothing is removed"
  artifacts:
    - path: "web/src/lib/util/gpx_segment_util.ts"
      provides: "routeToStoredGpx(gpx: GPX): string, which serializes a joined deep copy"
      contains: "export function routeToStoredGpx"
    - path: "web/src/routes/trail/edit/[id]/+page.svelte"
      provides: "Save path serializes through routeToStoredGpx"
      contains: "routeToStoredGpx(valhallaStore.route)"
    - path: "web/src/lib/util/gpx_util.test.ts"
      provides: "vitest coverage for the save helper"
      contains: "describe(\"routeToStoredGpx\""
  key_links:
    - from: "web/src/routes/trail/edit/[id]/+page.svelte (felte onSubmit, about line 311)"
      to: "routeToStoredGpx in web/src/lib/util/gpx_segment_util.ts"
      via: "form.expand!.gpx_data = routeToStoredGpx(valhallaStore.route), then gpxFile Blob, then trails_create/trails_update"
      pattern: "routeToStoredGpx\\(valhallaStore\\.route\\)"
    - from: "routeToStoredGpx"
      to: "joinContiguousTrackSegments (existing equality rule)"
      via: "GPX.parse(gpx.toString()) clone, then join, then toString"
      pattern: "joinContiguousTrackSegments\\("
    - from: "web/src/routes/trail/edit/[id]/+page.svelte onMount + handleFileSelection"
      to: "splitTrackSegmentsAtAnchors"
      via: "existing split before setRoute/initRouteAnchors (kept)"
      pattern: "splitTrackSegmentsAtAnchors\\("
---

<objective>
Store a route drawn in the web route editor as ONE track segment per track, with `<wanderer:anchor/>` marks at leg boundaries, at SAVE time instead of only on export. This is a follow-up PR to #1370.

Purpose: Since #1370, export joins the legs and the editor splits them again on load. Stored files are still written one trkseg per leg, though, so any consumer that reads the stored file directly (the federation copy, the raw PocketBase file, other clients) sees the broken multi-segment form from #1354. Writing the joined form at save time puts the anchor-marked file at the source.

Output: the helper `routeToStoredGpx` in gpx_segment_util.ts, the save path wired through it, vitest coverage, and one headline-only commit on branch `fix/route-anchor-marks-on-save`.

WORKING DIRECTORY (critical): all code lives in the git worktree `/Users/christianbeutel/Documents/svelte/wanderer-route-anchors` (branch `fix/route-anchor-marks-on-save`, cut from origin/dev, which already contains #1370 = ae31bb26e). All file paths in tasks below are relative to that worktree root. node_modules are installed in `web/` there. Do NOT edit the main checkout's `web/` and do NOT create `.planning` files in the worktree. The SUMMARY goes to the main checkout's quick dir (see output).

Locked decisions (from the orchestrator, numbered here for traceability):
- D-01: At save, serialize a joined deep COPY of `valhallaStore.route` and never mutate the store. Reuse `joinContiguousTrackSegments` with its existing exact numeric equality rule. No tolerance-based or unconditional joining. Mismatched legs stay separate segments, and since segment boundaries are still read as anchors on reload, nothing is lost.
- D-02: Extract a small helper `routeToStoredGpx(gpx: GPX): string` in gpx_segment_util.ts, shared by the save path and the tests. Match the file style: 4-space indent, semicolons, minimal comments.
- D-03: Keep working anything that reads `form.expand.gpx_data` or the saved string after save. Planner's audit result: nothing in the editor needs per-leg segments from that string. It is only used to build `gpxFile`. `trails_create`/`trails_update` return the server model, and `setFields` replaces the form. The only `GPX.parse` of `gpx_data` in the page is inside `onMount`. The other readers (handleMapInit, removeAnchor, the button labels) only check whether it is set. The map draws `valhallaStore.route` through `updateTrailOnMap`. Elsewhere: `trail2gpx` export, the view map (`GPX.parse(...).toGeoJSON()`), the Go backend (`db/util/geometry.go`, `polyline.go` and `importer.go` iterate every segment) and trail duplication (the edit page splits on load) all handle a single marked segment.
- D-04: Review #1370. Result: nothing is obsolete. The export-time join in `trail2gpx` STAYS, because legacy stored trails and trails saved by app versions without the mirror are still multi-segment, and that join is what fixes #1354 for them. On an already-joined file it is a no-op that keeps the parsed `xmlns:wanderer`. The split on both load paths STAYS and is now essential. The #1370 vitest and Playwright tests stay. The only #1370 text that changes is the header comment in gpx_segment_util.ts, which says marks are written "for export". It is reworded to cover storage too. No code removals.
- D-05: Add vitest coverage in `web/src/lib/util/gpx_util.test.ts`. Verify with `npx vitest run src/lib/util/gpx_util.test.ts` and `npm run check` (no NEW svelte-check errors against a baseline).
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@/Users/christianbeutel/Documents/svelte/wanderer/.planning/STATE.md
@/Users/christianbeutel/Documents/svelte/wanderer/CLAUDE.md
@/Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web/src/lib/util/gpx_segment_util.ts
@/Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web/src/lib/util/gpx_util.test.ts

<interfaces>
From web/src/lib/util/gpx_segment_util.ts (current, as of ae31bb26e):
- `import type GPX from "$lib/models/gpx/gpx";` (type-only today; must become a value import for GPX.parse)
- `export const routeAnchorNamespace = "https://wanderer.to/xmlns/gpx/1";`
- `export function joinContiguousTrackSegments(gpx: GPX)`: mutates in place. It merges segment i+1 into segment i when neither has segment extensions and the touching lat/lon are Number-equal, drops the duplicate first point, marks the junction point's extensions with `"wanderer:anchor": ""`, and sets `gpx.$["xmlns:wanderer"]` only if something was marked.
- `export function splitTrackSegmentsAtAnchors(gpx: GPX)`: mutates in place, the inverse. It ignores marks on segment ends, strips marks, and deletes `gpx.$["xmlns:wanderer"]`.

From web/src/lib/models/gpx/gpx.ts:
- `static parse(gpxString: string): GPX` throws on a missing root, and the constructor computes `features`.
- `toString(options?)` builds from a shallow `new GPX(this)`. Calling it on the store route is what the save path already does today, so it adds no new mutation.

From web/src/routes/trail/edit/[id]/+page.svelte:
- line 68: `import { splitTrackSegmentsAtAnchors } from "$lib/util/gpx_segment_util";`
- about line 311 (inside the felte onSubmit try block): the line that assigns `form.expand!.gpx_data` from a direct `.toString()` of the store route, followed by the `gpxFile = new Blob([form.expand!.gpx_data], { type: "text/xml" })` condition block.
- lines 419 and 540: the existing `splitTrackSegmentsAtAnchors(...)` calls before `setRoute` / `initRouteAnchors` (KEEP).

From web/src/lib/stores/valhalla_store.svelte.ts:
- `route: GPX = $state(new GPX({ trk: [emtpyTrack] }))`. The in-memory model is one trkseg per leg (MUST NOT change). Empty route = `new GPX({ trk: [new Track({ trkseg: [] })] })`.

Test helpers already in gpx_util.test.ts: `pt(lat, lon)`, `segment(points, extensions?)`, `coordinates(gpx)`, `anchors(gpx)` (returns per-track arrays of marked point indices), the `multiLegGpx` fixture (3 legs: 47.0 to 47.2, 47.2 to 47.3, 47.3 to 47.0), and imports of `routeAnchorNamespace`, `joinContiguousTrackSegments` and `splitTrackSegmentsAtAnchors`.

Planner probe result (run in vitest, probe deleted): `GPX.parse(new GPX({ trk: [new Track({ trkseg: [] })] }).toString())` does NOT throw and yields `trk === undefined`. The same holds for parsing the direct serialization, so the clone loses nothing on reload. Joining a parsed copy of constructor-built legs emits `xmlns:wanderer` on the root and `<extensions><wanderer:anchor/></extensions>` on the junction point. Browser serialization of the same join+toString sequence is already covered by the #1370 Playwright test `restores route legs from an exported single segment` in web/tests/playwright/e2e/gpx/parser.spec.ts.
</interfaces>
</context>

<tasks>

<task type="tracer" tdd="true">
  <name>Task 1: End-to-end "save stores one marked segment": helper, save wiring, round-trip test</name>
  <files>web/src/lib/util/gpx_segment_util.ts, web/src/routes/trail/edit/[id]/+page.svelte, web/src/lib/util/gpx_util.test.ts</files>
  <read_first>
    - web/src/lib/util/gpx_segment_util.ts (whole file, 101 lines)
    - web/src/lib/util/gpx_util.test.ts lines 1-35 (imports, helpers, multiLegGpx) and 150-230 (anchors helper, "route anchors" describe)
    - web/src/routes/trail/edit/[id]/+page.svelte lines 60-70 (imports) and 305-335 (save block)
  </read_first>
  <behavior>
    - Test A: `routeToStoredGpx(GPX.parse(multiLegGpx))` returns a string that parses to exactly 1 trk with exactly 1 trkseg. Its coordinates are `[[[47,11],[47.1,11.1],[47.2,11.2],[47.3,11.3],[47.4,11.4],[47,11]]]`, `anchors(...)` is `[[2, 3]]`, and the string contains `xmlns:wanderer="` followed by `routeAnchorNamespace` (assert on the raw string AND on the parsed `$["xmlns:wanderer"]`).
    - Test B: parse that string, run `splitTrackSegmentsAtAnchors` on it, and `coordinates(...)` equals `coordinates(GPX.parse(multiLegGpx))` (3 legs, each boundary point present as the last point of leg i and the first of leg i+1). `anchors(...)` is `[[]]` and `$["xmlns:wanderer"]` is undefined.
  </behavior>
  <action>
    Step 0 (baseline, BEFORE any edit): run `cd /Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web && npm run check 2>&1 | tail -3` and record the final "svelte-check found N errors and M warnings" line verbatim. Task 2 compares against it, and it goes into the SUMMARY.

    RED: in web/src/lib/util/gpx_util.test.ts, add `routeToStoredGpx` to the existing import from "./gpx_segment_util". Add a new `describe("routeToStoredGpx", ...)` block, placed after the "route anchors" describe and before the "trail2gpx" describe, with Tests A and B from the behavior block. Reuse the existing helpers (`coordinates`, `anchors`, `multiLegGpx`) and the 4-space style. Run vitest and confirm it fails because the export is missing.

    GREEN, per D-01 and D-02: in web/src/lib/util/gpx_segment_util.ts, change the type-only GPX import to a value import (`import GPX from "$lib/models/gpx/gpx";`), which is needed for `GPX.parse`. There is no circular import, because gpx.ts does not import this module. Add `export function routeToStoredGpx(gpx: GPX): string`. It deep-clones via `GPX.parse(gpx.toString())`, calls `joinContiguousTrackSegments` on the clone and returns `clone.toString()`. Clone by parsing the serialized string, NOT structuredClone or JSON: GPX, Track, TrackSegment and GPXWaypoint are classes, and those clones lose prototypes and methods. Never touch the argument beyond the `toString()` call that the save path already makes today. Keep the existing exact equality rule: no tolerance, no unconditional joining, no changes to joinContiguousTrackSegments or splitTrackSegmentsAtAnchors. No inline comment is needed beyond what the header comment covers (see Task 2).

    Wire the save path (D-01): in web/src/routes/trail/edit/[id]/+page.svelte, extend the line-68 import to `import { routeToStoredGpx, splitTrackSegmentsAtAnchors } from "$lib/util/gpx_segment_util";`. At the save site (about line 311), replace the direct `.toString()` serialization of the store route with `form.expand!.gpx_data = routeToStoredGpx(valhallaStore.route);`. Leave the rest of the save block untouched: the gpxFile Blob condition, the lat/lon fallback that reads `valhallaStore.route` directly, and the trails_create/trails_update calls. Per D-03 they need no change. Keep both existing `splitTrackSegmentsAtAnchors(...)` calls (lines ~419 and ~540) exactly as they are (D-04).

    Do not commit yet; Task 2 makes the single commit.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web && npx vitest run src/lib/util/gpx_util.test.ts && test "$(grep -c 'routeToStoredGpx(valhallaStore.route)' 'src/routes/trail/edit/[id]/+page.svelte')" -ge 1 && test "$(grep -c 'splitTrackSegmentsAtAnchors(' 'src/routes/trail/edit/[id]/+page.svelte')" -ge 2 && test "$(grep -c 'valhallaStore.route.toString' 'src/routes/trail/edit/[id]/+page.svelte')" -eq 0 && test "$(grep -c 'export function routeToStoredGpx' src/lib/util/gpx_segment_util.ts)" -ge 1 && echo WIRING-OK</automated>
  </verify>
  <acceptance_criteria>
    - vitest: every test in src/lib/util/gpx_util.test.ts passes, including the new routeToStoredGpx Tests A and B and all the existing #1370 tests.
    - `grep -n "routeToStoredGpx(valhallaStore.route)"` on the page matches the save site (count >= 1).
    - `grep -c "valhallaStore.route.toString"` on the page == 0.
    - `grep -n "splitTrackSegmentsAtAnchors("` on the page still matches both load paths, onMount and handleFileSelection (count >= 2).
    - `grep -n "export function routeToStoredGpx"` on web/src/lib/util/gpx_segment_util.ts matches (count >= 1).
  </acceptance_criteria>
  <done>The save path uploads one marked trkseg per track for a contiguous drawn route, and a stored file round-trips back to the same legs. The baseline svelte-check line is recorded.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Lock down non-mutation, mismatched legs and the empty route; reword the header comment; check and commit</name>
  <files>web/src/lib/util/gpx_util.test.ts, web/src/lib/util/gpx_segment_util.ts</files>
  <read_first>
    - web/src/lib/util/gpx_util.test.ts (the routeToStoredGpx describe added in Task 1)
    - web/src/lib/util/gpx_segment_util.ts lines 1-10 (header comment)
  </read_first>
  <behavior>
    - Test C (input not mutated, D-01): with `input = GPX.parse(multiLegGpx)` and `before = input.toString()`, after calling `routeToStoredGpx(input)`, `input.trk[0].trkseg` still has length 3, `anchors(input)` is `[[]]`, `input.$["xmlns:wanderer"]` is undefined, `coordinates(input)` equals `coordinates(GPX.parse(multiLegGpx))`, and `input.toString()` equals `before`.
    - Test D (in-memory legs like the store builds them): a `new GPX({ trk: [new Track({ trkseg: [segment([pt(1,1), pt(2,2)]), segment([pt(2,2), pt(3,3)]), segment([pt(3,3), pt(4,4)])] })] })` built with constructors, not parsed, serializes to 1 segment with `anchors` `[[1, 2]]`, and the original object still has 3 trksegs afterwards.
    - Test E (mismatched legs stay separate, D-01): legs `[1,1]-[2,2]`, `[2,2]-[3,3]` and `[3.0001,3]-[4,4]` parse to 2 trksegs. The first is `[[1,1],[2,2],[3,3]]` with a mark at index 1, the second is `[[3.0001,3],[4,4]]` with no mark. After `splitTrackSegmentsAtAnchors` on the parsed copy, `coordinates` equals the original 3 legs.
    - Test F (empty route): `routeToStoredGpx(new GPX({ trk: [new Track({ trkseg: [] })] }))` does not throw, the result parses without throwing, it has no track points, and it does NOT contain `xmlns:wanderer`.
  </behavior>
  <action>
    RED/GREEN: add Tests C to F to the `describe("routeToStoredGpx", ...)` block, reusing `pt`, `segment`, `coordinates`, `anchors` and `multiLegGpx`. With Task 1's helper they should pass as written. If one fails, fix the helper, not the test. The likely candidate is F: if `GPX.parse` ever throws on the empty serialization, guard inside routeToStoredGpx by returning `gpx.toString()` unchanged when the route has no track points. Do not loosen the equality rule to make E pass (D-01).

    Header comment (D-04): reword the comment at the top of web/src/lib/util/gpx_segment_util.ts (lines 5-7). It currently says the junction point is marked "when segments are joined for export". Make it say that segments are joined when a drawn route is saved or exported, and that the junction point is marked so the editor can split the track again. Keep it to 3 lines in the same style. This is the only #1370 text that changes. Make no code removals: per D-04, the trail2gpx export join (web/src/lib/util/gpx_util.ts), both load-path splits, and the #1370 vitest and Playwright tests all stay, and web/src/lib/util/gpx_util.ts must not appear in the diff.

    Type check: run `cd /Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web && npm run check 2>&1 | tail -3` and compare it with the Task 1 baseline line. The error count must not increase. Pre-existing errors are acceptable. If the count rose, fix the new errors (most likely the GPX value import or the page import) before committing.

    Commit (user rules override defaults): from the worktree, stage ONLY the three files by explicit path. Never use `git add -A` or `git add .`, because the worktree may carry skip-worktree files such as web/vite.config.ts. Then make one commit with the headline-only message `fix(web): store drawn routes as one marked track segment`: no body and no Co-Authored-By trailer (use a single `-m`). Do NOT push and do NOT open a PR; the orchestrator does that.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web && npx vitest run src/lib/util/gpx_util.test.ts && npm run check 2>&1 | tail -3 && git -C /Users/christianbeutel/Documents/svelte/wanderer-route-anchors show --name-only --format=%B HEAD && test "$(git -C /Users/christianbeutel/Documents/svelte/wanderer-route-anchors log -1 --format=%B | sed '/^$/d' | wc -l | tr -d ' ')" = 1 && test -z "$(git -C /Users/christianbeutel/Documents/svelte/wanderer-route-anchors diff --name-only HEAD~1 HEAD -- web/src/lib/util/gpx_util.ts)" && echo COMMIT-OK</automated>
    <human-check>In the dev server (web, worktree): draw a route with 3+ anchors, save, and download the stored GPX (the trail's PocketBase gpx file, or the export). It shows one trkseg with a wanderer:anchor at each interior anchor and xmlns:wanderer on the root. Right after saving, without a reload, drag an anchor and press undo: both work. Reload the editor: the same anchors appear.</human-check>
  </verify>
  <acceptance_criteria>
    - vitest: all tests in src/lib/util/gpx_util.test.ts pass (A to F plus the existing ones).
    - The svelte-check error count is less than or equal to the Task 1 baseline (both lines quoted in the SUMMARY).
    - The HEAD commit message is exactly `fix(web): store drawn routes as one marked track segment` (one line, no trailer).
    - The HEAD commit touches exactly web/src/lib/util/gpx_segment_util.ts, web/src/lib/util/gpx_util.test.ts and web/src/routes/trail/edit/[id]/+page.svelte. web/src/lib/util/gpx_util.ts is unchanged.
    - Nothing pushed: `git -C /Users/christianbeutel/Documents/svelte/wanderer-route-anchors status -sb` shows the branch with no upstream or ahead of it.
  </acceptance_criteria>
  <done>The edge cases are pinned by tests, the header comment reflects save-time joining, there are no new type errors, and there is one local headline-only commit on fix/route-anchor-marks-on-save. The SUMMARY states explicitly that #1370 had no dead code to remove (D-04).</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| browser editor to trail form API | The user-authored GPX string uploaded as gpxFile. No new input path: the same string is still built from the in-memory route, just joined. |
| stored GPX to other readers (export, view map, Go backend, federation, app) | The stored shape changes from per-leg segments to one marked segment |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-261014-nqn-01 | Tampering | routeToStoredGpx and valhallaStore.route | medium | mitigate | Clone via GPX.parse(gpx.toString()) and join only the clone. Test C and Test D assert the input keeps 3 trksegs, carries no marks or xmlns, and serializes identically afterwards. |
| T-261014-nqn-02 | Tampering (data integrity) | joinContiguousTrackSegments equality rule | medium | mitigate | Exact Number equality only, with no tolerance (D-01). Test E proves mismatched legs stay separate and reload restores all 3 legs. |
| T-261014-nqn-03 | Denial of Service | save with an empty or degenerate route | low | mitigate | Test F: an empty route serializes and parses without throwing, so save cannot start failing for routes with no points. |
| T-261014-nqn-04 | Denial of Service | extra parse+serialize at save for long routes | low | accept | One extra O(n) client-side pass per save. The parse already ran on load. |
| T-261014-nqn-05 | Information Disclosure | stored GPX contents | low | accept | Only adds an empty namespaced extension element and a root xmlns. No new data leaves the client. |
| T-261014-nqn-SC | Tampering | npm/pip/cargo installs | low | accept | Not applicable: this plan installs no packages. |
</threat_model>

<verification>
All commands run with cwd in the worktree `/Users/christianbeutel/Documents/svelte/wanderer-route-anchors/web`:
- `npx vitest run src/lib/util/gpx_util.test.ts`: all green.
- `npm run check`: no new errors compared with the Task 1 baseline.
- `grep -n "routeToStoredGpx(valhallaStore.route)" "src/routes/trail/edit/[id]/+page.svelte"` matches the save site (>= 1), and `grep -n "splitTrackSegmentsAtAnchors(" ...` still matches both load paths (>= 2).
- `git show --name-only HEAD` lists exactly the three planned files, with a headline-only message.
</verification>

<success_criteria>
- A drawn, contiguous multi-leg route is stored as one trkseg with anchor marks and the wanderer xmlns (ANCHOR-01).
- The editor's per-leg model, anchors and undo stack survive a save untouched (ANCHOR-02).
- Mismatched legs stay separate, reload restores every leg, and an empty route still saves (ANCHOR-03).
- The #1370 export join and load splits are kept, nothing is removed, and the review result is recorded in the SUMMARY (ANCHOR-04).
- One local commit, nothing pushed, no PR.
</success_criteria>

<output>
Create `/Users/christianbeutel/Documents/svelte/wanderer/.planning/quick/261014-nqn-namespace-based-route-anchors-web-save-t/261014-nqn-SUMMARY.md` in the MAIN checkout, not in the worktree. Include: the commit hash on fix/route-anchor-marks-on-save, the before and after svelte-check lines, the D-03 audit result (no post-save consumer needs per-leg segments), and the D-04 statement that #1370 had no dead code (export join and load splits kept; only the header comment reworded).
</output>

## Follow-up: app mirror (NOT part of this plan; planned separately after the other app session settles)

On feature/app, `app/lib/util/route/planner_handoff.dart` `buildFinalPlannedGpx` currently emits one Trkseg per leg and drops each leg's closing point. The mirror:
- Serialize the planner route as ONE trkseg. Put `{'wanderer:anchor': ''}` in the extensions of the first point of each leg after the first, and declare `xmlns:wanderer="https://wanderer.to/xmlns/gpx/1"` on the root via `GpxWriter().asXml(gpx)` + `rootElement.setAttribute`, because gpx 2.3.0 has no namespace support in `asString`.
- Reading goes through `parseGpxSafely` plus a Dart port of `splitTrackSegmentsAtAnchors` that strips the marks. It must also ignore marks on segment ends, and an anchor point becomes both the last point of leg i and the first point of leg i+1.
- Keep the in-memory Gpx per-leg; only the serialized form changes.
- Check while planning: confirm `parseGpxSafely` accepts `<extensions><wanderer:anchor/></extensions>` on a trkpt. Until the mirror ships, the app sees web-saved drawn routes as a single leg when it re-opens them in its planner. Rendering and navigation are unaffected, because every reader concatenates segments.
