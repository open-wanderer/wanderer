# Wanderer Plugin SDK for Go

TinyGo-compatible helpers for Wanderer WASM plugins.

```go
import "github.com/open-wanderer/wanderer/plugins/sdk"
```

The SDK contains only plugin-side protocol types and host-function helpers. It
does not depend on Wanderer core or PocketBase.

Common protocol types:

- `ListInput`, `ListOutput`, `TrailImport`, `Track`, `Waypoint`, `Photo`
- `RefreshSessionInput`, `RefreshSessionOutput`
- `TrailSendInput`, `TrailSendPlan`
- `HostRequestSpec`, `HostResponse`, `PluginError`

Provider HTTP requests use connector targets. Plugins provide a connector name,
a relative path, and ordered query parameters; the host owns the final base URL,
path scope, redirects, TLS, and private-network policy. Public external media
URLs remain available only through `MediaSource{Type: "url"}`.

Host HTTP request bodies support JSON, `application/x-www-form-urlencoded`, and
multipart. Use `PostJSON` for JSON and `PostForm` for ordered form fields. Any
request body, including a login form POST, is governed by manifest
`permissions.uploads.maxBytes` and `permissions.uploads.contentTypes`; in this
contract "uploads" means plugin-to-provider request bodies, not only media/file
uploads.

Set `HostRequestSpec.FollowRedirects` to `sdk.Bool(false)` when a plugin needs
to inspect a redirect response itself, for example to collect `Location` and
`Set-Cookie` during a provider login flow. `HostResponse.HeaderValues` is the
only response-header representation and preserves all values. Prefer
`FirstHeader` for scalar headers and `HeaderValuesFor` for headers that can
appear more than once.

Plugins can emit host-visible logs with `LogDebug`, `LogInfo`, `LogWarn`, and
`LogError`. Log levels are strict (`debug`, `info`, `warn`, `error`) and
messages must be non-empty. Use logs for short diagnostics and timing markers;
they are best-effort and should not be part of plugin control flow.

Small sync helpers are included for the repeated mechanics that every provider
needs:

- `StringField` / `StringOption`
- `IntState`
- `IntOption`
- `KnownIDs`
- `SyncLimit`
- `NextPageState`

Additional TinyGo-compatible helper packages:

```go
import sdkgpx "github.com/open-wanderer/wanderer/plugins/sdk/gpx"
import "github.com/open-wanderer/wanderer/plugins/sdk/polyline"
```

- `gpx` writes simple GPX 1.1 track documents from provider track points.
- `polyline` decodes Google-style encoded polylines and provides small helpers
  for coordinate scale normalization, coordinate swap detection, and mapping
  shorter elevation arrays onto track points.
