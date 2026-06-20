# Phase 9: Notifications - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-20
**Phase:** 9-notifications
**Areas discussed:** Toggle row layout

---

## Toggle Row Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Table-style with column headers | Matches web exactly. Sticky header row with "Web" / "Email" columns above all rows. Each row: label (expanded) + Web Switch + Email Switch. | |
| ListTile with inline labeled switches | Each notification is a ListTile with trailing Row containing [Text("Web") + Switch] + [Text("Email") + Switch]. No column headers. | |
| Two rows per notification type | Each notification type gets a section header (its description) followed by two SwitchListTile rows — one for web, one for email. | ✓ |

**User's choice:** Two rows per notification type
**Notes:** No further clarification provided. All other decisions (auto-save, default true, data model, ordering) were pre-determined by existing code and prior phases.

---

## Claude's Discretion

- Notification type display order: follow web client order (trail_comment, new_follower, trail_share, trail_like, list_share, summit_log_create, trail_mention, comment_mention, summit_log_mention)
- Missing `"web"` ARB key: add `"web": "Web"` to app_en.arb and regenerate
- Default values: `?? true` for both web and email (matching web client)

## Deferred Ideas

None.
