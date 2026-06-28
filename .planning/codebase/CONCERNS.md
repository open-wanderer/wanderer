# Codebase Concerns

**Analysis Date:** 2026-06-07

## Tech Debt

### FIT Parser Library - Incomplete CRC Validation

**Issue:** Critical FIT file integrity checks are disabled in the parser
**Files:** 
- `web/src/lib/vendor/fit-parser/fit_parser.ts` (lines 106-110, 127-130)
- `web/src/lib/vendor/fit-parser/binary.ts` (line 419)

**Impact:** FIT files (Garmin activity files) imported into the system may have corrupted data without detection. The parser silently skips CRC validation unless forced, meaning invalid file data can be processed and saved to the database. This could lead to inaccurate elevation, distance, and timing information in trail data.

**Current Situation:** 
- Header CRC check is commented out with `TODO: fix Header CRC check`
- File CRC check is commented out with `TODO: fix File CRC check`
- Both checks have a `force` flag that continues processing even when validation fails

**Fix Approach:**
1. Implement proper CRC-16 calculation and validation in `fit_parser.ts`
2. Add error logging for CRC mismatches before deciding whether to continue
3. Provide user feedback when FIT files fail validation (suggest re-export from Garmin)
4. Consider adding a setting to allow/disallow forcing past CRC errors for power users

### FIT Parser - Compressed Header Support Missing

**Issue:** Compressed timestamp headers in FIT files are not implemented
**Files:** `web/src/lib/vendor/fit-parser/binary.ts` (line 419)

**Impact:** FIT files with compressed timestamps may be parsed incorrectly or lose timing information. This is a less common case but could affect recent Garmin devices.

**Current Situation:** Line 419 has `TODO: handle compressed header ((recordHeader & 128) == 128)`

**Fix Approach:**
1. Implement compressed timestamp decompression in `readRecord()`
2. Add test coverage for FIT files with compressed headers
3. Document which Garmin devices use compressed headers

### FIT Parser - Global State Management

**Issue:** The FIT parser uses module-level global variables to track state
**Files:** `web/src/lib/vendor/fit-parser/binary.ts` (line 12: `let monitoring_timestamp = 0`)

**Impact:** Concurrent FIT file parsing could produce incorrect results. If two users upload FIT files simultaneously, the `monitoring_timestamp` variable could be corrupted across both parses, leading to incorrect timestamp calculations.

**Current Situation:** Comment at line 493 acknowledges: `// TODO weirdly uses global variables?`

**Fix Approach:**
1. Move `monitoring_timestamp` into a parser instance context object
2. Pass context through `readRecord()` call chain
3. Add concurrency tests to verify thread safety

### FIT Parser - Type System Incomplete

**Issue:** FIT message output types are incomplete and use `any` extensively
**Files:** `web/src/lib/vendor/fit-parser/fit.ts` (line 47-50)

**Impact:** Type safety is lost during FIT parsing. Invalid field access patterns won't be caught at compile time, leading to potential runtime errors when processing activity data.

**Current Situation:** 
- Comment acknowledges: `// TODO we dont have comprehensive output types bc its a bit cumbersome to maintain typescript types`
- Suggests using Zod for inference but never implemented

**Fix Approach:**
1. Generate TypeScript types from FIT specification using code generation
2. Implement Zod schemas for validation and runtime type checking
3. Create validation on parsed FIT output before saving to database

### Large Component Files - Maintenance Burden

**Issue:** Multiple components exceed 1000+ lines, combining logic, UI, and state management
**Files:**
- `web/src/routes/trail/edit/[id]/+page.svelte` (1,867 lines)
- `web/src/lib/util/icon_util.ts` (1,541 lines)
- `web/src/lib/models/api/openapi_schemas.ts` (1,364 lines)
- `web/src/lib/vendor/maplibre-elevation-profile/elevationprofile.ts` (1,069 lines)
- `web/src/lib/components/trail/map_with_elevation_maplibre.svelte` (1,056 lines)

**Impact:** Difficult to test, understand, and maintain. Changes carry high risk of unintended side effects. Onboarding new developers is harder.

**Current Situation:** Trail edit page handles route editing, photo management, summit log editing, waypoint management, GPX import, and API calls all in one component.

**Fix Approach:**
1. Break `trail/edit/[id]/+page.svelte` into:
   - Route editing sub-component
   - Photo picker sub-component
   - Waypoint management sub-component
   - Summit log management sub-component
   - Summary/info panel sub-component
2. Extract business logic from components into composable stores/services
3. Aim for components under 500 lines
4. Add integration tests for complex workflows

### Trail Store - Module-Level State Management

**Issue:** Trail store uses module-level arrays and state variables that could be mutated unexpectedly
**Files:** `web/src/lib/stores/trail_store.ts` (line 17: `let trails: Trail[] = []`)

**Impact:** Global mutable state can cause bugs when trails are fetched from multiple sources (search, index, show). The `trails` array persists across page navigations and could cause stale data issues.

**Current Situation:** The `trails` variable is updated inconsistently:
- Line 37: Overwritten by `trails_index()`
- Line 124: Merged by `trails_search_bounding_box()` with conditional logic
- Exposed as direct module state without clear ownership

**Fix Approach:**
1. Encapsulate trail list state in a Svelte store with clear update semantics
2. Distinguish between different trail queries (index vs search vs bounding box)
3. Add state invalidation logic when switching between views
4. Consider using a state machine to manage fetch states (loading, error, success)

### Valhalla Store - Undo/Redo Without Persistence

**Issue:** Route editing undo/redo stack is lost on page navigation or refresh
**Files:** `web/src/lib/stores/valhalla_store.svelte.ts` (lines 19-20)

**Impact:** Users lose their edit history if they accidentally refresh or navigate away. The stacks are reset by `clearUndoRedoStack()` but there's no warning to users.

**Current Situation:** Undo/redo stacks are in-memory only

**Fix Approach:**
1. Add sessionStorage-based persistence for undo/redo stacks
2. Add navigation guards to warn users if they have unsaved undo states
3. Provide "Save as draft" option to persist route edits

---

## Known Bugs

### Strava Integration - Multiple Passes Required

**Issue:** Strava activity synchronization sometimes doesn't complete in a single run
**Files:** Database/backend integration (referenced in CHANGELOG.md v0.18.2)

**Trigger:** Large number of Strava activities from user history

**Workaround:** Run Strava sync multiple times; it will eventually complete. However, this is inefficient and confusing for users.

**Fix Status:** Partially addressed in v0.18.2 with "significant performance improvements"

### MapLibre Layer Manager - Visibility Tracking Issues

**Issue:** Map layers may not be tracked correctly after data updates, preventing proper re-rendering
**Files:** Likely in `web/src/lib/vendor/maplibre-layer-manager/maplibre-layer-manager.ts`

**Trigger:** Rapid map updates or switching between different trail views

**Workaround:** Force map refresh by zooming or moving

**Fix Status:** Fixed in v0.19.0 (PR #960) but could still have edge cases

### GPX Upload - Duplicate Processing

**Issue:** Double GPX upload when creating a trail with a GPX file
**Files:** `web/src/lib/stores/trail_store.ts` (line 420+: `trails_create()`)

**Trigger:** Creating a trail with GPX file attachment

**Impact:** Two copies of the same trail data uploaded, causing duplicates in the database

**Fix Status:** Fixed in v0.19.0 (PR #969) - verify this was properly deployed

### List Save - Async State Race Condition

**Issue:** Lists fail to save due to async state issues
**Files:** `web/src/lib/stores/list_store.ts`

**Trigger:** Bulk assignment of trails to lists

**Symptoms:** Modal doesn't respond, list doesn't update

**Fix Status:** Fixed in v0.18.5 (PR #816) but async patterns throughout codebase could reintroduce similar bugs

---

## Security Considerations

### HTML Content Sanitization

**Risk:** Cross-site scripting (XSS) attacks through user-generated content
**Files:** 
- `web/src/lib/components/base/editor.svelte` (handles rich text)
- API endpoints for: descriptions, comments, summit logs, waypoints, profile bios

**Current Mitigation:** HTML content is sanitized on the server-side (v0.19.0, PR #930)

**Recommendations:**
1. Verify `@xmldom/xmldom` version is current and patched (CVE-2022-39299 fixed in v0.18.5)
2. Add Content Security Policy (CSP) headers
3. Add server-side HTML sanitization validation tests
4. Sanitize on both client and server (defense in depth)

### Anonymous API Access Removed

**Risk:** Previously allowed unauthenticated access to user data
**Files:** Backend API routes

**Current Mitigation:** Anonymous user API endpoints removed in v0.19.0 (PR #927)

**Recommendations:**
1. Audit remaining public endpoints to ensure they're intentionally public
2. Add rate limiting to prevent enumeration attacks

### ActivityPub/Federation Security

**Risk:** CSRF/SSRF attacks through outbound network calls and federation
**Files:** `db/federation/` directory and integration-related code

**Current Mitigation:** Additional CSRF/SSRF protections and rate limiting added in v0.19.0 (PR #930)

**Recommendations:**
1. Audit ActivityPub inbound handler for malicious payloads
2. Implement timeout and size limits on remote content fetches
3. Test federation with adversarial payloads

### Private Profile Bypass Risk

**Risk:** Visibility settings and sharing rules might not be consistently applied
**Files:** Profile access logic, remote trail/list access

**Current Mitigation:** Privacy fixes in v0.19.1 (PR #980, #986)

**Recommendations:**
1. Add comprehensive test suite for permission checks across all endpoints
2. Document which endpoints are public, authenticated, or require sharing
3. Add logging for permission denials to detect attacks

---

## Performance Bottlenecks

### Trail Search - Large Bounding Box Queries

**Issue:** `trails_search_bounding_box()` retrieves up to 500 trails per page without pagination limits
**Files:** `web/src/lib/stores/trail_store.ts` (line 116: `hitsPerPage: 500`)

**Impact:** Memory usage and rendering performance degrade when viewing areas with thousands of trails. Meilisearch returns 500 results, client parses polylines, creates GeoJSON, and renders markers.

**Current Situation:** Hard-coded limit of 500 hits per page on map view

**Improvement Path:**
1. Implement viewport-based clustering at Meilisearch query level
2. Use tile-based approach (quadtree or H3) instead of bounding box
3. Add server-side polyline simplification based on zoom level
4. Cache geohash-based cluster results

### FIT Parser - No Streaming

**Issue:** Entire FIT file loaded into memory before parsing
**Files:** `web/src/lib/vendor/fit-parser/fit_parser.ts` (line 73: `new Uint8Array(getArrayBuffer(content))`)

**Impact:** Large activity files (multi-hour hikes with high-frequency GPS) consume significant memory. Parsing is synchronous and blocks UI.

**Current Situation:** Binary is fully buffered and parsed synchronously

**Improvement Path:**
1. Implement streaming parser for large files
2. Move parsing to Web Worker
3. Add progress callback for UI updates
4. Implement chunked processing for memory efficiency

### Map Rendering - All Layers Rendered Immediately

**Issue:** Map renders all trail polylines, elevation profiles, markers at once
**Files:** `web/src/lib/components/trail/map_with_elevation_maplibre.svelte`

**Impact:** Slow initial load when viewing search results with many trails. Browser spends time rendering invisible/offscreen features.

**Improvement Path:**
1. Implement viewport culling for polylines
2. Use vector tiles for better performance at scale
3. Defer elevation profile rendering until requested
4. Implement tile-based cluster rendering

### Search Results - Eager Trail Object Creation

**Issue:** `searchResultToTrailList()` creates Trail objects from Meilisearch results synchronously
**Files:** `web/src/lib/stores/trail_store.ts` (line 90)

**Impact:** Hundreds of Trail objects instantiated at once when user searches popular areas.

**Improvement Path:**
1. Lazy-load Trail objects on demand
2. Use virtual scrolling for result list
3. Defer creation until trail is visible in viewport

---

## Fragile Areas

### Route Editing State Machine - Complex Undo/Redo

**Files:** `web/src/lib/stores/valhalla_store.svelte.ts`
**Why Fragile:** 
- Uses `json-diff-ts` with custom Changeset objects
- Multiple operations push to undo stack (insert, edit, delete, calculate route, etc.)
- Stack operations not tested
- Can enter invalid states if exceptions occur mid-operation

**Safe Modification:**
1. Always wrap state changes in try-catch
2. Call `pushToUndoStack()` at end of operation, not beginning
3. Add invariant checks: validate route structure after each operation
4. Test with complex sequences: insert → calculate → delete → undo → redo

**Test Coverage Gaps:**
- No unit tests for undo/redo
- No integration tests for route calculation + undo
- No tests for concurrent edits

### Waypoint Merge Logic - Geohash Calculation

**Files:** `web/src/lib/models/waypoint.ts`, `db/migrations/` (waypoint merge migrations)
**Why Fragile:**
- Merge radius configuration affects which waypoints are merged
- Category-specific configuration not consistently applied
- No validation of merge results

**Safe Modification:**
1. Verify merge radius configuration is loaded before merging
2. Test merge with edge cases: waypoints on category boundary, altitude differences
3. Validate that merged waypoints don't create loops in trail data

**Test Coverage Gaps:**
- No automated tests for waypoint merge algorithm
- No tests for multi-category scenarios

### Search State - No Cleanup on Navigation

**Files:** `web/src/lib/stores/search_store.ts`
**Why Fragile:**
- Search results cached in store without clear invalidation
- Switching between trail search and location search could mix results
- Component unmounting doesn't cancel pending requests

**Safe Modification:**
1. Cancel pending fetch requests on component destroy
2. Clear search state on route change
3. Add request deduplication to prevent concurrent identical searches

**Test Coverage Gaps:**
- No tests for request cancellation
- No tests for rapid navigation between search types

### Trail Merge API - Multi-summit Log Consolidation

**Files:** `web/src/lib/stores/trail_merge_api.ts`, `web/src/lib/stores/trail_merge_store.svelte.ts`
**Why Fragile:**
- Merges multiple trails into one with multiple summit logs
- Complex API call sequence
- No validation of merged data integrity
- Database constraints could fail mid-merge

**Safe Modification:**
1. Verify all trails have valid data before starting merge
2. Test merge with trails having conflicting metadata
3. Implement rollback on partial failure

**Test Coverage Gaps:**
- No automated tests for trail merge
- No tests for edge cases: empty summit logs, conflicting dates

---

## Scaling Limits

### Meilisearch Index Size

**Current Capacity:** System designed for ~100K trails per instance
**Limit:** Meilisearch memory usage grows linearly with indexed content
**Symptoms When Exceeded:** 
- Search latency increases
- Index rebuild takes hours
- Memory usage exceeds available RAM

**Scaling Path:**
1. Implement sharded Meilisearch deployment
2. Archive old/inactive trails to separate index
3. Implement index rotation/rolling
4. Consider alternative search engines (Elasticsearch, Typesense)

### PocketBase Database

**Current Capacity:** SQLite design limit, ~10-50GB practical limit
**Limit:** Single SQLite database file can't be distributed
**Symptoms When Exceeded:**
- Database locks during bulk operations
- Write conflicts on concurrent uploads
- Backup/restore times become prohibitive

**Scaling Path:**
1. Migrate to PostgreSQL for multi-instance deployment
2. Implement read replicas for search operations
3. Use connection pooling
4. Archive historical data to separate database

### File Storage

**Current Capacity:** Depends on deployment (local filesystem)
**Limit:** Photo storage grows with user uploads
**Symptoms When Exceeded:**
- Slow file serving
- Disk space exhaustion
- Backup times prohibitive

**Scaling Path:**
1. Integrate S3-compatible storage (MinIO, AWS S3)
2. Implement image optimization/compression on upload
3. Archive old photos to cold storage
4. Use CDN for photo delivery

---

## Dependencies at Risk

### `@xmldom/xmldom` - Inherited XSS Risk

**Package:** `@xmldom/xmldom@^0.8.12` (used transitively)
**Risk:** CVE-2022-39299 patched in v0.8.8+, but major update might have breaking changes
**Impact:** If vulnerable version is installed, XSS possible through XML processing
**Status:** Known to be patched (v0.18.5 changelog mentions fix)
**Migration Plan:** 
1. Pin version to `^0.8.12`
2. Monitor for new CVEs in XML parsing libraries
3. Consider moving off XML parsing if possible

### `three.js` - Large Dependency Tree

**Package:** `three@^0.183.1` with `@threlte/*` wrappers
**Risk:** Large bundle size, frequent updates
**Impact:** Longer build times, larger JS bundles, potential breaking changes
**Status:** Currently pinned to specific version, appears stable
**Migration Plan:**
1. Monitor three.js releases for breaking changes
2. Consider using three.js CDN for production to reduce build time
3. Implement lazy loading for 3D features (mountain.svelte is 879 lines)

### `pocketbase` - Single SDK Version Lock

**Package:** `pocketbase@^0.26.8` (JavaScript SDK)
**Risk:** SDK version must match server version
**Impact:** Upgrade requires synchronized client/server updates
**Status:** This is expected behavior but limits deployment flexibility
**Migration Plan:**
1. Document server/client version requirements clearly
2. Add version compatibility matrix to deployment docs
3. Consider implementing API versioning to allow skew

---

## Missing Critical Features

### Error Boundary Implementation

**Problem:** No global error handling for uncaught exceptions
**Blocks:** Graceful error recovery, preventing white screen of death
**Impact:** Users experience full app crash instead of error message

**Recommendation:** 
1. Implement SvelteKit error page in `src/routes/+error.svelte`
2. Add error reporting/logging (Sentry integration)
3. Implement component-level error boundaries

### Request Deduplication

**Problem:** Concurrent identical requests are not deduped
**Blocks:** Reducing unnecessary API calls, race conditions in search
**Impact:** Wasted bandwidth, potential race conditions

**Recommendation:**
1. Implement request deduplication in API utility layer
2. Use AbortController for request cancellation
3. Cache search results with TTL

### Offline Support

**Problem:** No offline-first architecture or service worker caching
**Blocks:** Using app without internet connectivity
**Impact:** App is unusable when offline (even viewing cached data)

**Recommendation:**
1. Implement service worker with cache-first strategy for static assets
2. Store critical data locally (IndexedDB)
3. Queue mutations for sync when online

---

## Test Coverage Gaps

### FIT Parser - No Unit Tests

**What's Not Tested:**
- CRC validation logic (when implemented)
- Compressed header handling (when implemented)
- All FIT message types parsing
- Round-trip: parse FIT → modify → export

**Files:** `web/src/lib/vendor/fit-parser/*`

**Risk:** High - imports are critical for activity data integrity
**Priority:** High

### Route Editing - No Tests for Complex Scenarios

**What's Not Tested:**
- Undo/redo with multiple operations
- Route calculation failure handling
- Concurrent anchor additions
- Route with self-intersections

**Files:** `web/src/lib/stores/valhalla_store.svelte.ts`, `web/src/routes/trail/edit/[id]/+page.svelte`

**Risk:** High - data loss or corruption possible
**Priority:** High

### Search - No Integration Tests

**What's Not Tested:**
- Trail search with filters
- Location search with geocoding
- Multi-search coordination
- Search result pagination
- Canceled requests cleanup

**Files:** `web/src/lib/stores/search_store.ts`, `web/src/lib/stores/trail_store.ts`

**Risk:** Medium - UX degradation
**Priority:** Medium

### API Error Handling - No Negative Tests

**What's Not Tested:**
- 4xx/5xx error responses
- Network timeouts
- Invalid API responses
- Rate limiting responses

**Files:** All files using fetch/API calls

**Risk:** Medium - poor error messages to users
**Priority:** Medium

### Component Props Validation - No Tests

**What's Not Tested:**
- Invalid props combinations
- Missing required props
- Type mismatches

**Files:** All component files, especially large ones

**Risk:** Low - TypeScript catches most issues
**Priority:** Low

### Permission Checks - No Comprehensive Suite

**What's Not Tested:**
- Public/private trail access
- List sharing rules
- Federation visibility
- Remote content access

**Files:** Backend API routes, trail/list access logic

**Risk:** High - security issue if missed
**Priority:** High

---

## Code Quality Issues

### Mixed Import Styles

**Issue:** Some files use ES6 `import`, others use `require()`
**Files:** Scattered throughout, especially in vendor code
**Recommendation:** Standardize on ES6 imports (already configured in `package.json` with `"type": "module"`)

### Inconsistent Error Handling

**Issue:** Some async functions throw errors, others return them
**Files:** Various stores
**Recommendation:** Standardize on throwing errors, use try-catch in callers

### Magic Numbers

**Issue:** Hard-coded values appear throughout (500 for search hits, timeouts, merge radius)
**Files:** Multiple
**Recommendation:** Extract to constants file (e.g., `src/lib/config/constants.ts`)

### Commented-Out Code

**Issue:** Debug code and commented-out features left in repository
**Files:** `web/vite.config.ts` (commented HTTPS config), `web/playwright.config.ts` (commented browsers)
**Recommendation:** Delete commented code or move to separate feature branch

---

*Concerns audit: 2026-06-07*
