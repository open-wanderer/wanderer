---
phase: 04-nodeinfo
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - db/routes/nodeinfo.go
  - db/routes/nodeinfo_test.go
  - db/main.go
autonomous: true
requirements: [SAFE-04]

must_haves:
  truths:
    - "GET /.well-known/nodeinfo returns a JSON discovery document with a links array whose rel is http://nodeinfo.diaspora.software/ns/schema/2.1"
    - "GET /.well-known/nodeinfo/2.1 returns NodeInfo 2.1 JSON with software.name == \"wanderer\""
    - "software.version reflects WANDERER_VERSION env var, or \"dev\" when unset"
    - "usage.localPosts equals the count of public trails"
    - "usage.users.total equals the count of all users"
  artifacts:
    - path: "db/routes/nodeinfo.go"
      provides: "NodeInfo discovery + 2.1 handlers and their pure payload builders"
      contains: "func NodeInfo"
    - path: "db/routes/nodeinfo_test.go"
      provides: "Unit tests for the payload builders against a bootstrapped test app"
      contains: "func Test"
    - path: "db/main.go"
      provides: "Route registration for both well-known endpoints"
      contains: "/.well-known/nodeinfo"
  key_links:
    - from: "db/main.go registerRoutes"
      to: "routes.NodeInfo / routes.NodeInfo21"
      via: "se.Router.GET(\"/.well-known/nodeinfo\", ...) and se.Router.GET(\"/.well-known/nodeinfo/2.1\", ...)"
      pattern: "se\\.Router\\.GET\\(\"/\\.well-known/nodeinfo"
    - from: "buildNodeInfo21"
      to: "trails + users collections"
      via: "app.CountRecords"
      pattern: "app\\.CountRecords\\(\"(trails|users)\""
---

## Phase Goal

**As a** peer ActivityPub instance or federation tool, **I want to** fetch this server's NodeInfo well-known endpoints, **so that** I can identify it as Wanderer software and read its live user and post counts.

<objective>
Serve the two NodeInfo well-known HTTP endpoints (SAFE-04) so peer instances can discover and identify this deployment:

1. `GET /.well-known/nodeinfo` — JRD discovery document linking to the 2.1 schema endpoint
2. `GET /.well-known/nodeinfo/2.1` — NodeInfo 2.1 payload with `software.name: "wanderer"`, version from env, and live user/post counts

Purpose: Closes the last v1 requirement (SAFE-04). Peer Wanderer instances and generic fediverse tooling probe NodeInfo to confirm software identity before/after establishing instance follows.

Output: New `db/routes/nodeinfo.go` (handlers + pure builders), `db/routes/nodeinfo_test.go` (builder unit tests), and two route registrations in `db/main.go`.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/ROADMAP.md
@.planning/REQUIREMENTS.md
@.planning/phases/04-nodeinfo/04-CONTEXT.md
@db/routes/health.go
@db/routes/activitypub.go
@db/main.go
</context>

## Artifacts this phase produces

New symbols created by this phase (exclude from drift verification — they do not yet exist in the codebase):

- `db/routes/nodeinfo.go` (new file)
- `db/routes/nodeinfo_test.go` (new file)
- `func NodeInfo(e *core.RequestEvent) error` — discovery handler in package `routes`
- `func NodeInfo21(e *core.RequestEvent) error` — NodeInfo 2.1 handler in package `routes`
- `func buildNodeInfoDiscovery(origin string) map[string]any` — pure discovery-doc builder
- `func buildNodeInfo21(app core.App) (map[string]any, error)` — pure NodeInfo 2.1 payload builder
- `WANDERER_VERSION` environment variable (consumed by `buildNodeInfo21`)
- Two route registrations in `db/main.go`: `GET /.well-known/nodeinfo` and `GET /.well-known/nodeinfo/2.1`

## NodeInfo target values (copy these exactly)

Discovery doc (`/.well-known/nodeinfo`) JRD shape — top-level `links` array, one entry:
- `rel`: `http://nodeinfo.diaspora.software/ns/schema/2.1`
- `href`: `{ORIGIN}/.well-known/nodeinfo/2.1`

NodeInfo 2.1 payload required top-level keys (all 7 must be present per schema):
- `version`: `"2.1"`
- `software`: object with `name: "wanderer"` (must match pattern `^[a-z0-9-]+$`) and `version: <WANDERER_VERSION or "dev">`. Include `homepage: "https://wanderer.to"` and `repository: "https://github.com/Flomp/wanderer"` (optional but recommended).
- `protocols`: `["activitypub"]`
- `services`: object `{ "inbound": [], "outbound": [] }`
- `openRegistrations`: `false`
- `usage`: object with `users: { "total": <count of users collection> }` and `localPosts: <count of trails WHERE public = true>`
- `metadata`: empty object `{}`

Per D-01: `software.version` = `os.Getenv("WANDERER_VERSION")`; if empty string, use `"dev"`.
Per D-02: `localPosts` = `app.CountRecords("trails", dbx.NewExp("public = true"))`.
Per D-03: `users.total` = `app.CountRecords("users", nil)`.

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Failing tests for NodeInfo payload builders</name>
  <files>db/routes/nodeinfo_test.go</files>
  <read_first>
    - db/routes/nodeinfo_test.go (will not exist yet — confirm, then create)
    - db/federation/instance_inbox_test.go (lines 1-70 — canonical `core.NewBaseApp` + `Bootstrap()` + `ImportCollectionsByMarshaledJSON` test-bootstrap pattern; replicate it inline in package `routes`)
    - db/federation/create_test.go (lines 90-138 — `app.CountRecords` usage and `t.Setenv` for env-dependent tests)
    - .planning/phases/04-nodeinfo/04-CONTEXT.md (D-01/D-02/D-03 — exact count and version semantics)
  </read_first>
  <behavior>
    - Test buildNodeInfoDiscovery: given origin "https://w.example.com", returned map has links[0].rel == "http://nodeinfo.diaspora.software/ns/schema/2.1" and links[0].href == "https://w.example.com/.well-known/nodeinfo/2.1".
    - Test buildNodeInfo21 software identity: software.name == "wanderer", version key present.
    - Test buildNodeInfo21 version fallback: with WANDERER_VERSION unset (t.Setenv to ""), software.version == "dev"; with WANDERER_VERSION="1.2.3", software.version == "1.2.3".
    - Test buildNodeInfo21 localPosts: seed 2 public trails + 1 private trail → usage.localPosts == 2 (private excluded, D-02).
    - Test buildNodeInfo21 users total: seed 3 user records → usage.users.total == 3 (D-03).
    - Test buildNodeInfo21 required keys: returned map contains all 7 top-level keys: version, software, protocols, services, openRegistrations, usage, metadata.
  </behavior>
  <action>
    Create db/routes/nodeinfo_test.go in package `routes`. Build a local test-app helper modeled on `newInboxTestApp` from db/federation/instance_inbox_test.go: call `core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir(), EncryptionEnv: "POCKETBASE_ENCRYPTION_KEY"})`, then `app.Bootstrap()`, then import minimal `trails` and `users` collections via `app.ImportCollectionsByMarshaledJSON`. The trails collection needs at minimum `id` and a `public` bool field; the users collection needs at minimum `id` (an `auth`-type or `base`-type collection is fine for counting). Set `POCKETBASE_ENCRYPTION_KEY` to a 32-byte value with `t.Setenv` (see create_test.go line 94). Seed records with `core.NewRecord(collection)` + `app.Save(record)` setting `public` true/false on trails. Assert builder outputs match the target values in the "NodeInfo target values" section above. Reference WANDERER_VERSION via `t.Setenv("WANDERER_VERSION", ...)`. These tests MUST fail to compile/run initially because buildNodeInfoDiscovery and buildNodeInfo21 do not exist yet — that is the RED state. Do NOT create nodeinfo.go in this task. Commit as `test(04-01): add failing NodeInfo builder tests`.
  </action>
  <verify>
    <automated>cd db && go test ./routes/ -run TestNodeInfo 2>&1 | grep -qE "undefined: buildNodeInfo|cannot find|build failed|FAIL" && echo RED-OK</automated>
  </verify>
  <acceptance_criteria>
    - db/routes/nodeinfo_test.go exists and declares `package routes`
    - nodeinfo_test.go contains `buildNodeInfoDiscovery(` and `buildNodeInfo21(`
    - nodeinfo_test.go contains `app.CountRecords` is NOT required, but it MUST seed trails with both public=true and public=false rows and assert localPosts excludes the private one
    - `cd db && go test ./routes/ -run TestNodeInfo` fails (RED) because builders are undefined — output contains `undefined: buildNodeInfo` or a compile error
    - Test asserts software.name == "wanderer" and the version fallback "dev" when WANDERER_VERSION is empty
  </acceptance_criteria>
  <done>Tests for both builders exist and fail because the builder functions do not yet exist (RED state confirmed).</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement NodeInfo payload builders (GREEN)</name>
  <files>db/routes/nodeinfo.go</files>
  <read_first>
    - db/routes/nodeinfo.go (will not exist yet — confirm, then create)
    - db/routes/nodeinfo_test.go (the tests written in Task 1 — implement to satisfy them exactly)
    - db/routes/health.go (canonical minimal handler / package `routes` import style)
    - db/federation/actor.go (lines 180-195 — `app.CountRecords(collection, dbx.NewExp(...))` and `dbx` import path `github.com/pocketbase/dbx`)
    - .planning/phases/04-nodeinfo/04-CONTEXT.md (D-01/D-02/D-03)
  </read_first>
  <action>
    Create db/routes/nodeinfo.go in package `routes`. Implement two pure builder functions (no `*core.RequestEvent`, so they are unit-testable):
    1. `func buildNodeInfoDiscovery(origin string) map[string]any` — returns the JRD discovery doc: a `links` slice with one entry `{"rel": "http://nodeinfo.diaspora.software/ns/schema/2.1", "href": origin + "/.well-known/nodeinfo/2.1"}`.
    2. `func buildNodeInfo21(app core.App) (map[string]any, error)` — returns the NodeInfo 2.1 map with all 7 required top-level keys using the exact values from the "NodeInfo target values" section. Compute `version` via `os.Getenv("WANDERER_VERSION")` falling back to `"dev"` when empty (D-01). Compute `localPosts` via `app.CountRecords("trails", dbx.NewExp("public = true"))` (D-02). Compute `users.total` via `app.CountRecords("users", nil)` (D-03). Return any CountRecords error to the caller. Hardcode `software.name` as `"wanderer"`, `protocols` as `[]string{"activitypub"}`, `services` as `map[string]any{"inbound": []string{}, "outbound": []string{}}`, `openRegistrations` as `false`, `metadata` as `map[string]any{}`. Include `homepage` "https://wanderer.to" and `repository` "https://github.com/Flomp/wanderer" in the software object.
    Import `github.com/pocketbase/dbx`, `github.com/pocketbase/pocketbase/core`, and `os`. Do NOT add the HTTP handlers or route registration yet (Task 3). Run the Task 1 tests — they MUST now pass (GREEN). Commit as `feat(04-01): implement NodeInfo payload builders`.
  </action>
  <verify>
    <automated>cd db && go test ./routes/ -run TestNodeInfo -v 2>&1 | tail -20 | grep -q "ok\|PASS" && echo GREEN-OK</automated>
  </verify>
  <acceptance_criteria>
    - db/routes/nodeinfo.go exists, declares `package routes`, imports `github.com/pocketbase/dbx`
    - nodeinfo.go contains `func buildNodeInfoDiscovery(origin string) map[string]any`
    - nodeinfo.go contains `func buildNodeInfo21(app core.App) (map[string]any, error)`
    - nodeinfo.go contains the literal string `"wanderer"` and the literal `"http://nodeinfo.diaspora.software/ns/schema/2.1"`
    - nodeinfo.go contains `app.CountRecords("trails"` and `app.CountRecords("users"`
    - nodeinfo.go references `os.Getenv("WANDERER_VERSION")`
    - `cd db && go test ./routes/ -run TestNodeInfo` exits 0 (GREEN)
    - `cd db && go build ./...` exits 0
  </acceptance_criteria>
  <done>Both builders implemented; all Task 1 tests pass; package builds cleanly.</done>
</task>

<task type="auto">
  <name>Task 3: Wire handlers and register the two well-known routes</name>
  <files>db/routes/nodeinfo.go, db/main.go</files>
  <read_first>
    - db/routes/nodeinfo.go (the builders from Task 2 — handlers call these)
    - db/routes/health.go (the `func Name(e *core.RequestEvent) error { return e.JSON(http.StatusOK, ...) }` handler shape)
    - db/routes/activitypub.go (lines 19-67 — `e.JSON(http.StatusOK, ...)` usage and `os.Getenv("ORIGIN")` with empty-check returning an error)
    - db/main.go (lines 168-203 — `registerRoutes()`; add the two GET registrations alongside the existing `/activitypub/...` routes)
  </read_first>
  <action>
    In db/routes/nodeinfo.go add two exported handlers:
    1. `func NodeInfo(e *core.RequestEvent) error` — read `origin := os.Getenv("ORIGIN")`; if empty return an error like `fmt.Errorf("ORIGIN not set")` (mirror activitypub.go line 67-69); return `e.JSON(http.StatusOK, buildNodeInfoDiscovery(origin))`.
    2. `func NodeInfo21(e *core.RequestEvent) error` — call `buildNodeInfo21(e.App)`; on error return it; otherwise `return e.JSON(http.StatusOK, payload)`. (Claude's discretion per CONTEXT.md: optionally set `Content-Type` to `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"` via `e.Response.Header().Set(...)` before the JSON write; standard `application/json` is acceptable.)
    Add imports `net/http` and `fmt` to nodeinfo.go as needed.
    In db/main.go `registerRoutes()` add exactly two lines near the existing `se.Router.GET("/activitypub/...")` block:
    `se.Router.GET("/.well-known/nodeinfo", routes.NodeInfo)` and `se.Router.GET("/.well-known/nodeinfo/2.1", routes.NodeInfo21)`.
    Do not modify any other route. Per CONTEXT.md integration note: if `go build` reveals `se.Router` does not accept the root-level `/.well-known/...` path, fall back to the documented PocketBase v0.26 router for root paths and record the deviation in the SUMMARY — do NOT prefix the path with `/api/v1/`. Commit as `feat(04-01): wire NodeInfo handlers and register well-known routes`.
  </action>
  <verify>
    <automated>cd db && go build ./... && go vet ./routes/ && grep -c '/.well-known/nodeinfo' main.go | grep -qE '^[2-9]' && echo WIRED-OK</automated>
  </verify>
  <acceptance_criteria>
    - db/routes/nodeinfo.go contains `func NodeInfo(e *core.RequestEvent) error` and `func NodeInfo21(e *core.RequestEvent) error`
    - Both handlers call `e.JSON(http.StatusOK, ...)` with the corresponding builder output
    - db/main.go contains `se.Router.GET("/.well-known/nodeinfo", routes.NodeInfo)`
    - db/main.go contains `se.Router.GET("/.well-known/nodeinfo/2.1", routes.NodeInfo21)`
    - `cd db && go build ./...` exits 0
    - `cd db && go test ./routes/ -run TestNodeInfo` still exits 0 (builders unchanged behavior)
  </acceptance_criteria>
  <done>Both endpoints registered and the backend compiles; NodeInfo builder tests still pass.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| public internet → /.well-known/nodeinfo[/2.1] | Unauthenticated GET requests from any peer instance or crawler reach these handlers |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-04-01 | Information disclosure | NodeInfo 2.1 usage counts | accept | Only aggregate, already-public counts are exposed: total users and count of public trails. No PII, no private-trail leakage — D-02 restricts localPosts to `public = true`. Standard fediverse behavior; Mastodon/others expose the same. |
| T-04-02 | Information disclosure | localPosts private-trail leak | mitigate | `buildNodeInfo21` counts trails with `dbx.NewExp("public = true")` only (D-02); private trails are never counted. Task 1 test asserts a private trail is excluded from the count. |
| T-04-03 | Denial of service | unauthenticated count queries on every request | accept | Two `CountRecords` queries per request are cheap and indexed; endpoints are GET-only with no write path. Rate limiting is handled at the reverse-proxy layer, consistent with other public well-known endpoints (webfinger). |
| T-04-04 | Spoofing | software.name / version forgery by this server | accept | This server self-reports its own identity; NodeInfo is advisory metadata, not an authentication mechanism. Peers do not make trust decisions solely on NodeInfo. |
| T-04-SC | Tampering | npm/pip/cargo installs | mitigate | No new packages installed this phase — only stdlib `os`/`net/http`/`fmt` and already-vendored `github.com/pocketbase/dbx` + `github.com/pocketbase/pocketbase/core`. No package legitimacy gate required. |
</threat_model>

<verification>
Phase-level checks (run after all tasks):

1. `cd db && go build ./...` exits 0 — backend compiles with the new routes wired.
2. `cd db && go test ./routes/ -run TestNodeInfo` exits 0 — all builder behavior verified (version fallback, public-only post count, user count, required keys, discovery rel/href).
3. `grep -c '/.well-known/nodeinfo' db/main.go` returns 2 — both routes registered.
4. Manual smoke (optional, post-merge with a running instance): `curl -s $ORIGIN/.well-known/nodeinfo` returns the discovery JRD; `curl -s $ORIGIN/.well-known/nodeinfo/2.1` returns JSON with `software.name == "wanderer"`.
</verification>

<success_criteria>
SAFE-04 is satisfied when:

1. `GET /.well-known/nodeinfo` returns a JSON discovery document whose `links` array contains an entry with `rel: "http://nodeinfo.diaspora.software/ns/schema/2.1"` and an `href` pointing at `{ORIGIN}/.well-known/nodeinfo/2.1` (ROADMAP success criterion 1).
2. `GET /.well-known/nodeinfo/2.1` returns valid NodeInfo 2.1 JSON with `software.name: "wanderer"`, a `software.version` string (from `WANDERER_VERSION` or `"dev"`), `usage.users.total` = all-users count, and `usage.localPosts` = public-trail count (ROADMAP success criterion 2).
3. Private trails (`public = false`) are excluded from `localPosts` (D-02, privacy hard constraint).
4. Existing user-level federation routes are unchanged — only two GET registrations added to `registerRoutes()`.
</success_criteria>

<output>
Create `.planning/phases/04-nodeinfo/04-01-SUMMARY.md` when done.
</output>
