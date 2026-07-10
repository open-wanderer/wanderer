---
quick_id: 260710-ptb
description: Clean up comments across all files added/changed in the v1.4 milestone
mode: quick
---

# Quick Task 260710-ptb: Clean up v1.4 comments

## Scope

All 42 Dart source/test files touched by the unpushed v1.4 milestone (`git diff --name-only @{u}..HEAD -- app/`, excluding generated `.g.dart`/`.freezed.dart` files and files since deleted). Full list in `260710-ptb-files.txt` alongside this plan.

## Task 1: Strip non-substantive comments

For each file in scope, remove comments that:
- Narrate what was just changed/added/removed ("added X", "now uses Y", "removed Z")
- Reference phase/task/plan identifiers (e.g. "Phase 16", "lem-02", "quick-260710", issue numbers tied to this milestone's internal tracking)
- Restate what the following line of code obviously does
- Are leftover TODO/FIXME tied to completed milestone work that is no longer actionable

Keep comments that explain:
- Non-obvious WHY (a constraint, a subtle invariant, a workaround for a specific bug)
- Behavior that would surprise a reader
- Public API doc comments (`///`) describing purpose/usage, trimmed of any narration cruft

**Verify:** `git diff` per file reads cleanly — no dangling comment fragments, code untouched. `flutter analyze` (or targeted `dart analyze` on touched files) shows no new errors.

**Done:** All 42 files reviewed; remaining comments are exclusively WHY/invariant/workaround explanations or clean API docs.
