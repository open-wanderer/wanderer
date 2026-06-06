---
plan: 01-02
phase: 01-foundation-config
status: complete
requirements: [INFRA-02, INFRA-04]
completed: 2026-06-06
self_check: PASSED
key-files:
  created:
    - web/tests/playwright/fixtures/trail.gpx
    - web/tests/playwright/helpers/data.ts
    - web/tests/playwright/helpers/data.test.ts
---

## Summary

Created the shared GPX fixture and test-data factory module for the Wanderer E2E test suite.

## Tasks Completed

| Task | Name | Commit | Status |
|------|------|--------|--------|
| 1 | Create trail.gpx fixture with embedded name and realistic elevation | 55e8a9c0 | ✓ |
| 2 | Create helpers/data.ts with unique-string factory functions | aa66df24 | ✓ |

## What Was Built

**`web/tests/playwright/fixtures/trail.gpx`** — GPX 1.1 fixture file with:
- 30 trackpoints at realistic alpine coordinates (lat ~47.67 / lon ~11.90)
- `<name>Test Trail</name>` nested inside `<trk>` (not metadata — matches gpx_util.ts resolution order)
- 175m raw elevation gain across a rising-then-falling profile (well above the 5m Z smoothing threshold)
- `xmlns="http://www.topografix.com/GPX/1/1"` and `version="1.1"` on root element
- Consumed by `setInputFiles()` in phases 2–4

**`web/tests/playwright/helpers/data.ts`** — factory module exporting:
- `trailName(): string` → `Test Trail ${Date.now()}`
- `listName(): string` → `Test List ${Date.now()}`
- `commentText(): string` → `Test Comment ${Date.now()}`

**`web/tests/playwright/helpers/data.test.ts`** — Vitest unit tests verifying prefix and uniqueness contracts.

## Acceptance Criteria Verified

- ✓ trail.gpx parses as well-formed XML
- ✓ 30 trackpoints (≥25 required)
- ✓ 30 `<ele>` elements matching trackpoint count
- ✓ 175m raw cumulative gain (>50m required)
- ✓ `Test Trail` name inside `<trk>` element
- ✓ GPX 1.1 root element with correct xmlns
- ✓ data.ts exports 3 factory functions with Date.now() suffix
- ✓ No secrets or credentials in either file

## Deviations

None. Implemented exactly per PATTERNS.md and RESEARCH.md.

## Self-Check: PASSED
