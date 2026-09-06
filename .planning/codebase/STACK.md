# Technology Stack

**Analysis Date:** 2026-09-06

## Languages

**Primary:**
- **TypeScript** 6.0.3 - Web frontend and backend (`web/src/**/*.ts`, `web/src/**/*.tsx`)
- **Go** 1.25.0 - Backend database and federation logic (`db/`)
- **Dart** ^3.11.5 - Flutter mobile app (`app/lib/**/*.dart`)
- **JavaScript** - Build tooling and runtime scripts (`web/watcher.js`, `scripts/`)

**Secondary:**
- **Svelte** 5.56.4 - UI template language (`web/src/**/*.svelte`)
- **YAML** - Configuration (`docker-compose.yml`, `pubspec.yaml`)
- **Markdown** - Documentation (`CHANGELOG.md`, `CONTRIBUTING.md`)

## Runtime

**Environment:**
- **Node.js** 22 (alpine) - Web backend and build system; specified in `web/Dockerfile` and `package.json` (`npm` engine requirement)
- **Flutter SDK** ^3.11.5 - Mobile app runtime
- **Dart** ^3.11.5 - Flutter language runtime
- **Go** 1.25.0 - Backend service runtime (PocketBase + federation hooks)

**Package Managers:**
- **npm** 22.x - JavaScript/TypeScript package management (`web/package.json`, `web/package-lock.json`)
  - Engine-strict mode enabled in `web/.npmrc`
- **pub** - Dart/Flutter package management (`app/pubspec.yaml`)
- **go mod** - Go module management (`db/go.mod`, `db/go.sum`)

## Frameworks

**Core Web:**
- **SvelteKit** 2.68.0 - Full-stack web framework with Node.js adapter (`web/svelte.config.js`)
- **Svelte** 5.56.4 - Reactive UI component framework
- **Vite** 8.1.1 - Build tool and dev server (`web/vite.config.ts`)

**Core Mobile:**
- **Flutter** 3.11.5 - Cross-platform mobile framework (`app/pubspec.yaml`)
- **Dart** - Flutter's language
- **Go Router** 17.2.1 - Mobile navigation and routing
- **Flutter Riverpod** 3.3.1 - State management for mobile

**Documentation:**
- **Astro** 7.0.4 - Static site generator for docs (`docs/astro.config.mjs`, `docs/package.json`)
- **Starlight** 0.41.1 - Astro documentation theme
- **Starlight OpenAPI** 0.26.0 - OpenAPI schema documentation

**Styling & UI:**
- **TailwindCSS** 4.3.2 - Utility-first CSS framework (`web/tailwind.config.js`, `docs/`)
- **@tailwindcss/vite** 4.3.2 - Tailwind Vite plugin
- **@tailwindcss/typography** 0.5.20 - Prose styling
- **PostCSS** 8.5.16 - CSS processing (`web/postcss.config.js`)
- **Autoprefixer** 10.5.2 - CSS vendor prefixing

**Forms & Input:**
- **Felte** 1.3.0 - Form state management (`web/package.json`)
- **@felte/validator-zod** 1.0.18 - Felte Zod integration
- **flutter_form_builder** 10.3.0+2 - Flutter forms

**Maps & Geospatial:**
- **MapLibre GL** 5.24.0 - Vector map rendering and interaction
- **maplibre** 0.3.5 - Flutter MapLibre wrapper
- **Flutter Map** 8.3.0 - OSM-based mapping for Flutter
- **Flutter Map Location Marker** 10.0.2 - User location on map
- **Flutter Map Marker Cluster** 8.2.2 - Marker clustering
- **PMTiles** 1.2.0 - Cloud-optimized tile format (downloaded in `db/Dockerfile`)
- **@turf/distance** 7.3.5 - Geospatial distance calculations
- **@turf/destination** 7.3.5 - Geospatial coordinate calculations
- **ngeohash** 0.6.3 - Geohash encoding/decoding

**Rich Text Editing:**
- **@tiptap/core** 3.31.0 - Rich text editor core
- **@tiptap/starter-kit** 3.31.0 - TipTap standard extensions
- **@tiptap/extension-heading** 3.31.0 - Heading support
- **@tiptap/extension-link** 3.31.0 - Link handling
- **@tiptap/extension-mention** 3.31.0 - User mentions
- **@tiptap/extension-placeholder** 3.31.0 - Placeholder text
- **@tiptap/pm** 3.31.0 - ProseMirror state management
- **@tiptap/suggestion** 3.31.0 - Autocomplete/suggestion
- **tiptap_flutter** - Flutter TipTap integration (vendored: `app/vendor/tiptap_flutter`)

**3D Graphics:**
- **Three.js** 0.185.0 - 3D rendering library
- **@threlte/core** 8.5.16 - Svelte Three.js bindings
- **@threlte/extras** 9.21.0 - Additional Three.js components

**Data Visualization:**
- **Chart.js** 4.5.1 - Chart and graph visualization
- **chartjs-plugin-crosshair** 2.0.0 - Crosshair plugin for charts
- **chartjs-plugin-zoom** 2.2.0 - Zoom plugin for charts
- **FL Chart** 1.2.0 - Flutter charting library

**File & Document Processing:**
- **jszip** 3.10.1 - ZIP archive creation and reading
- **jspdf** 4.2.1 - PDF generation (JavaScript)
- **pdfkit** 0.19.1 - PDF generation library
- **canvg** 4.0.3 - SVG to canvas/PNG rendering
- **qrcode** 1.5.4 - QR code generation
- **qr_flutter** 4.1.0 - Flutter QR code generation
- **marked** 18.0.5 - Markdown parsing
- **heic2any** 0.0.4 - HEIC/HEIF image conversion

**Image & Media:**
- **@xmldom/xmldom** 0.9.12 - XML DOM for SVG/document processing
- **@sveltejs/enhanced-img** 0.11.0 - Image optimization
- **photoswipe** 5.4.4 - Image gallery and lightbox
- **file_picker** 11.0.2 - File selection (Flutter)
- **image_picker** 1.2.2 - Photo/image selection (Flutter)
- **flutter_svg** 2.2.4 - SVG rendering (Flutter)
- **cached_network_image** 3.4.1 - Image caching (Flutter)
- **exif** 3.3.0 - EXIF data extraction (Flutter)

**Internationalization:**
- **svelte-i18n** 4.0.1 - Frontend i18n
- **intl** - Flutter localization

**State Management & Local Storage:**
- **SvelteKit stores** - Built-in reactive stores (`web/src/lib/stores/`)
- **flutter_riverpod** 3.3.1 - Flutter state management
- **riverpod_annotation** 4.0.2 - Riverpod code generation
- **riverpod_generator** 4.0.3 - Dev dependency for Riverpod
- **objectbox** 5.3.1 - Mobile local database (Flutter)
- **objectbox_flutter_libs** 5.3.1 - Native libraries for ObjectBox
- **objectbox_generator** 5.3.1 - ObjectBox code generation

**HTTP & Networking:**
- **dio** 5.9.2 - HTTP client for Flutter
- **dio_cookie_manager** 3.4.0 - Cookie management for Dio
- **cookie_jar** 4.0.9 - HTTP cookie storage

**Geolocation & Sensors:**
- **geolocator** 14.0.0 - Device geolocation (Flutter)
- **flutter_rotation_sensor** 0.2.0 - Device orientation/rotation

**Data Validation & Parsing:**
- **zod** 3.24.1 - TypeScript schema validation
- **freezed_annotation** 3.1.0 - Code generation for immutable classes (Flutter)
- **freezed** 3.2.5 - Freezed code generation (dev)
- **json_annotation** 4.11.0 - JSON serialization (Flutter)
- **json_serializable** 6.13.0 - JSON serialization code gen (Flutter dev)
- **xml** 6.3.0 - XML parsing (Flutter)
- **html** 0.15.6 - HTML parsing (Flutter)
- **flutter_html** 3.0.0 - HTML rendering (Flutter)

**Miscellaneous Web:**
- **activitypub-types** 1.1.0 - ActivityPub protocol type definitions
- **canvas-confetti** 1.9.4 - Confetti animation effects
- **crypto-random-string** 5.0.0 - Cryptographic random string generation
- **instead** 1.0.3 - String replacement utilities
- **isomorphic-xml2js** 0.1.3 - XML to JSON conversion
- **json-diff-ts** 4.10.4 - JSON difference comparison
- **nouislider** 15.8.1 - Range slider UI component
- **supercluster** 9.0.0 - Point clustering
- **timeago** 3.7.1 - Relative time formatting
- **latlong2** 0.9.1 - Latitude/longitude calculations (Flutter)
- **path_provider** 2.1.5 - File system paths (Flutter)
- **disk_space_2** 1.0.12 - Disk space info (Flutter)
- **duration** 4.0.3 - Duration formatting (Flutter)

**Testing & Code Quality:**
- **Playwright** 1.58.2 - E2E testing (`web/playwright.config.ts`)
- **@playwright/test** 1.61.1 - Playwright test runner
- **Vitest** 4.1.9 - Unit testing framework (`web/vite.config.ts`)
- **flutter_test** - Flutter testing framework (SDK)
- **flutter_lints** 6.0.0 - Flutter linting rules
- **svelte-check** 4.7.1 - Svelte compiler type checking
- **@astrojs/check** 0.9.9 - Astro type checking
- **build_runner** 2.13.1 - Code generation runner (Flutter dev)
- **json_to_arb** 0.1.14 - JSON to ARB conversion (dev)

**Development Tools:**
- **sveltekit-openapi-generator** 0.1.6 - OpenAPI schema generation from routes
- **chokidar** 5.0.0 - File system watching
- **@types/node** 26.0.1 - Node.js type definitions
- **@types/chart.js** 4.0.1 - Chart.js types
- **@types/supercluster** 7.1.3 - Supercluster types
- **@types/three** 0.185.0 - Three.js types
- **@types/canvas-confetti** 1.9.0 - Canvas confetti types
- **tslib** 2.8.1 - TypeScript helpers
- **sharp** 0.35.2 - Image processing (for Astro)
- **flutter_launcher_icons** 0.14.4 - App icon generation
- **form_builder_validators** 11.3.0 - Form validation rules
- **receive_sharing_intent** 1.8.1 - Handle app sharing (Flutter)
- **share_plus** 12.0.2 - Share functionality (Flutter)
- **url_launcher** 6.3.2 - URL launching (Flutter)
- **pointer_interceptor** 0.10.1 - Pointer event handling (Flutter)
- **skeletonizer** 2.1.3 - Loading skeleton UI (Flutter)
- **textfield_tags** 3.0.1 - Tag input field (Flutter)
- **tracelet** 3.5.0 - Trail recording library (Flutter)
- **gpx** 2.3.0 - GPX file parsing (Flutter)
- **font_awesome_flutter** 11.0.0 - Font Awesome icons (Flutter)
- **path** 1.9.1 - Path utilities (Flutter)
- **meta** 1.18.0 - Metadata annotations
- **collection** 1.19.1 - Collection utilities (Flutter)
- **vector_map_tiles** 10.0.0-beta.2 - Vector tile rendering (Flutter)
- **vector_tile_renderer** 7.0.0-beta.1 - Vector tile renderer (Flutter)

## Key Backend Dependencies (Go)

- **pocketbase** 0.38.0 - Backend database and ORM (`db/main.go`, `db/routes/`)
- **dbx** 1.12.0 - Database abstraction layer
- **go-ap/jsonld** - JSON-LD for ActivityPub
- **go-ap/activitypub** - ActivityPub types and utilities
- **go-fed/httpsig** - HTTP Signature verification/signing (federation)
- **meilisearch-go** 0.36.2 - Meilisearch client for Go
- **safeurl** 0.2.5 - URL safety validation
- **mimetype** 1.4.13 - MIME type detection
- **gpxgo** 1.4.0 - GPX file parsing
- **go-xsd-duration** - ISO 8601 duration parsing
- **bluemonday** 1.0.27 - HTML sanitization
- **cobra** 1.10.2 - CLI framework
- **ozzo-validation** 4.3.0 - Input validation
- **cast** 1.10.0 - Type casting
- **go-polyline** 1.1.1 - Polyline encoding/decoding
- **extism** 1.7.1 - WebAssembly plugin system
- **observe-sdk** - OpenTelemetry observability

## Configuration

**Environment Variables (Web):**
- `MEILI_URL` - Meilisearch service URL (default: http://127.0.0.1:7700)
- `MEILI_MASTER_KEY` - Meilisearch master API key
- `PUBLIC_POCKETBASE_URL` - PocketBase backend URL (exposed to client: default: http://127.0.0.1:8090)
- `PUBLIC_DISABLE_SIGNUP` - Disable user registration (default: false)
- `VALHALLA_URL` - Valhalla routing service URL (defaults to https://valhalla1.openstreetmap.de if not set)
- `NOMINATIM_URL` - Nominatim geocoding service URL (defaults to https://nominatim.openstreetmap.org if not set)
- `OVERPASS_API_URL` - Overpass API URL (defaults to https://overpass-api.de if not set)
- `UPLOAD_FOLDER` - Directory for file uploads (default: /app/uploads)
- `UPLOAD_USER` - Optional HTTP basic auth username for upload endpoint
- `UPLOAD_PASSWORD` - Optional HTTP basic auth password for upload endpoint
- `PUBLIC_MAP_MAX_POLYLINES` - Maximum polylines on map display (default: 100)
- `ORIGIN` - Server origin URL (e.g., http://localhost:3000)
- `BODY_SIZE_LIMIT` - Request body size limit (default: unlimited in Docker)

**Environment Variables (Backend/Go):**
- `MEILI_URL` - Meilisearch service URL
- `MEILI_MASTER_KEY` - Meilisearch master API key
- `POCKETBASE_ENCRYPTION_KEY` - PocketBase encryption key for sensitive data
- `ORIGIN` - Server origin URL for federation and links

**Configuration Files:**
- `web/svelte.config.js` - SvelteKit configuration with Node.js adapter
- `web/tsconfig.json` - TypeScript configuration (strict mode enabled, bundler resolution)
- `web/vite.config.ts` - Vite build configuration with SvelteKit plugin
- `web/tailwind.config.js` - TailwindCSS configuration
- `web/postcss.config.js` - PostCSS configuration
- `web/playwright.config.ts` - E2E testing configuration
- `web/.npmrc` - npm configuration (engine-strict enabled)
- `web/openapi.config.js` - OpenAPI schema generation
- `app/pubspec.yaml` - Flutter/Dart dependencies
- `docs/astro.config.mjs` - Documentation site configuration
- `docker-compose.yml` - Local development orchestration (search, db, web)
- `docker/docker-compose.dev.yml` - Development deployment
- `docker/docker-compose.prod.yml` - Production deployment

## Platform Requirements

**Development:**
- Node.js 22.x LTS
- npm 10+ (with engine-strict enabled)
- Flutter SDK 3.11.5+
- Dart 3.11.5+
- Go 1.25.0+
- Git

**Web Production:**
- Node.js 22 Alpine container (specified in `web/Dockerfile`)
- 512MB+ RAM minimum
- File system for uploads directory

**Database & Search:**
- Meilisearch v1.36.0+ container (or v1.11.3 in dev/prod compose files)
- PocketBase v0.26.8+ (flomp/wanderer-db or ghcr.io/open-wanderer/wanderer-db)
- SQLite (embedded in PocketBase)

**Mobile (iOS):**
- iOS 12.0+
- Xcode 16.2+

**Mobile (Android):**
- Android SDK API 21+
- Android Studio or Flutter CLI

**External Services (configurable):**
- Nominatim geocoding service (default: nominatim.openstreetmap.org)
- Valhalla routing service (default: valhalla1.openstreetmap.de)
- Overpass API (default: overpass-api.de)
- OpenStreetMap tile servers

---

*Stack analysis: 2026-09-06*
