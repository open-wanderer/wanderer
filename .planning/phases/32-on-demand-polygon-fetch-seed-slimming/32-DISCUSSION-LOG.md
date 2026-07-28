# Phase 32: On-Demand Polygon Fetch & Seed Slimming - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-28
**Phase:** 32-on-demand-polygon-fetch-seed-slimming
**Areas discussed:** Todo folding, Cron failure behavior, Source URL configurability, Migration path for existing instances, Test strategy for the fetch, Admin-UI polygon dependency (blocker)

---

## Todo Folding

| Option | Description | Selected |
|--------|-------------|----------|
| Fold it in | Purge todo becomes part of Phase 32 — phase isn't done until the blob is gone from history | ✓ |
| Keep it separate | Phase delivers the code change; purge stays a standalone operational task | |

**User's choice:** Fold it in
**Notes:** The Phase 29 pmtiles-extraction spike also matched at 0.9 but was not offered — it is stale (`resolves_phase: 29`, and Phase 29 shipped).

---

## Cron Failure Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Mark error, continue loop | Call existing `setError` (`status: "error"` + `error_message`), then next region | ✓ |
| Match today's behavior | Log and return, leaving the archive record untouched | |
| Error only after N cycles | Tolerate transient failures, flip to error after several consecutive failed runs | |

**User's choice:** Mark error, continue loop
**Notes:** Also fixes a latent wart — today's polygon-missing path at `builder.go:231-241` skips `setError`, leaving records stuck at `"building"`.

| Option | Description | Selected |
|--------|-------------|----------|
| Never — always continue | Preserves `buildRegionSafely` per-region isolation | ✓ |
| Abort after consecutive failures | Treat repeated failures as "upstream is down" and stop early | |

**User's choice:** Never — always continue

| Option | Description | Selected |
|--------|-------------|----------|
| Tighter budget for builds | Build path gets 2–3 attempts; `seed-regions` keeps its patient 10× | ✓ |
| Reuse fetch unchanged | One code path, no new knobs; accepts multi-hour runs on a bad upstream day | |
| Overall run deadline | Bound total fetch time for the whole run | |

**User's choice:** Tighter budget for builds
**Notes:** Raised because `fetch`'s 10 retries × 30s default `Retry-After` could burn ~5 min per region across ~100 regions.

---

## Source URL Configurability

| Option | Description | Selected |
|--------|-------------|----------|
| Env-configurable override | Follow the `NOMINATIM_URL`/`OVERPASS_API_URL`/`VALHALLA_URL` convention | |
| Hardcoded, both hosts | GitHub + Codeberg as constants, no override | ✓ |
| Hardcoded now, env var later | Ship constants, add configurability only on demand | |

**User's choice:** Hardcoded, both hosts
**Notes:** Follow-up question on replace-vs-prepend override semantics answered "See 1" — moot given no override exists. Counter-argument raised and set aside: those env vars point at *services* an admin might self-host, whereas an override here would need to replicate a git file-tree path shape.

---

## Migration Path for Existing Instances

| Option | Description | Selected |
|--------|-------------|----------|
| Edit 1785 in place | Rewrite the existing migration; cleanest history, requires local `pb_data` reset | ✓ |
| Edit in place + defensive drop | Rewrite, plus a small migration dropping `region_polygons` if present | |
| New forward migration only | Leave 1785 untouched, append a drop + re-seed migration | |

**User's choice:** Edit 1785 in place
**Notes:** Decided after verifying the migration has never shipped — absent from `origin/main` and from tags v1.6, v1.5, v0.20.0. The idempotency guard (`CountRecords("regions") > 0`) means already-seeded dev boxes won't re-run it and need a manual reset; accepted knowingly.

---

## Test Strategy for the Fetch

| Option | Description | Selected |
|--------|-------------|----------|
| Package-level var, not const | Unexported `var`s tests reassign; `httptest` exercises the real fallback path | |
| Injectable fetcher interface | Stub replaces HTTP client; URL construction and fallback ordering untested | |
| Don't test the network layer | Pure unit tests for `ParsePoly` and fallback decision logic only | ✓ |

**User's choice:** Don't test the network layer
**Notes:** Cost was stated before the choice — the GitHub→Codeberg fallback is the one genuinely new behavior and ships without automated coverage. Recorded in CONTEXT.md as deliberate so the planner does not add `httptest` back on its own initiative. Clean consequence: with no test seam needed, base URLs can stay `const`, which resolves against the hardcoded-hosts choice.

| Option | Description | Selected |
|--------|-------------|----------|
| Polygon value-equality | Fetched GeoJSON deep-equals what the old seed held (`490a685f` precedent) | ✓ |
| Byte-identical archive | Compare `.pmtiles` files byte for byte | |
| Same tile set, hashed content | Compare tile counts and a content hash, tolerating metadata drift | |

**User's choice:** Polygon value-equality
**Notes:** Makes ROADMAP.md's Phase 32 success criterion 3 ("byte-equivalent archive") obsolete — flagged in CONTEXT.md as a required correction, not edited from this workflow.

---

## Admin-UI Polygon Dependency (blocker)

Raised at the end of the session, after a verification pass found that `db/routes/regions_ext/regions_ui.html` reads `/api/collections/region_polygons/records` in three flows (lines 1110, 1157, 1183) — including a hover preview needing polygons for **disabled** regions. This contradicted the earlier claim that `buildRegion` was the only geometry consumer, which had come from a `*.go`-only grep.

| Option | Description | Selected |
|--------|-------------|----------|
| Keep table as lazy cache | Stays in schema, never seeded; rows written on first demand | |
| Backend proxy endpoint, no table | Drop collection, add `GET /api/v1/regions/{path}/polygon` with a cache | |
| Admin map falls back to bbox | Draw rectangles from catalog `bbox`; amends ADMINUI-03 | |
| Let me think about it | Hold Phase 32 context; decide separately before planning | ✓ |

**User's choice:** Let me think about it
**Notes:** Phase 32 context left in BLOCKED status. Auto-advance to planning deliberately not triggered despite `workflow.auto_advance: true`, because planning on an invalidated premise would waste the pass.

---

## Claude's Discretion

- Exact retry counts and timeout caps for the tighter build-path budget
- Field name and placement of the commit SHA inside the catalog artifact
- Whether the build-path fetch is a new function or a parameterized variant of `fetch`
- Error message wording (subject to naming which upstreams were tried)
- Whether the slim catalog keeps the `regions_seed` filename lineage

## Deferred Ideas

- **One-object-per-line seed format** — 283.0 KB vs pretty-printed's 386.9 KB, one diff line per changed region instead of ~10. Revisit if refresh diffs prove hard to review.
- **Env-configurable polygon source** — rejected in D-04; small self-contained follow-up if a restricted-egress self-hoster asks.
- **Automated fallback coverage** — known remedy if the fallback misfires: promote base URLs from `const` to package-level `var` and add `httptest`.
