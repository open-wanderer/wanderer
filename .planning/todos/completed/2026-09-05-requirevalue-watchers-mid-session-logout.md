---
created: 2026-09-05T00:00:00.000Z
title: Harden requireValue auth watchers against a logout arriving mid-session
area: app
files:
  - app/lib/provider/auth_provider.dart
  - app/lib/components/trail/trail_panel.dart
  - app/lib/components/trail/like_button.dart
  - app/lib/components/trail/waypoint_card.dart
  - app/lib/routes/list_detail_screen.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/routes/trail_detail_map_screen.dart
---

## Problem

`logout()` opens with `state = const AsyncLoading()` (`app/lib/provider/auth_provider.dart:222`).
That assignment carries no previous value, so `AsyncValue.requireValue` throws rather than
returning the old entity.

Six widgets read auth that way:

| File | Line |
|---|---|
| `app/lib/components/trail/trail_panel.dart` | 55 |
| `app/lib/components/trail/like_button.dart` | 15 |
| `app/lib/components/trail/waypoint_card.dart` | 29 |
| `app/lib/routes/list_detail_screen.dart` | 115 |
| `app/lib/routes/navigation_screen.dart` | 1366 |
| `app/lib/routes/trail_detail_map_screen.dart` | 106 |

Today this is masked by timing. A cached session the server has since rejected (password change,
account deletion, admin revoke — inside a still-valid token window, so the cookie has not
expired) is validated *during* the splash await, and `logout()` runs before any of these widgets
mount.

The fast-gate change in `2026-09-05-fast-local-auth-gate-on-splash.md` moves that validation off
the critical path. The rejection can now land with `/map` up and a trail panel open, in the
window before the router redirect fires — and the widget throws instead of flickering.

`navigation_screen.dart:1366` is the worst case: a logout landing during an active recording.

## Approach

Two candidate fixes, not mutually exclusive:

1. **Preserve the value across the loading transition** — `state = const AsyncLoading()` →
   `AsyncLoading<UserEntity?>().copyWithPrevious(state)` in `logout()` (and check `register`,
   `login` and `loginWithOAuth`, which use the same bare pattern). Cheapest, fixes all six at
   once, and keeps the UI rendering the outgoing user for the frame or two before the redirect.
2. **Make the reads defensive** — `.value` with a null guard instead of `.requireValue!` at each
   site. More churn, but removes the invariant rather than restoring it.

Prefer (1) as the fix and treat (2) as belt-and-braces for `navigation_screen.dart`, where the
consequence of a throw is the loss of an in-progress recording.

Worth checking whether `purgeAccountScopedData` (called from `logout()`) can run while those
screens hold references to purged entities — same window, separate hazard.

## Done when

- Simulating a 401 from the background validation while `/map` is open with a trail panel
  expanded routes to `/welcome` without throwing.
- The same during an active recording on `navigation_screen` does not lose the session.
