# Deferred Items — Phase 18

Out-of-scope items discovered during execution. Not fixed per the executor's
scope boundary rule (only auto-fix issues directly caused by the current
task's changes).

## Pre-existing `flutter test` failures (confirmed unrelated to 18-01)

Found during 18-01 Task 2's `flutter test` verification gate. Confirmed via
`git stash` + re-run on the pre-18-01 commit (`c06d9546`) that all three
failures reproduce identically without any of 18-01's changes applied —
these are pre-existing failures, not regressions introduced by the
`LocationMarkerPosition`/`ServiceDisabledException` relocation.

- `test/models/feed_item_test.dart`: `FeedItem.fromJson type "list" returns FeedItemList with ListSearchResult`
- `test/models/feed_item_test.dart`: `FeedItem.fromJson type "trail" returns FeedItemTrail with TrailSearchResult`
- `test/routes/settings_screen_test.dart`: `settings screen lists all 5 rows in D-06 order (SETNAV-01)` — fails with "is too many" `ListTile` finder assertion, unrelated to maps/location code.

Not fixed here — out of scope for Phase 18 (map/flomp-fork removal). Flag for
a future quick task or the next phase touching `feed_item.dart` /
`settings_screen.dart`.
