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

Most plugins use:

- `sdk.HostRequest` for provider API calls through `wanderer.http_request`
- `sdk.Get` and `sdk.PostJSON` convenience helpers
- `sdk.HostRequestSpec`, `sdk.ResponseExpect`, and multipart body constants
- auth/header constants such as `sdk.AuthHeaderAuthorization`

## Manifest

Each plugin must define a static `plugin.json` manifest. The manifest is the
security and capability contract used by the host.

Minimal shape:

```json
{
  "manifestVersion": "1.0",
  "id": "example",
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
    }
  ],
  "permissions": {
    "network": {
      "staticHosts": ["api.example.com"]
    },
    "downloads": {
      "maxBytes": 1048576,
      "contentTypes": ["application/json"]
    }
  }
}
```

Important rules:

- `runtime.entrypoint` must be relative to the plugin directory.
- `id` must match the installed directory name by convention.
- `capabilities[].export` names the WASM export the runtime calls.
- `permissions.network.staticHosts` lists provider hosts the plugin may call.
- per-request limits may narrow manifest limits, but never expand them.

## Capabilities

Implemented sync/send capabilities:

| Capability | Export Example | Purpose |
| --- | --- | --- |
| `list_routes.v1` | `list_routes_v1` | Import planned routes |
| `list_activities.v1` | `list_activities_v1` | Import completed activities |
| `prepare_send_route.v1` | `prepare_send_route_v1` | Prepare an outbound route upload |

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
    "planned": true,
    "completed": true,
    "after": "2026-01-01"
  },
  "limits": {
    "maxItems": 10
  },
  "recentExternalIds": ["123", "456"]
}
```

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

## Sync output

Sync capabilities return imported trails plus capability-local state:

```json
{
  "items": [
    {
      "source": {
        "provider": "strava",
        "externalId": "123",
        "url": "https://provider.example/routes/123"
      },
      "kind": "route",
      "name": "Morning Ride",
      "track": {
        "format": "gpx",
        "contentBase64": "..."
      }
    }
  ],
  "state": {
    "page": 2
  },
  "hasMore": true
}
```

The host imports the trails, writes PocketBase records, applies visibility
rules, deduplicates by provider/external ID, and stores the returned state.

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

The request shape is `HostRequestSpec`:

```json
{
  "method": "GET",
  "url": "https://api.example.com/routes",
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

- URL scheme and host
- auth context reference
- manifest network permissions
- response content type
- response size
- redirect target host

The shared Go SDK wraps this host function:

```go
response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
    Method: "GET",
    URL:    "https://api.example.com/routes",
    Expect: sdk.ResponseExpect{
        ContentTypes: []string{"application/json"},
        MaxBytes:     1048576,
    },
})
```

## Sending routes

`prepare_send_route_v1` receives the trail GPX from wanderer and returns an
upload plan. The plugin prepares the provider-specific request; the host
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
  "route": {
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
    "url": "https://provider.example/routes",
    "auth": "provider_session",
    "body": {
      "type": "multipart",
      "parts": [
        {
          "name": "file",
          "source": "route"
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

Supported multipart route sources:

```text
route
route.gpx
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
host-executed requests that reference the auth context.

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
  next_retry_at
```

`auth` is encrypted by PocketBase hooks. `config` stores user settings such as
enabled sync modes or an `after` date. `state` stores per-capability cursors.

The host also caches discovered plugin manifests in `installed_plugins`.
Installed plugins and user plugin instances are intentionally separate: a user
configuration can exist even if the plugin bundle is not currently installed.

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

