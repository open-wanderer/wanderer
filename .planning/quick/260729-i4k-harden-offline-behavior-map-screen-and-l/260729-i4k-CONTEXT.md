# Quick Task 260729-i4k: Harden offline behavior across map, list, nav, and profile screens - Context

**Gathered:** 2026-07-29
**Status:** Ready for planning

<domain>
## Task Boundary

Harden the app's behavior when opened while offline:
1. map_screen: show a clear "You are offline" message with a "Try again" CTA instead of attempting to load the map.
2. list screen: same treatment — currently shows a generic "Something went wrong" error instead of a clear offline state.
3. Bottom navigation bar profile picture should be cached and still display when offline.
4. Profile screen currently shows a loading spinner then an error when offline. Instead it should render the cached profile, with settings always available.

</domain>

<decisions>
## Implementation Decisions

### Offline detection
- Introduce a global Riverpod `onlineStatusProvider` (keepAlive) as the single source of truth for "is the backend reachable right now":
  - Seeded at app launch by calling the existing `isBackendReachable(ref)` probe (`lib/util/connectivity_util.dart`) once.
  - Kept live afterward via a `dio` `Interceptor` registered on the `api_provider` client: a successful response (`onResponse`) sets the state to online; a connection-level failure (`onError` where the `DioException.type` is a connectivity failure — `connectionTimeout`, `connectionError`, `unknown` wrapping a `SocketException`, etc.) sets it offline. Ordinary HTTP error responses (404/422/500 — anything that reached the server) must NOT flip the state to offline, since those prove the backend IS reachable.
  - Map/list/profile screens (and the bottom-nav avatar) consume `onlineStatusProvider` reactively instead of each independently probing — this gives auto-recovery the moment any subsequent API call succeeds, without polling or a new dependency like `connectivity_plus`.
  - The "Try again" CTA on map/list screens triggers a fresh explicit probe (re-running `isBackendReachable`, which itself goes through the same dio client/interceptor) so the user can force a re-check rather than waiting for an incidental API call.
- **Migrate existing consumers of `isBackendReachable`** (`lib/main.dart` lines ~240 and ~287, `lib/routes/trail_source_select_screen.dart` line ~99) to read `onlineStatusProvider` instead of calling `isBackendReachable(ref)` directly, so there is a single reactive source of truth app-wide. Keep `isBackendReachable` itself as the underlying probe function used to seed/refresh the provider — do not delete it, just stop calling it directly from those three call sites.

### Map screen offline UI
- Full-screen takeover of the map area with the offline message + Try again button — but the draggable/scrollable bottom sheet (already part of map_screen) stays intact and usable. Only the map itself is replaced, not the whole screen including the sheet.

### List screen offline UI
- Same full-screen-takeover treatment (replacing the generic "Something went wrong" error) with the offline message + Try again CTA.

### Profile cache
- Bottom-nav avatar and profile screen basic info (name, avatar) are reconstructed from the locally stored `UserEntity` (existing local DB) — no new caching layer needed for the core profile fields.
- Anything beyond basic profile info that requires a network fetch (e.g. stats, remote-only data) is replaced by a "You are offline" empty state rather than attempting to load and erroring.

### Settings offline
- Settings are visible and browsable while offline (read-only in the sense that anything requiring network — save/sync — is disabled or deferred until connectivity returns), per "Read-only until online (Recommended)". Settings must always be reachable from the profile screen even when offline.

### Claude's Discretion
- Exact wording of the "You are offline" message and Try again button styling — follow existing app copy/style conventions (Polar Night palette, IBM Plex Sans, flat elevation per DESIGN.md if applicable to the Flutter app, otherwise existing Flutter widget conventions).
- Whether individual settings rows need per-row disabling vs. a single banner — pick whichever matches existing settings screen patterns with least structural change.

</decisions>

<specifics>
## Specific Ideas

No specific UI mockups or exact copy provided — implement using existing app conventions and the `isBackendReachable` pattern already present in the codebase.

</specifics>

<canonical_refs>
## Canonical References

- `lib/util/connectivity_util.dart` — existing `isBackendReachable(ref)` probe, the canonical offline-detection mechanism for this task.
- `lib/main.dart` (around lines 233-291) — existing usage pattern of `isBackendReachable` for offline-aware navigation.

</canonical_refs>
