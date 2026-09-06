---
status: awaiting_human_verify
trigger: "In my closed test, two testers have an encountered an issue where the app reports they are offline after logging in when in fact they are not. I cannot replicate this problem on my phone but maybe you can see a reason in the code."
created: 2026-09-06T00:00:00Z
updated: 2026-09-06T01:30:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: |
    `onlineStatusProvider` is a single global latch that ANY failed request on the
    shared Dio client can flip — including requests that have nothing to do with the
    Wanderer backend. The interceptor in api_provider.dart:55-70 excludes exactly one
    host (`unknown-server.local`) and otherwise calls `markOffline()` for every
    `connectionError` / `connectionTimeout` / SocketException, regardless of which host
    the request went to.

    The shared client also fetches map glyphs and sprites, and those default to a
    THIRD-PARTY host: `/api/v1/map/style-sources` falls back to
    `https://protomaps.github.io/basemaps-assets` whenever the operator has not set
    `MAP_ASSETS_URL` (web/src/routes/api/v1/map/style-sources/+server.ts:50). The warm
    is 4 fontstacks x 256 ranges = 1024 downloads plus 8 sprite files, pooled 8 at a
    time (glyph_sprite_cache_provider.dart:17,21,85-107), fired on the first map open
    of a fresh install. One dropped connection anywhere in that burst — GitHub Pages
    throttling, a carrier NAT, a slow link — flips the whole app to "offline" while the
    Wanderer server is perfectly reachable.

    It then LATCHES: offline screens stop issuing requests, so no successful response
    arrives to flip it back, and every "retry" re-probes `/api/v1/health`, whose own
    probe is stricter than the transport it runs on (200-only, 5s) and whose cost
    QUADRUPLES once a user is signed in.

    First-run-only by construction: the on-disk cache plus the `.missing-urls.txt`
    404 memo make the whole warm a no-op on later runs — which is exactly why fresh
    closed-test installs hit it and the developer's long-warm phone does not.
  confirming_evidence:
    - "api_provider.dart:55-70 — the onError interceptor host-checks ONLY `kUnconfiguredApiHost`; any other host's connection failure calls `markOffline()` app-wide."
    - "web/src/routes/api/v1/map/style-sources/+server.ts:50 — `env.MAP_ASSETS_URL ?? 'https://protomaps.github.io/basemaps-assets'`; glyphUrl and spriteUrl are built from it, so unset => third-party GitHub Pages host."
    - "glyph_sprite_cache_provider.dart:17 `_rangeCount = 256`, :21 `_maxConcurrentDownloads = 8`, :85-107 builds 4x256 glyph jobs + 8 sprite jobs, :131 `_runPooled(api, ...)` — all through the SHARED api client (`ref.read(apiProvider)` at :76)."
    - "glyph_sprite_cache_provider.dart:113-118 — the code's own comment records this used to be '~1000 HTTP round-trips per app start forever'; the 404 memo made it first-run-only, which matches 'new testers only, not my phone'."
    - "online_status_provider.dart:28-35 — `isConnectionFailure` returns true for `unknown` wrapping a SocketException, i.e. a dropped/reset connection during the burst counts as 'we are offline'."
    - "Latch behaviour: wanderer_layout.dart:23 (banner), map_screen.dart:366, list_screen.dart:66, profile_screen.dart:389/450/584, trail_source_select_screen.dart:200 all render offline states instead of issuing requests once the flag is false, so the interceptor's markOnline path never gets a response to see."
    - "account_scope_invalidation.dart:12 — `onlineStatusProvider` is deliberately excluded from account-scope invalidation, so a false verdict survives logout/login and account switches."
    - "connectivity.dart:19-26 — the probe demands `statusCode == 200` within a hard `.timeout(5s)`, while the client's own `connectTimeout` is 8s (api_provider.dart:36): the probe gives up before the transport would."
    - "web/src/hooks.server.ts:132-147 — once a pb_auth cookie is present EVERY request, `/api/v1/health` included, costs `authRefresh` + `settings.getFirstListItem` + `activitypub_actors.getFirstListItem` server-side, on top of the route's own `pb.health.check()` (health/+server.ts:38). An unauthenticated probe costs one PocketBase hop; an authenticated one costs four. That is why the probe starts timing out specifically AFTER login on a loaded shared instance."
    - "hooks.server.ts:145-146 — `settings`/`actor` lookups are unguarded `getFirstListItem` calls: an account missing either row makes every authenticated request 500, which the 200-only probe also reads as offline. Account-specific, hence reproducible for some testers and not the developer."
  falsification_test: |
    Ask an affected tester what still works while the app claims offline. If remote
    trail lists / search / profile still load, the backend is reachable and the flag is
    lying (this hypothesis). If nothing loads, the base URL or the account itself is
    broken (see Secondary Findings 2-3) and this hypothesis is wrong.
    Second: have them force-stop and reopen the app. Recovering until the next map open
    points at the glyph/sprite burst; staying offline immediately after login points at
    the probe (5s timeout, or a 500/503 from the auth hook).
    Third, server-side, decisive for the third-party-host half: check whether the demo
    instance sets `MAP_ASSETS_URL`. Unset => glyph warm hits protomaps.github.io.
  fix_rationale: |
    The flag must only ever be set by traffic that actually proves something about the
    Wanderer backend, and must never be set by a single unconfirmed failure.
    1. Interceptor: ignore any request whose host is not the configured server host
       (today it ignores only the placeholder host). Third-party glyph/sprite/asset
       fetches then cannot report the backend down.
    2. Probe: any HTTP response proves reachability — align `isBackendReachable` with
       the rule `isConnectionFailure` already documents; keep 503 as "database down";
       raise the 5s timeout to at least the 8s connectTimeout.
    3. Optional hardening: make `markOffline` from a single request failure provisional
       — confirm with one probe before flipping the global flag.
  blind_spots: |
    Not verified on a tester device or against the live demo instance; no network calls
    were made. Which of the three triggers fired for these two testers is not yet
    pinned down — the falsification tests above discriminate. Fix (1) is correct
    regardless; fix (2) only helps if the probe (not a real connection failure) is what
    latched it.

next_action: |
  Fixes 1 and 2 applied and self-verified (flutter analyze clean on the changed files,
  full suite 1105/1105 passing, 13 new tests). Fix 3 (provisional markOffline) NOT
  applied — not requested.

  Awaiting human verification: user builds/installs and confirms with the two affected
  testers. The discriminating questions are still worth asking, because they say WHICH
  trigger fired — and if the answer is "nothing loads at all", neither applied fix is
  the cause and Secondary Findings 2-3 become the next lead.

## Fixes Applied (2026-09-06, uncommitted)

- app/lib/provider/api_provider.dart — new `isBackendRequest(dio, uri)`; both
  interceptor callbacks now gate on it, so only traffic to the configured server's
  host moves `onlineStatusProvider`. Third-party glyph/sprite fetches
  (protomaps.github.io by default) can no longer report the backend down, and a
  third-party SUCCESS can no longer clear a genuine offline state either. Subsumes the
  old `kUnconfiguredApiHost` special case.
- app/lib/util/connectivity.dart — `isBackendReachable` now treats any HTTP response
  as reachable (`validateStatus: (_) => true`, `statusCode != 503`), maps only
  connection-level failures to offline via `isConnectionFailure`, and its ceiling went
  from 5s to 10s so it no longer gives up before the client's own 8s connectTimeout.
  `isConnectionFailure` moved here (a pure helper must not depend on a provider) and is
  re-exported from `online_status_provider.dart`, so no caller changed.
- app/test/provider/online_status_provider_test.dart — 13 new tests: the probe's
  verdict per status (200/404/500/503) and per transport failure, plus end-to-end
  interceptor host gating (backend failure latches offline; third-party failure does
  not; third-party success does not clear offline; placeholder host stays inert).

## Symptoms

expected: After signing in against a reachable Wanderer instance, the app treats itself as online — no offline banner, screens load remote content.
actual: The app reports offline after login although the device has working connectivity and the server is up.
affected_set: Two closed-test testers, both on the demo instance. Not reproducible on the developer's own phone.
errors: None reported.
platform: Flutter app (app/lib/).
notes: Demo instance runs the `feature/app` code, so `/api/v1/health` DOES exist there.

## Eliminated

- hypothesis: "The `/api/v1/health` SvelteKit route is missing on the testers' server (it exists only on feature/app, not main), so the 200-only probe reads 404 as offline."
  why: User confirmed the testers use the demo instance, which runs feature/app code and therefore serves the route. NOTE: this remains a real latent bug for anyone pointing the app at a released/self-hosted Wanderer build — `web/src/routes/api/v1/health/+server.ts` is absent from main and origin/main (verified via `git ls-tree main`), and `db/routes/health.go` is registered on PocketBase's own router (db/main.go:201), not on the SvelteKit proxy the app talks to. Fix (2) neutralises it.
- hypothesis: "Device-level connectivity / airplane mode on the testers' phones."
  why: Login itself succeeded — a request that reached the server through the same Dio client.
- hypothesis: "The interceptor misclassifies an HTTP error status as a connection failure."
  why: `isConnectionFailure` (online_status_provider.dart:28-35) excludes badResponse/badCertificate/cancel and the interceptor calls `markOnline()` for those. A wrong verdict has to come from a real socket-level failure or from the probe.

## Secondary Findings (independent defects in the same path)

1. connectivity.dart:21 — the probe's hard `.timeout(5s)` is shorter than the client's
   own `connectTimeout` of 8s. The probe gives up before the transport would, and it
   does so on the request that got 4x more expensive after login (hooks.server.ts).
2. server_selection_screen.dart:32-39 — `_selectAndGoBack` computes a normalized `url`
   (prepending `https://` for a bare host, exactly what the field hint "e.g.
   wanderer.to" invites) and then discards it, passing the RAW `server.url` to both
   `setSelectedServer` and `updateBaseUrl`. A typed bare domain yields the scheme-less
   base URL `wanderer.to/api/v1`.
3. models/user.dart:48-49 — `serverUrl` is rebuilt from the actor IRI as
   `Uri(scheme: userIri.scheme, host: userIri.host)`, dropping the PORT and any path
   prefix. The IRI is built from the server's `ORIGIN` env var (db/federation/actor.go:164),
   so a non-default port, a subpath deployment, or a misconfigured ORIGIN persists a
   wrong `serverUrl`, which `Auth.build()` (auth_provider.dart:52) applies as the API
   base URL on the NEXT cold start — after which the app is genuinely unreachable and
   the offline report is correct. Also poisons `getFileUrl` avatar/photo URLs.

## Follow-up: Secondary Findings 2 and 3 fixed (2026-09-06)

Both were fixed in their own commit, on top of the connectivity work.

- Finding 2's user-visible face: the picker did not just mis-set the base URL, it
  never closed. `Dio.options.baseUrl`'s setter THROWS
  (`ArgumentError.value(..., 'Must be a valid URL on platforms other than Web')`,
  dio/lib/src/options.dart:101) when `Uri.parse(value).host` is empty, so
  `updateBaseUrl("wanderer.to/api/v1")` threw before `context.pop()` ran. The
  selection had already been stored by then, leaving a selected server the client
  was not pointed at.
- New `app/lib/util/server_url.dart`: `normalizeServerUrl` (fills in https://,
  strips trailing slashes, keeps port and subpath, null when still hostless) and
  `serverUrlFromActorIri` (strips the `/api/v1/activitypub/user/` suffix the
  backend builds IRIs with — db/util/activitypub.go:68 — keeping port AND subpath
  prefix).
- Call sites: server_selection_screen.dart uses the normalized URL for both the
  stored selection and `updateBaseUrl`; models/user.dart derives `serverUrl` via
  `serverUrlFromActorIri`; api_provider.dart's `updateBaseUrl` normalizes as a
  choke-point defence. `Auth.build()` deliberately still reads the stored
  `serverUrl` scalar: an install whose persisted value already lost its port
  would need a re-login to recover, but no such install exists yet, so the
  self-heal was not worth the extra code path.
- 17 new tests; full suite 1121 passing.

## Evidence

- timestamp: 2026-09-06T00:10:00Z
  checked: app/lib/util/connectivity.dart, app/lib/provider/online_status_provider.dart, app/lib/provider/api_provider.dart
  found: Probe requires 200 within 5s; `refresh()` overwrites the interceptor-maintained state; the interceptor marks offline on any connection-level failure from any host except the placeholder.
- timestamp: 2026-09-06T00:40:00Z
  checked: git ls-tree main/origin/main; git branch --contains 38255e77; web/src/hooks.server.ts on main; db/main.go:201
  found: The health route is feature/app-only and PocketBase's own /health is on a different port. Falsified as the cause here (demo runs feature/app) but retained as a latent bug for other servers.
- timestamp: 2026-09-06T01:05:00Z
  checked: web/src/hooks.server.ts:78-150 and web/src/routes/api/v1/health/+server.ts
  found: An authenticated request costs authRefresh + settings + actor lookups before the route body runs; the health route then makes a fourth PocketBase call. Unauthenticated it is one. The probe's cost — and its failure probability against a 5s ceiling — jumps precisely at login.
- timestamp: 2026-09-06T01:20:00Z
  checked: app/lib/provider/glyph_sprite_cache_provider.dart, app/lib/provider/map_style_sources_provider.dart, web/src/routes/api/v1/map/style-sources/+server.ts
  found: 1024+8 asset downloads on first map open, through the shared status-bearing client, against protomaps.github.io by default. Callers: trail_map.dart:106, navigation_screen.dart:1311, trail_download_state_provider.dart:112.
