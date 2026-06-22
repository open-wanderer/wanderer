# Codebase Structure

**Analysis Date:** 2026-06-07

## Directory Layout

```
wanderer/
├── web/                          # SvelteKit frontend application
│   ├── src/
│   │   ├── routes/              # SvelteKit file-based routing
│   │   │   ├── +page.svelte     # Homepage
│   │   │   ├── +page.ts         # Homepage data loading
│   │   │   ├── +layout.svelte   # Root layout wrapper
│   │   │   ├── +layout.server.ts # Server-side layout data
│   │   │   ├── api/v1/          # API endpoints (/api/v1/*)
│   │   │   ├── map/             # Map view (/map)
│   │   │   ├── trail/           # Trail routes (/trail/*)
│   │   │   ├── trails/          # Trails list (/trails)
│   │   │   ├── settings/        # Settings (/settings/*)
│   │   │   ├── profile/         # Profile (/profile/*)
│   │   │   ├── auth/            # Auth flows (/auth/*)
│   │   │   ├── login/           # Login (/login)
│   │   │   ├── register/        # Registration (/register)
│   │   │   ├── lists/           # Lists management (/lists)
│   │   │   └── .well-known/     # WebFinger/federation (.well-known/*)
│   │   ├── lib/
│   │   │   ├── stores/          # Svelte stores (state management)
│   │   │   ├── models/          # TypeScript type definitions
│   │   │   │   └── api/         # API schema definitions
│   │   │   ├── components/      # Reusable Svelte components
│   │   │   │   ├── base/        # Base UI components
│   │   │   │   ├── trail/       # Trail-specific components
│   │   │   │   ├── map/         # Map components
│   │   │   │   ├── settings/    # Settings components
│   │   │   │   └── ...
│   │   │   ├── server/          # Server-side utilities
│   │   │   ├── util/            # Shared utilities
│   │   │   ├── i18n/            # Internationalization
│   │   │   ├── vendor/          # Third-party code
│   │   │   ├── assets/          # Static assets
│   │   │   ├── css/             # Global styles
│   │   │   └── pocketbase.ts    # PocketBase client singleton
│   │   └── css/
│   ├── package.json             # Dependencies
│   ├── tsconfig.json            # TypeScript config
│   ├── svelte.config.js         # SvelteKit config
│   └── vite.config.js           # Vite bundler config
├── app/                         # Flutter mobile application
│   ├── lib/
│   │   ├── models/              # Data models (Dart)
│   │   ├── provider/            # Riverpod state providers
│   │   ├── routes/              # Navigation structure
│   │   ├── services/            # API clients and business logic
│   │   ├── components/          # Flutter widgets
│   │   ├── entities/            # ObjectBox entities
│   │   ├── util/                # Utilities
│   │   ├── theme/               # UI theming
│   │   ├── i18n/                # Internationalization
│   │   └── main.dart            # App entry point
│   ├── pubspec.yaml             # Dart dependencies
│   └── android/, ios/           # Native platform code
├── db/                          # Go backend (PocketBase extensions)
│   ├── main.go                  # Entry point
│   ├── go.mod                   # Go dependencies
│   ├── routes/                  # HTTP route handlers
│   ├── hooks/                   # Database hooks (before/after operations)
│   ├── migrations/              # Database schema migrations
│   ├── federation/              # ActivityPub implementation
│   ├── integrations/            # External service integrations (Strava, etc.)
│   ├── services/                # Business logic services
│   ├── util/                    # Go utilities
│   ├── tests/                   # Integration tests
│   ├── pb_data/                 # Runtime database (SQLite)
│   └── pocketbase*              # PocketBase binary (compiled Go)
├── search/                      # Meilisearch integration service
│   ├── src/
│   └── package.json
├── docs/                        # Documentation site (Astro)
│   ├── src/
│   └── package.json
├── data/                        # Static/seed data files
├── docker/                      # Docker build files
├── docker-compose.yml           # Multi-container orchestration
├── Makefile                     # Build commands
├── README.md                    # Project overview
├── CHANGELOG.md                 # Version history
├── CONTRIBUTING.md              # Contribution guidelines
└── .claude/                     # Claude Code configuration
    ├── skills/                  # Project skills (if any)
    └── settings.local.json
```

## Directory Purposes

**web/ - Frontend:**
- Purpose: SvelteKit frontend application serving user interface
- Contains: Routes, components, state management, utilities, styles
- Key patterns: File-based routing, server-side data loading via `load()` functions, Svelte stores for state

**web/src/routes/ - Route Definitions:**
- Purpose: Define URL structure and page logic using SvelteKit's file-based routing
- Contains: `.svelte` for UI, `.ts` for data loading, `.server.ts` for server-only logic
- Pattern: `+page.svelte` renders page, `+page.ts`/`+page.server.ts` loads data, `+layout.svelte` wraps children

**web/src/routes/api/v1/ - API Endpoints:**
- Purpose: RESTful API endpoints that proxy/process requests to PocketBase backend
- Contains: GET/POST/PUT/DELETE handlers for trails, users, comments, etc.
- Pattern: `+server.ts` files handle HTTP methods; use `handleError()` for consistency

**web/src/lib/stores/ - State Management:**
- Purpose: Centralized reactive state via Svelte stores
- Contains: Store definitions and async fetch functions (e.g., `trails_index()`, `profile_store()`)
- Pattern: Writable stores hold current state; async functions fetch, parse, update stores; return data to components

**web/src/lib/models/ - Type Definitions:**
- Purpose: TypeScript/runtime models for all domain entities
- Contains: Classes with constructors, expand fields for relational data, type aliases for filters/results
- Pattern: Domain models (Trail, User, Waypoint, etc.); schema models for API validation

**web/src/lib/components/ - UI Components:**
- Purpose: Reusable Svelte components for building pages
- Contains: Base UI elements, feature-specific components (TrailCard, MapWithElevation, etc.)
- Pattern: Props for inputs, slots for composition; event dispatch for communication

**web/src/lib/components/base/ - Base UI Library:**
- Purpose: Fundamental reusable components
- Contains: Button, TextInput, Modal, Select, Dropdown, etc.
- Pattern: Tailwind-styled, accessibility-focused, composable

**web/src/lib/util/ - Shared Utilities:**
- Purpose: Helper functions for common tasks
- Contains: Geospatial utilities (geojson_util.ts, maplibre_util.ts), formatting (format_util.ts), file handling, etc.
- Pattern: Functional utilities with minimal dependencies; used across components and stores

**web/src/lib/server/ - Server-only Code:**
- Purpose: Backend utilities accessed in `+page.server.ts` and API routes
- Contains: Nominatim client, Overpass client, Valhalla routing, HTTP utilities
- Pattern: Server-only imports via `$lib/server`; not bundled to client

**web/src/css/ - Styling:**
- Purpose: Global CSS and component-level styles
- Contains: Tailwind CSS setup, theme variables, component styles
- Pattern: app.css (global), components.css (component-specific), theme.css (theming)

**app/ - Mobile Frontend:**
- Purpose: Flutter mobile application
- Contains: Dart models, providers (state), routes, widgets, services
- Pattern: Riverpod for state management; ObjectBox for local storage; freezed for immutable models

**db/ - Backend:**
- Purpose: Go backend extending PocketBase with custom logic
- Contains: Route handlers, database hooks, migrations, federation, integrations
- Pattern: Custom routes registered in main.go; hooks intercept before/after DB operations

**db/migrations/ - Schema:**
- Purpose: Database schema evolution
- Contains: Numbered migration files that create/alter collections
- Pattern: SemVer naming (e.g., 1672531200_initial_schema.go)

**db/federation/ - ActivityPub:**
- Purpose: Decentralized social federation
- Contains: Actor representations, activity handling, inbox/outbox logic
- Pattern: ActivityPub types and handlers; integrates with PocketBase events

**db/hooks/ - Event Handlers:**
- Purpose: Trigger custom logic on database operations
- Contains: Before/after hooks for create/update/delete events
- Pattern: Register in main.go; called by PocketBase during record operations

**docs/ - Documentation:**
- Purpose: User-facing documentation website
- Contains: Astro site with deployment, feature, and API documentation
- Pattern: Markdown-based content; SSG for static hosting

**search/ - Search Service:**
- Purpose: Meilisearch integration for full-text search
- Contains: Indexing logic, search handlers
- Pattern: Standalone service; called from backend and frontend

## Key File Locations

**Entry Points:**
- Web: `web/src/routes/+layout.svelte` - Root layout component
- Web data: `web/src/routes/+layout.server.ts` - Global data loading (user, settings, notifications)
- Mobile: `app/lib/main.dart` - Flutter app initialization
- Backend: `db/main.go` - Go server and PocketBase setup

**Configuration:**
- Web: `web/tsconfig.json`, `web/svelte.config.js`, `web/package.json`
- Mobile: `app/pubspec.yaml`
- Backend: `db/go.mod`, `docker-compose.yml`
- Root: `.github/workflows/` (CI/CD), `Makefile`

**Core Logic:**
- Trail management: `web/src/lib/stores/trail_store.ts`, `db/routes/trail*`
- User authentication: `web/src/lib/stores/user_store.ts`, `db/routes/user*`
- Search: `web/src/lib/stores/search_store.ts`, `db/routes/search*`
- Map interactions: `web/src/routes/map/+page.svelte`, `web/src/lib/components/trail/map_with_elevation_maplibre.svelte`
- API schema: `web/src/lib/models/api/*_schema.ts`

**Testing:**
- Web unit: `web/**/*.test.ts`, `web/**/*.spec.ts`
- Web integration: `web/tests/` (Playwright)
- Backend: `db/tests/`
- Mobile: Unit tests in `app/test/`

## Naming Conventions

**Files:**
- Svelte components: PascalCase (e.g., `TrailCard.svelte`, `MapWithElevation.svelte`)
- Utilities: snake_case (e.g., `api_util.ts`, `file_util.ts`, `geojson_util.ts`)
- Routes: kebab-case with brackets for params (e.g., `[id]/`, `[handle]/`)
- Store files: snake_case ending in `_store.ts` or `.svelte.ts` (e.g., `trail_store.ts`, `toast_store.svelte.ts`)
- Models: PascalCase class names in files (e.g., `Trail` in `trail.ts`)
- API schemas: snake_case ending in `_schema.ts` (e.g., `trail_schema.ts`)

**Directories:**
- Components: PascalCase (e.g., `TrailCard/`, `base/`, `map/`)
- Utilities: snake_case (e.g., `util/`, `vendor/`)
- Routes: kebab-case (e.g., `trail/`, `trail-like/`, `api/v1/`)
- Stores: `stores/` directory with snake_case files
- Models: `models/` directory with snake_case files

## Where to Add New Code

**New Feature (e.g., Trail Comments):**
- **Model:** `web/src/lib/models/comment.ts` (define Comment type)
- **Store:** `web/src/lib/stores/comment_store.ts` (async fetch/create/update functions)
- **Component:** `web/src/lib/components/comment/comment_list.svelte` (UI)
- **API endpoint:** `web/src/routes/api/v1/comment/+server.ts` (proxy to backend)
- **Backend route:** `db/routes/comment.go` (business logic, if needed)
- **Schema:** `web/src/lib/models/api/comment_schema.ts` (validation with Zod)

**New Route/Page (e.g., /trails/trending):**
- Create directory: `web/src/routes/trails/trending/`
- Add: `+page.svelte` (UI), `+page.ts` (data loading), `+page.server.ts` (if server-only)
- Store function: Add to existing store or create `web/src/lib/stores/trending_store.ts`
- Component usage: Use existing components or create new in `web/src/lib/components/`

**New Component:**
- Location: `web/src/lib/components/[category]/ComponentName.svelte`
- Pattern: Export props interface, use Svelte 5 `$props()`, dispatch events with `createEventDispatcher()`
- Styling: Use Tailwind classes; import global CSS if needed

**New Utility:**
- Location: `web/src/lib/util/[domain]_util.ts` (e.g., `elevation_util.ts`)
- Pattern: Export pure functions; no side effects
- Examples: `web/src/lib/util/geojson_util.ts`, `web/src/lib/util/format_util.ts`

**New Store:**
- Location: `web/src/lib/stores/[domain]_store.ts` or `.svelte.ts`
- Pattern: Export writable stores and async functions that update them
- Error handling: Use try/catch; return defaults on error
- Example: `web/src/lib/stores/trail_store.ts`

**New Backend Route:**
- Location: `db/routes/[resource].go`
- Pattern: Register in `main.go`; implement GET/POST/PUT/DELETE handlers
- Validation: Use schemas from `web/src/lib/models/api/`
- Authentication: Check `r.Request().Context()` for user info

**New API Schema:**
- Location: `web/src/lib/models/api/[resource]_schema.ts`
- Pattern: Zod schema for request validation; used in SvelteKit API routes
- Naming: `[Resource]CreateSchema`, `[Resource]UpdateSchema`

**New Svelte Store Type:**
- Use `svelte/store` exports: `writable()`, `readable()`, `derived()`
- For async state: Use `Writable<Data>` and async fetch functions
- Modern stores (Svelte 5+): Use `.svelte.ts` files and runes

## Special Directories

**node_modules/, .svelte-kit/:**
- Purpose: Generated/installed dependencies
- Generated: Yes
- Committed: No (in .gitignore)

**pb_data/:**
- Purpose: PocketBase SQLite database runtime
- Generated: Yes (created on first run)
- Committed: No (contains user data)

**web/.next/, build/, dist/:**
- Purpose: Build output
- Generated: Yes (via `npm run build`)
- Committed: No

**migrations/ (db/):**
- Purpose: Database schema history
- Generated: No (manually created)
- Committed: Yes (essential for reproducibility)

---

*Structure analysis: 2026-06-07*
