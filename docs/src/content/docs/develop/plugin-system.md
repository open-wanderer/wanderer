---
title: Plugin System
description: Build, install, and run WASM provider plugins in wanderer
---

Plugins let wanderer connect to external providers such as Strava, komoot, and
Hammerhead without adding provider-specific API code to the core application.

A plugin is a local directory with a `plugin.json` manifest and a WASM
entrypoint:

```text
data/plugins/
  strava/
    plugin.json
    plugin.wasm
    icon.svg
```

wanderer discovers plugins from direct child directories of `data/plugins`.
Plugin configuration, credentials, sync state, and status are stored per user in
`plugin_instances`.

## Quickstart

Use an existing first-party plugin as a starting point:

- [Hammerhead plugin source](https://github.com/open-wanderer/wanderer/tree/main/plugins/hammerhead)
- [komoot plugin source](https://github.com/open-wanderer/wanderer/tree/main/plugins/komoot)
- [Strava plugin source](https://github.com/open-wanderer/wanderer/tree/main/plugins/strava)

For local development:

```sh
make plugins-build
make plugins-install-local
```

Start wanderer and open the plugin settings page. The plugin should appear once
its bundle exists at:

```text
data/plugins/<plugin-id>/plugin.json
data/plugins/<plugin-id>/plugin.wasm
```

## 1st-party plugins

First-party plugin source lives in the repository under `plugins/`:

```text
plugins/
  hammerhead/
  komoot/
  strava/
  sdk/
```

Build all bundled plugins:

```sh
make plugins-build
```

Build and install them into the local runtime directory:

```sh
make plugins-install-local
```

Package release archives:

```sh
make plugins-package
```

Release archives are published as separate GitHub release assets. The database
Docker image does not contain provider plugins.

## Plugin layout

A provider plugin should use this layout:

```text
plugins/<provider>/
  go.mod
  plugin.json
  main.go
  assets/icon.svg
  Makefile
```

Generated runtime files are written to `dist/<plugin-id>/` and are ignored by
git:

```text
plugins/strava/dist/strava/
  plugin.json
  plugin.wasm
  icon.svg
```

The generated `dist/<plugin-id>` directory is the directory users install below
`data/plugins`.

Icons are referenced from `plugin.json` metadata and copied from `assets/` into
the dist directory by the plugin `Makefile`:

```json
{
  "metadata": {
    "icons": {
      "light": "icon.svg",
      "dark": "icon_dark.svg"
    }
  }
}
```

`dark` is optional.

## Go SDK

Go/TinyGo plugins should import the plugin SDK:

```go
import "github.com/open-wanderer/wanderer/plugins/sdk"
```

The SDK contains plugin-side protocol types and host-function helpers. It does
not depend on wanderer core or PocketBase.

SDK versions are tagged as `plugins/sdk/vX.Y.Z`. A wanderer release that
changes the SDK tags the next SDK version on its own commit and names it in its
release notes. New fields in an SDK version take effect from that release on;
older hosts ignore them.

Most plugins use:

- `sdk.HostRequest` for provider API calls through `wanderer.http_request`
- `sdk.Get` and `sdk.PostJSON` convenience helpers
- `sdk.HostRequestSpec`, `sdk.ResponseExpect`, and multipart body constants
- auth/header constants such as `sdk.AuthHeaderAuthorization`

## Manifest

Each plugin must define a static `plugin.json` manifest. The manifest is the
security and capability contract used by the host.

The repository includes a JSON Schema at
`plugins/schema/plugin.schema.json`. Add a `$schema` field in source manifests
to get editor completion and inline validation:

```json
{
  "$schema": "../schema/plugin.schema.json"
}
```

Minimal shape:

```json
{
  "manifestVersion": "1.0",
  "id": "example",
  "type": "trails",
  "name": "Example",
  "version": "0.1.0",
  "runtime": {
    "type": "wasm",
    "entrypoint": "plugin.wasm"
  },
  "capabilities": [
    {
      "name": "list_routes",
      "version": "v1",
      "export": "list_routes_v1"
    },
    {
      "name": "get_route_detail",
      "version": "v1",
      "export": "get_route_detail_v1"
    }
  ],
  "permissions": {
    "network": {
      "connectors": [
        {
          "name": "api",
          "type": "public_api",
          "fixedBaseURL": "https://api.example.com",
          "allowedPathPrefixes": ["/v1"]
        }
      ]
    },
    "downloads": {
      "maxBytes": 1048576,
      "contentTypes": ["application/json"]
    }
  }
}
```

Important rules:

- `type` is the functional plugin category. Currently only `trails` is supported.
- `runtime.entrypoint` must be relative to the plugin directory.
- `id` must match the installed directory name by convention.
- `capabilities[].export` names the WASM export the runtime calls.
- `permissions.network.connectors` declares every provider target the plugin may
  request through the host.
- per-request limits may narrow manifest limits, but never expand them.
- `configSchema[].required` marks plugin-owned settings that the settings UI
  must collect before saving.

### Network connectors

Provider HTTP is connector-based. Plugins do not send absolute provider URLs to
the host; they name a connector and a relative path. The host resolves that
connector to a concrete base URL, validates the path scope, injects auth, and
executes the request.

Connector types:

| Type | Purpose |
| --- | --- |
| `public_api` | Fixed public provider API declared in the manifest. Use this for SaaS APIs such as Strava, komoot, or Hammerhead. |
| `configured` | Provider target configured by the host under `config.host.connectors`. Use this for self-hosted services. |

`public_api` connectors must declare `fixedBaseURL`:

```json
{
  "name": "api",
  "type": "public_api",
  "fixedBaseURL": "https://api.example.com",
  "allowedPathPrefixes": ["/v1"],
  "auth": ["oauth_access_token"]
}
```

`configured` connectors must declare `configKey`; the host supplies the concrete
base URL and trust settings:

```json
{
  "name": "media",
  "type": "configured",
  "configKey": "immich",
  "allowedPathPrefixes": ["/api"],
  "auth": ["api_key"],
  "supportsMediaAuth": true,
  "supportsStorageRedirects": true,
  "supportsCustomTLS": true
}
```

Connector fields:

| Field | Meaning |
| --- | --- |
| `name` | Connector identifier used by `HostRequestSpec.target.connector` and `MediaRef.connector`. |
| `type` | `public_api` or `configured`. |
| `fixedBaseURL` | Fixed URL for public APIs. Must not include credentials, query, or fragment. |
| `configKey` | Host config key for configured connectors. |
| `allowedPathPrefixes` | Relative provider paths the plugin may request. Defaults to `/` when empty. |
| `auth` | Auth contexts allowed for this connector. |
| `supportsMediaAuth` | Allows connector media downloads to reference an auth context. |
| `supportsStorageRedirects` | Allows connector media downloads to redirect to configured storage origins. |
| `supportsCustomTLS` | Allows the host to attach a custom CA bundle to this connector. |

The host validates scheme, host, effective port, base path, path prefixes,
redirect targets, TLS policy, and IP policy. `allowPrivate`, custom CA bundles,
and storage origins are host-owned settings; plugin output can never enable
private-network access.

## Capabilities

Implemented sync/send capabilities:

| Capability | Export Example | Purpose |
| --- | --- | --- |
| `list_routes.v1` | `list_routes_v1` | List planned route IDs |
| `get_route_detail.v1` | `get_route_detail_v1` | Return one planned route import |
| `list_activities.v1` | `list_activities_v1` | List completed activity IDs |
| `get_activity_detail.v1` | `get_activity_detail_v1` | Return one completed activity import |
| `prepare_trail_send.v1` | `prepare_trail_send_v1` | Prepare sending a trail |

Import sync is a two-step protocol. A plugin that declares `list_routes.v1`
must also declare `get_route_detail.v1`; a plugin that declares
`list_activities.v1` must also declare `get_activity_detail.v1`. If the matching
detail capability is missing, the host skips that list capability and logs a
warning. This is a breaking change from older one-step sync plugins whose
`list_*` exports returned full trail imports.

Session-based plugins may also export an auth refresh function declared by the
manifest, for example:

```json
{
  "auth": {
    "contexts": {
      "provider_session": {
        "type": "session",
        "fields": ["email", "password"],
        "secretFields": ["password"],
        "refresh": {
          "mode": "plugin",
          "function": "refresh_session_v1"
        }
      }
    }
  }
}
```

## Sync input

`list_routes_v1` and `list_activities_v1` receive JSON input:

```json
{
  "instance": {
    "id": "abc123",
    "pluginId": "strava"
  },
  "auth": {},
  "state": {},
  "options": {
    "after": "2026-01-01"
  },
  "limits": {
    "maxItems": 50
  }
}
```

The host enforces `limits.maxItems` for each sync response. Returning more items fails the call with `sync_item_limit_exceeded`; the host does not silently truncate the response.

For sync plugins, the hard budgets are 8 for list and trail-send calls, 32 for
detail calls, and 4 for session refresh. Each call resets its budget even when
the worker session is reused. Syncs check the session before list, detail and
category-backfill calls and recycle aged workers as described below.

`auth` contains only values the host is allowed to pass to the plugin. For
OAuth plugins, refresh tokens and client secrets are not included in normal sync
capability input. Depending on the auth model, `auth` may contain values such
as:

```json
{
  "accessToken": "short-lived-token"
}
```

or, for session-based providers:

```json
{
  "email": "user@example.com",
  "password": "encrypted-at-rest-but-decrypted-for-plugin-login"
}
```

## List output

List capabilities return lightweight summaries plus capability-local state. The
host uses `source.provider` and `source.externalId` for deduplication and calls
the matching detail capability only for new items.

```json
{
  "items": [
    {
      "source": {
        "provider": "strava",
        "externalId": "123",
        "url": "https://provider.example/routes/123"
      },
      "kind": "planned"
    }
  ],
  "state": {
    "page": 2
  },
  "hasMore": true
}
```

State returned by a plugin is first fed back into the next batch of the same
sync run. Only persistent provider cursors belong in `plugin_instances.state`.
Transient batch cursors such as `page` are not stored in the database.

## Detail input

`get_route_detail_v1` and `get_activity_detail_v1` receive the summary selected
by the host:

```json
{
  "instance": {
    "id": "abc123",
    "pluginId": "strava"
  },
  "auth": {},
  "options": {
    "after": "2026-01-01"
  },
  "summary": {
    "source": {
      "provider": "strava",
      "externalId": "123"
    },
    "kind": "planned"
  }
}
```

For a new import, a fatal worker-session failure during a detail fetch is retried
once with the same input on a new worker, under the same manifest permissions,
connector policy, auth and per-export host-request budget. These detail
capabilities fetch import data; the host does not retry after import writes
begin. Ordinary per-item errors such as `not_importable` keep their existing
skip behavior. Cancellation, switch-off or deletion prevents a retry, including
when it happens while the replacement worker is starting. A failed live-instance
lookup cannot authorize a retry.

If the retry cannot produce an item, the host keeps committed imports and
continues with later items and capabilities. After **three unsuccessful extra
detail retries in total per instance sync**, later items still receive their
baseline single attempt, but no extra retry. This bounds unsuccessful retry
overhead during systemic failures without starving healthy later pages or
capabilities in a stable provider order. Successful recoveries do not consume
this budget, so it is not a limit of three total retries. Dead sessions are
closed; replacements open only when another call needs one. There is no
persistent cursor or failed-item list. Replacement startup failure stops the
instance run. List calls retain their separate one-retry behavior. Category
backfills retain their single-attempt skip behavior, without consuming the
new-import retry budget or marking the sync incomplete.

An incomplete run normally uses the existing `error` status with
`last_error.code = "sync_incomplete"` and a safe message, not worker diagnostics
or a guessed resource-limit cause. Structured auth failures, rate limits,
explicit provider retry delays and typed list-item limits take precedence over
the generic incomplete status. An actionable main-import detail error stops
further calls on the initial attempt as well as a retry, preserving its required
action or backoff regardless of item order; ordinary expected skips are
unchanged. Wrapped causes remain available to the server, but automatic
diagnostic logging uses only bounded, allowlisted cause metadata.

Cron reports incomplete runs as errors without disabling the instance or
inventing a backoff window; real provider retry delays remain honored. The next
scheduled run starts listing again with a fresh retry budget. Committed external
references prevent duplicate imports, and failed items remain eligible. A fully
traversed but incomplete run updates `last_sync_at` without clearing its error.
The UI displays a localized, wrapping explanation for `sync_incomplete`, with or
without a previous sync timestamp. It does not treat this code as an OAuth
disconnect, so the instance can still be switched off or have its settings saved
without reconnecting.

## Detail output

Detail capabilities return the full trail import:

```json
{
  "item": {
    "source": {
      "provider": "strava",
      "externalId": "123",
      "url": "https://provider.example/routes/123"
    },
    "kind": "planned",
      "name": "Morning Ride",
      "track": {
        "format": "gpx",
        "contentBase64": "..."
      },
      "waypoints": [
        {
          "name": "Viewpoint",
          "lat": 47.3769,
          "lon": 8.5417,
          "photos": [
            {
              "filename": "viewpoint.jpg",
              "contentType": "image/jpeg",
              "source": {
                "type": "url",
                "url": "https://provider.example/photo.jpg"
              }
            }
          ]
        }
      ],
      "metadata": {
        "distance": 12345.6,
        "elevationGain": 320.5,
        "elevationLoss": 318.1,
        "duration": 4567,
        "providerCategory": "Ride"
      }
    }
  }
}
```

The host imports the trails, writes PocketBase records, applies visibility
rules, deduplicates by provider/external ID, and stores the returned state.
Trail photos are attached to the imported trail. Waypoint photos are attached to
the corresponding waypoint records. Waypoint `distance_from_start` is derived
by the host from the nearest position on the imported GPX track.

Trail photos with a position move to waypoints before the import. A photo near
one of the item's waypoints joins it; the others are grouped into new, unnamed
waypoints using the user's waypoint merge settings. The first photo stays on the
trail as its cover, and so do photos without a position, photos whose
`externalId` is already on a waypoint, and photos over the waypoint limits. An
unnamed waypoint whose photos all fail to download is not saved.

### Provider difficulty

`item.difficulty` is an optional string containing the provider's coarse
difficulty rating. The accepted values are `easy`, `moderate`, and `difficult`.
Omit the field or send an empty string when no supported rating is available;
the host then stores an unknown difficulty. For example:

```json
{ "difficulty": "moderate" }
```

This rating follows the provider's meaning and is not a universal physical or
technical scale. Komoot's `difficulty.grade` maps directly to these three
values. Swiss-hike maps its physical requirement (`tief`, `mittel`, `hoch`) to
`easy`, `moderate`, `difficult`; it does not map technical hiking grades to this
field. Plugins must leave unrecognized provider ratings unknown.

The host stores unsupported string values as an unknown difficulty, so such a
trail is imported without a rating. The field must be a JSON string; any other
type is invalid plugin output and the import fails.
`metadata.difficulty` and other raw provider metadata do not set the stored
difficulty. Existing imports are skipped by deduplication and keep their current
difficulty, including empty values and user edits.

The field is optional, so older plugins remain valid without changing their
capability version. External Go plugins need an SDK release containing
`TrailImport.Difficulty` to use the typed field. A host with support for this
field is required to persist the provider rating; older hosts ignore it.

### Media sources

Media sources have two trust models:

| Source type | Meaning |
| --- | --- |
| `url` | Public external media URL. The host fetches it with public-only SSRF protections and bounded size limits. |
| `connector` | Provider-owned media fetched through a declared connector, optional host-injected auth, connector TLS/IP policy, and connector-scoped redirects. |

Public media example:

```json
{
  "filename": "cover.jpg",
  "contentType": "image/jpeg",
  "source": {
    "type": "url",
    "url": "https://cdn.example.com/photos/cover.jpg"
  }
}
```

Connector media example:

```json
{
  "filename": "original.jpg",
  "contentType": "image/jpeg",
  "source": {
    "type": "connector",
    "mediaRef": {
      "connector": "media",
      "auth": "api_key",
      "path": "/api/assets/123/original",
      "query": [
        { "name": "size", "value": "preview" }
      ],
      "assetId": "123"
    }
  }
}
```

`mediaRef.path` is required for connector downloads. `assetId` is metadata only
for now; the host does not resolve `assetId` into a URL.

Plugins should return GPX as the canonical track. If the provider exposes
authoritative summary metrics, the plugin may additionally return them in
`metadata`:

| Metadata key | Unit | Meaning |
| --- | --- | --- |
| `distance` | meters | Provider-reported trail distance. |
| `elevationGain` | meters | Provider-reported positive elevation gain. |
| `elevationLoss` | meters | Provider-reported negative elevation loss. |
| `duration` | seconds | Provider-reported elapsed duration. |
| `providerStart` | object | Provider-reported intended start coordinate, for example `{ "lat": 47.123, "lon": 8.456 }`. |
| `providerCategory` | string | Raw provider activity/category value used by host category mapping. |

The host uses positive provider metrics when present and falls back to GPX
derived metrics otherwise. Start location comes from the GPX unless
`providerStart` is present and close enough to the imported GPX track to be
plausible. Plugins should not map `providerCategory` to local category IDs; the
host owns that mapping.

## Host config

Plugin manifests may suggest defaults for host-owned settings with
`hostConfig`. These values are stored in `installed_plugins.config.host` and can
be overridden per plugin instance with `plugin_instances.config.host` only for
the explicitly supported user-level fields listed below. Connector targets and
trust settings under `host.connectors` remain exclusively in the
administrator-controlled `installed_plugins.config.host` and are ignored if
submitted through a plugin instance. Host config is never passed to plugin
exports.

Supported host fields:

| Field | Type | Used by | Meaning |
| --- | --- | --- | --- |
| `planned` | boolean | `list_routes.v1` | Enables planned route sync for the instance. |
| `completed` | boolean | `list_activities.v1` | Enables completed activity sync for the instance. |
| `privacy` | string | Trail import | `original` keeps provider visibility; `settings` uses the local user trail privacy setting. |
| `merge.enabled` | boolean | Trail import | Runs auto-merge after creating imported trails. |
| `createSummitLogForCompleted` | boolean | Trail import | Creates summit logs for completed imported trails. Defaults to `true`. |
| `categoryMapping` | object | Trail import | Maps plugin-provided `metadata.providerCategory` values to local category or subcategory targets. |
| `connectors` | object | Host request/media policy | Administrator-owned concrete settings for configured connectors; never overridable per plugin instance. |

The settings UI lets users edit `categoryMapping` per plugin instance for trail
import plugins. A mapping value can be a string for broad category-only
compatibility, or an object with `category` and optional `subcategory`. Category
and subcategory values may be local record IDs or canonical names. Unknown or
empty provider categories still fall back to the host's activity-type mapping.

Example:

```json
{
  "hostConfig": {
    "categoryMapping": {
      "Ride": {
        "category": "Biking",
        "subcategory": "Road"
      },
      "GravelRide": {
        "category": "Biking",
        "subcategory": "Gravel"
      },
      "Hike": "Hiking"
    }
  },
  "metadata": {
    "providerCategories": {
      "Ride": {
        "labels": {
          "de": "Radfahren",
          "en": "Ride"
        }
      },
      "Hike": {
        "labels": {
          "de": "Wandern",
          "en": "Hike"
        }
      }
    }
  }
}
```

`metadata.providerCategories` is display-only metadata for provider-owned
category values. The `categoryMapping` keys still use the raw values emitted as
`metadata.providerCategory`.

Configured connector host config shape:

```json
{
  "hostConfig": {
    "connectors": {
      "immich": {
        "baseURL": "https://photos.example.com",
        "basePath": "/immich",
        "allowPrivate": false,
        "tls": {
          "mode": "system"
        },
        "storageOrigins": {
          "object-storage": {
            "baseURL": "https://storage.example.com",
            "basePath": "/assets",
            "allowPrivate": false,
            "tls": {
              "mode": "system"
            }
          }
        }
      }
    }
  }
}
```

`tls.mode` supports `system` and `customCA`. Custom CA bundles are trusted only
when the manifest connector declares `supportsCustomTLS`; certificate
verification is not disabled.

The host defines the semantics of these fields. Plugins only provide defaults
or hints; custom plugin settings belong in `configSchema` and are passed to the
plugin under `options`.

Plugin errors should use the structured error format:

```json
{
  "error": {
    "code": "rate_limited",
    "message": "Provider rate limit exceeded",
    "retryAfterSeconds": 3600
  }
}
```

Supported status-relevant error codes include:

```text
auth_failed
invalid_grant
unauthorized
rate_limited
provider_unavailable
temporary_unavailable
```

## Host requests

Plugins cannot perform arbitrary provider I/O. They ask the host to execute
provider requests through the WASM host function `wanderer.http_request`.
Absolute provider URLs are not part of the request ABI.

The request shape is `HostRequestSpec`:

```json
{
  "method": "GET",
  "target": {
    "type": "connector",
    "connector": "api",
    "path": "/routes",
    "query": [
      { "name": "page", "value": "1" }
    ]
  },
  "auth": "oauth_access_token",
  "headers": {
    "accept": "application/json"
  },
  "expect": {
    "contentTypes": ["application/json"],
    "maxBytes": 1048576
  }
}
```

The host validates:

- connector identity, scheme, host, effective port, base path, and path scope
- auth context reference and connector-specific auth allowance
- manifest network permissions
- response content type
- response size
- redirect target scope

The shared Go SDK wraps this host function:

```go
response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
    Method: "GET",
    Target: sdk.RequestTarget{
        Type:      "connector",
        Connector: "api",
        Path:      "/routes",
        Query:     []sdk.QueryParam{{Name: "page", Value: "1"}},
    },
    Expect: sdk.ResponseExpect{
        ContentTypes: []string{"application/json"},
        MaxBytes:     1048576,
    },
})
```

Auth referenced by `HostRequestSpec.auth` is injected by the host. OAuth,
bearer, and API-key contexts are supported for plugin-initiated host requests.
Session auth requires handler-managed injection; if a plugin calls
`wanderer.http_request` with a session auth context, the host rejects the
request instead of silently sending it unauthenticated.

## Sending trails

`prepare_trail_send_v1` receives the trail GPX from wanderer and returns a
send plan. The plugin prepares the provider-specific request; the host
executes it.

Input:

```json
{
  "instance": {
    "id": "abc123",
    "pluginId": "hammerhead"
  },
  "auth": {},
  "config": {},
  "name": "Lunch Loop",
  "trail": {
    "format": "gpx",
    "contentBase64": "..."
  }
}
```

`config` contains the saved plugin instance configuration, for example sync
modes, an `after` date, or provider-specific options. `auth` follows the same
rules as sync input.

Output:

```json
{
  "request": {
    "method": "POST",
    "target": {
      "type": "connector",
      "connector": "api",
      "path": "/routes"
    },
    "auth": "provider_session",
    "body": {
      "type": "multipart",
      "parts": [
        {
          "name": "file",
          "source": "trail"
        }
      ]
    },
    "expect": {
      "contentTypes": ["application/json"],
      "maxBytes": 1048576
    }
  }
}
```

Supported multipart trail sources:

```text
trail
trail.gpx
```

## Auth

Auth contexts are declared in the manifest and referenced by name from
`HostRequestSpec.auth`.

### OAuth2

OAuth is declarative. The host runs authorization, token exchange, token
storage, and refresh:

```json
{
  "auth": {
    "contexts": {
      "oauth_access_token": {
        "type": "oauth2",
        "fields": ["clientId", "clientSecret"],
        "secretFields": ["clientSecret", "accessToken", "refreshToken"],
        "authorizationUrl": "https://provider.example/oauth/authorize",
        "tokenUrl": "https://provider.example/oauth/token",
        "scopes": ["activity:read_all"],
        "scopeSeparator": ",",
        "tokenRequestFormat": "json",
        "tokenAuth": "client_secret_post",
        "refresh": {
          "mode": "host",
          "grantType": "refresh_token"
        }
      }
    }
  }
}
```

The plugin may receive the short-lived access token in normal capability input.
It does not receive refresh tokens or client secrets during normal sync.
OAuth token endpoints must be covered by a fixed `public_api` connector in the
manifest. Token exchange does not use user-configured connector origins.

### Session

Session auth is for providers that require plugin-mediated login:

```json
{
  "auth": {
    "contexts": {
      "provider_session": {
        "type": "session",
        "fields": ["email", "password"],
        "secretFields": ["password"],
        "refresh": {
          "mode": "plugin",
          "function": "refresh_session_v1"
        }
      }
    }
  }
}
```

The host passes only the declared secret fields to the refresh export. The
returned session token is stored encrypted and injected by the host into future
handler-managed host-executed requests that reference the auth context, such as
`prepare_trail_send.v1` send plans. Plugin-initiated `wanderer.http_request`
calls cannot refresh session auth themselves.

### API key and bearer

API key and bearer contexts use a configured secret field:

```json
{
  "auth": {
    "contexts": {
      "api_key": {
        "type": "api_key",
        "placement": "header",
        "name": "x-api-key",
        "secretField": "apiKey"
      }
    }
  }
}
```

## Runtime isolation

WASM plugins run in separate worker processes. Exports within a sync or
trail-upload job are called sequentially and can share a worker session, but a
long sync may use several workers in turn. If a plugin calls
`wanderer.http_request`, the worker forwards the
request bytes back to the backend; the backend remains the only process that
holds connector policy, decrypted host auth, custom CA bundles, and HTTP
execution logic.

The worker boundary protects the backend from plugin crashes and hangs and
enforces request/response frame limits and timeouts. It is not an OS-level
sandbox for outbound network access; plugin-controlled provider traffic must
still go through the host request API.

Worker lifetime has three separate limits:

- `WANDERER_PLUGIN_WORKER_SESSION_TIMEOUT` (default `15m`) is both the idle
  timeout and the soft maximum worker age. Before each list, detail or
  category-backfill call, a sync renews the idle timeout or, once the worker has
  reached that age, closes it and opens a fresh worker. Renewal does not reset
  the worker's age.
- The idle timer is suspended during an export, even if the configured session
  timeout is shorter than the export timeout. A healthy return starts a new idle
  interval. A running detail call may finish after the soft age limit; recycling
  happens before the next call, not halfway through the current item.
- `WANDERER_PLUGIN_WORKER_EXPORT_TIMEOUT` (default `2m`) remains a hard per-export
  deadline. Caller cancellation also ends a running export and invalidates its
  worker. A fatal detail failure for a new import can be retried once as described
  above; category backfills are not automatically retried.

The host retains the current page cursor and supplies auth and plugin options
again after recycling, so imports and category backfills can continue without
losing their position. Plugin-side globals and cached auth do not survive a
replacement worker; each export must be able to initialize from its input.
Closing waits for the old process to exit before releasing its worker slot.
The sync then competes for admission again, waiting up to `5m` (or its caller's
earlier deadline); other callers retain the default `30s` admission timeout.
If the instance is disabled or deleted while a sync waits to reopen its worker,
the host checks again before starting the next export. This does not cancel an
already-running detail call, but the host checks the live context and instance
again before starting a new import write. Writes already in progress are not
rolled back by a later switch-off.
Recycling is cooperative, not a hard memory limit or a guarantee that a worker
exits exactly at its maximum age.

## Plugin state

User plugin configuration is stored in `plugin_instances`:

```text
plugin_instances
  user
  plugin_id
  enabled
  auth
  config
  state
  status
  last_error
  last_sync_at
  retry_not_before
```

`auth` is encrypted by PocketBase hooks. `config.plugin` stores settings passed
to the plugin, such as an `after` date. `config.host` stores host-owned settings
such as enabled capabilities, privacy handling, merge settings, and category
mapping. `state` stores per-capability provider cursors. It should only contain
values that remain valid across separate sync runs, such as provider sync tokens
or delta cursors. Batch-local cursors such as `page` are discarded before the
instance is saved.

The host also caches discovered plugin manifests in `installed_plugins`.
Installed plugins and user plugin instances are intentionally separate:
`installed_plugins.config` stores admin defaults, while
`plugin_instances.config` stores per-instance overrides. A user configuration
can exist even if the plugin bundle is not currently installed.

## Release and installation

The release workflow builds plugin archives:

```text
wanderer-plugin-hammerhead.tar.gz
wanderer-plugin-komoot.tar.gz
wanderer-plugin-strava.tar.gz
SHA256SUMS
```

Users install a plugin by extracting the archive below `data/plugins`:

```text
data/plugins/hammerhead/plugin.json
data/plugins/hammerhead/plugin.wasm
```

Docker deployments mount the runtime directory into the DB container:

```yaml
services:
  db:
    volumes:
      - ./data/plugins:/data/plugins
```
