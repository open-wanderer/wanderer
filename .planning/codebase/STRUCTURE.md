# Codebase Structure

**Analysis Date:** 2026-09-06

## Directory Layout

```
wanderer/                          # Monorepo root
├── web/                           # SvelteKit frontend application
│   ├── src/
│   │   ├── routes/                # SvelteKit file-based routing (pages, API endpoints)
│   │   ├── lib/
│   │   │   ├── components/        # Reusable Svelte UI components
│   │   │   ├── stores/            # Reactive Svelte stores and data fetching
│   │   │   ├── models/            # TypeScript domain models and types
│   │   │   ├── util/              # Utility functions (arrays, dates, geospatial, etc.)
│   │   │   ├── server/            # Server-only utilities (run on +server.ts)
│   │   │   ├── assets/            # Images, SVGs, fonts
│   │   │   ├── vendor/            # Vendored third-party code (MapLibre plugins, etc.)
│   │   │   ├── i18n/              # Internationalization files
│   │   │   ├── config/            # Configuration constants
│   │   │   └── pocketbase.ts      # PocketBase client singleton
│   │   └── css/                   # Global CSS and Tailwind
│   ├── svelte.config.js           # SvelteKit configuration
│   ├── tsconfig.json              # TypeScript configuration
│   ├── tailwind.config.js         # Tailwind CSS configuration
│   ├── vite.config.ts             # Vite bundler configuration
│   └── package.json               # Dependencies
│
├── db/                            # Go backend (PocketBase + custom services)
│   ├── main.go                    # Entry point, PocketBase initialization
│   ├── routes/                    # Custom API endpoints (Go http handlers)
│   ├── hooks/                     # Event listeners on collection changes
│   ├── federation/                # ActivityPub implementation
│   ├── services/                  # Business logic services (regions, trail merging)
│   ├── integrations/              # External API integrations (Strava, etc.)
│   ├── plugins/                   # Plugin system and importers
│   ├── pluginsystem/              # Plugin worker and execution
│   ├── migrations/                # Database schema migrations
│   ├── util/                      # Go utilities (geospatial, network, etc.)
│   ├── templates/                 # Email templates for notifications
│   ├── commands/                  # CLI commands
│   ├── tests/                     # Backend tests
│   ├── pb_data/                   # PocketBase runtime data (git-ignored)
│   └── go.mod                     # Go module definition
│
├── app/                           # Flutter mobile application
│   ├── lib/
│   │   ├── main.dart              # App entry point
│   │   ├── routes/                # Screen definitions (pages in mobile)
│   │   ├── provider/              # Riverpod state management providers
│   │   ├── components/            # Reusable Flutter widgets
│   │   ├── entities/              # Dart model classes (freezed)
│   │   ├── models/                # API model mappers and converters
│   │   ├── services/              # Business logic services (GPS, sync, etc.)
│   │   ├── store/                 # Local stores (ObjectBox, preferences)
│   │   ├── actions/               # User-triggered actions
│   │   ├── util/                  # Utility functions
│   │   ├── theme/                 # UI theme and styling
│   │   ├── i18n/                  # Internationalization
│   │   └── vendor/                # Vendored code (GPX parser, etc.)
│   ├── ios/                       # iOS platform-specific code
│   ├── android/                   # Android platform-specific code
│   ├── test/                      # Flutter tests
│   ├── pubspec.yaml               # Flutter dependencies
│   └── pubspec.lock               # Locked dependency versions
│
├── docs/                          # Astro documentation site
│   ├── src/
│   │   ├── pages/                 # Documentation pages (Markdown)
│   │   ├── content/               # Content collection files
│   │   ├── components/            # Reusable doc components (Astro)
│   │   ├── assets/                # Images, stylesheets
│   │   └── models/                # TypeScript types for content
│   └── astro.config.mjs           # Astro configuration
│
├── docker/                        # Docker-related files
├── plugins/                       # Plugin SDK and example plugins
├── fixtures/                      # Test fixtures (GPX corpus, etc.)
├── data/                          # Runtime data directories (git-ignored)
├── docker-compose.yml             # Development environment setup
├── .planning/                     # Planning and analysis documents
├── .claude/                       # Claude-specific project config
├── Makefile                       # Development convenience commands
└── CLAUDE.md                      # Project documentation and conventions
```

## Directory Purposes

**`web/src/routes/`:**
- Purpose: SvelteKit file-based routing — defines all web URLs and API endpoints
- Contains: `+page.svelte` (pages), `+server.ts` (API routes), `+layout.svelte` (layouts), `+error.svelte` (error pages)
- Key files: 
  - `web/src/routes/+layout.svelte` — Root layout with auth guard, theme, navbar
  - `web/src/routes/+page.ts` — Home page loader fetching feed and recommendations
  - `web/src/routes/api/v1/trail/+server.ts` — Trail CRUD endpoints
  - `web/src/routes/api/v1/activitypub/` — Federation endpoints

**`web/src/lib/components/`:**
- Purpose: Reusable Svelte UI components organized by domain
- Contains: `.svelte` files using SvelteKit 5 syntax
- Subdirectories:
  - `base/` — Low-level components (Button, Input, Toast, etc.)
  - `trail/` — Trail-specific components (TrailCard, ElevationProfile, etc.)
  - `map/` — Map-related components (MapContainer, Markers, etc.)
  - `comment/` — Comment UI elements
  - `settings/` — Settings page components
  - `profile/` — User profile components
- Naming: PascalCase (e.g., `TrailCard.svelte`, `NavigationBar.svelte`)

**`web/src/lib/stores/`:**
- Purpose: Svelte reactive stores and async data-fetching functions
- Contains: `.ts` or `.svelte.ts` files defining writable stores and fetch logic
- Pattern: Named exports; store + associated fetch functions in same file (e.g., `trail_store.ts` exports `trailsIndex` store and `trails_index()` fetch function)
- Examples:
  - `trail_store.ts` — Trail data store and fetching (`trails_index()`, `trails_show()`, `trails_create()`)
  - `user_store.ts` — Current user store and auth operations (`login()`, `logout()`, `register()`)
  - `feed_store.ts` — Feed items and fetching logic
  - `search_store.ts` — Search results store

**`web/src/lib/models/`:**
- Purpose: TypeScript domain models and API types
- Contains: TypeScript class definitions, interfaces, Zod schemas
- Key files:
  - `trail.ts` — Trail class with expand fields for relations
  - `user.ts` — User entity
  - `waypoint.ts` — Waypoint class
  - `comment.ts` — Comment entity
  - `activitypub/actor.ts` — ActivityPub Actor type
  - `api/` — Request/response schemas using Zod
- Pattern: Classes with optional `expand` field for lazy-loaded relations

**`web/src/lib/util/`:**
- Purpose: Utility and helper functions
- Naming: snake_case suffixed by domain (e.g., `array_util.ts`, `date_util.ts`, `geospatial_util.ts`)
- Key files:
  - `api_util.ts` — Generic CRUD functions, error handling
  - `authorization_util.ts` — Auth checks
  - `array_util.ts` — Array manipulation
  - `date_util.ts` — Date helpers
  - `gpx_util.ts` — GPX parsing
  - `polyline_util.ts` — Polyline encoding/decoding
  - `maplibre_util.ts` — Map utilities

**`web/src/lib/server/`:**
- Purpose: Server-only utilities running in `+server.ts` and `+page.server.ts`
- Key files:
  - `category_preference_filter.ts` — Filter builder for user's category preferences
- Accessed via `import { ... } from '$lib/server/...'` only in server context

**`db/routes/`:**
- Purpose: Custom Go HTTP handlers for API endpoints beyond PocketBase CRUD
- Naming: `{entity}_{operation}.go` or descriptive name
- Key files:
  - `activitypub.go` — Federation endpoints setup
  - `plugin_system.go` — Plugin system endpoints
  - `trail_merge_routes.go` — Trail merging logic
  - `regions_*.go` — Offline region management
  - `remote_*.go` — Remote instance fetching
- Pattern: Each file registers routes in `init()` that hook into PocketBase's router

**`db/hooks/`:**
- Purpose: Event listeners triggered on collection changes (CRUD operations)
- Naming: `{collection_name}.go`
- Key files:
  - `trails.go` — Trail creation/update/delete hooks (indexing, federation fanout)
  - `users.go` — User events (Meilisearch indexing)
  - `comments.go` — Comment federation
  - `follow.go` — Follow/unfollow logic
  - `activitypub_actor.go` — Actor lifecycle
- Pattern: `OnRecordAfterCreateSuccess()`, `OnRecordAfterUpdateSuccess()` bindings

**`db/federation/`:**
- Purpose: ActivityPub protocol implementation
- Key files:
  - `actor.go` — Instance actor management
  - `activity.go` — Generic activity handling, signature verification, delivery
  - `create.go` — Create activity generation and fanout
  - `update.go` — Update activity generation
  - `delete.go` — Delete activity generation
  - `follow.go` — Follow/accept/undo logic
  - `like.go` — Like activity handling
- Pattern: Receive incoming activities at inbox endpoints, verify signatures, process and store locally

**`db/services/`:**
- Purpose: Specialized domain-specific services
- Subdirectories:
  - `regions/` — Offline map region management (download, sync, PMTiles)
  - `trailmerge/` — Trail deduplication and merging logic

**`db/migrations/`:**
- Purpose: Database schema migrations managed by PocketBase
- Files: Numbered SQL files (e.g., `1694000000_create_trails.sql`)
- Pattern: Auto-run on startup; define tables, indexes, foreign keys
- Subdirectory: `initial_data/` for seed data

**`app/lib/routes/`:**
- Purpose: Flutter screen/page definitions (analogous to web routes)
- Naming: `{feature}_screen.dart`
- Examples:
  - `home_screen.dart` — Home/feed page
  - `map_screen.dart` — Map view
  - `trail_detail_screen.dart` — Trail detail view
  - `trail_create_screen.dart` — Trail creation form
- Pattern: `StatelessWidget` or `ConsumerWidget` (Riverpod); navigated via `GoRouter`

**`app/lib/provider/`:**
- Purpose: Riverpod providers for state management and async data fetching
- Naming: `{entity}_provider.dart` for main providers, subdirectories for grouped providers
- Subdirectories:
  - `trail/` — Trail-related providers
  - `profile/` — User profile providers
  - `search/` — Search providers
  - `region/` — Offline region providers
- Pattern: `final trailProvider = FutureProvider<Trail>((ref) => ...)` or async `AsyncNotifier`

**`app/lib/services/`:**
- Purpose: Business logic and platform services
- Key files:
  - `tile_proxy_server.dart` — Local HTTP server for offline map tiles
  - `tracelet_position_source.dart` — GPS position stream
  - `trail_download_service.dart` — Offline trail caching
  - `session_gap_backfill.dart` — Fill gaps when resuming recording

**`app/lib/store/`:**
- Purpose: Local persistent storage and app-level state
- Key files:
  - `local_trail_store.dart` — ObjectBox queries for local trails
  - `local_photo_store.dart` — Photo metadata storage
  - `active_navigation_store.dart` — Current navigation state
  - `current_account.dart` — Current logged-in user
- Pattern: Wrapper around ObjectBox ORM or SharedPreferences

**`docs/src/pages/`:**
- Purpose: User-facing documentation in Markdown
- Pattern: Astro pages auto-converted from `.md` files to HTML
- Examples: Installation guides, API docs, FAQ

## Key File Locations

**Entry Points:**
- Web: `web/src/routes/+layout.svelte` — Root layout and auth guard
- Web API: `web/src/routes/api/v1/` — All backend endpoints accessible to frontend
- Backend: `db/main.go` — PocketBase initialization and server startup
- Mobile: `app/lib/main.dart` — Flutter app initialization

**Configuration:**
- Web TypeScript: `web/tsconfig.json` — Strict mode, path aliases
- Web styling: `web/tailwind.config.js` — Tailwind setup; `web/src/css/app.css` — Global styles
- Backend: `db/main.go` — Environment variable verification, Meilisearch initialization
- Mobile: `app/pubspec.yaml` — Flutter SDK version, dependencies, localization

**Core Logic:**
- Frontend state: `web/src/lib/stores/` — All data fetching and reactive state
- Backend services: `db/routes/`, `db/hooks/`, `db/services/` — Business logic
- Backend federation: `db/federation/` — ActivityPub implementation
- Mobile state: `app/lib/provider/` — Riverpod providers for async data

**Testing:**
- Web unit tests: `web/src/lib/**/*.test.ts` or `**/*.spec.ts`
- Web E2E: `web/e2e/` (if exists) or `web/tests/` with Playwright config in `web/playwright.config.ts`
- Backend tests: `db/tests/` — Go test files
- Mobile tests: `app/test/` — Flutter/Dart tests

## Naming Conventions

**Files:**
- Svelte components: PascalCase (e.g., `TrailCard.svelte`, `NavigationBar.svelte`)
- TypeScript modules: snake_case (e.g., `trail_store.ts`, `api_util.ts`, `authorization_util.ts`)
- Utilities: snake_case with suffix indicating domain (e.g., `array_util.ts`, `date_util.ts`, `geospatial_util.ts`)
- Store files: snake_case with `_store` suffix (e.g., `trail_store.ts`, `user_store.ts`, `feed_store.ts`)
- Model files: snake_case (e.g., `trail.ts`, `user.ts`, `category.ts`)
- API schemas: `{entity}_schema.ts` (e.g., `trail_schema.ts`, `user_schema.ts`)
- Routes: SvelteKit convention — `+page.svelte`, `+server.ts`, `+layout.svelte`, `+error.svelte`, `[param]` for dynamic segments
- Go files: snake_case (e.g., `trail_merge_routes.go`, `activitypub_actor.go`)
- Flutter screens: `{feature}_screen.dart` (e.g., `home_screen.dart`, `trail_detail_screen.dart`)
- Flutter providers: `{entity}_provider.dart` (e.g., `trail_provider.dart`, `auth_provider.dart`)

**Directories:**
- Web components: `web/src/lib/components/{feature}/` — Organized by feature/domain
- Web stores: `web/src/lib/stores/` — Flat structure, one store per entity typically
- Web utilities: `web/src/lib/util/` — Flat structure with domain suffix
- API routes: `web/src/routes/api/v1/{entity}/` — RESTful path structure
- Backend services: `db/services/{domain}/` — Grouped by concern

**Functions:**
- camelCase for all function names
- Store fetch functions: `{entity}_{operation}` (e.g., `trails_index()`, `users_create()`, `profile_fetch()`)
- Utility functions: verb-first or domain-focused (e.g., `range()`, `isToday()`, `dateExistsInList()`, `isSameDay()`)
- Handler functions in routes: HTTP method names (`GET`, `PUT`, `POST`, `DELETE`, `PATCH`)

**Variables:**
- camelCase for all variable names
- Boolean prefix: `is` or `has` (e.g., `isToday`, `isRouteProtected`, `hasError`)
- Readonly arrays: explicitly typed `ReadonlyArray<T>` (e.g., `ReadonlyArray<number>`)
- Constants: `const CONSTANT_NAME` at module level (e.g., `const privateRoutes = [...]`)

**Types:**
- PascalCase for all types and interfaces (e.g., `Trail`, `User`, `TrailCreateSchema`)
- Enum values: lowercase snake_case (e.g., `Collection.users`, `Language.en`)
- Classes: PascalCase (e.g., `Trail`, `User`, `Settings`, `IndexPage`)

## Where to Add New Code

**New Feature (Trail-related example):**
- Primary code: `web/src/lib/stores/trail_store.ts` — Add new fetch function, extend `Trail` model if needed
- Component: `web/src/lib/components/trail/` — Add new component for the feature
- API route: `web/src/routes/api/v1/trail/` — Add `+server.ts` handler or subpath
- Backend: `db/routes/` — Extend or create route handler; `db/hooks/trails.go` — Add hook if reacting to changes
- Tests: `web/src/lib/stores/trail_store.test.ts` — Unit tests for store functions
- Mobile: `app/lib/provider/trail/` — Add Riverpod provider, `app/lib/routes/` — Add screen

**New Component/Module:**
- Implementation: `web/src/lib/components/{feature}/{ComponentName}.svelte` — Svelte component
- Types: `web/src/lib/models/{entity}.ts` — Model class if introducing new entity
- Utilities: `web/src/lib/util/{domain}_util.ts` — Helper functions for component
- Stories/tests: Co-locate `.test.ts` or documentation

**Utilities:**
- Shared helpers: `web/src/lib/util/` — Create `{domain}_util.ts`, export named functions
- Backend utilities: `db/util/` — Create `.go` file with helper functions
- Mobile utilities: `app/lib/util/{domain}/` — Dart utilities organized by domain

**Backend Business Logic:**
- Custom endpoints: `db/routes/{entity}_{operation}.go` — Register in PocketBase router
- Event hooks: `db/hooks/{collection_name}.go` — Add listener to collection lifecycle
- Services: `db/services/{domain}/` — Create service for complex multi-step logic
- Migrations: `db/migrations/{timestamp}_{description}.sql` — Add schema changes

**Internationalization:**
- Web: `web/src/lib/i18n/locales/` — Add translation keys to locale files
- Mobile: `app/lib/i18n/` — Add strings to Flutter localization files

## Special Directories

**`web/src/lib/vendor/`:**
- Purpose: Vendored third-party code (not from npm)
- Generated: No (manually copied)
- Committed: Yes
- Examples: MapLibre plugin variants, custom adapters, compatibility shims
- Rationale: For code patches or unavailable packages

**`web/src/lib/assets/`:**
- Purpose: Static images, SVGs, fonts
- Generated: No
- Committed: Yes
- Subdirectories: `fonts/`, `svgs/` (with `logos/`, `pois/`, `empty_states/`), `images/`

**`db/pb_data/`:**
- Purpose: PocketBase runtime data (SQLite database, uploaded files)
- Generated: Yes (auto-created on startup)
- Committed: No (in `.gitignore`)
- Structure: `pb_data/storage/` for file uploads, `pb_data/` contains `.db` file

**`app/ios/` and `app/android/`:**
- Purpose: Native platform-specific code
- Generated: Partially (managed by Flutter)
- Committed: Yes
- Content: Gradle configuration, Xcode project, platform channels, native permissions

**`data/uploads/`:**
- Purpose: Server-side storage for file uploads (trail photos, etc.)
- Generated: Yes (created on first upload)
- Committed: No (in `.gitignore`)

---

*Structure analysis: 2026-09-06*
