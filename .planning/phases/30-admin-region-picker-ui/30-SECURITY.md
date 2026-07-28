---
phase: 30-admin-region-picker-ui
audited: 2026-07-28
asvs_level: 1
block_on: high
verdict: SECURED
threats_total: 16
threats_closed: 16
threats_open: 0
unregistered_flags: 1
register_authored_at_plan_time: true
scope_extension: >
  Register extended beyond plan time at user request — T-30-11 (auth binding on
  post-plan routes), T-30-12 (path traversal on DELETE /region-catalog/{id}/archive),
  T-30-13 (sync-trigger concurrency) cover routes added after 30-01/30-02 were
  authored. All three verified CLOSED in shipped code.
---

# SECURITY.md — Phase 30: admin-region-picker-ui

**Audited:** 2026-07-28
**ASVS Level:** 1
**block_on:** high
**Verdict:** SECURED (16/16 register threats CLOSED; 1 pre-existing/low-severity unregistered flag noted, non-blocking)

Verification method: adversarial code inspection. No mitigation was accepted on the basis of documentation, comments, or plan intent — every CLOSED row below cites a grep-verifiable line in the shipped implementation, re-checked directly by this audit (not merely copied from the plan's own acceptance-criteria greps).

---

## Part 1 — Declared Threat Register (30-01-PLAN.md / 30-02-PLAN.md `<threat_model>`)

### Mitigate dispositions (9)

| Threat ID | Category | Status | Evidence |
|-----------|----------|--------|----------|
| T-30-01 | Spoofing / EoP — superuser auth on `/region-catalog/` | **CLOSED** | `db/main.go:238` — `se.Router.GET("/region-catalog/", routes.RegionsDashboard)` is a bare route, no `.Bind(apis.RequireAuth())`. `db/routes/regions_ext/regions_ui.html:726` reads `localStorage.getItem('__pb_superusers__/_')`; `:759-766` `apiFetch` forwards it as a raw `Authorization` header (grep for `Bearer` = 0 hits, confirming no static/embedded token). Collections `regions`/`region_polygons` confirmed nil `listRule`/`viewRule`/`updateRule` in `db/migrations/1785000000_create_regions_collection.go` (no `Rule` field set at all → PocketBase default = superuser-only) — re-verified no later migration touches these rules (`1784658610`, `1785100000` also both nil-rule / unrelated). |
| T-30-02 | Tampering — clickjacking | **CLOSED** | `db/routes/regions_ui.go:42` `X-Frame-Options: DENY`; `:58` CSP `frame-ancestors 'none'`. |
| T-30-03 | Tampering / Info Disclosure — name/path rendered in tree | **CLOSED** | `grep -c 'x-html' db/routes/regions_ext/regions_ui.html` = 0 (verified directly, not taken from plan). Region names/paths/errors all render via `x-text` (e.g. `:1394` `x-text="rowErrors[row.id]"`, `:1423` `x-text="deleteDialog ? deleteDialog.row.name : ''"`). |
| T-30-04 | EoP — over-broad collection access if rules relaxed | **CLOSED** | Full read of `db/migrations/1785000000_create_regions_collection.go`, `1784658610_created_region_archives.go`, `1785100000_rename_region_archives_region_id_to_path.go` — the only three migrations touching `regions`/`region_polygons`/`region_archives` — confirms `createRule`/`listRule`/`viewRule`/`updateRule`/`deleteRule` are `null` (or unset, same default) throughout; none was ever relaxed. |
| T-30-05 | Info Disclosure / Denial — CSP scope | **CLOSED** | `db/routes/regions_ui.go:50-58` — full CSP string read; scoped to exactly `cdn.jsdelivr.net`, `unpkg.com`, `fonts.googleapis.com`, `fonts.gstatic.com`, `tiles.openfreemap.org`, `worker-src blob:`, `default-src 'none'`. No `*` wildcard present anywhere in the directive string. |
| T-30-06 | Tampering — PATCH body `{enabled}` | **CLOSED** | `regions_ui.html:952` `body: JSON.stringify({ enabled: row.enabled })`; second call site `:1043` `body: JSON.stringify({ enabled: target })` (group-toggle, see Part 2) — both send a literal JS boolean only, never string-interpolated free text. |
| T-30-07 | Spoofing / CSRF — toggle PATCH | **CLOSED** | Same `apiFetch` wrapper (`:759-766`) used for every mutating call (toggle, group-toggle, delete, sync) sets an explicit `Authorization` header from `this.token`, never a cookie. `grep -c 'Bearer'` = 0 and no `document.cookie`/`credentials: 'include'` usage found — confirms no ambient-credential (cookie-session) surface exists for a cross-site forgery to ride on. |
| T-30-08 | Tampering / Injection — region_polygons OR-filter | **CLOSED** | `regions_ui.html:1109` `chunk.map(p => "path='" + p.replace(/'/g, "\\'") + "'").join('||')` — single quotes escaped; `p` sourced from `this.regions` (server-fetched, trusted) never from `filterQuery`. `:1106` `chunkSize = Math.max(1, Math.floor(3400 / (maxPathLen + 10))) || 60` keeps each built filter under the 3500-char cap. Same escape pattern repeated at `:1156` and `:1182` for single-region fetches. |
| T-30-09 | Tampering / Info Disclosure — server error + name rendered inline | **CLOSED** | `regions_ui.html:961` `"Couldn't update" + (data.message ? ': ' + data.message : '') + ' — try again.'` written into `rowErrors[row.id]`, rendered at `:1394` via `x-text` (never `x-html`, see T-30-03). |

### Accept dispositions (3 register entries, 2 distinct claims)

| Threat ID | Category | Status | Evidence |
|-----------|----------|--------|----------|
| T-30-SC (30-01) | Tampering — package-manager installs | **CLOSED** (accepted risk holds) | `git log` for the 30-01/30-02 execution commits (`f77422b0`, `9156f550`, `be08500f`, `4653d115`) shows no `go.mod`/`go.sum`/`package.json`/`package-lock.json` changes. CDN tags confirmed version-pinned: `remixicon@4.5.0` (`:11`), `maplibre-gl@5.24.0` (`:12`, `:1218`), `alpinejs@3.14.8` (`:1217`). |
| T-30-SC (30-02) | Tampering — package-manager installs | **CLOSED** (accepted risk holds) | Same evidence as above; 30-02 added no new CDN tag beyond what 30-01 already pinned. |
| T-30-10 | Info Disclosure / Denial — external `tiles.openfreemap.org` dependency | **CLOSED** (accepted risk holds) | CSP `connect-src`/`img-src` scoped to exactly `https://tiles.openfreemap.org`, no wildcard (`regions_ui.go:56-57`); no API key or secret present in the map init code. |

**Register subtotal: 12/12 CLOSED** (9 mitigate + 3 accept entries, 2 of which are the same accepted claim recorded once per plan file).

---

## Part 2 — Post-plan surface (added after the register was authored; not covered by any threat above)

Per the auditor's explicit mandate, the following routes/files were reviewed for STRIDE issues even though they postdate 30-01-PLAN.md/30-02-PLAN.md's `<threat_model>` block. All were found to have a mitigation already present in the shipped code; none was blindly trusted on the basis of a docstring.

| Threat ID | Category | Component | Status | Evidence |
|-----------|----------|-----------|--------|----------|
| T-30-11 | Elevation of Privilege — new custom Go routes have no PocketBase collection rules of their own, so must self-enforce auth | **CLOSED** | `db/main.go:244` `se.Router.DELETE("/region-catalog/{id}/archive", routes.RegionArchiveDelete).Bind(apis.RequireSuperuserAuth())`; `:250-251` `POST`/`GET /region-catalog/sync` both `.Bind(apis.RequireSuperuserAuth())`. Confirmed `GET /region-catalog/` (`:238`) remains deliberately unbound per T-30-01's design (page content, not the route, is the auth boundary) — the asymmetry is intentional and correctly applied, not an oversight. All three new mutating/status routes are called client-side only through the same `apiFetch` (`Authorization` header, no cookie) already verified under T-30-07, so the CSRF argument extends to them without a separate bypass path. |
| T-30-12 | Tampering — Path Traversal via `{id}` reaching filesystem ops | **CLOSED** | `db/routes/regions_delete.go:31-32` `id := e.Request.PathValue("id"); if !regions.IsValidRegionID(id) { return e.BadRequestError(...) }` gates *before* `regions.RegionArchivePath(id)`/`RegionDemPath(id)` (`:36,39`) or the `region_archives` DB filter (`:48`) are ever built. `db/services/regions/config.go:33` allow-list regex `^[a-z0-9][a-z0-9_.'-]*$` — no `/` in the character class (blocks traversal via path separator) and leading-char restricted to `[a-z0-9]` (blocks a leading `.`/`-`); `:40` additionally hard-rejects any `..` substring as defense-in-depth. Same guard reused unchanged by the pre-existing `RegionArchiveDownload`/`RegionArchiveDownloadDem` (`db/routes/regions_get.go:156,171`), so the delete route's traversal posture is consistent with the rest of the codebase, not a weaker one-off. Client-side, `regions_ui.html:975` `encodeURIComponent(row.path)` sources the id from server-fetched, already-validated data, never from user-typed text. |
| T-30-13 | Denial of Service — repeated/concurrent invocation of `POST /region-catalog/sync` spawning overlapping `BuildAllLocked` passes | **CLOSED** | `db/services/regions/builder.go:54-66` `TryStartSync()`/`FinishSync()` — a mutex-guarded single-flight slot. `db/routes/regions_sync.go:29-31` claims the slot **synchronously**, before the build goroutine is spawned (`:33-36`), and returns `409 Conflict` immediately if a pass is already running — a second click (or a click racing the nightly cron) cannot start a second concurrent pass. `builder.go` additionally dedupes any single-region overlap via the independent `inFlight` map (`:34-52`), an orthogonal but reinforcing guard. |

### Unregistered flags (informational, non-blocking)

| ID | Category | Component | Assessment |
|----|----------|-----------|------------|
| T-30-14 | Information Disclosure — build status/error surfaced to any authenticated app user, not just superusers | `db/routes/regions_get.go:117,130-131` `entry["status"] = status; ...; entry["error"] = archive.GetString("error_message")` returned by `GET /regions` (proxied publicly as `/api/v1/regions`), gated only by `apis.RequireAuth()` (`db/main.go:267`, any logged-in end user) — not superuser-only. **Severity: LOW.** This code and its auth posture predate Phase 30 (Phase 21.5/29) and were not modified by either 30-01 or 30-02 — the admin UI's own archive-size/error display goes through the separate superuser-gated PocketBase collection REST API (`region_archives` records, nil rule, confirmed under T-30-04), not through this endpoint. The disclosed `error_message` originates from `exec.CommandContext(...).Run()` failures (`db/services/regions/builder.go:328,388`), whose `Stderr` is redirected to the **server's own** `os.Stderr` (`:326,386`) — the returned Go `error` for a non-zero exit is typically just `"exit status 1"`, not full subprocess output, so no internal file paths or command details are actually leaked in practice; the field mainly discloses "a build failed for region X" to any logged-in user. Not a Phase-30 regression and not high-severity — **flagged for the record, not a blocker**. Recommendation: if this disclosure needs tightening, do so under a Phase 21.5/29 security follow-up (out of this phase's mitigate/accept scope), or explicitly record it as an accepted risk in a future SECURITY.md revision. |

---

## Summary

| Metric | Value |
|--------|-------|
| Register threats (mitigate) | 9/9 CLOSED |
| Register threats (accept) | 3/3 CLOSED (2 distinct claims) |
| Post-plan threats identified & verified (T-30-11..13) | 3/3 CLOSED |
| Unregistered flags (non-blocking) | 1 (T-30-14, LOW severity, pre-existing) |
| **Threats open (block_on: high)** | **0** |

No BLOCKER-level gap was found. Every `mitigate` threat has a grep-verifiable, line-cited implementation of its declared control; both `accept` dispositions still hold under the stated scope (no new npm/go installs; CSP scoped to exactly `tiles.openfreemap.org`, no wildcard). The three post-plan routes (`regions_delete.go`, `regions_sync.go`) added after the register was authored were independently checked against path-traversal, missing-auth, and concurrency-abuse STRIDE categories and found to already carry adequate mitigations in the shipped code — not merely described in comments. One low-severity, pre-existing information-disclosure flag (T-30-14) is logged for visibility; it does not block this phase under `block_on: high`.
