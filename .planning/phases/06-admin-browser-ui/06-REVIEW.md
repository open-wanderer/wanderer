---
phase: 06-admin-browser-ui
reviewed: 2026-06-27T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - db/routes/federation_ui.html
  - db/routes/federation_ui.go
  - db/main.go
findings:
  critical: 1
  warning: 4
  info: 2
  total: 7
status: issues_found
---

# Phase 6: Code Review Report

**Reviewed:** 2026-06-27
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

## Summary

Reviewed the federation admin SPA (`federation_ui.html`), its Go handler (`federation_ui.go`), and the single new route registration in `db/main.go`. The Go handler is minimal and correct. The route registration is correct. The SPA has one critical security gap (missing Content-Security-Policy on an admin page that handles a superuser JWT) and several logic bugs — the most impactful being that the `loading` flag is never set by any action function, making all action button disabling inoperative and permitting duplicate submissions. Error feedback for the `connect()` action is silently swallowed.

---

## Critical Issues

### CR-01: Missing Content-Security-Policy on admin page that handles a superuser JWT

**File:** `db/routes/federation_ui.go:23-27`

**Issue:** `FederationDashboard` sets `X-Frame-Options: DENY` (good) but does not set a `Content-Security-Policy` header. The page loads third-party scripts from `cdn.tailwindcss.com` and `cdn.jsdelivr.net` via plain CDN URLs with no subresource integrity (SRI) hashes. If either CDN is compromised or the URLs are intercepted (e.g., via MITM on an HTTP deployment), injected script runs in the context of a page that reads `localStorage.__pb_superusers__/_` — the admin JWT — and makes privileged API calls. A strict CSP (`script-src 'sha256-...' 'sha256-...'`) combined with SRI attributes would eliminate this attack surface.

**Fix:**
```go
// In FederationDashboard, add before WriteHeader:
e.Response.Header().Set("Content-Security-Policy",
    "default-src 'none'; "+
    "script-src 'sha256-<TAILWIND_HASH>' 'sha256-<ALPINE_HASH>'; "+
    "style-src 'unsafe-inline'; "+
    "connect-src 'self'; "+
    "frame-ancestors 'none'")
e.Response.Header().Set("X-Content-Type-Options", "nosniff")
```

And add `integrity` + `crossorigin` attributes to the CDN `<script>` tags in `federation_ui.html`:
```html
<script src="https://cdn.jsdelivr.net/npm/alpinejs@3.14.8/dist/cdn.min.js"
        integrity="sha256-<HASH>"
        crossorigin="anonymous"
        defer></script>
```

For the Tailwind Play CDN, the preferred fix is to vendor the CSS at build time rather than using the Play CDN in production, which Tailwind themselves discourage for production use.

---

## Warnings

### WR-01: `loading` flag is never set by action functions — button disabling is inoperative

**File:** `db/routes/federation_ui.html:119-177` (functions `connect`, `approve`, `reject`, `disconnect`) and lines 236, 295, 303, 316

**Issue:** The `loading` boolean is set only inside `loadPeers()` (lines 72/77). Every action button in the UI uses `:disabled="loading"` and `x-show="loading"` to show a spinner, but because `connect()`, `approve()`, `reject()`, and `disconnect()` never toggle `this.loading`, these guards are permanently `false` during the in-flight requests. A user who double-clicks any action button will dispatch duplicate HTTP requests. For `connect()`, this sends two Follow activities to the remote instance; for `approve()`/`reject()`/`disconnect()`, it races two state-modifying POSTs against the same record.

**Fix:** Each action function needs its own loading flag (or the global `loading` flag must be set/unset in a `try/finally` block):

```javascript
async connect() {
  this.loading = true;
  try {
    var res = await this.apiFetch('/federation/follow', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ actor_id: this.preview.actor_id })
    });
    if (!res) return;
    if (res.ok) {
      this.discoverUrl = '';
      this.preview = null;
      await this.loadPeers();
    }
  } finally {
    this.loading = false;
  }
},
```

Apply the same `try/finally` pattern to `approve`, `reject`, and `disconnect`.

---

### WR-02: `connect()` silently swallows non-ok responses — user receives no feedback on failure

**File:** `db/routes/federation_ui.html:119-131`

**Issue:** When `/federation/follow` returns a non-2xx status, `connect()` checks `res.ok` (line 126) and does nothing on the `else` branch — no error is stored, no message is shown. The user clicks Connect, the request fails (e.g., the remote instance rejects the Follow, or the local server returns 409/500), and the UI silently returns to the preview card with no indication of what happened. This is a user-experience regression from the `discover()` function which correctly populates `this.discoverError`.

**Fix:**
```javascript
async connect() {
  // ...
  if (res.ok) {
    this.discoverUrl = '';
    this.preview = null;
    await this.loadPeers();
  } else {
    var data = await res.json();
    this.discoverError = data.error || 'Failed to connect. Try again.';
  }
},
```

---

### WR-03: `rowErrors` are never cleared — stale error messages persist after retry or success

**File:** `db/routes/federation_ui.html:145-147, 158-160, 173-175`

**Issue:** `rowErrors` is populated on failure for a given `peer.follow_id`. On a subsequent successful action (or after `loadPeers()` refreshes the list), the entry in `rowErrors` is never removed. This means:

1. If the user retries and succeeds, the old error message still appears in the row.
2. After `loadPeers()` returns a refreshed peer list, the error message persists because `rowErrors` keys survive across list refreshes.

A particular edge case: after `approve()` succeeds and `loadPeers()` is called, the peer's `follow_id` may still match a key in `rowErrors` from a previous failed attempt, permanently displaying an outdated error on an otherwise healthy row.

**Fix:** Clear `rowErrors[peer.follow_id]` on success and reset entirely in `loadPeers()`:
```javascript
async loadPeers() {
  this.loading = true;
  this.rowErrors = {};   // clear stale errors on full refresh
  try {
    var res = await this.apiFetch('/federation/peers');
    if (res) this.peers = await res.json();
  } finally {
    this.loading = false;
  }
},
```

And within each action's success branch:
```javascript
if (res.ok) {
  // Remove the error for this row before reloading
  var errs = Object.assign({}, this.rowErrors);
  delete errs[peer.follow_id];
  this.rowErrors = errs;
  await this.loadPeers();
}
```

---

### WR-04: `tailwind.config` script sets a property on an undefined global — runtime error on strict environments

**File:** `db/routes/federation_ui.html:20`

**Issue:** Line 20 executes `tailwind.config = { darkMode: 'class' }` synchronously before the Tailwind Play CDN script has loaded (the CDN script is at line 23). At the time line 20 executes, `window.tailwind` does not exist, so `tailwind.config = ...` throws `ReferenceError: tailwind is not defined` in strict-mode environments or in browsers that enforce temporal dead zone semantics for undeclared identifiers (i.e., when `tailwind` has never been declared via `var`/`let`/`const`).

In practice, most browsers treat assignment to an undeclared identifier in non-strict mode as creating a global variable (`window.tailwind = { config: {...} }`), which Tailwind Play CDN then reads after load. However, this is relying on sloppy-mode implicit global creation — it will fail if the page is ever wrapped in a strict-mode module context, and it is the reason Tailwind's own documentation uses `window.tailwind = window.tailwind || {}; tailwind.config = {...}` as the recommended pattern.

**Fix:**
```html
<script>
  window.tailwind = { config: { darkMode: 'class' } };
</script>
```

---

## Info

### IN-01: Tailwind Play CDN is not recommended for production use

**File:** `db/routes/federation_ui.html:23`

**Issue:** The comment at line 23 references the Tailwind Play CDN as "D-07" (a deliberate design decision). Tailwind's own documentation explicitly states: "Do not use the Play CDN in production." The Play CDN generates CSS at runtime via JavaScript, increases page weight significantly, and introduces an external dependency at request time. For an admin page that is embedded at compile time via `//go:embed`, the correct approach is to generate the CSS at build time and embed the result.

**Fix:** Run `tailwindcss --input input.css --output federation_ui.css --content federation_ui.html` in a build step and embed the output stylesheet as a `<link>` tag or inline `<style>` block. Remove the CDN `<script>` tag entirely.

---

### IN-02: `X-Content-Type-Options: nosniff` header is absent

**File:** `db/routes/federation_ui.go:23-27`

**Issue:** The response sets `Content-Type: text/html; charset=utf-8` but does not set `X-Content-Type-Options: nosniff`. While the content type is explicit and correct, the defense-in-depth header is standard practice for all server-rendered HTML pages and is trivial to add.

**Fix:**
```go
e.Response.Header().Set("X-Content-Type-Options", "nosniff")
```

(This is also noted in CR-01 Fix — address both in one change.)

---

_Reviewed: 2026-06-27_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
