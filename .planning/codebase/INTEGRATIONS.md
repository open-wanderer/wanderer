# External Integrations

**Analysis Date:** 2026-09-06

## APIs & External Services

**Geolocation & Geocoding:**
- **Nominatim** (OpenStreetMap) - Forward and reverse geocoding
  - SDK/Client: HTTP fetch via `web/src/lib/server/nominatim.ts`
  - Configuration: `NOMINATIM_URL` env var (default: https://nominatim.openstreetmap.org)
  - Rate limiting: 1 request per second enforced for public Nominatim
  - Endpoints: `/search`, `/reverse`
  - Used in: `web/src/routes/api/v1/geocoding/search/+server.ts`, `web/src/routes/api/v1/geocoding/reverse/+server.ts`

**Routing & Directions:**
- **Valhalla** - Route planning, navigation, elevation profiles
  - SDK/Client: HTTP POST via `web/src/lib/server/valhalla.ts`
  - Configuration: `VALHALLA_URL` env var (default: https://valhalla1.openstreetmap.de)
  - Endpoints used: `/route`, `/height`, `/trace_route`, `/locate`
  - Used in: 
    - `web/src/routes/api/v1/valhalla/route/+server.ts` - Route calculation
    - `web/src/routes/api/v1/valhalla/height/+server.ts` - Elevation data
    - `web/src/routes/api/v1/valhalla/navigate/+server.ts` - Turn-by-turn navigation
    - `web/src/routes/api/v1/valhalla/trace-route/+server.ts` - Snap GPS trace to road network
  - Mobile integration: `web/src/lib/stores/valhalla_store.svelte.ts` (fetch via `/api/v1/valhalla/*`)

**Map Data & Queries:**
- **Overpass API** (OpenStreetMap) - OSM data queries
  - SDK/Client: HTTP GET via `web/src/lib/server/overpass.ts`
  - Configuration: `OVERPASS_API_URL` env var (default: https://overpass-api.de)
  - Endpoint: `/api/interpreter`
  - Used in: `web/src/routes/api/v1/overpass/interpreter/+server.ts`
  - Layer integration: `web/src/lib/vendor/maplibre-layer-manager/overpass-layer.ts`

**Map Tiles & Rendering:**
- **OpenStreetMap Tile Servers** - Raster map tiles
  - Configuration: Can be customized via map config
  - Format: Standard XYZ tile URLs
  - Used by: MapLibre GL for map rendering

- **PMTiles** - Cloud-optimized vector tile format
  - Tool: `pmtiles` binary (v1.28.0, downloaded in `db/Dockerfile`)
  - Purpose: Archive and serve map tile data
  - Location: `db/Dockerfile` (stages 2 and 4)

## Data Storage

**Primary Database:**
- **PocketBase** 0.38.0 - Self-hosted SQLite-based backend
  - Connection: `MEILI_URL` (local service: http://search:7700 in Docker)
  - Exposed URL: `PUBLIC_POCKETBASE_URL` env var
  - Client libraries:
    - Web: `pocketbase` npm package (v0.27.0 in `web/package.json`)
    - Flutter: `pocketbase` pub package (implicitly via HTTP)
  - Authentication: Session-based via cookies or API tokens (`wanderer_key_*`)
  - Collections: trails, users, comments, summit_logs, waypoints, categories, tags, lists, regions, shares, notifications, summaries
  - Location: `db/main.go`, `db/routes/`, `db/migrations/`
  - Data persistence: `/pb_data/` volume in Docker

**Search Engine:**
- **Meilisearch** v1.36.0 (or v1.11.3 in dev/prod)
  - Connection: `MEILI_URL` env var (local service: http://search:7700 in Docker)
  - Client: `meilisearch` npm package (v0.58.0)
  - Authentication: `MEILI_MASTER_KEY` env var
  - Indexes: trails, users, lists, comments, summit_logs, summaries, notifications
  - Used in: `web/src/hooks.server.ts` for Meilisearch token generation
  - Used by: Search stores (`web/src/lib/stores/search_store.ts`, `web/src/lib/stores/list_store.ts`, `web/src/lib/stores/profile_store.ts`)
  - Encryption: Off (`MEILI_NO_ANALYTICS: "true"` in Docker compose)

**Local Mobile Database:**
- **ObjectBox** 5.3.1 - Local NoSQL database for Flutter
  - Purpose: Offline storage on mobile devices
  - Generated code: Via `objectbox_generator` (dev dependency)
  - Used by: Flutter app for cached data, offline support

**File Storage:**
- **Local Filesystem** - File uploads
  - Configuration: `UPLOAD_FOLDER` env var (default: /app/uploads)
  - Optional HTTP auth: `UPLOAD_USER`, `UPLOAD_PASSWORD` env vars
  - Endpoint: `web/src/routes/api/v1/file/upload/+server.ts`
  - Served via: PocketBase file serving
  - Docker volume: `./data/uploads:/app/uploads`

## Authentication & Identity

**Auth Provider:**
- **PocketBase Built-in** - Self-hosted authentication
  - Implementation: Session-based with cookies
  - OAuth2 support: Configured via PocketBase UI
  - API Token auth: `Authorization: Bearer wanderer_key_*` header support
    - Verification: `web/src/hooks.server.ts` (lines 59-75)
    - Token endpoint: `/auth/token` (POST)
  - Logout: Cookie-based session termination
  - Protected routes: Enforced in `web/src/routes/+layout.svelte` via `isRouteProtected()` utility

**OAuth Providers (via PocketBase):**
- Google
- GitHub
- Discord
- Gitea (custom OAuth)
- Others configurable through PocketBase admin panel
- Client integrations: `web/src/lib/stores/user_store.ts` (auth methods retrieval)
- OAuth icon retrieval: `${env.PUBLIC_POCKETBASE_URL}/_/images/oauth2/{provider}.svg`

## ActivityPub & Federation

**Protocol:**
- **ActivityPub** (W3C standard)
  - Implementation: Custom Go hooks in PocketBase
  - Types defined: `web/src/lib/models/activitypub/` (TypeScript client types)
  - Core types: Actor, Activity, WebFinger responses
  - Location: `db/federation/` (Go implementation)

**ActivityPub Services:**
- **Instance Actor** - Federation entry point
  - Type: `Application` actor
  - Endpoint: `/.well-known/webfinger` for WebFinger queries
  - Used for: Instance-to-instance follows and content sync

**Supported Activities:**
- `Create` - New content creation
- `Update` - Content updates
- `Delete` - Content deletion
- `Follow` / `Accept` - Instance follows
- `Undo` - Undo activities

**Content Types Federated:**
- Public trails (is_public = true)
- Public comments (is_public = true)
- Public lists (is_public = true)
- Public summit logs (is_public = true)
- Waypoints (inherit trail privacy)

**Federation Flow:**
- Initiated via mutual Follow between instances
- Bidirectional sync of public content
- Original ownership preserved (actor attribution)
- Privacy enforcement: Private records (is_public = false) never federated

## Webhooks & Callbacks

**Incoming:**
- ActivityPub Federation Inbox
  - Endpoint: `/inbox` (standard ActivityPub)
  - Receives activities from remote instances
  - Verification: HTTP Signature validation (go-fed/httpsig)

**Outgoing:**
- ActivityPub Federation Outbox
  - Endpoint: `/outbox` (standard ActivityPub)
  - Sends activities to remote instance inboxes
  - Format: Standard ActivityPub JSON-LD
  - Signing: HTTP Signature for authenticated delivery

## Email

**Email Services:**
- Not detected in core codebase
- PocketBase has email capabilities but not actively configured in provided compose files

## Monitoring & Observability

**Error Tracking:**
- Not detected (no Sentry, Rollbar, or DataDog integration)
- Frontend error handling via `web/src/lib/util/api_util.ts` (local error objects)

**Logs:**
- Frontend: Browser console logging
- Backend: Go standard logging (via pocketbase/pocketbase logging)
- Approach: Direct stdout/stderr capture in Docker containers

## CI/CD & Deployment

**Hosting:**
- **Docker-based** - Container orchestration
- Development: `docker-compose.yml` (local development)
- Dev deployment: `docker/docker-compose.dev.yml` (development environment)
- Prod deployment: `docker/docker-compose.prod.yml` (production environment)
- Container images:
  - Web: `flomp/wanderer-web` or `ghcr.io/open-wanderer/wanderer-web`
  - DB: `flomp/wanderer-db` or `ghcr.io/open-wanderer/wanderer-db`
  - Search: `getmeili/meilisearch`

**CI Pipeline:**
- Not detected in config files provided
- GitHub likely (based on org in compose files)
- Build images: flomp/* (Docker Hub) or ghcr.io/open-wanderer/* (GitHub Container Registry)

**Platform Support:**
- Web: Node.js 22 Alpine
- Mobile: iOS (Flutter), Android (Flutter)
- Desktop: Not currently supported

## Environment Configuration

**Required Env Vars (Web):**
- `PUBLIC_POCKETBASE_URL` - Backend service URL
- `MEILI_URL` - Meilisearch service URL
- `MEILI_MASTER_KEY` - Meilisearch authentication
- `ORIGIN` - Server origin for federation and links

**Required Env Vars (Database/Go):**
- `MEILI_URL` - Meilisearch service URL
- `MEILI_MASTER_KEY` - Meilisearch authentication
- `POCKETBASE_ENCRYPTION_KEY` - Data encryption key
- `ORIGIN` - Server origin

**Optional Env Vars:**
- `VALHALLA_URL` - Custom Valhalla service (uses default if not set)
- `NOMINATIM_URL` - Custom Nominatim service (uses default if not set)
- `OVERPASS_API_URL` - Custom Overpass API (uses default if not set)
- `UPLOAD_FOLDER` - Custom upload directory
- `UPLOAD_USER` / `UPLOAD_PASSWORD` - HTTP basic auth for uploads
- `PUBLIC_DISABLE_SIGNUP` - Disable user registration
- `PUBLIC_MAP_MAX_POLYLINES` - Map polyline limit
- `BODY_SIZE_LIMIT` - Request body size limit
- `REGION_ARCHIVE_CRON_SCHEDULE` - Archive regions schedule (commented out in compose)

**Secrets Location:**
- Environment file (`.env` or Docker compose variables)
- PocketBase encryption key required for data protection
- Meilisearch master key required for search operations

## Plugins & Extensions

**Strava Integration:**
- Location: `db/integrations/strava/` or `plugins/strava/`
- Purpose: Sync trails from Strava
- API: Strava v3 API
- Migration note: Strava API host migration to https://www.api-v3.strava.com

**Plugin System:**
- Framework: Extism (Go SDK v1.7.1 in dependencies)
- Purpose: WebAssembly-based plugin support in PocketBase
- Routes: `db/routes/plugin_system_auth.go`, `db/routes/plugin_system_session_auth.go`

## Data Integration Patterns

**Trail Data Import:**
- **GPX Files** - GPS trace import via `gpxgo` (Go) and `heic2any` for image conversion
- **Third-party sources** - Strava integration
- **Trace snapping** - Valhalla trace-route for GPS noise reduction

**Search Indexing:**
- Real-time indexing via Meilisearch
- Tracks: trails, users, lists, comments, summit_logs, summaries, notifications
- Filters: Public content, user-specific results

**Geographic Features:**
- Bounding box queries via MapLibre GL
- Marker clustering via `supercluster` package
- Polyline encoding via `go-polyline` (backend) and Turf.js (frontend)

---

*Integration audit: 2026-09-06*
