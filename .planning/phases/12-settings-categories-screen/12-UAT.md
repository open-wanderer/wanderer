---
status: complete
phase: 12-settings-categories-screen
source: [12-VERIFICATION.md]
started: 2026-07-02T10:00:00Z
updated: 2026-07-02T12:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Sort, icon, name, drag-handle and switch render correctly on SettingsCategoriesScreen
expected: Categories listed sorted by priority ascending (alphabetical tie-break), each row shows a drag handle, category icon, locale-resolved name, and a visibility switch
result: pass
note: Row layout iterated during live testing — drag handle icon replaced with long-press-to-drag on the whole row; sort/icon/name/switch confirmed working.

### 2. Drag a category row by the handle to reorder; release, leave and re-enter the screen
expected: New order persists after reload; simulating a network failure (e.g. airplane mode) during a drag reverts the list to the prior order and shows an error toast
result: pass
note: Confirmed after fixing two live-testing regressions: non-optimistic snap-back (guard cleared before refetch settled) and a related pre-reorder flash.

### 3. Toggle a category OFF that has the signed-in user's own trails
expected: A confirm dialog appears with the trail count, 'View trails', 'Disable anyway', and 'Cancel' actions. Tapping 'View trails' opens the user's own profile trail list filtered to only that category, without saving or changing the switch. 'Cancel' reverts the switch to ON. 'Disable anyway' saves the change. Toggling ON never shows this dialog.
result: pass

### 4. Tap a category row body (not the switch or drag handle)
expected: Navigates to SettingsSubcategoriesScreen for that category, showing the category's localized name as the AppBar title
result: pass

### 5. Open a category with zero subcategories from SettingsCategoriesScreen
expected: SettingsSubcategoriesScreen shows the empty-state copy ('No subcategories' / body text) and the screen remains reachable (no crash, no blank hang)
result: pass

### 6. Drag a subcategory row by the handle to reorder within SettingsSubcategoriesScreen; release, leave and re-enter
expected: New order persists, scoped to that parent category only; simulated network failure reverts the list with an error toast
result: pass
note: Same optimistic-guard fix applied here as category reorder (test 2). Reordering deliberately stays enabled even when the parent category is disabled — confirmed during live testing.

### 7. Toggle a subcategory OFF that has the signed-in user's own trails
expected: A dialog titled 'Hide this subcategory?' appears with the count, 'View trails', 'Disable anyway', 'Cancel'. 'View trails' opens the user's own profile trail list filtered to only that subcategory without saving. Cancel reverts to ON; Disable anyway saves. ON never triggers this dialog.
result: pass

### 8. Full end-to-end reachability: Settings tab -> Categories tile -> SettingsCategoriesScreen -> tap category row -> SettingsSubcategoriesScreen
expected: The whole flow is navigable on a real device/simulator without crashes, including the CR-01 deep-link/restart fallback (state.extra as Category guard) not being hit under normal in-app navigation
result: pass

## Summary

total: 8
passed: 8
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

None. Live testing this session additionally surfaced and fixed several issues not covered by the original test list, plus shipped follow-on UI polish beyond the original phase scope:

- Non-optimistic reorder snap-back (category + subcategory screens) — fixed.
- White flash on toggle/reorder from AsyncLoader falling back to empty mockData mid-refresh — fixed.
- Pre-reorder flash from clearing the optimistic guard before the post-reorder refetch settled — fixed.
- Category visibility cascade into the subcategory screen (dims/disables subcategory switches when parent category is off; reordering stays available) — added.
- Read-only subcategory chips under each category row (badge icon + name, dimmed by the subcategory's own visibility) — added.
- Row interaction redesigned: explicit drag-handle icon removed in favor of long-press-anywhere-on-row to reorder (both screens); tap navigates to subcategories (category screen only).
- Explanatory header text added to SettingsCategoriesScreen describing the screen's purpose and the tap vs. long-press gestures.

A web-side caching bug (stale `GET /api/v1/user-category-preference` response on a normal, non-hard page reload) was found and confirmed via cache-bypass but root cause not yet isolated — explicitly deferred to a separate main-branch fix, not part of this milestone's scope.
</content>
