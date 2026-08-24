# Routing plugins: technical reference

This document describes the current routing plugin stack in wanderer from the
trail editor to an external routing provider. It covers the generic contract
for plugins of type `routing`, the first-party Valhalla and BRouter
implementations, category-to-profile resolution, custom profiles, standard and
provider-native controls, route and elevation calculation, persistence, error
handling, and the files and functions involved in every workflow.

“First-party” describes ownership and host trust policy, not packaging. Official
Wanderer images contain no provider plugin runtime bundles; Valhalla and BRouter
are installed separately from release assets or built from source.

The general plugin runtime is documented in
[`plugin-system.md`](../src/content/docs/develop/plugin-system.md). Architectural
architectural decisions and routing phase history live in
[`openspec/design/routing-plugin.md`](../../openspec/design/routing-plugin.md);
this reference documents the code's current operational behavior.

## 1. Terminology and scope

| Term | Meaning |
| --- | --- |
| Routing plugin | A WASM plugin with manifest type `routing` and at least one routing capability. |
| Route engine | One enabled plugin instance that can serve `route.v1`. |
| Elevation engine | One enabled plugin instance that can serve `elevation.v1`. It may differ from the route engine. |
| Maneuver engine | An enabled plugin instance that can serve `maneuvers.v1`. The configured engine is preferred independently from route and elevation engines; the host falls back to another eligible instance when necessary. |
| Native profile | A provider-owned routing dialect, such as Valhalla `bicycle` or BRouter `trekking`. |
| Custom profile | A host-owned user or admin record in `routing_profiles`, currently most visibly a BRouter `.brf` file. |
| Mapping | The resolved association from a canonical Wanderer category and optional subcategory to one native or custom profile. |
| Mode | The generic internal class `foot`, `bike`, `motor`, `mixed`, or unresolved `other`. The selected profile supplies it. |
| Standard preference | A provider-neutral option such as `hillPreference` that a plugin translates into its native dialect. |
| Native control | A provider/profile-specific option such as Valhalla `bicycle_type` or BRouter `SAC_scale_limit`. |
| Candidate | A normalized route returned by a plugin, including geometry, segments, metrics, and provider provenance. |
| Anchor | An ordered route point selected or moved in the editor. |
| Round trip | A generated closed route whose unique start and synthetic anchors are materialized into normal cyclic editor segments. |

Normal route drawing is category-driven. The user selects a Wanderer category
or subcategory; the host resolves profile and mode. Profile keys, mode, provider
configuration, and provider protocols are implementation details outside the
normal trail-editor workflow.

## 2. Architecture

The route call path is:

```text
trail edit page / route editor
  -> routing_store.calculateRouteBetween
  -> SvelteKit proxy POST /api/v1/routing/route
  -> PocketBase POST /plugins/routing/route
  -> enabled plugin instance and capability resolution
  -> category/subcategory mapping resolution
  -> profile, mode, preferences, and native config resolution
  -> dedicated plugin-worker process
  -> WASM export route_v1
  -> wanderer.http_request through a declared connector
  -> external routing provider
  -> plugin-normalized candidate
  -> host validation and provenance normalization
  -> editor TrackSegment and GPX mutation
```

Elevation is independent:

```text
route candidate has complete point-aligned elevation
  -> editor uses it directly

route candidate has no usable elevation
  -> POST /api/v1/routing/elevation
  -> PocketBase /plugins/routing/elevation
  -> selected elevation plugin export elevation_v1
  -> heights are attached to decoded route points
```

Custom profile management follows a separate path:

```text
plugin settings upload/edit
  -> host-owned routing_profiles record
  -> optional profile_introspect_v1 call
  -> plugin detects mode, supported standard preferences, native controls
  -> host persists generic metadata
  -> mapping selects the profile for a category/subcategory
  -> route call passes the bounded profile data to that plugin only
```

Providers that declare `profile_prepare.v1` can move expensive profile work
out of the segment fan-out:

```text
route drawing starts or profile-relevant controls change
  -> debounced POST /api/v1/routing/profile-prepare without anchors
  -> host resolves the same profile, mode, preferences, and native config
  -> profile_prepare_v1 uploads or compiles the effective provider profile
  -> host caches only the opaque prepared key

later route request
  -> host reuses the cached key in profile.preparedKey
  -> route_v1 skips the repeated provider-side profile upload
```

Round-trip generation uses a separate capability and then rejoins the normal
editor model:

```text
first editor anchor + target distance
  -> POST /api/v1/routing/round-trip
  -> enabled engine with supportsRoundTrip and round_trip.v1
  -> provider returns one closed geometry and optional suggested anchors
  -> host validates closure and materializes bounded synthetic anchors
  -> normal GPX track with cyclic routed segments and persisted provenance
```

Plugins do not open arbitrary network connections and do not persist files.
Every provider request goes through the host HTTP function and its connector
policy. Uploaded profile content remains in PocketBase and is supplied only to
a concrete plugin invocation.

## 3. Primary files

### 3.1 Generic host and persistence

| Area | File | Responsibility |
| --- | --- | --- |
| HTTP registration | `db/main.go` | Registers all `/plugins/routing/...` endpoints. |
| Route/elevation contract | `db/routes/plugin_system_routing.go` | Request and response DTOs, check/elevation handlers, typed plugin calls, validation, normalization, base limits, and structured errors. |
| Route variants | `db/routes/plugin_system_routing_variants.go` | Route and candidate-set orchestration, engine selection, routing modes, fan-out budgets, parallel execution, composition, elevation-assisted ranking, diversity curation, and provenance. |
| Round trips | `db/routes/plugin_system_routing_round_trip.go` | Dedicated host endpoint, bounds, capability invocation, closure validation, synthetic anchors, cyclic segments, and round-trip provenance. |
| Profile preparation | `db/routes/plugin_system_routing_prepare.go` | Warm-up endpoint, absolute-TTL cache, concurrent singleflight, preparation fallback, invalidation, and coordinated prepared-key refresh. |
| Settings and mappings | `db/routes/plugin_system_routing_config.go` | Settings precedence, engine discovery, category mappings, profiles, control discovery, profile introspection, authorization, and persistence. |
| Plugin type migration | `db/migrations/1781000001_installed_plugins_routing_type.go` | Adds and removes only the `routing` select value. It preserves values owned by separately deployed plugin migration bundles; rollback removes only routing plugin records and instances. |
| Collections | `db/migrations/1781000002_routing_settings_profiles_mappings.go` | Creates `routing_settings`, `routing_profiles`, and `routing_profile_mappings`, including the base64 capacity required for a 256 KiB decoded profile. |
| Local plugin discovery | `db/pluginsystem/manifest.go`, `db/pluginsystem/installed.go`, `db/pluginsystem/manager.go` | Validates bundles, caches manifests, and resolves installed plugins. |
| Runtime | `db/pluginsystem/runtime.go`, `db/pluginsystem/worker.go`, `db/pluginsystem/worker_process.go` | Runs WASM in a dedicated worker process and enforces per-call request budgets. |
| Host networking | `db/pluginsystem/host_http.go`, `db/pluginsystem/policy.go`, `db/services/pluginhost/config.go` | Resolves connectors, effective configuration, authentication, SSRF policy, response types, and size limits. |

### 3.2 First-party plugins

| Area | File | Responsibility |
| --- | --- | --- |
| Valhalla manifest | `plugins/valhalla/plugin.json` | `route.v1`, `elevation.v1`, profiles, defaults, controls, connector, and feature discovery. |
| Valhalla entry points | `plugins/valhalla/main.go` | TinyGo exports `route_v1` and `elevation_v1`. |
| Valhalla provider logic | `plugins/valhalla/valhalla.go` | Costing translation, `/route`, `/height`, geometry, segments, and snapped anchors. |
| BRouter manifest | `plugins/brouter/plugin.json` | `route.v1`, `round_trip.v1`, `profile_introspect.v1`, `profile_prepare.v1`, curated profiles, upload metadata, controls, alternatives, and connector policy. |
| BRouter entry points | `plugins/brouter/main.go` | TinyGo exports `route_v1`, `round_trip_v1`, `profile_introspect_v1`, and `profile_prepare_v1`. |
| BRouter provider logic | `plugins/brouter/brouter.go` | Profile selection, validation, rendering, preparation/upload, introspection, provider requests, and errors. |
| BRouter shared route logic | `plugins/brouter/core/core.go` | Candidate construction, encoded polyline, elevation, segment splitting, and snapped anchors. |
| BRouter preference translation | `plugins/brouter/core/preferences.go` | Converts standard preferences into bounded BRouter parameters. |
| BRouter profile capabilities | `plugins/brouter/core/profile.go` | Detects mode and derives supported standard preferences from exposed parameters. |
| BRouter bases | `plugins/brouter/profiles/*.brf` | Embedded upstream profiles every curated and generated profile is rendered from. |

### 3.3 Frontend

| Area | File | Responsibility |
| --- | --- | --- |
| Frontend types | `web/src/lib/models/routing.ts` | Browser DTOs for settings, engines, mappings, profiles, controls, candidates, and route options. |
| Routing store/API client | `web/src/lib/stores/routing_store.svelte.ts` | API calls, route calculation, elevation fallback, GPX segment mutations, and undo/redo. |
| Trail integration | `web/src/routes/trail/edit/[id]/+page.svelte` | Category selection, engine initialization, effective controls, anchors, drawing, snapping, and recalculation. |
| Route controls | `web/src/lib/components/trail/route_editor.svelte` | Shows only effective standard controls and edits the internal route options. |
| Global routing settings | `web/src/routes/settings/routing/+page.svelte` | Enables or disables routing and selects the engines and route-editor startup behavior while routing is active. |
| Plugin settings shell | `web/src/lib/components/settings/plugins/plugin_instance_settings_modal.svelte` | Embeds routing-specific settings into a plugin instance modal. |
| Profile mapping UI | `web/src/lib/components/settings/plugins/plugin_routing_settings.svelte` | Category tree, mappings, advanced controls, custom-profile list, upload, download, toggle, and delete. |
| Profile editor | `web/src/lib/components/settings/plugins/routing_profile_editor_modal.svelte` | Name/content editing, syntax highlighting, download, automatic re-introspection, and unresolved-mode fallback. |
| Code editor | `web/src/lib/components/base/code_editor.svelte`, `web/src/lib/util/code_editor_language.ts` | Generic editor and plugin-declared language highlighting. |
| Profile utilities | `web/src/lib/util/routing_profile_util.ts`, `web/src/lib/util/file_util.ts` | UTF-8/base64 conversion, safe download filenames, and browser download. |
| Variant budget utility | `web/src/lib/util/routing_variant_util.ts` | Mirrors server fan-out estimation for anchors, engines, native alternatives, and profile preparation support. |
| Round-trip discovery utility | `web/src/lib/util/routing_round_trip_util.ts` | Gates loop generation and selects an enabled capable engine. |
| SvelteKit proxies | `web/src/routes/api/v1/routing/**/+server.ts` | Proxies authenticated browser calls to PocketBase without exposing backend details. |

## 4. Manifest and capability contract

A routing plugin has `type=routing`, a WASM entrypoint, and one or more versioned
capabilities:

```json
{
  "manifestVersion": "1.0",
  "id": "example-router",
  "type": "routing",
  "runtime": {
    "type": "wasm",
    "entrypoint": "plugin.wasm"
  },
  "capabilities": [
    { "name": "route", "version": "v1", "export": "route_v1" },
    { "name": "round_trip", "version": "v1", "export": "round_trip_v1" },
    { "name": "elevation", "version": "v1", "export": "elevation_v1" },
    {
      "name": "profile_introspect",
      "version": "v1",
      "export": "profile_introspect_v1"
    },
    {
      "name": "profile_prepare",
      "version": "v1",
      "export": "profile_prepare_v1"
    }
  ]
}
```

| Capability | Required input | Output | Current providers |
| --- | --- | --- | --- |
| `route.v1` | Resolved profile, mode, anchors, preferences, native config, and options | Normalized route candidates or structured plugin error | Valhalla, BRouter |
| `round_trip.v1` | Start, target distance, optional direction/seed, and resolved profile inputs | Closed geometry, summary, optional suggested anchors, and calibration metadata | BRouter |
| `elevation.v1` | Encoded polyline or coordinates | Heights and optional status or structured error | Valhalla |
| `profile_introspect.v1` | One host-owned profile | Detected mode, supported preferences, native control groups, metadata, or structured error | BRouter |
| `profile_prepare.v1` | Resolved profile, mode, preferences, and native config without anchors | Opaque provider preparation key or structured error | BRouter |

`profile_introspect.v1` is optional. Without it, a custom profile has no
provider-assisted mode or capability detection.

`profile_prepare.v1` is optional. The host caches its opaque result by plugin
instance and effective-profile fingerprint, never exposes it to the browser,
and injects it into subsequent `route.v1` calls. The route editor warms the
primary engine in the background when ordinary drawing starts or
profile-relevant controls change. Actual variant requests prepare all selected
variant engines. Cache entries expire absolutely after 15 minutes and are
invalidated immediately when a provider rejects the prepared key with
`unsupported_profile`. Concurrent route calls for the same engine wait for one
shared refresh and then all retry with its result. Initial preparation failure
falls back to normal routing without the key when the resulting work remains
within the fan-out budget; otherwise the response uses
`profile_preparation_fanout_limit_exceeded` and includes the preparation
errors. If an in-flight prepared key is rejected and its refresh also fails,
all waiting calls retry once without a key. The provider-side inline uploads in
this rare recovery path are intentionally not part of the preflight estimate;
the amplification remains bounded to one retry per outstanding route call.
`profilePreparationSupported` in effective controls lets the editor count one
profile upload per engine only when this capability is available; otherwise
the fan-out budget retains one upload per segment. Plugins without the
capability continue to route normally.

### 4.1 Routing discovery metadata

`metadata.routing` is read by the host and frontend. Important fields are:

| Field | Meaning |
| --- | --- |
| `version` | Routing metadata contract version. |
| `roles` | Informational roles such as `route` and `elevation`; executable `*.v1` capabilities define the canonical roles exposed by the host. |
| `modes` | Generic modes provided by the plugin. |
| `nativeProfiles` | Built-in profile keys, labels, modes, base config, optional standard-preference display defaults, and optional control groups. |
| `categoryMappings` | Manifest defaults from canonical category/subcategory names to native profiles. |
| `standardPreferences` | Explicit allowlist of standard preferences understood by the plugin, optionally restricted by mode. An absent or empty list accepts no canonical preferences. |
| `providerPreferences` | Explicit allowlist of provider-specific scalar preference keys. Undeclared keys are removed before invocation. |
| `nativeControlDiscovery` | Whether profile-scoped advanced controls can be requested. |
| `nativeProfileUpload` | Upload enablement, file types, UI byte limit, and optional editor language. |
| `supportsViaRouting` | Discovery signal for ordered multi-anchor routes. |
| `supportsAlternatives`, `maxAlternatives` | Provider-native candidate discovery and its per-call bound. |
| `supportsRouteElevation` | Whether route candidates may include elevation. |
| `supportsElevation` | Whether the independent elevation capability is available. |

Manifest native profiles and category mappings are virtual built-ins. They are
read from discovery metadata and are not copied into `routing_profiles` or
`routing_profile_mappings`.

### 4.2 Network and configuration boundary

Provider endpoints are declared as connectors in `permissions.network`. For
example, Valhalla permits `/route` and `/height`, while BRouter permits
`/brouter` and `/brouter/profile`.

The effective configuration is:

```text
installed_plugins.config
  + permitted plugin_instances.config overrides
  -> config.plugin passed to WASM
  -> config.host used by host policy and connector resolution
```

`pluginhost.EffectiveConfig` and `pluginhost.InstancePolicy` construct this
boundary. Connector targets, private-network access, TLS policy, path prefixes,
content types, redirect trust, and download/upload limits remain host-owned.

An administrator may authorize a provider on the Wanderer host only with a
fixed literal loopback connector target such as `http://127.0.0.1:17777` (or
`localhost`) plus `allowPrivate=true`. The dialer evaluates loopback permission
again for every resolved dial host, so a public hostname cannot inherit that
trust by resolving or rebinding to loopback. DNS resolution and all sequential
address attempts share one 30-second dial deadline.

A route request uses at most eight reusable worker-session lanes in total
across all selected plugin instances. The host distributes those lanes in
stable, primary-first rounds and maps an engine's segments to its lanes by
segment index. Lane zero is also the engine's profile-preparation session, so
preparation does not consume a ninth worker slot. Calls on one lane remain
serialized by the worker protocol, while different lanes and engines run in
parallel and amortize process startup across segments. Route sessions close
before the bounded, parallel elevation phase, which uses up to four reusable
lanes and cannot be starved by idle route workers.

## 5. Host contracts

### 5.1 Route input

The browser sends a provider-neutral request such as:

```json
{
  "pluginId": "brouter",
  "engines": [
    { "pluginId": "brouter", "instanceId": "instance-id" },
    { "pluginId": "valhalla" }
  ],
  "engineMode": "parallel",
  "desiredVariants": 3,
  "requestVariants": true,
  "referenceGeometry": {
    "format": "encoded_polyline",
    "precision": 6,
    "coordinates": "..."
  },
  "routingMode": "segment",
  "anchors": [
    { "lat": 47.0, "lon": 8.0 },
    { "lat": 47.1, "lon": 8.1 }
  ],
  "category": "Biking",
  "subcategory": "Gravel",
  "profile": {
    "pluginId": "brouter",
    "key": "",
    "nativeConfig": {}
  },
  "preferences": {
    "hillPreference": 0.55,
    "avoidBadSurfaces": 0.7
  },
  "options": {
    "alternatives": 1,
    "includeElevation": false
  }
}
```

`requestVariants=false` is the normal drawing path. It forces a single primary
engine and one final candidate even if additional engines are enabled or a larger default
variant count are configured. An explicit variant request sets
`requestVariants=true`, may select up to four `engines`, chooses `single` or
`parallel` engine mode, and asks for one to three final alternatives through
`desiredVariants`. The editor also sends its current route as optional
`referenceGeometry`; host curation removes candidates that are too similar to
that route as well as duplicates within the newly calculated set. Because
provider candidate counts include their primary route, the host requests one
additional provider candidate when a reference is present, up to the provider's
declared `maxAlternatives` bound. Thus
`desiredVariants` consistently counts alternatives to the current route, not
the provider primary. Providers may still return fewer distinct candidates;
the response then emits `routing_variants_fewer_than_requested`.
`route-candidates` always uses the broad candidate-set path
and applies its separate exposure and rate limits.

An empty profile key is the normal category-driven form. Before invoking WASM,
`applyRoutingCategoryMapping` fills in:

- native or materialized profile identity;
- profile kind and bounded content where applicable;
- generic mode from the selected profile;
- merged standard preferences;
- merged provider-native config.

The client may still carry an internal mode value because existing editor
option buckets are named `pedestrian`, `bicycle`, and `auto`. The mapping's
profile mode is authoritative and replaces it before plugin invocation.

`profile.preparedKey` is host-owned. Any value submitted by an HTTP client is
discarded before resolution. Only `profile_prepare.v1` and the in-memory host
cache can supply it to `route.v1`.

### 5.2 Route output

Every plugin returns zero or more candidates in this shape:

```json
{
  "candidates": [
    {
      "id": "plugin-local-id",
      "profileKey": "gravel",
      "geometry": {
        "format": "encoded_polyline",
        "precision": 6,
        "coordinates": "..."
      },
      "elevation": {
        "heights": [500.0, 501.2],
        "status": "included",
        "source": "brouter"
      },
      "summary": {
        "distance": 1234.5,
        "duration": 420,
        "elevationGain": 85
      },
      "segments": [
        {
          "fromAnchor": 0,
          "toAnchor": 1,
          "geometry": {
            "format": "encoded_polyline",
            "precision": 6,
            "coordinates": "..."
          },
          "distance": 1234.5,
          "duration": 420
        }
      ],
      "snappedAnchors": [
        { "lat": 47.0, "lon": 8.0 },
        { "lat": 47.1, "lon": 8.1 }
      ],
      "warnings": []
    }
  ],
  "engineErrors": []
}
```

Plugin candidate IDs are not trusted as public identity. The host replaces them
with a stable `routing:<mode>:<composition>:<provenance-and-geometry-hash>` identity and
adds plugin, instance, provider, profile, and per-segment provenance.

The host requires:

- encoded polyline format with precision 6;
- one segment for each adjacent anchor pair;
- segment indices `0->1`, `1->2`, and so on;
- valid WGS84 coordinates and finite non-negative metrics;
- point-aligned elevation when heights are present;
- snapped anchors either absent or equal in count to request anchors.

Invalid candidates are converted into engine errors. If no valid candidate
remains, the entire HTTP call fails with a structured error.

### 5.3 Round-trip input and materialization

`POST /api/v1/routing/round-trip` accepts one start coordinate, a target from
1,000 through 300,000 metres, category/profile inputs, and optional direction
and seed. The engine must declare both the `supportsRoundTrip` discovery flag
and `round_trip.v1`; discovery suppresses the flag when the capability is
missing.

BRouter sends `engineMode=4`, `roundTripPoints=5`, and requests provider
waypoints. Its `roundTripDistance` is a construction radius, so the adapter
starts at `targetDistance / (2π)`, makes at most three calls, and corrects the
radius proportionally using the measured track length. An explicit direction
becomes an integer bearing, a seed hashes deterministically to a bearing, and
an omitted direction/seed omits the provider parameter and uses BRouter's
random direction. The best valid loop
is returned; if it is outside the current internal ±10% tolerance, it carries
`round_trip_target_tolerance_not_met`. The tolerance is intentionally internal
and is expected to change after practical trials.
If a provider returns more than one round-trip candidate, the host retains the
first and adds `round_trip_candidates_truncated`; the dedicated contract is
intentionally single-candidate.

The host does not persist the provider result as a special route type. It
requires the provider start within 100 metres of the requested start and the
provider geometry to close within 10 metres, leaves that geometry untouched,
snaps the first editor anchor to it, accepts at
most eight on-route suggestions, or derives four distance-distributed anchors.
The start occurs once. With `N` unique anchors the candidate has `N` segments and the last
segment closes to `toAnchor=0`. Segment provenance includes normal routing
identity plus round-trip source, request id, target/actual distance, direction,
seed, synthetic endpoint flags, and an independent
`routeTopology=closed_loop` marker. GPX persistence reloads loop state only
from that topology marker—not by guessing from closed geometry. Normal routed
edits retain their actual `segment`/`via` engine provenance; manual straight
segments and reversed routes retain only topology metadata and never claim an
engine or profile that did not calculate them. Subsequent edits use the normal
route endpoint with the start appended only in the transient request.
If persisted topology becomes incomplete, the editor warns once, disables loop
state, and immediately recalculates the same anchors as an open route so anchor
and geometry state cannot diverge.

If generation falls back from the selected engine to another capable engine,
the request drops the original engine's native profile key, native config, and
preference values;
the host resolves the target engine's category mapping instead. The editor
shows the engine that will be used. Replacing an existing route requires an
explicit confirmation and remains undoable.

### 5.4 Elevation input and output

The elevation endpoint accepts one encoded polyline or an explicit coordinate
array:

```json
{
  "pluginId": "valhalla",
  "encodedPolyline": "..."
}
```

The normalized result is:

```json
{
  "heights": [500.0, 501.2],
  "status": "included"
}
```

`normalizeRoutingElevationOutput` converts provider sentinel values, reports
`empty` for no samples, `partial` when only some points have valid height, and
`included` otherwise. Candidate and standalone elevation therefore use the
same success vocabulary.

### 5.5 Profile introspection output

An introspection-capable plugin may return:

```json
{
  "mode": "foot",
  "supportedPreferences": [
    "hillPreference",
    "maxHikingDifficulty"
  ],
  "nativeControlGroups": [
    {
      "key": "profile",
      "label": "Profile",
      "controls": []
    }
  ],
  "metadata": {}
}
```

The host owns the generic interpretation:

- a returned `foot`, `bike`, `motor`, or `mixed` becomes the profile mode and
  `metadata.modeDetection=automatic`;
- no detected mode becomes `mode=other` and
  `metadata.modeDetection=unresolved`, unless the request deliberately carries
  a valid manual mode and `modeDetection=manual`;
- `supportedPreferences` is persisted in profile metadata;
- an unsupported returned mode is an `invalid_plugin_response`.

### 5.6 Profile preparation input and output

The optional plugin capability receives the effective routing inputs that can
change the provider profile, but no route anchors or route options:

```json
{
  "mode": "bike",
  "profile": {
    "pluginId": "brouter",
    "key": "custom-gravel",
    "kind": "custom_file",
    "contentBase64": "...",
    "nativeConfig": {}
  },
  "preferences": {
    "hillPreference": 0.55,
    "avoidBadSurfaces": 0.7
  },
  "requiredPreferences": []
}
```

Successful output contains only an opaque provider reference:

```json
{ "preparedKey": "custom_1786873526709" }
```

The browser warm-up endpoint accepts the same flat provider-neutral request as
the route endpoint, normally without anchors and with `requestVariants=false`.
It returns `{ "prepared": 1, "engineErrors": [] }`; `prepared` counts engines
that produced or reused a prepared key. Ordinary drawing includes only the
active editor engine. A true variant request prepares the enabled route
engines discovered by the host, or the explicit engine list supplied by an API
client, as part of the route flow.

### 5.7 Effective-control routing metadata

In addition to visible and hidden controls,
`POST /api/v1/routing/effective-controls` returns three engine-keyed maps and,
for a single selected engine, its effective `nativeControlGroups`:

- `profileRevisions` lets persisted segment provenance detect profile changes;
- `profileUploadRequired` tells the editor that the effective profile requires
  provider-side materialization;
- `profilePreparationSupported` reports whether uploads can be prepared once
  per engine. Exact fan-out admission remains server-owned.

## 6. Persistence and resolution

### 6.1 Collections

`routing_settings` contains one JSON configuration per scope:

```text
routing_settings
  scope       builtin | admin | user
  user        relation for user scope
  config      JSON
```

Important config keys are `primaryRoutePluginId`, `elevationPluginId`,
`maneuverPluginId`, `defaultVariantCount`, `defaultPreferences`, and
`exposedFeatures`. Builtin settings contain no provider IDs. After the scoped
settings merge, a missing role selection is resolved from enabled instances
that implement the corresponding executable capability, using stable instance
setup order and then plugin ID. Elevation and maneuver roles prefer the resolved
primary engine when it supports that capability. Explicit admin and user
selections remain authoritative. Parallel variant requests that omit an
explicit engine list discover additional enabled `route.v1` instances on the
server. Explicit API engine lists remain authoritative. Automatic discovery
keeps the active engine first, then uses the stable setup order of the user's
enabled plugin instances (`created`, with record ID as tie-breaker), up to the
host's four-engine bound.
Users can disable all routing through
`exposedFeatures.routing` and can independently disable variant selection
through `exposedFeatures.variants`; builtin and administrator feature gates
remain upper bounds and cannot be re-enabled at user scope.

`exposedFeatures.navigation` is a builtin/administrator policy gate. User-scope
values are ignored and user patches cannot change it. Maneuver engines are not
lifecycle-protected selections: disabling the preferred engine makes the host
try another eligible enabled engine and otherwise returns
`maneuver_engine_unavailable`.

A plugin referenced by the effective primary or elevation selection cannot be
disabled or deleted until the user selects a replacement or disables routing.
Additional variant engines are derived from enabled instances and are not
protected as persistent selections. Disabling routing retains the engine selections for later
reactivation, hides routing controls, and rejects route, variant,
profile-preparation, and elevation execution.
Default activation is controlled by a host-owned first-party plugin policy, not
by plugin manifests. Valhalla is currently the only default-active plugin;
BRouter and community routing plugins remain opt-in. When a default-active
plugin's separately installed bundle is discovered successfully, synchronization
creates any missing instances but never re-enables an existing instance. Existing
users are processed in bounded transactional batches; new users receive instances
for all installed default-active plugins. Absence is treated as incomplete
provisioning and repaired; an existing disabled row is the persistent opt-out.
The first successful Valhalla discovery is the only point at which the trusted
policy imports `VALHALLA_URL`; failed discovery and routine synchronizations do
not import or overwrite it. Removing the bundle retains a non-executable cache
marker plus administrator configuration, so reinstalling it does not reopen the
import. This trust and compatibility policy is separate from routing resolution:
builtin routing settings contain no provider ID, and the generic routing host
resolves missing roles by executable capability and stable setup order. It
contains no provider-specific profile, request, response, or control translation.

`routing_profiles` stores materialized provider-native profiles:

```text
routing_profiles
  scope             admin | user
  user              relation for user scope
  plugin_id
  key, name
  kind              custom_file | generated | native_config
  mode              foot | bike | motor | mixed | other
  content_base64
  content_type
  metadata
  native_config
  enabled
```

`routing_profile_mappings` stores overrides:

```text
routing_profile_mappings
  scope                 admin | user
  user                  relation for user scope
  category              canonical Wanderer category name
  subcategory           optional canonical subcategory name
  plugin_id
  instance_id           optional plugin-instance specialization
  native_profile_key    optional manifest profile
  profile               optional relation to routing_profiles
  preferences
  native_config
```

A mapping must select a `native_profile_key` or a materialized `profile`.
Built-in records are read-only; ordinary authenticated writes become user scope,
and admin scope requires superuser authentication.

### 6.2 Routing settings precedence

`ResolveRoutingSettings` merges:

```text
builtin -> admin -> user
```

Scalar engine selections are replaced by the highest non-empty layer. Default
preferences are deep-merged. Legacy invalid preference entries are discarded
per layer on reads and reported once; writes remain strict and repair the
persisted map before merging a valid patch. If otherwise-valid layers together
exceed the 64-key effective limit, admission follows scope precedence
(`user`, then `admin`, then `builtin`) rather than map-key order. Feature
exposure is a ceiling: for a feature to be
enabled, every layer that defines it must allow it. A user therefore cannot
re-enable a feature disabled by an administrator or built-in policy.
The plugin-selection guard and HTTP settings use the same effective resolver
from `pluginsystem`. It selects one record per scope deterministically by record
ID and owns scalar/list precedence, preference deep-merging, and feature
ceilings. The HTTP layer only converts the resolved JSON config into its
response model, so duplicate legacy rows or later merge-policy changes cannot
produce different selections in the two call paths.
Only the documented boolean `exposedFeatures` keys are accepted. Patching the
feature map preserves known sibling settings and removes legacy unknown keys.
`routeCandidates` is additionally an administrator exposure switch: it defaults
off for ordinary users and can be opened by an explicit admin-scope setting;
superusers always retain access for diagnostics.

`InitRoutingDefaults` seeds no provider IDs. It seeds three variants for the
explicit variant UI, segment routing, the mode-independent neutral hill value,
and enabled routing, standard/native controls, profile upload, parallel routing,
and variants. Effective settings fill missing engine roles dynamically as
described above. Ordinary routing still requests one candidate. The broader
candidate-set endpoint remains disabled until an administrator exposes it.

### 6.3 Mapping precedence

Mappings start with `metadata.routing.categoryMappings`, then admin and user
records overlay the same resolution identity. Selection favors:

1. higher scope: `user > admin > manifest`;
2. instance-specific over plugin-wide within a scope;
3. exact subcategory over category-only fallback within otherwise equal
   precedence.

The canonical category and subcategory `name` values are keys. Translated labels
and database IDs are presentation and relation details only.

No mapping means `mapping_missing`; the host never guesses a provider profile.
An exact subcategory mapping is optional: the category-only mapping remains the
fallback.

### 6.4 Profile, config, and preference merge order

For a built-in profile, native config is merged as:

```text
manifest profile nativeConfig
  -> request profile nativeConfig
  -> mapping nativeConfig
```

For a materialized profile it is:

```text
routing_profiles.native_config
  -> request profile nativeConfig
  -> mapping nativeConfig
```

The mapping is therefore authoritative for profile-specific configuration.

Standard preferences are merged as:

```text
resolved routing_settings.defaultPreferences
  -> mapping preferences
  -> request preferences
```

For a custom file, unsupported canonical preferences are removed after this
merge. Provider-specific keys are retained.

### 6.5 Standard preference discovery

The canonical keys currently known to the host are:

| Key | Typical mode | Valhalla target | BRouter target where exposed |
| --- | --- | --- | --- |
| `speedPreference` | foot, bike, motor | `walking_speed`, `cycling_speed`, `top_speed` | bike `bikerPower`; motor `vmax`; no target on the Poutnik bases |
| `hillPreference` | foot, bike | `use_hills` | `consider_elevation`, `uphillcost`, `uphillcostvalue`, `hills`, `avoid_steep_inclines` |
| `maxHikingDifficulty` | foot | `max_hiking_difficulty` | `SAC_scale_limit`, `SAC_scale_preferred` |
| `roadPreference` | bike | `use_roads` | `avoid_unsafe`, `consider_traffic`, `consider_traffic_estimate` |
| `avoidBadSurfaces` | bike, motor | `avoid_bad_surfaces` | `unpavedPenalty`, `prefer_unpaved_paths`, `MTB_factor`, motor `avoid_unpaved` |
| `vehicleWidth` | motor | `width` | not declared by the current BRouter manifest |
| `vehicleHeight` | motor | `height` | not declared by the current BRouter manifest |

BRouter targets differ per base, because the upstream profiles expose different
knobs: the BRouter core profiles carry numeric costs and a kinematic model,
while the Poutnik templates (`hiking-mountain`, `mtb`) and quaelnix' `gravel`
expose stepped switches and no rider power at all. A curated profile therefore
declares its own `supportedPreferences`, which narrows the plugin-wide
declaration. A per-profile capability may narrow manifest support but cannot
expand it, and a profile that declares an empty list supports none.

`ResolveRoutingEffectiveControls` computes the visible controls by intersecting:

1. the selected engine manifests' `standardPreferences` for the resolved mode;
2. for custom files, each selected profile's introspected
   `supportedPreferences`.

Explicit settings and category-mapping values populate the returned controls
as current values. Unsupported known preferences appear in `hiddenControls` with
`reason=unsupported_for_selection`. The route editor renders `controls` and
`nativeControlGroups` directly from their labels, types, bounds, options,
optional display units, targets, and paths. Standard-control labels and bucket
values use the browser's locale catalog; provider labels remain the fallback
for plugin-owned native controls. It contains no provider-specific control
table; when no effective-control response is available it renders none.

A native profile may declare `preferenceDefaults` for canonical controls whose
neutral value is profile-specific. These values are presentation fallbacks, not
request preferences: the editor displays them for an untouched control but only
sends a value after the user changes it. Settings and category-mapping values
remain explicit overrides. With several selected engines, a profile default is
shown only when every selected profile declares the same value.

For custom BRouter files, introspection probes both ends of each canonical
preference range. A preference is advertised only if at least one exposed
profile parameter actually changes, so a recognized but ineffective mapping
does not produce a no-op control.

### 6.6 Native advanced controls

Native controls are not comparable across providers. A manifest profile can
declare `nativeControlGroups`; a custom profile can return groups from
`profile_introspect.v1`.

Each control declares a target path, for example:

```json
{
  "key": "SAC_scale_limit",
  "type": "number",
  "ui": "slider",
  "min": 0,
  "max": 6,
  "step": 1,
  "unit": "level",
  "path": ["parameters", "SAC_scale_limit"]
}
```

`unit` is optional presentation metadata. Standard `speedPreference` values
declare `km/h`; the browser converts them through the user's configured unit
system. Unknown plugin-owned units are shown as a suffix without adding
provider-specific editor logic.

The plugin settings UI writes the selected value into mapping `nativeConfig` at
the declared path. `ResolveRoutingNativeControls` returns current/default values
for rendering but does not make native values part of the standard preference
contract.

## 7. HTTP API

Every browser route has a SvelteKit proxy under `/api/v1/routing`; PocketBase
implements the corresponding `/plugins/routing` endpoint.

| Browser endpoint | Method | PocketBase handler | Purpose |
| --- | --- | --- | --- |
| `/api/v1/routing/engines` | GET | `PluginSystemRoutingEnginesGet` | Lists installed routing plugins and whether the current user enabled them. |
| `/api/v1/routing/settings` | GET, PATCH | `PluginSystemRoutingSettingsGet/Patch` | Resolves or updates user routing settings. |
| `/api/v1/routing/admin/settings` | GET, PATCH | `PluginSystemRoutingAdminSettingsGet/Patch` | Reads or updates admin routing settings. |
| `/api/v1/routing/mappings` | GET, PUT | `PluginSystemRoutingMappingsGet/Put` | Lists effective mappings or upserts an override. |
| `/api/v1/routing/mappings/{id}` | PATCH | `PluginSystemRoutingMappingsPatch` | Updates an owned mapping. |
| `/api/v1/routing/profiles` | GET, PUT | `PluginSystemRoutingProfilesGet/Put` | Lists discovery/materialized profiles or creates one. |
| `/api/v1/routing/profiles/{id}` | PATCH, DELETE | `PluginSystemRoutingProfilesPatch/Delete` | Updates or removes an owned profile. |
| `/api/v1/routing/effective-controls` | POST | `PluginSystemRoutingEffectiveControls` | Resolves standard controls for category, subcategory, and engines. |
| `/api/v1/routing/native-controls` | POST | `PluginSystemRoutingNativeControls` | Resolves advanced controls for a profile. |
| `/api/v1/routing/profile-prepare` | POST | `PluginSystemRoutingProfilePrepare` | Resolves and prepares profile inputs without anchors; returns only counts and structured engine errors. |
| `/api/v1/routing/check` | POST | `PluginSystemRoutingCheck` | Checks route-provider reachability with draft plugin config. |
| `/api/v1/routing/route` | POST | `PluginSystemRoutingRoute` | Resolves, curates, and calculates up to the explicitly requested UI candidate count. |
| `/api/v1/routing/route-candidates` | POST | `PluginSystemRoutingRouteCandidates` | Returns a broader normalized candidate set for superusers or when exposed by an administrator. |
| `/api/v1/routing/elevation` | POST | `PluginSystemRoutingElevation` | Completes elevation independently. |

## 8. Workflows

### 8.1 Plugin installation and discovery

Official Wanderer images provide the plugin host but no provider bundles. The
operator first downloads a separate release asset or builds a first-party or
trusted community plugin from source.

1. The extracted or locally built bundle is a direct child of `data/plugins`, containing
   `plugin.json` and `plugin.wasm`.
2. `pluginsystem.DiscoverLocalPlugins` reads and validates the manifest and
   entrypoint.
3. `Manager.SyncInstalledPlugins` mirrors the manifest, path, status, and
   defaults into `installed_plugins`.
4. First successful Valhalla discovery applies the one-time trusted
   `VALHALLA_URL` compatibility import. Every successful synchronization
   provisions enabled instances for existing users who lack one, repairing a
   partial earlier run while preserving stored connector configuration and
   existing user choices. A removed default-active bundle leaves only a
   non-executable cache marker; reinstalling it preserves the first-success
   boundary and administrator configuration.
5. New users receive instances for installed plugins marked default-active by
   host policy. The effective-settings resolver fills missing engine roles from
   enabled executable capabilities without naming a provider.
6. `LoadInstalledPlugin` serves request hot paths from that cache and falls
   back to disk when the record is missing or unusable.
7. `routingEngines` combines installed routing plugins with enabled
   `plugin_instances` owned by the current user.
8. `routingDiscoveryProfiles` and `builtinRoutingProfileMappings` expose
   manifest profiles and mappings as virtual built-ins.

### 8.2 Enabling and checking a routing plugin

1. The generic plugin settings modal collects instance configuration.
2. `checkRoutingPlugin` posts plugin ID and draft plugin config.
3. `PluginSystemRoutingCheck` loads `route.v1`, rejects non-routing plugins, and
   merges the draft plugin config without allowing submitted host config.
4. `routingPluginCheckRequest` selects a safe manifest default profile/mode and
   constructs a small probe route.
5. Provider/network failures make the check fail. Coverage errors such as
   `no_route` or `unsupported_profile` prove that the provider was reachable and
   are accepted as a successful connectivity check.

Normal route/elevation calls require an enabled user-owned `plugin_instances`
record. `routingPluginInvocation` resolves the instance, decrypts its auth, and
constructs effective config and host policy.

### 8.3 Selecting global route and elevation engines

1. `web/src/routes/settings/routing/+page.svelte` loads `routingSettings` and
   `routingEngines`.
2. Engine selectors show enabled plugins that advertise the corresponding
   role.
3. `updateRoutingSettings` patches user scope without replacing unrelated
   settings.
4. The trail editor prefers the configured engine and falls back to the first
   enabled engine with the needed role.
5. Route and elevation selections are independent; BRouter can route while
   Valhalla supplies missing elevation.

### 8.4 Editing category/profile mappings

1. `plugin_routing_settings.svelte` loads settings, profiles, and resolved
   mappings when the plugin modal opens.
2. Categories use the user's custom category order and visibility preferences.
   Subcategories use their custom order and are shown below collapsible category
   rows. Category and subcategory icons come from the common category utilities.
3. Profile choices include enabled built-ins and enabled resolved custom
   profiles for this plugin, sorted by localized display label.
4. An unresolved custom profile with `mode=other` is deliberately absent from
   assignment choices.
5. Selecting a category profile updates the displayed inherited subcategory
   choices locally. Saving persists only dirty mapping rows.
6. `saveMapping` writes either `nativeProfileKey` or `profileId` plus native
   config through the mapping PUT/PATCH endpoints.
7. At route time the server re-resolves precedence; frontend inheritance is a
   presentation aid, not the authority.

### 8.5 Opening and saving advanced controls

1. The icon-only Advanced button is enabled only when discovery metadata or
   profile introspection indicates possible native controls.
2. `loadNativeControls` posts plugin/instance and the selected native profile or
   profile ID, including the current mapping config.
3. `ResolveRoutingNativeControls` loads manifest groups or calls profile
   introspection for a materialized profile.
4. The UI edits values at each declared `path` inside its local native config.
5. Saving the mapping persists that config; the route resolver merges it last.

### 8.6 Uploading a custom profile

1. The plus button opens a hidden file input using the plugin manifest's
   accepted extensions.
2. `uploadProfile` checks the manifest UI byte limit, derives name and a unique
   slug key, base64-encodes the bytes, and creates a `custom_file` profile.
3. The generic host validates required fields, valid base64, non-empty decoded
   content, a 256 KiB decoded ceiling, ownership, and content type presence.
4. `enrichCustomRoutingProfile` calls `profile_introspect.v1` when available.
5. The provider performs its own stricter validation. BRouter requires UTF-8,
   no NUL bytes, lines no longer than 4096 bytes, and at most 64 KiB.
6. The host stores detected mode, detection status, and supported preference
   keys with the original content.
7. A resolved profile appears in the alphabetical profile list and assignment
   selector. An unresolved profile opens directly in the editor for manual
   classification.

The collection field has capacity for a base64 representation of 256 KiB. This
is intentionally larger than BRouter's current plugin-specific 64 KiB maximum.

### 8.7 Editing a custom profile

1. `RoutingProfileEditorModal.openModal` decodes the stored UTF-8 text and opens
   the generic code editor.
2. The profile name remains editable. The plugin's declarative
   `editorLanguage` controls highlighting; BRouter declares directives,
   keywords, atoms, built-ins, and `#` comments.
3. Saving validates the UTF-8 byte length and PATCHes name/content.
4. If content changed after an automatic result, the editor removes
   `modeDetection`; the backend runs automatic introspection again.
5. If the plugin still cannot classify the content, the modal remains open and
   displays the only visible mode selector.
6. Selecting `foot`, `bike`, or `motor` stores
   `metadata.modeDetection=manual`. Mode is otherwise not shown in normal UI.
7. Editing does not require re-uploading a file and does not change the stable
   profile record ID, so existing mappings remain valid.

### 8.8 Downloading, disabling, and deleting a custom profile

- Download is client-side. Both the list and editor decode `contentBase64`,
  build a safe filename from original metadata/name, and save a Blob.
- The ordinary toggle PATCHes `enabled`. Disabled profiles remain visible in
  the custom-profile management list but are not available for new mappings.
- A mapping to a disabled profile causes route resolution to reject it.
- Delete is disabled in the UI when any resolved mapping references the
  profile. `routingProfileInUse` repeats that protection server-side before
  deletion.
- Removing a profile deletes only its host record. A provider-side BRouter
  upload may be reused through an opaque host cache for at most 15 minutes, but
  it is not durable Wanderer state and is never treated as the profile source.

### 8.9 Initializing the trail editor

1. The edit page initializes route, anchor, and undo/redo stores.
2. `initRoutingPhaseThreeState` loads settings and engines in parallel.
3. It selects enabled route and elevation engines. The route engine remains
   active when a variant from another engine is applied; users may explicitly
   switch it in the route-editor panel without changing the global default or
   the existing geometry.
4. `selectedRoutingCategory` translates stored category/subcategory relations
   into canonical names.
5. `refreshRoutingEffectiveControls` posts those names and the selected engine.
6. The returned profile mode selects the internal option bucket
   (`pedestrian`, `bicycle`, or `auto`).
7. Effective control defaults populate that bucket. The route editor shows only
   the returned control keys.
8. Changing category or subcategory repeats this resolution automatically.
9. While drawing and automatic routing are active, a 500 ms debounced effect
   calls `prepareRoutingProfile` for the primary engine with
   `requestVariants=false`. It reruns when profile-relevant category, engine,
   mode, profile revision, preference, or native-control inputs change.

The warm-up intentionally excludes additional engines. Normal segment drawing
does not use them, and including them could both waste provider work and turn a
valid single-engine interaction into a forbidden parallel request. Explicit
variant routing resolves and prepares its participating engines in the real route
request.

### 8.10 Drawing a new segment

1. The first map click creates an anchor without a route call.
2. Each later click runs `addAnchorAndRecalculate` between the previous and new
   anchor.
3. `calculateRouteBetween` forwards the generic preference/native-config maps
   produced by host control metadata and posts category/subcategory with an
   empty profile key.
4. `PluginSystemRoutingRoute` chooses the primary plugin if omitted, resolves
   enabled instance, mapping, profile, mode, preferences, and config, then
   validates the request and its fan-out budget.
5. `prepareRoutingRuntimeProfiles` obtains or reuses a prepared key for each
   participating engine that declares `profile_prepare.v1`. Preparation errors
   are soft unless the now-required inline fallback exceeds the budget.
6. `callPreparedRoutingRoutePlugin` opens the normal `route_v1` path. If the
   provider rejects a prepared key with `unsupported_profile`, concurrent
   segment calls share one invalidation and re-prepare through `sync.Once`, then
   all retry with the common refreshed key.
7. `normalizeRoutingRouteOutput` validates each candidate and assigns host-owned
   provenance.
8. The frontend accepts the first candidate's first segment, decodes its
   polyline, and optionally completes elevation.
9. `applySnappedAnchors` moves anchor state and markers to provider-snapped
   coordinates.
10. `insertIntoRoute`, `normalizeRouteTime`, and
   `updateTrailWithRouteData` append the GPX segment and refresh trail totals.

The default request always asks for one result and invokes only the active
engine. “Find route variants” is an explicit action: when parallel routing is
enabled, the host discovers additional enabled route-capable engines, requests bounded native alternatives, curates by
quality and geometry diversity, and preserves partial provider failures in
`engineErrors`. The variant panel reports each candidate's engine provenance.
The browser exposes the public count and anchor limits, while the host remains
the sole authority for exact fan-out work and returns structured limit details
when a request does not fit.
For automatically discovered engines, the server first reserves one candidate
per segment for as many engines as fit, always retaining the active engine. It
then distributes the remaining budget across provider-native alternatives in
balanced, primary-first rounds. Additional engines are dropped only when their
baseline work no longer fits; the server emits
`routing_parallel_engines_reduced_for_fanout` once when that happens. Explicit
API engine lists continue to fail preflight instead of being rewritten when
they exceed the budget. Clamping does not overwrite the configured default;
“Find more” becomes available if later route changes permit additional
variants.

Variant GPX objects are materialized once per candidate set and reused for map
and elevation previews. Hover and keyboard focus select the same preview. When
an anchor changes, existing candidates are retained but marked stale; they
cannot be applied until refresh recalculates them against the new original
route, and their geometry and elevation snapshots are removed from the map in
the meantime.

Geometric diversity is measured as bidirectional route overlap. Each route is
sampled at 200 distance-distributed points and compared against the other
route's sampled corridor. Candidates are treated as too similar when at least
80 percent of both routes lie inside a corridor of 0.1 percent of route length,
bounded to 30–200 metres. This filters routes that differ only by a localized
detour while retaining alternatives that follow a sustained different corridor.

Routes of at most 300 metres are reduced to one final candidate after provider
execution and emit `routing_variants_reduced_for_short_route`. This keeps the
result useful without making short-route policy a hidden provider behavior.

The normal single-engine, single-result segment request sends the complete
anchor list in one plugin call. Explicit variants route adjacent anchor pairs
independently and may compose a whole route from different engines at anchor
boundaries. Via mode sends the
ordered anchor list once to each via-capable engine and never mixes via and
segment candidates in the same request. Accepted segments persist provenance
in the trail's `routing_provenance` metadata; imported GPX has unknown
provenance and remains inert. The host emits this metadata with route results,
but the browser persists it through the normal trail form. It is therefore an
untrusted UI hint, not authoritative server-owned audit data.

When automatic routing is off, `calculateRouteBetween` creates a straight
encoded line and duration zero; it may still request elevation if an elevation
engine is configured.

### 8.11 Moving, inserting, removing, and reordering anchors

- Dragging an anchor calls `recalculateRoute` for the adjacent one or two
  segments only.
- Dragging a segment inserts an anchor and calculates the two replacement
  segments.
- Removing an endpoint deletes one segment. Removing an interior anchor deletes
  one side and recalculates the new bridge segment.
- `recalculateRouteFromAnchors` reuses unaffected segments after reordering and
  recalculates only boundary segments in parallel.
- Every successful recalculation applies snapped anchors, replaces affected
  `TrackSegment` values, normalizes time, and participates in undo/redo.
- Errors leave unrelated segments intact and surface through a toast.

### 8.12 Elevation completion

1. A route candidate may include `elevation.heights` aligned to its geometry.
2. `calculateRouteBetween` uses those heights only when their count equals the
   decoded point count.
3. Otherwise it posts the same encoded polyline to the selected elevation
   engine.
4. Valhalla forwards it to `/height`; the host normalizes status and the browser
   assigns heights to GPX waypoints.
5. The separate `recalculateElevationData` editor action recalculates elevation
   for an existing route through the GPX/store flow.

## 9. Provider behavior

### 9.1 Valhalla

Valhalla exposes native profiles `pedestrian`, `bicycle`, and `auto`.

`handleRoute`:

1. uses the resolved profile key or falls back from generic mode;
2. translates standard preferences with `valhallaCostingOptions`;
3. overlays `profile.nativeConfig` as provider-native costing options;
4. posts to `/route` with `directions_type=none`;
5. converts legs to one normalized segment per anchor pair;
6. joins multi-leg geometry into one precision-6 encoded polyline;
7. returns provider-snapped locations in original anchor order;
8. requests up to three provider `alternates` for a two-anchor request and
   normalizes the primary plus `alternates[].trip` as at most four provider
   candidates.

Valhalla forces native alternatives to zero for routes with more than two
waypoints. Such via routes can still receive alternatives from parallel
via-capable engines, while segment mode requests alternatives independently for
each anchor pair.

`handleElevation` forwards encoded polyline or coordinates to `/height` and
returns the provider height array.

Valhalla does not support custom profile upload. Its advanced controls are
manifest-defined and ultimately become costing options such as
`bicycle_type`.

### 9.2 BRouter native, generated, and custom profiles

`brouterProfileKey` has four paths:

1. **Prepared profile:** use the host-supplied opaque `preparedKey` directly;
   do not decode, render, or upload profile content again.
2. **Custom file:** decode and validate the stored `.brf`, infer its upstream
   base from the parameters it exposes, translate supported preferences into
   parameter values, render only exposed parameters into a temporary copy,
   upload it to `/brouter/profile`, and route with the returned profile ID.
3. **Generated or curated profile:** load the embedded base by `templateKey`,
   validate and replace the template parameters, upload it, and use the returned
   ID. Curated profiles always take this path: routing with the bare provider
   key would drop the category preset, and the provider's own copy of that key
   can differ from the embedded one.
4. **Profile without a base:** use the provider's native key directly. This
   applies to `shortest`, which exposes nothing Wanderer would rewrite.

`handleProfilePrepare` runs the same profile resolution without anchors and
returns the resulting profile ID. This makes custom/generated uploads reusable
across segment calls and later equivalent route requests. Native profiles also
produce a prepared key, but preparation is effectively a cheap identity
resolution for them.

`requestBRouterRoute` sends anchors as `lon,lat|lon,lat`, requests GeoJSON from
`/brouter`, and requests bounded alternatives with `alternativeidx=0..3`.
`core.CandidateFromFeature` converts
the geometry to Wanderer's precision-6 polyline, calculates segments and
snapped anchors, and preserves BRouter elevation when present.

The original stored `.brf` is never rewritten during routing. Only the
temporary provider upload contains resolved preference and advanced-control
values. Its opaque ID can remain in the host cache for up to 15 minutes.

There is currently no GraphHopper adapter in the repository, so no GraphHopper
via or native-alternative behavior is advertised by discovery.

### 9.3 BRouter mode detection

`DetectProfileMode` ignores comments and recognizes enabled assignments for:

| Assignment | Mode |
| --- | --- |
| `validForFoot` | `foot` |
| `validForBikes` | `bike` |
| `validForCars` | `motor` |

`true`, `1`, and numeric `1.0` forms are accepted, with or without `=`. Exactly
one distinct enabled mode must be present. None or several yield no automatic
mode and trigger the manual fallback workflow.

This logic is BRouter-specific and lives entirely in the BRouter plugin/core,
not in generic Wanderer host code.

### 9.4 BRouter parameter annotations and supported controls

Custom profile parameters are opt-in. The parser recognizes lines shaped like:

```text
assign consider_elevation 1 # %Consider elevation% | Include elevation cost | boolean
assign SAC_scale_limit 3  # %Maximum SAC scale% | Highest allowed scale | [0=T0,1=T1,2=T2,3=T3,4=T4,5=T5,6=T6]
assign bikerPower 100      # %Biker power% | Estimated rider power | number
```

The pieces are default assignment, display label, description, and type.
Supported types are `boolean`, `number`, and numeric enum lists.

Only the enum form carries a range. A plain `number` is therefore unbounded: the
annotation syntax has no way to express limits, and inventing them rejects
values the profiles themselves use — Poutnik's terrain factors are negative and
a car's mass is in the thousands. Such a parameter renders as a number field
rather than a slider, and only finiteness is enforced. Enum parameters keep
their range, and because the host renders any control carrying options as a
select, the selected option arrives as text and is normalized back to the
control's declared type before it is stored or forwarded.

Only such explicitly annotated parameters become native controls and count as evidence
that a custom profile understands a standard preference.

The capability matrix is derived rather than tabulated. `TemplateKeyForParameters`
infers the upstream base from the exposed parameters, and
`SupportedStandardPreferences` then asks the preference adapter itself which
parameters it would write for each preference of that mode; a preference counts
as supported when at least one of them is exposed. A profile can therefore never
advertise a control the adapter would not actually write.

A profile may contain similarly named internal assignments without exposing
them. Those do not create UI controls or claim standard-option support. This is
deliberately conservative: Wanderer must not mutate arbitrary profile internals
merely because a variable name happens to match.

### 9.5 BRouter preference translation

`NativeConfigWithPreferences` dispatches on the base the profile is rendered
from. It clones input config before writing `parameters`, and it writes nothing
for a preference the base cannot honor.

- `hike`: hill preference drives the numeric `uphillcostvalue`/`downhillcostvalue`
  down to zero and deliberately leaves `consider_elevation` switched on. That
  switch has a second job in this profile — it also triples the cost of steps —
  so moving it with the slider would make the hill-tolerant end of the range
  quietly start avoiding stairs; hiking difficulty maps to
  `SAC_scale_limit` plus a clearly lower `SAC_scale_preferred`; speed has no
  target because the profile has no kinematic model;
- `bike-balanced`: speed maps to bounded `bikerPower`, hill preference to
  interpolated `uphillcost`, road preference to the two boolean thresholds
  `avoid_unsafe` and `consider_traffic`, surfaces to `unpavedPenalty`;
- `fastbike`: same, except road preference maps to the real 0..1
  `consider_traffic` scale;
- `gravel`: speed maps to `bikerPower` around the profile's own 150 W default;
  hill preference and surfaces collapse into three steps each, because the
  profile derives its costs from booleans;
- `mtb`: hill preference maps to the discrete `hills` mode, road preference to
  `avoid_unsafe`, surfaces to the `MTB_factor` shift; speed has no target;
- `car`: speed maps to `vmax` bounded to `MinCarSpeed..300` — lower target
  speeds make the kinematic model explore so much of the graph that providers
  cut the search short — and surfaces map to `avoid_unpaved`.

Neutral preference values keep the base profile's own cost values, so a category
preset only has to name what it wants to deviate in, and a parameter no
preference supplies is left untouched rather than rewritten with a Wanderer
default. The one deliberate exception is the hiking base, which ships with
elevation costs switched off; Wanderer keeps them switched on and expresses the
preference through the cost values instead.

For custom files, `renderBRouterCustomProfile` applies only parameters that were
explicitly parsed from annotations. Values are type-checked and bounded before
the temporary upload.

## 10. Validation, errors, and failure semantics

### 10.1 Limits

The generic route host currently enforces:

| Limit | Value |
| --- | --- |
| Plugin invocation timeout | 30 seconds |
| Host requests per routing call | 6 |
| Engines per route flow | 4 |
| Curated UI variants | 3 |
| Provider candidates per call | 4 |
| Broader candidate set | 12 |
| Parallel plugin calls | 8 |
| Anchors | 2..50 |
| Anchors for explicit variants | 2..16 |
| Segment/engine/provider-request fan-out work | 64 |
| Decoded polyline points | 20,000 |
| Elevation points | 20,000 |
| Plugin response body | 4 MiB |
| Decoded persisted profile | 256 KiB generic host maximum |
| Preference object | 64 scalar entries; canonical values use host ranges |
| Metadata/native config | 64 KiB, 256 total entries, 8 nesting levels, 4 KiB strings |
| BRouter `.brf` | 64 KiB plugin maximum |
| Profile warm-up rate | 20 requests per authenticated user per minute |
| Effective/native controls | 60 requests per authenticated user per rolling minute |
| Explicit variant rate | 60 requests per authenticated user per minute |
| Broader candidate-set rate | 10 requests per authenticated user per minute |
| Prepared-profile cache | 512 process-local entries |
| Prepared-profile lifetime | Absolute 15 minutes; reads do not extend it |
| Prepared-key length | 1,024 bytes |
| Short-route variant reduction | 300 metres or less |

Provider manifests add connector-specific content-type, upload/download, path,
and response limits.

The fan-out estimate counts provider-native candidates per segment and per
engine. A profile upload is counted once per preparation-capable engine, but
once per segment for an engine without `profile_prepare.v1`. After an initial
preparation failure, `routingSegmentExecutionWork` recalculates the latter
cost before execution and repeats the same breadth-first planning. Warning
codes are appended idempotently across both planning stages. The exceptional
case where an already prepared key is
rejected and its re-prepare also fails happens after preflight: every waiting
route call retries once through the canonical inline-profile path, so those
provider uploads are intentionally outside the estimate but remain bounded by
the number of outstanding route calls.

Executable `route.v1`, `elevation.v1`, `maneuvers.v1`, and `round_trip.v1`
capabilities are authoritative for engine roles. The host exposes one
canonical role list to clients instead of letting metadata-only and
capability-only checks disagree.

Request preferences are fail-closed against the selected plugin's
mode-filtered `standardPreferences` plus its explicit `providerPreferences`
allowlist. Every undeclared key is removed consistently for normal routes,
round trips, and maneuver generation. Client native configuration is likewise
reduced to declared `nativeControlGroups` paths and type/range checked.

### 10.2 Structured errors

Routing errors use:

```json
{
  "data": {
    "code": "mapping_missing",
    "message": "no routing profile mapping exists for the selected category"
  },
  "message": "no routing profile mapping exists for the selected category",
  "status": 422,
  "detail": {}
}
```

`detail` is omitted when no additional context exists. Total route failure may
include `engineErrors` plus resolved request context.

| Status | Representative codes |
| --- | --- |
| `400` | `invalid_request`, `invalid_coordinate`, `anchor_limit_exceeded`, `variant_limit_exceeded`, `engine_limit_exceeded`, `point_limit_exceeded`, `fanout_limit_exceeded` |
| `422` | `no_route`, `unsupported_profile`, `mapping_missing`, `missing_preference`, `invalid_candidate`, `invalid_geometry`, `routing_mode_unavailable`, `profile_preparation_fanout_limit_exceeded` |
| `504` | `provider_timeout`, `timeout` |
| `502` | `provider_error`, `provider_unavailable`, `connector_error`, `plugin_error`, `response_too_large`, `internal_error`, invalid plugin output |

The profile PUT/PATCH handlers convert introspection failures through the same
structured error path, so WASM or provider failures retain their actual message
instead of becoming an empty generic validation response.

### 10.3 Failure ownership

| Layer | Failure meaning |
| --- | --- |
| Browser/SvelteKit | Authentication proxy or malformed HTTP response. |
| PocketBase request validation | Invalid anchors, profile content, ownership, limits, or mapping. |
| Worker runtime | WASM trap, timeout, oversized response, or host-function failure. |
| Plugin protocol `error` | Provider-specific but expected failure with stable code/message. |
| Candidate normalization | Plugin completed, but its route output violated host invariants. |
| Provider connector | Network, status, content type, redirect, trust, or size failure. |

## 11. Security properties

- Every normal route/elevation call requires authentication and an enabled
  plugin instance owned by that user.
- Admin settings/profiles/mappings require superuser authentication; user
  records are ownership-checked.
- Plugins can access only declared connectors and allowed path prefixes.
- Private networks, TLS overrides, redirect origins, credentials, content
  types, response sizes, and timeouts remain host policy.
- The worker runs out of process and communicates through bounded RPC frames.
- Plugin output is untrusted until normalized.
- Custom profile content is bounded opaque text/data to the host. BRouter
  interprets it only inside its plugin flow and never executes it as host code.
- Custom profile deletion is blocked while any mapping references it.
- Unsupported standard preferences for custom profiles are neither displayed
  nor sent to the plugin.

## 12. Function index

### 12.1 Host route/elevation path

| Function | File | Role |
| --- | --- | --- |
| `PluginSystemRoutingRoute` | `db/routes/plugin_system_routing.go` | Curated-route HTTP entry point delegating to the shared route pipeline. |
| `PluginSystemRoutingRouteCandidates`, `pluginSystemRoutingRoute` | `db/routes/plugin_system_routing_variants.go` | Gated broad endpoint and full shared route orchestration. |
| `routingRuntimesForMode`, `routingSelectionsForRequest` | same | Resolve participating engine instances and per-engine requests. |
| `routingSegmentFanoutWork`, `routingSegmentExecutionWork` | same | Pre-preparation and post-failure work accounting. |
| `routeWholeCandidates`, `routeSegmentCandidates`, `routeViaCandidates` | same | Execute the selected routing mode and collect partial engine failures. |
| `composeRoutingCandidate`, `curateRoutingCandidates` | same | Compose complete anchor-pair choices and curate quality/diversity. |
| `enrichRoutingCandidateElevation` | same | Adds elevation to a bounded shortlist before final curation. |
| `PluginSystemRoutingCheck` | `db/routes/plugin_system_routing.go` | Draft provider connectivity check. |
| `PluginSystemRoutingElevation` | same | Independent elevation orchestration. |
| `routingPluginInvocation` | same | Resolves capability, enabled instance, auth, and config. |
| `callRoutingRoutePlugin`, `callRoutingElevationPlugin` | same | Typed route/elevation wrappers around generic WASM invocation. |
| `callRoutingProfileIntrospectPlugin`, `callRoutingProfilePreparePlugin` | same | Typed optional profile-capability wrappers. |
| `callRoutingPlugin` | same | Worker session, timeout, policy, JSON, request budget, and response bound. |
| `validateRoutingRouteRequest`, `validateRoutingInlineProfileContent` | same | Anchor, coordinate, variant, and inline-profile validation. |
| `validateRoutingElevationRequest` | same | Elevation input and point validation. |
| `normalizeRoutingRouteOutput`, `normalizeRoutingCandidate` | same | Keep valid candidates and enforce geometry, metrics, elevation, snapping, and provenance invariants. |
| `decodeRoutingGeometry`, `normalizeRoutingElevationOutput` | same | Canonical polyline and elevation-status normalization. |
| `routingErrorResponse`, `routingJSONError` | same | Stable HTTP error envelope. |

### 12.2 Prepared-profile path

| Function | File | Role |
| --- | --- | --- |
| `PluginSystemRoutingProfilePrepare` | `db/routes/plugin_system_routing_prepare.go` | Authenticated/rate-limited anchorless warm-up endpoint. |
| `prepareRoutingRuntimeProfiles` | same | Prepares participating engines concurrently while retaining soft failures. |
| `prepareRoutingRuntimeProfile` | same | Fingerprints effective inputs and calls the optional capability through the cache. |
| `routingPreparedProfileCache.getOrPrepare` | same | Absolute-TTL cache plus concurrent singleflight with caller-independent work context. |
| `callPreparedRoutingRoutePlugin` | same | Routes with a prepared key and coordinates invalidation, one refresh, and waiting retries. |
| `routingPreparedProfileRetryable` | same | Restricts healing to `unsupported_profile`. |
| `routingPreparedProfileFingerprint` | same | Hashes plugin/instance/auth/config/profile/preference identity without the prepared key. |

### 12.3 Settings, mappings, profiles, and controls

| Function | File | Role |
| --- | --- | --- |
| `InitRoutingDefaults`, `seedRoutingBuiltinSettings` | `db/routes/plugin_system_routing_config.go` | Creates built-in routing defaults. |
| `ResolveRoutingSettings` | same | Merges built-in, admin, and user settings. |
| `ResolveRoutingProfileMappings` | same | Overlays manifest, admin, and user mapping records. |
| `ResolveRoutingProfileMapping` | same | Selects one category/subcategory mapping. |
| `selectRoutingProfileMapping` | same | Applies scope, instance, and subcategory precedence. |
| `applyRoutingCategoryMapping` | same | Materializes profile/mode/config/preferences into a route request. |
| `RoutingProfiles` | same | Combines virtual manifest profiles with stored records. |
| `routingDiscoveryProfiles` | same | Converts manifest native profiles to view DTOs. |
| `builtinRoutingProfileMappings` | same | Converts manifest category mappings to virtual defaults. |
| `ResolveRoutingEffectiveControls` | same | Computes mode and standard-control intersection/defaults. |
| `routingProfileMappingSupportedPreferences` | same | Reads or refreshes per-custom-profile capabilities. |
| `restrictRoutingPreferences` | same | Removes unsupported canonical values before plugin invocation. |
| `ResolveRoutingNativeControls` | same | Returns manifest or introspected advanced groups. |
| `manifestRoutingNativeControlGroups` | same | Selects profile control source. |
| `enrichCustomRoutingProfile` | same | Persists introspected mode/capability metadata. |
| `introspectRoutingProfile` | same | Invokes optional provider introspection. |
| `validateRoutingProfileContentSize` | same | Decodes base64 and enforces generic profile size. |
| `routingProfileInUse` | same | Protects mapped profiles from deletion. |

The public CRUD handler names mirror the API table: `...SettingsGet/Patch`,
`...MappingsGet/Put/Patch`, `...ProfilesGet/Put/Patch/Delete`,
`...EnginesGet`, `...EffectiveControls`, and `...NativeControls`. Route-specific
handlers additionally include `...ProfilePrepare`, `...Route`,
`...RouteCandidates`, `...Check`, and `...Elevation`.

### 12.4 Valhalla

| Function | File | Role |
| --- | --- | --- |
| `routeV1`, `elevationV1` | `plugins/valhalla/main.go` | WASM entry points and structured failures. |
| `handleRoute` | `plugins/valhalla/valhalla.go` | Builds provider route request and normalized candidate. |
| `handleElevation` | same | Builds provider height request. |
| `postJSON` | same | Connector-bound provider POST and response decoding. |
| `costingForMode` | same | Generic-mode fallback to native costing. |
| `valhallaCostingOptions` | same | Standard-to-native preference translation. |
| `segmentsFromValhalla` | same | Converts legs to anchor-pair segments. |
| `snappedAnchors` | same | Restores provider locations to original anchor order. |

### 12.5 BRouter

| Function | File | Role |
| --- | --- | --- |
| `routeV1`, `profileIntrospectV1`, `profilePrepareV1` | `plugins/brouter/main.go` | WASM entry points and structured failures. |
| `handleRoute` | `plugins/brouter/brouter.go` | Profile resolution, provider request, and candidate construction. |
| `handleProfileIntrospect` | same | Content validation, mode, preferences, and native controls. |
| `handleProfilePrepare` | same | Resolves/uploads an effective profile without route anchors. |
| `brouterProfileKey` | same | Chooses prepared, native, generated, or uploaded temporary profile. |
| `validateBRouterProfileContent` | same | BRouter-specific UTF-8, NUL, line, and byte bounds. |
| `renderBRouterGeneratedProfile` | same | Applies template parameter values to an embedded preset. |
| `renderBRouterCustomProfile` | same | Applies only explicitly exposed custom parameters. |
| `parseBRouterProfileParameters` | same | Parses label/description/type annotations. |
| `uploadBRouterProfile` | same | Uploads temporary `.brf` and returns provider profile ID. |
| `requestBRouterRoute` | same | Calls `/brouter` with anchors and profile. |
| `brouterProviderError` | same | Converts HTTP status/body to stable plugin errors. |
| `DetectProfileMode` | `plugins/brouter/core/profile.go` | Parses `validFor*` assignments. |
| `SupportedStandardPreferences` | same | Derives per-profile canonical capability keys. |
| `NativeConfigWithPreferences` | `plugins/brouter/core/preferences.go` | Translates standard values to bounded native parameters. |
| `CandidateFromFeature` | `plugins/brouter/core/core.go` | Converts GeoJSON/property output into the shared route contract. |
| `SegmentsFromCoordinates` | same | Splits route geometry at nearest anchor positions. |
| `SnappedAnchors` | same | Projects requested anchors onto returned geometry. |

### 12.6 Frontend

| Function | File | Role |
| --- | --- | --- |
| `routingApi` and exported CRUD calls | `web/src/lib/stores/routing_store.svelte.ts` | Typed browser API client. |
| `calculateRouteBetween` | same | Route request, candidate selection, elevation fallback, and waypoint construction. |
| `calculateRouteForAnchors`, `calculateRouteVariants` | same | Whole-anchor request construction and explicit variant state. |
| `prepareRoutingProfile` | same | Sends a primary-only or explicit-variant anchorless warm-up request. |
| `routingPreferences`, `routingNativeConfig` | same | Clone the provider-neutral maps populated by effective controls. |
| `insertIntoRoute`, `editRoute`, `deleteFromRoute` | same | Mutates GPX segments. |
| `setRoute`, `undo`, `redo` | same | Route state history. |
| `initRoutingPhaseThreeState` | `web/src/routes/trail/edit/[id]/+page.svelte` | Loads engine/settings state. |
| `selectedRoutingCategory` | same | Resolves canonical names from selected relations. |
| `refreshRoutingEffectiveControls` | same | Loads mode/control contract for the selection. |
| `selectedRoutingVariantEngines`, `routingVariantRequestError` | same | Resolve the active editor engine and gate requests by the public anchor limit. |
| `addAnchorAndRecalculate`, `recalculateRoute` | same | Main incremental drawing/recalculation. |
| `recalculateRouteFromAnchors` | same | Reorders anchors with minimal segment recomputation. |
| `applySnappedAnchors` | same | Synchronizes anchor coordinates and markers. |
| `routeControls`, `setControlValue` | `web/src/lib/components/trail/route_editor.svelte` | Render and update standard/native controls from host metadata. |
| `routingPlanningKey`, `routingPreparationOptions`, `routeCalculationErrorText` | `web/src/lib/util/trail_editor_routing_util.ts` | Isolate routing coordinator snapshots and error presentation from the trail page. |
| `routingProxy` | `web/src/lib/server/routing_proxy.ts` | Shared authenticated SvelteKit proxy for routing endpoints. |
| `saveMapping`, `loadNativeControls` | `web/src/lib/components/settings/plugins/plugin_routing_settings.svelte` | Persists mapping and advanced-control state. |
| `uploadProfile`, `setProfileEnabled`, `deleteProfile` | same | Custom-profile lifecycle. |
| `saveProfile`, `downloadProfile` | `web/src/lib/components/settings/plugins/routing_profile_editor_modal.svelte` | Edits/re-introspects and downloads profile text. |

## 13. Tests and build verification

| Test area | File |
| --- | --- |
| Host contract, limits, normalization, errors | `db/routes/plugin_system_routing_test.go` |
| Route/candidate HTTP orchestration, feature gates, preparation fallbacks, coordinated refresh | `db/routes/plugin_system_routing_handler_test.go` |
| Prepared-profile TTL, singleflight, caller cancellation, failure, and panic cleanup | `db/routes/plugin_system_routing_prepare_test.go` |
| Settings merge, mapping precedence, custom capabilities, profile deletion, native controls | `db/routes/plugin_system_routing_config_test.go` |
| Independent routing type migration, strict schema checks, and rollback cleanup | `db/migrations/installed_plugins_routing_type_test.go` |
| Routing collection schema and profile base64 capacity | `db/migrations/routing_collections_test.go` |
| Connector loopback policy, DNS rebinding, and shared dial deadline | `db/util/network_test.go` |
| BRouter geometry, preferences, bounds, immutability | `plugins/brouter/core/core_test.go` |
| BRouter mode and supported preferences | `plugins/brouter/core/profile_test.go` |
| BRouter alternative/preparation manifest contract and provider error classification | `plugins/brouter/manifest_contract_test.go`, `plugins/brouter/provider_error_test.go` |
| Valhalla alternative manifest/adapter contract | `plugins/valhalla/manifest_contract_test.go` |
| Frontend profile encoding/filenames | `web/src/lib/util/routing_profile_util.test.ts` |
| Editor-language normalization | `web/src/lib/util/code_editor_language.test.ts` |
| Frontend anchor/engine/variant/profile work estimate | `web/src/lib/util/routing_variant_util.test.ts` |

Useful verification commands are:

```bash
cd db
GOCACHE=/tmp/wanderer-go-cache go test ./...

cd plugins/brouter
GOCACHE=/tmp/wanderer-go-cache go test ./...
XDG_CACHE_HOME=/tmp/wanderer-tinygo-cache make build

cd web
npm run check
npm run test:unit -- --run
npm run build
```

The TinyGo build matters in addition to ordinary Go tests because the actual
provider entry points and runtime behavior use WASM build tags.

## 14. Adding another routing plugin

At minimum:

1. Add a direct plugin bundle with `type=routing`, a valid WASM entrypoint, and
   `route.v1` and/or `elevation.v1`.
2. Declare connector targets, allowed paths, response content types, and bounds.
3. Declare `metadata.routing.roles`, modes, native profiles, category mappings,
   and the exact standard preferences the plugin translates.
4. Translate the resolved shared request into provider protocol; do not expose
   the provider API directly to the frontend.
5. Return canonical precision-6 geometry, one segment per adjacent anchor pair,
   finite metrics, and optionally point-aligned elevation and snapped anchors.
6. Return structured plugin errors with stable code and useful message.
7. Put provider-native controls in manifest metadata or implement
   `profile_introspect.v1` when they depend on uploaded content.
8. If profile upload is supported, declare editor/upload metadata and enforce
   provider-specific validation inside the plugin.
9. If rendering/uploading the same effective profile is expensive, implement
   `profile_prepare.v1`, treat `profile.preparedKey` as opaque, and classify a
   rejected prepared key as `unsupported_profile` so the host can heal it.
10. Add category defaults that use canonical Wanderer names and tests for every
   preference translation and geometry invariant.
11. Build with TinyGo and exercise the actual WASM export, not only shared Go
    helpers.

## 15. Current boundaries and planned work

Implemented today:

- one active editor engine for ordinary drawing and host-discovered, bounded
  parallel engines for explicit variant requests;
- independent elevation engine;
- direct category/subcategory-to-profile mappings;
- Valhalla route/elevation, via routing, and native alternatives;
- BRouter route, curated/generated profiles, `.brf` upload, introspection, code
  editing, mode detection, reusable profile preparation, via routing, native
  alternatives, and per-profile standard-option filtering;
- segment and via routing modes, cross-engine segment composition, normalized
  variants, diversity curation, partial failures, and persisted segment
  provenance;
- a curated editor endpoint plus a gated broader candidate-set endpoint;
- primary-engine profile warm-up, capability-aware profile/upload budgeting,
  and coordinated prepared-key invalidation/retry;
- BRouter round-trip generation with bounded distance calibration, host-owned
  synthetic anchors, cyclic editor segments, discovery gating, and persisted
  round-trip provenance;
- provider-neutral maneuver generation for persisted GPX trails, including
  capability-gated engine selection, host-side authorization and validation,
  Valhalla trace matching, and request-local SvelteKit instruction localization.

Still outside the current routing host flow:

- Flutter navigation-screen and offline-cache integration;
- anonymous link-share maneuver generation, live rerouting, and trailhead
  routing;
- GraphHopper or BRouter maneuver adapters and persistent server-side maneuver
  caching.
