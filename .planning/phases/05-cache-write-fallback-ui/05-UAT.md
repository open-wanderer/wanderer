---
status: testing
phase: 05-cache-write-fallback-ui
source: [05-VERIFICATION.md]
started: 2026-06-14T17:30:00Z
---

# UAT — Phase 05: Cache-Write Fallback UI

All 4 automated requirement checks passed (OFFLINE-01 through OFFLINE-04). Three scenarios require physical device testing.

## Items

### UAT-1: Offline navigation from cache

**Scenario:** Download a trail while online, disable network, tap Navigate.

**Expected:** Navigation screen opens with maneuver instructions and a wifi_off icon visible in the maneuver banner row. Navigation proceeds without error.

**Status:** pending

---

### UAT-2: Online navigation — no offline icon, silent re-cache

**Scenario:** Navigate a downloaded trail while online.

**Expected:** Navigation screen opens normally with NO wifi_off icon. No visible cache activity. After navigation completes, `navCacheJson` on the entity should be updated (best-effort, silent).

**Status:** pending

---

### UAT-3: No-cache error toast path

**Scenario:** Navigate while offline on a trail that was NOT previously downloaded (no `navCacheJson` in ObjectBox).

**Expected:** An error toast appears. No crash. No navigation screen push.

**Status:** pending

## Also Recommended (from Code Review)

Before final sign-off, consider running `/gsd-code-review 05 --fix` to address:
- **CR-02** `catch (_)` swallows `cancelToken` cancellation in `trail_download_service.dart`
- **CR-03** Force-unwrap `!` on `getFileUrl()` crash on empty photo filename
- **WR-01** `_costingFor` costing logic duplicated across two files
