# Wanderer Plugin SDK for Go

TinyGo-compatible helpers for Wanderer WASM plugins.

```go
import "github.com/open-wanderer/wanderer/plugins/sdk"
```

The SDK contains only plugin-side protocol types and host-function helpers. It does not depend on Wanderer core or PocketBase.

Common protocol types:

- `ListInput`, `ListOutput`, `TrailImport`, `Track`, `Waypoint`, `Photo`
- `RefreshSessionInput`, `RefreshSessionOutput`
- `TrailSendInput`, `TrailSendPlan`
- `ManeuverRequest`, `ManeuverResult`, `Maneuver`, `ManeuverGeometry`, `ManeuverLimits`
- `HostRequestSpec`, `HostResponse`, `PluginError`

`ManeuverRequest` is the provider-neutral `maneuvers.v1` boundary. It contains normalized track parts, resolved routing profile inputs, a language hint, and host-owned output limits. Persisted trail IDs, share tokens, and raw GPX are intentionally absent. The SDK owns `testdata/maneuvers_v1.json`; the backend owns an identical fixture in `db/routes/testdata/`. Each module can therefore test full and sparse payloads plus unordered field/JSON-tag sets independently, while a full repository checkout additionally compares both fixture files byte for byte. The fixture's separate `pluginError` key supplies the capability-independent error field contract to dedicated tests without making it part of the maneuver-named test.

Provider HTTP requests use connector targets. Plugins provide a connector name, a relative path, and ordered query parameters; the host owns the final base URL, path scope, redirects, TLS, and private-network policy. Returned media can use either a public external URL with `MediaSource{Type: "url"}` or provider-owned connector media with `MediaSource{Type: "connector", MediaRef: ...}`. Connector media is fetched through the same manifest policy, optional host-injected auth, TLS/IP checks, and redirect limits as host HTTP requests.

Host HTTP request bodies support JSON, `application/x-www-form-urlencoded`, and multipart. Use `PostJSON` for JSON and `PostForm` for ordered form fields. Any request body, including a login form POST, is governed by manifest `permissions.uploads.maxBytes` and `permissions.uploads.contentTypes`; in this contract "uploads" means plugin-to-provider request bodies, not only media/file uploads.

Set `HostRequestSpec.FollowRedirects` to `sdk.Bool(false)` when a plugin needs to inspect a redirect response itself, for example to collect `Location` and `Set-Cookie` during a provider login flow. `HostResponse.HeaderValues` is the only response-header representation and preserves all values. Prefer `FirstHeader` for scalar headers and `HeaderValuesFor` for headers that can appear more than once.

Plugins can emit host-visible logs with `LogDebug`, `LogInfo`, `LogWarn`, and `LogError`. Log levels are strict (`debug`, `info`, `warn`, `error`) and messages must be non-empty. Use logs for short diagnostics and timing markers; they are best-effort and should not be part of plugin control flow.

TinyGo capability exports can return a structured Extism failure with `sdk.Fail(code, message)`. `sdk.Fail` is available only when the `tinygo` build tag is active because it calls the Extism PDK; its platform-neutral JSON encoder is covered by ordinary SDK tests, and plugin WASI builds compile the actual export helper. `PluginError` carries `code`, an optional `message`, and an optional `retryAfterSeconds` hint. Protocol outputs may populate that hint; the first-party BRouter adapter does so for exactly one positive delta-seconds `Retry-After` value on HTTP 429 or 503.

Small sync helpers are included for the repeated mechanics that every provider needs:

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
- `polyline` decodes Google-style encoded polylines and provides small helpers for coordinate scale normalization, coordinate swap detection, and mapping shorter elevation arrays onto track points.
