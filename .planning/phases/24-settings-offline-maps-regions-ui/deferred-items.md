# Phase 24 — Deferred Items

## Pre-existing `flutter test` failures (out of scope, not caused by this phase)

Discovered while running the full `flutter test` suite as part of 24-02's overall verification. Confirmed pre-existing via `git diff 46a81ea6 d0a98970 --stat -- app/test/components/route_planner/ app/lib/components/route_planner/` (empty diff — no files in this area were touched by either 24-01 or 24-02).

| Test | File | Status |
|------|------|--------|
| `SettingsTab renders exactly 5 bucket options and an auto-routing switch` | `app/test/components/route_planner/settings_tab_test.dart` | Failing (pre-existing) |
| `tapping the 'Biking / Road' card calls switchProfile('bicycle', roadOpts)` | `app/test/components/route_planner/settings_tab_test.dart` | Failing (pre-existing) |
| `the option matching the current state is visually marked selected` | `app/test/components/route_planner/settings_tab_test.dart` | Failing (pre-existing) |
| `toggling the auto-routing switch flips autoRoutingEnabled` | `app/test/components/route_planner/settings_tab_test.dart` | Failing (pre-existing) |

Not fixed — out of `files_modified` scope for 24-02-PLAN.md (`app/lib/routes/settings_offline_regions_screen.dart`, `app/lib/routes/settings_screen.dart`, `app/lib/provider/router_provider.dart`). Separate from the previously-known 3 pre-existing failures logged in Phase 18's `deferred-items.md` (`feed_item_test.dart` x2, `settings_screen_test.dart` x1) — that `settings_screen_test.dart` failure no longer applies; 24-02 updated that test directly since its own Task 3 change caused the row-count assertion to go stale (see 24-02-SUMMARY.md Deviations).
