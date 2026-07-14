---
phase: quick-260714-qtl
plan: 01
subsystem: api
tags: [meilisearch, cookies, sveltekit, hooks.server.ts, token-expiry]

# Dependency graph
requires: []
provides:
  - Server-side authoritative expiry check for the meilisearch_token cookie
affects: [search, meilisearch-integration]

# Tech tracking
tech-stack:
  added: []
  patterns: [server-computed expiry timestamp embedded in cookie payload, checked on every read instead of trusting client Max-Age]

key-files:
  created: []
  modified: [web/src/hooks.server.ts]

key-decisions:
  - "Extended cookie payload to token|ownerId|version|expiresAtMs rather than decoding the Meilisearch JWT itself — server already knows the mint time and TTL, no new dependency needed"
  - "Reused a single SEARCH_TOKEN_TTL_MS constant for both cookie maxAge and the expiry computation so the two can't drift apart"
  - "60s safety buffer on the expiry check to avoid a token expiring mid-request"

patterns-established:
  - "Server-owned expiry fields embedded in a cookie value, checked against Date.now() on every read, rather than relying solely on client-enforced Max-Age"

requirements-completed: [QUICK-260714-qtl]

# Metrics
duration: ~5min
completed: 2026-07-14
---

# Quick Task 260714-qtl: Fix meilisearch_token Cookie Never-Valid Expiry Summary

**Server now embeds and checks a real `expiresAtMs` timestamp in the `meilisearch_token` cookie instead of trusting the client's `Max-Age` alone, closing the gap where a stale token surviving past 24h caused every search call to 500 indefinitely.**

## Performance

- **Duration:** ~5 min
- **Completed:** 2026-07-14
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Extended `meilisearch_token` cookie payload from `token|ownerId|version` to `token|ownerId|version|expiresAtMs`
- Read path now rejects (and transparently re-mints, same path as owner/version mismatch) any cookie whose `expiresAtMs` is at or past a 60s-buffered `Date.now()`
- Write path computes `expiresAtMs = Date.now() + SEARCH_TOKEN_TTL_MS` at mint time, with `SEARCH_TOKEN_TTL_MS` (24h) shared between the cookie's `maxAge` and the expiry field so they can't drift apart

## Task Commits

1. **Task 1: Track and check real token expiry in the meilisearch_token cookie** - `ed114163` (fix)

**Plan metadata:** docs commit handled by orchestrator (not created by this executor)

## Files Created/Modified
- `web/src/hooks.server.ts` - Added `SEARCH_TOKEN_TTL_MS` constant; extended cookie read/write logic with `expiresAtMs` tracking and a 60s-buffered expiry check

## Decisions Made
- No new dependency added — the server computes and owns the expiry itself rather than parsing the actual Meilisearch JWT, matching the plan's explicit constraint.
- Reused the existing owner/version-mismatch re-mint branch for expired tokens rather than adding a separate code path, per the plan's instruction.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Fix is self-contained to `web/src/hooks.server.ts`; no follow-up work required.
- `cd web && npx tsc --noEmit -p .` reports no new errors attributable to `hooks.server.ts`.
- `grep -n "expiresAtMs" web/src/hooks.server.ts` confirms both read (destructure + comparison) and write (computation + interpolation) sites are present.

---
*Quick task: 260714-qtl*
*Completed: 2026-07-14*

## Self-Check: PASSED

- FOUND: .planning/quick/260714-qtl-fix-meilisearch-token-cookie-never-valid/260714-qtl-SUMMARY.md
- FOUND: commit ed114163
