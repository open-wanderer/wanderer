---
phase: 06-admin-browser-ui
plan: "01"
subsystem: federation-admin-ui
tags: [alpine-js, tailwind, html, federation, admin, browser-ui]
dependency_graph:
  requires:
    - "Phase 5 federation admin API endpoints (GET /federation/peers, POST /federation/discover, follow, approve, reject, disconnect)"
  provides:
    - "db/routes/federation_ui.html — self-contained admin SPA for federation peer management"
  affects:
    - "Plan 06-02 (Go handler that embeds federation_ui.html via go:embed)"
tech_stack:
  added:
    - "Alpine.js 3.14.8 (CDN, browser runtime)"
    - "Tailwind CSS Play CDN (browser runtime)"
  patterns:
    - "Alpine x-data single-component SPA on body element"
    - "Synchronous theme IIFE before body render (avoids dark/light flash)"
    - "PocketBase admin localStorage token piggyback (__pb_superusers__/_)"
    - "Authorization: <raw-jwt> header (no prefix, PocketBase format)"
    - "Field mapping: API user_count/trail_count -> preview.users/preview.trails"
    - "x-if in <template> for conditional table cells (not x-show on <tr>)"
key_files:
  created:
    - path: "db/routes/federation_ui.html"
      description: "Complete self-contained Alpine.js + Tailwind CDN federation admin SPA (359 lines)"
  modified: []
decisions:
  - "Scripts placed in mandated order: theme IIFE -> tailwind.config darkMode:class -> Tailwind CDN -> federationApp() -> Alpine defer"
  - "Both tasks implemented atomically in one file pass — Task 2 markup and methods written together with Task 1 shell"
  - "rowErrors uses Object.assign pattern for Alpine reactivity"
  - "x-show on <tbody> elements (not <tr>) for loading/empty state — valid pattern per UI-SPEC"
metrics:
  duration_minutes: 15
  completed_date: "2026-06-27T20:02:30Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 1
  files_modified: 0
---

# Phase 06 Plan 01: Federation Admin Browser UI (HTML SPA) Summary

**One-liner:** Self-contained Alpine.js + Tailwind CDN federation admin SPA that reads the PocketBase superuser JWT from localStorage and drives the full discover -> connect -> approve/reject/disconnect peer management workflow.

## What Was Built

Created `db/routes/federation_ui.html` (359 lines) — a single-file HTML admin dashboard for federation peer management. The page:

1. Detects dark/light theme synchronously before the body renders by reading `pbColorScheme` from localStorage, falling back to `prefers-color-scheme`.
2. On load, reads `localStorage.getItem('__pb_superusers__/_')`, JSON.parses it to extract `.token`, checks JWT expiry, and redirects to `/_/` if absent/expired.
3. Attaches `Authorization: <raw-jwt>` (no prefix — PocketBase format) on every `fetch()` call; redirects to `/_/` on 401.
4. Renders a discovery form: admin pastes a URL, clicks Discover -> POST `/federation/discover` -> preview card shows domain/version/user count/trail count -> Connect button POSTs to `/federation/follow`.
5. Renders a peer table with Domain, Direction (Outbound/Inbound/Mutual badges), Status (pending/accepted/rejected badges), and conditional action buttons per row:
   - Inbound pending: Approve + Reject
   - Accepted: Disconnect (with `window.confirm` guard using template literal for domain interpolation)
   - Outbound pending: "Pending..." text only
   - Rejected: row shown with opacity-60, no buttons
6. Re-fetches `GET /federation/peers` after every successful action (D-16).
7. Maps `user_count`/`trail_count` from discover API response to `preview.users`/`preview.trails` for template binding.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 (shell + state machine) | f7ffc04c | feat(06-01): build federation admin page shell with Alpine state machine |
| Task 2 (UI + action methods) | f7ffc04c | Same commit — both tasks implemented atomically |

## Deviations from Plan

### Implementation Note

**Tasks 1 and 2 implemented atomically in a single commit.**

- **Reason:** The Alpine `federationApp()` function and HTML markup are tightly coupled. Writing Task 1's state machine separately would have created a partial-file state that fails Task 1's verification (the main element references state fields defined in Task 2's methods).
- **Impact:** None — both task verification checks pass against the single commit.
- **Files modified:** `db/routes/federation_ui.html` (Tasks 1 and 2 combined)
- **Commit:** f7ffc04c

## Verification Results

**Task 1 automated check:** PASS
```
grep -q "function federationApp" -> PASS
grep -q "__pb_superusers__/_" -> PASS
grep -q "darkMode: 'class'" -> PASS
grep -q "alpinejs@3.14.8" -> PASS
```

**Task 2 automated check:** PASS
```
grep -q "Connect a new instance" -> PASS
grep -q "user_count" -> PASS
grep -q "No peer connections yet" -> PASS
grep -q "Undo{Follow} or Reject{Follow}" -> PASS
grep -Eq "x-for=\"peer in peers\"" -> PASS
```

**Overall verification:**
- `grep -c "Bearer|x-html" federation_ui.html` -> 0 (PASS)
- All 6 endpoints referenced: /federation/peers, /federation/discover, /federation/follow, /federation/approve, /federation/reject, /federation/disconnect (PASS)
- Field mapping: `user_count`/`trail_count` consumed in JS, `preview.users`/`preview.trails` rendered in template (PASS)
- Script ordering: theme IIFE (line 11) -> tailwind.config (line 20) -> Tailwind CDN (line 23) -> federationApp (line 27) -> Alpine defer (line 183) (PASS)
- File length: 359 lines >= min_lines 180 (PASS)
- `x-for="peer in peers" :key="peer.follow_id"` present (PASS)
- No `x-show` on `<tr>` elements — conditional row content uses `<template x-if>` (PASS)

## Known Stubs

None. The page fetches live data from Phase 5 API endpoints. No hardcoded or mock data.

## Threat Flags

None beyond the plan threat model. All server-derived values rendered via `x-text` (safe from XSS). Zero `x-html` directives confirmed by grep.

## Self-Check: PASSED

- db/routes/federation_ui.html: FOUND (359 lines)
- Commit f7ffc04c: FOUND (git log confirms)
- Task 1 verification: PASS
- Task 2 verification: PASS
- No Bearer occurrences: PASS (grep -c returns 0)
- No x-html occurrences: PASS (grep -c returns 0)
- All 6 endpoints present: PASS
