# Phase 11: Trail Filter Subcategory Support - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-29
**Phase:** 11-trail-filter-subcategory-support
**Areas discussed:** Subcategory chips in FilterScreen, Quick filter bar redesign

---

## Subcategory chips in FilterScreen

### Where should subcategory chips live?

| Option | Description | Selected |
|--------|-------------|----------|
| New titled section below categories | A 'Subcategories' heading appears below the Category section, conditionally rendered when categories are selected. Matches the existing labelled-block layout pattern. | ✓ |
| Inline below category chips, no section header | More compact but blurs visual separation. | |
| You decide | Claude picks most idiomatic Flutter approach. | |

**User's choice:** New titled section below categories
**Notes:** None

---

### Should the section animate in/out?

| Option | Description | Selected |
|--------|-------------|----------|
| Animate with AnimatedSize | Smooth expand/collapse, standard Flutter widget, zero extra deps. | ✓ |
| Instant show/hide | Simple conditional rendering. Simpler code. | |

**User's choice:** AnimatedSize

---

### Chip component for subcategory chips?

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse WandererFilterChip<Subcategory> | Same component, same styling. | ✓ (with icon requirement) |
| Smaller or different style | New chip variant. | |

**User's choice:** Reuse WandererFilterChip — but **mandatory to show icons** (FA icon and badge). User explicitly referenced `category_util.ts` for how icons and badges are rendered on the web.

---

### Icon rendering approach?

| Option | Description | Selected |
|--------|-------------|----------|
| Extend WandererFilterChip with optional avatarBuilder | Add `Widget? Function(T item)? avatarBuilder`. Keeps component reusable. | ✓ |
| Custom CategoryChip / SubcategoryChip widget | Baked-in icon rendering. More code to maintain. | |

**User's choice:** Extend WandererFilterChip with avatarBuilder

---

### Badge icon on subcategory chips?

| Option | Description | Selected |
|--------|-------------|----------|
| Primary icon only | Just the main FA icon. Simpler chip. | |
| Primary icon + badge overlay | Badge_icon as small Stack overlay (bottom-right). Matches web. | ✓ |

**User's choice:** Primary icon + badge overlay (Stack: 16px primary / 10px badge, Alignment.bottomRight)

---

## Quick filter bar redesign

### Single sheet or two chips in the bar?

| Option | Description | Selected |
|--------|-------------|----------|
| Extend single sheet — Subcategories section below | Keep one Category chip; sheet gains subcategory section. Same pattern as FilterScreen. | ✓ |
| Two chips — Category + Subcategory | Separate chips in bar, each with own sheet. | |

**User's choice:** Extend single sheet

---

### Category chip active state with subcategories?

| Option | Description | Selected |
|--------|-------------|----------|
| Activate on category OR subcategory selected | Chip activates when either filter type is active. Clear signal. | ✓ |
| Only activate on category selection | Subcategory selection alone gives no indicator. | |

**User's choice:** Activate on category OR subcategory

---

### Bottom sheet initial size?

| Option | Description | Selected |
|--------|-------------|----------|
| Keep 0.5 — user can drag up | Existing scroll handles overflow. | ✓ |
| Increase to 0.7 when subcategory section visible | Shows more upfront. More state complexity. | |

**User's choice:** Keep 0.5

---

### Animate subcategory section inside the bottom sheet?

| Option | Description | Selected |
|--------|-------------|----------|
| Same AnimatedSize pattern | Consistent with FilterScreen. | ✓ |
| Instant show/hide in the sheet | Simpler. Sheet already animates in. | |

**User's choice:** Same AnimatedSize pattern

---

## Claude's Discretion

None — all decisions explicitly made by user.

## Deferred Ideas

None — discussion stayed within phase scope.
