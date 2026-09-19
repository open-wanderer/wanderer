# Immich Plugin

Suggests geotagged Immich photos that match a wanderer trail or waypoint and
imports the selected photos as wanderer asset records.

The plugin exposes `asset_library.v1` and uses the host HTTP connector named
`api`. The host owns the configured Immich base URL and injects the API key.

## Configuration

Create an API key in Immich, then configure the plugin from wanderer's plugin
settings page:

The API key only needs these Immich permissions:

| Permission | Used for |
| --- | --- |
| `user.read` | Calls `/api/users/me` and resolves the current Immich user for "Own photos only". |
| `asset.read` | Searches `/api/search/metadata` and reads `/api/assets/{id}` metadata. |
| `asset.view` | Loads `/api/assets/{id}/thumbnail` previews and imports photos when import size is `Preview`. |
| `asset.download` | Loads `/api/assets/{id}/original` when import size is `Original`. |

No write, delete, upload, album, library, or admin permissions are required.

| Setting | Meaning |
| --- | --- |
| Immich server URL | Base URL of the Immich server. |
| API key | Immich API key. Stored encrypted in wanderer. |
| Time window | Minutes around the trail or waypoint time range to search. |
| Search radius | Maximum distance in meters between a photo and a trail point. |
| Maximum waypoints | Maximum number of photo waypoints to create from a trail search. |
| Import size | `Preview` stores Immich preview images; `Original` stores the original file. |
| Own photos only | Restrict candidates to the Immich user returned by the plugin check action. |

Host-owned settings control storage and automatic attachment:

| Setting | Meaning |
| --- | --- |
| Photo mode | `copy` or `link_private`. |
| Auto attach trail plugins | Attach matching photos after completed trail plugin imports with track timestamps. |
| Auto attach upload | Attach matching photos after GPX uploads with track timestamps. |

New photo waypoints are merged using the trail category's waypoint merge radius.
Their names come from nearby OpenStreetMap points of interest when possible,
falling back to reverse geocoding and then the photo coordinate.

## Actions

The `asset_library.v1` export handles these actions:

| Action | Meaning |
| --- | --- |
| `check` | Calls Immich `/api/users/me` and performs a tiny search to validate auth and connector settings. |
| `candidates` | Searches Immich image assets with EXIF coordinates and ranks them against the provided trail points or coordinate. |
| `import` | Returns `sdk.Photo` descriptors for selected Immich assets using the configured import size. |
| `thumbnail` | Returns a connector-backed preview descriptor for one asset. |

The plugin never contacts Immich directly. All Immich requests go through the
wanderer host request boundary, which enforces the manifest connector policy,
content-type limits, auth injection, private-network policy, TLS settings, and
storage redirect policy.
