---
phase: 01-foundation-config
plan: "01"
subsystem: test-infrastructure
tags: [playwright, config, timeouts, reporter, screenshot]
status: checkpoint
dependency_graph:
  requires: []
  provides: [playwright-config-timeouts, playwright-config-reporter, playwright-config-screenshot]
  affects: [all-playwright-specs]
tech_stack:
  added: []
  patterns: [playwright-config-defineConfig]
key_files:
  created: []
  modified:
    - web/playwright.config.ts
decisions:
  - "timeout: 60_000 at config root (D-06) — per-test budget for all specs"
  - "reporter: CI-conditional github+html vs html-only locally (D-10)"
  - "actionTimeout: 15_000 in use block (D-07) — covers click/fill/hover"
  - "navigationTimeout: 30_000 in use block (D-08) — covers goto/waitForURL/SSR cold-start"
  - "screenshot: 'only-on-failure' in use block (D-09) — failure debugging artifacts"
metrics:
  duration: "~5 minutes"
  completed_date: "2026-06-06"
  tasks_completed: 1
  tasks_total: 2
  files_modified: 1
---

# Phase 01 Plan 01: Playwright Config Timeouts & Reporter Summary

**One-liner:** Updated `playwright.config.ts` with 60s per-test timeout, 15s action/30s navigation timeouts, screenshot-on-failure, and CI-conditional github+html reporter.

## What Was Built

Task 1 completed: `web/playwright.config.ts` updated with five new/changed config values per the INFRA-01 requirement and decisions D-06 through D-10.

Task 2 (checkpoint:human-verify) reached: The Docker stack is not currently running (localhost:3000 unreachable). Human verification of the existing three specs is required once the stack is up.

## Tasks Completed

| Task | Description | Commit | Status |
|------|-------------|--------|--------|
| 1 | Add timeouts, reporter, and screenshot config to playwright.config.ts | f6b44c37 | Done |
| 2 | Verify existing three specs pass under new config (Walking Skeleton) | — | Awaiting human verify |

## Acceptance Criteria Status (Task 1)

All five grep assertions pass:

- `timeout: 60_000` present at config root: 1 match
- `actionTimeout: 15_000` in use block: 1 match
- `navigationTimeout: 30_000` in use block: 1 match
- `screenshot: 'only-on-failure'` in use block: 1 match
- CI reporter conditional: 1 match
- `trace: 'on-first-retry'` still present: confirmed
- `projects` array (`setup`, `teardown`, `chromium`): all three unchanged

TypeScript check: `npx tsc --noEmit -p tsconfig.json` — no errors from playwright.config.ts.

## Checkpoint Details

**Type:** checkpoint:human-verify (gate: blocking)

**Reason for checkpoint:** Docker stack (localhost:3000) was not reachable at execution time. The plan requires running `npm run test:integration -- --project=chromium` from `web/` to confirm all three existing specs (`index.spec.ts`, `list.spec.ts`, `user.spec.ts`) pass under the new config.

**To verify after starting the stack:**
1. `docker compose up -d` (wait for localhost:3000 to respond)
2. `cd web && npm run test:integration -- --project=chromium`
3. All three existing specs MUST pass (2 index tests, 3 list tests, 1 user test)
4. Confirm `web/playwright-report/index.html` exists after the run
5. If `index.spec.ts` "location search works" times out at 30s — this is documented Pitfall 1; report it rather than raising the timeout (D-08 is locked)

## Deviations from Plan

None — plan executed exactly as written. Task 1 applied precisely the diff specified in PATTERNS.md lines 62–79.

## Threat Flags

No new security-relevant surface introduced. The only change is a static config file affecting the local test runner (T-01-01 and T-01-SC both accepted as per plan threat model).

## Known Stubs

None — no UI rendering or data flow introduced in this plan.

## Self-Check

- [x] `web/playwright.config.ts` modified with all five required values (grep verified above)
- [x] Commit f6b44c37 exists on branch `worktree-agent-abe583fcb0a75d819`
- [x] No unexpected file deletions in commit
- [x] TypeScript type check passes (no playwright.config.ts errors)
- [ ] Task 2 (running specs) — awaiting human verify (Docker stack not running)
