# Codebase Concerns

**Analysis Date:** 2026-09-06

## Tech Debt

### Federation Feature Lacks Test Coverage

**Issue:** The entire federation module (`db/federation/`) contains 8 files (create.go, delete.go, follow.go, like.go, undo.go, announce.go, actor.go, activity.go) with zero test files. This is the core v1 feature described in CLAUDE.md but is untested.

**Files:** 
- `db/federation/create.go` (23,269 lines)
- `db/federation/delete.go` (8,905 lines)
- `db/federation/follow.go` (4,374 lines)
- `db/federation/actor.go` (13,079 lines)
- `db/federation/activity.go` (4,310 lines)
- `db/federation/like.go` (2,853 lines)
- `db/federation/undo.go` (5,418 lines)
- `db/federation/announce.go` (6,648 lines)

**Impact:** ActivityPub instance federation logic—including activity creation, validation, and distribution—cannot be verified to work correctly. Private data could leak if is_public checks are bypassed. Remote instance incompatibilities discovered only in production.

**Fix approach:** Add integration tests covering:
- Create activity fanout with is_public=false filtering
- Delete activity cascades
- Follow/Accept/Undo flows between instances
- Announce (boost/share) validation and privacy checks
- Error handling on unreachable remotes

### Missing API Route Test Coverage

**Issue:** Out of ~20 SvelteKit API routes in `web/src/routes/api/v1/`, only 5 have test files. Major operations (user, trail, comment, follow, like, settings, notification) lack automated verification.

**Files:** 
- Tested: `trail/bounding-box/server.test.ts`, `trail/convert/convert.test.ts`, `regions/[id]/download/server.test.ts`, `regions/[id]/download-dem/server.test.ts`, `regions/[id]/geometry/server.test.ts`
- Untested: `user/+server.ts`, `trail/+server.ts`, `comment/+server.ts`, `follow/+server.ts`, `trail-like/+server.ts`, `settings/+server.ts`, `notification/+server.ts`, `list/+server.ts`, and 10+ others

**Impact:** Request validation, error handling, and authorization logic in proxy endpoints are unverified. Breaking changes to PocketBase or business logic rules deploy without catching data corruption.

**Fix approach:** Establish test stubs for CRUD endpoints, focusing on:
- Zod schema validation (request bodies, query params)
- Authorization checks (is user authenticated, can they edit)
- Response shape and status codes
- Error handling (400, 401, 403, 500)

### Plugin System Insufficient Coverage

**Issue:** Plugin OAuth, hooks, manifest parsing, and policy validation code has minimal test coverage. Critical auth flows in `db/pluginsystem/oauth.go` and `db/pluginsystem/manifest.go` are partially or untested.

**Files:** 
- `db/pluginsystem/oauth.go` (336 lines, no test file)
- `db/pluginsystem/manifest.go` (not checked)
- `db/pluginsystem/policy.go` (not checked)

**Impact:** Plugin token refresh, OAuth redirect validation, and manifest validation could fail silently or introduce security holes (e.g., downgrade https→http, unsanitized redirect URIs).

**Fix approach:** Add tests for:
- OAuth state/verifier generation and validation
- Redirect URI validation (https downgrade, path allowlist)
- Token refresh retry logic
- Manifest network policy enforcement

### Large, Complex Screens Risk Bugs

**Issue:** Two screens exceed 1500+ lines, making them difficult to understand, test, and maintain:

**Files:** 
- `app/lib/routes/navigation_screen.dart` (2,301 lines) — Recording mode, map interaction, stats display, sheet management, GPS lifecycle
- `app/lib/routes/trail_create_screen.dart` (1,586 lines) — Form state, file uploads, waypoint editing, elevation profile, sync status

**Impact:** Bugs hidden in massive component logic (especially around state mutation and lifecycle). Regression testing requires full E2E. Changes to one feature risk breaking another.

**Fix approach:** Refactor into smaller, single-responsibility components:
- Split navigation_screen into: RecordingSession, MapInteraction, StatsDisplay, GPSManager
- Split trail_create_screen into: TrailForm, WaypointEditor, ElevationChart, FileUpload

### FIT Parser CRC Checks Disabled

**Issue:** File integrity checks are bypassed by default in the FIT parser vendor code.

**Files:** 
- `web/src/lib/vendor/fit-parser/fit_parser.ts` (lines 106-110, 125-131) — Header CRC and File CRC checks commented out with `// TODO: fix Header CRC check` and `// TODO: fix File CRC check`
- `web/src/lib/vendor/fit-parser/binary.ts` (line 419) — Compressed header handling not implemented (`// TODO: handle compressed header`)

**Impact:** Corrupted FIT files (from Garmin watches, etc.) are silently accepted and may produce incorrect GPS traces. No way to detect incomplete uploads or bit-flips.

**Fix approach:** 
- Implement CRC calculation for both header and file sections
- Add compression support for record headers
- Emit warnings (not errors) on CRC mismatch but allow force-import for user recovery

### Panics in Critical Paths

**Issue:** The backend has `panic()` calls in paths that should handle errors gracefully:

**Files:** 
- `db/util/safe_fetch.go:269` — `mustPrefixes()` panics on invalid IP CIDR blocks (acceptable at init-time only; but list is hardcoded, so should never panic)
- `db/pluginsystem/oauth.go:332` — `randomURLToken()` panics if `rand.Read()` fails. This is called during OAuth flow setup; crashes the request instead of returning 500.

**Impact:** OAuth token generation failure crashes the server request instead of returning an error response. Harmless at scale (entropy pool is always available) but poor error discipline.

**Fix approach:** 
- Change `randomURLToken()` to return `(string, error)` and wrap call sites with error handling
- Return 500 Internal Server Error when token generation fails

## Known Bugs

### Unverified Elevation Recording Fixes (Field Test Pending)

**Issue:** Two elevation-related fixes committed 2026-08-09 in recording mode are unverified in the field.

**Files:** 
- `app/lib/components/trail/elevation_profile.dart` (line 69) — `buildElevationTrackPoints()` now requires a `List<Wpt>.of(breadcrumb)` copy per rebuild because the breadcrumb is identity-stable and gpx 2.3.0's `==` has no identity short-circuit
- `app/lib/provider/navigation_stats_provider.dart` (line 228) — `hasUsableAltitude()` gate treats `altitude: 0, altitudeAccuracy: 0` as "no reading" to skip the synthetic seed fix

**Symptoms:** 
- Elevation profile stuck at zero for entire recording (may indicate `hasUsableAltitude` rejecting real fixes)
- Empty elevation chart on imported trails or older recordings (deliberate removal of `?? 0` fallback)
- Elevation gain jump on first GPS fix (seed fix no longer applied)

**Workaround:** None. User confirmed on next hike (from 2026-08-09 memory entry).

**Follow-up:** See `.planning/debug/recording-elevation-gain-jump.md` (unarchived, left open for this reason) for judgment calls and detailed trade-offs.

### Dismissed Tracking Notification Does Not Re-Post

**Issue:** Android 14+ allows users to swipe away the tracelet foreground-service notification while tracking continues. The notification does not return on its own, making live tracking appear dead.

**Files:** 
- Tracelet SDK integration (not directly in Wanderer source; SDK behavior)
- App would need changes to `app/lib/provider/foreground_position_stream_provider.dart` or tracelet config to implement re-posting

**Impact:** User loses the only signal that recording survived app being closed. Confusing UX: swipe to dismiss → app appears "stopped" but is still recording.

**Workaround:** User must reopen app to verify recording status.

**Fix approach:** (Deferred, complex) — Use `Tracelet.registerHeadlessTask()` + `onHeartbeat` event to re-call `setConfig()` every 15–60s, which should re-post the notification. Requires SDK behavior verification first (test whether `setConfig()` actually re-posts a dismissed notification).

## Security Considerations

### Content-Security-Policy Allows Inline Scripts

**Issue:** Regions admin dashboard has loosened CSP to allow MapLibre functionality.

**Files:** 
- `db/routes/regions_ui.go:54` — `script-src 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://unpkg.com`

**Risk:** Potential XSS via maplibre.js or cached third-party scripts. The SPA does read the admin's JWT from localStorage (line 38 note says it's never embedded), but inline script execution + data attributes still pose attack surface.

**Current mitigation:** 
- Page contains no secrets or tokens
- Privileged operations go through PocketBase JWT-validated REST API
- `X-Frame-Options: DENY` prevents clickjacking
- Admin JWT is runtime-read from localStorage, not embedded

**Recommendations:** 
- Audit maplibre 0.3.5 for known XSS bugs
- Consider nonce-based CSP for inline scripts instead of blanket `'unsafe-inline'`
- Monitor for newer maplibre versions that support stricter CSP

### Connector HTTP Policy Validation

**Issue:** Custom connector TLS configs allow private IPs and custom CAs, which could be exploited for SSRF if plugin manifests are not carefully vetted.

**Files:** 
- `db/util/safe_fetch.go:154-192` — `ConnectorHTTPClient()` applies `AllowPrivate` flag

**Risk:** A malicious plugin manifest could declare connectors pointing to internal services (e.g., `127.0.0.1:8080`, `192.168.x.x` ranges) if `AllowPrivate: true`.

**Current mitigation:** 
- IP filtering in DialContext (lines 166-185) rejects loopback, link-local, multicast
- Special-purpose IP prefixes blocked (lines 243-262: 0.0.0.0/8, 127.0.0.0/8, 192.0.0.0/24, etc.)
- Manifest permissions model controls which connectors each plugin can use

**Recommendations:** 
- Audit plugin manifest loader for path traversal or injection in connector definitions
- Log all connector requests (especially to private IPs) for audit trails
- Consider plugin sandboxing or signing to prevent untrusted manifests

## Performance Bottlenecks

### Map Panning CPU Burn (Addressed, But History Preserved)

**Issue:** (Resolved 2026-08-06) Map panning was consuming ~187% avg / 237% peak CPU (nearly two cores) due to `androidTextureMode: true` + `androidMode: tlhc_vd`.

**Files:** 
- `app/lib/components/base/trail_map.dart` — MapOptions constructor (4 call sites)

**Fix:** Applied `androidTextureMode: false` + `androidMode: AndroidPlatformViewMode.hc` to all call sites.

**Remaining baseline:** `nderer.wanderer` platform thread at 33.7% (MapLibre's own render loop + gesture handling) is acceptable.

**Why preserved:** Reference for future performance regressions if MapLibre or dependencies update.

### GPS 100% Duty Cycle

**Issue:** Foreground position stream has no `LocationSettings`, keeping GPS enabled full-time even when stationary.

**Files:** 
- `app/lib/provider/foreground_position_stream_provider.dart` — `getPositionStream()` call site

**Impact:** ~2–3 mAh direct drain, but triggers ~80 WiFi scans per session (24% wall time), causing secondary heat from Play Services.

**Fix approach:** Add `LocationSettings` with `accuracy: LocationAccuracy.best` and `timeInterval: 1000` (1s) + stationary mode config (e.g., `stationaryPeriodicInterval: 20s`).

### Large Generated Files Slow IDE/Builds

**Issue:** Several auto-generated files exceed 3000+ lines and slow IDE indexing, search, and build times:

**Files:** 
- `app/lib/vendor/tiptap_flutter/lib/src/editor/tiptap_bridge.dart` (1,334 lines, generated)
- `app/lib/vendor/tiptap_flutter/lib/src/editor/tiptap_editor.dart` (1,226 lines, generated)
- `app/lib/i18n/app_localizations.dart` (2,074 lines, generated)
- `app/lib/models/trail.freezed.dart` (2,114 lines, generated)

**Impact:** IDE lag when searching/refactoring. Builds include these in their dependency analysis unnecessarily.

**Fix approach:** 
- Ensure `build_runner` excludes generated files from analysis (check `analysis_options.yaml`)
- Consider code splitting i18n (e.g., lazy-load per language) if file size continues growing
- Verify freezed configuration generates minimal code (no unnecessary methods)

## Fragile Areas

### Riverpod Late-Final Rebuild Bug Pattern

**Issue:** Riverpod Notifier instances survive across rebuilds; only `build()` re-runs. A `late final` field assigned in `build()` throws `LateInitializationError` on the second rebuild.

**Files:** 
- `app/lib/provider/trail/trail_filter_provider.dart` — Fixed 2026-07-30 (commit 33fad8a4), was `late final TrailFilterData defaultFilter`

**Safe modification:** Before adding a provider to `lib/util/account_scope_invalidation.dart`'s `accountScopedProviders`, grep the notifier for `late final` fields assigned in `build()`. Change to `late` (non-final) or nullable.

**Test coverage:** When fixing this, verify regression test fails against old code (temporarily restore `late final` and re-run). A test passing both ways proves nothing.

### iOS Build Requires Hand-Patched Swift Files

**Issue:** Dev machine (2016 MacBookPro + Sonoma via OpenCore Legacy Patcher) is capped at Xcode 16.2 (Swift 6.0.3). Several Flutter plugins ship Swift 6.1 trailing commas in call argument lists, which Swift 6.0.3 rejects.

**Files:** 
- `~/.pub-cache/hosted/pub.dev/maplibre_ios-0.3.5/.../MapLibreViewFactory.swift:26` (hand-patched)
- `~/.pub-cache/hosted/pub.dev/receive_sharing_intent-1.9.0/.../RSIShareViewController.swift:257` (hand-patched)
- `~/.pub-cache/hosted/pub.dev/tracelet_ios-3.5.0/.../TraceletIosPlugin.swift:241, :285` (hand-patched)
- `~/.pub-cache/hosted/pub.dev/maplibre_ios-0.3.5/lib/src/maplibre_ffi.g.dart` (73 call sites forced off objc_msgSend_stret for x86_64 simulator)

**Risk:** `dart pub cache repair` or dependency bump silently reverts patches, causing build failures with "Unexpected ',' separator" in third-party Swift code.

**Fix approach:** If iOS build fails with parser error:
1. Re-scan every package in `app/pubspec.lock` for trailing commas before `)` (array literals are legal)
2. Drop comma from function argument lists only
3. Verify fix applies to both Swift and Dart code

### iOS Share Extension Blocked

**Issue:** Xcode App Groups setup requires an Apple Developer account, which is not available on the dev machine.

**Files:** 
- iOS target configuration (not directly in source)

**Impact:** Share extension not yet built; cannot test "Share to Wanderer" flow.

**Workaround:** None. Blocked until Apple Developer account is available.

## Scaling Limits

### Instance Federation Fanout Not Load-Tested

**Issue:** The `db/federation/create.go` and related files implement activity fanout to all followed remote instances, but no load testing exists for 100+ federated instances or 1000+ trail objects per day.

**Files:** 
- `db/federation/create.go` (fanout logic)
- `db/federation/activity.go` (activity construction)

**Limit:** Unknown; likely acceptable for small numbers of instances but untested at scale. If one remote is slow or down, no retry queue exists (activities dropped if unreachable, per CLAUDE.md).

**Scaling path:** 
1. Add activity queue + retry mechanism with exponential backoff
2. Implement activity delivery concurrency limits (e.g., max 5 concurrent fanouts)
3. Monitor fanout latency and implement circuit breaker for consistently failing remotes

## Dependencies at Risk

### iOS Swift Trailing Comma Workaround Blocks Upgrades

**Risk:** Cannot upgrade maplibre_ios, receive_sharing_intent, or tracelet_ios without re-patching Swift files. Patch is temporary; long-term solution is upgrading to a macOS/Xcode version supporting Swift 6.1.

**Migration plan:** Upgrade dev machine to newer hardware or switch to arm64 Apple Silicon simulator (which uses different Objective-C calling convention and doesn't need the maplibre_ffi.g.dart patches).

### FIT Parser Vendor Code Lacks Maintenance

**Risk:** `web/src/lib/vendor/fit-parser/` (8,339 lines) has multiple TODO comments and unfinished CRC/compression implementations. Upstream maintenance unknown.

**Impact:** Cannot import certain Garmin file formats; data integrity not verified.

**Migration plan:** 
- Survey FIT parsing libraries in npm ecosystem (e.g., `fit-parser`, `garmin-fit`)
- If none suitable, commit to finishing CRC + compression in current vendor fork
- Add tests to prevent regression

## Missing Critical Features

### No Offline Sync Queue

**Issue:** Per CLAUDE.md constraint "Online-only: No offline sync buffer; if a remote instance is unreachable, the activity is dropped (existing behavior for user federation)". This is by design but creates data loss risk.

**Impact:** User publishes trail → federation fanout starts → Remote A is down → activity dropped → remote A never sees the trail. Manual admin intervention required to resync.

**Blocks:** None; this is accepted constraint for v1.

### Notification Re-Posting Deferred

**Issue:** Android 14+ dismissed tracking notification does not return, blocking full tracking UX. This is tracked in memory but deferred indefinitely.

**Impact:** Users must manually reopen app to verify recording, missing the "set it and forget it" experience.

**Blocks:** None; logged as known limitation.

## Test Coverage Gaps

### No E2E Tests for Multi-Instance Federation

**Issue:** The federation feature has no end-to-end tests that spin up two Wanderer instances, configure federation, and verify bidirectional content sync.

**Files:** 
- `app/test/` — Mobile tests exist for trails/maps/recording but not federation
- `web/` — No federation E2E tests

**Risk:** Instance federation flows (Follow, Accept, Create, Delete, Undo) are untested in integrated environment. Breaking changes discover only on production deployments.

**Priority:** High

**Fix approach:** Add Playwright E2E test that:
1. Spin up two Docker containers with Wanderer instances
2. Create admin accounts on each
3. Execute Follow handshake between instance actors
4. Publish trail on instance A → Verify appears on instance B
5. Delete trail on instance A → Verify deleted on instance B

### Mobile Trail Sync Provider Lacks Unit Tests

**Issue:** `app/lib/provider/trail/trail_sync_provider.dart` and related sync logic have no isolated unit tests.

**Files:** 
- `app/lib/provider/trail/trail_sync_provider.dart` (not checked for tests)
- `app/lib/store/local_trail_store.dart` (1,408 lines, complex state machine)

**Risk:** Trail sync edge cases (partial uploads, conflict resolution, local-vs-server divergence) are untested. Regressions surface only in user reports.

**Priority:** Medium

### Plugin System Integration Tests Missing

**Issue:** Plugin OAuth flows, manifest loading, and network policy enforcement lack integration tests.

**Files:** 
- `db/pluginsystem/` (OAuth, manifest, policy code untested)

**Risk:** Plugin system cannot be confidently refactored or extended. Security issues (SSRF, token exposure) not caught.

**Priority:** High (security-adjacent)

---

*Concerns audit: 2026-09-06*
