---
phase: 04-nodeinfo
fixed_at: 2026-06-26T13:30:00Z
review_path: .planning/phases/04-nodeinfo/04-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 04: Code Review Fix Report

**Fixed at:** 2026-06-26
**Source review:** .planning/phases/04-nodeinfo/04-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (2 Critical, 4 Warning)
- Fixed: 6
- Skipped: 0

## Fixed Issues

### CR-01: SQLite boolean literal `public = true` → `public = 1`

**Files modified:** `db/routes/nodeinfo.go`
**Commit:** `03f8002a`
**Applied fix:** Changed `dbx.NewExp("public = true")` to `dbx.NewExp("public = 1")` in `buildNodeInfo21`. SQLite stores booleans as integers; `true` as an identifier is unreliable on SQLite < 3.23 and diverges from the established codebase pattern used in migrations and views.

---

### CR-02: NodeInfo endpoints unreachable — added SvelteKit proxy routes

**Files modified:** `web/src/routes/.well-known/nodeinfo/+server.ts` (new), `web/src/routes/.well-known/nodeinfo/2.1/+server.ts` (new)
**Commit:** `ebe2839d`
**Applied fix:** Created two SvelteKit well-known routes. The discovery route (`/+server.ts`) returns the static JRD document built from `env.ORIGIN`. The 2.1 route (`/2.1/+server.ts`) proxies to `PUBLIC_POCKETBASE_URL/.well-known/nodeinfo/2.1` via `event.fetch`, delegating count queries to the Go handler which uses `app.CountRecords` with admin-level DB access — bypassing PocketBase collection rules that block unauthenticated reads of the `users` collection. Import style follows the `webfinger` route pattern (`// @ts-ignore` on `$env` and `$lib` imports).

---

### WR-01: `Content-Type` profile header overwritten by `e.JSON()`

**Files modified:** `db/routes/nodeinfo.go`
**Commit:** `48444556`
**Applied fix:** Replaced `e.JSON(http.StatusOK, payload)` in `NodeInfo21` with manual `json.Marshal` + `e.Response.Header().Set(...)` + `e.Response.WriteHeader(200)` + `e.Response.Write(b)`. This preserves the `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"` Content-Type header, which `e.JSON()` was silently overwriting with plain `application/json`.

---

### WR-02: `initData()` return error silently ignored

**Files modified:** `db/main.go`
**Commit:** `7280296c`
**Applied fix:** Wrapped the `initData(se.App, client)` call site with `if err := initData(...); err != nil { se.App.Logger().Error(...) }` so failures from `federation.InitInstanceActor` and related startup work are logged rather than silently dropped.

---

### WR-03: `initCategories()` return error silently ignored

**Files modified:** `db/main.go`
**Commit:** `b25a3916`
**Applied fix:** Wrapped the `initCategories(app)` call site with `if err := initCategories(app); err != nil { app.Logger().Error(...) }` so category initialization failures are logged.

---

### WR-04: `verifySettings` calls `app.Logger()` before bootstrap

**Files modified:** `db/main.go`
**Commit:** `595d146b`
**Applied fix:** Replaced `app.Logger().Warn(...)` calls inside `verifySettings` with `log.Printf(...)` (stdlib logger, always available). Removed the now-unused `app core.App` parameter from `verifySettings` to keep the Go compiler happy. Updated the single call site accordingly.

---

_Fixed: 2026-06-26_
_Fixer: Claude (gsd-code-fixer)_
_Scope: critical_warning_
