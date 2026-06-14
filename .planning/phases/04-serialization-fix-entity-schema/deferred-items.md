# Deferred Items — Phase 04

Out-of-scope discoveries logged during execution. NOT fixed (per scope boundary rule — only auto-fix issues directly caused by the current task's changes).

## Plan 04-02

### Pre-existing test failures in `app/test/models/feed_item_test.dart`

- **Discovered during:** Task 2 (full `flutter test` run)
- **Failures (2):**
  - `FeedItem.fromJson type "trail" returns FeedItemTrail with TrailSearchResult` — `type 'int' is not a subtype of type 'String' in type cast` at `package:wanderer/models/trail.g.dart 55:67` (`_$TrailFromJson`)
  - `FeedItem.fromJson type "list" returns FeedItemList with ListSearchResult` — `type 'int' is not a subtype of type 'List<dynamic>?' in type cast` at `package:wanderer/models/list.g.dart 34:27` (`_$WandererListFromJson`)
- **Why out of scope:** These failures originate in `trail.g.dart` / `list.g.dart` / `feed_item.dart`, none of which were modified by plan 04-02. The 04-02 change is entity-only (`TrailEntity.navCacheJson`) and does not touch the `Trail`/`WandererList` JSON models or their codegen. Confirmed unchanged: `git status` / `git diff` report no modifications to these files. The failures are test-fixture type mismatches (test JSON supplies `int` where the model expects `String`/`List`) predating Phase 4.
- **Action taken:** None. Logged for a future fix outside the Phase 4 serialization/entity scope.
