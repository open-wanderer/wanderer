# Phase 12: Settings Categories Screen - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-01
**Phase:** 12-settings-categories-screen
**Areas discussed:** Reordering interaction, Row layout & expansion, Save/error feedback, Empty/loading states, Scope amendment (subcategory screen + own-trail confirm dialog)

---

## Reordering interaction

| Option | Description | Selected |
|--------|-------------|----------|
| Long-press drag handle | ReorderableListView.builder with trailing drag-handle icon | ✓ |
| Up/down arrow buttons | Explicit per-row up/down IconButtons | |

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated drag handle icon | Only handle triggers reorder; rest of row taps normally | ✓ |
| Whole row draggable | Entire ListTile is the drag target | |

| Option | Description | Selected |
|--------|-------------|----------|
| Not reorderable, server order | Subcategories keep API order inside expanded category, no drag | |
| Alphabetical order instead | Sort subcategories alphabetically | |

**User's choice:** Long-press drag handle, dedicated handle icon for gesture separation. Subcategory-order question was superseded by the Row layout redesign below (subcategories became reorderable via a dedicated screen).
**Notes:** None.

---

## Row layout & expansion

| Option | Description | Selected |
|--------|-------------|----------|
| Always collapsed on load | ExpansionTile starts collapsed each session | |
| Persist expanded state | Remember expanded categories | |

**User's choice:** Neither — user redirected: "Instead of an expansion tile make the tile clickable. This navigates to a screen where the user can toggle and reorder (!) the subcategories."
**Notes:** This is a deviation from REQUIREMENTS.md's original SETCAT-08 wording (ExpansionTile) and from the Out-of-Scope table (subcategory reordering was listed as deferred to a future milestone). User confirmed "Amend scope now" when asked how to handle the conflict — REQUIREMENTS.md and ROADMAP.md were updated (SETCAT-08 redefined, SETCAT-09 clarified, SETCAT-10 added for subcategory reorder).

Follow-up questions on the new subcategory screen:

| Option | Description | Selected |
|--------|-------------|----------|
| Whole row tappable except switch/handle | Switch toggles, handle reorders, rest of row navigates | ✓ |
| Dedicated chevron/arrow to navigate | Separate chevron icon for navigation only | |

| Option | Description | Selected |
|--------|-------------|----------|
| Same list pattern as parent | New screen reuses ReorderableListView + SwitchListTile + drag-handle, AppBar title = category name | ✓ |
| Simpler flat list, no reorder styling | Plain SwitchListTile rows, no drag styling | |

**Notes:** None further.

---

## Save/error feedback

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, same toast pattern | Reuse toastProvider + error toast on failure, no success toast | ✓ |
| Add success toast too | Show success toast on every successful toggle | |

| Option | Description | Selected |
|--------|-------------|----------|
| Revert order + error toast | Revert list to pre-drag order on reorder failure, show error toast | ✓ |
| Keep new order, just toast error | Leave dragged order even on failure | |

**Notes:** None.

---

## Empty/loading states

| Option | Description | Selected |
|--------|-------------|----------|
| Centered CircularProgressIndicator | Standard spinner while loading | |
| Skeleton list placeholders | Shimmer/skeleton rows | |

**User's choice:** Neither — "Use the AsyncLoader component" (`app/lib/components/async_loader.dart`), an existing app-wide loading wrapper not offered as an option.

| Option | Description | Selected |
|--------|-------------|----------|
| Allowed silently | No warning when all categories are hidden | |
| Warn via toast/banner | Non-blocking warning if zero visible categories | |

**User's choice:** Neither exactly — "The web version shows a warning when toggling a category hides trails the user has created. This should be the same in the app." This introduced the own-trail disable-confirm feature (see below), which is distinct from the "zero visible categories" question (which defaulted to "allowed silently" per no further objection).

---

## Scope amendment: own-trail disable confirmation

Follow-up questions after discovering `web/src/routes/settings/categories/+page.svelte`'s `promptBeforeDisable`/`confirmDisable` flow:

| Option | Description | Selected |
|--------|-------------|----------|
| Own-trails warning only | Confirm dialog shows only trail-count warning, no plugin-mapping messaging | ✓ |
| Include plugin warnings too | Port full web logic including plugin mapping messages | |

| Option | Description | Selected |
|--------|-------------|----------|
| Fetch on toggle-off attempt | Query trail list API filtered by author+category on switch-off, no preload | ✓ |
| Preload counts for all categories | Fetch full trail list once on screen load, compute counts client-side | |

| Option | Description | Selected |
|--------|-------------|----------|
| Add as new SETCAT requirements | New SETCAT-10/11 requirement IDs in REQUIREMENTS.md | ✓ |
| Fold into existing SETCAT-07 | Reword SETCAT-07 only, no new IDs | |

**Notes:** REQUIREMENTS.md amended with SETCAT-11 (own-trail confirm dialog); "Subcategory reordering" removed from Out of Scope table and replaced with "Plugin-mapping warnings" as the new out-of-scope item.

---

## Claude's Discretion

- Exact confirm-dialog widget choice (AlertDialog vs a reusable app confirm component)
- Exact drag-handle icon sizing/spacing, row padding conventions

## Deferred Ideas

None — all scope changes discussed were folded into Phase 12's requirements (see CONTEXT.md `<deferred>`).
