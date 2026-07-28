---
phase: 31-flutter-settings-hierarchy
audited: 2026-07-28
asvs_level: 1
block_on: high
verdict: SECURED
threats_total: 12
threats_closed: 12
threats_obsolete: 1
threats_open: 0
register_authored_at_plan_time: true
scope_extension: "T-31-07..T-31-12 added by this audit (server-side rewrite, commit 3667e058, post-dates the plan-time register) — user-authorized extension, not a plan-time threat_model entry."
---

# Phase 31 Security Audit — flutter-settings-hierarchy

## Context

The plan-time threat register (31-01/31-02/31-03-PLAN.md `<threat_model>` blocks) was authored against a **client-side filtering design**: a leaf-only `enabled` field parsed onto `RegionHierarchyRow`/`RegionTreeNode`, pruned in Dart via `pruneToDownloadable`. Commit `3667e05877ab5e2879495c8b3c0e1e553f02efe1` ("feat(31): filter region catalog to enabled regions server-side", 2026-07-28) **replaced** this design: the backend now honors `?enabled=true` and does the pruning in Go (`db/services/regions/hierarchy_filter.go`); the client sends the query param and renders `buildRegionTree` output directly. `pruneToDownloadable`, the `enabled` field on `RegionHierarchyRow`/`RegionTreeNode`, and their ~160 lines of tests were deleted.

Verified deletion (confirms the `<critical_context>` claim, not accepted on faith):
```
$ grep -rn "pruneToDownloadable" app/          -> no matches
$ grep -n "enabled" app/lib/models/region_hierarchy_row.freezed.dart app/lib/models/region_hierarchy_row.g.dart -> no matches
$ grep -n "enabled" app/lib/models/region_tree_node.dart -> no matches
```

This audit verifies the surviving plan-time mitigations against current code, marks the deleted-component threats OBSOLETE, and — per explicit user authorization — extends the register to the unreviewed server-side rewrite (T-31-07 through T-31-12).

No SUMMARY.md in this phase contains a `## Threat Flags` section (`grep -rn "Threat Flags" 31-0{1,2,3}-SUMMARY.md` → no matches), so there is no executor-declared new-attack-surface list to cross-reference; the server-side rewrite's attack surface was entirely undeclared prior to this audit.

## Threat Verification

| Threat ID | Category | Disposition | Status | Evidence |
|-----------|----------|-------------|--------|----------|
| T-31-01 | Denial of Service | mitigate | **CLOSED** | `app/lib/provider/region/region_provider.dart:63-77` — `parseRegionHierarchyRows` still throws `RegionCatalogException` on non-`List`, and wraps every element in `try { RegionHierarchyRow.fromJson(...) } catch (_) {}` (skip, don't abort). Proven by passing tests: `flutter test test/provider/region_provider_test.dart` → `parseRegionHierarchyRows a malformed element (missing required id) is skipped, not fatal` PASS. Note: the 31-03 revision of this threat (nullable `enabled` parse) is moot — see OBSOLETE note below — but the core per-element resilience mitigation is untouched by the rewrite and independently verified. |
| T-31-02 | Denial of Service | mitigate | **CLOSED** | `app/lib/routes/settings_offline_regions_screen.dart:291-296` — `final entity = byId[row.node.id]; if (entity == null) { return SizedBox.shrink(key: ValueKey(row.node.id)); }`. `flutter test test/routes/settings_offline_regions_screen_test.dart` passes (4/4 including the pruning/no-entity-match scenarios). |
| T-31-03 | Information Disclosure | accept | **CLOSED** | `db/routes/regions_get.go:83` still emits `"sort_order": r.GetInt("sort_order")` on every row. `/regions` group confirmed `apis.RequireAuth()`-gated at `db/main.go:266-267` (unchanged by the rewrite — filtering is an additional query-param branch inside the same auth-gated handler). Accepted-risk entry logged below. |
| T-31-04 | Injection (V5) | accept | **CLOSED** | `app/lib/util/region_tree_util.dart:133` — `node.name.toLowerCase().contains(q)`, still the sole use of the search query; no SQL/shell/template sink. `_searchQuery` flows only into `computeFilterMatches` (`settings_offline_regions_screen.dart:173-176`). |
| T-31-05 | Tampering (path traversal) | accept | **CLOSED** | `db/services/regions/config.go` `IsValidRegionID` (regex allow-list) still gates `db/routes/regions_get.go:156,171` and `db/routes/regions_delete.go:32`; `app/lib/util/region_file_path.dart:27` `assertValidRegionId` still gates every on-disk path builder in `tile_repository_manager.dart` (5 call sites). Neither this phase nor the rewrite touched these guards. |
| T-31-06 | Tampering | accept | **OBSOLETE** | `pruneToDownloadable` (the mitigated component) was deleted by `3667e058` — confirmed zero remaining references in `app/`. The pruning responsibility moved server-side; its replacement is assessed fresh as T-31-11 below (not a like-for-like carry-forward, since the mutation target, trust boundary, and code changed entirely). |
| T-31-SC | Tampering (package installs) | accept | **CLOSED** | `cd db && go build ./...` exits 0 with no `go.mod`/`go.sum` changes in `3667e058` (`git show --stat` confirms). No `pubspec.yaml`/`pubspec.lock` changes across any Phase 31 commit. Zero new packages, as declared. |
| T-31-07 | Injection / Tampering | mitigate (new) | **CLOSED** | `db/routes/regions_get.go:48` — `filtering := e.Request.URL.Query().Get("enabled") == "true"`. This is a Go string **equality comparison**, never interpolated into a PocketBase filter expression, `dbx.Params`, or any query string. Any value other than the literal `"true"` (including absent) silently falls through to the unfiltered branch — no error oracle, no injection sink. Traced the full data flow: query param → `bool` → conditional branch only (lines 53-74); never reaches `FindAllRecords`'s filter arguments (that call at line 38 takes no filter expression at all). |
| T-31-08 | Denial of Service | accept (new) | **CLOSED** | `db/services/regions/hierarchy_filter.go:19-30` `AncestorGroupPaths` does `strings.Split`/`strings.Join` per leaf path with no length/depth cap — worst case O(n²) in path-segment count per leaf. Input (`regions.path`) is trust-bounded: the `regions` PocketBase collection has no `listRule`/`createRule`/`updateRule`/`deleteRule` set in `db/migrations/1785000000_create_regions_collection.go` (defaults to **superuser-only**), and the only custom Go routes touching `regions` records (`regions_delete.go`, `region-catalog/sync`) are both `apis.RequireSuperuserAuth()`-gated (`db/main.go:244,250-251`) and neither mutates `path`. No authenticated-but-non-superuser caller can influence `path` length/depth. Real-world cost is bounded by the ~1,306-row seeded CoMaps catalog. Accepted-risk entry logged below, with a forward-looking recommendation. |
| T-31-09 | Information Disclosure / Access Control | accept (new, non-regression) | **CLOSED** | Confirmed omitting `?enabled=true` still returns the full, unfiltered catalog (~1,306 group+leaf rows) to any `RequireAuth()`-gated (non-superuser) user — but this is **pre-existing**, not introduced by the rewrite: `regions_get.go`'s own doc comment ("Every row... is returned, regardless of `enabled`... trimming is a trivial later filter") describes behavior that predates Phase 31 (Phase 29's `RegionsList`), and the identical disclosure posture was already logged as `T-30-14` in `.planning/phases/30-admin-region-picker-ui/30-SECURITY.md:72` (LOW severity, "not a Phase-30 regression"). `?enabled=true` only narrows the response; it introduces no new disclosure. Carried forward as non-blocking, consistent with the prior audit's disposition. |
| T-31-10 | Information Disclosure | accept (new) | **CLOSED** | `db/routes/regions_get.go:101` — `entry["enabled"] = r.GetBool("enabled")` unconditionally on every leaf row. Same rationale as T-31-03: non-sensitive admin-config boolean, already readable via the superuser PocketBase collection API, no new auth surface (same `RequireAuth()` route, same handler). |
| T-31-11 | Tampering | mitigate (new, successor to T-31-06) | **CLOSED** | The prune logic that moved server-side operates entirely on **per-request local variables**: `entries := make([]map[string]any, 0, len(records))` (`regions_get.go:64`) and `groupPaths := regions.AncestorGroupPaths(leafPaths)` (`regions_get.go:61`) — both freshly allocated per HTTP request, never stored on a shared/package-level variable, never mutating the `records` slice from `FindAllRecords` in place. `AncestorGroupPaths` (`hierarchy_filter.go`) is a pure function with no shared state, unit-tested (`go test ./services/regions/...` → PASS, 5 cases including sibling-dedup and unrelated-branch isolation). No cross-request tampering surface. |
| T-31-12 | SSRF / Param Smuggling / Auth Bypass | mitigate (new) | **CLOSED** | `web/src/routes/api/v1/regions/+server.ts:100` — `pb.send('/regions?' + event.url.searchParams, ...)`. (a) Base path `/regions` is a hardcoded literal; `searchParams.toString()` can only append a query string after the literal `?`, never alter the path or inject a host — no path/host injection. (b) `pb.send` targets the SvelteKit backend's fixed internal PocketBase base URL, never an attacker-suppliable one — no SSRF. (c) Auth is carried by `event.locals.pb`'s authStore, loaded from the request's own session cookie in `hooks.server.ts:80` (`pb.authStore.loadFromCookie(...)`); an unauthenticated caller forwards no valid token and is rejected by the backend's `apis.RequireAuth()` (`db/main.go:267`) — the proxy adds no bypass, it is a dumb passthrough as its own comment states. |

## Accepted Risks Log

The following threats are formally accepted (not mitigated by code) as of this audit, per ASVS Level 1 / `block_on: high` — none rise to HIGH severity:

1. **T-31-03 / T-31-10** — `sort_order` (int) and `enabled` (bool, leaf-only) are non-sensitive catalog metadata disclosed to any authenticated user via `GET /api/v1/regions`. Already readable by the same population via the superuser-gated admin path is irrelevant (regular users don't have that path) — but the fields carry no secret/PII value (an ordering hint and an admin toggle state for a public regional map catalog), and the route's auth posture (`apis.RequireAuth()`) is unchanged. **Severity: LOW. Non-blocking.**
2. **T-31-08** — `AncestorGroupPaths`'s unbounded prefix-splitting has no length/depth cap, but `regions.path` is superuser-write-only (no collection API rule configured = superuser-only default; no non-superuser Go route mutates it). **Severity: LOW (trust-boundary-bounded). Non-blocking.** *Recommendation:* if `regions` write access is ever extended to non-superuser roles (e.g., a future multi-tenant admin), add a defensive max-segment-count guard to `AncestorGroupPaths` before that ships.
3. **T-31-09** — `GET /api/v1/regions` without `?enabled=true` returns the full seeded catalog (including disabled/building leaves) to any authenticated user, not just superusers. Confirmed **pre-existing** (predates Phase 31; already logged as `T-30-14` in `30-SECURITY.md`), **not a regression** introduced by this rewrite. **Severity: LOW. Non-blocking, carried forward for the record.**
4. **T-31-05 / T-31-SC / T-31-04** — carried forward unchanged from the plan-time register; rationale re-verified against current code (see table above).

## Obsolete Mitigations

| Threat ID | Plan-time component | Deletion evidence |
|-----------|---------------------|--------------------|
| T-31-06 | `pruneToDownloadable` in-place `children.retainWhere` mutation (`app/lib/util/region_tree_util.dart`) | `git show 3667e058 --stat` shows `app/lib/util/region_tree_util.dart \| 33 -------` (net removal); `grep -rn pruneToDownloadable app/` → zero matches. Superseded by T-31-11 (server-side equivalent), independently assessed and CLOSED. |

## Verification Performed (not just grep)

- `cd db && go build ./...` — exits 0.
- `cd db && go test ./services/regions/...` — PASS (`TestAncestorGroupPaths`, 5 subtests).
- `cd app && flutter analyze lib/models/region_hierarchy_row.dart lib/models/region_tree_node.dart lib/util/region_tree_util.dart lib/provider/region/region_provider.dart lib/routes/settings_offline_regions_screen.dart` — No issues found.
- `cd app && flutter test test/util/region_tree_util_test.dart test/provider/region_provider_test.dart test/routes/settings_offline_regions_screen_test.dart` — 32/32 passed, including the T-31-01/T-31-02 resilience assertions exercised at runtime (not just present in source).

## Unregistered Flags

None outstanding. T-31-07 through T-31-12 were unregistered new attack surface (server-side rewrite, commit `3667e058`) prior to this audit — no SUMMARY.md `## Threat Flags` section existed to declare them (verified absent in all three plan summaries), and no threat_model entry covered them. Per explicit user authorization, they have been registered, investigated, and resolved to CLOSED above rather than left as open WARNINGs.

## Verdict

**SECURED** — 11/12 registered threats CLOSED, 1/12 OBSOLETE (successor threat independently CLOSED). Zero OPEN threats. Zero HIGH-severity findings. `block_on: high` is satisfied.
