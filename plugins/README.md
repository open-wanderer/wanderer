# wanderer plugins

This directory contains first-party WASM provider plugins.

Each plugin is a standalone Go/TinyGo module with:

- `plugin.json` as the source manifest
- `go run github.com/open-wanderer/wanderer/plugins/sdk/cmd/manifestcheck`
  for normalized dist manifest output
- ignored `dist/<plugin-id>/plugin.json` and `dist/<plugin-id>/plugin.wasm`
  build output for runtime discovery

Build the dist bundles before running from a fresh checkout:

```sh
make plugins-build
```

The runtime loads plugins from direct child directories of `data/plugins`, for
example `data/plugins/strava/plugin.json`. To build and install the bundled
plugins into that gitignored local runtime directory, run:

```sh
make plugins-install-local
```

To rebuild a single plugin, install TinyGo and run:

```sh
cd plugins/strava
make build
```

Repeat for `hammerhead` and `komoot` as needed.

Release builds create plugin bundle archives in CI. The database Docker image
does not include plugins; users install release bundles into `data/plugins`.

## Runtime flows

This section lists the main backend call paths for debugging and maintenance.
It is kept here because understanding when and how the host invokes plugin
exports (sync, host requests, send route) is useful when developing plugins.
The code these flows reference lives in the core backend under `db/` (PocketBase
handlers, sync manager, host functions), not in this `plugins/` directory.

### Plugin discovery

Used when the backend refreshes the list of plugin bundles installed on disk and
caches their manifests in PocketBase.

```text
Manager.SyncInstalledPlugins
  -> LoadLocalPlugins(data/plugins)
  -> LoadLocalPlugin(pluginDir)
  -> ValidateManifest
  -> save installed_plugins record
```

Manifest `configSchema` defines plugin-owned settings that are passed to plugin
exports. Host-owned settings are documented by the host and are not passed to
plugins. A manifest may only suggest host defaults via `hostConfig`; the current
host fields are:

| Field | Purpose |
| --- | --- |
| `planned` | Enables `list_routes.v1` sync. |
| `completed` | Enables `list_activities.v1` sync. |
| `privacy` | Chooses provider visibility or local user privacy settings. |
| `merge.enabled` | Runs auto-merge after trail import. |
| `createSummitLogForCompleted` | Creates summit logs for completed imports. |
| `categoryMapping` | Maps `metadata.providerCategory` to local category IDs or names. |
| `connectors` | Provides host-owned base URL, TLS, private-network, and storage redirect settings for configured connectors. |

Trail import plugins should keep provider-specific category values in
`metadata.providerCategory`. They may also provide provider summary metrics in
`metadata.distance`, `metadata.elevationGain`, `metadata.elevationLoss`, and
`metadata.duration`; the host uses those positive values instead of GPX-derived
summary metrics and falls back to GPX when a value is missing.

Photo descriptors may be returned either on the imported trail or on individual
waypoints. The host downloads those media files and stores them on the
corresponding PocketBase records.

### List plugins

Used by the settings UI to show locally available plugins, their metadata,
icons, capabilities, and current availability status.

```text
GET /plugins
  -> PluginSystemPluginsList
  -> Manager.ListLocalPlugins
  -> LoadInstalledPlugins
  -> pluginIcons
  -> return PluginInfo[]
```

### Save plugin instance

Used whenever a user creates or updates their personal plugin configuration.
This path is where auth values are encrypted and default status is assigned.

```text
PocketBase plugin_instances create/update
  -> CreatePluginInstanceHandler / UpdatePluginInstanceHandler
  -> ensurePluginInstanceStatus
  -> encryptPluginInstanceAuth
  -> pluginInstanceSecretFields
  -> installed_plugins or local manifest lookup
```

### OAuth start

Used when the UI starts an OAuth connection flow and needs the provider
authorization URL.

```text
POST /plugins/oauth/start
  -> PluginSystemOAuthStart
  -> localPlugin
  -> OAuthContext
  -> decryptedInstanceAuth
  -> ValidateOAuthRedirectURI
  -> NewOAuthState / PKCEChallenge
  -> save oauthState/oauthCodeVerifier/auth context
  -> return provider authorization URL
```

### OAuth callback

Used after the provider redirects back with an OAuth code. This exchanges the
code for tokens and stores them on the plugin instance.

```text
POST /plugins/oauth/callback
  -> PluginSystemOAuthCallback
  -> localPlugin
  -> decryptedInstanceAuth
  -> ExchangeOAuthToken
  -> StoreOAuthToken
  -> clear transient OAuth fields
  -> save plugin_instance as configured
```

### Cron sync

Used by the scheduled background sync. It refreshes installed plugin metadata,
selects enabled plugin instances, and skips instances that are still backing
off.

```text
PluginSystemSyncConfigured
  -> Manager.SyncInstalledPlugins
  -> LoadInstalledPlugins
  -> pluginInstances
  -> shouldSkipPluginInstance
  -> syncPluginInstance
```

### Sync one instance

Used to prepare one user/plugin instance for sync: actor lookup, runtime
selection, auth decryption, OAuth refresh, and capability dispatch.

```text
syncPluginInstance
  -> find ActivityPub actor
  -> RuntimeFor(plugin)
  -> decryptedInstanceAuth
  -> RefreshOAuthAuthIfNeeded
  -> runtime.OpenSession(plugin, policy.WithHostAuth(auth))
  -> loop syncCapabilityDescriptors
  -> pluginCapability
  -> syncPluginCapability
  -> session.Close
```

### Sync capability

Used for one concrete import capability such as `list_routes.v1` or
`list_activities.v1`. This is where plugin output becomes imported trails.

```text
syncPluginCapability
  -> session.Call(list export)
  -> plugin-worker process
  -> worker host-function RPC bridge
  -> plugin export returns TrailSummary[]
  -> host filters already imported external ids
  -> session.Call(detail export) for new summaries
  -> plugin detail export returns TrailImport
  -> importer.ImportTrail
  -> update capability state and counters
```

### Plugin host request

Used when plugin code needs to call a provider API. The host applies manifest
policy, executes the HTTP request, and returns the response to WASM.

```text
plugin sdk.HostRequest
  -> WASM import wanderer.http_request
  -> worker-side extism host function http_request
  -> worker host_http_request RPC
  -> backend executeHostHTTPRequest
  -> ExecuteHostRequest
  -> InjectHostRequestAuthFromPolicy
  -> ValidateAndResolveHostRequestSpec
  -> hostRequestBody
  -> validateHostRequestUpload
  -> connector-scoped http.Client.Do
  -> validateHostHTTPResponse
  -> worker host_http_response RPC
  -> return HostResponse to plugin
```

### Send route

Used when a user sends an existing wanderer trail to an external provider.

```text
POST /plugins/send-route
  -> PluginSystemSendRoute
  -> localPluginCapability(prepare_send_route.v1)
  -> util.TrailAccessibleByUser
  -> readTrailGPX
  -> runtime.OpenSession(plugin, policy.WithHostAuth(auth))
  -> session.Call(prepare_send_route_v1)
  -> plugin returns UploadPlan
  -> ValidateHostRequestSpec
  -> InjectHostRequestAuth
  -> ExecuteHostRequest
  -> session.Close
```
