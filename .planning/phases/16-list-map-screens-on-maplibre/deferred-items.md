# Phase 16 — Deferred Items

Out-of-scope issues discovered during execution but not fixed (per Scope
Boundary rule — only auto-fix issues directly caused by the current task's
changes).

## From 16-01 (list map screens)

Full-project `flutter analyze` (37 issues) surfaced pre-existing issues
unrelated to 16-01's three files (`search_map.dart`,
`list_detail_map_screen.dart`, `list_detail_screen.dart` — all individually
clean, "No issues found!"):

- `lib/routes/map_screen.dart:27` — unused `package:wanderer/models/subcategory.dart`
  import. Pre-existing (confirmed present before this plan's first commit).
  In scope for **16-03** (`map_screen.dart` is that plan's file) — fix there,
  not here.
- `lib/components/trail/trail_dropdown.dart:126` — dead code warning. Unrelated file, not touched by Phase 16.
- `lib/routes/settings_categories_screen.dart:551`, `lib/routes/settings_subcategories_screen.dart:523` — `use_build_context_synchronously` info-level lints. Unrelated files.
- `lib/util/icon_util.dart` — ~30 `deprecated_member_use` info-level lints for renamed Font Awesome icon constants (e.g. `behanceSquare` → `squareBehance`). Unrelated, pre-existing, out of Phase 16's scope entirely.
- `test/models/feed_item_test.dart:3` — unused `package:wanderer/models/trail.dart` import. Unrelated test file.
