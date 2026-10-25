---
status: awaiting_reporter
trigger: "#1411. Probably too little info to infer a full causation chain. Let me know what info I should ask for"
goal: find_and_fix
created: 2026-10-19
updated: 2026-10-19
---

## Symptoms

- source: GitHub #1411 (reporter rrole, also filed #1412)
- expected: The app stays signed in after it is closed or the screen is locked.
- actual: "The app logs out every time I exit or if I lock the screen. This makes recording a trail impossible."
- errors: none reported
- timeline: app 0.1.0 (Play open test), instance v0.21.0-app
- reproduction: sign in, exit or lock the screen, come back → signed out. Reporter says it happens every time.
- device: Pixel 6a, Android 17

## Current Focus

hypothesis: |
  "Every time" points to a deterministic cause rather than the transient one
  diagnosed in app-logout-on-close.md. Open candidates:
  H1 The process is killed on lock/exit and the persisted pb_auth cookie is
     missing or expired at cold start (Auth.build returns null without it).
  H2 The pb_auth cookie expires almost immediately: a short "Auth token
     duration" on the users collection, or device clock ahead of the server.
     Cookie expiry is the token's exp (exportToCookie, hooks.server.ts:197),
     and cookie_jar drops expired cookies on load.
  H3 The cookie never authenticates at all: a reverse proxy rewrites or strips
     Set-Cookie, so the session only looks signed in until the next launch.
  H4 _validateInBackground gets 401/403/404 on every cold start and calls the
     full logout() (auth_provider.dart:85-109, 133-139).
  H5 The still-unfixed hooks.server.ts:153-160 catch, which clears auth on any
     authRefresh error, fires on every request because PocketBase is slow or
     failing behind this instance.
next_action: |
  Round 3 on #1411: airplane-mode test (log in, go offline, swipe away,
  reopen) and whether the map flashes before the welcome screen. Splits
  "session missing on disk" from "startup check logs out". In parallel:
  reproduce on our own Android device through a cloudflared quick tunnel.

## Evidence

- timestamp: 2026-10-19
  checked: app/lib/main.dart:260-264, other didChangeAppLifecycleState handlers
  found: Resume only drains trail sync. Nothing re-checks auth on resume.
  implication: A sign-out on screen lock means the process was restarted (H1) or the logout happened earlier and only shows on return.

- timestamp: 2026-10-19
  checked: cookie_jar-4.0.9 default.dart _check (line 199), persist.dart (persistSession defaults to true)
  found: Secure cookies are returned for http URIs too, and session cookies are persisted.
  implication: A Secure-flag or session-cookie mismatch alone does not drop the cookie.

- timestamp: 2026-10-19
  checked: web/src/hooks.server.ts:152-160
  found: The catch still clears the auth store on any authRefresh error; the fix from app-logout-on-close.md was never applied.
  implication: H5 stays open.

- timestamp: 2026-10-19
  checked: Reporter answers on #1411
  found: |
    Logo animation shows (cold start; Android kills the process, also while
    recording). Account works in-session. Password login. Web stays logged in.
    Token duration 1209600s. App URL equals ORIGIN, behind a Cloudflare Tunnel.
    Clock automatic. Their curl login showed pb_auth with an empty token and a
    1970 expiry; status line not included. No web logs.
  implication: |
    H2 eliminated (14-day token, clock fine). H3 unlikely (account works
    in-session). The curl result matches a failed login (wrong password or
    shell quoting), because hooks.server.ts:197 exports an empty, expired
    pb_auth on every anonymous response. A successful authWithPassword exports
    the real token. Not conclusive without the status line.

- timestamp: 2026-10-19
  checked: app/lib (all Dio/CookieJar construction)
  found: One PersistCookieJar (main.dart:64) and one Dio with CookieManager (api_provider.dart:69-91). server_selection_provider's bare Dio has no jar.
  implication: No second writer that could overwrite the cookie file.

- timestamp: 2026-10-19
  checked: auth_provider.dart:85-139 against a Cloudflare Tunnel setup
  found: |
    Every cold start calls GET /api/v1/user/:id, and _isAuthError treats any
    401/403/404 as an invalid session, with no check that the response came
    from wanderer. A Cloudflare challenge or block (Bot Fight Mode, Browser
    Integrity Check, Security Level, WAF rule, Access) answers a non-browser
    client with a 403 HTML page, which triggers the full logout(). The browser
    passes the same challenges, and the web never logs out on a 403.
  implication: |
    Leading hypothesis (H6). Fits cold-start-only, web unaffected, and "after
    some time in the background" (IP change or new challenge).

- timestamp: 2026-10-20
  checked: Reporter round 2 on #1411
  found: |
    Cloudflare shows no security events at logout. No relevant CF settings.
    The recording was gone after the restart ("as if I have started the app
    for the first time"). Login via curl: HTTP/2 200 and a valid pb_auth,
    Expires 14 days out, Path=/, Secure, SameSite=Lax. Offline, the app opens
    on the welcome screen with no servers (unclear whether they logged in
    again before that test).
  implication: |
    H3 out (server issues a good cookie). H6 weakened (no CF events).
    Losing the recording fits logout() having run, since it purges
    account-scoped data. Still undecided: is the session gone from disk
    before startup, or does the startup check log the user out?

- timestamp: 2026-10-20
  checked: main.dart:52-67, cookie_jar persist.dart (_save, _load, saveFromResponse), default.dart _isPathMatch
  found: |
    ObjectBox and the cookie jar both live in getApplicationDocumentsDirectory.
    _save writes host cookies to disk even when the response also has a
    Domain cookie (e.g. Cloudflare __cf_bm). Path "/" matches every URL.
    _load deletes a host's cookie file if parsing fails.
    Nothing at startup deletes data except logout().
    The startup check (GET /user/:id with the same expands) is identical to
    the call login makes.
  implication: No setup-specific mechanism found in code. Need a test that splits disk loss from server-check logout.

## Eliminated

- hypothesis: H3 reverse proxy breaks the pb_auth cookie
  evidence: curl through the tunnel shows HTTP/2 200 and a valid pb_auth with a 14-day Expires.
  timestamp: 2026-10-20

- hypothesis: H2 short token duration or clock skew
  evidence: Token duration 1209600s (14 days), phone clock automatic.
  timestamp: 2026-10-19

## Resolution

root_cause:
fix:
verification:
files_changed: []
