---
phase: quick-260929-n4d
plan: 01
status: complete
subsystem: app/profile
tags: [flutter, profile, empty-state, l10n]
requirements: [PFE-01, PFE-02, PFE-03, PFE-04]
key-files:
  modified:
    - app/lib/routes/profile_screen.dart
    - app/lib/i18n/app_*.arb (14 locales)
    - app/lib/i18n/app_localizations*.dart (regenerated)
  created:
    - app/test/routes/profile_screen_feed_empty_state_test.dart
commits:
  - e846aa47: "feat(app): add profile feed empty-state strings"
  - 59692beb: "feat(app): show an empty state for profiles without a feed"
---

# Quick 260929-n4d: Profile feed empty state

One-liner: a profile whose feed loaded with zero items now shows "{username} has no activity yet." (or, on your own profile, "You have no activity yet." plus a New Trail button to /trail/create), gated so it never appears while loading or on error.

## What changed

- Two ARB keys, `profile_feed_empty_own` and `profile_feed_empty_other(username)`, added to all 14 locales; `flutter gen-l10n` regenerated the localizations; `untranslated_messages.json` remains `{}`.
- `profile_screen.dart`: extracted `_FeedHeading`, added `_FeedEmptyState`, threaded `isOwn` into `_FeedSection`. The `AsyncLoader` builder shows the empty state only when `items.isEmpty && !feedAsync.isLoading`. An empty feed being pull-to-refreshed renders the mock skeleton items instead. The offline-error and WandererError paths are untouched.
- New widget test pumps the real `ProfileScreen` in a real `GoRouter`; 5 cases (other-empty, own-empty + CTA navigation, loading, error, non-empty).

## Verification

- `flutter analyze`: 13 issues, all pre-existing infos (vendor/, objectbox imports, etc.); none in touched files.
- `flutter test` (full suite): all passed (1186 passed, 1 skipped).
- RED confirmed: the two empty-state cases failed before the lib change; the other three passed.

## Deviations from Plan

**1. [Rule 3 - Blocking] Test harness adjustments (test file only)**
- Actors in the test carry `followerCount: 0` / `followingCount: 0`; a null count renders a Skeletonizer that never lets `pumpAndSettle` finish.
- `_StubFollow` returns synchronously (`FutureOr`), and the test filters RenderFlex "overflowed" reports: the 100px follow-button slot overflows under the test environment's wide font (a test-font artefact, not something this task changes).
- `_StubOwnProfile` awaits `authProvider.future` like the real `OwnProfile`, since the screen derives its own handle from auth.

**2. Plan verification detail not met literally (no impact)**
- The plan's "all ARBs share one line count" check does not hold: `app_en.arb` has a trailing newline (604 lines by `wc -l`) while the other 13 do not (603). This predates the change (HEAD: 590 vs 589) and the same 14 lines were added to every file.

## Known Stubs

None.

## Self-Check: PASSED

- Commits e846aa47 and 59692beb exist on feature/app; nothing pushed.
- Test file and profile_screen.dart present; no changes under web/.
