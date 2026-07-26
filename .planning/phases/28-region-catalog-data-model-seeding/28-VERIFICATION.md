---
phase: 28-region-catalog-data-model-seeding
verified: 2026-07-25T22:37:24Z
status: gaps_found
score: 9/10 must-haves verified
overrides_applied: 0
gaps:
  - truth: "The committed db/migrations/initial_data/regions_seed.json is a real, distributable git artifact that a fresh checkout of this repo's actual git host will contain"
    status: failed
    reason: "regions_seed.json is committed at 730,548,521 bytes (697 MiB) — by far the largest blob ever recorded in this repository's git history (14x the next-largest, an accidental debug binary). GitHub hard-rejects any pushed file over 100 MB without Git LFS, and this repo has no .gitattributes / Git LFS configuration. The commit (67d056c9) exists only on the local feature/app branch, which is 12 commits ahead of origin/feature/app — it has never actually been pushed. When this branch (or any branch containing commit 67d056c9) is pushed to github.com/open-wanderer/wanderer, GitHub will reject the push outright. Every real checkout obtained via `git clone`/`git pull` from the project's actual git host (the only distribution path a self-hosted product ships through) will be missing this file entirely, and the Phase 28-03 migration's `os.ReadFile(\"migrations/initial_data/regions_seed.json\")` will fail with a file-not-found error, aborting the migration and preventing the instance from booting — the exact opposite of SEED-02's 'zero admin action' promise. Switching json.MarshalIndent to compact json.Marshal only reduces the file to ~216 MB (still 2x over GitHub's hard limit), so this is not a one-line formatting fix; it requires Git LFS, seed compression, geometry simplification, or a different distribution mechanism for the polygon data."
    artifacts:
      - path: "db/migrations/initial_data/regions_seed.json"
        issue: "730 MB committed blob exceeds GitHub's 100 MB hard per-file push limit; not yet pushed to origin; will break any real clone of the repo"
      - path: "db/commands/seed_regions.go"
        issue: "json.MarshalIndent (line 132) triples the file's size relative to compact encoding, but even compact encoding (~216MB, verified) remains over GitHub's limit — indentation is a contributing but not sole cause"
    missing:
      - "A plan to make the seed data pushable to the project's actual GitHub remote: Git LFS tracking for db/migrations/initial_data/*.json (with corresponding CI/deploy support for LFS pointers), or gzip-compressing the committed seed and decompressing it in the migration's os.ReadFile step, or reducing polygon coordinate precision/simplifying high-vertex geometries (RESEARCH's own spot-check found rows over 1.4MB of raw coordinates each), or restructuring the seed into a fetched-at-build-time artifact rather than a committed one."
deferred: []
human_verification: []
---

# Phase 28: Region Catalog Data Model & Seeding Verification Report

**Phase Goal:** A fresh, self-hosted Wanderer instance boots with a fully populated, hierarchical, toggleable region catalog — sourced from CoMaps' extract hierarchy — with zero admin action required.
**Verified:** 2026-07-25T22:37:24Z
**Status:** gaps_found
**Re-verification:** No — initial verification (three prior attempts were interrupted before producing a VERIFICATION.md; none exist to build on)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `regions` collection exists with `comaps_id`, self-referencing `parent`, `path`, `depth`, `sort_order`, `name`, `kind` on every row (CATALOG-01) | VERIFIED | `db/migrations/1785000000_create_regions_collection.go` builds all fields via Go SDK; live `migrate up` against a scratch pb_data produced a `regions` table with exactly these columns; `SELECT` confirmed 1306 rows with all fields populated. |
| 2 | Leaf rows carry `polygon` (GeoJSON) + derived `bbox`; group rows carry neither (CATALOG-02) | VERIFIED | Live DB query: 1153/1153 leaf rows have non-empty `polygon`+`bbox`; 153/153 group rows have `polygon IS NULL/empty`; 0 group rows have a polygon. Sample checked: Abkhazia (leaf) has a real coordinate-ring polygon + 4-element bbox; Algeria (group) has none. |
| 3 | Leaf rows' `enabled` defaults to `false` on first seed; group rows carry no enabled/polygon/bbox semantics (CATALOG-03) | VERIFIED | `record.Set("enabled", false)` is only reached inside `if row.Kind == "leaf"` (migration line ~131); live query: all 1153 leaf rows have `enabled=0`; no row is pre-enabled. |
| 4 | `db/commands/seed_regions.go` parses fetched CoMaps `hierarchy.txt`+`.poly` files and writes a flattened JSON seed matching the `regions` schema (SEED-01) | VERIFIED | `SeedRegions()` registered in `db/main.go` (`grep` confirms); calls `ParseHierarchy`/`ParsePoly` directly (no re-implementation); `go build`/`go vet` clean; committed `regions_seed.json` is a well-formed 1306-row array (153 group + 1153 leaf) with every field the schema expects, no `World`/`WorldCoasts` rows, Fiji correctly a `MultiPolygon`, Germany correctly a `group` with 16 children — matches every structural assertion in the plan and SUMMARY. |
| 5 | A fresh instance auto-runs the migration and bulk-inserts the full catalog with zero admin action, correct parent/path/depth (SEED-02 core) | VERIFIED (live-tested) | Ran `go run . migrate up` against a genuinely empty scratch `pb_data` (with a locally-spun-up Meilisearch to satisfy an unrelated pre-existing migration dependency): the `regions` migration applied automatically as part of the normal migration chain, no manual seeding step invoked. Resulting DB: 1306 rows, 1082 rows resolved a non-null `parent` (child rows), 224 top-level rows have no parent, **zero orphaned parent references** (verified via `parent NOT IN (SELECT id FROM regions)` returning 0 rows). |
| 6 | Re-running migrations (`migrate down` then `up`) does not duplicate rows | VERIFIED (live-tested) | Ran `migrate down 1` (dropped the `regions` collection) then `migrate up` again against the same scratch pb_data: final row count is exactly 1306 (not 2612) — the `CountRecords("regions") > 0` idempotency guard combined with the collection being fully recreated makes the cycle clean, no duplicate/unique-index errors. |
| 7 | `ParsePoly` correctly converts single-ring / multi-outer-ring / hole `.poly` shapes with RFC 7946 winding | VERIFIED | `cd db && go test ./commands/ -run TestParsePoly -v` — all 7 subtests pass (Polygon, MultiPolygon, hole, winding-reversal, 3 malformed-input error cases). `grep -c panic` = 0. |
| 8 | `ParseHierarchy` correctly derives depth/parent/path/name/kind and skips `World`/`WorldCoasts` | VERIFIED | `cd db && go test ./commands/ -run TestParseHierarchy -v` — all 8 subtests pass. `grep -c panic` = 0. |
| 9 | Malformed/missing upstream data aborts loudly, never a panic or silent gap | VERIFIED | Both parsers' malformed-input tests pass; `seed_regions.go` uses `log.Fatalf` on fetch/parse failure naming the offending region; migration wraps read/parse errors with `%w`. `grep -c panic` = 0 across all four phase files. |
| 10 | The committed `regions_seed.json` is a real, distributable git artifact that a fresh checkout of this repo's actual git host will contain | **FAILED** | See Gaps below — the file is 730 MB, exceeds GitHub's 100 MB hard push limit, has never been pushed (local branch is 12 commits ahead of `origin/feature/app`), and no Git LFS is configured. |

**Score:** 9/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/commands/poly_parser.go` | `ParsePoly` Osmosis .poly -> GeoJSON | VERIFIED | Exists, exports `ParsePoly(data []byte) (map[string]any, [4]float64, error)`, wired and tested. |
| `db/commands/hierarchy_parser.go` | `ParseHierarchy`/`HierarchyNode` | VERIFIED | Exists, exports both symbols, wired and tested. |
| `db/commands/poly_parser_test.go` | Table tests | VERIFIED | 7 subtests, all pass. |
| `db/commands/hierarchy_parser_test.go` | Table tests | VERIFIED | 8 subtests, all pass. |
| `db/commands/seed_regions.go` | `SeedRegions`/`SeedRow` Cobra command | VERIFIED | Registered in `main.go`, validated `--commit` regex present, `io.LimitReader` present, calls both plan-28-01 parsers. |
| `db/migrations/initial_data/regions_seed.json` | Committed flattened CoMaps seed | ⚠️ VERIFIED-BUT-UNDISTRIBUTABLE | Structurally correct (1306 rows, correct leaf/group semantics) but 730 MB — see gap above; the artifact "exists" locally but cannot reach the project's actual git remote as committed. |
| `db/migrations/1785000000_create_regions_collection.go` | regions schema + bulk-insert migration | VERIFIED | All ten fields, self-relation (two-pass create-then-patch), 3 indexes, `RunInTransaction` two-pass insert-then-link, `CountRecords` idempotency guard, `down()` deletes collection. Live-tested via actual `migrate up`/`down`/`up` cycle. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `seed_regions.go` | `ParseHierarchy`/`ParsePoly` (28-01) | direct in-package calls | WIRED | `grep -E 'ParseHierarchy|ParsePoly' db/commands/seed_regions.go` matches both. |
| `db/main.go setupCommands` | `commands.SeedRegions()` | `app.RootCmd.AddCommand` | WIRED | `grep -n 'commands.SeedRegions()' db/main.go` matches, immediately after `Dedup` registration. |
| `seed_regions.go` | `regions_seed.json` | `os.WriteFile` | WIRED | Confirmed via full-generation run producing the committed file (per SUMMARY + structural checks). |
| `1785000000_create_regions_collection.go` | `regions_seed.json` | `os.ReadFile` + `json.Unmarshal` | WIRED | Live-tested: migration read the real committed file and inserted all 1306 rows correctly. |
| `regions.parent` RelationField | `regions` collection (self-reference) | `CollectionId: collection.Id` (two-pass) | WIRED | Live-tested: 1082 rows correctly resolved a parent id; 0 orphaned references. |
| bulk-insert pass 2 | each child record's `parent` | `record.Set("parent", idByPath[...])` | WIRED | Live-tested, keyed by `path` (not `comaps_id`, per the documented Task 2 deviation) — confirmed correct via the zero-orphan check. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `regions` table (post-migration) | all 1306 rows | `regions_seed.json` -> `json.Unmarshal` -> `RunInTransaction` inserts | Yes — verified via live SQLite query, not just code inspection | FLOWING |

### Behavioral Spot-Checks / Probe Execution

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Parser unit tests | `cd db && go test ./commands/ -count=1 -v` | All 15 subtests (7 poly + 8 hierarchy) PASS | PASS |
| Build/vet | `cd db && go build ./... && go vet ./...` | Exit 0, no issues | PASS |
| Fresh-instance migrate up (real, not simulated) | `go run . migrate up --dir <scratch>` against empty pb_data, with a locally-run Meilisearch to satisfy an unrelated migration dependency | All 29 migrations applied; `regions` populated with 1306 rows, correct leaf/group/enabled/polygon/bbox semantics, 0 orphaned parents | PASS |
| Idempotency (down + up cycle) | `migrate down 1` then `migrate up` again on the same scratch pb_data | Row count remained exactly 1306 after the cycle (no duplication) | PASS |
| Spot-check Fiji/Germany/no-header-rows | `python3` structural check against the real committed `regions_seed.json` | Fiji is a `MultiPolygon` leaf; Germany is a `group` with 16 children; 0 `World`/`WorldCoasts` rows | PASS |
| Committed seed file distributability | `git cat-file`, `git rev-list --objects`, `git ls-remote origin`, `.gitattributes` check | 730,548,521-byte blob (largest in repo history by 14x), commit `67d056c9` not present on `origin/feature/app` (12 commits behind), no Git LFS configured anywhere in the repo | **FAIL** |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|--------------|-------------|-------------|--------|----------|
| CATALOG-01 | 28-01, 28-03 | `regions` collection with hierarchy fields + self-relation | SATISFIED | Live migration + DB query. |
| CATALOG-02 | 28-01, 28-03 | Leaf polygon (GeoJSON) + derived bbox | SATISFIED | Live migration + DB query. |
| CATALOG-03 | 28-03 | Leaf `enabled` default false; group carries no geometry/enabled semantics | SATISFIED | Live migration + DB query. |
| SEED-01 | 28-01, 28-02 | Maintainer-run `seed_regions.go` producing flattened JSON seed | SATISFIED (with a downstream distribution caveat — see Gaps) | Command exists, registered, tested, produced the real committed file with correct structure. |
| SEED-02 | 28-03 | Auto-run migration bulk-inserts on every fresh boot, zero admin action | **BLOCKED for any checkout that doesn't already have the local commit** | Live-tested successfully against the current local working tree, but the committed seed file cannot reach `origin` as committed (see gap) — so "every fresh instance" boot claim fails for any instance built from a real `git clone` of the project's GitHub remote post-push. |

No orphaned requirements: all 5 IDs (CATALOG-01/02/03, SEED-01/02) declared in Phase 28's REQUIREMENTS.md traceability table are claimed across the three plans' frontmatter, and all were checked above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `db/migrations/initial_data/regions_seed.json` | n/a (whole file) | 730 MB committed JSON blob, indented (`json.MarshalIndent`), 3.4x larger than compact encoding would be | 🛑 Blocker | Exceeds GitHub's 100 MB hard per-file push limit; repo cannot actually publish this artifact to its own git host without further work (Git LFS, compression, or precision reduction) |
| — | — | No `TODO`/`FIXME`/`XXX`/`HACK`/`PLACEHOLDER` markers found in any of the four modified/created phase files | ℹ️ Info | Clean — no debt markers to gate on |
| — | — | No `panic(` in any of the four phase files (`poly_parser.go`, `hierarchy_parser.go`, `seed_regions.go`, `1785000000_create_regions_collection.go`) | ℹ️ Info | Matches threat-model mitigation claims |

### Human Verification Required

None. All must-haves that the plans flagged for human checking (fresh-boot query, Fiji/Germany/header-row spot-checks) were verified programmatically against a real `migrate up` run and the real committed seed file in this pass — no residual item requires a human to test something a script/query cannot.

### Gaps Summary

Nine of ten observable truths hold up under direct, live testing (not just source review): the schema, the parsers, the seed-generation command, and — critically — an actual `migrate up`/`down`/`up` cycle against a real scratch PocketBase instance, which confirmed the migration auto-seeds 1306 correctly-linked rows with the right leaf/group/polygon/bbox/enabled semantics and is idempotent.

The one failing truth is structural, not functional, within this local working tree — but it is fatal to the phase's actual delivery goal ("a fresh, self-hosted Wanderer instance boots... with zero admin action"). The `regions_seed.json` committed in this branch is 730 MB, roughly 14 times larger than the biggest file previously ever committed to this repository, and about 7x over GitHub's hard 100 MB per-file push limit. This branch (`feature/app`) is 12 commits ahead of `origin/feature/app` and this file has never actually been pushed. No `.gitattributes`/Git LFS configuration exists anywhere in the repo to accommodate it. The practical consequence: the moment this branch (or any branch/rebase carrying commit `67d056c9`) is pushed to `github.com/open-wanderer/wanderer`, GitHub will reject the push outright, and every other checkout of this project obtained the normal way (`git clone`/`git pull` from GitHub) will simply not have this file — causing the Phase 28-03 migration's `os.ReadFile` to fail and the instance to fail to boot, which is the exact scenario SEED-02 exists to prevent.

This wasn't a silent shortcut by the executor — the plan itself specified `json.MarshalIndent` (28-02-PLAN.md) with no anticipation of file size, and neither 28-RESEARCH.md, 28-PATTERNS.md, nor 28-CONTEXT.md discuss git-hosting size limits at all. It's a genuine planning blind spot that only surfaces once you check the artifact against the project's actual distribution mechanism (git push to GitHub) rather than just running the migration locally.

Recommended remediation directions (not prescriptive — a closure plan should choose): (a) Git LFS-track `db/migrations/initial_data/*.json` with corresponding CI/deploy support for LFS smudge/checkout, (b) gzip-compress the committed seed and decompress it inside the migration's read step (note: compact encoding alone, ~216 MB, is not sufficient on its own), (c) reduce coordinate precision / simplify the highest-vertex-count polygons (several leaf rows carry 1+ MB of raw coordinate data each), or (d) fetch/generate the seed at build/release time rather than committing the fully-expanded JSON to source control.

---

*Verified: 2026-07-25T22:37:24Z*
*Verifier: Claude (gsd-verifier)*
