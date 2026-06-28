# Phase 6: Admin Browser UI - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-27
**Phase:** 6-Admin Browser UI
**Areas discussed:** Auth flow, JS approach, Discovery UX flow, Peer list actions

---

## Auth Flow

| Option | Description | Selected |
|--------|-------------|----------|
| Login form on the page | Page shows email+password form, calls /api/admins/auth-with-password, stores JWT in sessionStorage | |
| Paste token manually | Admin copies JWT from PocketBase admin panel and pastes it into a field | |
| Piggyback on PB admin session | Page reads `__pb_superusers__/_` from localStorage (same key PocketBase admin panel writes), redirects to `/_/` if missing/expired | ✓ |
| Basic HTTP auth | Go handler exchanges HTTP Basic credentials for JWT internally | |

**User's choice:** Redirect to PocketBase admin panel (`/_/`) if unauthenticated; read the existing superuser JWT from localStorage on return. No separate login form on the page.

**Notes:** Research revealed PocketBase admin panel (`/_/`) stores its JWT via `new LocalAuthStore("__pb_superusers__" + currentPath)` in `ui/src/pb.js:13`. For a root-hosted instance, the localStorage key is `__pb_superusers__/_`. The `/federation/` page reads this key, uses the token for all API calls, and redirects to `/_/` if missing or expired. Token storage: sessionStorage question was superseded by the piggyback approach — the page never writes the token.

---

## JS Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Alpine.js + Tailwind CDN | Declarative reactivity via CDN, Tailwind Play CDN for styling, zero build step, one HTML file | ✓ |
| Vanilla JS | No CDN dependency, manual DOM updates | |
| HTMX | Server-side HTML fragments, adds Go template complexity | |
| Svelte + Tailwind with build step | Separate db/federation-ui/ npm project, compiled assets embedded via //go:embed | (considered) |

**User's choice:** Alpine.js + Tailwind CDN. User initially asked whether Svelte+Tailwind could be used to match Wanderer's look — confirmed that Alpine + Tailwind CDN uses the exact same Tailwind classes with zero build step, resolving the concern.

**Notes:** Theme follows `pbColorScheme` localStorage key (`"light"`/`"dark"`) set by the PocketBase admin panel; falls back to `prefers-color-scheme` if key is absent. Dark class applied to `<html>` for Tailwind `dark:` variant activation.

---

## Discovery UX Flow

| Option | Description | Selected |
|--------|-------------|----------|
| Two-step: preview → Connect button | Discover shows preview card, admin explicitly clicks Connect | ✓ |
| One-step: auto-connect on success | If discover succeeds, immediately POST /follow without confirmation | |

**User's choice:** Two-step. Admin reviews the preview card (domain, version, user/trail counts) before clicking Connect.

**Notes:** After Connect, the peer list re-fetches automatically. The discovery form resets. The new peer appears as Outbound/Pending in the table immediately. Errors are shown inline below the URL input.

---

## Peer List Actions

| Option | Description | Selected |
|--------|-------------|----------|
| Inline buttons per row | Actions visible immediately per row, context-sensitive | ✓ |
| Row click opens side panel or modal | Extra click needed, cleaner table | |

**Confirmation for destructive actions:**

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm Disconnect only | browser confirm() before Disconnect; Reject fires immediately | ✓ |
| No confirmation | All buttons fire immediately | |
| Confirm both Disconnect and Reject | Extra friction on both | |

**User's choice:** Inline buttons, confirm only Disconnect.

**Notes:** Approve and Reject are for inbound/pending rows only. Disconnect shows for any accepted peer. After any action, peer list re-fetches.

---

## Claude's Discretion

- Page title and heading copy
- Whether rejected peers are shown in the table or filtered out
- Loading/spinner states during API calls
- Exact Tailwind class choices for button variants
- Whether the discovery preview card is dismissible (cancel button) or cleared only on Connect/error

## Deferred Ideas

- Realtime peer list via PocketBase subscription → v2 UX-01
- Manual "Refresh" button for peer list → v2 (auto-refresh on actions is sufficient for v1.1)
- Refresh peer metadata button → v2 UX-02
- Connection activity log → v2 UX-03
