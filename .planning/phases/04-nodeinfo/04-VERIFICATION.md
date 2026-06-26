---
phase: 04-nodeinfo
verified: 2026-06-26T00:00:00Z
status: passed
score: 5/5 must-haves verified
overrides_applied: 0
---

# Phase 4: NodeInfo Verification Report

**Phase Goal:** Serve the two NodeInfo well-known HTTP endpoints (SAFE-04) so peer instances can discover and identify this deployment
**Verified:** 2026-06-26
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | GET /.well-known/nodeinfo returns a JSON discovery document with a links array whose rel is `http://nodeinfo.diaspora.software/ns/schema/2.1` | VERIFIED | `buildNodeInfoDiscovery` returns `{"links": [{"rel": "http://nodeinfo.diaspora.software/ns/schema/2.1", "href": origin+"/.well-known/nodeinfo/2.1"}]}`; TestNodeInfoDiscoveryRel and TestNodeInfoDiscoveryHref both PASS |
| 2 | GET /.well-known/nodeinfo/2.1 returns NodeInfo 2.1 JSON with `software.name == "wanderer"` | VERIFIED | `buildNodeInfo21` hardcodes `"name": "wanderer"`; TestNodeInfo21SoftwareName PASS |
| 3 | software.version reflects WANDERER_VERSION env var, or "dev" when unset | VERIFIED | `os.Getenv("WANDERER_VERSION")` with `if version == ""` fallback to `"dev"`; TestNodeInfo21VersionFallback (empty → "dev") and TestNodeInfo21VersionFromEnv ("1.2.3" → "1.2.3") both PASS |
| 4 | usage.localPosts equals the count of public trails | VERIFIED | `app.CountRecords("trails", dbx.NewExp("public = true"))` — private trails excluded by WHERE clause; TestNodeInfo21LocalPostsExcludesPrivate seeds 2 public + 1 private and asserts localPosts == 2, PASS |
| 5 | usage.users.total equals the count of all users | VERIFIED | `app.CountRecords("users", nil)` — nil filter counts all; TestNodeInfo21UsersTotal seeds 3 users and asserts total == 3, PASS |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/routes/nodeinfo.go` | NodeInfo discovery + 2.1 handlers and pure payload builders | VERIFIED | Exists, 92 lines; exports `NodeInfo`, `NodeInfo21`, `buildNodeInfoDiscovery`, `buildNodeInfo21`; imports `github.com/pocketbase/dbx`, `core`, `os`, `fmt`, `net/http` |
| `db/routes/nodeinfo_test.go` | Unit tests for the payload builders against a bootstrapped test app | VERIFIED | Exists, 253 lines; 8 test functions all PASS: `TestNodeInfoDiscoveryRel`, `TestNodeInfoDiscoveryHref`, `TestNodeInfo21SoftwareName`, `TestNodeInfo21VersionFallback`, `TestNodeInfo21VersionFromEnv`, `TestNodeInfo21LocalPostsExcludesPrivate`, `TestNodeInfo21UsersTotal`, `TestNodeInfo21RequiredKeys` |
| `db/main.go` | Route registration for both well-known endpoints | VERIFIED | Lines 189-190: `se.Router.GET("/.well-known/nodeinfo", routes.NodeInfo)` and `se.Router.GET("/.well-known/nodeinfo/2.1", routes.NodeInfo21)`; grep count returns 2 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `db/main.go` `registerRoutes` | `routes.NodeInfo` / `routes.NodeInfo21` | `se.Router.GET("/.well-known/nodeinfo", ...)` and `se.Router.GET("/.well-known/nodeinfo/2.1", ...)` | WIRED | Both registrations confirmed at lines 189-190; existing activitypub routes at lines 192-204 are unchanged |
| `buildNodeInfo21` | trails + users collections | `app.CountRecords("trails", dbx.NewExp("public = true"))` and `app.CountRecords("users", nil)` | WIRED | Both CountRecords calls present in nodeinfo.go lines 56 and 63; tests confirm both return real live counts |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `buildNodeInfo21` | `localPosts` | `app.CountRecords("trails", dbx.NewExp("public = true"))` | Yes — live DB aggregation query, tested with seeded records | FLOWING |
| `buildNodeInfo21` | `usersTotal` | `app.CountRecords("users", nil)` | Yes — live DB aggregation query, tested with seeded records | FLOWING |
| `buildNodeInfo21` | `version` | `os.Getenv("WANDERER_VERSION")` with `"dev"` fallback | Yes — env-driven runtime value | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All NodeInfo builder tests pass | `cd db && go test ./routes/ -run TestNodeInfo -v` | 8/8 PASS, exit 0 | PASS |
| Backend compiles cleanly | `cd db && go build ./...` | Exit 0, no output | PASS |
| Both routes registered in main.go | `grep -c '/.well-known/nodeinfo' db/main.go` | Returns 2 | PASS |

### Probe Execution

No probe scripts declared for this phase. Step 7c: SKIPPED (no probe-*.sh files in this phase).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SAFE-04 | 04-01-PLAN.md | NodeInfo endpoint at `/.well-known/nodeinfo` and `/.well-known/nodeinfo/2.1` returns instance software metadata (name: wanderer, version, user count, post count) | SATISFIED | Both endpoints implemented and registered; all 5 observable truths verified; 8 unit tests pass; build clean |

### Anti-Patterns Found

No debt markers (TBD, FIXME, XXX), placeholder patterns, stub patterns, or hardcoded empty data found in `db/routes/nodeinfo.go` or `db/routes/nodeinfo_test.go`. Both files contain substantive production implementations.

### Human Verification Required

None — all observable truths are verifiable programmatically. Manual smoke test of a running instance (curl against live endpoints) is optional post-deploy but not required to confirm implementation correctness.

### Gaps Summary

No gaps. All 5 must-have truths are VERIFIED, all 3 required artifacts are substantive and wired, both key links are confirmed, data flows from live DB queries, 8 unit tests pass, and the package builds cleanly. SAFE-04 is satisfied.

---

_Verified: 2026-06-26_
_Verifier: Claude (gsd-verifier)_
