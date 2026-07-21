# Phase 22: Region & Package Data Model - Context

**Gathered:** 2026-07-21
**Status:** Ready for planning (replaces prior CONTEXT.md — discarded due to Phase 21.5 backend redesign)

<domain>
## Phase Boundary

This phase delivers the foundation data model for v1.6's offline region tile repository: a typed fetch-and-upsert function for Phase 21.5's `GET /api/v1/regions` catalog API, plus ObjectBox `Region` and `DownloadedTilePackage` entities. Nothing downstream (download engine, UI, rendering) reads from these entities yet — the app builds and runs unchanged. Purely additive; zero UI; no fetch call-site wiring (that's Phase 24's concern — this phase only needs the fetch/upsert function to exist and be callable).

**Supersedes:** The previous 22-CONTEXT.md (2026-07-21, same day) assumed a bundled `assets/map/regions.json` app asset. Phase 21.5 was inserted afterward and established that regions are instead an admin-configured, per-instance catalog fetched at runtime from a new backend API. See `.planning/notes/region-catalog-backend-decision-trail.md` and `.planning/todos/pending/replan-phase-22-region-manifest.md` for the full reasoning. This discussion re-derives Phase 22's decisions against that new contract.

</domain>

<decisions>
## Implementation Decisions

### Catalog Fetch & Merge Strategy
- **D-01:** Region rows are **upserted by id**, not replaced wholesale. On a successful catalog fetch, find-or-create each `Region` by its catalog `id`; update catalog-owned fields (`name`, `bbox`, `catalogStatus`, `version`, `vectorUrl`/`vectorSize`, `demStatus`/`demUrl`/`demSize`, `error`) in place. Never touch local-only fields (download status, `ToOne` package links) during this upsert.
  - **Rejected:** the existing `subcategory_provider.dart`/`category_provider.dart` pattern of `box.removeAll()` + `box.putMany()` on every refresh. That destroys `ToOne` package links and any local download status, since ObjectBox relations and rows are keyed by internal `obxId` which `removeAll()` invalidates. Explicitly not following this established app pattern here — flag this deviation for the planner/researcher.
- **D-02:** This phase builds the fetch-and-upsert function only (e.g. a `RegionRepository`/similar with a `refreshCatalog()` method) — it is not wired to any call site or Riverpod provider yet. Phase 24 (Settings — Offline Maps/Regions UI) decides when it's actually invoked; current intent leans toward **on-demand fetch when the Settings/Regions screen opens**, not an eager app-launch background refresh like `subcategory_provider`'s `build()`-triggers-refresh shape — but this phase should not hard-code that assumption into the function's design (keep it a plain callable, not tied to screen lifecycle).
- **D-03:** On fetch failure (offline, 401, backend error), existing local `Region` rows are left completely untouched — but unlike `subcategory_provider`'s silent `catch (_) {}`, this phase's fetch function should **not swallow the error internally**. It should throw/return a typed error so Phase 24's screen can decide how to surface it (this phase has no UI, so there's no natural place to swallow to yet).

### Backend Build-Status vs Local Download-Status
- **D-04:** These are two distinct concepts and get two distinct fields — never merged into one enum:
  - `Region.catalogStatus` — mirrors the backend's `status` field (`building`/`ready`/`error`) verbatim from the last successful fetch. Answers "is this region's archive available to download yet?"
  - `Region.status` (existing computed getter from D-07 below) — the local download lifecycle (`notDownloaded`/`downloading`/`downloaded`/`updateAvailable`). Answers "has the user downloaded this?"
- **D-05:** `catalogStatus` follows the **same explicit-int-enum persistence pattern** as the local `RegionStatus` (D-01/D-02 from the original discussion, still valid) — an enhanced enum with an explicit `code` int field, persisted via the shadow-property pattern, reading/writing `.code` never `.index`. Consistency with REGN-02's anti-pattern warning was judged to matter more than the fact that this field is fully overwritten on every fetch.

### Staleness → updateAvailable Mapping
- **D-06:** `Region` persists a `lastDownloadedVersion` string field — set to the catalog's `version` value (the vector archive's Protomaps build-date string) at the moment a vector download completes successfully. On every catalog upsert (D-01), if `catalogStatus == ready` AND a vector package is already downloaded AND the newly-fetched `version` differs from `lastDownloadedVersion`, the local download status resolves to `updateAvailable`. Plain string comparison, mirroring the backend's own D-10 staleness mechanism (`21.5-CONTEXT.md`).
- **D-07 (note, not a question — derived directly from the API contract):** The backend's `GET /api/v1/regions` response has **no DEM-equivalent version field** (`dem_status` exists, but no `dem_version`/`dem_built_date`) — confirmed by reading `db/routes/regions_get.go`. This matches Phase 21.5's D-11: DEM archives never auto-rebuild except on a config bbox change, so there is no DEM staleness/`updateAvailable` concept in this phase. Only the vector package can go stale.

### Region Removed from Catalog / Partial DEM Handling
- **D-08:** `Region` gains an `inCatalog: bool` field, defaulted `true` on creation and flipped `false` when a fetch completes without that id appearing in the response. Downloaded files/packages for an orphaned region are **left on disk untouched** — deleting a user's offline map because an admin edited a config file would be a surprising, destructive side effect for a data-model-only phase to introduce. Later phases (Settings UI / trail guard) decide what to show/offer for `inCatalog: false` regions.
  - **Rejected:** full referential sync (delete `Region` + `DownloadedTilePackage` rows and files immediately when a region drops out of the catalog response). Simpler invariant, but destroys user data based on an unrelated admin action with no warning — rejected for that reason.
- **D-09:** No `DownloadedTilePackage` row is created for a DEM (or vector) that isn't ready yet or hasn't been downloaded — package rows only come into existence when Phase 23's download engine actually begins downloading that specific package. `Region.demPackage` stays `null` for the entire time a region's DEM is `building`, `error`'d, or simply absent from the catalog response (no DEM configured for that region). The catalog-level `demStatus`/`demUrl`/`demSize` fields (present or absent per-fetch) are the only signal of DEM availability before any download starts. This confirms the original D-06 nullable `ToOne` design from the discarded context needs **no schema change**.

### Carried Forward from Original Discussion (still valid)
- **D-10 (was D-01/D-02):** `RegionStatus` (and any `DownloadedTilePackage` status) uses Dart enhanced enums with an explicit `code` int field, persisted via the shadow-property pattern (`@Transient()` enum field + `int` get/set shadow property) reading/writing `.code` — never `.index`. This deliberately deviates from `TrailEntity`/`ActiveNavigationEntity`, which both use `.index` today (the exact anti-pattern REGN-02 forbids).
- **D-11 (was D-06):** `Region` has two nullable `ToOne<DownloadedTilePackage>` fields: `vectorPackage` and `demPackage`. No package-type discriminator field, no `ToMany`/`@Backlink` — direct field access (`region.vectorPackage.target?.status`).
- **D-12 (was D-07):** `Region.status` (local download status) is a **computed getter, not a stored field** — derives from `vectorPackage.target?.status` (folding in `demPackage` status when a DEM is required/present), defaulting to `RegionStatus.notDownloaded` when no package rows exist yet. This guarantees Region and package status can never drift out of sync.

### Claude's Discretion
- None — every gray area identified was explicitly decided above.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/ROADMAP.md` — Phase 22 section (goal, success criteria, dependencies on Phase 21.5)
- `.planning/REQUIREMENTS.md` — REGN-01 (rewritten 2026-07-21 for the API-fetch design), REGN-02, REGN-03

### Decision history for this redesign
- `.planning/notes/region-catalog-backend-decision-trail.md` — full reasoning for why Phase 21.5 was inserted and why the bundled-asset design was rejected.
- `.planning/todos/pending/replan-phase-22-region-manifest.md` — the todo that triggered this re-discussion; can be resolved/removed once Phase 22 is replanned against this CONTEXT.md.

### Phase 21.5 — the API contract this phase's fetch function must match
- `.planning/phases/21.5-region-catalog-archive-pre-build-backend/21.5-CONTEXT.md` — D-06 through D-11 (endpoint shape, status semantics, staleness mechanism)
- `.planning/phases/21.5-region-catalog-archive-pre-build-backend/VERIFICATION.md` — confirms the exact live response shape (`id/name/bbox/status/version/vector_url/vector_size/dem_status/dem_url/dem_size/error`), auth requirement (any logged-in user, cookie-based session — not bearer token), and that the public path is the SvelteKit proxy at the same `/api/v1/regions` path.
- `db/routes/regions_get.go` — the exact field-by-field response construction (`RegionsList`). Ground truth for the parse model's field names/optionality: `version` only present once a build has succeeded; `vector_url`/`vector_size` only present when `status == "ready"` AND the file exists on disk; `dem_status` only present if a DEM build was ever attempted; `dem_url`/`dem_size` only present when `dem_status == "ready"` AND the file exists; `error` only present when `status == "error"`.
- `web/src/routes/api/v1/regions/+server.ts` — the public (SvelteKit-proxied) path the Flutter app actually calls in production; confirms no field remapping happens in the proxy (passes through Go's JSON verbatim).

### Prior Research (Phase 22, still applicable)
- `.planning/phases/22-region-package-data-model/22-RESEARCH.md` — ObjectBox entity patterns, enum persistence pitfalls (still valid; the fetch-source change doesn't affect the ObjectBox schema research)
- `.planning/research/SUMMARY.md`, `.planning/research/PITFALLS.md` — Critical Pitfall 5 (index-backed enum)

### Existing App API Client Pattern (for the fetch function)
- `app/lib/provider/api_provider.dart` — `apiProvider` Dio client; auth is cookie-based (`dio_cookie_manager` + `CookieManager`), not bearer-token — the fetch function should use `ref.read(apiProvider).get('/regions')`, no manual auth header needed.
- `app/lib/provider/trail/subcategory_provider.dart`, `app/lib/provider/trail/category_provider.dart` — closest existing "fetch + ObjectBox persist" precedent, but their `removeAll()`+`putMany()` merge strategy is explicitly **not** to be copied here (see D-01's rejected-alternative note). Their cookie-based Dio call shape and try/catch structure around the HTTP call are still useful references.
- `app/lib/models/list_result.dart` — generic paginated-wrapper parse model; **not applicable** here since `GET /api/v1/regions` returns a bare top-level JSON array, not a `{items: [...]}` wrapper. No existing precedent for a bare-array parse in this codebase — the fetch function parses the response body directly as `(response.data as List).map(RegionCatalogEntry.fromJson).toList()`.

### Existing Backend Tile Pipeline (background, not directly touched by this phase)
- `db/services/tiles/generator.go` — the older per-cell pipeline; explicitly NOT what Phase 22 talks to (superseded by the region-archive endpoints above for this phase's purposes).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `app/lib/entities/trail_entity.dart` — dual-id convention (`@Id() int obxId = 0` + separate business `@Index() @Unique(onConflict: ConflictStrategy.replace) String id`), `ToOne`/`@Backlink` relation patterns, manual `fromModel`/`toModel` mapping extension — mirror this structure for `Region`/`DownloadedTilePackage`. The `@Unique(onConflict: ConflictStrategy.replace)` on the business `id` field is what makes D-01's upsert-by-id cheap (`box.put()` with a matching unique id replaces in place rather than erroring).
- `app/lib/entities/active_navigation_entity.dart` — closest existing analog for a status-enum entity; shows the shadow-property pattern to adapt (swap `.index` for `.code`).
- `app/lib/provider/trail/subcategory_provider.dart` — shows the fetch/parse/persist shape to adapt, minus its destructive replace-all merge (see D-01).
- `app/lib/models/*.dart` (e.g. `trail.dart`) — `@freezed` + `part '*.freezed.dart'` + `part '*.g.dart'` + `factory fromJson` pattern; use this for the `RegionCatalogEntry` parse model, with all backend-optional fields (`version`, `vector_url`, `vector_size`, `dem_status`, `dem_url`, `dem_size`, `error`) modeled as nullable per `regions_get.go`'s conditional field construction.

### Established Patterns
- **Anti-pattern to avoid:** `TrailEntity.dbDifficulty` / `ActiveNavigationEntity.dbSessionType` both persist via `enum.index` — this is the exact anti-pattern REGN-02 forbids. Do not copy this pattern verbatim; only copy its structural shape (`@Transient()` + shadow int property), swapping `.index` for the new enhanced-enum `.code`.
- **Anti-pattern to avoid (new):** `subcategory_provider.dart`/`category_provider.dart`'s `removeAll()` + `putMany()` full-replace merge strategy. Do not copy this for `Region` — it would sever `ToOne` package links and destroy local download status on every refresh (D-01).
- ObjectBox Store/box registration is code-gen driven (`objectbox.g.dart`, `objectbox-model.json`) — new `@Entity()` classes under `app/lib/entities/` just need `build_runner` re-run; no manual registry list to update.

### Integration Points
- None yet — this phase is purely additive with no read/write integration. Phase 23 (`TileRepositoryManager`) is the first consumer of these entities and the first thing that actually calls the fetch/upsert function's downloaded-package side.
- Phase 24 (Settings UI) is the first thing that decides when/how the catalog fetch function (D-02) is actually invoked from the UI layer.

</code_context>

<specifics>
## Specific Ideas

- None beyond the decisions above — this discussion was entirely about implementation mechanics forced by the Phase 21.5 API contract, not new UX/vision references.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. The bigger scope question (what UI shows for orphaned/updateAvailable regions) was explicitly deferred to Phase 24/26 per D-08, not lost.

</deferred>

---

*Phase: 22-region-package-data-model*
*Context gathered: 2026-07-21*
