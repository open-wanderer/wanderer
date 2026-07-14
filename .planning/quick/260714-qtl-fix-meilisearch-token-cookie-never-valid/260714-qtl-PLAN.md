---
phase: quick-260714-qtl
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/hooks.server.ts
autonomous: true
requirements: [QUICK-260714-qtl]
---

<objective>
Fix the root cause of "all search API calls return 500" bugs: the `meilisearch_token` cookie is minted with a 24h TTL matching its JWT expiry, but the server never actually checks that expiry before reusing the cookie — it only re-mints on a missing cookie or an owner/version mismatch ([hooks.server.ts:88-97](web/src/hooks.server.ts#L88-L97)). If a stale token ever survives past its real expiry (clock skew, long-suspended mobile session, replayed cookie), the server keeps handing it to Meilisearch indefinitely, and every search call fails with a raw 500 (`MeilisearchApiError: Tenant token expired...`). Full rationale in /Users/christianbeutel/.claude/plans/sometimes-when-i-open-tidy-reef.md ("Part 2").

Make the server the authority on token freshness instead of trusting the cookie's client-enforced `Max-Age` alone.
</objective>

<context>
@web/src/hooks.server.ts

Current relevant block (hooks.server.ts:84-117), for reference:
```ts
const secure = event.url.protocol === "https:"
let meiliCookie = event.cookies.get('meilisearch_token');
let meilisearchToken: string | undefined = undefined;
const currentUserId = pb.authStore.record?.id || 'public';

if (meiliCookie) {
  const [token, ownerId, version] = meiliCookie.split('|');

  if (ownerId === currentUserId && Number(version) === SEARCH_TOKEN_VERSION) {
    meilisearchToken = token;
  } else {
    // Identity mismatch (e.g. just logged in/out) or stale token version
    event.cookies.delete('meilisearch_token', { path: '/' });
  }
}

if (!meilisearchToken) {
  try {
    const tokenResponse = await pb.send("/search/token", { method: "GET", fetch: event.fetch });
    meilisearchToken = tokenResponse.token
    event.cookies.set('meilisearch_token', `${meilisearchToken}|${currentUserId}|${SEARCH_TOKEN_VERSION}`, {
      path: '/',
      httpOnly: false,
      maxAge: 60 * 60 * 24,
      sameSite: 'lax',
      secure: secure
    });
  } catch (e) {
    if (url.pathname.startsWith("/api")) {
      return handleError(e)
    }
    throw error(500, "Failed to invalidate meilisearch token: " + e)
  }
}
```

`SEARCH_TOKEN_VERSION` is defined near the top of the file (currently `= 1`), used purely for cache-busting, not expiry.
</context>

<tasks>

<task type="auto">
  <name>Task 1: Track and check real token expiry in the meilisearch_token cookie</name>
  <files>web/src/hooks.server.ts</files>
  <action>
Extend the cookie payload format from `token|ownerId|version` to `token|ownerId|version|expiresAtMs`:

1. Introduce a shared constant for the TTL in milliseconds near the existing `SEARCH_TOKEN_VERSION` constant, e.g. `const SEARCH_TOKEN_TTL_MS = 60 * 60 * 24 * 1000;` (24h, matching the existing `maxAge: 60 * 60 * 24` and the Go backend's `24 * time.Hour` in `db/util/meilisearch.go:532`). Reuse this constant for both the cookie's `maxAge` (in seconds: `SEARCH_TOKEN_TTL_MS / 1000`) and the new expiry calculation, so the two stay in lockstep.

2. When reading the cookie back (the `if (meiliCookie)` block), destructure the 4th field as `expiresAtMs` and additionally check it before accepting the cached token: reuse the cached token only if `ownerId === currentUserId && Number(version) === SEARCH_TOKEN_VERSION && Number(expiresAtMs) > Date.now() + 60_000` (the `+ 60_000` is a 60s safety buffer so a token doesn't expire mid-request). If the expiry check fails, fall into the same `event.cookies.delete(...)` + re-mint path already used for owner/version mismatches — do not add a separate branch, just extend the existing condition.

3. When minting a fresh token (the `if (!meilisearchToken)` block), compute `const expiresAtMs = Date.now() + SEARCH_TOKEN_TTL_MS;` and include it in the cookie value: `` `${meilisearchToken}|${currentUserId}|${SEARCH_TOKEN_VERSION}|${expiresAtMs}` ``. Keep `maxAge: SEARCH_TOKEN_TTL_MS / 1000` (or the existing `60 * 60 * 24` literal — either is fine as long as it stays numerically 24h).

Do not add any new dependency (no JWT decoding needed — the server computes and owns this expiry itself, it doesn't need to parse the actual Meilisearch JWT). Do not change the `/search/token` PocketBase call, the `SEARCH_TOKEN_VERSION` cache-busting semantics, or anything outside this cookie read/write block.
  </action>
  <verify>
    <automated>cd web && grep -q "expiresAtMs" src/hooks.server.ts && npx tsc --noEmit -p . 2>&1 | grep -i "hooks.server.ts" | grep -v "^$" ; echo "tsc check done (no hooks.server.ts errors expected above)"</automated>
  </verify>
  <done>The meilisearch_token cookie payload includes an expiresAtMs field set at mint time; the read path rejects (and re-mints past) a token whose expiresAtMs is at or past a 60s-buffered "now"; the existing owner/version mismatch re-mint path is unchanged in structure; no new dependency added; TypeScript compiles clean for this file.</done>
</task>

</tasks>

<verification>
- `grep -n "expiresAtMs" web/src/hooks.server.ts` shows both the read (destructure + comparison) and write (computation + cookie value interpolation) sites.
- `cd web && npx tsc --noEmit -p .` reports no new errors introduced by this file.
- Manually reason through: a freshly minted token (expiresAtMs = now + 24h) is reused on the next request without re-minting (normal case, no behavior change for the common path). A cookie with a past `expiresAtMs` is treated identically to a missing cookie — deleted and re-minted transparently, with no 500 surfaced to the caller.
- Login/logout and identity-switch re-minting (existing ownerId/version check) is untouched.
</verification>

<success_criteria>
- The server never reuses a `meilisearch_token` whose real expiry (as it itself computed at mint time) has passed, closing the gap where a stale cookie surviving past 24h would cause every search call to 500 indefinitely.
- No new dependencies; change contained entirely to the existing cookie read/write block in `web/src/hooks.server.ts`.
</success_criteria>

<output>
Create `.planning/quick/260714-qtl-fix-meilisearch-token-cookie-never-valid/260714-qtl-SUMMARY.md` when done.
</output>
