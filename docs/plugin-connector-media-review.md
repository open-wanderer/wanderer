# Plugin Connector and Media Fetch Review

## Goal

This document summarizes a proposed change to the Wanderer plugin and media fetch model for external review.

The main objective is to prevent plugins from handing the host arbitrary internal URLs while still supporting:

- self-hosted providers
- provider-authenticated media downloads
- controlled access to private network targets that are explicitly configured by the host admin/user

## Security Invariants

The implementation should preserve the following invariants:

1. Plugin-controlled provider traffic goes through host-controlled execution paths.

- plugins describe requests or media references
- the host resolves and executes outbound traffic
- network policy is enforced by backend code, not by plugin code

2. Private-network reachability is always host-owned.

- plugins do not grant themselves private access
- manifests do not grant private access
- only resolved host-owned connector policy may enable `allowPrivate`

3. Public arbitrary media URLs remain a separate, bounded mode.

- they stay public-only
- they use explicit SSRF protections
- they use bounded downloads

4. Runtime capability changes are security-sensitive.

- the plugin model assumes WASM modules do not get independent outbound network capability
- if the Extism/WASI setup or host-function surface changes, that change must be reviewed against these invariants

## Proposed Changes

1. A plugin must not be allowed to return arbitrary internal URLs.
2. Self-hosted providers should be treated as configured connector targets.
3. Plugin-controlled host requests should not use absolute URLs in the stable ABI.
4. The host should build provider requests itself from a configured base URL plus an allowed relative path.
5. Arbitrary external media URLs may remain supported, but only as an explicit public-only mode.
6. Configured provider URLs may allow private network access, but only within the exact configured base URL scope.

## Why This Change Is Needed

Today there are two different outbound network models:

- Plugin host HTTP requests use plugin manifest policy enforcement.
- Imported media URLs are validated separately during trail import.

This split is workable, but the trust boundary is still too weak for self-hosted/private provider setups.

The current problem is not just "private vs public". The core issue is that plugins can still influence the final request URL too directly.

For a secure self-hosted design, the plugin should describe what resource it wants, while the host should decide:

- which upstream base URL is valid
- whether private network access is allowed
- which path prefixes are allowed
- whether redirects remain inside the same scope
- which auth context is attached

## Current State in the Codebase

Relevant files:

- `db/pluginsystem/policy.go`
- `db/pluginsystem/host_http.go`
- `db/routes/plugin_system_policy.go`
- `db/pluginsystem/protocol.go`
- `plugins/sdk/types.go`
- `db/plugins/importer/importer.go`
- `db/util/network.go`

## Immediate Existing Vulnerabilities

Three relevant issues already exist today and should be prioritized independently of the connector redesign.

In `db/plugins/importer/importer.go`, photo import currently does this:

- resolve and validate the media hostname in `validateRemoteMediaURL(...)`
- then call `filesystem.NewFileFromURL(...)` to fetch the same hostname again

That creates a classic TOCTOU gap for DNS rebinding:

- the SSRF check and the actual fetch do not share the same resolution result
- the second resolution can return a different IP than the first

This is already acknowledged in the current code comments, but it is still present in the implementation.

At the same time, the codebase already contains a mitigation in `db/util/network.go`:

- `SafeHTTPClient()` resolves the hostname
- rejects private/reserved IPs
- dials the first resolved IP directly

That avoids the check-then-re-resolve gap during the actual connection setup.

Because of that, the immediate recommendation is not to invent a new transport later, but to use the existing safe client in the importer now.

There is also a current redirect-policy gap in `db/pluginsystem/host_http.go` and `db/pluginsystem/policy.go`:

- redirects are revalidated through `ValidateHostRequestSpec(...)`
- but `ValidateHostRequestSpec(...)` only enforces hostname-level allow-listing

That means a redirect to a different path on the same allowed host is accepted today, even if that path is outside the intended provider API surface.

Example:

- initial request: `https://same-allowed-host.com/api/v1/...`
- redirect target: `https://same-allowed-host.com/internal/admin`
- current result: accepted, because only the hostname is checked

This is not just a future connector concern. It is already relevant anywhere host requests rely on "allowed host" as a proxy for "allowed URL".

There is also a current port-scope gap in `db/pluginsystem/policy.go`:

- `NetworkURLAllowed(...)` compares `parsedURL.Hostname()`
- `Hostname()` strips the port from the request URL
- configured origins are also reduced to hostname-only matching

That means the current allow-list effectively authorizes all ports on an allowed hostname.

Example:

- intended target: `https://myserver.com:8080/api`
- unintended target: `https://myserver.com:9090/admin`
- current result: accepted, because both requests share the same hostname

This is a fundamental weakness in the current allow-list model, not just a redirect issue.

There is also no response-size limit on the current media download path in `db/plugins/importer/importer.go`:

- plugin media URLs are fetched through a generic file-download path
- unlike `HostRequestSpec`, there is no equivalent to `DownloadPermissions.MaxBytes`
- a plugin can therefore point to a very large file and force the server into an unnecessarily large download

This is primarily a resource-consumption problem rather than a scope-escape problem, but it belongs in the same hardening wave because the fetch path is being rebuilt anyway.

## Immediate Recommendations

Before any connector or ABI redesign, the photo import path should stop using `filesystem.NewFileFromURL(...)` for plugin-supplied media URLs.

Recommended short-term change:

- fetch photo bytes with `util.SafeHTTPClient()`
- keep the current URL syntax validation
- create the file from bytes instead of delegating the fetch to `filesystem.NewFileFromURL(...)`
- introduce a hard or configurable maximum download size for plugin media fetches

In other words:

- `photoFile(...)` should own both validation and fetch
- the fetch must use the same resolution result that passed SSRF enforcement
- the fetch must stop reading once the media size limit is exceeded

This should be treated as an immediate hardening fix, not as part of the later connector redesign.

For plugin host HTTP requests, the host should also stop treating "same hostname" as a sufficient allow-list or redirect policy.

Recommended short-term change:

- tighten both the base allow-list check and redirect validation to an explicit URL scope, not just hostname equality
- include effective port in the authorization decision for both initial requests and redirects
- if path scoping cannot be implemented immediately, consider disabling redirects for plugin-controlled requests until the scope model is in place

At minimum, redirect handling should not silently expand an originally intended API request into a different path space on the same host.

### 1. Plugin host HTTP requests

Current plugin-controlled requests go through:

- `wanderer.http_request`
- `ExecuteHostRequest(...)`
- `ValidateHostRequestSpec(...)`

Current policy behavior:

- validates `http` and `https`
- validates host against manifest static hosts
- optionally allows configured user origins
- validates auth context references
- validates response limits and content types

Main limitation:

- allow-listing is effectively hostname-based
- port is not part of the authorization decision
- it does not strongly bind requests to `scheme + host + port + path scope`

This is too coarse for self-hosted connectors.

### 2. Configured origins

`db/routes/plugin_system_policy.go` currently normalizes configured provider values to a plain origin string.

This is useful, but still too broad for a secure connector model because it does not express:

- allowed path prefix
- whether private access is allowed
- a connector identity

### 3. Imported media

`db/plugins/importer/importer.go` currently supports media like this:

- plugin returns `MediaSource{type:"url", url:"..."}`
- host validates the URL with a public-only SSRF check
- host then performs the remote fetch

Current media behavior is reasonable for arbitrary public URLs, but it is not suitable for authenticated or self-hosted provider media because:

- the plugin still hands out a fully formed URL
- private/internal provider targets are blocked entirely
- there is no host-owned connector resolution step

## Assessment of the Proposed Direction

The proposed direction is correct.

It improves the trust model by moving URL construction and private-network authorization back into the host.

This is the right abstraction:

- plugins return references
- the host resolves references into concrete URLs
- the host enforces connector scope

That is better than trusting a plugin to produce the final absolute URL.

## Recommended Design

## Core Principle

A plugin should not directly control the final request URL for private or configured provider traffic.

Instead:

- plugin output should contain a structured reference
- the host should resolve that reference using trusted configuration

## Connector Model

Self-hosted providers should be represented as connector targets resolved by the host.

Each connector target should define:

- `baseURL`
- `allowPrivate` as a host-side decision
- `allowedPathPrefixes`
- connector identifier or name

Example conceptual shape:

```go
type ResolvedConnectorTarget struct {
    BaseURL             string
    AllowPrivate        bool
    AllowedPathPrefixes []string
}
```

This should replace the current weak concept of user-configured origins as plain strings.

Important trust-boundary rule:

- `AllowPrivate` must not be granted by the plugin manifest
- private network access must be enabled only by host-owned configuration for a concrete connector target

The plugin manifest should not contain any flag that tries to predict whether a deployment will require private network access. That depends on the concrete configured `baseURL`, which is only known to the host/admin.

## Request ABI Proposal

The current host request model uses:

```go
type HostRequestSpec struct {
    Method  string
    URL     string
    Auth    string
    Headers map[string]string
    Body    *HostRequestBody
    Expect  ResponseExpect
}
```

Recommended direction:

```go
type HostRequestSpec struct {
    Method  string            `json:"method"`
    Target  *RequestTarget    `json:"target,omitempty"`
    Auth    string            `json:"auth,omitempty"`
    Headers map[string]string `json:"headers,omitempty"`
    Body    *HostRequestBody  `json:"body,omitempty"`
    Expect  ResponseExpect    `json:"expect,omitempty"`
}

type RequestTarget struct {
    Type      string            `json:"type"` // "connector"
    Connector string            `json:"connector,omitempty"`
    Path      string            `json:"path,omitempty"`
    Query     map[string]string `json:"query,omitempty"`
}
```

Design intent:

- `Target.Type=="connector"` means the host builds the final URL
- `Path` must be relative only
- `Connector` must refer to a configured connector target
- absolute request URLs are not part of the stable host-request ABI

## Media ABI Proposal

Current media source:

```go
type MediaSource struct {
    Type string `json:"type"`
    URL  string `json:"url,omitempty"`
}
```

Recommended direction:

```go
type MediaSource struct {
    Type     string    `json:"type"` // "url" | "connector"
    URL      string    `json:"url,omitempty"` // public external only
    MediaRef *MediaRef `json:"mediaRef,omitempty"`
}

type MediaRef struct {
    Connector string            `json:"connector"`
    AssetID   string            `json:"assetId,omitempty"`
    Path      string            `json:"path,omitempty"`
    Query     map[string]string `json:"query,omitempty"`
}
```

Design intent:

- `Type=="url"` remains for arbitrary public external media
- `Type=="connector"` is for provider-owned or self-hosted media
- `mediaRef` lets the host resolve the final request

### Unresolved point: `assetId` vs `path`

`assetId` must not remain an underspecified alternative to `path`.

By itself, `assetId` is not resolvable. The host needs a deterministic rule that turns plugin output into a concrete URL within connector scope.

That means the design must choose one of these models before implementation:

1. `Path`-only model for connector media

- plugin returns a relative `path`
- host joins `baseURL + normalized path`
- connector scoping is enforced on the resulting path
- `assetId` may exist only as opaque metadata and must not participate in URL resolution

2. Host-resolver model for connector media

- plugin returns `assetId`
- host owns a connector-specific resolver or template that maps `assetId` to a relative path or request shape
- plugin still does not control the final absolute URL

Examples of a host-resolver model:

- connector config contains a trusted relative path template such as `/api/assets/{assetId}/original`
- connector implementation code contains provider-specific resolution logic

Without one of these two models, `assetId` leaves a design gap: the host cannot know how to transform it into a request safely.

Recommended implementation stance:

- prefer the `Path`-only model unless and until a concrete host-owned resolver model exists
- do not treat `assetId` as a supported resolution input until a concrete host-owned resolver model exists
- if `assetId` is kept in the ABI early, document it as metadata only

## Security Model

### Public external media URLs

For arbitrary external media URLs, keep the current policy:

- only `http` and `https`
- host must resolve to public IPs only
- private, loopback, link-local, multicast, and unspecified addresses are blocked
- the actual fetch must reuse the safe resolution path instead of performing a second unconstrained resolution

This remains the correct policy for plugin-provided free-form external URLs.

### Configured provider URLs

For configured provider targets:

- private network access may be allowed
- but only when the host resolves a connector target explicitly marked as trusted

This must be scoped to:

- scheme
- host
- port
- allowed path prefix

Hostname-only matching is not enough.

### Redirects

Redirect validation must use the same connector scope.

A redirect should only be accepted if it remains inside the same authorized target boundary.

At minimum that means:

- same scheme policy
- same host
- same effective port
- same allowed path scope

Otherwise a redirect can become a scope escape.

### URL construction rules

When the host builds a URL from a connector target:

- only relative paths should be accepted
- absolute URLs must be rejected
- `..` path traversal must be rejected
- path normalization must happen before authorization
- query handling should be explicit

### Path prefix enforcement details

`AllowedPathPrefixes` needs stricter definition than "string starts with".

Otherwise the scope check is vulnerable to common boundary and normalization bugs.

The host should define connector path authorization like this:

- prefixes are URL path prefixes, not filesystem paths
- use slash-delimited boundaries
- decode and normalize the request path before the prefix check
- reject any path that changes scope after decoding or cleaning

Concrete issues to guard against:

- `/api/v1` must not also authorize `/api/v1-evil`
- percent-encoded segments such as `/api/%76%31` must be treated as `/api/v1`
- traversal attempts such as `/api/v1/../admin` must be normalized before the scope check

Recommended rules:

- store allowed prefixes in canonical form, for example `/api/v1/`
- canonicalize request paths into the same form before comparison
- require a trailing slash for prefix scopes unless the scope is exactly `/`
- if a connector needs one exact file/resource path rather than a subtree, represent that explicitly instead of overloading prefix matching

Recommended normalization order:

1. Reject absolute URLs in the `Path` field.
2. Parse as URL path, not as filesystem path.
3. Percent-decode path segments.
4. Normalize dot segments using `path.Clean`, not `filepath.Clean`.
5. Re-add a trailing slash if the authorization model is subtree-based.
6. Perform the prefix comparison on the normalized form.

Go-specific guidance:

- use `path.Clean`, not `filepath.Clean`
- `filepath.Clean` is OS-dependent and can introduce incorrect behavior for URL paths, especially on Windows
- authorization should operate on URL semantics only

Example:

- allowed prefix: `/api/v1/`
- accepted: `/api/v1/assets/123`
- rejected: `/api/v1-evil/assets/123`
- rejected: `/api/v1/../admin`
- accepted after decode+normalize: `/api/%76%31/assets/123`

## Recommended Host-Side Policy Shape

Current request policy:

```go
type RequestPolicyContext struct {
    UserConfiguredOrigins []string
}
```

Recommended direction:

```go
type RequestPolicyContext struct {
    Connectors map[string]ResolvedConnectorTarget
}
```

This makes the policy object express actual trust decisions rather than raw user strings.

## Importer Changes

The media importer should no longer rely on direct URL handoff for provider-scoped media.

Recommended behavior:

- `MediaSource.Type=="url"`:
  - keep current public-only SSRF validation
- `MediaSource.Type=="connector"`:
  - resolve via trusted connector target
  - apply connector-scoped auth if needed
  - allow private only for that connector

Important implementation detail:

The importer should not hand a connector-derived or plugin-derived URL to a generic download helper that re-resolves independently without policy awareness.

The host should fetch through a scoped HTTP client and then create the file from bytes. This preserves enforcement of:

- connector scoping
- private/public access policy
- redirect policy
- DNS pinning / anti-rebinding behavior already available in `SafeHTTPClient()`

## Manifest and Config Direction

The current `userConfiguredOrigins` model should be replaced or extended with connector definitions.

Conceptually:

```go
type ConnectorTargetPermission struct {
    Name                string   `json:"name"`
    ConfigKey           string   `json:"configKey"`
    AllowedPathPrefixes []string `json:"allowedPathPrefixes,omitempty"`
}
```

This gives the host enough information to:

- read a base URL from trusted config
- normalize it
- validate it
- scope requests to a safe subset

But the authorization boundary must remain host-owned:

- manifest declares connector structure only
- host configuration decides whether private access is actually allowed
- resolved runtime policy contains the final `AllowPrivate` value

This avoids two failure modes:

- a compromised or malicious plugin granting itself private network reachability via a manifest flag
- a well-meaning plugin author making the wrong guess about whether a specific deployment is public or private

## Concrete Host Config Shape

To make Phase 2 implementable without further concept work, connector trust should use the existing `config.host` namespace and a fixed connector map shape.

Recommended shape:

```json
{
  "plugin": {
    "...": "plugin-owned runtime config"
  },
  "host": {
    "connectors": {
      "<configKey>": {
        "baseURL": "https://photos.example.com:2283",
        "allowPrivate": false
      }
    }
  }
}
```

Notes:

- `plugin` remains plugin-owned runtime configuration
- `host` remains host-owned configuration
- connector entries live under `config.host.connectors`
- `<configKey>` comes from manifest connector metadata
- `baseURL` is the concrete target origin configured for that connector
- `allowPrivate` is the host-owned trust flag for that concrete connector target

Enforcement rules:

- only connector keys declared in manifest metadata may be used
- `baseURL` must be normalized to `scheme://host[:port]`
- path/query/fragment must be stripped from stored connector `baseURL`
- `allowPrivate` must never be writable from untrusted plugin output

Ownership rules:

- normal plugin config fields remain under `config.plugin`
- connector trust fields remain under `config.host.connectors`
- if the existing user-facing plugin settings UI writes `config.host`, it must not expose or accept `allowPrivate` for non-admin flows
- if needed, `baseURL` and `allowPrivate` may be split across different host-owned write paths later, but the runtime shape above should remain the resolved form consumed by backend policy code

## Concrete Connector Transport Shape

To make self-hosted connectors usable, Phase 2 should introduce a dedicated connector transport in addition to `SafeHTTPClient()`.

Recommended API shape:

```go
func ConnectorHTTPClient(target ResolvedConnectorTarget) *http.Client
```

Required behavior:

- resolve the request hostname before dialing
- pin the connection to a resolved IP to avoid DNS rebinding
- preserve the original hostname for TLS verification and SNI
- reject requests whose resolved host/port/path are outside the already validated connector scope
- use the same redirect scope rules as the host request policy

IP policy:

- when `target.AllowPrivate == false`:
  - allow only public IPs
- when `target.AllowPrivate == true`:
  - allow public IPs and private RFC1918 / IPv6 ULA addresses for that connector target
  - continue to reject loopback, link-local, multicast, and unspecified addresses unless there is an explicit future decision to support them

This gives `allowPrivate` a concrete and bounded meaning:

- it enables access to explicitly configured private-network connector targets
- it does not become a blanket bypass for all non-public address classes

Implementation boundary:

- `SafeHTTPClient()` remains the public-only fetch path
- `ConnectorHTTPClient(...)` is the connector-scoped fetch path
- request execution chooses between them based on resolved connector policy, not based on plugin preference

## Implementation Order

No migration window is required here if the ABI is still being finalized and there are no external plugins that depend on the current shape.

Recommended implementation order:

1. Immediately harden media import by replacing `filesystem.NewFileFromURL(...)` with a `SafeHTTPClient()`-based fetch path.
2. Tighten redirect handling for plugin-controlled requests so redirects do not escape the intended URL scope.
3. Decide the connector media resolution model explicitly: `Path`-only, or a host-owned `assetId` resolver.
4. Define the connector request ABI and policy model before treating the plugin interface as stable.
5. Implement connector-based request targets and media references.
6. Implement a connector-scoped transport that can honor `allowPrivate` for explicitly trusted connector targets.
7. Restrict self-hosted/private provider support to connector-based resolution only.
8. Keep public external media URLs as a separate public-only mode where provider-independent asset fetching is still needed.

## Decisions For Implementation

The following points should be treated as the implementation baseline, not as open design questions:

1. Immediate hardening fixes ship first.

- replace `filesystem.NewFileFromURL(...)` in plugin media import with a `SafeHTTPClient()`-based fetch path
- tighten host request allow-listing beyond hostname-only matching
- include effective port in the authorization decision
- add a maximum response size for plugin media downloads

2. Connector targets are host-owned trust objects.

- plugin manifests declare connector structure only
- host configuration provides the concrete `baseURL`
- host configuration decides `allowPrivate`
- runtime policy resolves the final connector target used for enforcement

3. Connector request scoping is `scheme + host + port + normalized path scope`.

- hostname-only authorization is insufficient
- redirect validation must use the same scope model
- path authorization must use normalized URL-path semantics, not raw string prefix checks

4. The first connector media implementation should use the `Path`-only model.

- connector media requests are resolved from `baseURL + normalized relative path`
- `assetId` is not a supported resolution input in the first implementation
- if `assetId` remains in the ABI, it is metadata only

5. Private-network access is never implied by the plugin.

- no manifest flag grants private access
- no plugin output grants private access
- only host-owned connector configuration may enable it

6. Self-hosted connector support requires a dedicated connector-scoped transport.

- `SafeHTTPClient()` is correct for public-only media fetches
- it is not sufficient for connector targets that intentionally allow private IPs
- Phase 2 must introduce a transport/client path that:
  - resolves connector targets within the host-owned scope model
  - permits private IPs only when `allowPrivate` is true for that connector target
  - keeps DNS pinning / anti-rebinding behavior
  - preserves connector scoping across redirects and subsequent requests
- without this transport, `allowPrivate` is only configuration with no usable self-hosted behavior behind it

7. First-party self-hosted connector support must not ship before transport and host config are both concrete.

- the connector-scoped transport is required for actual private-network reachability
- the host-owned admin config shape is required so `allowPrivate` and `baseURL` can be configured intentionally
- together, these are the practical prerequisites for the first usable self-hosted connector, even if they are no longer open architectural questions

8. Stable ABI decision: absolute request URLs are removed, public media URLs remain.

- `HostRequestSpec` should use connector targets, not plugin-supplied absolute URLs
- `MediaSource.Type=="url"` remains supported for public-only external assets
- this keeps one strict model for provider/API traffic while preserving a pragmatic model for CDN-style media assets that are not naturally expressible as connector-relative paths

## Remaining Implementation Notes

The following items still need concrete implementation choices, but they are no longer architectural blockers:

1. Which connector metadata belongs in the manifest, and which belongs exclusively in host-owned configuration.

- manifest-owned data should describe connector structure and request constraints
- host-owned data should describe concrete deployment trust, such as `baseURL` and `allowPrivate`
- this is now a boundary-definition and schema-shaping task, not an open trust-model question

2. Which public media cases should continue to use `MediaSource.Type=="url"` versus a connector media reference.

- public CDN-style or signed asset URLs may still need direct `url` support
- provider-scoped or self-hosted media should move to connector-based references
- this is now a provider-mapping question, not a host-request ABI question

## File-by-File Implementation Checklist

This section turns the design into concrete implementation work items.

### `db/plugins/importer/importer.go`

- [Phase 1] replace `filesystem.NewFileFromURL(...)` in `photoFile(...)`
- [Phase 1] fetch plugin-supplied media through `util.SafeHTTPClient()`
- [Phase 1] create media files from fetched bytes instead of delegating fetches to PocketBase helpers
- [Phase 1] keep URL syntax validation for public external URLs
- [Phase 1] keep `MediaSource.Type=="url"` as public-only behavior
- [Phase 1] enforce a maximum response size for public media downloads
- [Phase 2] extend `photoFile(...)` and related helpers so `MediaSource.Type=="connector"` resolves via host-owned connector policy

### `db/plugins/importer/importer_test.go`

- [Phase 1] add tests that cover the new byte-fetch path instead of `NewFileFromURL(...)`
- [Phase 1] add tests for public URL acceptance and private URL rejection in the importer path
- [Phase 1] add tests that ensure expired media is still skipped
- [Phase 1] add tests that reject oversized media responses
- [Phase 1] if the fetch logic is split into a helper, add focused tests for that helper rather than relying only on `photoFile(...)`
- [Phase 2] add tests for `MediaSource.Type=="connector"` once connector media is introduced

### `db/util/network.go`

- [Phase 1] reuse `SafeHTTPClient()` for importer fetches
- [Phase 1] verify whether `SafeHTTPClient()` needs a small shared helper for "fetch bytes safely" so importer code stays simple
- [Phase 1] if a shared safe fetch helper is added, make it support bounded reads for media downloads
- [Phase 2] add a connector-scoped transport/client path for host-owned connector requests
- [Phase 2] allow private IPs only when `allowPrivate` is true for the resolved connector target
- [Phase 2] allow RFC1918 / IPv6 ULA targets when `allowPrivate=true`, while continuing to reject loopback, link-local, multicast, and unspecified addresses
- [Phase 2] preserve DNS pinning / anti-rebinding behavior for connector traffic
- [Phase 2] preserve hostname-based TLS verification while dialing a pinned IP
- [Phase 2] keep `SafeHTTPClient()` as the public-only path rather than weakening it

### `db/pluginsystem/policy.go`

- [Phase 1] replace hostname-only authorization with URL-scope authorization for existing absolute-URL requests
- [Phase 1] include effective port in the authorization decision
- [Phase 1] stop reducing configured targets to `Hostname()` for enforcement
- [Phase 1] tighten redirect revalidation to the same scope model used for the initial request
- [Phase 2] introduce connector-aware policy structures in place of plain `UserConfiguredOrigins []string`
- [Phase 2] define a normalized path-scope check based on URL semantics
- [Phase 2] use `path.Clean`-based normalization before prefix comparison
- [Phase 2] require canonical slash-delimited prefix boundaries
- [Phase 2] reject scope escapes caused by encoding, traversal, or alternate ports
- [Phase 2] carry `allowPrivate` through the resolved connector policy so request execution can choose the correct transport behavior

### `db/pluginsystem/policy_test.go`

- [Phase 1] replace hostname-only tests with scope-aware tests for existing absolute-URL requests
- [Phase 1] add coverage for port mismatches such as `:8080` vs `:9090`
- [Phase 1] add redirect tests for same-host but different-path / different-port cases if they are implemented through policy helpers
- [Phase 2] add coverage for allowed and denied path prefixes
- [Phase 2] add coverage for encoded-path normalization
- [Phase 2] add coverage for traversal attempts such as `../`
- [Phase 2] add coverage for connector-target resolution once the policy object changes

### `db/pluginsystem/host_http.go`

- [Phase 1] update redirect validation to use the same scope model as the initial request
- [Phase 1] ensure redirects cannot escape to a different port on the same host
- [Phase 1] if needed, reduce or disable redirect support until scope-aware validation is in place
- [Phase 2] ensure redirects cannot escape normalized path scope on the same host
- [Phase 2] resolve connector-based targets instead of relying only on absolute `URL`
- [Phase 2] execute connector requests through the connector-scoped transport required for `allowPrivate`

### `db/pluginsystem/host_http_test.go`

- [Phase 1] add a test that rejects redirects to a different port on the same host
- [Phase 1] keep the existing undeclared-host redirect test
- [Phase 2] add a test that rejects redirects to a different path scope on the same host
- [Phase 2] add tests for connector-target execution once the ABI is added
- [Phase 2] add tests that verify private IPs are still blocked by default for connector traffic
- [Phase 2] add tests that verify private IPs are allowed only for connector targets with `allowPrivate=true`

### `db/routes/plugin_system_policy.go`

- [Phase 2] replace origin-string extraction with host-owned connector target resolution
- [Phase 2] read connector trust config from `config.host.connectors`
- [Phase 2] stop collapsing configured targets to plain origin/hostname semantics
- [Phase 2] construct the runtime `RequestPolicyContext` from resolved connector entries rather than `UserConfiguredOrigins`
- [Phase 2] keep host-only ownership of `allowPrivate`

### `db/routes/plugin_system_config.go`

- [Phase 2] keep the runtime split between `config.plugin` and `config.host`
- [Phase 2] ensure effective config merging preserves the resolved `config.host.connectors` structure
- [Phase 2] if host-owned connector trust fields are sourced differently than ordinary plugin-instance config, centralize that resolution here before policy construction

### `db/pluginsystem/protocol.go`

- [Phase 2] add connector-aware request types to `HostRequestSpec`
- [Phase 2] remove plugin-supplied absolute request URLs from the stable host-request ABI
- [Phase 2] encode the first stable connector-only request shape for provider traffic

### `db/pluginsystem/import_types.go`

- [Phase 2] change the media import ABI types (`MediaSource`, `Photo`, related import structs) for connector-aware media
- [Phase 2] prefer a `Path`-only connector media model for the first cut
- [Phase 2] if `assetId` remains present early, mark it as metadata-only rather than a resolution input

### `plugins/sdk/types.go`

- [Phase 2] mirror the request ABI changes made in `db/pluginsystem/protocol.go`
- [Phase 2] mirror the media import ABI changes made in `db/pluginsystem/import_types.go`
- [Phase 2] expose connector request types to plugin authors
- [Phase 2] keep the first public ABI minimal and explicit
- [Phase 2] avoid publishing an `assetId` resolution contract until the host actually owns a resolver model
- [Phase 2] remove absolute request-URL usage from the stable SDK request API while keeping `MediaSource.Type=="url"` for public-only assets

### `plugins/sdk/host_http.go`

- [Phase 2] update SDK helpers if `HostRequestSpec` gains connector-target support
- [Phase 2] consider adding helper constructors for connector requests so plugin code does not build raw structures repeatedly
- [Phase 2] keep helper behavior aligned with the host ABI exactly
- [Phase 2] remove or de-emphasize helpers that encourage plugin-supplied absolute request URLs

### `db/pluginsystem/manifest.go`

- [Phase 2] validate any new manifest-level connector metadata
- [Phase 2] keep manifest validation limited to connector structure and request constraints
- [Phase 2] do not allow manifest fields to grant private access
- [Phase 2] validate path-prefix declarations into canonical URL-path form if they are stored in manifest metadata

### `db/pluginsystem/manifest_test.go`

- [Phase 2] add tests for any new connector-level manifest metadata
- [Phase 2] add tests that reject manifest attempts to grant private access
- [Phase 2] add tests for path-prefix validation and canonicalization rules if those are enforced at manifest-parse time

### `db/routes/plugin_system_send.go`

- [Phase 2] update send-route request preparation once connector targets are added to the host request ABI
- [Phase 2] ensure upload plans are validated against the new scope-aware policy, including port and path constraints
- [Phase 2] keep auth injection behavior unchanged except where connector target resolution changes the request build step

### `db/routes/plugin_system.go`

- [Phase 2] verify whether plugin listing or plugin metadata responses need to expose connector metadata for the UI
- [Phase 2] keep any host-owned trust fields out of untrusted plugin-defined metadata unless intentionally surfaced

### `web/src/lib/components/settings/plugins/plugin_instance_settings_modal.svelte`

- [Phase 2] if `baseURL` is configured through the existing plugin settings modal, place it under `config.host.connectors.<configKey>.baseURL`
- [Phase 2] do not expose `allowPrivate` in normal user-editable plugin settings flows
- [Phase 2] keep plugin-owned fields under `config.plugin` and connector trust fields under `config.host`

### `plugins/README.md`

- [Phase 2] update the plugin-system flow documentation once connector targets replace origin-style policy
- [Phase 2] document that private access is host-owned
- [Phase 2] document that public media URLs and connector media use different trust models

### `plugins/sdk/README.md`

- [Phase 2] document the first stable connector request ABI exposed to plugin authors
- [Phase 2] document the connector media ABI exposed to plugin authors
- [Phase 2] document which fields are metadata only versus actual resolution inputs
- [Phase 2] document explicitly that request targets are connector-based, while `MediaSource.Type=="url"` remains available only for public external assets

### First-party plugins under `plugins/*`

- [Phase 2] update bundled plugins to the first stable connector/media ABI once it exists
- [Phase 2] prefer connector `Path`-based media references over absolute provider URLs
- [Phase 2] remove assumptions that provider URLs can be emitted directly by the plugin

### Suggested first implementation slice

If the work is split into phases, the first phase should touch only the files needed for immediate hardening:

- `db/plugins/importer/importer.go`
- `db/plugins/importer/importer_test.go`
- `db/util/network.go`
- `db/pluginsystem/policy.go`
- `db/pluginsystem/policy_test.go`
- `db/pluginsystem/host_http.go`
- `db/pluginsystem/host_http_test.go`

Phase 1 should also define one concrete media-download limit policy:

- either a fixed hard limit in code
- or a host-owned configurable limit for plugin media fetches

The important point is that media downloads must not remain unbounded once the fetch path moves to `SafeHTTPClient()`.

The second phase should introduce the connector ABI and runtime policy model:

- `db/pluginsystem/protocol.go`
- `db/pluginsystem/import_types.go`
- `db/util/network.go`
- `plugins/sdk/types.go`
- `plugins/sdk/host_http.go`
- `db/pluginsystem/manifest.go`
- `db/pluginsystem/manifest_test.go`
- `db/routes/plugin_system_policy.go`
- `db/routes/plugin_system_config.go`
- `db/routes/plugin_system_send.go`
- `web/src/lib/components/settings/plugins/plugin_instance_settings_modal.svelte`
- `plugins/README.md`
- `plugins/sdk/README.md`
- first-party plugins under `plugins/*`

## Advantages of the Proposed Model

- clearer trust boundary
- stronger SSRF protection
- better support for self-hosted providers
- host-controlled private network exceptions
- cleaner auth injection model
- more auditable policy decisions
- less reliance on plugin correctness for network safety

## Recommended Conclusion

The proposed direction is ready for implementation with the decisions above.

The critical requirement is to avoid a half-step where private/self-hosted provider traffic still depends on plugin-supplied absolute URLs, hostname-only checks, or generic download helpers that bypass the intended network policy.

The secure version of this feature is:

- plugin returns a reference
- host resolves the reference
- host enforces connector scope
- public arbitrary URLs remain public-only
- private access is allowed only for explicitly configured connector targets

That is the cleanest way to support self-hosted providers without weakening SSRF protections, and it is specific enough to move directly into implementation.
