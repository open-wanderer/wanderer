---
phase: 02-profile-screen
plan: "04"
subsystem: mobile/flutter
tags: [flutter, riverpod, profile, screen, routing, social, follow, infinite-scroll]
dependency_graph:
  requires:
    - 02-01 (ListCard, FeedItemShimmer widgets)
    - 02-02 (FollowProvider, FollowState)
    - 02-03 (FeedItemCard widget)
    - Phase 1 providers: profileProvider, ownProfileProvider, profileFeedProvider, profileListsProvider, authProvider
  provides:
    - app/lib/routes/profile_screen.dart (ProfileScreen ConsumerStatefulWidget — full profile UI)
    - app/lib/provider/router_provider.dart (/profile shell route + /profile/:handle top-level route)
  affects:
    - All navigation to /profile (bottom nav tab) now shows full profile
    - All navigation to /profile/:handle now works full-screen
tech_stack:
  added: []
  patterns:
    - ConsumerStatefulWidget with ScrollController lifecycle (initState/dispose)
    - CustomScrollView + SliverAppBar(pinned, expandedHeight) + FlexibleSpaceBar
    - 80% scroll trigger → profileFeedProvider.loadNextPage() guarded by hasMore && !isLoading
    - isOwn check (handle==null) gating Settings/Share vs Follow action buttons
    - _ProfileHeaderBackground: CircleAvatar + dicebear fallback + SafeArea
    - _BioSection: StatefulWidget with 150-char truncation + Show more/less toggle
    - _ListsPreview: ConsumerWidget watching profileListsProvider, horizontal ListView
    - _FollowButton: ConsumerWidget watching followProvider(actor.id) with optimistic toggle
    - RefreshIndicator wrapping CustomScrollView for pull-to-refresh
    - GoRoute(/profile/:handle) outside ShellRoute — matches /trail/:id pattern
key_files:
  created: []
  modified:
    - app/lib/routes/profile_screen.dart (rewritten — ConsumerWidget stub → full ConsumerStatefulWidget profile)
    - app/lib/provider/router_provider.dart (ShellRoute /profile updated; /profile/:handle top-level added)
decisions:
  - "isOwn = widget.handle == null (display decision only; security boundary is server-side per T-02-09)"
  - "WandererError used for actor async error state (Claude discretion — matches trail_detail_screen pattern)"
  - "_handle getter uses ref.read(authProvider).value?.preferredUsername for own profile — synchronous read (not watch) since handle resolves once at scroll time, not in build"
  - "RefreshIndicator wraps the full CustomScrollView, onRefresh invalidates actor + feed + lists providers"
  - "_FollowButton uses WandererButton primary/secondary styling — primary when not following, secondary when following"
  - "/profile/:handle placed immediately after /trail/:id in top-level routes list — matches RESEARCH Pattern 8"
  - "feedHasMore footer uses FeedItemShimmer not a spinner — visually consistent with initial load skeleton"
metrics:
  duration: "~25 minutes"
  completed: "2026-06-08T09:14:27Z"
  tasks_completed: 2
  tasks_pending: 1
  files_created: 0
  files_modified: 2
---

# Phase 02 Plan 04: ProfileScreen Assembly and Route Wiring Summary

**One-liner:** ProfileScreen rewritten as ConsumerStatefulWidget with CustomScrollView+SliverAppBar header (avatar/handle/joined/counts), own/remote action buttons (Settings+Share vs Follow), bio with expand/collapse, horizontal lists preview, and an infinite feed that loads the next page at 80% scroll; routes wired for both /profile (bottom nav) and /profile/:handle (full-screen).

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Build full ProfileScreen | 991936ac | app/lib/routes/profile_screen.dart |
| 2 | Wire /profile (shell) and /profile/:handle (top-level) routes | db1ee826 | app/lib/provider/router_provider.dart |

## Tasks Pending (checkpoint)

| Task | Name | Type | Status |
|------|------|------|--------|
| 3 | End-of-phase human verification | checkpoint:human-verify | awaiting user |

## What Was Built

### ProfileScreen (app/lib/routes/profile_screen.dart)

A complete `ConsumerStatefulWidget` replacing the logout/library stub. Key design:

**Widget hierarchy:**
- `ProfileScreen(String? handle)` — `ConsumerStatefulWidget`
- `_ProfileScreenState` — `ConsumerState<ProfileScreen>` (owns `ScrollController`)
- `_ProfileHeaderBackground(actor)` — `StatelessWidget` (avatar, handle@domain, joined, follower/following)
- `_BioSection(summary)` — `StatefulWidget` (expand/collapse, empty state)
- `_ListsPreview(handle)` — `ConsumerWidget` (horizontal ListCard scroll, watches profileListsProvider)
- `_FollowButton(profileActorId)` — `ConsumerWidget` (watches followProvider, drives toggle)

**Data wiring:**
- Own profile (`handle == null`): watches `ownProfileProvider`
- Remote profile: watches `profileProvider(widget.handle!)`
- Feed: watches `profileFeedProvider(h)` where h = `widget.handle ?? authProvider.value?.preferredUsername`
- Lists: watches `profileListsProvider(h)` inside `_ListsPreview`
- Follow state: watches `followProvider(actor.id)` inside `_FollowButton`

**Scroll pagination (CONT-03):**
- `_onScroll` checks `pos.pixels / pos.maxScrollExtent >= 0.8`
- Guards: `hasContentDimensions`, `maxScrollExtent > 0`, `hasMore == true`, `!isLoading`
- Calls `ref.read(profileFeedProvider(h).notifier).loadNextPage()`

**Loading states:**
- Actor: `CircularProgressIndicator` while loading, `WandererError` on error
- Feed initial load: 3 x `FeedItemShimmer` placeholders
- Feed next-page: 1 x `FeedItemShimmer` footer when `feedHasMore`
- Lists: `CircularProgressIndicator` during load, silently hidden on error/empty

**Action buttons (HEAD-03/04/05):**
- Own: `IconButton(FaIcon(gear) -> /settings)` + `IconButton(FaIcon(shareNodes) -> no-op Phase 3)`
- Remote: `_FollowButton` with `WandererButton(primary/secondary, loading, onPressed)`
- Follow toggle: non-null `() {}` onPressed when `isLoading` (Pitfall 6 — spinner shows)

**Threat mitigations applied (T-02-09 to T-02-12):**
- T-02-09: `isOwn` is display-only; server enforces authorization
- T-02-10: handle from path param is string-only; server validates
- T-02-11: all NetworkImages have `onBackgroundImageError` fallbacks
- T-02-12: own public profile data — no additional disclosure

### Router (app/lib/provider/router_provider.dart)

Two targeted edits only:
1. ShellRoute `/profile` child: `ProfileScreen()` to `const ProfileScreen(handle: null)` (NAV-01)
2. New top-level `GoRoute(path: '/profile/:handle', ...)` added after `/trail/:id` (NAV-02)

## Verification

```
flutter analyze lib/routes/profile_screen.dart               -> No issues found!
flutter analyze lib/provider/router_provider.dart lib/routes/profile_screen.dart -> No issues found!
flutter analyze lib/                                          -> 42 issues, all pre-existing (deprecated FA icons in icon_util.dart + 3 pre-existing warnings) — ZERO new errors introduced by Phase 2
```

## Deviations from Plan

None — plan executed exactly as written.

The plan's `_onScroll` spec included `if (pos.maxScrollExtent <= 0) return;` as an additional guard — this was included exactly as specified.

## Known Stubs

Two intentional Phase 3 stubs:
1. **Settings button:** `onPressed: () => context.push('/settings')` — `/settings` route does not yet exist. The plan explicitly states this is correct: "pushing it now is correct, it will resolve once Phase 3 adds the route."
2. **Share button:** `onPressed: () {}` — share screen is a Phase 3 dependency. Button is rendered; navigation wired in Phase 3.

Neither stub prevents the plan's goal.

## Threat Flags

None — all threats in the plan's threat model are addressed.

## Self-Check: PASSED

- [x] app/lib/routes/profile_screen.dart modified — FOUND
- [x] app/lib/provider/router_provider.dart modified — FOUND
- [x] Commit 991936ac (Task 1) — FOUND
- [x] Commit db1ee826 (Task 2) — FOUND
- [x] `class ProfileScreen extends ConsumerStatefulWidget` — FOUND
- [x] `final String? handle;` and `const ProfileScreen({super.key, this.handle})` — FOUND
- [x] `CustomScrollView(`, `SliverAppBar(`, `pinned: true`, `FlexibleSpaceBar(` — FOUND
- [x] `ref.watch(ownProfileProvider)` and `ref.watch(profileProvider(widget.handle!))` — FOUND
- [x] `_onScroll` with `>= 0.8` and `loadNextPage()` — FOUND
- [x] `initState` creates `_scrollController`, `dispose()` disposes it — FOUND
- [x] Settings + Share for own; `_FollowButton(profileActorId: actor.id)` for remote — FOUND
- [x] `'No bio yet.'` empty state and Show more/Show less toggle — FOUND
- [x] `profileListsProvider` + `ListCard` in horizontal ListView — FOUND
- [x] `FeedItemCard(item: ..., profileActor: actor)` + `FeedItemShimmer()` — FOUND
- [x] `followerCount ?? 0` and `followingCount ?? 0` — FOUND
- [x] `const ProfileScreen(handle: null)` in ShellRoute /profile — FOUND
- [x] `/profile/:handle` as top-level GoRoute outside ShellRoute — FOUND
- [x] `flutter analyze lib/` — no new errors introduced by Phase 2
