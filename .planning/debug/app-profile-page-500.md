---
status: investigating
trigger: "Users are reporting that they see this error when they click on the profile page in the app. Unfortunately, I do not have more intel right now."
created: 2026-09-29
updated: 2026-09-29
---

## Current Focus

hypothesis: The 500s are specific to AUTHENTICATED requests to /api/v1/profile/{handle}/feed and /api/v1/profile/{handle}/lists (own profile, is_local actor). Anonymous requests to the exact same endpoints/params against the live server succeed with real data, so the bug does not reproduce without a logged-in session. Code reading has not found a concrete authenticated-only failure path in web/ or db/ that explains it — need the actual server-side error/stack trace to proceed.
test: Requested checkpoint — ask user to pull the web/db container log lines for the failing request (see checkpoint below).
expecting: A Go panic/error (db logs) or a JS exception (web logs, from handleError's console.error(e) in feed/+server.ts, or an uncaught throw in lists/+server.ts) that names the actual failure point.
next_action: Awaiting checkpoint response with the server log line(s) for the 500. Once received, match the error message/stack to the specific line in web/src/routes/api/v1/profile/[handle]/{feed,lists}/+server.ts or db/federation/actor.go and re-enter investigation_loop Phase 2 with a concrete hypothesis.

## Symptoms

expected: Tapping the "Profil" tab in the Flutter app (app/) shows the user's own profile with its content sections (trails, lists, etc.).
actual: The profile header renders fine (avatar, username "huggenknubbel", handle "@huggenknubbel@wanderer.knubbel.me", settings + share buttons). Below it, at least TWO content sections each render the generic error empty-state ("Etwas ist schief gelaufen" / "Something went wrong"). A partially visible row of cards/tabs sits right under the header, cut off.
errors: |
  DioException [bad response]: This exception was thrown because the response has a status code of 500 and RequestOptions.validateStatus was configured to throw for this status code.
  The status code of 500 has the following meaning: "Server error - the server failed to fulfil an apparently valid request"
  (Same error shown in each failing section. The request URL/endpoint is not shown.)
timeline: Unknown. Reported by users of the app (Android, German locale). Context: feature/app recently merged origin/dev which included Release v0.21.0 (#1279/#1280); the server wanderer.knubbel.me may run a different backend/web version than the app expects. Not known whether it ever worked for these users.
reproduction: Open the app logged in to wanderer.knubbel.me and tap the "Profil" tab in the bottom navigation. No further details from the reporters.
notes: |
  - HTTP 500 means the SERVER failed. Could come from the SvelteKit API layer (web/src/routes/api/v1/...) or from the PocketBase/Go backend (db/). Identify which endpoints the Flutter profile screen calls, and check which of those can 500 for this user/request shape (e.g. query/filter params the app sends that the server version rejects, missing collections/fields, federation/actor lookups for the handle, expand params).
  - Two sections fail while the header succeeds — compare the header's request with the failing sections' requests.
  - Consider app/server version skew (app on feature/app vs deployed release v0.21.0).
  - Per project memory: the /app folder exists only on feature/app. User builds and installs the app themselves — do not run flutter build or adb install. Use GOTOOLCHAIN=go1.26.1 for db tests that run migrations.

## Eliminated

- hypothesis: The "@" prefix in the handle the app sends for own-profile sub-sections (`@huggenknubbel`, from `profile_screen.dart`'s `_handle` getter) breaks Go's `acct:` resource parsing differently than the plain `huggenknubbel` used by `OwnProfile.build()` for the header fetch.
  evidence: Traced db/routes/activitypub.go ActivitypubActor → db/federation/actor.go GetActorByHandle → db/util/activitypub.go SplitHandle. SplitHandle does `strings.TrimPrefix(handle, "@")` before checking for a second "@", so a single leading "@" is stripped cleanly; "@huggenknubbel" and "huggenknubbel" resolve identically (both → username="huggenknubbel", domain="", is_local=true lookup).
  timestamp: 2026-09-29

- hypothesis: Meilisearch "lists"/"trails" index is missing "shares" (or another) attribute in filterableAttributes on the deployed instance, so the authenticated search-token's embedded rule ("public = true OR author = X OR shares = X") 400s and handleError() maps it to 500 (MeiliSearchApiError isn't a ClientResponseError so it falls through to the generic 500 branch).
  evidence: db/main.go initMeilisearchConfig() calls client.Index(name).UpdateSettings() unconditionally on every boot (not gated behind "index missing"), and FilterableAttributes for "lists" already includes {"author","public","shares"} — has for a long time (also present in the original 1742167033 migration). No version-skew plausible here.
  timestamp: 2026-09-29

- hypothesis: PocketBase collection-level privacy rule (listRule/viewRule) on `profile_feed` blocks the request for some viewer state, producing a 500.
  evidence: Read db/migrations/1752321031_created_profile_feed.go — `listRule` is `""` (fully public, no restriction expression at all — actually a separate privacy concern worth flagging separately, since the view's raw SQL doesn't filter on `public`/`is_public` at all, but that would leak data, not 500).
  timestamp: 2026-09-29

- hypothesis: getActorResponseForHandle (shared by /profile/{handle}, /profile/{handle}/feed, /profile/{handle}/lists) itself 500s only when authenticated, e.g. via `e.Auth != nil` branch in db/routes/activitypub.go doing `FindFirstRecordByData("activitypub_actors","user",e.Auth.Id)` for a user with no matching actor record.
  evidence: web/src/hooks.server.ts's `auth` handle does the equivalent lookup (`pb.collection("activitypub_actors").getFirstListItem('is_local=1&&user=...')`) unwrapped in try/catch on EVERY authenticated request, for every route. If that failed for this user, the whole app (not just the profile screen) would 500 on every request. The bug report is scoped to the profile page specifically, implying the rest of the app functions, which implies this lookup already succeeds for this account.
  timestamp: 2026-09-29

- hypothesis: The Go "private profile" check in assembleActor() (db/federation/actor.go) mis-marks the OWN profile as private for the authenticated owner, causing ErrProfilePrivate handling to misfire as a 500.
  evidence: Read assembleActor() private-flag logic — it explicitly excludes the viewer's own actor id (`dbActor.Id != strings.TrimPrefix(actorVal, "actor:")`) from the private check, and even when ErrProfilePrivate IS returned, db/routes/activitypub.go explicitly special-cases `e.Auth != nil && actor.GetString("user") == e.Auth.Id` to still return 200. Also the live account's settings.privacy is effectively empty/public (confirmed via anonymous fetch returning real data), so the `if privacy != ""` gate wouldn't even engage.
  timestamp: 2026-09-29

## Evidence

- timestamp: 2026-09-29
  checked: app/lib/routes/profile_screen.dart — which sliver sections can render the literal WandererError "Etwas ist schief gelaufen" ("something_went_wrong") text vs. degrade silently.
  found: Only `_ListsPreview` (profileListsProvider) and `_FeedSection` (profileFeedProvider) route through `AsyncLoader`, which renders `WandererError` on `asyncValue.hasError` when online. `_CountsRow` (profileCountsProvider) only reads `.value`, never renders an error card — on failure it just stays in skeleton/dash state. `_StatsRow`/header have no independent fetch (data comes from the already-loaded `actor`).
  implication: The "at least TWO content sections" showing the generic error text are almost certainly Lists preview + Feed, i.e. `POST /api/v1/profile/{handle}/lists` and `GET /api/v1/profile/{handle}/feed`. The "partially visible row of cards ... cut off" is consistent with `_CountsRow` being stuck in perpetual skeleton (its own fetch failing silently, not shown as text).

- timestamp: 2026-09-29
  checked: app/lib/provider/profile/profile_provider.dart (OwnProfile) vs app/lib/routes/profile_screen.dart (_handle getter used by Counts/Lists/Feed sections)
  found: `OwnProfile.build()` fetches `/profile/${user.preferredUsername}` (NO "@" prefix) for the header, and on ANY failure (network, 500, offline) silently falls back to the cached `Actor` from ObjectBox (`user.actor.target`) and swallows the error. `_handle` (used by Counts/Lists/Feed) is `"@${...preferredUsername}"` (WITH "@" prefix), a *different* provider family (`profileProvider`/direct fetch), with no such fallback.
  implication: The header rendering successfully is NOT proof that `/profile/{handle}` itself is succeeding right now for this account — it could be failing too and simply masked by the offline-cache fallback. This weakens any "header succeeds so getActorResponseForHandle works" inference. Still, the single-vs-double "@" difference was independently eliminated above (SplitHandle handles both identically on the Go side).

- timestamp: 2026-09-29
  checked: Live anonymous curl against https://wanderer.knubbel.me for the exact endpoints/params the app sends, using the real account from the bug report (huggenknubbel, actor id vk4w3ib4i43869p):
    - GET /api/v1/profile/huggenknubbel → (not shown, inferred via feed expand) actor.is_local=true, domain=wanderer.knubbel.me
    - GET /api/v1/profile/%40huggenknubbel/feed?page=1&perPage=10&sort=-created → 200, real feed items
    - POST /api/v1/profile/%40huggenknubbel/lists with body {"q":"","options":{"hitsPerPage":5,"page":1,"sort":["created:desc"]}} (matches app's default ListFilter: sort=created, order=desc) → 200, real list data
    - GET /api/v1/trail?filter=author='vk4w3ib4i43869p'&perPage=1 → 200, totalItems=73
    - GET /api/v1/list?filter=author='vk4w3ib4i43869p'&perPage=1 → 200, totalItems=1
  found: Every profile-related endpoint the app calls for this exact account, with the exact params the app sends, returns 200 with real data when called anonymously (unauthenticated).
  implication: The backend/data are NOT broken in general for this account. The 500 the reporters see must depend on something specific to the authenticated in-app request (auth cookie / session) that an anonymous curl doesn't reproduce, OR was transient/already resolved server-side, OR depends on client behavior (headers, concurrency) not replicated by curl. Could not go further without either real user credentials (not available and creating a throwaway account on the user's live production instance without explicit permission was avoided) or actual server logs.

## Resolution

root_cause:
fix:
verification:
files_changed:
