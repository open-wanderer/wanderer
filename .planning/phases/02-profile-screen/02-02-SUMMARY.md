---
phase: 02-profile-screen
plan: 02
subsystem: mobile-app
tags: [flutter, riverpod, freezed, follow, social, provider]
dependency_graph:
  requires:
    - 02-01 (profileProvider, OwnProfile, apiProvider, authProvider, toastProvider)
  provides:
    - FollowState Freezed model (app/lib/models/follow.dart)
    - followProvider auto-dispose family (app/lib/provider/profile/follow_provider.dart)
  affects:
    - 02-04 (ProfileScreen Follow button will consume followProvider)
tech_stack:
  added: []
  patterns:
    - "@riverpod auto-dispose family keyed on profileActorId"
    - "Optimistic state update with rollback on error"
    - "Freezed model without JSON serialization (hand-built from API map)"
key_files:
  created:
    - app/lib/models/follow.dart
    - app/lib/models/follow.freezed.dart
    - app/lib/provider/profile/follow_provider.dart
    - app/lib/provider/profile/follow_provider.g.dart
  modified:
    - app/lib/provider/auth_provider.g.dart (hash update from build_runner)
    - app/lib/provider/objectbox_store_provider.g.dart (hash update)
    - app/lib/provider/profile/profile_feed_provider.g.dart (hash update)
    - app/lib/provider/profile/profile_lists_provider.g.dart (hash update)
    - app/lib/provider/profile/profile_provider.g.dart (hash update)
    - app/lib/provider/profile/profile_trails_provider.g.dart (hash update)
decisions:
  - "FollowState defined in follow.dart (not in follow_provider.dart) to keep model separate from provider — no freezed part in follow_provider.dart"
  - "Use ref.read(authProvider).requireValue (synchronous) per plan spec, consistent with keepAlive auth provider"
  - "profileActorId referenced directly in toggle() — generated _$FollowNotifier base class exposes String get profileActorId getter"
  - "Doubled && in PocketBase filter string is intentional (PocketBase filter AND operator, not URL param separator)"
metrics:
  duration: "8m 40s"
  completed: "2026-06-08T08:57:00Z"
  tasks_completed: 2
  files_created: 4
  files_modified: 6
---

# Phase 02 Plan 02: FollowProvider Summary

**One-liner:** FollowState Freezed model and FollowNotifier auto-dispose family provider implementing optimistic follow/unfollow toggle with rollback and error toast, keyed on profileActorId.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create FollowState Freezed model | 2c3a7150 | app/lib/models/follow.dart, app/lib/models/follow.freezed.dart |
| 2 | Create FollowNotifier auto-dispose family provider | f60753a6 | app/lib/provider/profile/follow_provider.dart, app/lib/provider/profile/follow_provider.g.dart |

## What Was Built

### FollowState (app/lib/models/follow.dart)

A Freezed model encapsulating the follow relationship state:
- `isFollowing` (bool, required) — current follow state
- `followRecordId` (String?, nullable) — PocketBase record ID needed for DELETE unfollow, null when not following
- `isLoading` (@Default(false) bool) — true while optimistic toggle API call is in flight

No `fromJson` factory — FollowState is hand-built from the raw API response map inside the provider (no `.g.dart` part).

### FollowNotifier / followProvider (app/lib/provider/profile/follow_provider.dart)

An auto-dispose Riverpod family provider keyed on `profileActorId` (not handle — the follow API uses actor IDs per Pitfall 2 in RESEARCH.md).

**build(String profileActorId) — FutureOr<FollowState>:**
- Reads auth state synchronously via `ref.read(authProvider).requireValue`
- Returns `FollowState(isFollowing: false)` if user is null (not authenticated)
- GETs `/follow?filter=follower='${user.actorId}'&&followee='$profileActorId'` (D-13)
- Returns FollowState with `isFollowing: true, followRecordId: follow['id']` if record exists

**toggle() — Future<void>:**
- Guards: returns early if `current == null || current.isLoading`
- Captures `wasFollowing`, sets optimistic `AsyncData(current.copyWith(isFollowing: !wasFollowing, isLoading: true))` (D-15)
- If following: PUTs `/follow` with `{followee: profileActorId}`, extracts new record id from response
- If unfollowing: DELETEs `/follow/${current.followRecordId}`
- On error: rolls back to `current.copyWith(isLoading: false)` + shows `ToastType.error` toast

## Verification Results

- `dart run build_runner build --delete-conflicting-outputs` exits 0
- `app/lib/models/follow.freezed.dart` generated
- `app/lib/provider/profile/follow_provider.g.dart` generated with `followProvider` family symbol
- `flutter analyze lib/models/follow.dart` — No issues found
- `flutter analyze lib/provider/profile/follow_provider.dart` — No issues found
- `flutter analyze lib/provider/profile/follow_provider.dart lib/models/follow.dart` — No issues found

## Deviations from Plan

None — plan executed exactly as written.

The PATTERNS.md suggested adding `part 'follow_provider.freezed.dart';` to follow_provider.dart, but the plan explicitly stated "NO freezed part directive here" since FollowState lives in follow.dart. The plan's instruction was followed. The build confirmed this was correct — no freezed output needed for follow_provider.dart.

## Known Stubs

None — all fields wired to actual API responses.

## Threat Surface Scan

No new threat surface beyond what is already documented in the plan's threat model:
- T-02-03: Server enforces auth on PUT/DELETE /follow via cookie session
- T-02-04: followRecordId is provider-internal, never rendered to UI
- T-02-05: profileActorId is server-sourced actor ID, not free user input

## Self-Check: PASSED

- [x] app/lib/models/follow.dart exists
- [x] app/lib/models/follow.freezed.dart exists
- [x] app/lib/provider/profile/follow_provider.dart exists
- [x] app/lib/provider/profile/follow_provider.g.dart exists
- [x] Commit 2c3a7150 (Task 1) verified in git log
- [x] Commit f60753a6 (Task 2) verified in git log
