# Phase 31: Flutter Settings Hierarchy - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-27
**Phase:** 31-flutter-settings-hierarchy
**Areas discussed:** Tree rendering approach, Group node data handling, Search/filter in the new hierarchy, Sort order

---

## Tree rendering approach

| Option | Description | Selected |
|--------|-------------|----------|
| Port the admin's flatten algorithm | Build tree once, compute flat depth-tagged visible-row array (regions_ui.html approach), render via ListView.builder with indentation | ✓ |
| Native ExpansionTile per group | Recursive Flutter ExpansionTile widgets; built-in animation but awkward virtualization/scroll-position, diverges from admin pattern | |

**User's choice:** Port the admin's flatten algorithm (recommended).
**Notes:** Leaf row UI stays exactly as today, just indented.

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror admin: auto-expand branches with downloaded content | Branch auto-expands if it contains an already-downloaded or mid-download leaf; else collapsed | ✓ |
| All collapsed by default | Every group starts collapsed regardless of state | |
| Top level expanded, deeper levels collapsed | Fixed rule ignoring download state | |

**User's choice:** Mirror admin (recommended).
**Notes:** Matches admin Phase 30's D-07.

---

## Group node data handling

| Option | Description | Selected |
|--------|-------------|----------|
| Ephemeral client-side tree, not persisted | Groups rebuilt from API response each fetch, used only for rendering/nesting; no ObjectBox entity | ✓ |
| Persist groups as ObjectBox entities | New GroupEntity or extended RegionEntity for offline availability | |

**User's choice:** Ephemeral client-side tree (recommended).
**Notes:** Groups own no downloadable content, disk usage, or version.

| Option | Description | Selected |
|--------|-------------|----------|
| Flat list fallback when offline | Render today's flat ListView from ObjectBox when no fresh fetch/cache available, preserving full offline management | |
| Empty state when offline, accept the regression | Deliberately drop offline region management as a scoped exception to APPUI-02 | ✓ |

**User's choice:** Empty state when offline, accept the regression.
**Notes:** User explicitly confirmed accepting this as a deliberate, scoped trade-off after Claude flagged the conflict with APPUI-02's "no download-UX regression" requirement. Captured in CONTEXT.md as D-04 with an explicit call-out for downstream agents.

---

## Search/filter in the new hierarchy

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror admin: match + ancestors, auto-expanded | Name match pulls in whole ancestor chain, auto-expanded, same computeFilterMatches algorithm | ✓ |
| Flatten to matches only while filtering | Temporarily switch to flat list of matches only, matching today's exact flat-filter behavior | |

**User's choice:** Mirror admin (recommended).
**Notes:** None.

---

## Sort order

| Option | Description | Selected |
|--------|-------------|----------|
| Add sort_order to the API response | Small backend touch to regions_get.go to surface the existing DB column; matches admin's sort order | ✓ |
| Sort alphabetically client-side | Flutter-only, no backend change, but diverges from admin's CoMaps-canonical order | |

**User's choice:** Add sort_order to the API response (recommended).
**Notes:** Flagged as a small, surgical backend change that doesn't otherwise expand phase scope.

---

## Claude's Discretion

- Exact debounce timing on the filter input, if any.
- Visual treatment of the offline empty state (icon, copy, whether it explains why).
- Whether depth-indentation uses fixed padding per level or other visual grouping.

## Deferred Ideas

None — discussion stayed within phase scope.
