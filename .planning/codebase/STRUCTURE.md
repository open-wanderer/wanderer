# Codebase Structure

**Analysis Date:** 2026-06-10

## Directory Layout

```
wanderer/
├── app/                          # Flutter mobile app
│   └── lib/
│       ├── components/           # Reusable Flutter widgets
│       │   ├── base/            # Base widgets (buttons, dialogs, etc.)
│       │   ├── map/             # Map-related components
│       │   ├── profile/         # Profile UI components
│       │   ├── trail/           # Trail detail/list components
│       │   └── welcome/         # Welcome screen components
│       ├── entities/            # Local database entities (ObjectBox)
│       ├── i18n/                # Internationalization (Flutter)
│       ├── models/              # Data models and converters
│       │   └── converter/       # Entity converters (API ↔ local)
│       ├── provider/            # Riverpod state management
│       │   ├── profile/         # Profile-related providers
│       │   ├── search/          # Search providers
│       │   ├── trail/           # Trail-related providers
│       │   └── welcome/         # Welcome screen providers
│       ├── routes/              # Screen definitions (go_router)
│       ├── services/            # Business logic services
│       ├── theme/               # App theme configuration
│       ├── util/                # Utility functions
│       └── main.dart            # App entry point
│
├── db/                           # Go backend with PocketBase
│   ├── main.go                  # Server initialization and setup
│   ├── commands/                # CLI commands for admin tasks
│   ├── federation/              # ActivityPub protocol implementation
│   │   ├── actor.go
│   │   ├── activity.go
│   │   ├── follow.go
│   │   ├── like.go
│   │   └── ...
│   ├── hooks/                   # Database event handlers (create/update/delete)
│   │   ├── trails.go            # Trail events (sync to Meilisearch)
│   │   ├── users.go             # User events (search indexing)
│   │   ├── activitypub_actor.go # Actor sync
│   │   └── ...
│   ├── integrations/            # External service integrations
│   │   ├── strava/              # Strava OAuth and sync
│   │   ├── komoot/              # Komoot integration
│   │   └── hammerhead/          # Hammerhead integration
│   ├── migrations/              # Database schema (SQL)
│   │   ├── initial_data/        # Seed data
│   │   └── 1234567890_*.go      # Timestamped migrations
│   ├── routes/                  # Custom HTTP endpoints
│   │   ├── activitypub.go       # ActivityPub endpoints
│   │   ├── remote_profile.go    # Profile fetching
│   │   ├── map_cells.go         # Map tile serving
│   │   └── ...
│   ├── services/                # Business logic
│   │   ├── tiles/               # Tile generation
│   │   └── trailmerge/          # Trail merging algorithm
│   ├── templates/               # Email templates
│   ├── tests/                   # Go integration tests
│   ├── util/                    # Utility functions
│   └── pb_data/                 # PocketBase data directory (runtime)
│
├── web/                          # SvelteKit web frontend
│   ├── src/
│   │   ├── routes/              # SvelteKit page routes (filesystem-based)
│   │   │   ├── +layout.svelte   # Root layout
│   │   │   ├── +page.svelte     # Home page
│   │   │   ├── +page.ts         # Home page load function
│   │   │   ├── profile/[handle]/ # Profile routes (dynamic segment)
│   │   │   │   ├── +page.svelte
│   │   │   │   ├── +page.ts
│   │   │   │   ├── trails/      # Profile trails subroute
│   │   │   │   ├── stats/       # Profile stats subroute
│   │   │   │   └── users/[type]/ # Followers/following lists
│   │   │   ├── trail/           # Trail routes
│   │   │   │   ├── view/[handle]/[id]/ # Trail detail
│   │   │   │   ├── edit/[id]/   # Trail editor
│   │   │   │   └── ...
│   │   │   ├── map/             # Map page
│   │   │   ├── lists/           # Trail lists
│   │   │   ├── search/          # Search results
│   │   │   ├── settings/        # Settings pages
│   │   │   │   ├── account/
│   │   │   │   ├── profile/
│   │   │   │   ├── integrations/
│   │   │   │   ├── privacy/
│   │   │   │   └── ...
│   │   │   ├── auth/            # Authentication pages
│   │   │   │   ├── login/
│   │   │   │   ├── register/
│   │   │   │   ├── reset/
│   │   │   │   └── confirm-*/
│   │   │   ├── api/v1/          # SvelteKit API routes (proxy/validation layer)
│   │   │   │   ├── trail/
│   │   │   │   │   ├── +server.ts # GET/POST trails
│   │   │   │   │   ├── [id]/
│   │   │   │   │   │   ├── +server.ts
│   │   │   │   │   │   ├── comment/ # Trail comments
│   │   │   │   │   │   └── file/ # Trail file upload
│   │   │   │   │   ├── form/ # Trail form data (initial state)
│   │   │   │   │   ├── bounding-box/ # Bbox search
│   │   │   │   │   ├── filter/ # Advanced filtering
│   │   │   │   │   └── ...
│   │   │   │   ├── profile/[handle]/ # Profile endpoints
│   │   │   │   ├── search/ # Search endpoints
│   │   │   │   ├── user/ # User management
│   │   │   │   ├── comment/ # Comments
│   │   │   │   ├── category/ # Categories
│   │   │   │   └── ...
│   │   │   └── .well-known/ # ActivityPub webfinger
│   │   │
│   │   ├── lib/
│   │   │   ├── assets/          # Static assets (fonts, SVGs)
│   │   │   │   ├── fonts/
│   │   │   │   └── svgs/
│   │   │   │       └── empty_states/ # No-data UI illustrations
│   │   │   │
│   │   │   ├── components/      # Reusable Svelte components (PascalCase)
│   │   │   │   ├── base/        # Base UI components (Button, Modal, etc.)
│   │   │   │   ├── trail/       # Trail-specific components
│   │   │   │   ├── profile/     # Profile components
│   │   │   │   ├── map/         # Map components
│   │   │   │   ├── comment/     # Comment display/form
│   │   │   │   ├── list/        # List UI components
│   │   │   │   ├── 3D/          # Three.js/Threlte 3D components
│   │   │   │   ├── notification/ # Toast/notification UI
│   │   │   │   ├── empty_states/ # Empty state illustrations
│   │   │   │   └── ...
│   │   │   │
│   │   │   ├── stores/          # Svelte stores (snake_case with _store suffix)
│   │   │   │   ├── trail_store.ts # Trail operations and state
│   │   │   │   ├── user_store.ts  # User/auth state
│   │   │   │   ├── feed_store.ts  # Feed data
│   │   │   │   ├── search_store.ts # Search state
│   │   │   │   ├── profile_store.ts # Profile data
│   │   │   │   └── ...
│   │   │   │
│   │   │   ├── models/          # TypeScript type definitions and classes
│   │   │   │   ├── trail.ts     # Trail class with expand relations
│   │   │   │   ├── user.ts      # User/Actor types
│   │   │   │   ├── waypoint.ts  # Waypoint for trails
│   │   │   │   ├── comment.ts   # Comment model
│   │   │   │   ├── tag.ts       # Tag model
│   │   │   │   ├── api/         # API-specific types and schemas
│   │   │   │   │   └── base_schema.ts # Zod schemas (RecordListOptions, etc.)
│   │   │   │   ├── activitypub/ # ActivityPub types
│   │   │   │   ├── gpx/         # GPX parsing types
│   │   │   │   └── ...
│   │   │   │
│   │   │   ├── util/            # Utility functions (snake_case with _util suffix)
│   │   │   │   ├── api_util.ts  # Generic CRUD functions (list, show, create, etc.)
│   │   │   │   ├── authorization_util.ts # Route protection checks
│   │   │   │   ├── array_util.ts # Array helpers
│   │   │   │   ├── date_util.ts # Date/time helpers
│   │   │   │   ├── file_util.ts # File upload/download
│   │   │   │   ├── icon_util.ts # Icon selection
│   │   │   │   └── ...
│   │   │   │
│   │   │   ├── server/          # Server-only utilities (SSR)
│   │   │   ├── config/          # Configuration constants
│   │   │   ├── vendor/          # Third-party libraries (vendored)
│   │   │   │   ├── exif-js/
│   │   │   │   ├── fit-parser/
│   │   │   │   ├── maplibre-*/  # MapLibre plugins
│   │   │   │   └── ...
│   │   │   ├── i18n/            # Internationalization
│   │   │   │   └── locales/     # Translation JSON files
│   │   │   └── pocketbase.ts    # PocketBase client singleton
│   │   │
│   │   └── css/
│   │       ├── app.css          # Global styles
│   │       ├── components.css   # Component-level styles
│   │       └── theme.css        # Theme variables
│   │
│   ├── tests/
│   │   └── playwright/          # E2E tests
│   │       ├── pages/           # Page Object Models
│   │       ├── *.setup.ts       # Setup/teardown
│   │       └── *.spec.ts        # Test specs
│   │
│   ├── vite.config.ts           # Vite build configuration
│   ├── svelte.config.js         # SvelteKit configuration
│   ├── tsconfig.json            # TypeScript config
│   ├── tailwind.config.js        # Tailwind CSS config
│   ├── playwright.config.ts     # E2E test configuration
│   └── package.json             # npm dependencies
│
├── docs/                         # Documentation site (Astro)
│   ├── astro.config.mjs
│   ├── package.json
│   └── src/
│
├── search/                       # Meilisearch configuration (Docker)
├── docker/                       # Docker build artifacts
├── data/                         # Sample/test data
│
├── docker-compose.yml           # Multi-container orchestration
├── Dockerfile                   # Web/backend container definitions
├── CHANGELOG.md                 # Version history
├── CLAUDE.md                    # Project instructions (this file)
├── CONTRIBUTING.md              # Contribution guidelines
├── README.md                    # Project overview
└── Makefile                     # Development commands
```

## Directory Purposes

**`app/lib/`:**
- Purpose: Flutter mobile app source code
- Contains: UI screens, state management (Riverpod), models, services, utilities
- Key files: `main.dart` (entry point), `routes/` (screens), `provider/` (state)

**`db/`:**
- Purpose: Go backend with PocketBase
- Contains: Custom routes, event hooks, integrations, database migrations, ActivityPub federation
- Key files: `main.go` (server startup), `hooks/` (event handlers), `migrations/` (schema)

**`web/src/routes/`:**
- Purpose: SvelteKit filesystem-based routing
- Contains: Page components (`.svelte`), page load functions (`.ts`), API endpoints (`/api/v1/`)
- Pattern: `+page.svelte` (UI), `+page.ts` (load), `+layout.svelte` (nested layout), `+server.ts` (API)
- Route segments: `[dynamic]` for single param, `[[optional]]` for optional param

**`web/src/lib/stores/`:**
- Purpose: Svelte reactive state management
- Contains: Writable stores and async fetch functions with error handling
- Naming: `{entity}_store.ts` with functions like `{entity}_index()`, `{entity}_show()`, `{entity}_create()`
- Examples: `trail_store.ts` (trails CRUD), `user_store.ts` (auth), `feed_store.ts` (feed data)

**`web/src/lib/components/`:**
- Purpose: Reusable Svelte UI components
- Contains: PascalCase `.svelte` files organized by domain
- Subdirs: `base/` (generic UI), `trail/` (trail-specific), `profile/` (profile UI), `map/` (map widgets), `3D/` (Three.js components)

**`web/src/lib/models/`:**
- Purpose: TypeScript type definitions and data classes
- Contains: Class definitions with constructor logic, type interfaces, expand relations
- Examples: `Trail` class with expand field for nested relations; `User` interface for actor data
- Subdirs: `api/` (Zod schemas for validation), `activitypub/` (ActivityPub types), `gpx/` (GPX parsing)

**`web/src/lib/util/`:**
- Purpose: Shared utility functions and helpers
- Naming: `{purpose}_util.ts` (e.g., `api_util.ts`, `date_util.ts`, `file_util.ts`)
- Contents: Verb-first function names (e.g., `range()`, `isToday()`, `getIconForLocation()`)

**`web/src/lib/vendor/`:**
- Purpose: Vendored third-party libraries (included in source, not npm)
- Contains: Modified versions of maplibre plugins, custom QR code renderer, chart.js wrapper

**`db/hooks/`:**
- Purpose: Database event handlers
- Contains: PocketBase event listeners for create/update/delete on collections
- Pattern: `{collection}.go` with handler functions called by `main.go` via `.BindFunc()`
- Key hooks: `CreateTrailHandler` (sync to Meilisearch), `UpdateUserHandler` (search index update)

**`db/migrations/`:**
- Purpose: Database schema definitions
- Contains: Go migration files with SQL statements, executed on startup
- Naming: `{timestamp}_{description}.go` (e.g., `1742167033_init_meilisearch.go`)

**`db/routes/`:**
- Purpose: Custom HTTP endpoints beyond CRUD
- Contains: Go request handlers for special operations (ActivityPub, remote profile, map cells, etc.)

**`app/lib/provider/`:**
- Purpose: Riverpod state management providers
- Contains: Code-generated providers (`*.g.dart` files) with dependency injection
- Key providers: `apiProvider` (HTTP client), `authProvider` (auth state), `routerProvider` (navigation)

**`app/lib/routes/`:**
- Purpose: Flutter screen definitions
- Naming: `{screen_name}_screen.dart` (e.g., `profile_screen.dart`, `trail_detail_screen.dart`)

**`app/lib/entities/`:**
- Purpose: ObjectBox local database entities
- Contains: Entity classes for offline storage
- Examples: `TrailEntity`, `WaypointEntity`, `UserEntity`

## Key File Locations

**Entry Points:**

| File | Purpose |
|------|---------|
| `web/src/routes/+layout.svelte` | Root layout, auth guard, navigation |
| `web/src/routes/+page.svelte` | Home page (home feed + recommendations) |
| `app/lib/main.dart` | Mobile app initialization |
| `db/main.go` | Backend server startup |

**Configuration:**

| File | Purpose |
|------|---------|
| `web/vite.config.ts` | Vite bundler, plugins, test config |
| `web/svelte.config.js` | SvelteKit framework config (Node.js adapter) |
| `web/tsconfig.json` | TypeScript strict mode and compiler options |
| `web/tailwind.config.js` | Tailwind CSS design tokens |
| `app/pubspec.yaml` | Flutter dependencies |
| `docker-compose.yml` | Service orchestration (web, db, meilisearch) |

**Core Logic:**

| File | Purpose |
|------|---------|
| `web/src/lib/stores/trail_store.ts` | Trail CRUD operations and search |
| `web/src/lib/util/api_util.ts` | Generic API functions (list, show, create, update, delete) |
| `web/src/lib/models/trail.ts` | Trail data class with expand relations |
| `db/routes/activitypub.go` | ActivityPub webfinger and actor endpoints |
| `db/hooks/trails.go` | Trail event handlers (Meilisearch sync) |

**Testing:**

| File | Purpose |
|------|---------|
| `web/playwright.config.ts` | E2E test configuration |
| `web/tests/playwright/` | Playwright test specs |
| `web/tests/playwright/pages/` | Page Object Models |

## Naming Conventions

**Files:**
- **Svelte components:** PascalCase (e.g., `TrailCard.svelte`, `Scene.svelte`, `NavBar.svelte`)
- **Svelte pages/routes:** SvelteKit convention (`+page.svelte`, `+layout.svelte`, `+server.ts`, `+error.svelte`)
- **Utilities:** snake_case with `_util` suffix (e.g., `api_util.ts`, `date_util.ts`, `authorization_util.ts`)
- **Stores:** snake_case with `_store` suffix (e.g., `trail_store.ts`, `user_store.ts`, `search_store.ts`)
- **Models:** snake_case (e.g., `trail.ts`, `user.ts`, `waypoint.ts`)
- **Go files:** snake_case (e.g., `activitypub.go`, `remote_profile.go`)
- **Dart files:** snake_case (e.g., `trail_detail_screen.dart`, `auth_provider.dart`)
- **Test files:** `*.spec.ts`, `*.setup.ts`, `*.teardown.ts` (Playwright)

**Directories:**
- **Feature domains:** lowercase (e.g., `trail/`, `profile/`, `map/`, `search/`)
- **Generic ui:** lowercase (e.g., `base/`, `empty_states/`, `notification/`)
- **Internal structure:** `lib/` (shared), `routes/` (pages), `models/` (types), `stores/` (state)

**Functions:**
- **Verb-first utilities:** `range()`, `isToday()`, `getIconForLocation()`, `dateExistsInList()`
- **Store operations:** `{entity}_{operation}()` (e.g., `trails_index()`, `trails_show()`, `trails_create()`, `trails_delete()`)
- **Handler functions:** HTTP method names (e.g., `GET`, `POST`, `PUT`, `DELETE`, `PATCH`)
- **Event handlers:** `{noun}{Event}()` (e.g., `CreateTrailHandler()`, `UpdateUserHandler()`)

**Variables:**
- **camelCase:** All variables and properties (e.g., `currentUser`, `trailId`, `isLoading`)
- **Boolean prefix:** `is` (e.g., `isValid`, `isLoading`, `isRouteProtected`)
- **Constants:** `const` with CONSTANT_CASE or camelCase (e.g., `const privateRoutes = [...]`, `const MAP_MAX_POLYLINES = 100`)

**Types:**
- **PascalCase:** All types and interfaces (e.g., `Trail`, `User`, `Waypoint`, `TrailFilter`, `APIError`)
- **Enums:** lowercase values (e.g., `Collection.trails`, `Language.en`)

## Where to Add New Code

**New Feature (e.g., "summit logs"):**

1. **Web Frontend:**
   - Add page: `web/src/routes/summit-log/view/[handle]/[id]/+page.svelte`
   - Add store: `web/src/lib/stores/summit_log_store.ts` with `summit_logs_index()`, `summit_logs_show()`, etc.
   - Add model: `web/src/lib/models/summit_log.ts` with class definition
   - Add API route: `web/src/routes/api/v1/summit-log/+server.ts` (proxy to backend)
   - Add components: `web/src/lib/components/summit_log/` with UI components

2. **Mobile Frontend:**
   - Add screen: `app/lib/routes/summit_log_screen.dart`
   - Add provider: `app/lib/provider/summit_log/` with Riverpod providers
   - Add model: `app/lib/models/` and entity in `app/lib/entities/`
   - Add components: `app/lib/components/summit_log/`

3. **Backend:**
   - Add migration: `db/migrations/{timestamp}_created_summit_logs.go` (schema)
   - Add routes: `db/routes/summit_logs.go` (custom endpoints if needed beyond CRUD)
   - Add hooks: `db/hooks/summit_logs.go` (Meilisearch sync, validations)
   - Add models/types in Go code (if custom business logic)

**New Component/Module:**

- **Shared utilities:** `web/src/lib/util/{purpose}_util.ts`
- **Shared models:** `web/src/lib/models/{entity}.ts`
- **UI components:** `web/src/lib/components/{domain}/{ComponentName}.svelte`
- **Mobile components:** `app/lib/components/{domain}/` with `.dart` files

**Integration with Third-Party Service:**

- Create folder: `db/integrations/{service}/` (e.g., `db/integrations/komoot/`)
- Implement OAuth flow and sync logic
- Add routes in `db/routes/{service}_*.go`
- Add settings/configuration page in `web/src/routes/settings/integrations/`
- Add mobile screen in `app/lib/routes/` for service setup

## Special Directories

**`web/src/lib/vendor/`:**
- Purpose: Vendored third-party libraries
- Generated: No (manually maintained)
- Committed: Yes (included in source for reliability)
- Usage: Import directly like `import { ... } from '$lib/vendor/...'`

**`db/pb_data/`:**
- Purpose: PocketBase runtime data (SQLite database, uploads, logs)
- Generated: Yes (created at runtime by PocketBase)
- Committed: No (in `.gitignore`; data only)
- Usage: Not directly accessed; accessed via PocketBase API

**`web/tests/playwright/`:**
- Purpose: E2E tests
- Generated: No (written manually)
- Committed: Yes
- Usage: Run with `npm run test:playwright`

**`db/migrations/`:**
- Purpose: Database schema definitions
- Generated: No (written manually when schema changes)
- Committed: Yes
- Usage: Auto-applied on server startup via PocketBase

**`.planning/codebase/`:**
- Purpose: GSD codebase analysis documents
- Generated: Yes (by GSD tools)
- Committed: Yes
- Usage: Referenced by GSD planning and execution phases

---

*Structure analysis: 2026-06-10*
