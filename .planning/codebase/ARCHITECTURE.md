<!-- refreshed: 2026-06-10 -->
# Architecture

**Analysis Date:** 2026-06-10

## System Overview

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Browser / Mobile Clients                             │
│              (Web: Svelte UI | Mobile: Flutter UI with Riverpod)            │
└──────────────┬─────────────────────────────────────────────────┬────────────┘
               │                                                   │
               ▼                                                   ▼
    ┌──────────────────────────────────┐          ┌──────────────────────────┐
    │    Web Frontend (SvelteKit)      │          │   Mobile App (Flutter)   │
    │  ├─ Routes (`/routes`)          │          │  ├─ Routes (go_router)   │
    │  ├─ Stores (`/lib/stores`)      │          │  ├─ Providers (Riverpod) │
    │  ├─ Components (`/lib/`)        │          │  ├─ Components           │
    │  └─ Utilities (`/lib/util`)     │          │  └─ Services             │
    └──────────────┬──────────────────┘          └──────────┬───────────────┘
                   │                                         │
                   └─────────────────┬─────────────────────┘
                                     │ HTTP/REST
                                     ▼
        ┌────────────────────────────────────────────────────────┐
        │        API Layer (SvelteKit Server Routes)             │
        │        `/api/v1/*` endpoints with Zod validation       │
        └────────────────┬─────────────────────────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────────────────────────┐
        │           Backend (Go + PocketBase)                    │
        │  ├─ Routes (`/db/routes`)                              │
        │  ├─ Hooks (`/db/hooks`) - Event handlers              │
        │  ├─ Federation (`/db/federation`) - ActivityPub       │
        │  ├─ Integrations (`/db/integrations`)                │
        │  └─ Services (`/db/services`) - Business logic        │
        └────────────────┬────────────────┬────────────────────┘
                         │                │
        ┌────────────────▼──┐    ┌─────────▼──────────┐
        │  PocketBase DB    │    │  Meilisearch       │
        │  (SQLite/Postgres)│    │  (Full-text search)│
        └─────────────────┘      └────────────────────┘
```

## Component Responsibilities

| Component | Responsibility | File(s) |
|-----------|----------------|---------|
| **Web Frontend** | Render UI, handle user interactions, client-side state | `web/src/routes/`, `web/src/lib/components/` |
| **Svelte Stores** | Manage reactive state for trails, users, feeds, search | `web/src/lib/stores/` |
| **Models & Types** | Define domain entity structures with expand relations | `web/src/lib/models/` |
| **API Utilities** | Generic CRUD functions, error handling, validation | `web/src/lib/util/api_util.ts` |
| **Mobile Frontend** | Flutter UI with Riverpod state management | `app/lib/` |
| **Mobile Providers** | Riverpod-based state and dependency injection | `app/lib/provider/` |
| **SvelteKit API Routes** | Bridge between frontend and backend, request validation | `web/src/routes/api/v1/` |
| **Go Backend** | Core business logic, integrations, federation | `db/main.go`, `db/routes/` |
| **PocketBase** | Database, authentication, file storage | `db/migrations/` |
| **Meilisearch** | Full-text search indexing and querying | Configured in backend hooks |
| **Federation** | ActivityPub protocol for federated social features | `db/federation/` |
| **Integrations** | Third-party services (Strava, Komoot, Hammerhead) | `db/integrations/` |
| **Hooks** | Event handlers for database changes (search sync, etc.) | `db/hooks/` |

## Pattern Overview

**Overall:** Three-tier architecture with decoupled frontend clients (web/mobile) communicating through REST API with a shared Go/PocketBase backend.

**Key Characteristics:**
- **API-first design:** All data flows through RESTful endpoints; no direct database access from frontends
- **Reactive state management:** Svelte stores (frontend) and Riverpod providers (mobile) subscribe to data changes
- **Type safety:** TypeScript on frontend (with Zod validation), Go on backend with generated types
- **Search integration:** Meilisearch synced via database hooks for full-text indexing
- **Federation support:** ActivityPub implementation for social interoperability

## Layers

**Presentation Layer (Web):**
- Purpose: Render interactive UI, manage user interactions
- Location: `web/src/routes/`, `web/src/lib/components/`
- Contains: `.svelte` components, page layouts, route definitions
- Depends on: Stores, utilities, models
- Used by: Browser via HTTP

**Presentation Layer (Mobile):**
- Purpose: Render Flutter UI for iOS/Android
- Location: `app/lib/` (routes, components, theme)
- Contains: Dart screens, widgets, navigation via go_router
- Depends on: Providers, models, services
- Used by: iOS/Android devices

**State Management (Web):**
- Purpose: Manage reactive client/shared state
- Location: `web/src/lib/stores/`
- Contains: Writable stores, async data fetching functions (`trails_index()`, `currentUser`, etc.)
- Depends on: API utilities, models, PocketBase client
- Used by: Components via `$store` syntax

**State Management (Mobile):**
- Purpose: Manage app-wide state and provide dependencies
- Location: `app/lib/provider/`
- Contains: Riverpod providers for API client, auth, database, routing
- Depends on: Services, models, entities
- Used by: Screens and widgets via `ref.watch(provider)`

**Models & Types Layer:**
- Purpose: Define data structures and type contracts
- Location: `web/src/lib/models/` and `app/lib/models/` + `app/lib/entities/`
- Contains: Classes and types for Trail, User, Waypoint, Comment, etc.
- Depends on: None (leaf layer)
- Used by: Stores, API routes, screens

**API Bridge Layer (SvelteKit):**
- Purpose: Validate requests, transform data, enforce permissions
- Location: `web/src/routes/api/v1/`
- Contains: GET/POST/PUT/DELETE handlers with Zod schemas and error handling
- Depends on: PocketBase client, models, utilities, validation schemas
- Used by: Frontend stores and components

**Backend Services Layer (Go):**
- Purpose: Business logic, integrations, federation, caching
- Location: `db/routes/`, `db/services/`, `db/integrations/`, `db/federation/`
- Contains: Custom endpoints beyond CRUD, external API calls, ActivityPub handling
- Depends on: PocketBase ORM, Meilisearch client, external APIs
- Used by: HTTP clients via REST endpoints

**Persistence Layer:**
- Purpose: Data storage and indexing
- Location: `db/pb_data/` (runtime), `db/migrations/` (schema)
- Contains: PocketBase collections (trails, users, comments, etc.), Meilisearch indexes
- Depends on: Migration scripts for schema evolution
- Used by: Backend via ORM, hooks for event handlers

## Data Flow

### Primary Request Path (Trail Display)

1. User navigates to `/trail/view/[handle]/[id]` → Browser calls `web/src/routes/trail/view/[handle]/[id]/+page.ts`
2. Page load function calls `trails_show()` store function (`web/src/lib/stores/trail_store.ts:line ~240`)
3. Store fetches `/api/v1/trail/[id]?expand=category,waypoints_via_trail,comments_via_trail,author`
4. SvelteKit API route (`web/src/routes/api/v1/trail/[id]/+server.ts`) validates request with `RecordIdSchema` and calls `show<Trail>()` from `api_util.ts`
5. Backend PocketBase returns trail record with expanded relations
6. Store updates writable `Trail` store; component subscribes via `$trail`
7. Trail data rendered with waypoints map, comments, metadata

### Feed/Profile Flow

1. User navigates to `/profile/[handle]` → calls `profile_show()` in store
2. Fetches `/api/v1/profile/[handle]` (custom endpoint in backend)
3. Backend handler in `db/routes/remote_profile.go` fetches user, follower counts, recent trails
4. Returns `Profile` model with nested `User` and recent `Trail[]`
5. Store updates `currentProfile` writable store
6. Component subscribes and renders profile card with avatar, bio, follower stats, recent trails feed

### Search Flow

1. User types in global search component
2. Calls `searchMulti(query)` or `searchActors(query)` from `web/src/lib/stores/search_store.ts`
3. Fetches `/api/v1/search/multi` with Meilisearch query
4. Backend query hits Meilisearch indexes (trails, actors, lists) synced via hooks
5. Returns `Hits<TrailSearchResult | ActorSearchResult | ListSearchResult>`
6. Store updates dropdown items; component displays results (click navigates to detail)

### Mobile Trail Detail Flow

1. User taps trail from list in Flutter app
2. `TrailDetailScreen` provider watches `profileTrailProvider` (fetches from API)
3. Provider calls API via `ApiProvider.build()` → Dio client with cookie jar
4. Fetches `/api/v1/trail/[id]?expand=...` same as web
5. Parses response into `TrailEntity` via converter in `app/lib/models/converter/`
6. Stores locally in ObjectBox for offline access
7. Displays map (via Flutter Map), waypoints, photos, comments

**State Management:**
- Web: Svelte stores hold fetched data as `Writable<T>`; components subscribe with `$store` syntax
- Mobile: Riverpod `AsyncValue<T>` providers manage async data; widgets use `ref.watch()` for reactivity
- Server-side: SvelteKit `event.locals.pb` (PocketBase instance per request), `event.locals.user` (current user)
- Real-time: PocketBase subscriptions available but not heavily used in current codebase

## Key Abstractions

**Trail Entity:**
- Purpose: Represents a recorded/planned path with GPS data and social metadata
- Examples: `web/src/lib/models/trail.ts`, `app/lib/entities/trail_entity.dart`, `db/migrations/` (trails collection)
- Pattern: Class-based model with optional `expand` field for related data (waypoints, comments, author, shares)
- Relations: author (User/Actor), waypoints (Waypoint[]), comments (Comment[]), tags (Tag[]), category (Category)

**Store Functions:**
- Purpose: Encapsulate async data fetching with error handling and parsing
- Examples: `trails_index()`, `trails_show()`, `profile_show()`, `feed_index()` in `web/src/lib/stores/`
- Pattern: Async functions that fetch from API, validate with Zod schemas, update writable stores, return parsed models
- Error handling: Throw `APIError` on non-2xx; components guard with optional chaining (`trail?.expand?.waypoints ?? []`)

**Waypoint Abstraction:**
- Purpose: Point along a trail with location, metadata, optional photos
- Examples: `web/src/lib/models/waypoint.ts`, `app/lib/entities/waypoint_entity.dart`
- Pattern: Ordered by `distance_from_start` along trail; expanded nested inside Trail

**PocketBase Client Singleton:**
- Purpose: Singleton for backend communication
- Location: `web/src/lib/pocketbase.ts`, initialized in `+layout.server.ts`
- Pattern: Lazy singleton accessed via `getPb()`; browser-only; session tokens stored in AuthStore

**API Utilities:**
- Purpose: Shared HTTP logic, error handling, response parsing
- Location: `web/src/lib/util/api_util.ts`
- Pattern: Generic functions (`list<T>()`, `create<T>()`, `show<T>()`, `update<T>()`, `remove()`) for CRUD
- Validation: Zod schemas enforce request/response contracts

**Hook System (Backend):**
- Purpose: Event-driven data sync and validation
- Location: `db/hooks/`
- Pattern: PocketBase event handlers bound in `main.go` → trigger on create/update/delete → sync to Meilisearch, update relations
- Examples: `CreateTrailHandler()` syncs new trails to Meilisearch; `UpdateUserHandler()` updates actor index

## Entry Points

**Web Root:**
- Location: `web/src/routes/+layout.svelte`
- Triggers: Browser navigation to `/`
- Responsibilities: Root layout, navigation bar, theme switching, authentication guard, initialization
- Code: Checks `$currentUser` store; redirects unauthenticated users to `/login` via `isRouteProtected()`

**Web Home Page:**
- Location: `web/src/routes/+page.svelte` + `+page.ts`
- Triggers: Navigation to `/`
- Responsibilities: Load recommended trails (via `trails_recommend()`), render feed of recent activity
- Code: Server-side load fetches feed data; component displays 3D background scene + trail cards

**Web Profile Page:**
- Location: `web/src/routes/profile/[handle]/+page.svelte` + `+page.ts`
- Triggers: Navigation to `/profile/[handle]` (dynamic route)
- Responsibilities: Load profile by handle, display avatar/bio/stats, list user's trails and followers
- Code: Load fetches profile via `profile_show()` store; component renders profile card + nested routes for trails/stats/followers

**Web Map Page:**
- Location: `web/src/routes/map/+page.svelte` + `+page.ts`
- Triggers: Navigation to `/map`
- Responsibilities: Render interactive MapLibre map, filter trails by bounding box, display trail clusters
- Code: Map events trigger bbox search via `trails_search_bbox()`; server-side caching for performance

**Web Trail Editor:**
- Location: `web/src/routes/trail/edit/[id]/+page.svelte` + `+page.ts`
- Triggers: Navigation to `/trail/edit/[id]` (edit) or `/trail/new` (create)
- Responsibilities: Form for creating/editing trails, upload GPX/photos, validate with Zod schema
- Code: Uses Felte form state management; submits to `/api/v1/trail` (POST for create, PUT for update)

**Mobile Root:**
- Location: `app/lib/main.dart`
- Triggers: App launch on iOS/Android
- Responsibilities: Initialize Riverpod providers (ObjectBox, cookie jar, API client), set up navigation
- Code: `ProviderScope` wraps app; initializes local database and HTTP interceptors; routes via `routerProvider`

**Mobile Home Screen:**
- Location: `app/lib/routes/home_screen.dart`
- Triggers: App launch or tab selection
- Responsibilities: Display welcome screen or recent trails list
- Code: Watches `welcomeStateProvider` or `profileTrailProvider` depending on auth state

**Mobile Profile Screen:**
- Location: `app/lib/routes/profile_screen.dart`
- Triggers: User taps profile tab or navigates from trail author
- Responsibilities: Display current user's profile, settings, logout
- Code: Watches `authProvider` for current user; displays profile UI with settings link

**Backend HTTP Server:**
- Location: `db/main.go`
- Triggers: Application startup
- Responsibilities: Initialize PocketBase, register routes/hooks, listen on `:8090`
- Code: `pocketbase.New()` → `registerMigrations()` → `setupEventHandlers()` → `app.Start()`

## Architectural Constraints

- **Threading:** SvelteKit runs in single-threaded Node.js event loop; Go backend may use goroutines for concurrent operations (e.g., processing trail uploads)
- **Global state:** PocketBase instance stored in `event.locals.pb` (per-request in SvelteKit); Flutter uses Riverpod providers for app-level state; web frontend uses `writable` stores
- **Circular imports:** Models may have circular references (Trail ↔ Waypoint); patterns avoid by using `expand` field instead of eager loading; TypeScript `import type` used to break cycles
- **Authentication:** Session-based via PocketBase AuthStore; frontend checks user via `$currentUser` store; API routes validate `event.locals.user` or `ref.read(authProvider)` for authorization
- **Real-time:** PocketBase subscriptions available (websocket) but not heavily used; REST polling used instead for feed updates
- **CORS:** SvelteKit proxy at `/api/v1` handles CORS; backend allows all origins (configured in PocketBase settings)
- **File storage:** Uploads stored in PocketBase file collection; served via `/api/v1/files/[collection]/[record]/[file]`

## Anti-Patterns

### Direct PocketBase Access in Components

**What happens:** Components import `getPb()` and call `pb.collection().getList()` directly instead of using stores.

**Why it's wrong:** Breaks separation of concerns; makes components responsible for data fetching and error handling; data not shared across component tree; difficult to test.

**Do this instead:** Use store functions in `web/src/lib/stores/` that manage state and expose writable stores. Example: `trails_index()` fetches and updates `trails` store; components subscribe with `$trails`.

### Missing Error Boundaries

**What happens:** API failures in store functions bubble up to component level with no graceful fallback.

**Why it's wrong:** Components may render `undefined` data, showing blank screens instead of error messages or empty states.

**Do this instead:** Wrap store function calls in try/catch; return default values (empty arrays, null) instead of throwing. Example: `trails_index()` catches `APIError` and returns empty array, then component guards with `trails ?? []`.

### Tightly Coupled Store Logic

**What happens:** Store functions mix data fetching, parsing, validation, and side effects (updating multiple stores).

**Why it's wrong:** Difficult to test; changes to one entity ripple through multiple stores; hard to reuse logic.

**Do this instead:** Keep store functions focused on single responsibility. Use composition: `trails_create()` creates trail, then calls `tags_create()` for tags. Keep validation in Zod schemas, not store logic.

### Forgetting Expand Relations

**What happens:** Fetch trail without `expand=waypoints_via_trail`, then component tries to access `trail.expand.waypoints` → undefined.

**Why it's wrong:** Silent failures; data missing but no error thrown.

**Do this instead:** Always include required `expand` fields in API calls. Example: `trails_show(id, { expand: 'waypoints_via_trail,author,comments_via_trail' })`. Use TypeScript types to enforce which expands are required.

## Error Handling

**Strategy:** Defensive multi-layer validation with type-safe fallbacks.

**Patterns:**
- **Input validation:** Zod schemas enforce request shape before business logic (`web/src/routes/api/v1/` endpoints)
- **API errors:** Throw `APIError` with status code and detail message; caught by components
- **Graceful degradation:** Components guard with optional chaining (`trail?.waypoints ?? []`) and render empty states
- **User feedback:** Toast notifications for errors via `toast_provider` (mobile) or Toast component (web)
- **Logging:** `console.warn()` for non-critical issues (e.g., markdown fetch failures); no structured logging in frontend
- **Backend logging:** Go `log.Println()` in routes; PocketBase logger for migrations and hooks

## Cross-Cutting Concerns

**Logging:**
- Frontend: `console.log()`, `console.warn()` for development; no production logging framework
- Backend: Go logger in `db/main.go`; PocketBase built-in logger for collections and authentication

**Validation:**
- Frontend: Zod schemas in `web/src/lib/models/api/` for request bodies and query params
- Backend: PocketBase collection rules and custom validation hooks in `db/hooks/`

**Authentication:**
- PocketBase built-in auth with session tokens stored in browser storage and cookie jar (mobile)
- Protected routes guarded in `web/src/routes/+layout.svelte` via `isRouteProtected()` utility
- API endpoints may check `event.locals.pb.authStore.record` (web) or `ref.read(authProvider)` (mobile) for authorization

**Internationalization:**
- Frontend: `svelte-i18n` package; translations in `web/src/lib/i18n/locales/`
- Mobile: Flutter localization in `app/lib/i18n/`
- Centralized message keys referenced in components

---

*Architecture analysis: 2026-06-10*
