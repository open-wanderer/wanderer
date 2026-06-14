# Technology Stack

**Analysis Date:** 2026-06-10

## Languages

**Primary:**
- **TypeScript** 5.9.3 - Web frontend and API routes (`web/src/**/*.ts`, `web/src/**/*.tsx`)
- **Dart** 3.11.5+ - Flutter mobile app (`app/lib/**/*.dart`)
- **Go** 1.25.0 - Backend services and PocketBase integration (`db/**/*.go`)

**Secondary:**
- **JavaScript** - Build tooling and scripts
- **YAML** - Configuration (`pubspec.yaml`, `docker-compose.yml`, migrations)
- **Markdown** - Documentation

## Runtime

**Web Frontend:**
- **Node.js** 22 (Alpine) - SvelteKit application server
- **npm** - JavaScript/TypeScript package manager
- Lockfile: `web/package-lock.json` present

**Mobile:**
- **Flutter SDK** 3.11.5+ - Cross-platform framework
- **Dart** 3.11.5+ - Language runtime
- **pub** - Dart package manager (`app/pubspec.yaml`)

**Backend:**
- **Go** 1.25.0 - Compiled binaries
- Statically compiled (`CGO_ENABLED=0`) for minimal dependencies

## Frameworks

**Core Web:**
- **SvelteKit** 2.60.1 - Full-stack framework with Node.js adapter (`web/svelte.config.js`)
- **Svelte** 5.55.7 - Component framework
- **Vite** 8.0.9 - Build tool and dev server (`web/vite.config.ts`)

**Mobile:**
- **Flutter** 3.11.5 - Cross-platform mobile framework
- **Flutter Riverpod** 3.3.1 - State management with codegen (`riverpod_annotation`, `riverpod_generator`)
- **Go Router** 17.2.1 - Navigation and routing
- **Flutter Material Design** - UI components

**Backend:**
- **PocketBase** 0.38.0 - BaaS database and auth backend (`github.com/pocketbase/pocketbase`)
- **Meilisearch** v1.36.0 - Full-text search engine (Docker service)

**Styling & UI:**
- **TailwindCSS** 4.2.4 - Utility-first CSS (`web/tailwind.config.js`)
- **PostCSS** 8.5.6 - CSS processing (`web/postcss.config.js`)
- **Autoprefixer** 10.4.24 - Browser vendor prefixes

**Data & Content:**
- **Felte** 1.3.0 - Form state management (web)
- **Zod** 3.24.1 - TypeScript schema validation
- **TipTap** 2.14.0 - Rich text editor (`@tiptap/core`, `@tiptap/starter-kit`)
- **Marked** 17.0.4 - Markdown parsing

**Mapping & Geospatial:**
- **MapLibre GL** 4.7.1 - Vector map rendering
- **Flutter Map** 8.3.0 - OSM-based mapping (mobile)
- **Flutter Map Location Marker** 10.0.2 - User location on map
- **Flutter Map Marker Cluster** 8.2.2 - Marker clustering
- **Vector Map Tiles** 10.0.0-beta.2 - Vector tile rendering (mobile)
- **Vector Tile Renderer** 7.0.0-beta.1 - Tile rendering (mobile)
- **PMTiles** 1.2.0 - Cloud-optimized tile format

**3D & Graphics:**
- **Three.js** 0.183.1 - 3D rendering
- **Threlte** 8.5.9 - Svelte components for Three.js
- **Canvg** 4.0.3 - SVG rendering

**Charts & Visualizations:**
- **Chart.js** 4.5.1 - Charts (web)
- **FL Chart** 1.2.0 - Charts (mobile)
- **Photoswipe** 5.4.3 - Image gallery

**QR Code & Sharing:**
- **QR Flutter** 4.1.0 - QR code generation (mobile)
- **QRCode** 1.4.4 - QR code generation (web)
- **Share Plus** 13.1.0 - Native share sheet (mobile)

**Testing:**
- **Playwright** 1.58.2 - E2E testing (`web/playwright.config.ts`)
- **Vitest** 4.1.4 - Unit testing (`web/vite.config.ts`)
- **Flutter Test** - Built-in testing framework

**Build & Development:**
- **@sveltejs/kit** 2.60.1 - SvelteKit framework
- **@sveltejs/adapter-node** 5.5.2 - Production Node.js adapter
- **@sveltejs/enhanced-img** 0.10.4 - Image optimization
- **@sveltejs/vite-plugin-svelte** 7.0.0 - Vite integration
- **@tailwindcss/vite** 4.2.4 - Tailwind Vite plugin
- **svelte-check** 4.3.6 - TypeScript/Svelte compiler validation

**Internationalization:**
- **Svelte-i18n** 4.0.0 - Web localization
- **Intl** - Flutter localization
- **Flutter Localizations** - Material localization (mobile)

**Code Generation (Mobile):**
- **Freezed** 3.2.5 - Immutable class generation
- **JSON Serializable** 6.13.0 - JSON serialization
- **Build Runner** 2.13.1 - Code generation runner
- **Riverpod Generator** 4.0.3 - Riverpod provider codegen

**Icons:**
- **Font Awesome Flutter** 11.0.0 - Icons (mobile)
- **@fortawesome/fontawesome-free** 7.1.0 - Icons (web)
- **Flutter SVG** 2.2.4 - SVG rendering (mobile)

**Data Storage (Mobile):**
- **ObjectBox** 5.3.1 - NoSQL object database
- **ObjectBox Flutter Libs** 5.3.1 - Native libraries
- **ObjectBox Generator** 5.3.1 - Code generation

**HTTP & Networking:**
- **Dio** 5.9.2 - HTTP client (mobile)
- **Cookie Jar** 4.0.9 - Cookie management (mobile)
- **Dio Cookie Manager** 3.4.0 - Cookie persistence

**Utilities & Helpers:**
- **GPX** 2.3.0 - GPX file parsing (mobile)
- **Latlong2** 0.9.1 - Geolocation types
- **Geolocator** 13.0.2 - Device geolocation (mobile)
- **Path Provider** 2.1.5 - File paths (mobile)
- **Duration** 4.0.3 - Duration handling (mobile)
- **Timeago** 3.7.1 - Relative time formatting
- **Textfield Tags** 3.0.1 - Tag input component (mobile)
- **Form Builder Validators** 11.3.0 - Form validation (mobile)
- **Flutter Form Builder** 10.3.0+2 - Form building (mobile)
- **Flutter HTML** 3.0.0 - HTML rendering (mobile)

**Geospatial Utilities:**
- **@turf/distance** 7.3.3 - Distance calculations
- **@turf/destination** 7.3.3 - Coordinate calculations
- **ngeohash** 0.6.3 - Geohash encoding
- **Nouislider** 15.7.1 - Range slider UI

**PDF & Document Generation:**
- **jsPDF** 4.2.1 - PDF generation
- **PDFKit** 0.17.2 - PDF generation (Node.js)
- **JSZip** 3.10.1 - ZIP archive creation

**Security & Crypto:**
- **Crypto Random String** 5.0.0 - Cryptographic random generation

**Data Formats & Parsing:**
- **@xmldom/xmldom** 0.8.12 - XML DOM handling
- **Isomorphic XML2JS** 0.1.3 - XML to JSON conversion
- **JSON Diff TS** 4.8.2 - JSON diffing
- **ActivityPub Types** 1.1.0 - ActivityPub protocol types (`web/src/lib/models/activitypub/`)

**Backend Dependencies:**
- **Meilisearch Go** 0.36.2 - Search client for Go backend
- **DBX** 1.12.0 - Database abstraction (PocketBase)
- **GPXGO** 1.4.0 - GPX parsing (Go)
- **JSONld** 0.0.0-20250905102310-8480b0fe24d9 - JSON-LD for ActivityPub
- **Go-AP Activitypub** 0.0.0-20250905102448-e9df599e4528 - ActivityPub library
- **Go-Fed HTTPSig** 1.1.0 - HTTP signatures for federation
- **Bluemonday** 1.0.27 - HTML sanitization
- **JWT Go** 5.3.1 - JSON Web Token signing
- **Polyline** 1.1.1 - Polyline encoding (geospatial)

## Key Dependencies

**Critical (Web):**
- `pocketbase` 0.26.8 - Backend database and auth (`web/src/lib/pocketbase.ts`)
- `meilisearch` 0.57.0 - Search engine client (`web/src/hooks.server.ts`)
- `zod` 3.24.1 - Input validation on all API endpoints
- `@tiptap/core` 2.14.0 - Rich text editing

**Critical (Mobile):**
- `flutter_riverpod` 3.3.1 - State management and dependency injection
- `dio` 5.9.2 - HTTP requests to backend
- `go_router` 17.2.1 - Navigation
- `objectbox` 5.3.1 - Local cache and offline data
- `qr_flutter` 4.1.0 - QR code generation (profile sharing)
- `share_plus` 13.1.0 - Native share sheet

**Critical (Backend):**
- `github.com/pocketbase/pocketbase` 0.38.0 - Core BaaS framework
- `github.com/meilisearch/meilisearch-go` 0.36.2 - Search integration
- ActivityPub libraries for federation

## Configuration Files

**TypeScript:**
- `web/tsconfig.json` - Strict mode enabled, `skipLibCheck: true`, bundler resolution

**SvelteKit:**
- `web/svelte.config.js` - Node.js adapter for production (`@sveltejs/adapter-node`)
- `web/vite.config.ts` - Vite configuration with SvelteKit plugin

**Styling:**
- `web/tailwind.config.js` - Tailwind CSS configuration
- `web/postcss.config.js` - PostCSS configuration with Tailwind

**Build & Deployment:**
- `web/Dockerfile` - Node.js 22 Alpine, multi-stage build, exposes port 3000
- `db/Dockerfile` - Go 1.25.0, statically compiled, exposes port 8090
- `docker-compose.yml` - Production services (search, db, web)
- `docker/docker-compose.dev.yml` - Development services
- `docker/docker-compose.prod.yml` - Production services with image overrides

**Testing:**
- `web/playwright.config.ts` - E2E test configuration
- `app/pubspec.yaml` - Flutter test configuration (flutter_test SDK)

**Mobile:**
- `app/pubspec.yaml` - Flutter app manifest, Dart 3.11.5+
- Asset configuration: SVG assets (`assets/svgs/`), custom fonts (IBMPlexSans, Noto Sans)

**Environment Variables (Docker):**
- `MEILI_URL` - Meilisearch server URL
- `MEILI_MASTER_KEY` - Meilisearch authentication
- `PUBLIC_POCKETBASE_URL` - Backend database URL
- `POCKETBASE_ENCRYPTION_KEY` - 32-byte encryption key (required)
- `ORIGIN` - CORS origin for backend
- `UPLOAD_FOLDER` - File upload directory
- `UPLOAD_USER` / `UPLOAD_PASSWORD` - Upload authentication (optional)
- `OVERPASS_API_URL` - Overpass API endpoint
- `VALHALLA_URL` - Valhalla routing service
- `NOMINATIM_URL` - Nominatim geocoding service
- `PUBLIC_DISABLE_SIGNUP` - Disable user registration
- `PUBLIC_MAP_MAX_POLYLINES` - Map performance limit
- `BODY_SIZE_LIMIT` - HTTP request size limit
- `PUBLIC_OVERPASS_API_URL` - Public Overpass endpoint

## Platform Requirements

**Development (Local):**
- Node.js 22.x with npm (engine-strict enabled via `.npmrc`)
- Flutter SDK 3.11.5+ with Dart 3.11.5+
- Go 1.25.0 (for backend development)
- Git (version control)

**Production (Docker):**
- **Web:** Node.js 22 Alpine container (flomp/wanderer-web)
- **Search:** Meilisearch v1.36.0 container
- **Database:** Custom flomp/wanderer-db container (PocketBase-based Go backend)
- **Storage:** Mapped volume for file uploads (`data/uploads`)
- **Network:** Docker bridge network named "wanderer"

**Mobile Deployment:**
- iOS 12+ (via Flutter)
- Android API 21+ (via Flutter)

---

*Stack analysis: 2026-06-10*
