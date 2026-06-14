# Deferred Items — Phase 03 Stats Sheet

Out-of-scope discoveries logged during execution. NOT fixed (scope boundary).

## Pre-existing test failures (test/models/feed_item_test.dart)

- **Found during:** Wave 1 (03-01) and re-confirmed Wave 2 (03-02) full `flutter test` runs
- **Failures:** 2 tests in `test/models/feed_item_test.dart`
  - `FeedItem.fromJson type "trail"` → `type 'int' is not a subtype of type 'String' in type cast` in `trail.g.dart:55`
  - `FeedItem.fromJson type "list"` → `type 'int' is not a subtype of type 'List<dynamic>?' in type cast` in `list.g.dart:34`
- **Origin:** `trail.g.dart` / `list.g.dart` generated deserialization — files NOT touched by this phase
- **Status:** Deferred. Unrelated to navigation/stats work. Phase 03 source + test files all pass.
