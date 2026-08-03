---
phase: 36-local-first-recording-automatic-upload
plan: 36-21
reviewed: 2026-08-03T21:30:00Z
depth: standard
diff_base: bdfde398^
files_reviewed: 2
files_reviewed_list:
  - app/lib/components/trail/trail_panel.dart
  - app/test/components/trail/trail_panel_sync_badge_test.dart
findings:
  critical: 0
  warning: 4
  info: 5
  total: 9
status: issues_found
---

# Phase 36-21: Code Review Report

**Reviewed:** 2026-08-03T21:30:00Z
**Depth:** standard (delta-scoped: `9419f872` RED test + `bdfde398` fix)
**Files Reviewed:** 2
**Status:** issues_found

## Summary

The delta is small and surgical: one import, one narrowed guard on the "Offline"
pill (`trail.isLocal` → `trail.isLocal && !isUnsyncedState(trail.syncState)`),
and one new `isUnsyncedState`-gated `SyncStatusChip` below the trail title, plus a
284-line widget test pinning five badge cases.

Verification performed:
- `flutter analyze` on both files — clean.
- `flutter test test/components/trail/trail_panel_sync_badge_test.dart` — 5/5 pass.
- The RED claim holds by construction: pre-`bdfde398` there is no `SyncStatusChip`
  import in `trail_panel.dart` at all, so Case A's
  `find.byType(SyncStatusChip), findsOneWidget` and `find.text('Offline'), findsNothing`
  could not have passed. The test is not vacuous.
- Reachability of the invariants the fix leans on was traced through
  `TrailEntity.toModel()` (hardcodes `isLocal: true`), `Trail.syncState`
  (`@Default(TrailSyncState.synced)` + `includeFromJson/includeToJson: false`),
  `trailLibraryProvider` (`savedByUserIds` query — never contains an unsynced
  row, per `local_trail_store.dart:85`), and both `TrailPanel` mount sites
  (`trail_detail_screen.dart:162`, `library_detail_screen.dart:36`).

**No BLOCKER found in this delta.** The state partition the fix claims (D-10) does
hold today for every reachable trail, and the five badge states
(unsynced-pending / unsynced-in-flight / unsynced-failed / downloaded / remote)
each resolve to exactly one indicator.

The four warnings are: a **user-visible localization regression for 13 locales**
(the delta replaces a translated string with an English-only one on the detail
screen), a **guard asymmetry** that makes the fix depend on an invariant nothing
enforces, and two **test-coverage holes** that leave the exact regression this
plan closed unpinned — specifically, the guard could be narrowed to
`pending || failed` and all five tests would still pass.

## Warnings

### WR-01: Delta swaps a fully-translated string for an English-only one on the detail screen — 13 locales regress

**File:** `app/lib/components/trail/trail_panel.dart:205`, `:296-302`
**Issue:**
Before this commit an unsynced trail's detail screen rendered `l18n.offline`,
which is translated in **all 14** ARB files. After it, that trail renders
`sync_pending` / `sync_uploading` / `sync_failed`, which exist **only** in
`app_en.arb`:

```
app_cs.arb pending=0 uploading=0 failed=0 offline=1
app_de.arb pending=0 uploading=0 failed=0 offline=1
...   (13 locales identical)
app_en.arb pending=1 uploading=1 failed=1 offline=1
```

gen-l10n's template fallback makes this compile and ship silently —
`lib/i18n/app_localizations_de.dart:758` reads
`String get sync_pending => 'Waiting to upload';`. So a German user who records a
trail now sees an English sentence where a German word used to be, on the trail's
primary screen.

This is *known* (the strings are listed in the committed
`lib/i18n/untranslated_messages.json` for all 13 locales, and `l10n.yaml`'s own
comment names "the sync-status surface" as having "shipped English-only without
anyone noticing"). It is still a regression introduced by *this* delta, because
this delta is what puts those strings on a surface that previously had a
translated one. The card/list-item mount sites added the strings alongside
existing badges; this one **replaces** a translated badge.

**Fix:** Add the three keys to the 13 non-English ARBs (machine translation is
acceptable for a status chip and is better than English), then regenerate:

```bash
# lib/i18n/app_de.arb (and cs, es, eu, fr, hu, it, nl, no, pl, pt, ru, zh)
  "sync_pending": "Warten auf Upload",
  "sync_uploading": "Wird hochgeladen…",
  "sync_failed": "Upload fehlgeschlagen · Zum Wiederholen tippen",

flutter gen-l10n   # untranslated_messages.json should shrink by 39 entries
```

If translating is out of scope for a gap-closure plan, the cheaper mitigation is
to keep the translated `offline` pill *alongside* the chip for unsynced trails
rather than replacing it — but that contradicts the plan's REC-03 rationale, so
translation is the correct fix.

---

### WR-02: Guard asymmetry — the sync chip is not gated on `isLocal`, the Offline pill is

**File:** `app/lib/components/trail/trail_panel.dart:205` vs `:296`
**Issue:**
The two new guards are deliberately different shapes:

```dart
if (trail.isLocal && !isUnsyncedState(trail.syncState)) ...[   // line 205, Offline pill
if (isUnsyncedState(trail.syncState)) ...[                      // line 296, sync chip
```

The 12-line D-10 comment above line 205 argues the partition is total — and it is,
*for local trails*. But the chip's guard drops `isLocal` entirely, so the
correctness of the whole arrangement rests on the unstated converse invariant
`syncState != synced ⇒ isLocal == true`. That invariant is real today (the only
producer of a non-`synced` `syncState` is `TrailEntity.toModel()`
(`trail_entity.dart:362`), which hardcodes `isLocal: true` twelve lines earlier at
`:355`; `Trail.syncState` is `@Default(TrailSyncState.synced)` and JSON-excluded,
so no API-parsed trail can carry one) — but nothing states or enforces it, and it
is exactly the class of provenance-vs-state conflation that produced WR-11 and
this very UAT gap.

If it is ever broken — e.g. an optimistic-update path doing
`serverTrail.copyWith(syncState: TrailSyncState.pending)` — the panel will
simultaneously render "Waiting to upload" *and* hide the summit-log/comment tabs
(`showsServerTabs`, line 72, keys off the same predicate) on a trail that is
demonstrably on the server. The failure is silent and looks like data loss to the
user.

**Fix:** Compute the predicate once and make both guards read from the same
provenance-qualified booleans, so the asymmetry is either intentional-and-visible
or gone:

```dart
final isUnsynced = isUnsyncedState(trail.syncState);
final showsServerTabs = !isUnsynced;
// ...
if (trail.isLocal && !isUnsynced) ...[      // Offline pill
if (trail.isLocal && isUnsynced) ...[       // sync chip — same axis, same guard
```

Adding `trail.isLocal` to the chip guard changes no reachable behaviour today
(every unsynced trail is local) and removes the dependency on an unenforced
invariant.

---

### WR-03: No panel-level coverage for `TrailSyncState.uploading` — the guard can be narrowed back without a test failing

**File:** `app/test/components/trail/trail_panel_sync_badge_test.dart:136-229`
**Issue:**
The panel's guard is `isUnsyncedState(...)`, i.e. `!= synced`, which spans three
states. The tests exercise only two of them: `pending` (Cases A and B) and
`failed` (Case C). The persisted `uploading` state is never mounted through
`TrailPanel`.

Case B does render "Uploading…", but only by way of the *in-flight set*
(`inFlight: {'local-1-0'}`) with the row still persisted as `pending` — it takes
the `inFlight.contains(localId)` branch in `sync_status_chip.dart`, not the
`trail.syncState == TrailSyncState.uploading` branch. Those are different inputs to
the panel's guard: the chip's own test covers the persisted-`uploading` branch
(`sync_status_chip_test.dart:166-170`), but the *panel* gate is what this plan
changed.

Concretely: rewrite line 296 as
`if (trail.syncState == TrailSyncState.pending || trail.syncState == TrailSyncState.failed)`
and all five tests still pass — while a trail whose row is persisted `uploading`
(the app was killed mid-drain and relaunched, the scenario
`sync_status_chip.dart`'s own comment calls out) falls back to showing the generic
"Offline" pill. That is precisely the round-2 UAT gap, re-opened, with green tests.

**Fix:** Table-drive the unsynced cases over all three states, or add a sixth test:

```dart
for (final state in TrailSyncState.values.where(isUnsyncedState)) {
  testWidgets('unsynced ($state): chip shown, no Offline pill', (tester) async {
    final controller = ScrollController();
    addTearDown(controller.dispose);
    final trail = baseFixture.copyWith(
      id: '', localId: 'local-1-0', isLocal: true, syncState: state,
    );
    await tester.pumpWidget(_harness(trail, scrollController: controller));
    await tester.pump();   // pump, not pumpAndSettle -- uploading spins forever
    expect(find.byType(SyncStatusChip), findsOneWidget);
    expect(find.text('Offline'), findsNothing);
  });
}
```

The loop also future-proofs the file against a fifth enum member being added to
`TrailSyncState` without a corresponding panel case.

---

### WR-04: The placement rationale (overflow avoidance) is untested — no narrow-viewport case

**File:** `app/test/components/trail/trail_panel_sync_badge_test.dart:92-125`
**Issue:**
The 11-line comment at `trail_panel.dart:285-295` justifies putting the chip below
the title rather than in the badge `Row` with a specific, checkable claim: the Row
"has no overflow protection and already holds the date `Text`, so adding a ~180px
'Upload failed · Tap to retry' chip to it would overflow on a narrow screen."

No test encodes that. `_harness` uses the default 800x600 test surface — wider than
every phone the app targets (360-430 logical px) — and every fixture has
`date == null` (`Trail.empty()`), so the date `Text` that the rationale names as
the co-occupant is never present in any of the five cases. A regression that moved
the chip back into the Row would produce a yellow-stripe `RenderFlex overflowed`
on a real device and a green test suite here.

**Fix:** Add one case at phone width with a date present, so the documented hazard
is actually guarded:

```dart
testWidgets('failed chip does not overflow at 360px with a date', (tester) async {
  tester.view.physicalSize = const Size(360, 800);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);
  final controller = ScrollController();
  addTearDown(controller.dispose);

  final trail = baseFixture.copyWith(
    id: '', localId: 'local-1-0', isLocal: true,
    syncState: TrailSyncState.failed, date: DateTime(2026, 8, 3),
  );
  await tester.pumpWidget(_harness(trail, scrollController: controller));
  await tester.pumpAndSettle();
  expect(find.text('Upload failed · Tap to retry'), findsOneWidget);
  // pumpAndSettle rethrows any RenderFlex overflow as a test failure.
});
```

## Info

### IN-01: `isUnsyncedState(trail.syncState)` is now evaluated three times in one `build()`

**File:** `app/lib/components/trail/trail_panel.dart:72`, `:205`, `:296`
**Issue:** `showsServerTabs` (line 72) already caches `!isUnsyncedState(trail.syncState)`;
lines 205 and 296 recompute the same predicate instead of reading it. Three call
sites for one decision is how the two halves drift apart later — and line 205's
`!isUnsyncedState(...)` is literally `showsServerTabs`, which is not obvious to a
reader because the name describes tabs, not sync.
**Fix:** Hoist `final isUnsynced = isUnsyncedState(trail.syncState);` next to line 72
and derive `showsServerTabs` from it; see WR-02's snippet, which fixes both.

---

### IN-02: New file is not `dart format` clean; import inserted out of alphabetical order

**File:** `app/lib/components/trail/trail_panel.dart:14`, `:205`
**Issue:**
- Line 205 exceeds the 80-column page width; `dart format` rewraps it to
  `if (trail.isLocal &&\n    !isUnsyncedState(trail.syncState)) ...[`.
- The new `sync_status_chip.dart` import (line 14) is placed *before*
  `summit_log_list.dart` (line 15), breaking the otherwise-alphabetical
  `components/trail/` import block (`summit_` sorts before `sync_`).

Severity is Info, not Warning, because the repo is not uniformly formatted today
(`dart format --set-exit-if-changed lib/ test/` reports 84 of 482 files changed)
and no CI workflow enforces it — this is not a rule the delta is uniquely
violating.
**Fix:** `dart format lib/components/trail/trail_panel.dart` and move the import
one line down.

---

### IN-03: Three near-identical `_StubAuth` / `_StubTrailSync` fixtures now exist across three test files

**File:** `app/test/components/trail/trail_panel_sync_badge_test.dart:57-90`
**Issue:** The file's own header documents the duplication ("Copied from
`trail_dropdown_menu_test.dart`", "Copied from `sync_status_chip_test.dart`"), and
the `_StubAuth` copy had to be *modified* (sync `build()`) because the original's
async shape breaks `requireValue`. That is exactly the state in which copies drift:
a `UserEntity` field addition now needs three edits, and the two `_StubAuth`
variants have subtly different loading semantics with nothing pointing that out
from the other file's side.
**Fix:** Extract to `test/support/stubs.dart` with both auth shapes as named
constructors (`_StubAuth.immediate()` / `_StubAuth.async()`), and have the three
test files import it.

---

### IN-04: Case B is the only case with no negative assertions

**File:** `app/test/components/trail/trail_panel_sync_badge_test.dart:184-185`
**Issue:** Cases A, C, D and E each assert both what *is* rendered and what is
*not* (`findsNothing` on the Offline pill or on `SyncStatusChip`). Case B asserts
only the two positives, so it would not catch a regression that rendered
"Uploading…" *and* the generic "Offline" pill simultaneously — the double-badge
state the plan exists to prevent.
**Fix:** Add the two missing lines:

```dart
expect(find.byType(SyncStatusChip), findsOneWidget);
expect(find.text('Offline'), findsNothing);
```

---

### IN-05: The retry affordance the delta surfaces has a ~24px tap target

**File:** `app/lib/components/trail/trail_panel.dart:298-301` (via
`sync_status_chip.dart:_Chip`)
**Issue:** The failed-state chip's `InkWell` wraps a `Container` with
`vertical: 4` padding around an 11px label — roughly 22-24 logical px tall, well
under the 44-48px minimum for a touch target (Material guidance; the project's
design context commits to WCAG 2.1 AA). This delta makes the detail screen the
primary place a user retries a failed upload, so the undersized target now matters
more than it did on a scrolling card list.

The defect lives in `sync_status_chip.dart`, which is outside this delta and was
covered by the earlier phase review — logged here only because this change is what
promotes it to a primary recovery path.
**Fix:** Wrap the failed-state `InkWell` in a
`ConstrainedBox(constraints: const BoxConstraints(minHeight: 44))`, or give the
chip `MaterialTapTargetSize.padded` behaviour via a surrounding
`SizedBox(height: 44)` with the visual chip centered inside.

---

## Notes on scope

Two adjacent issues were traced and deliberately **not** filed, to avoid
re-litigating code the earlier `36-REVIEW.md` already covered:

1. `_TabContentState._index` (`trail_panel.dart:411`, `:440`) is not clamped when
   `widget.children` shrinks from 3 to 1 — a `synced → unsynced` transition while
   the user sits on the Comments tab would throw `RangeError`. The transition is
   not reachable today (`local_trail_store` only moves *local* rows out of
   `synced`), and `36-REVIEW.md:641` already flagged the child-count derivation.
2. `library_detail_screen.dart:36` mounts `TrailPanel` with no `Scaffold`/`Material`
   ancestor in its own build, and the new failed-state chip introduces an
   additional `InkWell` (which asserts `debugCheckHasMaterial`). Not reachable:
   `trailLibraryProvider` queries `savedByUserIds`, and
   `local_trail_store.dart:85` states `saveNewLocalTrail` never writes that field,
   so an unsynced row can never reach that screen. The panel also already mounts
   `InkWell`s on that route today.

---

_Reviewed: 2026-08-03T21:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
