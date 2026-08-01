---
status: diagnosed
trigger: "b) pass. Check if the json content type is still needed. If not remove it alongside the associated tests"
created: 2026-08-01T00:00:00Z
updated: 2026-08-01T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED — the `application/json` REQUEST content-type branch in the convert
  handler has never had a caller, in any client, in the entire history of the file.
test: (complete) enumerated every call site across app/, web/, db/, scripts, docs and git history
expecting: (complete)
next_action: none — diagnose-only mode. Hand off to plan-phase for the removal.

## Symptoms

expected: The `POST /api/v1/trail/convert` endpoint carries no content type it no longer serves.
actual: A JSON content-type path may be left over from pre-Phase-34 behavior where the endpoint
  returned a computed Trail as JSON.
errors: none (dead-code / API-surface hygiene, not a malfunction)
reproduction: N/A — static code question
started: Raised during UAT of Phase 34 (dart-conversion-port), test 3(b)

## Eliminated

- hypothesis: "The endpoint still has a JSON RESPONSE branch (returns a Trail as JSON)"
  evidence: `+server.ts` has exactly one `return new Response(...)` on the success path, hardcoded
    to `Content-Type: application/gpx+xml`. `json()` was dropped from the imports in df61d581.
    The `D-06 regression guard` test (convert.test.ts:200) asserts the 200 body does not parse as
    JSON. There is no JSON response branch to remove.
  timestamp: 2026-08-01

- hypothesis: "The handler is in the Go backend under db/routes/ (per the debug brief)"
  evidence: `grep -rn "convert" --include="*.go" db/` returns only unrelated prose ("converts",
    "converted") in status.go, importer.go, seed_regions.go, poly_parser.go, geometry_fetch.go.
    The handler is a SvelteKit route: `web/src/routes/api/v1/trail/convert/+server.ts`.
  timestamp: 2026-08-01

- hypothesis: "The web frontend calls the endpoint for transcoding"
  evidence: `grep -rn "trail/convert|/convert" web/src/` excluding the route's own directory
    returns ZERO matches (exit 1). The trail-edit page transcodes client-side:
    `web/src/routes/trail/edit/[id]/+page.svelte:480` `fromFile(selectedFile)` then `:485`
    `gpx2trail(gpxData, selectedFile.name)`. Matches D-08 exactly.
  timestamp: 2026-08-01

- hypothesis: "Removing the JSON branch is a breaking change for published-API consumers"
  evidence: (a) `git merge-base --is-ancestor 60c663ee origin/main` -> NO. `git ls-tree -r
    origin/main --name-only | grep trail/convert` -> empty. The entire endpoint does not exist on
    origin/main (open-wanderer/wanderer, Release v0.20.0); it lives only on the unmerged
    feature/app branch (1209 commits ahead). It has never shipped.
    (b) Even setting that aside: the generated OpenAPI `requestBody` for this path declares ONLY
    `multipart/form-data`. `application/json` was never a declared request media type — it appears
    only inside the human-readable `description` prose.
  timestamp: 2026-08-01

## Evidence

- timestamp: 2026-08-01
  checked: `web/src/routes/api/v1/trail/convert/+server.ts` (full read)
  found: Content type is negotiated on the REQUEST only, via
    `event.request.headers.get("content-type")` at line 82, into three branches:
    (1) `multipart/form-data` -> `formData()` -> `file` Blob -> `fromFile()` transcode (line 84)
    (2) `application/json` -> `request.json()` -> `body.gpx || body.gpxData` (lines 99-102)
    (3) else -> `request.text()` raw body (lines 103-106)
    All three converge on one response, hardcoded `application/gpx+xml` (line 118-128).
  implication: "the json content type" can only mean input branch (2), or the prose describing it
    in the published docs. There is no JSON output to remove.

- timestamp: 2026-08-01
  checked: `app/lib/util/trail_import_util.dart:211-232` (`transcodeToGpx`)
  found: `FormData.fromMap({'file': await MultipartFile.fromFile(path, filename: name)})` then
    `.post('/trail/convert', data: formData)`. Dio sends `multipart/form-data`. Never JSON.
  implication: The one live app caller uses branch (1). It cannot reach branch (2).

- timestamp: 2026-08-01
  checked: `app/test/util/trail_import_util_test.dart:413-445` (PORT-03 gate test)
  found: An enforced test asserts exactly ONE occurrence of the string `trail/convert` in all of
    `app/lib/`, and that it is in `util/trail_import_util.dart`.
  implication: The app's caller count is structurally pinned at one. No hidden second call site.

- timestamp: 2026-08-01
  checked: repo-wide sweep for `trail/convert` (excluding node_modules, .svelte-kit, build, dist)
  found: only `app/lib/util/trail_import_util.dart`, `app/test/util/trail_import_util_test.dart`,
    the route's own `+server.ts` / `convert.test.ts`, the generated
    `web/static/docs/api/wanderer.openapi.json`, and .planning docs. No Go caller, no script,
    no migration, no docker/compose reference.
  implication: The complete consumer set is: the Flutter app (multipart) and nothing else.

- timestamp: 2026-08-01
  checked: git history — `git log --follow` on +server.ts, then `git show ca063023:...`
  found: The JSON branch was introduced in `ca063023` ("move to serverside trail conversion",
    2026-07-19), together with the raw-text fallback, in the same commit that first made the app
    call the endpoint. At that commit the only entry point was
    `Future<Trail> convertGpxToTrail(WidgetRef ref, FormData formData)` — parameter typed
    `FormData`, so multipart by construction — with two call sites
    (`route_planner_handoff_util.dart:58`, `trail_import_util.dart:60`), both passing FormData.
  implication: The JSON request branch has NEVER had a caller. It was speculative generality
    added the day the endpoint was written, not a leftover from a retired working feature.

- timestamp: 2026-08-01
  checked: `.planning/phases/34-dart-conversion-port/34-07-PLAN.md` decisions
  found: D-06 states verbatim: "the three input branches (multipart, JSON, raw text) and the
    empty-body 400 guard are unchanged, so the app's existing multipart upload keeps working."
  implication: Phase 34 deliberately did NOT touch the JSON branch. It was a minimal-diff
    decision (change the output, leave the inputs alone), not an oversight. So the user's UAT
    note is asking to finish a job Phase 34 consciously scoped out — this is follow-up cleanup,
    not a Phase 34 defect.

- timestamp: 2026-08-01
  checked: `web/static/docs/api/wanderer.openapi.json` (v0.20.0) and `docs/wanderer.openapi.json`
    (v0.19.2, stale); `git check-ignore`
  found: The convert entry's `requestBody.content` declares ONLY `multipart/form-data` (with
    `file` + `name`). `responses.200.content` declares only `application/gpx+xml`. JSON appears
    solely in the `description` string: "or a GPX string (JSON body with `gpx`/`gpxData`, or a
    raw text body)". Both spec files are gitignored build artifacts
    (`web/.gitignore:6`, `docs/.gitignore:7`), regenerated by `sveltekit-openapi-generator` and
    copied to the docs site by `docs/scripts/sync-openapi.js`.
  implication: What the user saw on /docs/api is prose, not a declared media type. Removing the
    branch requires only a description reword plus a regenerate; no committed artifact changes.

- timestamp: 2026-08-01
  checked: `npx vitest run src/routes/api/v1/trail/convert/convert.test.ts --reporter=verbose`
  found: 16 tests, all passing. Exactly 3 exercise the JSON request branch:
    - convert.test.ts:100 "400s on a non-GPX body submitted through the JSON branch"
      (inside the CR-04 `content validation` suite)
    - convert.test.ts:114 "accepts the `gpx` key" (inside the `JSON branch` suite)
    - convert.test.ts:128 "accepts the `gpxData` key" (inside the `JSON branch` suite)
  implication: Removal blast radius on tests is 3 of 16.

- timestamp: 2026-08-01
  checked: fall-through behaviour if branch (2) is deleted
  found: An `application/json` request would land in the `else` raw-text branch;
    `request.text()` returns the literal JSON string, `assertParsableGpx` throws, response is
    400 "Invalid GPX content". This is already directly proven by the existing passing test
    convert.test.ts:83 ("400s on a JSON blob submitted as a raw body, and does not echo it").
    Edge case that IMPROVES: `Content-Type: application/json` carrying a bare GPX string
    currently 500s (`request.json()` SyntaxError -> handleError); after removal it 200s.
  implication: Removal degrades gracefully. No 500s, no unhandled paths.

- timestamp: 2026-08-01
  checked: `app/lib/util/trail_import_util.dart:220-230` + `app/test/util/trail_import_util_test.dart:231`
  found: `transcodeToGpx` still tolerates a legacy JSON RESPONSE Map
    (`data['expand']['gpx_data']`), with a test pinning it. Deliberate per the 34-05 doc comment
    and threat T-34-37 (new-app/old-server skew, D-05 rationale "self-hosters control their own
    server/app pairing").
  implication: This is the OTHER piece of JSON leftover in the convert round trip — client-side.
    Since the endpoint never shipped upstream, no "old server" returning JSON exists outside this
    branch. Adjacent candidate for the same cleanup, but a separate decision from the server fix.

- timestamp: 2026-08-01
  checked: `git status --porcelain` / `git diff` (working tree is NOT clean, contrary to the
    session-start snapshot — concurrent agent activity)
  found: Uncommitted edits exist to `web/src/routes/api/v1/trail/convert/+server.ts:46-49`
    (deleted the line "Breaking change: prior to this version the endpoint returned a JSON
    `Trail`." and left a malformed stray backtick-period: "...trail metrics themselves.`.") and
    to `web/src/lib/util/gpx_util.ts:78-80` (deleted a bbox explanatory comment). Also present:
    scratch files `app/test/util/zz_tmp_debug_distance_test.dart` and
    `web/src/lib/util/zz_tmp_debug_distance.test.ts` from the parallel distance-mismatch debug.
  implication: The swagger description a fix will edit is currently dirty and malformed. Reconcile
    the working tree before applying the removal, or the stray "`." ships into the docs.

## Resolution

root_cause: |
  `POST /api/v1/trail/convert` has no JSON response — Phase 34-07 (df61d581) already removed it.
  What remains is an `application/json` REQUEST content-type branch
  (`web/src/routes/api/v1/trail/convert/+server.ts:99-102`) that reads `body.gpx || body.gpxData`.
  That branch has never had a caller: it was added speculatively in ca063023 alongside the app's
  first (and only ever) call site, which was typed `FormData` and therefore multipart by
  construction. Today the sole consumer is `transcodeToGpx` in
  `app/lib/util/trail_import_util.dart`, also multipart. The web frontend never calls the endpoint
  (D-08, verified by grep), no Go/script/migration caller exists, and the endpoint is absent from
  origin/main entirely — it has never been published. Phase 34's D-06 consciously left the input
  branches untouched to keep the diff minimal, so this is scoped-out follow-up cleanup rather than
  a Phase 34 defect.
fix: (not applied — diagnose-only mode)
verification: (n/a)
files_changed: []
