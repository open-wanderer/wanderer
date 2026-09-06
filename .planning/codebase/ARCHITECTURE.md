<!-- refreshed: 2026-09-06 -->
# Architecture

**Analysis Date:** 2026-09-06

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Browser / Mobile Client                             │
│         SvelteKit Frontend (web/)      │       Flutter App (app/)            │
│  `web/src/routes/`, `web/src/lib/`    │  `app/lib/routes/`, `app/lib/`      │
└──────────────────┬──────────────────┬─────────────────┬─────────────────────┘
                   │                  │                 │
                   ▼                  ▼                 ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                         API Layer (SvelteKit Routes)                          │
│           `web/src/routes/api/v1/`  — Server-side handlers                   │
│  Validation (Zod), error handling, data transformation before response       │
└──────────────────────────────────┬───────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                   PocketBase Backend + Go Services (db/)                      │
│  `db/main.go`, `db/routes/`, `db/hooks/`, `db/federation/`, `db/services/`  │
│    Core: Collections, auth, CRUD; Custom: ActivityPub, plugins, search      │
└──────────────────────────────────┬───────────────────────────────────────────┘
                                    │
                   ┌────────────────┼────────────────┐
                   ▼                ▼                ▼
┌───────────────────────────┐  ┌──────────────────────────────────┐
│   PocketBase Database     │  │  Meilisearch (Full-text search)  │
│   SQLite with migrations  │  │  `db/util/meilisearch.go`        │
└───────────────────────────┘  └──────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| **Web Frontend (SvelteKit)** | UI rendering, user interactions, client-side state management | `web/src/routes/`, `web/src/lib/components/`, `web/src/lib/stores/` |
| **Page Load Logic** | Server-side data fetching, initial state hydration | `web/src/routes/+page.ts`, `web/src/routes/+layout.server.ts` |
| **Svelte Stores** | Reactive client state (user, trails, feed, search, etc.) | `web/src/lib/stores/` |
| **API Routes** | SvelteKit server endpoints, request validation, error handling | `web/src/routes/api/v1/` |
| **Models/Types** | Domain entity definitions (Trail, User, Comment, etc.) | `web/src/lib/models/` |
| **Components** | Reusable UI elements (TrailCard, Map, NavBar, etc.) | `web/src/lib/components/` |
| **Utilities** | Date, array, geospatial, validation helpers | `web/src/lib/util/` |
| **PocketBase Core** | Database, authentication, collections | `db/main.go` |
| **Routes (Go)** | Custom API endpoints beyond CRUD (ActivityPub, plugin system, regions) | `db/routes/` |
| **Hooks (Go)** | Event listeners on collection changes (indexing, federation fanout) | `db/hooks/` |
| **Federation** | ActivityPub actor/activity handling, federation to remote instances | `db/federation/` |
| **Services** | Specialized business logic (region management, trail merging) | `db/services/` |
| **Flutter App** | Mobile UI, offline-first trail recording, local sync | `app/lib/routes/`, `app/lib/lib/` |
| **Riverpod Providers** | State management and async data fetching for Flutter | `app/lib/provider/` |
| **Local Storage** | ObjectBox (NoSQL database), file-based caches | `app/lib/store/` |
| **Mobile Services** | GPS recording, tile proxy, background sync | `app/lib/services/` |
| **Documentation** | Astro-based site for user docs and API references | `docs/` |

## Pattern Overview

**Overall:** Layered MVC-inspired architecture with clear separation between presentation (web UI + mobile), application logic (API layer + stores), and domain logic (backend services).

**Key Characteristics:**
- **API-first design:** All data flows through RESTful endpoints; no direct database access from frontend
- **Type-safe across stack:** TypeScript (frontend), Go (backend), Dart (mobile) enforce contracts
- **Reactive stores:** Svelte stores and Riverpod providers manage client-side state; subscriptions trigger UI updates
- **ActivityPub federation:** Instance-level federation via standard ActivityPub types; all public content syncs bidirectionally
- **Offline-first mobile:** Flutter app uses ObjectBox for local persistence; sync on connectivity
- **Plugin system:** Backend supports dynamically loaded plugins for trail importing and category remapping
- **Meilisearch integration:** Full-text search indexed in real-time via hooks on collection changes

## Layers

**Presentation (Frontend):**
- Purpose: Render UI, handle user interactions, display cached state
- Location: `web/src/routes/`, `web/src/lib/components/`, `app/lib/routes/`
- Contains: Svelte components (`.svelte`), Flutter screens (`.dart`), layouts, pages
- Depends on: Stores, models, utilities
- Used by: Web browser, mobile app

**State Management:**
- Purpose: Hold and update reactive client state; coordinate data fetching
- Location: `web/src/lib/stores/` (Svelte), `app/lib/provider/` (Riverpod)
- Contains: Writable stores, async data fetching functions (e.g., `trails_index()`, `profile_fetch()`)
- Depends on: API utilities, models, PocketBase client
- Used by: Presentation layer components

**Models/Domain:**
- Purpose: Define data structures and types for all domain entities
- Location: `web/src/lib/models/`, `app/lib/entities/`, `app/lib/models/`
- Contains: Classes (Trail, User, Waypoint, Comment), TypeScript interfaces, Dart freezed classes
- Depends on: Child models (e.g., Trail depends on Waypoint, Category)
- Used by: Stores, components, API routes, backend

**API Routes (SvelteKit):**
- Purpose: Bridge between frontend requests and backend; validate input, transform output
- Location: `web/src/routes/api/v1/`
- Contains: `+server.ts` GET/POST/PUT/DELETE handlers with Zod validation
- Depends on: PocketBase client, schemas, utilities, server utilities
- Used by: Frontend stores, browser fetch requests

**Backend Services (Go):**
- Purpose: Core business logic, integrations, federation, caching
- Location: `db/main.go`, `db/routes/`, `db/hooks/`, `db/services/`, `db/federation/`
- Contains: Custom API endpoints, event hooks, ActivityPub handlers, region/search services
- Depends on: PocketBase core, Meilisearch, external APIs (Nominatim, Overpass, Valhalla)
- Used by: SvelteKit API routes, PocketBase CRUD operations, scheduled tasks

**Data Persistence:**
- Purpose: Data storage and retrieval
- Location: `db/pb_data/` (PocketBase SQLite runtime), `db/migrations/` (schema)
- Contains: Collections (trails, users, comments, activitypub_actors), indexes, foreign keys
- Depends on: Migration files for schema evolution
- Used by: Backend routes, stores, hooks

**Local Storage (Mobile):**
- Purpose: Offline data persistence and caching on mobile
- Location: `app/lib/store/`, ObjectBox database files (app-managed)
- Contains: Trail data, photos, region tiles, session data
- Depends on: ObjectBox ORM, file system
- Used by: Flutter screens, providers

## Data Flow

### Primary Request Path (Trail Display)

1. **Browser navigation:** User navigates to `/trail/[id]` → SvelteKit routing
2. **Page load:** `web/src/routes/trail/[id]/+page.ts` calls `load()` function
3. **Fetch data:** Store function (e.g., `trails_show()` in `web/src/lib/stores/trail_store.ts`) calls fetch
4. **SvelteKit API:** Fetch hits `web/src/routes/api/v1/trail/[id]/+server.ts` (GET handler)
5. **Validation:** Request parameters validated via Zod schema (`RecordIdSchema`)
6. **Backend call:** API route calls `event.locals.pb.collection('trails').getOne(id, {...expand})`
7. **Hook execution:** PocketBase triggers `OnRecordAfterReadSuccess` hook in `db/hooks/trails.go`
8. **Response:** Trail object with expanded waypoints, comments, author returned to frontend
9. **Store update:** `trail_store.ts` sets `currentTrail` writable store; UI subscribes via `$currentTrail`
10. **Rendering:** Svelte reactivity renders trail data in `trail/[id]/+page.svelte`

### Feed/Recommendations Flow

1. **Page load:** `web/src/routes/+page.ts` calls `feed_index()` and `trails_recommend()`
2. **Store calls:** `feed_store.ts` and `trail_store.ts` fetch from API
3. **API aggregation:** `web/src/routes/api/v1/feed/+server.ts` calls PocketBase and filters by user preferences
4. **Meilisearch:** Recommendation endpoint may query Meilisearch for personalized results
5. **Response:** List of trails + feed items returned
6. **Client store:** Stores update `feed` and `recommendedTrails` writable stores
7. **Reactive display:** Components subscribe and render

### ActivityPub Federation Flow

1. **Create activity:** Local user creates trail → `web/src/routes/api/v1/trail/+server.ts` PUT handler
2. **Backend processing:** PocketBase hook in `db/hooks/trails.go` fires on record creation
3. **Fanout:** `db/federation/create.go` creates ActivityPub `Create` activity
4. **Actor lookup:** Instance actor found in `activitypub_actors` collection
5. **Follower delivery:** `db/federation/activity.go` queues delivery to followers (other instances)
6. **HTTP signature:** Go signs outbound activity with private key (`db/util/activitypub.go`)
7. **Inbox delivery:** Remote instance receives activity at `/api/v1/activitypub/user/{handle}/inbox`
8. **Sync verification:** Signature verified; activity processed; local trail created with original actor reference

### Search Flow

1. **User enters query:** Search box in `web/src/lib/components/search.svelte` fires
2. **Store update:** `search_store.ts` calls `trails_search(query)`
3. **API call:** `web/src/routes/api/v1/search/+server.ts` receives query
4. **Backend search:** Route calls Meilisearch client: `client.Index("trails").Search(query, opts)`
5. **Meilisearch index:** Full-text index populated via hook on trail creation (`db/hooks/trails.go`)
6. **Results:** Matching trail IDs returned; backend fetches full trail objects from PocketBase
7. **Response:** Array of trails returned to frontend
8. **Store subscription:** Components subscribe to search results store and render

**State Management Approach:**
- **Frontend client state:** Svelte writable stores hold current user, UI state, loaded trails
- **Server state:** Session/auth via `event.locals.pb.authStore.record` (PocketBase built-in)
- **Reactive updates:** Components subscribe to stores via `$store` syntax; mutations trigger subscribers
- **Mobile state:** Riverpod providers (async state and caching), local ObjectBox for persistence

## Key Abstractions

**Trail:**
- Purpose: Represents a recorded/planned hiking path with GPS data, metadata, social features
- Examples: `web/src/lib/models/trail.ts`, `app/lib/entities/trail_entity.dart`
- Pattern: Class-based model with optional `expand` field for related data (waypoints, comments, shares)
- Fields: `name`, `location`, `date`, `public`, `distance`, `elevation_*`, `author`, `expand?`

**Store Functions (Svelte & Riverpod):**
- Purpose: Encapsulate data fetching logic with error handling, caching, parsing
- Examples: `trails_index()`, `profile_fetch()`, `summit_logs_create()` in `web/src/lib/stores/`
- Pattern: Async functions that fetch from API, update writable stores, return parsed models
- Convention: Naming format `{entity}_{operation}` (e.g., `users_create`, `trails_update`)

**Waypoint:**
- Purpose: Point along a trail with location, metadata, optional photos
- Examples: `web/src/lib/models/waypoint.ts`, `app/lib/entities/waypoint_entity.dart`
- Pattern: Ordered by `distance_from_start` along trail; immutable data class

**ActivityPub Actor:**
- Purpose: Represents an entity (user or instance) in federation; holds inbox, followers collection
- Examples: `web/src/lib/models/activitypub/actor.ts`, `db/federation/actor.go`
- Pattern: Standard ActivityPub types; instance actor type is `Application`

**PocketBase Client:**
- Purpose: Singleton for backend communication; auth, CRUD operations
- Location: `web/src/lib/pocketbase.ts`, `app/lib/provider/api_provider.dart`
- Pattern: Lazy singleton accessed via `getPb()` (web) or `apiProvider` (app); browser-only on web

**API Utilities:**
- Purpose: Shared HTTP logic, error handling, response parsing
- Location: `web/src/lib/util/api_util.ts`, `app/lib/services/` (Dio-based)
- Pattern: Generic functions (`list<T>()`, `create<T>()`, `handleError()`) for CRUD operations
- Convention: All functions return typed results via `Zod` schemas or Dart models

## Entry Points

**Web Frontend:**
- Location: `web/src/routes/+layout.svelte`
- Triggers: Browser navigation to `/`
- Responsibilities: Root layout, navigation bar, theme switching, authentication guard, plugin loading
- Called from: SvelteKit runtime

**Home Page:**
- Location: `web/src/routes/+page.svelte` (component) and `web/src/routes/+page.ts` (load)
- Triggers: Navigation to `/`
- Responsibilities: Load feed, recommended trails; display hero and trail cards
- Dependencies: `feed_store`, `trail_store`, category/subcategory stores

**Map Page:**
- Location: `web/src/routes/map/+page.svelte`
- Triggers: Navigation to `/map`
- Responsibilities: Render interactive map, search trails by bounding box, apply filters
- Dependencies: MapLibre GL JS, geospatial utilities

**Trail Detail:**
- Location: `web/src/routes/trail/[id]/+page.svelte`
- Triggers: Navigation to `/trail/[id]`
- Responsibilities: Display trail details, waypoints, elevation profile, comments, social actions

**Trail Edit/Create:**
- Location: `web/src/routes/trail/edit/[id]/+page.svelte`
- Triggers: Navigation to `/trail/edit/[id]` or `/trail/create`
- Responsibilities: Form for creating/editing trails, file uploads (GPX/photos), validating input

**Backend (Go):**
- Location: `db/main.go`
- Triggers: HTTP requests to `/api/v1/*`, PocketBase initialization
- Responsibilities: Initialize PocketBase, register routes/hooks, set up Meilisearch, start server
- Dependencies: PocketBase core, migrations, routes, hooks

**Mobile App (Flutter):**
- Location: `app/lib/main.dart`
- Triggers: App launch
- Responsibilities: Initialize providers (Riverpod), set up navigation (GoRouter), load user session
- Dependencies: `router_provider`, `objectBoxProvider`, `authProvider`

**Mobile Home:**
- Location: `app/lib/routes/home_screen.dart`
- Triggers: App launch (after auth)
- Responsibilities: Display feed, offline sync status, quick access to trails
- Dependencies: Trail providers, feed providers

**Mobile Map:**
- Location: `app/lib/routes/map_screen.dart`
- Triggers: Navigation to map tab
- Responsibilities: Render offline/online map, trail display, location services
- Dependencies: MapLibre, tile proxy, GPS

## Architectural Constraints

- **Threading:** SvelteKit runs in Node.js event loop (single-threaded); Go backend may spawn goroutines for concurrent operations (Meilisearch indexing, federation fanout). Flutter uses Dart isolates for heavy computation (GPX parsing).
- **Global state:** PocketBase instance stored in `event.locals.pb` (per-request in SvelteKit); Flutter uses Riverpod for app-level providers; stores initialized once at app startup
- **Circular imports:** Models may circularly reference (e.g., Trail ↔ Comment); patterns avoid by using `expand` field for lazy loading instead of eager loading; type-only imports used
- **Authentication:** Session-based via PocketBase AuthStore (JWT tokens); frontend checks user via `$currentUser` store; backend verifies token on protected routes
- **Real-time:** PocketBase subscriptions available (e.g., `/api/v1/realtime`) but not heavily used in current codebase; polling used instead for feed updates
- **Privacy:** `is_public = false` records must never be included in outgoing activities; checked at fanout time in `db/federation/` before delivery to remote instances
- **No offline sync buffer:** If a remote instance is unreachable during federation, the activity is dropped (existing behavior for user federation); no retry queue
- **Data normalization:** Stores denormalize for performance (e.g., like count stored on trail record); updates via hooks ensure consistency

## Anti-Patterns

### Direct PocketBase Access in Components

**What happens:** Components call `getPb().collection().getList()` directly instead of using store functions

**Why it's wrong:** Duplicates data-fetching logic, makes caching/invalidation harder, couples UI to API details, breaks reactive updates

**Do this instead:** Create store function in `web/src/lib/stores/` that fetches and returns typed data. Components subscribe to store via `$store` syntax. Example in `web/src/lib/stores/trail_store.ts` — `trails_index()` wraps the fetch and updates `trailsIndex` writable store.

### Missing Error Boundaries

**What happens:** API errors silently fail; components render empty/partial state without indication

**Why it's wrong:** Users don't know if data failed to load or is genuinely empty; poor UX; debugging difficult

**Do this instead:** Wrap store functions in try/catch; return fallback data with `error` flag. Components check for errors and render error UI. Example: `api_util.ts` `handleError()` function catches and formats errors for consistent response.

### Tightly Coupled Store Logic

**What happens:** Store functions call other stores (e.g., `trails_index()` calls `categories_index()` directly)

**Why it's wrong:** Makes testing difficult; unclear dependency graph; unnecessary coupling causes cascading updates

**Do this instead:** Load dependencies separately or pass as parameters. Example: `web/src/routes/+page.ts` calls both `categories_index()` and `trails_index()` in parallel; doesn't nest calls inside store functions.

## Error Handling

- **API calls:** Wrap in try/catch; throw structured error (status code + message) on non-2xx status
- **Store functions:** Catch errors; return empty arrays/defaults rather than throwing (prevents UI crash)
- **Components:** Guard against missing data with optional chaining (e.g., `trail?.expand?.waypoints ?? []`)
- **Backend:** Return JSON error objects with status code and detail message; PocketBase built-in error handling for validation
- **Mobile:** Riverpod AsyncValue allows `hasError` state; screens render error UI when `isLoading == false && hasError == true`

## Cross-Cutting Concerns

**Logging:**
- Frontend: `console.log()`, `console.warn()`, `console.error()` for development; errors logged via error boundaries
- Backend: Go logger in `db/main.go`; structured logs with context

**Validation:**
- Frontend: Zod schemas for input validation (e.g., `TrailCreateSchema` in `web/src/lib/models/api/`)
- Backend: PocketBase collection rules and custom validation hooks; Zod for Go structs

**Authentication:**
- PocketBase built-in auth; session tokens stored in browser storage (web) or ObjectBox (mobile)
- Protected routes guarded in `web/src/routes/+layout.svelte` via `isRouteProtected()` utility
- API endpoints check `event.locals.pb.authStore.record` for authorization; return 401 if missing

**Internationalization:**
- Frontend: `svelte-i18n` package; translation files in `web/src/lib/i18n/locales/`
- Mobile: Flutter localization in `app/lib/i18n/`; supported languages configured in `app/pubspec.yaml`

---

*Architecture analysis: 2026-09-06*
