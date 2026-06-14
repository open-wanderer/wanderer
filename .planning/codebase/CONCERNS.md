# Codebase Concerns

**Analysis Date:** 2026-06-10

## Tech Debt

**FIT Parser Type Safety:**
- Issue: Large monolithic FIT parser file (8,339 lines) lacks comprehensive TypeScript types. Comments indicate output types are "cumbersome to maintain" and use `any` type extensively.
- Files: `web/src/lib/vendor/fit-parser/fit.ts`, `web/src/lib/vendor/fit-parser/fit_types.ts`, `web/src/lib/vendor/fit-parser/binary.ts`
- Impact: Type safety reduced for fitness activity data parsing; potential runtime errors when processing non-standard FIT file structures
- Fix approach: Either adopt Zod schema generation for FIT types or implement type guards for critical FIT message fields; consider using code generation tools to reduce maintenance burden

**Integration Secret Encryption Workaround:**
- Issue: Secret encryption detection relies on heuristic check (`!util.CanDecryptSecret()`) rather than explicit flag, preventing proper key rotation
- Files: `db/hooks/integrations.go` (lines 114-116)
- Impact: Cannot safely rotate encryption keys; secrets marked as already-encrypted are detected by pattern matching, not by metadata
- Fix approach: Add `encrypted_at` timestamp or `encryption_version` field to integration records; implement proper versioned encryption handling

**Komoot Integration Photo Processing Incomplete:**
- Issue: Komoot photos can contain location data but are not processed into waypoints
- Files: `db/integrations/komoot/komoot.go` (line 437)
- Impact: User location metadata from Komoot photos is discarded; trails imported from Komoot may miss location-tagged photo waypoints
- Fix approach: Parse photo metadata from Komoot API responses and create waypoint entries for geotagged photos

**Trail Edit Page Component Complexity:**
- Issue: Large monolithic Svelte component (2,203 lines) handles form state, map interactions, file uploads, route editing, and validation
- Files: `web/src/routes/trail/edit/[id]/+page.svelte`
- Impact: Difficult to test, maintain, and reason about; high risk of regressions on form changes
- Fix approach: Extract route editor logic to separate Riverpod provider; split form handling into smaller composed components; use Felte store for form state management

**Circular Store Dependencies:**
- Issue: Trail store depends on summit log, tag, list, and waypoint stores; these stores may depend back on trail operations
- Files: `web/src/lib/stores/trail_store.ts` (imports summit_log_store, tag_store, list_store, waypoint_store)
- Impact: Difficult to test stores in isolation; potential for circular update loops during cascade operations
- Fix approach: Centralize mutation logic in a service layer; use event-based updates rather than direct store imports

## Known Bugs

**Meilisearch Token Staleness:**
- Symptoms: User gets logged out/in, but stale Meilisearch token persists in cookie; search becomes unavailable until page refresh
- Files: `web/src/hooks.server.ts` (lines 84-117)
- Trigger: Logout followed by login in same browser session
- Workaround: Manual page refresh forces token refresh; could be triggered automatically on auth state change

**API Error Handling Generic Messages:**
- Symptoms: Frontend catches `APIError` but logs generic `console.error(e)` without capturing full error context
- Files: `web/src/routes/trail/edit/[id]/+page.svelte` (multiple catch blocks at lines 285, 456, 635, 804, etc.)
- Trigger: Any API request failure during trail editing
- Workaround: Check browser console for detailed error; implement error boundary with structured logging

**FIT File CRC Validation Disabled:**
- Symptoms: FIT files with invalid CRC checksums are silently accepted
- Files: `web/src/lib/vendor/fit-parser/fit_parser.ts` (lines 106, 127 marked with TODO comments)
- Trigger: Import of corrupted FIT file
- Workaround: Manual verification of file integrity before import; no automatic detection of corrupted data

**Waypoint Cluster Request Error Not Surfaced:**
- Symptoms: Waypoint clustering can fail silently if clustering service is unavailable; user sees no error notification
- Files: `web/src/routes/trail/edit/[id]/+page.svelte` (line 635-640)
- Trigger: Clustering endpoint returns error
- Workaround: Check network tab in browser DevTools; error is caught and partially logged

## Security Considerations

**HTML Content Injection in Maplibre Popups:**
- Risk: Icon HTML is set via `.innerHTML` in maplibre utility functions; if icon data comes from user input, could allow XSS
- Files: `web/src/lib/util/maplibre_util.ts` (lines 215, 221, 244)
- Current mitigation: Icon data is generated programmatically (Font Awesome icons) and not from user input; hardcoded static icons
- Recommendations: Use `.textContent` for user-provided data; add Content Security Policy header to prevent inline script execution

**Integration Password Decryption at Sync Time:**
- Risk: Komoot/Hammerhead passwords decrypted in memory during sync operations; if process crashes, key material may be in swap
- Files: `db/integrations/komoot/komoot.go` (line 66), `db/integrations/hammerhead/hammerhead.go`
- Current mitigation: Passwords encrypted at rest using `POCKETBASE_ENCRYPTION_KEY`
- Recommendations: Use sealed boxes or ephemeral key derivation; implement zero-copy patterns for sensitive data; audit memory safety of Go crypto libraries

**API Token Validation Missing Expiration:**
- Risk: API tokens accepted without expiration check; compromised token remains valid indefinitely
- Files: `web/src/hooks.server.ts` (line 62-75)
- Current mitigation: Tokens start with `wanderer_key` prefix, allowing basic validation
- Recommendations: Add token expiration timestamp; implement token revocation mechanism; rotate tokens on each use

**ActivityPub Actor Context Not Validated:**
- Risk: Remote actor contexts accepted without schema validation; could inject malicious JSON-LD context
- Files: `db/federation/create.go` (line 48), `db/federation/actor.go`
- Current mitigation: Contexts fetched from remote servers and stored; no validation of schema structure
- Recommendations: Validate actor context against ActivityPub spec; whitelist known context URLs; implement schema signing

## Performance Bottlenecks

**FIT Parser Global State:**
- Problem: FIT parser uses global variables and mutable state during parsing; cannot parallelize multiple file parsing
- Files: `web/src/lib/vendor/fit-parser/binary.ts` (line 493 TODO comment)
- Cause: Parser architecture designed for single-threaded sequential parsing
- Improvement path: Refactor parser to use immutable state machine; allow concurrent file parsing via Web Workers

**Trail Detailed Cache LRU Not Bounded by Memory:**
- Problem: Cache can grow to 200+ Trail objects (with expanded waypoints, comments); no memory pressure release
- Files: `web/src/lib/stores/trail_store.ts` (lines 92-121)
- Cause: Simple size-based LRU eviction doesn't account for Trail object sizes
- Improvement path: Implement memory-aware cache size limit; add cache metrics/telemetry; consider lazy-loading trail details

**Waypoint Clustering O(n²) for Large Trails:**
- Problem: Waypoint clustering request sent for every trail edit operation; no debouncing or batching
- Files: `web/src/routes/trail/edit/[id]/+page.svelte` (likely in waypoint change handlers)
- Cause: Direct synchronous clustering calls without request batching
- Improvement path: Debounce clustering requests; batch multiple waypoint changes into single request; cache clustering results

**Meilisearch Token Fetch on Every Request:**
- Problem: Token cookie checked for every route navigation; can generate unnecessary backend requests
- Files: `web/src/hooks.server.ts` (lines 99-117)
- Cause: Cookie validation does not cache results in request context
- Improvement path: Implement short-lived in-memory token cache per request; store token expiration in cookie

**Search Index Rebuild on Integration Sync:**
- Problem: Komoot/Strava integration sync rebuilds entire Meilisearch index for user's trails
- Files: `db/integrations/komoot/komoot.go`, `db/integrations/strava/strava.go`
- Cause: Full re-index performed after each sync operation
- Improvement path: Implement incremental search index updates; batch updates; use Meilisearch task queue

## Fragile Areas

**Valhalla Route Caching with JSON Diff:**
- Files: `web/src/lib/stores/valhalla_store.svelte.ts`
- Why fragile: Undo/redo implemented using JSON diff/patch on GPX objects; if GPX structure changes, patches become invalid; deep clone overhead for every route change
- Safe modification: Add unit tests for changeset generation and reversion; avoid structural changes to GPX model without migration; consider event sourcing alternative
- Test coverage: No visible unit tests for Valhalla store operations

**IntegrationSecrets Encryption Pattern:**
- Files: `db/hooks/integrations.go`
- Why fragile: Encryption state detection via heuristic function call; if `CanDecryptSecret()` logic changes, could double-encrypt or fail to encrypt new secrets
- Safe modification: Add integration tests for encryption/decryption round-trip; validate each secret field with explicit flag; implement migration to add encryption metadata
- Test coverage: `db/tests/secrets_test.go` exists but scope unclear from file list

**Federation Activity Posting Without Retry:**
- Files: `db/federation/create.go`, `db/federation/*.go`
- Why fragile: Activities posted to remote followers without retry mechanism; if network hiccup occurs, followers don't receive activity
- Safe modification: Implement activity queue with exponential backoff; add delivery status tracking; monitor failed deliveries
- Test coverage: No visible federation integration tests

**Komoot Password Persistence Without Rotation:**
- Files: `db/integrations/komoot/komoot.go`
- Why fragile: Passwords stored encrypted but not rotatable; if master key leaks, all Komoot credentials compromised simultaneously
- Safe modification: Implement key rotation with versioning; add password change detection to trigger re-encryption; audit decryption access
- Test coverage: No visible Komoot integration tests

## Scaling Limits

**Trail Cache Memory Usage:**
- Current capacity: 200 Trail objects with full expand (waypoints, summit logs, comments)
- Limit: Approximately 50-100 MB depending on trail complexity; degrades performance at ~150+ concurrent users viewing different trails
- Scaling path: Implement Redis-backed distributed cache; implement persistent trail cache in ObjectBox (Flutter) or similar; add cache warming strategies

**Meilisearch Index Size:**
- Current capacity: Can index ~1 million trails on standard Meilisearch container (1GB RAM)
- Limit: Beyond 2 million trails, indexing becomes slow; search latency increases
- Scaling path: Shard index by region/user; implement read replicas; separate primary and search indexes

**ActivityPub Federation Inbox Processing:**
- Current capacity: Sequential processing of incoming activities; typical batch size ~50 activities/second
- Limit: Beyond ~500 concurrent followers, inbox processing falls behind; remote followers see delays
- Scaling path: Implement activity queue with worker pool; add priority queues for critical activities (Undo/Delete); implement backpressure

**WebSocket Connections for Real-time Updates:**
- Current capacity: Not implemented; all updates require page refresh
- Limit: Scalability concern for future real-time feed; estimated ~500 connections per server before exhaustion
- Scaling path: Implement Svelte stores with PocketBase subscriptions; use Redis pub/sub for cross-server broadcasts; consider message queue (RabbitMQ)

## Dependencies at Risk

**FIT Parser Vendor Code:**
- Risk: FIT parser is vendored with extensive TODO comments; no upstream maintenance visible
- Impact: Security vulnerabilities in FIT parsing not patched; feature requests (e.g., compressed headers) unfulfilled
- Migration plan: Evaluate fitgo (Go FIT parser) as alternative; implement only required FIT field parsing; consider custom parser for subset of fields

**Maplibre Elevation Profile Vendor Code:**
- Risk: Elevation profile rendering is vendored; no upstream source visible
- Impact: Performance issues, missing features not addressed
- Migration plan: Evaluate Chart.js elevation rendering as alternative; implement custom elevation profile component using Canvas API

**Vector Tile Dependencies (Beta):**
- Risk: `vector_map_tiles` and `vector_tile_renderer` are `beta` versions from custom forks
- Impact: API breaking changes expected; no guarantee of long-term support
- Migration plan: Monitor upstream PMTiles/vector tile ecosystem for stable alternatives; plan for API migration every 6 months

**Dio HTTP Client with Custom Fork Tracking:**
- Risk: `flutter_map_marker_cluster` and other packages may depend on specific Dio versions
- Impact: Dependency conflicts possible when updating Dio
- Migration plan: Audit all transitive dependencies of Dio; test major version upgrades in isolated environment before merge

## Missing Critical Features

**Error Boundaries in Svelte Components:**
- Problem: No global error boundary for critical operations (trail editing, map rendering); failures cascade
- Blocks: Cannot safely recover from partial state corruption during complex operations
- Recommendation: Implement `+error.svelte` error boundary pages for major routes; add ErrorBoundary component wrapper for complex sections

**Transaction Support for Trail Updates:**
- Problem: Trail creation/update can create orphaned waypoints/comments if partial failure occurs
- Blocks: Data consistency cannot be guaranteed; cannot safely retry failed operations
- Recommendation: Implement PocketBase collection hooks for cascade operations; use database transactions where available

**Search Index Synchronization:**
- Problem: Trail updates in backend don't automatically trigger Meilisearch re-index; search results become stale
- Blocks: Cannot rely on search results for real-time trail information
- Recommendation: Implement PocketBase hook to queue Meilisearch updates; implement eventual consistency guarantees; add staleness detection in UI

**Activity Log/Audit Trail:**
- Problem: No record of trail modifications, deletions, or federation activities; cannot audit user actions
- Blocks: Cannot investigate disputes, security incidents, or data loss
- Recommendation: Implement audit log collection; hook into PocketBase record events; implement log retention policy

## Test Coverage Gaps

**Trail Store Mutations:**
- What's not tested: Trail create/update/delete operations, cache invalidation, undo/redo stack behavior
- Files: `web/src/lib/stores/trail_store.ts`
- Risk: Regressions in store logic could cause silent data corruption or lost updates
- Priority: High

**Valhalla Routing Integration:**
- What's not tested: Route calculation, waypoint insertion/deletion, segment splitting, undo/redo
- Files: `web/src/lib/stores/valhalla_store.svelte.ts`, `web/src/lib/components/trail/route_editor.svelte`
- Risk: Complex route editing operations could produce invalid GPX; no validation of intermediate states
- Priority: High

**Integration Encryption/Decryption:**
- What's not tested: Secret encryption round-trip, key rotation, partial encryption scenarios
- Files: `db/hooks/integrations.go`, `db/integrations/komoot/komoot.go`, `db/integrations/strava/strava.go`
- Risk: Encrypted secrets could become unrecoverable; migration operations could leak plaintext
- Priority: High

**Federation Activity Posting:**
- What's not tested: Activity creation, remote follower notification, error handling for failed deliveries
- Files: `db/federation/create.go`, `db/federation/actor.go`, `db/routes/activitypub.go`
- Risk: Activities silently fail to deliver; followers see stale data; mentions not properly propagated
- Priority: Medium

**Meilisearch Token Management:**
- What's not tested: Token generation, expiration, staleness detection, refresh behavior
- Files: `web/src/hooks.server.ts`, `db/routes/search_token.go`
- Risk: Search becomes unavailable without user awareness; token leaks not detected
- Priority: Medium

**E2E Test Coverage:**
- What's not tested: Trail creation flow (only 3 e2e specs visible: user, list, index)
- Files: `web/tests/playwright/e2e/`
- Risk: Critical user journeys (trail import, editing, sharing) have no automated coverage
- Priority: Medium

---

*Concerns audit: 2026-06-10*
