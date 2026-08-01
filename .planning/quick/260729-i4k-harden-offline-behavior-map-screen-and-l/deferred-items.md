# Deferred Items — quick-260729-i4k

## Pre-existing `wifiSlash` comment in `settings_offline_regions_screen.dart`

`lib/routes/settings_offline_regions_screen.dart:254` carries a pre-existing
doc comment literally containing the substring `wifiSlash` (documenting why
`linkSlash` was chosen instead, added in an earlier phase). This file is not
in this plan's `files_modified` list and the comment is out of scope for this
task (SCOPE BOUNDARY — pre-existing, unrelated file).

This plan's own Task 2 verify command greps repo-wide for `wifiSlash` and
expects zero matches; that pre-existing occurrence means the literal grep
count is 1, not 0, independent of anything changed here. This plan's own new
doc comment in `wanderer_offline_state.dart` was worded to avoid introducing
a second occurrence (mirroring the Phase 16-02 precedent of avoiding a
grep-sensitive literal substring in a doc comment), but the pre-existing one
was left untouched since fixing it is out of this plan's scope.

## Pre-existing `flutter test` failures unrelated to this plan

A full-suite `flutter test` run (after all 5 tasks) shows 4 failures, all in
`test/components/route_planner/settings_tab_test.dart`. These are caused by
an unrelated commit (`28cabf9d "Fix route planner leg calculation"`, made
outside this quick task's session) touching route-planner files this plan's
`files_modified` list never references (`route_anchor_provider.dart`,
`route_planner_screen.dart`, `planned_gpx_provider.dart`, etc.). None of this
plan's 5 tasks touch `settings_tab.dart` or its dependencies. Confirmed
out of scope.

The 3 previously-documented pre-existing failures referenced in this plan's
`<verification>` section (`feed_item_test.dart` x2, `settings_screen_test.dart`
x1, from `.planning/phases/18-.../deferred-items.md`) no longer reproduce:
`feed_item_test.dart` no longer exists as a file, and `settings_screen_test.dart`
now passes cleanly. Both were resolved by unrelated work between Phase 18 and
now.

## Two build-artifact regenerations left uncommitted

Running `dart run build_runner build` (required to generate
`online_status_provider.g.dart` in Task 1) also regenerated
`app/lib/provider/planned_gpx_provider.g.dart` and
`app/lib/provider/trail/trail_download_state_provider.g.dart` — both are
derived from source files already modified by unrelated concurrent work
(the `28cabf9d` route-planner commit), not by this plan. These two
regenerated files were deliberately left unstaged/uncommitted throughout
all 5 task commits (out of scope, pre-existing source drift) and remain so
after this plan's final task.
