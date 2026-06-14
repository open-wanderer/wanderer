# External Integrations

**Analysis Date:** 2026-06-10

## APIs & External Services

**Mapping & Geospatial:**
- **Nominatim** (OpenStreetMap) - Geocoding and reverse geocoding
  - SDK/Client: Native HTTP via `web/src/lib/server/nominatim.ts`
  - Rate limiting: 1 request per second enforced
  - Retry logic: Up to 2 retries on failure
  - Default: `https://nominatim.openstreetmap.org`
  - Config: `NOMINATIM_URL` environment variable

- **Overpass API** (OpenStreetMap) - POI and geographic feature queries
  - SDK/Client: Native HTTP via `web/src/lib/server/overpass.ts`
  - Retry logic: Up to 2 retries on failure
  - Default: `https://overpass-api.de`
  - Config: `OVERPASS_API_URL` environment variable

- **Valhalla** - Routing and turn-by-turn directions
  - SDK/Client: Native HTTP via `web/src/lib/server/valhalla.ts`
  - Config: `VALHALLA_URL` environment variable
  - Optional: Can be self-hosted or omitted
  - Example: `https://valhalla1.openstreetmap.de`

**OSM Tiles:**
- MapLibre GL compatible tile sources for map rendering
- Used in both web and mobile applications

## Data Storage

**Databases:**
- **PocketBase** 0.38.0
  - Type: Self-hosted BaaS (embedded SQLite or PostgreSQL-compatible)
  - Connection: `PUBLIC_POCKETBASE_URL` (default: `http://localhost:8090`)
  - Client: PocketBase JavaScript SDK v0.26.8 (`web/src/lib/pocketbase.ts`)
  - Mobile client: Dio HTTP library with native PocketBase integration
  - Collections: users, trails, waypoints, comments, lists, integrations, activitypub_actors, settings, summit_logs
  - Authentication: Session-based via PocketBase AuthStore
  - API Authentication: Bearer token with `wanderer_key` prefix for API access

**Search:**
- **Meilisearch** v1.36.0
  - Purpose: Full-text search indexing for trails, users, lists
  - Connection: `MEILI_URL` (default: `http://search:7700`)
  - Auth: Master key via `MEILI_MASTER_KEY` environment variable
  - Client: Meilisearch JavaScript client v0.57.0 (`web/src/hooks.server.ts`)
  - Search tokens: Per-user tokens generated via `/search/token` PocketBase endpoint
  - Token caching: Browser cookie-based (`meilisearch_token`) with version tracking
  - Indexed collections: Trails, Users, Lists, ActivityPub Actors (see `db/main.go` hooks)

**File Storage:**
- **Local filesystem** - File upload directory mapped via Docker volume
  - Path: `/app/uploads` in container
  - Host mapping: `./data/uploads:/app/uploads` (docker-compose)
  - Auth: Optional via `UPLOAD_USER` and `UPLOAD_PASSWORD` environment variables
  - Usage: Trail GPX files, user avatars, photos, documents

**Caching:**
- **ObjectBox** 5.3.1 (Mobile only)
  - Local NoSQL database for offline access
  - Caches trails, user profiles, feeds
  - Location: `app/lib/provider/objectbox_store_provider.dart`

## Authentication & Identity

**Primary Auth Provider:**
- **PocketBase Built-in**
  - Session-based authentication
  - JWT tokens stored in browser/app local storage
  - Cookie-based session management (`pb_auth` cookie)
  - Per-user authentication guard via `isRouteProtected()` utility

**Third-Party OAuth Integrations (Backend Support):**
- **Strava** - Fitness activity import
  - Config: `db/routes/integration_strava.go`
  - Tokens: Encrypted storage in PocketBase integrations collection
  - Refresh token handling for continuous access
  - Client credentials: Stored encrypted in `integration.strava` field

- **Komoot** - Trail data import
  - Config: `db/routes/integration_komoot.go`
  - Similar token handling to Strava

- **Hammerhead** - Cycling computer integration
  - Config: `db/routes/integration_hammerhead.go`

**Federation (ActivityPub):**
- ActivityPub protocol support for federated social features
- Actors stored in `activitypub_actors` PocketBase collection
- HTTP signature verification for inbound requests
- Profile follow, trail sharing, and comments federation

## Monitoring & Observability

**Error Tracking:**
- Not detected in codebase configuration
- Uses native error handling via try/catch blocks

**Logs:**
- Backend: Go logger in `db/main.go`
- Frontend: Console logging for development (no structured logging framework)
- Docker healthchecks: HTTP endpoint probes with retry logic
  - Web: `GET http://localhost:3000/`
  - DB: `GET http://localhost:8090/health`
  - Search: `GET http://localhost:7700/health`

## CI/CD & Deployment

**Hosting:**
- **Docker Compose** - Production and development deployments
- **Multi-container architecture:**
  - Web service: Node.js 22 Alpine (port 3000)
  - Database: Custom Go/PocketBase container (port 8090)
  - Search: Meilisearch official image (port 7700)
  - All services on shared `wanderer` Docker bridge network

**Container Images:**
- `flomp/wanderer-web` - Web frontend and SvelteKit backend
- `flomp/wanderer-db` - PocketBase custom backend with Go integrations
- `getmeili/meilisearch:v1.36.0` - Search engine

**CI Pipeline:**
- Not detected; likely external to codebase (GitHub Actions, etc.)

**Mobile Distribution:**
- Flutter app targets iOS 12+ and Android API 21+
- Build outputs: APK (Android), IPA (iOS) via Flutter build system

## Environment Configuration

**Required Environment Variables:**

Critical:
- `POCKETBASE_ENCRYPTION_KEY` - Must be exactly 32 bytes long (enforced in `db/main.go`)
- `MEILI_MASTER_KEY` - Recommended minimum 32 bytes
- `MEILI_URL` - Meilisearch server URL
- `PUBLIC_POCKETBASE_URL` - Backend database endpoint
- `ORIGIN` - CORS origin (e.g., `http://localhost:3000`)

Optional:
- `PUBLIC_DISABLE_SIGNUP` - Disable registration (default: false)
- `OVERPASS_API_URL` - Custom Overpass endpoint
- `VALHALLA_URL` - Routing service endpoint
- `NOMINATIM_URL` - Geocoding service endpoint
- `PUBLIC_MAP_MAX_POLYLINES` - Performance limit (default: 100)
- `UPLOAD_USER` / `UPLOAD_PASSWORD` - File upload authentication
- `UPLOAD_FOLDER` - Upload directory path
- `BODY_SIZE_LIMIT` - HTTP request size (default: Infinity in dev)

**Secrets Location:**
- Environment variables in Docker Compose `.env` files
- Default credentials present in `docker-compose.yml` (change before production)
- PocketBase encryption key: Must be externally provided
- Meilisearch master key: Must be externally provided

**Public Configuration:**
- `PUBLIC_POCKETBASE_URL` - Published to client
- `PUBLIC_DISABLE_SIGNUP` - Published to client
- `PUBLIC_OVERPASS_API_URL` - Published to client
- `PUBLIC_MAP_MAX_POLYLINES` - Published to client
- Accessed via `$env/dynamic/public` in SvelteKit

## Webhooks & Callbacks

**Incoming:**
- ActivityPub inbound activities (follow, create, like, announce, delete, undo)
- Routes: `db/routes/activitypub.go`
- HTTP signatures verified for authenticity

**Outgoing:**
- Trail creation/updates published to federated followers
- Comments and likes propagated via ActivityPub
- Profile follow requests sent to remote ActivityPub instances
- Implementation: `db/federation/` directory

**API Webhooks:**
- PocketBase subscriptions available but not heavily used in current codebase
- Real-time capability present but not required for core features

## Data Flow

### Search Indexing Flow

1. User/Trail/List/Actor created/updated in PocketBase
2. Hook handler triggered: `CreateTrailHandler()`, `UpdateUserHandler()`, etc. (`db/main.go:90-100`)
3. Meilisearch client indexes document via `db/hooks/` handlers
4. Browser requests search token via `/search/token` endpoint
5. Token cached client-side with version tracking
6. Search queries sent to Meilisearch with user-scoped token

### Authentication Token Flow

1. User logs in via PocketBase auth UI
2. Session token stored in browser local storage
3. PocketBase cookie set in response (`pb_auth`)
4. Server hook validates token on each request (`web/src/hooks.server.ts:79-80`)
5. API token auth supported via Bearer header for programmatic access (`web/src/hooks.server.ts:58-76`)

### Geocoding Flow

1. User adds location or searches address on web frontend
2. Request to Nominatim via server-side proxy (`web/src/lib/server/nominatim.ts`)
3. Rate limiter enforces 1 request/second
4. Response cached locally or in component state
5. Map updated with coordinates

### Third-Party Integration Flow

1. User initiates Strava/Komoot/Hammerhead integration
2. OAuth flow redirects to service provider
3. Authorization code exchanged for tokens at backend (`db/routes/integration_*.go`)
4. Tokens encrypted and stored in PocketBase `integrations` collection
5. Import endpoint fetches and syncs data periodically or on-demand

## Domain-Specific Integrations

**Trail Data Sources:**
- Strava API integration for activity import
- Komoot API integration for trail data
- Manual GPX file uploads via web and mobile

**Map Rendering:**
- MapLibre GL for vector tiles on web
- Flutter Map with vector tile support on mobile
- PMTiles format support for efficient tile serving

**Activity Publishing:**
- ActivityPub federation for social features
- Trail shares published to followers
- Comments and interactions federated across instances

---

*Integration audit: 2026-06-10*
