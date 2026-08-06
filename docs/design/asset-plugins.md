# Asset plugins: technical reference

This document describes the current asset plugin stack in wanderer from the browser to an external media provider. It covers the generic contract for plugins of type `assets`, the first-party Immich implementation, all import, storage, attachment, and maintenance workflows, and the files and functions involved in each workflow.

The general plugin infrastructure and trail plugins are documented in [`plugin-system.md`](../src/content/docs/develop/plugin-system.md). The asset plugin type is generic, but Immich is currently the only first-party implementation in this repository. The design and implementation decisions behind candidate pagination and per-call request budgets are recorded separately in [`asset-candidates-pagination.md`](asset-candidates-pagination.md); this reference documents their current operational behavior.

## 1. Terminology and scope

| Term | Meaning |
| --- | --- |
| Asset plugin | A WASM plugin with manifest type `assets` and capability `asset_library.v1`. |
| Provider asset | A photo in an external system such as Immich. Its stable provider ID is stored as `external_id`. |
| wanderer asset | A record in the PocketBase `assets` collection. It may contain a local file or only a private remote reference. |
| Asset link | A relation from an asset to a trail, waypoint, or summit log through `trail_assets`, `waypoint_assets`, or `summit_log_assets`. |
| Candidate | A provider photo returned as a possible match for a track, time range, or coordinate but not necessarily imported yet. |
| Materialization | Downloading a remote reference stored with `storage_mode=link_private` and converting it to `storage_mode=copy`. |

The internal wanderer photo library is not a plugin. The picker combines it with results from all active asset plugins, so it is included in the end-to-end workflows below.

## 2. Architecture

The normal asset plugin call path is:

```text
Svelte component
  -> SvelteKit API proxy /api/v1/plugins/assets/...
  -> PocketBase route /plugins/assets/...
  -> plugin and instance resolution
  -> dedicated plugin-worker process
  -> WASM export asset_library_v1
  -> wanderer.http_request
  -> host policy, authentication injection, and HTTP client
  -> external provider (currently Immich)
```

Plugin code does not open network connections directly. Every provider request passes through the `wanderer.http_request` host function, keeping connector targets, allowed paths, authentication, redirects, TLS, private-network access, content types, and size limits under host control.

Imports continue through a second path:

```text
plugin returns Photo descriptors
  -> importer.ImportPhotoAssets
  -> copy: fetch media and create a PocketBase file
     or
     link_private: store RemotePhotoAsset in metadata.remote
  -> util.CreatePhotoAsset
  -> reuse an existing provider asset or create a new asset
  -> link the asset to a trail, waypoint, or summit log
```

## 3. Primary files

| Area | File | Responsibility |
| --- | --- | --- |
| Manifest | `plugins/immich/plugin.json` | Plugin type, capability, authentication context, connector permissions, plugin configuration, and host configuration. |
| Provider entry point | `plugins/immich/main.go` | `asset_library_v1` export, input/output handling, and structured plugin failures. |
| Provider logic | `plugins/immich/immich.go` | `check`, `candidates`, `import`, and `thumbnail` actions plus Immich API access. |
| Provider pagination | `plugins/immich/pagination.go` | SDK-independent candidate page consumption, strict budgets, and raw-offset continuation. |
| Provider matching | `plugins/immich/matching.go` | Geographic filtering, distance calculation, and ordering. |
| Provider types | `plugins/immich/types.go`, `plugins/immich/types_tinygo.go` | Shared matching types plus TinyGo-only SDK protocol and Immich response types. |
| Plugin SDK | `plugins/sdk/types.go`, `plugins/sdk/host_http.go` | `Photo`, `MediaSource`, `MediaRef`, host-request protocol, and WASM host imports. |
| Host endpoints | `db/routes/plugin_system_assets.go` | Capability calls, candidate search, import, auto-attach, thumbnail cache, and authorization. |
| Waypoint naming | `db/routes/waypoint_naming.go` | Provider-independent server-side name resolution through Overpass, Nominatim, and coordinate fallback. |
| Clustering | `db/routes/waypoint_cluster.go` | Merging nearby photos and assigning them to existing waypoints. |
| Importer | `db/plugins/importer/importer.go` | Storage strategy, media retrieval, limits, and handoff to the asset data model. |
| Asset data model | `db/util/assets.go` | Asset creation, deduplication, reuse, and linking. |
| Remote asset service | `db/services/assets/service.go` | Streaming, status, repair, materialization, and deletion of private remote assets. |
| Asset routes | `db/routes/assets.go` | File endpoint and asynchronous remote-asset jobs. |
| Lifecycle hooks | `db/hooks/trails.go`, `db/hooks/plugin_instances.go`, `db/hooks/assets.go` | Publication safety, disable/delete protection, and search-index updates. |
| Database schema | `db/migrations/1778359896_created_assets.go` | Collections, fields, rules, and unique indexes. |
| Runtime | `db/pluginsystem/runtime.go`, `db/pluginsystem/worker.go`, `db/pluginsystem/worker_process.go` | Dedicated worker process and WASM invocation. |
| Host networking | `db/pluginsystem/host_http.go`, `db/pluginsystem/policy.go`, `db/services/pluginhost/config.go` | Connector resolution, host policy, authentication injection, and effective configuration. |
| Frontend picker | `web/src/lib/components/photo/photo_library_picker_modal.svelte` | Parallel search, filtering, deduplication, map display, and selection. |
| Frontend persistence | `web/src/lib/stores/asset_store.ts` | Uploads, existing links, and provider imports attached to a target. |
| Remote asset UI | `web/src/routes/settings/plugins/+page.svelte` | Summary, download, repair, delete, and disable flow. |

## 4. Manifest and capability contract

An asset plugin must declare at least:

```json
{
  "manifestVersion": "1.0",
  "id": "immich",
  "type": "assets",
  "runtime": {
    "type": "wasm",
    "entrypoint": "plugin.wasm"
  },
  "capabilities": [
    {
      "name": "asset_library",
      "version": "v1",
      "export": "asset_library_v1"
    }
  ]
}
```

`assetPluginInvocationForUser` in `db/routes/plugin_system_assets.go` accepts only an enabled `plugin_instances` record owned by the current user, a locally available plugin with `type=assets`, and exactly the `asset_library.v1` capability.

### 4.1 Actions

Every operation uses the same WASM export and selects behavior through `request.action`.

| Action | Input | Output | Purpose |
| --- | --- | --- | --- |
| `check` | Draft authentication and configuration | Optional `userId` | Validate configuration and credentials. |
| `candidates` | Track points and/or coordinate, time range, `search`, optional `state` | `candidates`, `state`, `hasMore`, `stats`, `hasTimestamps` | Find one bounded batch of matching provider photos. |
| `import` | `assetIds` and context | `photos`, `omittedAssetIds` | Resolve selected provider IDs to an exact partition of importable and explicitly omitted media. |
| `thumbnail` | Exactly one ID in `assetIds` | First item in `photos` | Resolve an authenticated preview descriptor. |

`import-to-target` and `import-to-waypoint` are host endpoints, not additional plugin actions. The host normalizes them to `request.action=import` before invoking WASM.

### 4.2 Capability input

The host serializes `pluginAssetLibraryInput`:

```json
{
  "instance": {
    "id": "instance-id",
    "pluginId": "immich"
  },
  "auth": {
    "apiKey": "decrypted-secret"
  },
  "config": {
    "url": "https://immich.example",
    "timeWindowMinutes": 30,
    "maxDistanceMeters": 150,
    "maxWaypoints": 25,
    "importSize": "preview",
    "ownedOnly": false,
    "userId": "immich-user-id"
  },
  "limits": {
    "maxPhotosPerTrail": 20,
    "maxPhotosPerWaypoint": 5,
    "maxPhotosPerSummitLog": 20
  },
  "search": {
    "maxItems": 100,
    "maxScannedItems": 2500,
    "maxProviderRequests": 10
  },
  "state": {
    "page": 2,
    "offset": 100
  },
  "request": {
    "action": "candidates",
    "trailId": "trail-id",
    "lat": 47.3769,
    "lon": 8.5417,
    "points": [
      {
        "lat": 47.3769,
        "lon": 8.5417,
        "distance": 1200,
        "timestamp": "2026-06-23T10:15:00Z"
      }
    ],
    "startedAt": "2026-06-23T10:00:00Z",
    "endedAt": "2026-06-23T12:00:00Z",
    "takenAfter": "2026-06-23T09:30:00Z",
    "takenBefore": "2026-06-23T12:30:00Z",
    "doubleRadius": false,
    "assetIds": []
  }
}
```

Important boundaries:

- `auth` is filtered through `pluginsystem.PluginInputAuth`. OAuth token material is not passed to ordinary plugin exports. API-key fields such as Immich's key remain visible to the plugin, while actual HTTP authentication is still injected by the host.
- `config` contains only `config.plugin`. `config.host` and connector trust data are not passed to the plugin export.
- `limits` advertises configured photo limits. Auto-attach enforces them when writing; current manual asset imports intentionally call the importer without per-target photo limits.
- `search` is a separate read-budget block. `maxItems` bounds returned candidates, `maxScannedItems` bounds provider assets evaluated in this call, and `maxProviderRequests` lets the plugin stop cooperatively before the runtime's hard request ceiling.
- `state` is provider-specific continuation data. The host passes it back unchanged to the next plugin call but never exposes it to the browser.
- `trailId` is context only. The plugin has no direct database access.
- `points[].distance` is cumulative distance from the beginning of the track in metres.
- Track points are reduced to approximately 2,000 points before plugin invocation.
- Explicit `takenAfter` and `takenBefore` values are validated as RFC 3339 and mirrored into `startedAt` and `endedAt`.

### 4.3 Success and failure semantics

A successful output contains only data for the requested action.

| Layer | Authoritative signal | Meaning |
| --- | --- | --- |
| WASM runtime | Error from `session.Call(...)` | The export did not complete successfully, for example because `assetLibraryV1` returned a non-zero code. |
| Plugin protocol | `error` with `code` and `message` | Execution was technically possible, but the requested operation failed. The host converts it to an API error. |
| HTTP API | HTTP status | `2xx` means the host accepted the result; `4xx` or `5xx` reports a request, plugin, or host failure. |

Plugins must either fail the export or return a structured `error` when an operation fails.

### 4.4 Candidate output

```json
{
  "candidates": [
    {
      "assetId": "provider-asset-id",
      "originalFileName": "IMG_1234.jpg",
      "takenAt": "2026-06-23T10:18:00Z",
      "lat": 47.377,
      "lon": 8.542,
      "distance": 32.4,
      "pointLat": 47.3769,
      "pointLon": 8.5417,
      "distanceFromStart": 1200,
      "city": "Zurich",
      "country": "Switzerland"
    }
  ],
  "state": {
    "page": 12,
    "offset": 100
  },
  "hasMore": true,
  "stats": {
    "scannedItems": 2500
  },
  "takenAfter": "2026-06-23T09:30:00Z",
  "hasTimestamps": true
}
```

`assetId` must be stable within the provider. `pointLat`, `pointLon`, and `distanceFromStart` describe the nearest track point, not the original photo position. The host and frontend use them for ordering and for positioning automatically generated camera waypoints. `stats.scannedItems` is plugin-reported observability data and is not a host enforcement mechanism.

`hasMore=true` requires a non-empty state that differs from the input state. The host compares canonical JSON hashes and rejects unchanged or cyclic states as protocol errors. Immich uses `{page, offset}`, where `offset` counts raw provider items evaluated within the current page. The browser receives only an opaque process-local `cursorId`; the server store binds it to the user, actor, plugin instance, normalized search, effective configuration, and authentication. Unknown, expired, or mismatched browser cursors produce `restartRequired=true`, causing the picker to discard that plugin's displayed results and restart its search. Full rationale and limits are recorded in [`asset-candidates-pagination.md`](asset-candidates-pagination.md).

### 4.5 Photo and media output

```json
{
  "photos": [
    {
      "externalId": "provider-asset-id",
      "filename": "IMG_1234.jpg",
      "contentType": "image/jpeg",
      "takenAt": "2026-06-23T10:18:00Z",
      "lat": 47.377,
      "lon": 8.542,
      "source": {
        "type": "connector",
        "mediaRef": {
          "connector": "api",
          "auth": "api_key",
          "path": "/api/assets/provider-asset-id/thumbnail",
          "query": [{ "name": "size", "value": "preview" }],
          "assetId": "provider-asset-id"
        }
      }
    }
  ],
  "omittedAssetIds": [
    {
      "assetId": "unsupported-provider-asset-id",
      "reason": "asset cannot be converted to a photo"
    }
  ]
}
```

For `action=import`, the `externalId` values in `photos` and the `assetId` values in `omittedAssetIds` must be unique, disjoint, requested by the host, and together equal the complete deduplicated request set. Missing, duplicated, overlapping, or unrequested IDs are protocol errors. Manual HTTP import endpoints return `{ "imported": [...], "omitted": [...] }` rather than a bare result array.

Supported source types:

| `source.type` | Required data | Behavior |
| --- | --- | --- |
| `connector` | `mediaRef.connector` and `mediaRef.path` | The host resolves a connector declared in the manifest and optionally injects the authentication context named by `mediaRef.auth`. |
| `url` | `source.url` | The host uses the SSRF-protected public URL fetcher. |

`mediaRef.assetId` is metadata only; a medium cannot be loaded without `mediaRef.path`. Sources with `expiresAt` may be copied but cannot be retained as `link_private` references.

## 5. Configuration and ownership

The effective configuration is assembled as follows:

```text
installed_plugins.config
  + permitted plugin_instances.config overrides
  -> config.plugin  (passed to WASM)
  -> config.host    (used only for host behavior and connector policy)
```

`pluginhost.EffectiveConfig` combines both layers. `pluginhost.InstanceConfigOverrides` allows users to override only explicitly exposed host fields. In particular, `host.connectors`, `allowPrivate`, custom TLS, and storage redirect origins remain administrator-owned.

### 5.1 Immich plugin configuration

| Field | Layer | Default | Use |
| --- | --- | --- | --- |
| `url` | Plugin | Required | Immich base URL; may provide the base URL for the configured `api` connector. |
| `timeWindowMinutes` | Plugin | `30` | Search padding before track start and after track end. |
| `maxDistanceMeters` | Plugin | `150` | Maximum distance from a photo to its nearest track point or the supplied coordinate. |
| `maxWaypoints` | Plugin | `25` | Maximum number of new waypoints in enforced auto-attach flows; the current host treats `0` as unlimited. |
| `importSize` | Plugin | `preview` | `preview` uses JPEG previews; `original` uses the original asset. |
| `ownedOnly` | Plugin | `false` | Restricts results to the Immich user ID obtained by `check`. |
| `userId` | Plugin, hidden | Empty | Result of `check`, required by `ownedOnly`. |
| `photoMode` | Host | `copy` | Either `copy` or `link_private`. |
| `maxPhotosPerTrail` | Host | `20` | Automatically imported photos attached directly to a trail. The current Immich manifest does not expose this field in its UI. |
| `maxPhotosPerWaypoint` | Host | `5` | Automatically imported photos per waypoint. |
| `maxPhotosPerSummitLog` | Host | `20` | Generic summit-log photo limit; currently not editable in the asset plugin UI and not enforced for manual asset plugin imports. |
| `autoAttach.trailPlugins` | Host | `true` | Auto-attach after completed provider trails are imported. |
| `autoAttach.upload` | Host | `true` | Auto-attach after GPX uploads. |

`pluginhost.AssetConnectorConfig` may use the user-configured `url` as a connector base when the administrator connector has no `baseURL`. This derived connector always sets `allowPrivate=false` and removes inherited TLS or storage-origin trust. An administrator must explicitly configure the connector as trusted when Immich is hosted on a private network.

## 6. Data model

### 6.1 The `assets` collection

Migration `db/migrations/1778359896_created_assets.go` defines, among other fields:

| Field | Meaning |
| --- | --- |
| `type` | Currently only `photo`. |
| `file` | Local PocketBase file; initially empty for `link_private`. |
| `storage_mode` | `copy` or `link_private`. |
| `remote_status` | `available`, `missing`, or `inaccessible`. |
| `author` | The user's local ActivityPub actor. |
| `external_provider` | Plugin ID, for example `immich`. |
| `external_id` | Stable provider asset ID. |
| `taken_at`, `lat`, `lon` | Photo metadata supplied by the plugin. |
| `metadata.remote` | For remote links: `pluginId`, filename, content type, and the complete `MediaSource`. |
| `remote_checked_at` | Time of the last remote fetch or repair check. |
| `remote_error` | Last remote error message. |
| `remote_missing_since` | First observation of a 404 or 410 response. |

The partial unique index on `(author, external_provider, external_id, type)` prevents duplicate provider photos for one user. `util.CreatePhotoAsset` also checks for an existing record before insertion and recovers from a concurrent unique conflict.

### 6.2 Link collections

| Collection | Target field | Unique index |
| --- | --- | --- |
| `trail_assets` | `trail` | `(asset, trail)` |
| `waypoint_assets` | `waypoint` | `(asset, waypoint)` |
| `summit_log_assets` | `summit_log` | `(asset, summit_log)` |

An asset may have multiple targets. `util.EnsureAssetLink` is idempotent. `util.PhotoAssetLinkTargets` links a photo directly to a trail only when no waypoint or summit log target is supplied.

## 7. Workflows

### 7.1 Installation and discovery

1. The bundle is a direct child of `data/plugins`, with `data/plugins/<plugin-id>/plugin.json` and `plugin.wasm`.
2. `pluginsystem.DiscoverLocalPlugins` in `db/pluginsystem/manifest.go` reads only direct subdirectories, loads and validates manifests, and records visible bundle errors.
3. `pluginsystem.Manager.SyncInstalledPlugins` in `db/pluginsystem/manager.go` mirrors the manifest, path, capability names, and default configuration into `installed_plugins`.
4. `defaultConfig` builds `config.plugin` from `configSchema` defaults and `config.host` from `hostConfig`.
5. `LoadInstalledPlugin` uses the cache on request hot paths and falls back to local discovery when the database record is missing or unusable.
6. `plugins_index` in `web/src/lib/stores/plugin_store.ts` loads the list and maps manifests to `PluginProvider`; asset plugins are grouped by `type=assets`.

### 7.2 Saving configuration and running `check`

The settings dialog runs `check` before saving an asset plugin.

1. `pluginInstanceFromForm` in `plugin_instance_settings_modal.svelte` builds `auth`, `config.plugin`, and permitted `config.host` overrides.
2. `applyAssetPluginCheckResult` posts the draft to `/api/v1/plugins/assets/{plugin}/check`.
3. `web/src/routes/api/v1/plugins/assets/[plugin]/[action]/+server.ts` validates plugin ID and action against a fixed allowlist and proxies the request to PocketBase `/plugins/assets/{plugin}/check`.
4. `PluginSystemAssetCheck` binds the body and calls `assetPluginDraftInvocation`; an enabled or previously saved instance is not required for this draft check.
5. `assetPluginDraftInvocation` loads the plugin and capability, decrypts existing secrets, adopts only non-empty submitted secrets, and projects draft configuration through `pluginhost.InstanceConfigOverrides`. User input therefore cannot inject connector trust policy.
6. `callAssetPlugin` invokes `asset_library_v1` with `action=check`.
7. `immichClient.check` loads `/api/users/me` and performs a small metadata search covering the previous hour, validating the user endpoint, asset read permission, and connector configuration.
8. The returned `userId` is stored in a hidden plugin field.
9. Hooks in `db/hooks/plugin_instances.go` add defaults, encrypt secret fields with `POCKETBASE_ENCRYPTION_KEY`, and censor them from ordinary API responses.

### 7.3 Manual candidate search

The entry point is `PhotoLibraryPickerModal.openModal`.

1. `dateSliderDefaults` determines the UI time range. Tracks with timestamps start with the track interval; tracks without timestamps default to the previous year. The slider may expand in steps of one, five, and ten years.
2. `requestBody` combines trail ID, unsaved `trailData`, optional target ID, coordinate, time range, and `doubleRadius`.
3. `loadCandidates` starts `loadWandererCandidates` for the internal library and one `loadPluginCandidates` call for every enabled asset plugin in parallel.
4. One failing provider does not hide results from other providers. The picker reports a fatal error only when all loaders fail.
5. Plugin requests pass through `PluginSystemAssetCandidates` and `pluginSystemAssetCall`.
6. `assetLibraryActionInputForApp` reads optional `trailData` first and otherwise loads the stored GPX. `trailTrackPointsFromBytes` calculates cumulative distance and time boundaries, then `decimateTrackPoints` reduces large tracks.
7. `assetPluginInvocationForUser` requires an active instance, decrypts authentication, and builds effective configuration and connector policy.
8. `callAssetPluginCandidates` loads or creates a process-local cursor entry, invokes `callAssetPlugin` with picker limits of 100 returned items, 2,500 scanned provider items, and 10 cooperative provider requests, and validates state progress.
9. `callAssetPlugin` starts a worker with a hard budget of 24 plugin-originated host requests, serializes the capability input, and invokes the manifest export.
10. Immich `candidates` determines the search window through `candidateWindow`, resumes from `{page, offset}`, loads unarchived images with EXIF data from `/api/search/metadata`, and stops strictly at either the item or scan budget.
11. The browser response contains candidates and, when more data exists, an opaque `cursorId`; raw plugin `state` and `stats` are never serialized to the browser.
12. The frontend adds `source=plugin`, provider identity, and the protected thumbnail URL. It stores pagination independently per plugin and renders a load-more action for each provider with another batch.
13. A continuation sends the same search parameters plus `cursorId`. New candidates are mixed into the existing list and `uniqueCandidates` deduplicates by `(externalProvider, externalId)`. When a corresponding wanderer asset already exists, the local candidate wins and the plugin result is hidden.
14. An expired, unknown, or mismatched cursor returns `restartRequired`; the picker removes the old candidates for that plugin and begins again. The display is globally re-sorted after every batch by `distanceFromStart`, distance, and capture time.

`AssetLibraryCandidates` serves the internal library. It filters the user's photos by date and a spatial bounding-box prefilter, calculates exact Haversine distance, and removes assets already linked to the current target. Searches without spatial context use page/offset pagination with a default of 100 and maximum of 250 records; spatial searches are unpaginated so distance ranking sees the full matching set. The internal radius is 1,000 metres or 2,000 metres with `doubleRadius`, independent of the provider-specific Immich radius.

### 7.4 Immich matching

`immichClient.searchAssetCandidates` sends the following request per provider page:

```json
{
  "takenAfter": "...",
  "takenBefore": "...",
  "type": "IMAGE",
  "withExif": true,
  "isArchived": false,
  "page": 1,
  "size": 250
}
```

A picker call returns at most 100 candidates, evaluates at most 2,500 raw provider assets, and cooperatively performs at most 10 provider requests. Auto-attach instead allows 2,500 candidates, 2,500 evaluated assets, and 16 provider requests per batch. A stop inside a 250-item provider page returns the current page and raw-item offset, so the next call reloads that page and skips the evaluated prefix. `nextPage` must parse as a strictly increasing page number. The separate `searchAssets` helper remains for `check`, which reads one page.

`matchAssetCandidates` removes assets owned by another user when `ownedOnly=true` and `userId` is known, assets without both EXIF coordinates, and assets farther than `maxDistanceMeters` or twice that radius when `doubleRadius=true`.

`candidateForAsset` compares a photo with every supplied track point and records the nearest point. Without a track, it uses the single coordinate. `sortMatches` orders by increasing distance and then by descending `takenAt` string.

### 7.5 Authenticated thumbnails

1. The picker generates `/api/v1/plugins/assets/{plugin}/thumbnail/{assetId}`.
2. The SvelteKit thumbnail proxy forwards the bearer token and `If-None-Match`, then streams the backend response unchanged.
3. `PluginSystemAssetThumbnail` requires authentication and an active plugin instance.
4. `fetchPluginAssetThumbnailEntryForUser` checks the in-memory cache using a key containing user, plugin, instance, and provider asset ID.
5. `beginPluginAssetThumbnailFetch` collapses concurrent misses for the same key into one provider fetch.
6. On a miss, `fetchPluginAssetThumbnailEntryUncached` invokes the `thumbnail` action. Immich returns a connector reference to `/api/assets/{id}/thumbnail?size=preview`.
7. `importer.FetchPhotoMedia` reads at most 8 MiB and applies connector, authentication, redirect, and content-type policy.
8. The host creates a SHA-256-based ETag and caches bytes for 24 hours. The cache holds at most 512 entries and 64 MiB and evicts the least recently used entry.
9. The HTTP response uses `private, no-cache` and `Vary: Authorization`; a matching ETag produces `304 Not Modified`.

### 7.6 Attaching a selection to a trail, waypoint, or summit log

The frontend initially holds only IDs: wanderer photos in `_assetLinks`, provider photos grouped by plugin in `_assetPluginLinks`, and local uploads as `File[]`.

`assets_attach_to_target` in `web/src/lib/stores/asset_store.ts` performs these steps when saving:

1. `assets_create` for new local files.
2. `assets_link` for existing wanderer assets.
3. `assets_import_plugin_links` once for each plugin ID.

The third step posts `trailId`, optional `waypointId` or `summitLogId`, and `assetIds` to `import-to-target`.

The backend then:

1. Requires trail ownership in `pluginSystemAssetCall`. `import-to-target` accepts at most one of `waypointId` and `summitLogId` and validates its association with the trail.
2. Invokes the plugin with `action=import`.
3. Resolves every ID individually through Immich `assetsByID` and `/api/assets/{id}`.
4. Uses `photoFromAsset` to create an original or preview `sdk.Photo` according to `importSize`.
5. Passes the photos and concrete target from `importAssetPluginPhotosToTarget` to `importer.ImportPhotoAssets`.
6. Chooses `copy` or `link_private` through `photoAssetStorageMode`; public trails always use `copy`.
7. Creates the asset or reuses one with the same external identity through `util.CreatePhotoAsset`.
8. Creates the idempotent target link through `LinkAssetToPhotoTargets`.
9. Returns `{ imported, omitted }`, where `imported` contains asset records and external IDs and `omitted` contains provider IDs with reasons; the frontend derives the new photo URLs from the imported records.

Before any plugin call, all import routes trim and stably deduplicate `assetIds` and reject more than 200 unique IDs. The host validates the exact `photos`/`omittedAssetIds` partition before persistence. Manual imports call `assetPluginPhotoImportLimits(..., false)`, so configured per-target photo limits are not enforced for an explicit user selection. In `copy` mode, global importer safety budgets still permit at most 20 fetched media items and 200 MiB per `ImportPhotoAssets` call, at most 50 MiB per item, and any tighter manifest limit. The direct `import-to-target` path uses one such call; the clustered trail import uses a separate call for every cluster.

### 7.7 Photo waypoints in the trail editor

1. `AssetWaypointModal` opens the same photo-library picker with track data.
2. `candidateClusterInput` prefers `pointLat` and `pointLon`, which identify the nearest track point, and falls back to the photo coordinate.
3. `clusterSelectedCandidates` calls `/api/v1/waypoint/cluster` with selected photos, existing waypoints, the trail category, and `resolveNames=true`.
4. `getWaypointMergeSettings` reads `wp_merge_enabled` and `wp_merge_radius` from category settings; the default is enabled with a radius of 50 metres.
5. `clusterWaypointPhotos` assigns a photo to the first cluster whose current center lies within the radius. The center becomes the arithmetic mean after every insertion. With merging disabled, each photo receives its own cluster.
6. With `resolveNames=true`, `resolveWaypointClusterNames` applies the provider-independent `resolveWaypointName` server function to every new spatial cluster. Clusters assigned to existing waypoints are not renamed. The local-photo workflow sets the same flag, while merge-only checks omit it because they do not create photo waypoints.
7. `clusterToWaypoint` reuses an existing waypoint or creates a new frontend waypoint model with the server-provided name and a camera icon. Provider IDs remain in `_assetPluginLinks` until the waypoint is saved.
8. `waypoints_create` or `waypoints_update` saves the waypoint first and then calls `assets_attach_to_target`.

### 7.8 Auto-attach after a trail plugin import

The trigger is in `db/routes/plugin_system_sync.go`.

1. A trail plugin returns detail data and `importer.ImportTrail` creates the trail.
2. When enabled, synchronous trail auto-merge runs first, ensuring that photo attachment targets the surviving trail rather than racing deletion of the source trail.
3. `startBoundedPluginAssetAutoAttach` starts photo work asynchronously with at most four auto-attach jobs active globally.
4. The trigger passes `item.Source.Provider`. Any value other than `upload` or the internal maintenance value uses the `autoAttach.trailPlugins` setting.
5. `autoAttachAssetPluginsForTrail` processes every active asset plugin instance for the user. Unavailable, incorrectly typed, or unauthenticated plugins are logged and skipped.
6. Non-upload providers require `completed=true` on the trail.
7. `autoAttachAssetPluginForTrail` builds a candidate request with `useTrailTime=true`. Without a complete GPX start and end time, nothing is imported automatically.
8. The host calls `candidates` repeatedly with 2,500-item and 2,500-scan limits, threading validated state through at most 25 batches. Empty batches are valid when state progresses; an unchanged state, a cycle, or reaching the cap with `hasMore=true` is a visible error.
9. Candidates from every batch are deduplicated by `assetId` and globally sorted by `distanceFromStart`, distance, ascending valid RFC 3339 capture time with invalid values last, and bytewise `assetId`.
10. `existingTrailAssetExternalIDs` removes provider photos that already exist for the same user and are already linked to the trail.
11. The host invokes `import` in blocks of at most 200 IDs, validates every exact output partition, collects all photos and omissions, and restores candidate rank by `externalId` even if a plugin changes response order.
12. Only after every plugin import block succeeds, `importAssetPluginPhotosForTrail(..., enforceLimits=true)` receives the complete ordered photo set once, clusters, imports, and links photos while enforcing configured host limits.

A failure in one asset plugin does not stop the remaining instances. The public summary records `imported`, `omitted`, and an optional `error` for each plugin, and the same counts are logged.

### 7.9 Auto-attach after GPX upload

`web/src/routes/api/v1/trail/upload/+server.ts` first creates the trail normally through `trails_create`, then makes a best-effort request containing `{trailId, provider:"upload"}` to `/plugins/assets/auto-attach`.

`PluginSystemAssetAutoAttach` validates ownership and immediately returns `202 Accepted`; the actual work runs in a goroutine. `completed` is not required for `provider=upload`. The instance is skipped when `autoAttach.upload=false`. Every later step is identical to the trail-plugin auto-attach workflow.

### 7.10 Maintenance for older trails without photos

The maintenance page uses `GET /plugins/assets/maintenance/trails` and `POST /plugins/assets/maintenance/attach`.

`PluginSystemAssetMaintenanceTrails` checks that at least one active `asset_library.v1` instance exists, loads all completed trails with GPX owned by the user, expands `trail_assets` through `trailAssetMaintenanceState`, excludes trails with at least one visible photo, and treats generated route previews only as possible maintenance-card thumbnails rather than photos.

`PluginSystemAssetMaintenanceAttach` runs synchronously in the request and uses the internal provider value `maintenance`. `autoAttachProviderKey` maps it to the `maintenance` setting, which defaults to enabled when absent. User instances currently expose overrides only for `trailPlugins` and `upload`.

### 7.11 Server-side clustering and automatic waypoints

`importAssetPluginPhotosForTrail` clusters provider photos as follows:

1. `assetPluginPhotoClusters` separates photos with coordinates from standalone photos without coordinates.
2. `assetPluginWaypointClusterContext` loads existing trail waypoints and the trail category.
3. `clusterWaypointPhotos` applies the category's merge settings.
4. A cluster assigned to an existing waypoint imports into that waypoint.
5. A new spatial cluster creates a camera waypoint through `assetPluginWaypointForCluster`.
6. Standalone clusters without coordinates are linked directly to the trail.
7. When limits are enforced, `limitAssetPluginWaypointClusters` preserves existing-waypoint clusters and standalone photos but limits the number of newly created spatial waypoints to `maxWaypoints`.
8. If photo import fails after a waypoint was created, that waypoint is deleted. A newly created waypoint is also deleted when import produces no asset records.

`assetPluginWaypointForCluster` calls the same provider-independent `resolveWaypointName` function as the frontend clustering route. `createAssetPluginWaypoint` stores the resolved name, an empty description, coordinates, `icon=camera`, author, and trail. `assetPluginDistanceFromStart` copies the cumulative distance of the nearest track point.

### 7.12 Automatic waypoint names

`resolveWaypointName` is server-side Wanderer domain logic in `db/routes/waypoint_naming.go`; asset plugins only provide candidate metadata and never determine waypoint names. Both automatic attachment and photo-based waypoint creation in the trail editor use this function with the cluster center and merge radius. It uses this fallback chain:

1. `waypointNameFromOverpassPOI` searches the merge radius for named, relevant OpenStreetMap points of interest.
2. `bestWaypointNameFromOverpass` ranks POI types. Attractions and viewpoints have the highest considered priority; parks and gardens have the lowest. Distance breaks ties.
3. Without a matching POI, `waypointNameFromNominatim` performs reverse geocoding at zoom levels 18, 16, 14, 12, and 10.
4. `waypointNameFromNominatimResponse` prefers the direct name, relevant address fields, and finally the first usable part of `display_name`.
5. The final fallback is `"lat, lon"` with five decimal places.

The default services are `https://overpass-api.de` and `https://nominatim.openstreetmap.org`. `OVERPASS_API_URL`, `PUBLIC_OVERPASS_API_URL`, `NOMINATIM_URL`, or `PUBLIC_NOMINATIM_URL` may replace them. Public default URLs use the SSRF-protected fetcher; explicitly configured trusted services use a client with an eight-second timeout and origin-bound redirects. Responses are limited to 1 MiB. Requests to the public Nominatim host are serialized with at least one second between them.

### 7.13 `copy` storage mode

1. `photoAssetStorageMode` chooses `copy` for unknown modes, sources that cannot be linked permanently, and every public trail.
2. `fetchPhotoFileForAsset` rejects sources that have already expired.
3. `FetchPhotoMedia` loads `url` sources through the public fetcher and `connector` sources through `fetchConnectorMedia`.
4. Connector media is checked against path, authentication context, redirect, optional storage-origin redirect, TLS, and content-type policy.
5. Authentication headers and query values are removed before following an allowed storage redirect to a different origin.
6. `photoFile` derives a safe filename and extension.
7. `util.CreatePhotoAsset` stores the PocketBase file with `storage_mode=copy` and `remote_status=available`.

### 7.14 `link_private` storage and on-demand streaming

A private URL or connector reference without an expiry may be retained without an immediate download.

1. `photoAssetInput` writes `pluginsystem.RemotePhotoAsset` to `metadata.remote` and leaves `file` empty.
2. `util.AssetPublicMediaURL` produces `/api/v1/assets/{asset-id}/file`.
3. `web/src/routes/api/v1/assets/[id]/file/+server.ts` validates the ID, share token, and thumbnail parameter and forwards authentication.
4. `routes.AssetFile` applies the PocketBase view rule. If a local file now exists, it redirects to the PocketBase file endpoint.
5. A remaining private remote asset linked to a public trail is refused as a safety measure.
6. A request with `?thumb=...` invokes the plugin thumbnail action. Otherwise `assetservice.FetchRemotePluginAsset` reconstructs the original `Photo` descriptor from `metadata.remote` and loads it without another WASM call.
7. Success sets `remote_status=available`; 404 and 410 set `missing`; every other failure sets `inaccessible`. Private bytes are returned with `Cache-Control: private, max-age=300` but are not persisted.

Remote retrieval requires the asset plugin to remain installed, the user's instance to remain enabled, and its authentication to remain valid.

### 7.15 Materialization before or after publication

Private provider media must not make anonymous viewing depend on private provider credentials. wanderer covers both transitions.

When a trail becomes public:

1. `MaterializePrivateRemoteAssetLinksAfterPublish` detects `false -> true`.
2. After the update it asynchronously starts `MaterializePrivateRemotePluginAssetsForTrail`.
3. The service gathers all trail, waypoint, and summit-log assets and materializes owned `link_private` photos.

When a remote asset is linked to an already public trail:

1. `MaterializePrivateRemoteAssetOnPublicLink("trail"|"waypoint"|"summit_log")` determines the related trail.
2. Materialization runs synchronously before link insertion when that trail is public.
3. A failed download rejects the link with `400 Bad Request`.

`MaterializeRemotePluginAsset` downloads the bytes, creates a local file, changes `storage_mode` to `copy`, and updates remote status. Historical `metadata.remote` information is retained. Trail merge and asset merge also call `assetservice.EnsurePublicTrailSafeAssetLink`, preserving the same guarantee when links move.

### 7.16 Remote asset management and plugin disabling

For each configured asset instance, the plugin settings page calls `fetchRemoteAssetSummary` and receives data shaped like:

```json
{
  "count": 12,
  "publicCount": 0,
  "missingCount": 1,
  "inaccessibleCount": 2
}
```

`RemotePluginAssetsSummaryForUser` counts only the user's `type=photo`, `external_provider=<plugin>`, `storage_mode=link_private` records.

| Action | Service function | Behavior |
| --- | --- | --- |
| Materialize | `MaterializeRemotePluginAssetsForUser` | Downloads each remote asset and converts it to `copy`, optionally restricted to publicly linked assets. |
| Repair | `RepairRemotePluginAssetsForUser` | Rechecks only `missing` and `inaccessible` assets without storing media bytes. |
| Delete | `DeleteRemotePluginAssetsForUser` | Deletes remote asset records; cascade rules remove their links. |

Routes create process-local jobs through `newRemotePluginAssetsJob` and return `202 Accepted`. The UI polls every 750 ms. Jobs do not survive process restarts, are removed after two hours, and are limited to five concurrently running jobs per user.

Before disabling a plugin, the UI requests the summary. Existing remote links must be downloaded or deleted. Independently of the UI, `preventAssetPluginDisableWithRemoteLinks` and `preventAssetPluginDeleteWithRemoteLinks` reject disabling or deleting an instance while remote assets remain.

### 7.17 Removing individual photos and orphan cleanup

`assets_delete_removed` extracts asset IDs from local and remote photo URLs and sends `DELETE /api/v1/assets/{id}` with the concrete target. `AssetDelete` removes only the matching link. `DeleteAssetIfOrphanedByAuthor` deletes the asset record only when no trail, waypoint, or summit-log link remains.

Before deleting a trail, `DeleteTrailAssetCleanupHandler` gathers affected assets and deletes author-owned orphans after the trail is removed. The maintenance route `DELETE /assets/orphans` can clean up as many as 500 explicitly named, unlinked assets owned by the current user.

### 7.18 Deduplication and asset merge

Import deduplication primarily happens in `CreatePhotoAsset` through external identity. The separate duplicate-maintenance service in `db/services/assetmerge/service.go` additionally recognizes:

- Equal SHA-256 file hashes.
- Equal `(external_provider, external_id)` pairs.
- Legacy source-file keys.
- Equal capture seconds and coordinates rounded to four decimal places.

`SuggestGroups` forms transitively connected groups. `chooseTargetAsset` prefers a local copy, more links, more complete metadata, an older asset, and finally the lexicographically smaller ID. `MergeWithContext` moves all links through `EnsurePublicTrailSafeAssetLink`, fills missing metadata, stores conflicts in a snapshot, and deletes source assets.

## 8. Runtime and network execution

### 8.1 WASM worker

`callAssetPlugin` opens one runtime session per capability call. `RuntimeRegistry.RuntimeFor` selects `WorkerRuntime` for WASM.

1. `WorkerRuntime.OpenSession` reserves a global worker slot and starts the wanderer binary with the `plugin-worker` subcommand.
2. Parent and worker exchange size-bounded JSON frames over stdin and stdout.
3. `workerRuntimeSession.Call` sends the WASM path, export name, Base64 input, and a diagnostic session ID and initializes the call-specific host-request counter.
4. `pluginWorkerProcess.openPlugin` loads the WASM module with Extism and WASI.
5. `handleCallExport` invokes the export. A non-zero result becomes a structured `PluginCallError`.
6. When WASM calls `wanderer.http_request`, the worker sends a host-HTTP RPC to the parent. Only the parent has effective connector policy and authentication.
7. `callAssetPlugin` closes the session and worker process after the response.

Defaults are a two-minute export timeout, a 15-minute session timeout, a 30-second slot-acquisition timeout, and `max(2, runtime.NumCPU())` workers. `WANDERER_PLUGIN_WORKER_*` environment variables configure these runtime boundaries.

Every asset candidate batch and import block opens its own worker session. Candidate continuation therefore lives in the top-level protocol `state`; the browser path stores it behind an opaque host cursor, while auto-attach threads it directly through its server-side loop.

### 8.2 Host HTTP policy

`sdk.HostRequest` serializes `HostRequestSpec` and invokes the WASM host import. `pluginsystem.ExecuteHostRequest` is the central network chokepoint:

1. `InjectHostRequestAuthFromPolicy` injects the authentication context referenced by the manifest.
2. `ValidateAndResolveHostRequestSpec` validates the connector, relative path, query parameters, and requested response limits.
3. `hostRequestBody` creates JSON, form, or permitted multipart content.
4. Upload type and size are checked against the manifest.
5. The connector HTTP client applies private-network and TLS policy.
6. Redirects remain within allowed connector or explicitly configured storage-origin boundaries.
7. Content type and response size are checked before bytes return to the worker.

Immich declares only paths below `/api`, the `api_key` authentication context, a 50 MiB ceiling, and JSON, JPEG, PNG, WebP, and octet-stream response types. JSON API calls use an additional 16 MiB limit. Every runtime `Call` also receives a hard host-request count budget: 4 for asset `check` and `thumbnail`, 24 for `candidates`, `len(assetIds)+8` for `import`, 8 for sync list and trail send, 16 for sync detail, and 4 for session refresh. Missing or invalid values default to 64 and all values are capped at 512. The counter resets for every call even when a sync session is reused, and exceeding it fails the complete call even if the plugin handles the individual host error.

## 9. Authorization and privacy

- Every plugin action, thumbnail, maintenance, and remote-job endpoint requires authentication.
- Plugin candidate search and plugin import with `trailId` currently require trail ownership through `ensureOwnsTrail`, not merely edit access.
- The internal wanderer library also accepts users with explicit `trail_share.permission=edit` through `ensureCanEdit*`.
- A waypoint or summit log must belong to the supplied trail.
- `plugin_instances.auth` is encrypted at rest and censored from ordinary API responses.
- User instances cannot override connector trust data.
- Remote files are served only after the PocketBase view rule passes; asset rules account for share tokens.
- `link_private` is not a public hotlinking mode. Public trails must have local copies.

## 10. Limits, ordering, and idempotency

| Mechanism | Current value or rule |
| --- | --- |
| Track points passed to a plugin | Approximately 2,000 after step-based decimation. |
| Immich search page | 250 assets. |
| Picker plugin candidate batch | At most 100 candidates, 2,500 evaluated assets, and 10 cooperative provider requests. |
| Auto-attach plugin candidate scan | At most 25 batches of 2,500 evaluated assets, for 62,500 evaluated assets total. |
| Browser cursor store | 10-minute sliding TTL, 512 entries, 8 MiB, 8 entries per user, 4 KiB state, 50 batches per cursor. |
| Plugin import request | At most 200 unique IDs after stable deduplication. |
| Runtime host requests | Default 64, absolute ceiling 512, with tighter call-specific budgets. |
| Plugin thumbnail | At most 8 MiB. |
| Thumbnail cache | 24 hours, 512 entries, 64 MiB, least-recently-used-style eviction. |
| Individual medium | 50 MiB by default, additionally bounded by the manifest. |
| Copy import budget | 20 fetched media items and 200 MiB per import operation. |
| Automatic trail photos | Default 20. |
| Automatic waypoint photos | Default 5 per target. |
| Summit-log photo limit | Host default 20; not enforced by the current manual asset plugin import and no automatic summit-log target flow exists. |
| Automatically created waypoints | Immich default 25; current `<=0` semantics mean unlimited. |
| Auto-attach after trail sync | At most four jobs concurrently. |
| Remote jobs | At most five running per user; process-local. |

Idempotency is established at three layers:

1. Candidate IDs are deduplicated within one auto-attach invocation.
2. External IDs already linked to the same trail are removed before `import`.
3. `CreatePhotoAsset` and the database unique index prevent duplicate provider assets, while `EnsureAssetLink` prevents duplicate links.

## 11. Failure semantics

### 11.1 Plugin failures

`assetLibraryV1` uses `fail(code, message)`. The Immich implementation reports `invalid_request` for an input that cannot be decoded, `provider_unavailable` for provider and action failures, and `internal_error` when output serialization fails.

The worker reads structured error JSON and creates `PluginCallError`. The host output type also supports an `error` object, which asset routes normally translate to `400 Bad Request`.

### 11.2 Partial failures

- Manual candidate search retains results from functioning providers when another provider fails.
- Auto-attach records a failure per asset plugin and continues with the remaining instances.
- The copy importer skips individual media that cannot be fetched and continues importing other photos.
- Remote materialization jobs count individual failures in `failed`; the asset status remains available for repair.
- A synchronous link to an already public trail is rejected entirely when required materialization fails.

## 12. API endpoints

The browser-facing `/api/v1` endpoints are implemented by SvelteKit; the second path is the internal PocketBase endpoint.

| Method | Browser path | PocketBase path | Purpose |
| --- | --- | --- | --- |
| POST | `/api/v1/plugins/assets/{plugin}/check` | `/plugins/assets/{plugin}/check` | Validate draft configuration. |
| POST | `/api/v1/plugins/assets/{plugin}/candidates` | `/plugins/assets/{plugin}/candidates` | Search provider candidates. |
| POST | `/api/v1/plugins/assets/{plugin}/import` | `/plugins/assets/{plugin}/import` | Import with trail clustering. |
| POST | `/api/v1/plugins/assets/{plugin}/import-to-waypoint` | `/plugins/assets/{plugin}/import-to-waypoint` | Specific import into one waypoint. |
| POST | `/api/v1/plugins/assets/{plugin}/import-to-target` | `/plugins/assets/{plugin}/import-to-target` | Import into a trail, waypoint, or summit log. |
| GET | `/api/v1/plugins/assets/{plugin}/thumbnail/{id}` | `/plugins/assets/{plugin}/thumbnail/{id}` | Protected provider preview. |
| POST | `/api/v1/assets/library` | `/assets/library` | Internal wanderer photo library. |
| POST | `/api/v1/plugins/assets/auto-attach` | `/plugins/assets/auto-attach` | Start asynchronous auto-attach manually. |
| GET | `/api/v1/plugins/assets/maintenance/trails` | `/plugins/assets/maintenance/trails` | Completed trails without visible photos. |
| POST | `/api/v1/plugins/assets/maintenance/attach` | `/plugins/assets/maintenance/attach` | Synchronous maintenance attachment. |
| GET | `/api/v1/plugins/assets/{plugin}/remote-assets-summary` | Same internal path | Remote asset counts. |
| POST | `/api/v1/plugins/assets/{plugin}/materialize-all` | Same internal path | Start materialization. |
| POST | `/api/v1/plugins/assets/{plugin}/repair-remote-assets` | Same internal path | Start repair. |
| POST | `/api/v1/plugins/assets/{plugin}/delete-remote-assets` | Same internal path | Start deletion. |
| GET | `/api/v1/plugins/assets/jobs/materialize/{id}` | Same internal path | Status for all three remote-job types. |
| GET | `/api/v1/assets/{id}/file` | `/assets/{id}/file` | Redirect to a local file or stream protected remote media. |
| DELETE | `/api/v1/assets/{id}` | `/assets/{id}` | Remove a target link and, when orphaned, its asset. |

The generic SvelteKit action proxy has a fixed method and action allowlist. New host actions must be added explicitly.

## 13. Function reference

### 13.1 Provider entry point: `plugins/immich/main.go`

| Function | Detailed responsibility |
| --- | --- |
| `assetLibraryV1` | Reads Extism input into `assetLibraryInput`, delegates to `handleAssetLibrary`, writes JSON output, and returns the WASM status code. |
| `fail` | Serializes `sdk.PluginError` into the Extism error string and falls back to plain text if serialization fails. |

### 13.2 Provider operations and pagination: `plugins/immich/immich.go`, `plugins/immich/pagination.go`

| Function | Detailed responsibility |
| --- | --- |
| `handleAssetLibrary` | Central action dispatcher; an empty action currently falls back to `candidates`. |
| `immichClient.check` | Validates `/api/users/me` and a small metadata search, then returns the Immich user ID. |
| `immichClient.candidates` | Normalizes configuration, determines the time window, executes one bounded resumable search, and returns candidates, state, and scan statistics. |
| `immichClient.importAssets` | Resolves selected IDs one at a time and creates either an `sdk.Photo` descriptor or an explicit `sdk.OmittedAsset`; an empty selection is a successful no-op. |
| `immichClient.thumbnail` | Builds a preview connector reference for the first ID without fetching bytes itself. |
| `immichClient.searchAssets` | Small page-limited metadata search retained for `check`. |
| `immichClient.searchAssetCandidates` | Maps SDK limits, builds the Immich metadata request, and supplies provider pages to the SDK-independent pagination state machine. |
| `searchAssetCandidatePages` | Enforces candidate, evaluated-item, and provider-request limits, resumes at the raw offset within a provider page, and returns candidates, statistics, and continuation state. |
| `assetCandidateStateInt` | Decodes numeric state fields from ordinary JSON representations without depending on the plugin SDK. |
| `nextCandidatePage` | Parses Immich's string page number and rejects a malformed or non-progressing value. |
| `candidateSearchState` | Constructs Immich's provider-specific `{page, offset}` continuation state. |
| `immichClient.assetsByID` | Calls `/api/assets/{id}` sequentially for every selected ID. |
| `immichClient.getJSON` | Sends an authenticated connector GET, expects no more than 16 MiB of JSON, and requires a 2xx response. |
| `immichClient.postJSON` | Equivalent JSON POST used by metadata search. |
| `configFromInput` | Reads dynamically typed plugin configuration and applies safe defaults. |
| `candidateWindow` | Prefers an explicit time range; otherwise expands complete track times by `timeWindowMinutes`. |
| `explicitRequestTimeWindow` | Parses individual or paired RFC 3339 boundaries; parsing failure means no explicit window. |
| `requestTrailTimeWindow` | Accepts only a complete, parseable `startedAt` and `endedAt` pair. |
| `photoFromAsset` | Maps external ID, filename, time, and optional EXIF coordinates into `sdk.Photo`. |
| `mediaSourceForAsset` | Selects the Immich original or preview path and sets connector, authentication, query, and asset ID. |
| `contentTypeForImportSize` | Fixes preview content type to `image/jpeg`; original imports let the response declare the type. |
| `previewFilename` | Replaces an existing extension with `.jpg` and uses ID-based fallbacks. |
| `normalizedImportSize` | Preserves only case-insensitive `original`; every other value becomes `preview`. |
| `hasPointTimestamps` | Reports whether at least one track point contains a timestamp. |
| `stringConfig`, `boolConfig`, `intConfig` | Read dynamic JSON configuration with trimming, type conversion, and fallbacks. |

### 13.3 Provider matching: `plugins/immich/matching.go`

| Function | Detailed responsibility |
| --- | --- |
| `matchAssetCandidates` | Applies owner, EXIF-coordinate, and radius filters; `doubleRadius` doubles the provider radius. |
| `sortMatches` | Orders by ascending distance and, for equal distance, descending capture time. |
| `candidateForAsset` | Finds the nearest track point or compares against one coordinate and constructs the candidate. |
| `haversineMeters` | Computes great-circle distance using an Earth radius of 6,371,000 metres. |
| `degreesToRadians` | Converts degrees to radians. |

### 13.4 Host invocation and request construction: `db/routes/plugin_system_assets.go`

| Function or group | Detailed responsibility |
| --- | --- |
| `PluginSystemAssetCheck` | Validates an unsaved or saved configuration draft. |
| `PluginSystemAssetCandidates`, `PluginSystemAssetImport`, `PluginSystemAssetImportToWaypoint`, `PluginSystemAssetImportToTarget` | Thin action wrappers around `pluginSystemAssetCall`. |
| `pluginSystemAssetCall` | Binds and validates request and target, stably deduplicates and caps imports at 200 IDs, checks ownership, invokes the plugin, validates exact import partitions, and dispatches the appropriate host path. |
| `callAssetPluginCandidates` | Resolves the browser cursor, binds it to invocation context, applies picker search limits, validates state progress, and returns a DTO without raw state or stats. |
| `stableUniqueAssetIDs`, `normalizeAssetPluginImportIDs` | Trim IDs, preserve first-occurrence order, and enforce the 200-unique-ID request limit. |
| `validateAssetPluginImportPartition` | Requires `photos.externalId` and `omittedAssetIds.assetId` to form a unique, disjoint, exact partition of requested IDs. |
| `assetPluginInvocationForUser` | Loads the active instance, asset plugin, capability, decrypted authentication, and effective configuration. |
| `assetPluginDraftInvocation` | Builds equivalent context for a settings draft without requiring an enabled instance. |
| `mergeAssetPluginDraftConfig` | Applies only permitted instance overrides to effective configuration. |
| `mergeSubmittedPluginAuth` | Replaces stored authentication only with non-empty, non-null submitted values. |
| `assetPluginConnectorConfig` | Delegates safe derivation of connector configuration from the plugin URL. |
| `callAssetPlugin` | Opens a worker session, filters authentication and configuration, serializes input, invokes the export, and decodes output. |
| `newAssetLibraryActionInput` | Copies request fields into the plugin protocol. |
| `assetLibraryActionInputForApp` | Adds track points, preferring explicit trail XML over stored GPX; auto-attach may use track times as the search window. |
| `assetLibraryActionInputForWandererLibrary` | More tolerant internal-library variant that logs some GPX failures instead of aborting. |
| `trailTrackPoints`, `trailTrackPointsFromBytes` | Read and parse GPX into positions, cumulative distance, and time boundaries. |
| `applyAssetLibraryTrackPoints` | Applies geometry without implicitly creating a time range. |
| `applyAssetLibraryExplicitTimeWindow` | Validates explicit boundaries and mirrors them into start and end. |
| `applyAssetLibraryTrailTimeWindow` | Sets a window only when the GPX provides both boundaries. |
| `decimateTrackPoints` | Keeps every `ceil(len/limit)`-th point. |

The cursor implementation in `db/routes/plugin_system_asset_cursor.go` provides `newPluginAssetCursorBinding` for query/configuration/authentication fingerprints, `validatePluginAssetContinuation` for canonical progress and cycle checks, and `pluginAssetCursorStore` for TTL, LRU-style eviction, per-user, memory, state-size, and batch limits.

### 13.5 Internal library and thumbnails: `db/routes/plugin_system_assets.go`

| Function or group | Detailed responsibility |
| --- | --- |
| `AssetLibraryCandidates` | Returns the user's wanderer photos not already linked to the target plus external identities for frontend deduplication. |
| `assetLibraryLinkedAssetIDs` | Collects existing links for every target named in the request. |
| `assetLibraryPaginationForRequest` | Enables page/offset pagination only when neither coordinate nor track is present. |
| `assetLibraryRecords` | Builds database filters for author, type, date, and spatial bounding box. |
| `assetLibraryExistingExternalRefs` | Returns every external identity belonging to the actor. |
| `assetLibraryCoordinateBounds` | Expands track or point bounds by the search radius and handles poles and the date line conservatively. |
| `assetLibraryCandidate` | Excludes generated route previews and records without usable URL or coordinate and calculates proximity to the track. |
| `nearestAssetLibraryTrackPoint` | Performs a linear nearest-point search over supplied track points. |
| `sortAssetLibraryCandidates` | Orders by track position, distance, newest capture time, and asset ID. |
| `PluginSystemAssetThumbnail` | Authenticated provider thumbnail endpoint. |
| `fetchPluginAssetThumbnailEntryForUser` | Resolves plugin context and coordinates cache and singleflight behavior. |
| `fetchPluginAssetThumbnailEntryUncached` | Obtains a plugin descriptor and fetches media under host policy. |
| `getPluginAssetThumbnailCache`, `putPluginAssetThumbnailCache` | Enforce TTL and size and perform eviction. |
| `writePluginAssetThumbnail`, `pluginAssetThumbnailETag` | Produce private cache headers, conditional responses, and the ETag. |

### 13.6 Auto-attach and import: `db/routes/plugin_system_assets.go`

| Function or group | Detailed responsibility |
| --- | --- |
| `PluginSystemAssetAutoAttach` | Ownership-checked asynchronous start returning `202 Accepted`. |
| `PluginSystemAssetMaintenanceTrails` | Lists completed GPX trails owned by the user without visible non-preview photos. |
| `PluginSystemAssetMaintenanceAttach` | Runs auto-attach synchronously for one maintenance trail. |
| `enabledAssetPluginCount` | Counts active, available `assets` plugins with `asset_library.v1`. |
| `trailAssetMaintenanceState` | Determines visible photos and an optional generated preview thumbnail per trail. |
| `autoAttachAssetPluginsForTrail` | Iterates active asset instances, checks provider flags, and isolates each plugin result. |
| `autoAttachAssetPluginForTrail` | Collects and validates all candidate batches, globally deduplicates and sorts, filters existing links, imports in 200-ID blocks, validates and reorders the complete result, and invokes host persistence once. |
| `validateAssetPluginCandidateBatch` | Rejects a plugin response that exceeds the caller's `maxItems`. |
| `sortAssetPluginAutoAttachCandidates` | Applies the batch-independent total order used before import and clustering. |
| `assetPluginAutoAttachAllowedForTrail` | Always allows uploads and otherwise requires completed trails. |
| `assetPluginAutoAttachHasTimeWindow` | Requires both time boundaries. |
| `assetPluginProviderEnabled`, `autoAttachProviderKey`, `autoAttachEnabled` | Map triggers to instance flags; missing values are enabled. |
| `existingTrailAssetExternalIDs` | Intersects candidate IDs with provider assets already linked to the trail. |
| `importAssetPluginPhotosToTarget` | Direct manual import into the explicitly selected target. |
| `importAssetPluginPhotosForTrail` | Shared import path for a waypoint target or trail clustering. |
| `assetPluginPhotoImportLimits` | Returns enforceable limits only when `enforce=true`. |
| `assetPluginPhotoClusters` | Prepares provider photos for common waypoint clustering. |
| `assetPluginWaypointForCluster` | Reuses an existing waypoint or creates a named camera waypoint. |
| `assetPluginMaxWaypoints`, `limitAssetPluginWaypointClusters` | Read and enforce the limit for new automatically generated waypoints. |
| `assetPluginWaypointClusterContext` | Loads trail waypoints and category. |
| `createAssetPluginWaypoint` | Persists an automatically generated waypoint. |
| `assetPluginDistanceFromStart` | Copies the nearest track point's cumulative distance. |

### 13.7 Authorization: `db/routes/plugin_system_assets.go`

| Function or group | Detailed responsibility |
| --- | --- |
| `userOwnsTrail`, `userOwnsWaypoint`, `userOwnsSummitLog` | Validate actor/user ownership and target-to-trail consistency. |
| `userCanEditTrailTarget` | Extends ownership with `trail_share.permission=edit`. |
| `userCanEditWaypointTarget`, `userCanEditSummitLogTarget` | Accept ownership of the child object or edit permission on its trail. |
| `actorBelongsToUser` | Resolves an actor ID to its local user. |
| `ensureOwns*`, `ensureCanEdit*` | Convert boolean authorization checks into API errors. |

### 13.8 Importer: `db/plugins/importer/importer.go`

| Function | Detailed responsibility |
| --- | --- |
| `ImportPhotoAssets`, `importPhotoAssets` | Apply optional per-target limits and the copy-media budget, build storage inputs, and persist photos. |
| `photoAssetInput` | Converts `pluginsystem.Photo` to `util.PhotoAssetInput`, including remote metadata. |
| `photoAssetStorageMode` | Forces a local copy for public targets and sources that cannot be linked permanently. |
| `isRemoteLinkablePhoto` | Accepts a non-expiring URL or connector reference with a path. |
| `fetchPhotoFileForAsset` | Checks expiry and loads a PocketBase file. |
| `maxPhotosForAssetTarget` and helpers | Select the trail, waypoint, or summit-log limit. |
| `photoFile` | Fetches media and creates a safe filename. |
| `FetchPhotoMedia` | Dispatches URL or connector retrieval and applies the effective size limit. |
| `fetchConnectorMedia` | Resolves connector and authentication and handles safe redirects. |
| `fetchStorageRedirectMedia` | Follows an explicitly permitted storage-origin redirect without provider credentials. |
| `effectivePluginMediaMaxBytes` | Uses the lower of the request and manifest limits. |
| `pluginMediaResponse` | Validates content type, reads through a size limit, and returns the final URL. |
| `stripConnectorAuth` | Removes authorization and configured authentication headers and query parameters before a storage redirect. |
| `safeMediaFileName` | Removes path and special-character hazards and adds an appropriate extension. |

### 13.9 Asset persistence: `db/util/assets.go`

| Function | Detailed responsibility |
| --- | --- |
| `CreatePhotoAsset` | Validates storage input and author, deduplicates external identity, creates the asset, and links targets. |
| `reuseExternalPhotoAsset` | Adds a local file or capture time when missing and links additional targets. |
| `findExistingExternalPhotoAsset` | Searches by author, provider, external ID, and photo type. |
| `LinkAssetToPhotoTargets` | Creates all links implied by the storage input. |
| `PhotoAssetLinkTargets` | Chooses a direct trail link or waypoint/summit-log link. |
| `EnsureAssetLink` | Idempotent find-or-create relation. |
| `ResolveAssetAuthor`, `AssetAuthorUserID` | Convert between user and actor identities for asset operations. |
| `IsAssetLinkedToPublicTrail` | Checks every direct and indirect trail relation. |
| `AssetPublicMediaURL`, `AssetPublicFileURL` | Select a local file path or remote file endpoint. |
| `AssetIDsForTrail`, `TrailIDsForAsset` | Traverse direct, waypoint, and summit-log links. |
| `DeleteAssetIfOrphanedByAuthor` | Deletes only author-owned assets without remaining links. |

### 13.10 Remote service: `db/services/assets/service.go`

| Function | Detailed responsibility |
| --- | --- |
| `RemotePluginAssetsSummaryForUser` | Counts remote assets and their problematic or public subsets. |
| `MaterializeRemotePluginAssetsForUser` | Optionally restricts to public links and materializes assets in batches. |
| `RepairRemotePluginAssetsForUser` | Retests problematic sources and updates status. |
| `DeleteRemotePluginAssetsForUser` | Deletes matching remote records with progress reporting. |
| `MaterializePrivateRemotePluginAssetsForTrail` | Materializes every owned private remote photo attached to a trail. |
| `EnsurePublicTrailSafeAssetLink` | Materializes when necessary and creates the link only afterward. |
| `MaterializePrivateRemotePluginAssetForPublicLink` | Resolves the target trail and delegates only when `public=true`. |
| `MaterializeRemotePluginAsset` | Fetches media, stores the file, and changes storage mode to `copy`. |
| `FetchRemotePluginAsset` | Reconstructs provider context and retrieves stored remote media. |
| `MarkAssetRemoteStatus` | Maintains status, check time, error, and missing-since time. |
| `RemoteStatusForError` | Maps 404 and 410 to `missing` and all other failures to `inaccessible`. |
| `processRemotePluginAssets` | Shared batching and progress logic. |
| `RemotePhotoAssetFromRecord` | Decodes and validates `metadata.remote`. |

### 13.11 Remote routes and lifecycle hooks

| Function | File | Detailed responsibility |
| --- | --- | --- |
| `PluginSystemAssetRemoteAssetsSummary` | `db/routes/assets.go` | Validates plugin type and returns the user's remote summary. |
| `PluginSystemAssetMaterializeAll` | `db/routes/assets.go` | Starts materialization, optionally with `publicOnly`. |
| `PluginSystemAssetRepairRemoteAssets` | `db/routes/assets.go` | Starts repair for problematic sources. |
| `PluginSystemAssetDeleteRemoteAssets` | `db/routes/assets.go` | Starts deletion of remote asset records. |
| `PluginSystemAssetMaterializeStatus` | `db/routes/assets.go` | Returns a snapshot only to the job owner. |
| `AssetFile` | `db/routes/assets.go` | Authorizes retrieval, redirects local files, or streams and marks remote media. |
| `newRemotePluginAssetsJob` | `db/routes/assets.go` | Creates a cryptographic job ID, removes old jobs, and enforces the per-user limit. |
| `updateRemotePluginAssetsJob`, `remotePluginAssetsJobSnapshot` | `db/routes/assets.go` | Synchronize mutation and return a defensive copy of in-memory job state. |
| `MaterializePrivateRemoteAssetLinksAfterPublish` | `db/hooks/trails.go` | Starts materialization after a private-to-public transition. |
| `MaterializePrivateRemoteAssetOnPublicLink` | `db/hooks/trails.go` | Materializes synchronously before linking to an already public trail. |
| `preventAssetPluginDisableWithRemoteLinks` | `db/hooks/plugin_instances.go` | Rejects `enabled: true -> false` while remote links remain. |
| `preventAssetPluginDeleteWithRemoteLinks` | `db/hooks/plugin_instances.go` | Rejects instance deletion under the same condition. |
| `encryptPluginInstanceAuth`, `censorPluginInstanceAuth` | `db/hooks/plugin_instances.go` | Encrypt stored secrets and remove them from user responses. |
| `ReindexTrailOnAssetChange` | `db/hooks/assets.go` | Reindexes every trail affected by a changed asset. |
| `ReindexTrailOnAssetLinkChange` | `db/hooks/assets.go` | Debounces reindexing after a link change. |

### 13.12 Clustering and name resolution

| Function | File | Detailed responsibility |
| --- | --- | --- |
| `WaypointCluster` | `db/routes/waypoint_cluster.go` | Validates the frontend clustering request and returns settings plus clusters; when `resolveNames=true`, new spatial clusters include server-resolved names. |
| `getWaypointMergeSettings` | `db/routes/waypoint_cluster.go` | Reads category settings with defaults of `enabled=true` and a 50-metre radius. |
| `clusterWaypointPhotos` | `db/routes/waypoint_cluster.go` | Assigns photos to existing or dynamically centered new clusters. |
| `newWaypointPhotoCluster`, `newWaypointCluster` | `db/routes/waypoint_cluster.go` | Initialize photo and existing-waypoint clusters. |
| `addPhotoToWaypointCluster` | `db/routes/waypoint_cluster.go` | Adds an ID and recalculates the arithmetic center. |
| `resolveWaypointClusterNames` | `db/routes/waypoint_cluster.go` | Resolves names for new spatial clusters while leaving existing-waypoint and empty clusters unchanged. |
| `resolveWaypointName` | `db/routes/waypoint_naming.go` | Orchestrates Overpass, Nominatim, and coordinate fallback. |
| `waypointNameFromOverpassPOI` | `db/routes/waypoint_naming.go` | Builds an Overpass URL, loads JSON, and chooses the best POI. |
| `bestWaypointNameFromOverpass` | `db/routes/waypoint_naming.go` | Filters by radius and ranks POI category before distance. |
| `waypointNameFromNominatim` | `db/routes/waypoint_naming.go` | Tries several zoom levels until a usable name is found. |
| `waypointNameFromNominatimResponse` | `db/routes/waypoint_naming.go` | Extracts a direct name, address hierarchy, or display-name component. |
| `nominatimRateLimit` | `db/routes/waypoint_naming.go` | Serializes public Nominatim requests with at least one second between calls. |
| `geocodingFetchJSON` | `db/routes/waypoint_naming.go` | Selects the SSRF-protected default fetcher or an explicitly trusted service client. |
| `geocodingExternalServiceURL` | `db/routes/waypoint_naming.go` | Normalizes a base URL, joins the path safely, and encodes query parameters. |
| `cleanWaypointName` | `db/routes/waypoint_naming.go` | Trims and normalizes whitespace and limits names to 120 Unicode code points. |

### 13.13 Runtime, policy, and effective configuration

| Function | File | Detailed responsibility |
| --- | --- | --- |
| `RuntimeRegistry.RuntimeFor` | `db/pluginsystem/runtime.go` | Selects the WASM worker runtime from the manifest. |
| `WorkerRuntime.OpenSession` | `db/pluginsystem/worker.go` | Reserves a slot, starts the worker process, and configures size and time boundaries. |
| `EffectiveMaxHostRequests` | `db/pluginsystem/runtime.go` | Applies the safe default of 64 and absolute ceiling of 512 to call options. |
| `workerRuntimeSession.Call` | `db/pluginsystem/worker.go` | Serializes calls, applies export timeout, and passes a fresh per-call host-request budget into the worker RPC loop. |
| `workerRuntimeSession.handleHostHTTPRequest` | `db/pluginsystem/worker.go` | Executes worker HTTP RPC in the parent with parent-owned policy. |
| `pluginWorkerProcess.handleCallExport` | `db/pluginsystem/worker_process.go` | Loads WASM once per session and invokes the requested export. |
| `pluginWorkerProcess.hostFunctions` | `db/pluginsystem/worker_process.go` | Registers only `wanderer.http_request` and `wanderer.log`. |
| `ExecuteHostRequest` | `db/pluginsystem/host_http.go` | Central policy-controlled network operation. |
| `ResolveRequestTarget`, `ValidateConnectorRedirect` | `db/pluginsystem/policy.go` | Resolve declared connectors and keep paths and redirects inside allowed scope. |
| `pluginhost.EffectiveConfig` | `db/services/pluginhost/config.go` | Combines administrator defaults with permitted instance values. |
| `pluginhost.InstanceConfigOverrides` | `db/services/pluginhost/config.go` | Projects only user-owned plugin and host fields. |
| `pluginhost.InstancePolicy` | `db/services/pluginhost/config.go` | Resolves manifest connectors against administrator host configuration. |
| `pluginhost.AssetConnectorConfig` | `db/services/pluginhost/config.go` | Safely derives a base URL from plugin configuration when the administrator target is empty. |
| `pluginhost.DecryptedInstanceAuth` | `db/services/pluginhost/config.go` | Decrypts stored values for host and plugin invocation. |
| `pluginhost.PhotoImportLimits` | `db/services/pluginhost/config.go` | Normalizes positive host limits with system defaults. |

### 13.14 Frontend

| Function | File | Detailed responsibility |
| --- | --- | --- |
| `loadCandidates` | `photo_library_picker_modal.svelte` | Runs local and plugin searches in parallel and handles partial failures. |
| `loadPluginCandidates`, `loadMorePluginCandidates` | `photo_library_picker_modal.svelte` | Invoke initial and cursor-based provider batches, keep per-plugin pagination state, restart expired cursors, and merge identity and thumbnail information. |
| `loadWandererCandidates` | `photo_library_picker_modal.svelte` | Loads the internal library with pagination and known external references. |
| `uniqueCandidates` | `photo_library_picker_modal.svelte` | Deduplicates across sources and orders candidates. |
| `photoLibraryPluginLinks` | `web/src/lib/models/photo_library.ts` | Groups a selection by plugin ID and unique asset IDs. |
| `assets_import_plugin_links` | `web/src/lib/stores/asset_store.ts` | Runs `import-to-target` once per plugin and consumes the `{imported, omitted}` response envelope. |
| `assets_attach_to_target` | `web/src/lib/stores/asset_store.ts` | Orchestrates uploads, local links, and plugin imports. |
| `assets_delete_removed` | `web/src/lib/stores/asset_store.ts` | Removes target-specific links and then possible orphans. |
| `assetPhotoURL` | `web/src/lib/util/asset_link_util.ts` | Selects a PocketBase file or remote file endpoint. |
| `applyAssetPluginCheckResult` | `plugin_instance_settings_modal.svelte` | Runs `check` and stores the hidden provider user ID. |
| `startRemoteAssetJob`, `pollMaterializeJob` | `web/src/routes/settings/plugins/+page.svelte` | Start and poll materialize, repair, and delete jobs. |

## 14. Tests and verification

| File | Existing coverage |
| --- | --- |
| `plugins/immich/matching_test.go` | Immich candidate filtering, ordering, and output shape. |
| `plugins/immich/pagination_test.go` | Dense-page continuation, strict mid-page scan limits, raw-asset offsets, provider-request limits, and monotonic `nextPage` validation without TinyGo build tags. |
| `db/routes/plugin_system_assets_test.go` | Trigger flags, time windows, GPX points, internal pagination, cursor binding and state validation, import partitions and limits, global ordering, clustering, naming fallbacks, and browser output shape. |
| `db/routes/waypoint_cluster_test.go` | Server-side naming of new spatial clusters without renaming existing-waypoint or empty clusters. |
| `db/pluginsystem/worker_test.go` | Runtime request defaults and ceilings, whole-call budget failure, per-call counter reset, and the maximum 200-photo import request count. |
| `db/routes/plugin_system_policy_test.go` | Host connector configuration and protection against connector overrides during draft checks. |
| `db/plugins/importer/importer_test.go` | Media policy, limits, photo metadata mapping, and trail import helpers. |
| `db/util/assets_test.go` | Asset creation, reuse, links, URLs, and cleanup. |
| `db/services/assetmerge/service_test.go` | Duplicate suggestions, link movement, metadata, and hash persistence. |
| `db/migrations/asset_share_rules_test.go` | Asset and link rules and the external identity unique index. |

Changes to this stack should run at least the Go tests for `plugins/immich`, `db/routes`, `db/plugins/importer`, `db/util`, and `db/services/assets`, or the corresponding full `go test ./...`, plus relevant frontend and documentation builds.

## 15. Implementing another asset plugin

1. Create a manifest with `type=assets` and `asset_library.v1`.
2. Declare every provider target as a connector and every required authentication context; do not assume direct network access.
3. Implement one export with all four actions and reject unknown actions explicitly.
4. Return stable `assetId` and `externalId` values; robust idempotency depends on them.
5. Return candidates with reliable positions when claiming spatial matching and set `pointLat` and `pointLon` to the track point actually used.
6. Do not embed media bytes in `import` JSON. Return a `url` or, preferably, a declared connector `MediaRef`.
7. Use non-expiring sources for durable `link_private` support; expiring sources are copied automatically.
8. Keep `thumbnail` small and deterministic. The host cache is scoped by user, instance, and asset ID.
9. Bound provider pagination with `search`, return a progressing state whenever `hasMore=true`, resume within provider pages without skipping raw assets, and report evaluated items in `stats.scannedItems`.
10. Make `check` detect an invalid URL, credentials, and essential read permissions.
11. Cover matching, import, failure, and policy behavior with plugin and host tests.

## 16. Known implementation properties

- Plugin candidate cursors are process-local and intentionally disappear on backend restart; the picker handles this through `restartRequired` and a clean provider restart.
- Provider candidate scans have weak snapshot consistency. Immich page changes caused by deletion or EXIF date edits can skip an item across calls, while duplicate results are removed by provider identity.
- Immich resolves selected import IDs sequentially with one metadata request per ID.
- Remote jobs and the thumbnail cache are process-local and are neither persisted nor shared between backend instances.
- Publication materialization runs asynchronously after a trail update. During that interval, the file endpoint refuses remaining `link_private` assets on public trails.
- Manual plugin endpoints require trail ownership, while the internal library supports edit shares.
- Auto-attach requires a complete GPX time span; geometry without timestamps is insufficient.
- `maxWaypoints<=0` currently means unlimited in `limitAssetPluginWaypointClusters`, although the manifest field is presented as a maximum.
