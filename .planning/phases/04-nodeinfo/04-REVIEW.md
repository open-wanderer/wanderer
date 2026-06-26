---
phase: 04-nodeinfo
reviewed: 2026-06-26T13:10:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - db/routes/nodeinfo.go
  - db/routes/nodeinfo_test.go
  - db/main.go
findings:
  critical: 2
  warning: 4
  info: 2
  total: 8
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-06-26T13:10:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Three files reviewed: the NodeInfo route handler (`nodeinfo.go`), its test suite (`nodeinfo_test.go`), and the main application entry point (`main.go`). The NodeInfo implementation is functionally straightforward — a two-endpoint JRD discovery + NodeInfo 2.1 payload — but contains one correctness defect in the SQL filter and several quality defects in `main.go` that predate this phase but are in scope because `main.go` is listed as a changed file.

Key concerns:
1. The `public = true` SQL fragment in `buildNodeInfo21` is SQLite-dialect-unsafe and will silently return zero posts on SQLite when the boolean is stored as `0`/`1` (not the string `true`).
2. `e.JSON()` in PocketBase overwrites response headers it sets itself, meaning the custom `Content-Type` profile header is silently discarded.
3. `initData()` return error is unconditionally ignored at the call site in `main.go`.
4. `initCategories()` returns `error` but the return value is ignored at its only call site.

---

## Critical Issues

### CR-01: SQLite boolean literal `public = true` will evaluate incorrectly

**File:** `db/routes/nodeinfo.go:56`
**Issue:** PocketBase uses SQLite as its underlying database. SQLite does not have a native boolean type; boolean columns are stored as integers (`0` for false, `1` for true). The expression `dbx.NewExp("public = true")` compiles to the raw SQL fragment `public = true`. In SQLite, the bare identifier `true` evaluates to `1` only in SQLite 3.23.0+ (released 2018). On older SQLite versions, `true` is treated as a column name and the query silently returns 0 rows. Even on modern SQLite, relying on unquoted `true` instead of the established project pattern of `public = 1` (used in migrations and views) is fragile. All other references in the codebase use `public = 1` in raw SQL contexts:

```go
// migrations/1778583475_updated_trails.go:22
"CREATE INDEX idx_trails_public ON trails (public) WHERE public = true;"
// migrations/1770000002_optimize_trails_filter_view.go:102
//   trails.public = 1 OR
```

The migration index uses `public = true` at DDL time (accepted by SQLite 3.23+), but query-time expressions in older embedded SQLite can silently mismatch. More importantly, the PocketBase `dbx` expression does not go through the ORM's type coercion, so the safest and most consistent form is:

```go
// Fix: use integer literal to match SQLite storage and established codebase pattern
localPosts, err := app.CountRecords("trails", dbx.NewExp("public = 1"))
```

### CR-02: NodeInfo endpoints unreachable in production — PocketBase port is not public

**Files:** `db/routes/nodeinfo.go`, `db/main.go`
**Issue:** The two NodeInfo routes are registered on PocketBase's internal HTTP router (`se.Router.GET`). In Wanderer's deployment topology, PocketBase runs on a separate port (default 8090) that is not exposed publicly — only SvelteKit (port 3000/80) is the public-facing server. Any peer instance that probes `https://your-instance.example.com/.well-known/nodeinfo` will get a 404 from SvelteKit, because the route does not exist there.

The established pattern for this exact problem is `web/src/routes/.well-known/webfinger/+server.ts`, which implements the Webfinger well-known endpoint directly in SvelteKit, querying PocketBase through `event.locals.pb`. NodeInfo must follow the same pattern.

**Fix:** Add two SvelteKit route files:

`web/src/routes/.well-known/nodeinfo/+server.ts` — returns the static JRD discovery document (no PocketBase needed):
```typescript
import { json, type RequestEvent } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';

export async function GET(_event: RequestEvent) {
    return json({
        links: [{
            rel: 'http://nodeinfo.diaspora.software/ns/schema/2.1',
            href: `${env.ORIGIN}/.well-known/nodeinfo/2.1`,
        }],
    });
}
```

`web/src/routes/.well-known/nodeinfo/2.1/+server.ts` — queries PocketBase for live counts via `event.locals.pb`:
```typescript
import { json, type RequestEvent } from '@sveltejs/kit';
import { env } from '$env/dynamic/private';
import { handleError } from '$lib/util/api_util';

export async function GET(event: RequestEvent) {
    try {
        const version = env.WANDERER_VERSION || 'dev';
        const [trailsResult, usersResult] = await Promise.all([
            event.locals.pb.collection('trails').getList(1, 1, { filter: 'public = true', fields: 'id' }),
            event.locals.pb.collection('users').getList(1, 1, { fields: 'id' }),
        ]);
        return json({
            version: '2.1',
            software: { name: 'wanderer', version, homepage: 'https://wanderer.to', repository: 'https://github.com/Flomp/wanderer' },
            protocols: ['activitypub'],
            services: { inbound: [], outbound: [] },
            openRegistrations: false,
            usage: { users: { total: usersResult.totalItems }, localPosts: trailsResult.totalItems },
            metadata: {},
        });
    } catch (err) {
        return handleError(err);
    }
}
```

The Go handlers in `db/routes/nodeinfo.go` and their registrations in `db/main.go` can be kept for internal use but are not the public path and should be noted as such, or removed to avoid confusion.

---

## Warnings

### WR-01: `e.JSON()` overwrites the `Content-Type` header set immediately before it

**File:** `db/routes/nodeinfo.go:27-28`
**Issue:** `NodeInfo21` sets a custom `Content-Type` with a `profile=` parameter (required by the NodeInfo spec so clients can identify the schema), then calls `e.JSON()`. PocketBase's `e.JSON()` implementation calls `e.Response.Header().Set("Content-Type", "application/json")` internally before writing the body, overwriting the profile header. The result is that consumers receive `application/json` instead of the spec-mandated `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`.

```go
// Current (broken ordering):
e.Response.Header().Set("Content-Type", `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`)
return e.JSON(http.StatusOK, payload) // overwrites Content-Type

// Fix: write JSON manually so you control the header
b, err := json.Marshal(payload)
if err != nil {
    return err
}
e.Response.Header().Set("Content-Type", `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`)
e.Response.WriteHeader(http.StatusOK)
_, err = e.Response.Write(b)
return err
```

No test covers the response `Content-Type` value, so this defect is invisible to the test suite.

### WR-02: `initData()` return error silently ignored

**File:** `db/main.go:161`
**Issue:** `initData()` has the signature `func initData(app core.App, client meilisearch.ServiceManager) error` but the return value is discarded at the call site:

```go
// line 161
initData(se.App, client)   // error silently dropped
```

Errors from `initData` include failures from `federation.InitInstanceActor` (which sets up the instance actor required for federation) and `initMeilisearchDocuments`. A failure here means the instance actor is not initialized and federation will silently not work, with no indication in logs from this call site.

```go
// Fix:
if err := initData(se.App, client); err != nil {
    se.App.Logger().Error(fmt.Sprintf("initData failed: %v", err))
    // Consider: return err to prevent serving traffic without a valid state
}
```

Note: `initData` currently always returns `nil` (the internal error paths log and continue), but the signature promises otherwise and a future change will introduce a latent silent failure.

### WR-03: `initCategories()` return error silently ignored

**File:** `db/main.go:224`
**Issue:** `initCategories()` has the signature `func initCategories(app core.App) error` but the return value is discarded:

```go
// line 224
initCategories(app)  // error silently dropped
```

If category initialization fails (e.g., the `categories` collection does not exist due to a failed migration), the application continues to start with an empty categories table and no diagnostic message. This is a silent data initialization failure.

```go
// Fix:
if err := initCategories(app); err != nil {
    app.Logger().Error(fmt.Sprintf("initCategories failed: %v", err))
}
```

### WR-04: `verifySettings` called before `app.Start()` but `app.Logger()` may not be initialized

**File:** `db/main.go:66`
**Issue:** `verifySettings(app)` is called at line 66, before `app.Start()` at line 73. PocketBase's `app.Logger()` is available only after the app has been bootstrapped (which occurs inside `app.Start()`). The `verifySettings` function calls `app.Logger().Warn(...)` for the default-key warnings (lines 44, 55). If the logger is not yet initialized, these warn calls may panic or silently no-op depending on the PocketBase version.

```go
// main() order:
app := pocketbase.New()     // line 63 — logger not yet ready
verifySettings(app)         // line 66 — calls app.Logger().Warn() BEFORE Start()
...
if err := app.Start(); err != nil {  // line 73 — bootstraps logger
```

The `log.Fatal` path (line 41) is safe because it uses the standard library logger, not the PocketBase logger. Only the `.Warn()` calls (lines 44, 55) are at risk.

```go
// Fix option A: move verifySettings() call into OnBootstrap or OnServe handler
// Fix option B: replace app.Logger().Warn() with log.Printf() for pre-start warnings
```

---

## Info

### IN-01: `NodeInfo` handler returns a raw `fmt.Errorf` — no HTTP status code set for the error case

**File:** `db/routes/nodeinfo.go:16-17`
**Issue:** When `ORIGIN` is not set, the handler returns a plain Go error:

```go
return fmt.Errorf("ORIGIN not set")
```

PocketBase will translate this to a 500 Internal Server Error with a generic message. This is acceptable for a misconfiguration scenario, but other handlers in the codebase (e.g., `db/util/activitypub.go`) use PocketBase's `apis.NewBadRequestError` or similar for more precise status codes. Consistency aside, this is low severity because `ORIGIN` being unset is a deployment error, not a user-triggerable condition.

### IN-02: No test covers the `NodeInfo` HTTP handler (only the builder functions are tested)

**File:** `db/routes/nodeinfo_test.go`
**Issue:** All tests exercise `buildNodeInfoDiscovery` and `buildNodeInfo21` directly. There are no tests that exercise the `NodeInfo` or `NodeInfo21` HTTP handlers through a test router, meaning:

- The `ORIGIN == ""` error path in `NodeInfo` is never tested.
- The `Content-Type` profile header in `NodeInfo21` (see WR-01) is never tested and is in fact broken without detection.

This is an info-level gap because the builder coverage is meaningful, but the header overwrite defect (WR-01) exists precisely because no HTTP-level test would catch it.

---

_Reviewed: 2026-06-26T13:10:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
