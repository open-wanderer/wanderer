---
phase: 24-settings-offline-maps-regions-ui
reviewed: 2026-07-22T00:00:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - app/lib/util/disk_space_util.dart
  - app/test/util/disk_space_util_test.dart
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 24: Code Review Report

**Reviewed:** 2026-07-22T00:00:00Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

This is a gap-closure fix for SETUI-03/04: `disk_space_util.dart` was refactored to
extract an injectable `resolveFreeDiskSpaceBytes` orchestrator that falls back to a
device-wide free-space query when the path-specific query throws (the documented
`disk_space_2` behavior for a not-yet-created region directory), while continuing to
fail closed to `null` when both queries fail. `freeDiskSpaceBytes` now delegates to
this orchestrator, and five new unit tests exercise the fallback ordering with fake
injected closures.

I traced the fix against the actual `disk_space_2@1.0.12` source (Dart wrapper +
Android Kotlin + iOS Swift, read directly from the pub cache) and against the real
caller (`app/lib/services/tile_repository_manager.dart:116` /
`:201`, which passes `regionStorageDir(root, id)` — a directory that legitimately
doesn't exist yet on a region's first download). For that real-world scenario the fix
is correct: the Dart-side `getFreeDiskSpaceForPath` throws before ever reaching the
platform channel when `Directory(path).existsSync()` is false, so the new catch/retry
path is reliably exercised and the device-wide number is used, restoring
`hasEnoughSpace`'s ability to pass on a first-ever download.

`dart analyze` and the full `resolveFreeDiskSpaceBytes`/`hasEnoughSpace` test group
both pass clean. No critical/blocker issues found. Two warnings on
robustness/observability of the new orchestrator, plus three minor quality/test-
completeness notes.

## Warnings

### WR-01: Device-wide fallback is skipped when the path query resolves to `null` instead of throwing

**File:** `app/lib/util/disk_space_util.dart:60-79`
**Issue:** `resolveFreeDiskSpaceBytes` only falls back to `deviceQuery` when `pathQuery`
*throws* (the `catch` block at line 65). If `pathQuery` completes without throwing but
resolves to `null` — which `PathSpaceQuery`'s own signature (`Future<double?> Function(String path)`)
explicitly allows, and which the file's own docs for `freeDiskSpaceBytes` describe as a
possible failure mode ("a `null` result is swallowed and reported as `null`") — the
function returns `null` immediately at line 79 without ever trying `deviceQuery`. This
is exactly the behavior the new `resolveFreeDiskSpaceBytes` test at
`disk_space_util_test.dart:163-170` locks in, but it is inconsistent with the
orchestrator's own doc comment ("falling back to `deviceQuery` when the path-specific
query throws (or when `forPath` is omitted...)") — a null-without-throw failure of the
path query is silently treated differently from a throwing failure, even though both
represent "the path-specific query failed."

In the currently-pinned `disk_space_2@1.0.12`, this is low-probability in practice:
the Android native side always calls `result.error(...)` (which surfaces as a thrown
`PlatformException` in Dart) on failure rather than returning null, and the Dart
wrapper's own `existsSync()` guard throws before the platform channel is ever invoked
for a missing directory. But the gap is real for any other quiet-null failure mode
(a future plugin version, a different platform, or a channel edge case that returns
null instead of throwing), and it silently defeats the very fallback this fix was
written to add.

**Fix:** Treat a `null` path-query result the same as a thrown exception for fallback
purposes, or explicitly document why it's intentionally excluded:
```dart
double? freeMebibytes;
try {
  freeMebibytes = forPath != null
      ? await pathQuery(forPath)
      : await deviceQuery();
  if (forPath != null && freeMebibytes == null) {
    // Treat a quiet null as a failure too, same as a thrown exception.
    freeMebibytes = await deviceQuery();
  }
} catch (e) {
  ...
}
```

### WR-02: Swallowed query failures are not logged anywhere

**File:** `app/lib/util/disk_space_util.dart:65,73`
**Issue:** Both `catch` blocks silently discard the caught exception (the `e` variable
is bound but never read, at line 65 and line 73) with no `debugPrint`/log call. Since
this function is the sole gate for TILE-03's disk-space check, a real production
failure here (e.g. `disk_space_2` throwing consistently on a device/OS combination, a
permissions issue, or the device query also failing) becomes an invisible download
refusal — the user sees the vector/DEM package silently flip to `error` status with no
diagnostic trail anywhere. This is inconsistent with the project's own convention for
swallowed errors elsewhere in `app/lib/services/` — e.g.
`trail_download_service.dart:480,483` logs via `debugPrint('Failed to download photo
$url: $e')` in an analogous swallow-and-continue path.
**Fix:**
```dart
} catch (e) {
  if (forPath == null) {
    debugPrint('freeDiskSpaceBytes: device query failed: $e');
    return null;
  }
  try {
    freeMebibytes = await deviceQuery();
  } catch (e) {
    debugPrint('freeDiskSpaceBytes: path and device queries both failed: $e');
    return null;
  }
}
```

## Info

### IN-01: Unused, shadowed `catch (e)` clause names

**File:** `app/lib/util/disk_space_util.dart:65,73`
**Issue:** Both catch clauses bind `e` but never reference it (this was previously
`catch (_)` before the refactor — confirmed via `git diff` against the pre-fix
version). The inner `catch (e)` at line 73 also shadows the outer `e` from line 65
within its scope, which is confusing to read even though harmless today (`dart
analyze` does not flag it). If WR-02 isn't adopted, rename both to `catch (_)` to make
the "intentionally discarded" intent explicit and remove the shadowing.
**Fix:** `catch (_) { ... }` for both blocks (or keep `e` and use it per WR-02).

### IN-02: New test additions are not `dart format`-clean

**File:** `app/test/util/disk_space_util_test.dart`
**Issue:** Running `dart format --output=none --set-exit-if-changed` on this file
reports it as changed — several of the new `resolveFreeDiskSpaceBytes` test blocks
(e.g. lines 86-99, 104-116, 118-130, 132-148) use manual line-wrapping that differs
from what `dart format` produces (it collapses/expands the `test(...)` call
differently once line length allows), and it also reformats the pre-existing
`hasEnoughSpace` null-check test at lines 39-48. No CI step currently enforces `dart
format` in this repo, so this won't break a pipeline, but it's a drift from the rest
of the (correctly formatted) file and from `disk_space_util.dart` itself.
**Fix:** Run `dart format app/test/util/disk_space_util_test.dart` before merging.

### IN-03: Null-return fallback test doesn't assert the device query was actually skipped

**File:** `app/test/util/disk_space_util_test.dart:163-170`
**Issue:** The `'query returns null mebibytes -> returns null bytes'` test passes a
`deviceQuery` stub that would return `400.0` but never asserts it was *not* called
(compare with the two other tests in this group, at lines 90-100 and 136-146, which
both track a call counter to prove the untaken branch really is untaken). As currently
written, the test would still pass if a future change accidentally started calling
`deviceQuery` in the null-path-query case and its return value happened to be
discarded — it only pins the final return value, not the call-skipping behavior WR-01
flags as questionable in the first place.
**Fix:**
```dart
test('query returns null mebibytes -> returns null bytes', () async {
  var deviceCallCount = 0;
  final result = await resolveFreeDiskSpaceBytes(
    forPath: '/app/regions/munich',
    pathQuery: (path) async => null,
    deviceQuery: () async {
      deviceCallCount++;
      return 400.0;
    },
  );
  expect(result, isNull);
  expect(deviceCallCount, 0); // documents current (see WR-01) behavior
});
```

---

_Reviewed: 2026-07-22T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
