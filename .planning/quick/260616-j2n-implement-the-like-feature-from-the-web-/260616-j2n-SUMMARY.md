---
phase: quick-260616-j2n
plan: 01
subsystem: mobile-trail
tags: [flutter, riverpod, trail, like, ui]
requires:
  - TrailNotifier (trailProvider) with trail_like_via_trail expand
  - PUT /trail-like and POST /trail-like/delete API endpoints
  - authProvider.actorId
provides:
  - like()/unlike() methods on TrailNotifier
  - LikeButton ConsumerWidget
  - LikeButton mounted in trail panel stats Wrap
affects:
  - app/lib/components/trail/trail_panel.dart
tech-stack:
  added: []
  patterns:
    - Optimistic state update via state = AsyncData(copyWith(...))
    - Graceful-degradation try/catch leaving state unchanged on API error
key-files:
  created:
    - app/lib/components/trail/like_button.dart
  modified:
    - app/lib/provider/trail/trail_provider.dart
    - app/lib/provider/trail/trail_provider.g.dart
    - app/lib/components/trail/trail_panel.dart
decisions:
  - Used state.value (nullable accessor for this Riverpod version) instead of valueOrNull
metrics:
  duration: ~7 min
  completed: 2026-06-16
---

# Quick Task 260616-j2n: Implement the like feature from the web app Summary

Added the web "like" feature to the Flutter trail panel: a heart button showing total like count that toggles the current user's like via the trail-like API, optimistically updating state and filling red when liked.

## What Was Built

### Task 1: like()/unlike() on TrailNotifier
- Added `Future<void> like(String actorId)` — reads current trail from `state.value`, calls `PUT /trail-like {actor, trail}`, then sets `state` to an `AsyncData` copy with `likeCount + 1` and a new `TrailLike` appended to `expand.trailLikeViaTrail`.
- Added `Future<void> unlike(String actorId)` — calls `POST /trail-like/delete {actor, trail}`, then sets `state` with `likeCount - 1` (clamped at 0) and the current user's `TrailLike` filtered out via `.where((l) => l.actor != actorId)`.
- Both wrap the API call in try/catch and leave `state` unchanged on error (graceful degradation; no AsyncError that would blank the screen).
- Commit: 2f498080

### Task 2: LikeButton widget mounted in trail panel
- Created `app/lib/components/trail/like_button.dart` as a `ConsumerWidget` taking `required Trail trail`.
- Reads actor id from `ref.watch(authProvider).requireValue!.actorId`; computes liked state via `.where((l) => l.actor == actorId).isNotEmpty` (no `firstWhereOrNull` — package:collection is not a dependency).
- Renders an `InkWell` wrapping a `Container` styled like `StatChip` (grey-tinted rounded box) with a vertical `Column`: `FaIcon` solidHeart/heart (red when liked, blueGrey otherwise, size 16), a 4px gap, and the count as `bodySmall` text.
- onTap reads `trailProvider(trail.id).notifier` and calls `unlike(actorId)`/`like(actorId)` based on current liked state.
- Mounted `LikeButton(trail: trail)` as the last child of the trail panel's existing stats `Wrap`, alongside the distance/duration/elevation StatChips. Existing chips untouched.
- Commit: 69d2dd0a

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `valueOrNull` getter undefined in this Riverpod version**
- **Found during:** Verification (flutter analyze after Task 1)
- **Issue:** The plan specified `state.valueOrNull`, but `flutter analyze` reported `The getter 'valueOrNull' isn't defined for the type 'AsyncValue<Trail>'`.
- **Fix:** Switched to `state.value`, the nullable accessor already used throughout the codebase (auth_provider.dart, trail_search_provider.dart, follow_provider.dart, etc.).
- **Files modified:** app/lib/provider/trail/trail_provider.dart
- **Commit:** 2f498080 (folded into Task 1 before commit)

## Codegen

Ran `dart run build_runner build --delete-conflicting-outputs`. No new `@riverpod` annotation was added (the methods live on the existing `TrailNotifier`), so the only generated change was the `_$trailNotifierHash()` value in `trail_provider.g.dart`, committed alongside the provider.

## Verification

- `dart format` — 3 files, clean after formatting.
- `flutter analyze lib/components/trail/like_button.dart lib/provider/trail/trail_provider.dart lib/components/trail/trail_panel.dart` — No issues found.
- Task automated greps for both tasks returned OK.

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: app/lib/components/trail/like_button.dart
- FOUND: commit 2f498080
- FOUND: commit 69d2dd0a
