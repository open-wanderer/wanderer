<!-- refreshed: 2026-06-07 -->
# Architecture

**Analysis Date:** 2026-06-07

## System Overview

Wanderer is a multi-platform trail management and discovery system with three main components: a web frontend (SvelteKit), a Flutter mobile app, and a backend API (Go/PocketBase).

```text
┌────────────────────────────────────────────────────────────────────┐
│                        Client Layer                                 │
├──────────────────────┬──────────────────────┬──────────────────────┤
│   Web Frontend       │   Mobile App         │   API Clients        │
│   (SvelteKit/TS)     │   (Flutter/Dart)     │   (Browser/HTTP)     │
│ `web/src/routes`     │  `app/lib`           │ `web/src/lib`        │
└───────────┬──────────┴──────────┬───────────┴──────────┬───────────┘
            │                     │                      │
            ▼                     ▼                      ▼
┌────────────────────────────────────────────────────────────────────┐
│                    API & Server Layer                              │
├──────────────────────────────────────────────────────────────────┤
│ SvelteKit API Routes   │  Backend Services  │  Federation (ActivityPub) │
│ `web/src/routes/api`   │  `db/*`            │  `db/federation`         │
└─────────────┬──────────┴──────────┬─────────┴──────────┬──────────┘
              │                     │                    │
              ▼                     ▼                    ▼
┌────────────────────────────────────────────────────────────────────┐
│                 Data Layer (PocketBase + Go Backend)                │
│                    `db/main.go`, migrations                         │
│  - Collections: trails, waypoints, users, comments, etc.            │
│  - External APIs: Valhalla, Nominatim, Overpass, Meilisearch       │
└─────────────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| **Web Frontend** | UI rendering, user interactions, client-side state | `web/src/routes/` |
| **Page Load Logic** | Data fetching, server-side initialization | `web/src/routes/+page.ts`, `+layout.server.ts` |
| **Stores** | Svelte stores managing client/shared state | `web/src/lib/stores/` |
| **Models** | Type definitions for domain entities | `web/src/lib/models/` |
| **Components** | Reusable UI elements | `web/src/lib/components/` |
| **API Routes** | SvelteKit server endpoints (proxy/processing) | `web/src/routes/api/v1/` |
| **Backend (Go)** | Core business logic, database, federation | `db/main.go`, `db/routes/` |
| **Mobile App** | Flutter application for mobile platforms | `app/lib/` |
| **Federation** | ActivityPub implementation for social features | `db/federation/` |
| **Integrations** | Third-party service integrations (Strava, etc.) | `db/integrations/` |

## Pattern Overview

**Overall:** Multi-tier client-server architecture with Svelte frontend communicating to Go backend via HTTP/REST API. Uses PocketBase (Dart/Go ORM) for database abstraction and real-time capabilities. Mobile app is standalone Flutter client. Federation layer implements ActivityPub for decentralized social features.

**Key Characteristics:**
- **Separation of concerns:** Clear split between client (SvelteKit), API layer (SvelteKit routes), and backend (Go)
- **Store-driven state management:** Reactive Svelte stores coordinate data fetching and UI updates
- **API-first design:** All data flows through RESTful endpoints (PocketBase or custom routes)
- **Type-safe:** TypeScript on frontend, Golang on backend with type generation from schemas

## Layers

**Client Layer (Web - SvelteKit):**
- Purpose: Render UI, handle user interactions, manage client state
- Location: `web/src/routes/`, `web/src/lib/components/`
- Contains: `.svelte` components, route structures, page layouts
- Depends on: Stores, models, utilities
- Used by: Browser/HTTP clients

**State Management Layer (Svelte Stores):**
- Purpose: Manage shared reactive state for trails, users, notifications, etc.
- Location: `web/src/lib/stores/`
- Contains: Writable stores, async data fetching functions (e.g., `trails_index()`, `profile_store()`)
- Depends on: PocketBase client, API utilities, models
- Used by: Components and pages via `$store` syntax

**Models Layer:**
- Purpose: Define data structures and types for all domain entities
- Location: `web/src/lib/models/`
- Contains: Classes (Trail, User, Waypoint, etc.) with constructor logic and expand fields
- Depends on: Child models (e.g., Trail depends on Waypoint, Category)
- Used by: Stores, components, API routes

**API Route Layer (SvelteKit API):**
- Purpose: Bridge between frontend requests and PocketBase backend
- Location: `web/src/routes/api/v1/`
- Contains: GET/POST/PUT/DELETE handlers with validation and error handling
- Depends on: PocketBase client, schemas, utilities
- Used by: Frontend stores and components

**Backend Service Layer (Go):**
- Purpose: Business logic, integrations, federation, caching
- Location: `db/main.go`, `db/routes/`, `db/services/`
- Contains: Custom business logic beyond PocketBase CRUD
- Depends on: Database migrations, external APIs (Valhalla, Nominatim)
- Used by: API routes and SvelteKit endpoints

**Database Layer:**
- Purpose: Data persistence via PocketBase
- Location: `db/pb_data/` (runtime), `db/migrations/` (schema)
- Contains: Collections (trails, users, comments, etc.), indexes
- Depends on: Migration files for schema evolution
- Used by: Backend routes, stores

## Data Flow

### Primary Request Path (Trail Display)

1. **User navigation to `/map`** (`web/src/routes/map/+page.ts:1`)
   - SvelteKit `load()` function runs server-side
   - Calls `trails_search_bounding_box()` or `trails_index()` 

2. **Store fetches data** (`web/src/lib/stores/trail_store.ts:96-100`)
   - Store function makes HTTP request to `/api/v1/trail?expand=...`
   - Includes query params for filtering, sorting, pagination

3. **API Route receives request** (`web/src/routes/api/v1/trail/+server.ts:43`)
   - SvelteKit handler calls `list<Trail>(event, Collection.trails)`
   - Proxies/forwards to PocketBase collection
   - Processes response (sorts waypoints, etc.)
   - Returns JSON

4. **Backend (PocketBase) processes query** (`db/pb_data/`)
   - Queries `trails` collection with filters and expands
   - Returns paginated results with related data (waypoints, tags, category)

5. **Component receives data** (`web/src/routes/map/+page.svelte:1-50`)
   - Routes data into Svelte components via props
   - Components render using reactive binding
   - User interactions trigger store updates

### Feed/Recommendations Flow

1. **Homepage load** (`web/src/routes/+page.ts:7-14`)
   - Calls `feed_index(1, 10, fetch)` and `trails_recommend(4, fetch)`
   
2. **Feed store queries** (`web/src/lib/stores/feed_store.ts`)
   - Calls `/api/v1/feed` endpoint
   - Retrieves ActivityPub timeline items

3. **Recommendations from Meilisearch** (`db/main.go` - Meilisearch integration)
   - Backend queries Meilisearch index
   - Returns randomly selected/recommended trails

### Search Flow

1. **User enters search query** (`web/src/routes/map/+page.svelte`)
   - Calls `trails_search_filter()` with query and filters

2. **Search store processes** (`web/src/lib/stores/search_store.ts:6`)
   - Calls POST `/api/v1/search/trails` with filter text
   - Meilisearch-compatible query format

3. **Backend executes** (`db/routes/` - search endpoint)
   - Meilisearch client searches index
   - Returns hits matching filters and text
   - Hydrates with full Trail objects

4. **Results rendered** (`web/src/routes/map/+page.svelte`)
   - Maps search results to Trail objects
   - Updates map markers and list

**State Management:**
- **Client state:** Stores hold fetched trails, filters, current user via `writable()` stores
- **Server state:** Session/auth via `locals.user` and `locals.pb` (PocketBase instance)
- **Reactive updates:** Components subscribe to stores via `$store` syntax; mutations trigger subscribers

## Key Abstractions

**Trail:**
- Purpose: Represents a recorded/planned path with GPS data, metadata, and social features
- Examples: `web/src/lib/models/trail.ts`, `web/src/lib/stores/trail_store.ts`
- Pattern: Class-based model with optional `expand` field for related data (waypoints, comments, shares)

**Store Functions:**
- Purpose: Encapsulate data fetching logic with error handling and parsing
- Examples: `trails_index()`, `profile_fetch()`, `summit_logs_create()`
- Pattern: Async functions that fetch from API, update writable stores, return parsed models

**Waypoint:**
- Purpose: Point along a trail with location, metadata, optional photos
- Examples: `web/src/lib/models/waypoint.ts`
- Pattern: Ordered by `distance_from_start` along trail

**PocketBase Client:**
- Purpose: Singleton for backend communication
- Location: `web/src/lib/pocketbase.ts`
- Pattern: Lazy singleton accessed via `getPb()`, browser-only

**API Utilities:**
- Purpose: Shared HTTP logic, error handling, response parsing
- Location: `web/src/lib/util/api_util.ts`
- Pattern: Generic functions (`list<T>()`, `create<T>()`, `handleError()`) for CRUD operations

## Entry Points

**Web Frontend:**
- Location: `web/src/routes/+layout.svelte`
- Triggers: Browser navigation to `/`
- Responsibilities: Root layout, navigation bar, theme, authentication guard

**Homepage:**
- Location: `web/src/routes/+page.svelte`
- Triggers: Navigation to `/`
- Responsibilities: Load and display recommended trails and feed

**Map View:**
- Location: `web/src/routes/map/+page.svelte`
- Triggers: Navigation to `/map`
- Responsibilities: Render map, search trails by bounding box, filter UI

**Trail Editor:**
- Location: `web/src/routes/trail/edit/[id]/+page.svelte`
- Triggers: Navigation to `/trail/edit/[id]`
- Responsibilities: Form for creating/editing trails, file uploads (GPX/photos)

**Mobile App:**
- Location: `app/lib/main.dart`
- Triggers: App launch
- Responsibilities: Initialize providers (Riverpod), set up navigation, load user session

**Backend API:**
- Location: `db/main.go`
- Triggers: HTTP requests to `/api/v1/*`
- Responsibilities: Initialize PocketBase, register routes/hooks, start server

## Architectural Constraints

- **Threading:** SvelteKit runs in event loop (Node.js); Go backend may use goroutines for concurrent operations
- **Global state:** PocketBase instance stored in `event.locals.pb` (per-request in SvelteKit); Flutter uses Riverpod providers for app-level state
- **Circular imports:** Models may circularly reference (Trail ↔ Waypoint); patterns avoid by using `expand` field instead of eager loading
- **Authentication:** Session-based via PocketBase AuthStore; frontend checks user via `$currentUser` store
- **Real-time:** PocketBase subscriptions available but not heavily used in current codebase

## Anti-Patterns

### Direct PocketBase Access in Components

**What happens:** Some components may call `getPb()` directly instead of using store functions
**Why it's wrong:** Bypasses centralized state management; makes testing harder; data not synchronized across app
**Do this instead:** Create store function in `web/src/lib/stores/` and use it in components

### Missing Error Boundaries

**What happens:** Network errors from API calls may not be caught consistently
**Why it's wrong:** Unhandled promise rejections; silent failures; poor UX
**Do this instead:** All store functions should use try/catch; wrap in `APIError` utility (`web/src/lib/util/api_util.ts`)

### Tightly Coupled Store Logic

**What happens:** Trail store functions contain UI logic (formatting, filtering) mixed with data fetching
**Why it's wrong:** Difficult to reuse; hard to test; logic scattered
**Do this instead:** Separate API calls from UI transformations; use utility functions in `web/src/lib/util/`

## Error Handling

**Strategy:** Exception-based with custom `APIError` class for HTTP errors; graceful degradation on client; validation via Zod schemas

**Patterns:**
- **API calls:** Wrap in try/catch; throw `APIError` on non-2xx status codes
- **Store functions:** Catch errors; return empty arrays/defaults rather than throwing
- **Components:** Guard against missing data (e.g., `trail?.expand?.waypoints ?? []`)
- **Backend:** Return JSON error objects with status code and detail message

## Cross-Cutting Concerns

**Logging:** 
- Frontend: `console.log()` for development; errors logged via error boundaries
- Backend: Go logger in `db/main.go`

**Validation:**
- Frontend: Zod schemas for input validation (e.g., `TrailCreateSchema`)
- Backend: PocketBase collection rules and custom validation hooks

**Authentication:**
- PocketBase built-in auth; session tokens stored in browser storage
- Protected routes guarded in `web/src/routes/+layout.svelte` via `isRouteProtected()` utility
- API endpoints may check `event.locals.pb.authStore.record` for authorization

**Internationalization:**
- Frontend: `svelte-i18n` package
- Location: `web/src/lib/i18n/`
- Mobile: Flutter localization in `app/lib/i18n/`

---

*Architecture analysis: 2026-06-07*
