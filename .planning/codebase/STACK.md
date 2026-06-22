# Technology Stack

**Analysis Date:** 2026-06-07

## Languages

**Primary:**
- **TypeScript** 5.9.3 - Web frontend and backend (`web/src/**/*.ts`)
- **JavaScript** - Build tooling and scripts
- **Dart** - Mobile app (`app/lib/**/*.dart`)
- **Svelte** 5.55.7 - UI framework components (`web/src/**/*.svelte`)

**Secondary:**
- **YAML** - Configuration files (`pubspec.yaml`, Docker compose)
- **Markdown** - Documentation (`CHANGELOG.md`, `CONTRIBUTING.md`)

## Runtime

**Environment:**
- **Node.js** 22 (alpine) - Web backend and build system (`web/Dockerfile`)
- **Flutter SDK** ^3.11.5 - Mobile app runtime (`app/pubspec.yaml`)
- **Dart** ^3.11.5 - Mobile language

**Package Manager:**
- **npm** - JavaScript/TypeScript dependencies
- **pub** - Dart/Flutter dependencies (`app/pubspec.yaml`)

## Frameworks

**Core Web:**
- **SvelteKit** 2.60.1 - Full-stack web framework with adapter-node for production (`web/svelte.config.js`)
- **Svelte** 5.55.7 - Component framework
- **Vite** 8.0.9 - Build tool and dev server

**Mobile:**
- **Flutter** 3.11.5 - Cross-platform mobile framework
- **Go Router** 17.2.1 - Mobile navigation and routing
- **Flutter Riverpod** 3.3.1 - State management for mobile

**Web State Management:**
- **SvelteKit stores** - Built-in reactive stores
- **Felte** 1.3.0 - Form state management
- **Svelte-i18n** 4.0.0 - Internationalization

**Web UI/Visualization:**
- **TailwindCSS** 4.2.4 - Utility-first CSS framework (`web/tailwind.config.js`)
- **MapLibre GL** 4.7.1 - Interactive maps and geospatial visualization
- **Three.js** 0.183.1 - 3D rendering library
- **Threlte** 8.5.9 - Svelte component library for Three.js
- **Chart.js** 4.5.1 - Chart visualization
- **Photoswipe** 5.4.3 - Image gallery

**Testing:**
- **Playwright** 1.58.2 - E2E testing (`web/playwright.config.ts`)
- **Vitest** 4.1.4 - Unit testing (`web/vite.config.ts`)

**Build/Dev:**
- **@sveltejs/adapter-node** 5.5.2 - Production Node.js adapter
- **@sveltejs/kit** 2.60.1 - SvelteKit framework
- **@tailwindcss/vite** 4.2.4 - Tailwind Vite plugin
- **sveltekit-openapi-generator** 0.1.5 - Generate OpenAPI schemas from routes

**Mobile UI:**
- **Flutter Material Design** - Material UI components
- **Font Awesome Flutter** 11.0.0 - Icon library
- **FL Chart** 1.2.0 - Charting library
- **Flutter HTML** 3.0.0 - HTML rendering

**Mobile Maps:**
- **Flutter Map** 8.3.0 - OSM-based mapping
- **Flutter Map Location Marker** 10.0.2 - User location on map
- **Flutter Map Marker Cluster** 8.2.2 - Marker clustering
- **Vector Map Tiles** 10.0.0-beta.2 - Vector tile rendering
- **Vector Tile Renderer** 7.0.0-beta.1 - Renders vector tiles
- **PMTiles** 1.2.0 - Cloud-optimized PMTiles format

## Key Dependencies

**Critical:**
- **pocketbase** 0.26.8 - Backend database and authentication (`web/src/lib/pocketbase.ts`, `app/`)
- **meilisearch** 0.57.0 - Full-text search engine (`web/src/hooks.server.ts`)
- **maplibre-gl** 4.7.1 - Vector map rendering
- **zod** 3.24.1 - TypeScript schema validation

**Infrastructure:**
- **@tiptap/core** 2.14.0 - Rich text editor (`web/package.json`)
- **@tiptap/starter-kit** 2.14.0 - TipTap extensions
- **activitypub-types** 1.1.0 - ActivityPub protocol types (`web/src/lib/models/activitypub/`)
- **dio** 5.9.2 - HTTP client for Flutter
- **flutter_form_builder** 10.3.0+2 - Forms for mobile
- **freezed_annotation** 3.1.0 - Code generation for immutable classes
- **json_annotation** 4.11.0 - JSON serialization
- **objectbox** 5.3.1 - Mobile local database
- **geolocator** 13.0.2 - Device geolocation for mobile
- **marked** 17.0.4 - Markdown parsing
- **canvg** 4.0.3 - SVG rendering
- **jszip** 3.10.1 - ZIP archive creation
- **jspdf** 4.2.1 - PDF generation
- **qrcode** 1.4.4 - QR code generation
- **@turf/distance** 7.3.3 - Geospatial distance calculations
- **@turf/destination** 7.3.3 - Geospatial coordinate calculations
- **ngeohash** 0.6.3 - Geohash encoding
- **nouislider** 15.7.1 - Range slider UI component
- **crypto-random-string** 5.0.0 - Cryptographic random string generation
- **intl** - Internationalization for Flutter

**Mobile Local Storage:**
- **objectbox** 5.3.1 - NoSQL object database
- **objectbox_flutter_libs** 5.3.1 - Flutter native libraries
- **cookie_jar** 4.0.9 - HTTP cookie storage

**Typography:**
- **@xmldom/xmldom** 0.8.12 - XML DOM for SVG/document processing
- **pdfkit** 0.17.2 - PDF generation library

**Development Dependencies:**
- **svelte-check** 4.3.6 - Svelte compiler checks
- **@sveltejs/enhanced-img** 0.10.4 - Image optimization
- **postcss** 8.5.6 - CSS processing
- **autoprefixer** 10.4.24 - CSS autoprefixer
- **eslint** - Not visible in config, check for linting setup
- **typescript** 5.9.3 - TypeScript compiler

## Configuration

**Environment:**
- **Public env vars** (injected at build/runtime):
  - `PUBLIC_POCKETBASE_URL` - Backend URL
  - `PUBLIC_DISABLE_SIGNUP` - Disable registration
  - `PUBLIC_PRIVATE_INSTANCE` - Private instance flag
- **Private env vars** (server-side only):
  - `MEILI_URL` - Meilisearch instance URL
  - `ORIGIN` - Application origin
  - `VALHALLA_URL` - Routing/elevation service
  - `NOMINATIM_URL` - Geocoding service
  - `OVERPASS_API_URL` - OpenStreetMap data API
  - `POCKETBASE_ENCRYPTION_KEY` - Database encryption key
  - `UPLOAD_FOLDER` - File upload directory
  - `UPLOAD_USER` / `UPLOAD_PASSWORD` - Upload credentials
  - `BODY_SIZE_LIMIT` - Request body size limit

**Build:**
- `web/vite.config.ts` - Vite configuration with SvelteKit plugin
- `web/svelte.config.js` - SvelteKit config (Node.js adapter, version tracking)
- `web/tsconfig.json` - TypeScript configuration (strict mode, bundler resolution)
- `web/tailwind.config.js` - Tailwind CSS configuration
- `web/postcss.config.js` - PostCSS configuration
- `web/playwright.config.ts` - E2E testing configuration
- `web/.npmrc` - npm engine-strict mode enabled
- `app/pubspec.yaml` - Flutter app configuration
- `docs/astro.config.mjs` - Documentation site (Astro)

## Platform Requirements

**Development:**
- Node.js 22.x
- npm (with engine-strict enabled)
- Flutter SDK 3.11.5+
- Dart 3.11.5+
- Git (version control)

**Production:**
- **Web:** Node.js 22 alpine container
- **Search:** Meilisearch v1.36.0 container
- **Database:** Custom flomp/wanderer-db container (PocketBase-based)
- **External APIs:** Nominatim, Overpass, Valhalla, OpenStreetMap tiles
- **Storage:** File system uploads directory (mapped volume in Docker)

**Mobile:**
- iOS 12+ (via Flutter)
- Android API 21+ (via Flutter)

---

*Stack analysis: 2026-06-07*
