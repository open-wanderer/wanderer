# Phase 28: Region Catalog Data Model & Seeding - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-24
**Phase:** 28-region-catalog-data-model-seeding
**Areas discussed:** CoMaps Snapshot Sourcing, .poly → GeoJSON Parsing

---

## CoMaps Snapshot Sourcing

| Option | Description | Selected |
|--------|-------------|----------|
| Vendor pinned snapshot in-repo | Commit raw hierarchy.txt + .poly files under a scratch dir, pinned to a CoMaps commit hash | |
| Fetch fresh from Codeberg at tool-run time | seed_regions.go downloads hierarchy.txt/.poly files directly from Codeberg at run time, pinned to a commit hash; only the JSON output is committed | ✓ |

**User's choice:** Fetch fresh from Codeberg at tool-run time.

**Follow-up — commit hash pinning:**

| Option | Description | Selected |
|--------|-------------|----------|
| Hardcoded constant in seed_regions.go | Commit hash lives as a Go constant; refresh means editing the constant | |
| CLI flag with a default | `--commit=<hash>` flag with a baked-in default, overridable without editing source | ✓ |

**Notes:** Refresh flow is: bump/override the commit hash, re-run the tool, review the JSON diff, commit, ship as a normal migration in a release. No raw upstream files are vendored in this repo — only the tool's flattened JSON output.

---

## .poly → GeoJSON Parsing

| Option | Description | Selected |
|--------|-------------|----------|
| Hand-roll a minimal parser | ~50-line Go parser for the Osmosis .poly plaintext format, no new dependency | ✓ |
| Find/adapt an existing Go library | Search for an existing Go package that parses Osmosis .poly format | |

**User's choice:** Hand-roll a minimal parser.

**Follow-up — ring support:**

| Option | Description | Selected |
|--------|-------------|----------|
| Support holes + multi-part from day one | Parse `!`-prefixed inner rings and multiple outer rings; output GeoJSON Polygon or MultiPolygon as needed | ✓ |
| Single outer ring only, flag/skip complex ones | Parse only the first outer ring, warn/skip regions with holes or multiple parts | |

**Notes:** Skipping multi-ring support would silently produce wrong/simplified boundaries for coastal or multi-part regions (e.g. exclaves, islands) with no easy way to notice, since seed_regions.go processes the full CoMaps hierarchy in one unattended pass.

---

## Claude's Discretion

- Exact seed JSON file name/location under `db/migrations/initial_data/` — follow existing migration conventions.
- Internal fetch-step implementation details (HTTP client, error handling) — simple, dev-time-only tool.
- `sort_order` derivation — preserve CoMaps' own sibling ordering from `hierarchy.txt`.

## Deferred Ideas

None — discussion stayed within phase scope.
