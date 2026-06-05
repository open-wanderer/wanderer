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

Small sync helpers are included for the repeated mechanics that every provider
needs:

- `StringField` / `StringOption`
- `IntState`
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
