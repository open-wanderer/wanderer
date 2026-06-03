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

That avoids the importer-style check-then-fetch gap by moving SSRF validation into
the same policy-aware connection path that performs the actual dial.

Because of that, the immediate recommendation is not to keep the generic PocketBase download path, but to move plugin media imports to a policy-aware helper now. Phase 1 should implement that helper as `FetchPublicURL(...)` using `safeurl`.

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

Phase 1 should bound both per-file media size and aggregate media import volume. The aggregate limit should be generic across plugin media imports rather than tied to trails or photos specifically.

## Immediate Recommendations

Before any connector or ABI redesign, the photo import path should stop using `filesystem.NewFileFromURL(...)` for plugin-supplied media URLs.

Recommended short-term change:

- fetch photo bytes with `util.FetchPublicURL(...)`
- keep the current URL syntax validation
- create the file from bytes instead of delegating the fetch to `filesystem.NewFileFromURL(...)`
- introduce a hard or configurable maximum download size for plugin media fetches

In other words:

- `photoFile(...)` should own both validation and fetch
- the fetch must perform SSRF validation and dialing in one policy-aware transport path
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
- `basePath`
- `allowPrivate` as a host-side decision
- `allowedPathPrefixes`
- connector identifier or name

Example conceptual shape:

```go
type ResolvedConnectorTarget struct {
    BaseURL             string
    BasePath            string
    AllowPrivate        bool
    TLS                 ConnectorTLSConfig
    StorageOrigins      map[string]ResolvedConnectorOrigin
    AllowedPathPrefixes []string
}

type ConnectorTLSConfig struct {
    Mode     string   // "system" | "customCA"
    CABundle []byte
}

type ResolvedConnectorOrigin struct {
    Name         string
    BaseURL      string
    BasePath     string
    AllowPrivate bool
    TLS          ConnectorTLSConfig
}
```

This should replace the current weak concept of user-configured origins as plain strings.

Important trust-boundary rule:

- `AllowPrivate` must not be granted by the plugin manifest
- private network access must be enabled only by host-owned configuration for a concrete connector target

The plugin manifest should not contain any flag that tries to predict whether a deployment will require private network access. That depends on the concrete configured `baseURL`, which is only known to the host/admin.

## Immich Compatibility Baseline

The connector design must support a self-hosted Immich asset plugin. This is a concrete compatibility target, not a later stretch goal.

Immich requires all of the following:

- authenticated media downloads such as `GET /api/assets/{id}/original` with an API key header
- an API path under `/api`
- optional deployment under an additional reverse-proxy base path such as `/immich`
- private LAN targets such as `http://192.168.1.50:2283`
- HTTPS with either public certificates, a private CA, or self-signed certificates
- optional redirects to configured object-storage origins such as S3 or MinIO

Therefore the connector model must include these baseline capabilities before first-party Immich support can ship:

1. Connector media auth

- `MediaRef` carries an auth context reference
- the media importer injects auth before executing connector media requests
- auth injection is shared with `HostRequestSpec` rather than duplicated as ad hoc header code

2. Mandatory base-path support

- connector targets must support `basePath`
- `/api` path scoping and reverse-proxy paths must be represented explicitly
- an origin-only first cut that rejects non-root connector paths is not Immich-compatible

3. Host-owned TLS policy

- connector targets must define TLS behavior
- default mode is normal system trust verification
- admin-owned `customCA` mode allows a connector-specific CA bundle
- certificate verification must not be disabled in production connector traffic
- plugin manifests and plugin output must not enable TLS trust changes

4. Declared storage redirect origins

- default connector redirects remain same-origin and same-scope
- providers that legitimately redirect asset downloads to object storage must declare storage-origin capability in the manifest
- the host/admin must configure and trust concrete storage origins
- redirects to those origins must be allowed only for media downloads and only after the original connector request has been authorized

If any of these four capabilities is absent, an Immich connector may still list or inspect assets through host requests, but it cannot reliably import original authenticated media.

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
    Query     []QueryParam      `json:"query,omitempty"`
}

type QueryParam struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}
```

Design intent:

- `Target.Type=="connector"` means the host builds the final URL
- `Path` must be relative only
- `Connector` must refer to a resolved connector target, either manifest-fixed `public_api` or host-configured
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
    Auth      string            `json:"auth,omitempty"`
    AssetID   string            `json:"assetId,omitempty"`
    Path      string            `json:"path,omitempty"`
    Query     []QueryParam      `json:"query,omitempty"`
}
```

Design intent:

- `Type=="url"` remains for arbitrary public external media
- `Type=="connector"` is for provider-owned or self-hosted media
- `mediaRef` lets the host resolve the final request
- `mediaRef.auth` names a manifest-declared auth context that the host injects during connector media fetches

Connector media auth is required for providers such as Immich, where the original asset download is a media fetch but still requires provider authentication.

Implementation rule:

- connector media fetches must run through the same auth-injection subsystem used by `HostRequestSpec`
- the auth injector should be refactored around a small request-mutation primitive rather than being coupled only to `HostRequestSpec`
- `MediaRef.Auth` must be validated against `manifest.Auth.Contexts` and `manifest.Permissions.Auth`
- plugins must not provide raw auth headers for connector media when an auth context is expected
- the importer must receive enough plugin/instance/auth context to resolve and inject media auth before executing the connector fetch

### Unresolved point: `assetId` vs `path`

`assetId` must not remain an underspecified alternative to `path`.

By itself, `assetId` is not resolvable. The host needs a deterministic rule that turns plugin output into a concrete URL within connector scope.

That means the design must choose one of these models before implementation:

1. `Path`-only model for connector media

- plugin returns a relative `path`
- host joins `baseURL + basePath + normalized path`
- connector scoping is enforced on the resulting path, including the connector base path when configured
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

- same scheme for provider-authenticated requests
- no `https` to `http` downgrade
- same host
- same effective port
- same allowed path scope

Otherwise a redirect can become a scope escape.

Exception for connector media:

- a connector may allow media redirects to named storage origins
- this is allowed only if the manifest declares storage redirect support and host configuration provides the concrete storage origin
- the redirect target must match the configured storage origin scope, port, TLS policy, and IP policy
- connector auth headers and auth query parameters must be stripped before following a redirect to a storage origin
- storage origins do not have their own auth context in this model
- storage redirects should rely on provider-generated presigned URLs or otherwise unauthenticated storage URLs
- this exception must not apply to ordinary provider API requests by default

### URL construction rules

When the host builds a URL from a connector target:

- only relative paths should be accepted
- absolute URLs must be rejected
- `..` path traversal must be rejected
- path normalization must happen before authorization
- query handling should be explicit
- encoded slash or backslash characters must not be allowed to change path segment boundaries
- invalid percent escapes, NUL bytes, control characters, and ambiguous double-encoded paths should be rejected rather than normalized leniently

### Query handling rules

Query parameters are part of request construction, but they are not part of path scope authorization.

Rules:

- query values must be represented as an ordered list of name/value pairs, not `map[string]string`
- repeated keys are allowed
- ordering is preserved
- names and values are percent-encoded by the host
- plugins must provide decoded names and values, not raw query strings
- query names must reject NUL/control characters
- query values must reject NUL/control characters
- query parameters must not be used to override connector identity, auth context, path, base path, scheme, host, or port
- auth injection runs after plugin query construction and may add auth query parameters when the auth context uses query placement
- if auth injection would collide with plugin-provided query names, the auth injector owns the final auth parameter value

This is a deliberate stable-ABI choice. It avoids raw query-string smuggling and still supports providers that need repeated keys or stable query ordering.

### Path prefix enforcement details

`AllowedPathPrefixes` needs stricter definition than "string starts with".

Otherwise the scope check is vulnerable to common boundary and normalization bugs.

The host should define connector path authorization like this:

- prefixes are URL path prefixes, not filesystem paths
- use slash-delimited boundaries
- decode and normalize the request path before the prefix check
- reject any path that changes scope after decoding or cleaning
- reject paths whose encoded representation would change segment boundaries after a second decode

Concrete issues to guard against:

- `/api/v1` must not also authorize `/api/v1-evil`
- percent-encoded segments such as `/api/%76%31` must be treated as `/api/v1`
- traversal attempts such as `/api/v1/../admin` must be normalized before the scope check
- encoded separators such as `%2f` and `%5c` must not smuggle additional path segments
- double-encoded traversal such as `%252e%252e` must not become valid after a later decoding step

Recommended rules:

- store allowed prefixes in canonical form, for example `/api/v1/`
- canonicalize request paths into the same form before comparison
- require a trailing slash for prefix scopes unless the scope is exactly `/`
- if a connector needs one exact file/resource path rather than a subtree, represent that explicitly instead of overloading prefix matching

Recommended normalization order:

1. Reject absolute URLs in the `Path` field.
2. Parse as URL path, not as filesystem path.
3. Reject malformed escapes, NUL/control characters, and encoded `/` or `\` if they could change segment boundaries.
4. Percent-decode path segments exactly once.
5. Reject any remaining percent-encoded sequence that would become `/`, `\`, `.`, or `..` after a second decode.
6. Normalize dot segments using `path.Clean`, not `filepath.Clean`.
7. Re-add a trailing slash if the authorization model is subtree-based.
8. Perform the prefix comparison on the normalized form.

Go-specific guidance:

- use `path.Clean`, not `filepath.Clean`
- `filepath.Clean` is OS-dependent and can introduce incorrect behavior for URL paths, especially on Windows
- authorization should operate on URL semantics only

Example:

- allowed prefix: `/api/v1/`
- accepted: `/api/v1/assets/123`
- rejected: `/api/v1-evil/assets/123`
- rejected: `/api/v1/../admin`
- rejected: `/api/v1%2fadmin` if encoded separators are not explicitly supported
- rejected: `/api/%252e%252e/admin`
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
  - validate `MediaRef.Auth` when present
  - inject connector media auth before the request is executed
  - allow private only for that connector
  - apply connector-scoped TLS, redirect, and storage-origin policy

Important implementation detail:

The importer should not hand a connector-derived or plugin-derived URL to a generic download helper that re-resolves independently without policy awareness.

The host should fetch through a scoped HTTP client and then create the file from bytes. This preserves enforcement of:

- connector scoping
- private/public access policy
- redirect policy
- DNS pinning / anti-rebinding behavior provided by `FetchPublicURL(...)` for public media and by `ConnectorHTTPClient(...)` for connector media

## Manifest and Config Direction

The current `userConfiguredOrigins` model should be replaced or extended with connector definitions.

Conceptually:

```go
type ConnectorTargetPermission struct {
    Name                string   `json:"name"`
    Type                string   `json:"type"` // "public_api" | "configured"
    FixedBaseURL        string   `json:"fixedBaseURL,omitempty"`
    SupportsMediaAuth   bool     `json:"supportsMediaAuth,omitempty"`
    SupportsStorageRedirects bool `json:"supportsStorageRedirects,omitempty"`
    SupportsCustomTLS   bool     `json:"supportsCustomTLS,omitempty"`
    ConfigKey           string   `json:"configKey,omitempty"`
    AllowedPathPrefixes []string `json:"allowedPathPrefixes,omitempty"`
}
```

This gives the host enough information to:

- read a base URL from trusted config
- normalize it
- validate it
- scope requests to a safe subset

But the authorization boundary must remain host-owned:

- manifest declares connector structure and, for fixed public APIs, the public provider origin
- host configuration decides whether private access is actually allowed
- host configuration provides concrete origins for configured self-hosted connectors
- resolved runtime policy contains the final `AllowPrivate` value

This avoids two failure modes:

- a compromised or malicious plugin granting itself private network reachability via a manifest flag
- a well-meaning plugin author making the wrong guess about whether a specific deployment is public or private

Connector target types:

1. `public_api`

- for fixed public provider APIs such as Strava, Komoot, or Hammerhead
- manifest owns `fixedBaseURL`
- `configKey` must be empty or ignored
- private access is never allowed
- custom TLS and storage redirect origins are not allowed unless a future provider explicitly needs them
- this replaces manifest static hosts without requiring admin-configured base URLs for public SaaS APIs

2. `configured`

- for self-hosted or deployment-specific targets such as Immich
- `configKey` is required
- host/admin owns `baseURL`, `basePath`, `allowPrivate`, TLS mode, and storage origins
- manifest may define path constraints and supported capabilities, but not concrete trust decisions

This distinction is required if absolute request URLs are removed from the stable ABI. Otherwise fixed public API plugins would have no clean way to name their provider API without contradicting the rule that self-hosted `baseURL` values are host-owned.

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
        "basePath": "/",
        "allowPrivate": false,
        "tls": {
          "mode": "system"
        },
        "storageOrigins": {
          "minio": {
            "baseURL": "https://storage.example.com",
            "basePath": "/",
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

Notes:

- `plugin` remains plugin-owned runtime configuration
- `host` remains host-owned configuration
- connector entries live under `config.host.connectors`
- `<configKey>` comes from manifest connector metadata
- `baseURL` is the concrete configured endpoint for that connector
- `basePath` is the canonical root path for that connector
- `allowPrivate` is the host-owned trust flag for that concrete connector target
- `tls` is the host-owned TLS policy for that connector target
- `storageOrigins` contains optional host-owned redirect targets for provider media storage

Enforcement rules:

- only connector keys declared in manifest metadata may be used
- `baseURL` must be parsed and normalized into `scheme + host + effective port`
- configured paths must be stored separately as canonical `basePath` and included in every scope decision
- query and fragment must be rejected for connector base configuration
- `allowPrivate` must never be writable from untrusted plugin output
- TLS trust settings must never be writable from untrusted plugin output
- storage origins must never be inferred from plugin output or from a redirect target alone

Important base-path decision:

Silently converting `https://example.com/immich` into `https://example.com` is too broad for common reverse-proxy deployments. It can turn a connector intended for one mounted application into access to the whole origin. Therefore the implementation must use the origin plus base path model:

- split the configured value into origin plus canonical `basePath`
- join plugin-provided relative paths under `basePath`
- enforce `basePath + allowedPathPrefixes` after URL-path normalization
- reject redirects that leave the base path even if they stay on the same origin

This is required for Immich-compatible self-hosted deployments behind a shared reverse proxy.

TLS policy:

- `tls.mode=="system"` uses normal system trust and hostname verification
- `tls.mode=="customCA"` uses a host/admin-provided CA bundle in addition to or instead of system trust
- `tls.mode=="insecure"` is not supported; connector transport must not disable certificate or hostname verification
- plugin manifests may declare that a connector supports custom TLS configuration, but they must not select the mode
- `customCA` does not fix hostname mismatch; `https://192.168.1.50:2283` still requires a certificate with a matching IP subject alternative name
- self-signed certificates with only a CN or the wrong hostname will still fail under `customCA`; the practical fix is issuing a certificate with a matching subject alternative name and trusting its CA

Storage-origin policy:

- storage origins are named host-owned connector subtargets
- storage origins have the same shape as connector targets: origin, `basePath`, `allowPrivate`, and TLS policy
- redirects to storage origins are allowed only when the manifest declares that the connector may redirect media downloads and host config provides a matching storage origin
- storage-origin redirects must not be allowed for arbitrary API requests unless explicitly required by a future provider

Ownership rules:

- normal plugin config fields remain under `config.plugin`
- connector trust fields remain under `config.host.connectors`
- if the existing user-facing plugin settings UI writes `config.host`, it must not expose or accept `allowPrivate` for non-admin flows
- if needed, `baseURL` and `allowPrivate` may be split across different host-owned write paths later, but the runtime shape above should remain the resolved form consumed by backend policy code

## Concrete Connector Transport Shape

To make self-hosted connectors usable, Phase 2 should introduce a dedicated connector transport in addition to the public-only `FetchPublicURL(...)` helper.

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
  - allow public IPs and explicitly selected private RFC1918 / IPv6 ULA addresses for that connector target
  - continue to reject loopback, link-local, multicast, and unspecified addresses unless there is an explicit future decision to support them
  - explicitly decide whether less obvious ranges such as CGNAT `100.64.0.0/10`, documentation networks, benchmark networks, IPv4-mapped IPv6 addresses, and other IANA special-purpose ranges are allowed or blocked
  - prefer `net/netip` prefix checks over scattered `net.IP` helper calls so the range policy is auditable in one place

This gives `allowPrivate` a concrete and bounded meaning:

- it enables access to explicitly configured private-network connector targets
- it does not become a blanket bypass for all non-public address classes

Implementation boundary:

- `FetchPublicURL(...)` is the public-only plugin media fetch path
- existing `SafeHTTPClient()` users outside plugin media may remain as-is unless they are intentionally migrated
- `ConnectorHTTPClient(...)` is the connector-scoped fetch path
- request execution chooses between them based on resolved connector policy, not based on plugin preference

## Concrete Transport Decision

Use `github.com/doyensec/safeurl` as the Phase 1 public-only SSRF transport primitive, behind a Wanderer-owned helper API.

This is no longer an open-ended research item. The implementation should start with `safeurl` and keep a narrow wrapper boundary so the dependency can be replaced later if tests uncover a mismatch.

Dependency decision:

- add `github.com/doyensec/safeurl` to `db/go.mod`
- pin the exact version used during implementation because the module is still pre-v1
- at the time of this review, pkg.go.dev lists [`github.com/doyensec/safeurl`](https://pkg.go.dev/github.com/doyensec/safeurl) as pre-v1; the implementation should pin the latest reviewed version rather than tracking floating `latest`
- do not expose `safeurl` types outside `db/util`

Source notes:

- The [`safeurl` README](https://github.com/doyensec/safeurl) describes it as an SSRF protection wrapper around Go's HTTP client with DNS rebinding protection and allow/block lists.
- The current implementation validates schemes, hosts, ports, credentials, and IP/CIDR allow/block lists, and its default private-network block list includes RFC1918, loopback, link-local, multicast, documentation ranges, benchmarking ranges, CGNAT, IPv6 ULA, IPv6 link-local, and other special-purpose ranges.
- It does not model Wanderer's connector identity, `basePath`, normalized URL-path scope, auth context, or media size policy. Those remain Wanderer-owned.

### Phase 1 Public Fetch Helper

Add a public-only helper in `db/util/network.go` or a small adjacent file such as `db/util/safe_fetch.go`.

Recommended API:

```go
const DefaultPluginMediaMaxBytes int64 = 50 << 20

type SafeFetchResult struct {
    Body        []byte
    ContentType string
    FinalURL    string
}

func FetchPublicURL(ctx context.Context, rawURL string, maxBytes int64) (*SafeFetchResult, error)
```

Required behavior:

- accept only `http` and `https`
- reject URLs with embedded credentials
- use `safeurl.Client(...)` rather than `http.Client` directly
- enable IPv6 so public IPv6 URLs continue to work
- use a timeout no larger than the current plugin host HTTP timeout
- default `maxBytes` to `DefaultPluginMediaMaxBytes` when the caller passes `0`
- read with `io.LimitReader(resp.Body, maxBytes+1)`
- fail if the response exceeds `maxBytes`
- return the response content type and final URL for logging and filename/content-type handling
- close the response body in the helper

Initial public URL policy:

```go
config := safeurl.GetConfigBuilder().
    SetTimeout(60 * time.Second).
    SetAllowedSchemes("http", "https").
    SetAllowedPorts(80, 443).
    EnableIPv6(true).
    AllowSendingCredentials(false).
    SetCheckRedirect(publicMediaRedirectPolicy).
    Build()
```

Public media URLs on non-standard ports should be rejected in Phase 1. A provider that requires a non-standard port should use the connector model, where the host admin explicitly configures the connector target and port.

Redirect policy for public media:

- reject redirects with embedded credentials
- reject schemes other than `http` and `https`
- reject `https` to `http` downgrades
- rely on `safeurl` for IP and port checks during each actual dial
- cap redirects through Go's default redirect limit unless a stricter limit is added

Importer integration:

- `photoFile(...)` keeps `MediaSource.Type=="url"` as the only supported Phase 1 media source
- `photoFile(...)` calls `util.FetchPublicURL(ctx, photo.Source.URL, util.DefaultPluginMediaMaxBytes)`
- `photoFile(...)` creates the PocketBase file with `filesystem.NewFileFromBytes(...)`
- filenames should come from the provider filename, URL path basename, or content type, using the existing `safeMediaFileName(...)` helper
- `validateRemoteMediaURL(...)` can be removed or reduced to syntax-only validation once `FetchPublicURL(...)` owns network safety

Fallback rule:

- If `safeurl` fails a required Wanderer test and cannot be configured to pass without weakening policy, keep the `FetchPublicURL(...)` API and replace only its internals with a custom `net/http.Transport`.
- Do not postpone Phase 1 on an abstract library review; the decision point is the concrete test suite below.

### Phase 1 Required Tests

Add focused tests for the new helper, preferably in `db/util/network_test.go`.

Required acceptance tests:

- public `https://` URL on port 443 succeeds
- public `http://` URL on port 80 succeeds
- response exactly at `DefaultPluginMediaMaxBytes` succeeds
- response larger than the limit fails
- content type and final URL are returned

Required rejection tests:

- `ftp://...`
- URL with username or password
- `http://127.0.0.1/...`
- `http://localhost/...`
- `http://10.0.0.1/...`
- `http://169.254.169.254/...`
- `http://[::1]/...`
- `http://[fc00::1]/...`
- public host on a non-standard port such as `:8080`
- redirect from `https` to `http`
- redirect to a blocked private/special-purpose target
- oversized response with misleading or absent `Content-Length`

Importer tests should then verify behavior at the importer boundary:

- a valid public URL produces a PocketBase file from bytes
- private/local URLs are skipped or rejected through the new helper
- expired media is still skipped before fetch
- oversized media is skipped or rejected without creating a file

### Phase 2 Connector Transport

Use `safeurl` for the connector transport only for scheme, host, port, DNS-rebinding, and IP/CIDR enforcement. Keep connector path authorization outside `safeurl`.

Recommended API:

```go
type ConnectorIPPolicy string

const (
    ConnectorPublicOnly ConnectorIPPolicy = "publicOnly"
    ConnectorAllowPrivate ConnectorIPPolicy = "allowPrivate"
)

func ConnectorHTTPClient(target ResolvedConnectorTarget) *http.Client
```

Implementation rule:

- build the concrete request URL only after connector policy resolution
- validate `scheme + host + effective port + basePath + path scope` in Wanderer code before calling the HTTP client
- configure `safeurl` with the resolved connector host and effective port
- when `AllowPrivate == false`, use safeurl's default private/special-purpose blocking
- when `AllowPrivate == true`, configure an explicit allow list for the connector target's resolved host/IP ranges, not a blanket private-network allow
- continue to reject loopback, link-local, multicast, unspecified, documentation, benchmarking, CGNAT, and other special-purpose ranges unless a specific future connector type deliberately supports one of them
- apply Wanderer's redirect path-scope validation in `CheckRedirect`

This means `safeurl` handles the low-level network guard, while Wanderer remains responsible for the security model.

## Implementation Order

This ABI change is breaking for current first-party plugins that use absolute `HostRequestSpec.URL` values. No public external plugin migration window is required if no third-party plugins depend on the current unstable shape, but bundled plugins must be migrated in the same implementation wave.

Recommended implementation order:

1. Immediately harden media import by replacing `filesystem.NewFileFromURL(...)` with the `safeurl`-backed `FetchPublicURL(...)` path.
2. Tighten redirect handling for plugin-controlled requests so redirects do not escape the intended URL scope.
3. Implement connector target resolution for both manifest-fixed `public_api` connectors and host-configured connectors.
4. Migrate Strava, Komoot, and Hammerhead API calls to manifest-fixed `public_api` connector targets.
5. Implement mandatory connector `basePath` handling.
6. Refactor auth injection so connector media can use `MediaRef.Auth`.
7. Implement connector-scoped transport with `allowPrivate`, TLS policy, and storage-origin redirect support.
8. Implement connector-based request targets and media references.
9. Use the `Path`-only connector media model for the first stable media ABI.
10. Use Immich as the first self-hosted connector acceptance case before treating the connector ABI as complete.
11. Restrict self-hosted/private provider support to connector-based resolution only.
12. Keep public external media URLs as a separate public-only mode where provider-independent asset fetching is still needed.

## Decisions For Implementation

The following points should be treated as the implementation baseline, not as open design questions:

1. Immediate hardening fixes ship first.

- replace `filesystem.NewFileFromURL(...)` in plugin media import with the `safeurl`-backed `FetchPublicURL(...)` path
- tighten host request allow-listing beyond hostname-only matching
- include effective port in the authorization decision
- add a maximum response size for plugin media downloads

2. Connector targets are host-owned trust objects.

- plugin manifests declare connector structure only
- public API connector manifests may provide fixed public `baseURL` values
- host configuration provides the concrete `baseURL` for configured self-hosted connectors
- connector paths are stored explicitly as canonical `basePath`
- host configuration decides `allowPrivate`
- host configuration decides TLS mode and storage redirect origins
- runtime policy resolves the final connector target used for enforcement

3. Connector request scoping is `scheme + host + port + basePath + normalized path scope`.

- hostname-only authorization is insufficient
- redirect validation must use the same scope model
- path authorization must use normalized URL-path semantics, not raw string prefix checks
- configured base paths must never be silently stripped

4. The first connector media implementation should use the `Path`-only model.

- connector media requests are resolved from `baseURL + basePath + normalized relative path`
- `assetId` is not a supported resolution input in the first implementation
- if `assetId` remains in the ABI, it is metadata only

5. Private-network access is never implied by the plugin.

- no manifest flag grants private access
- no plugin output grants private access
- only host-owned connector configuration may enable it

6. Self-hosted connector support requires a dedicated connector-scoped transport.

- `FetchPublicURL(...)` is correct for public-only plugin media fetches
- it is not sufficient for connector targets that intentionally allow private IPs
- Phase 2 must introduce a transport/client path that:
  - resolves connector targets within the host-owned scope model
  - permits private IPs only when `allowPrivate` is true for that connector target
  - keeps DNS pinning / anti-rebinding behavior
  - preserves connector scoping across redirects and subsequent requests
- a reviewed library such as `github.com/doyensec/safeurl` may provide part of this transport, but Wanderer still owns connector scope and path authorization
- without this transport, `allowPrivate` is only configuration with no usable self-hosted behavior behind it

7. First-party self-hosted connector support must not ship before transport and host config are both concrete.

- the connector-scoped transport is required for actual private-network reachability
- the host-owned admin config shape is required so `allowPrivate`, `baseURL`, and any `basePath` can be configured intentionally
- together, these are the practical prerequisites for the first usable self-hosted connector, even if they are no longer open architectural questions

8. Stable ABI decision: plugin-supplied absolute request URLs are removed, public media URLs remain.

- `HostRequestSpec` should use connector targets, not plugin-supplied absolute URLs
- fixed public SaaS APIs should use manifest-fixed `public_api` connector targets
- self-hosted providers should use host-configured connector targets
- `MediaSource.Type=="url"` remains supported for public-only external assets
- this keeps one strict model for provider/API traffic while preserving a pragmatic model for CDN-style media assets that are not naturally expressible as connector-relative paths

## Concrete Schema Decisions

The connector schema should use the following ownership split.

Manifest-owned connector fields:

- connector name
- connector type: `public_api` or `configured`
- `fixedBaseURL` only for `public_api`, normalized into fixed origin plus canonical `basePath`
- allowed path prefixes
- auth context names that may be used by connector requests or connector media
- whether connector media auth is supported
- whether storage redirects are supported
- whether custom TLS configuration is supported

Host-owned connector fields:

- `baseURL` for `configured` connectors
- canonical `basePath`
- `allowPrivate`
- TLS mode and CA bundle
- named storage origins
- storage-origin `allowPrivate`, `basePath`, and TLS settings

Plugin-output fields:

- connector name
- relative path
- query values
- auth context reference
- optional metadata such as `assetId`

Public media rule:

- public CDN-style or signed asset URLs may continue to use `MediaSource.Type=="url"`
- provider-scoped or self-hosted media must use connector media references
- Immich original asset downloads are connector media, not public URLs

## File-by-File Implementation Checklist

This section turns the design into concrete implementation work items.

### `db/plugins/importer/importer.go`

- [Phase 1] replace `filesystem.NewFileFromURL(...)` in `photoFile(...)`
- [Phase 1] fetch plugin-supplied media through `util.FetchPublicURL(...)`
- [Phase 1] route importer downloads through the Wanderer-owned helper rather than using `safeurl` directly at call sites
- [Phase 1] create media files from fetched bytes instead of delegating fetches to PocketBase helpers
- [Phase 1] keep URL syntax validation for public external URLs
- [Phase 1] keep `MediaSource.Type=="url"` as public-only behavior
- [Phase 1] enforce a maximum response size for public media downloads
- [Phase 1] enforce `util.DefaultPluginMaxImportMediaItems` across imported media items
- [Phase 1] if aggregate byte accounting is implemented, enforce `util.DefaultPluginMaxImportMediaBytes`
- [Phase 1] log skipped media counts when the aggregate media item limit is exceeded
- [Phase 2] extend `photoFile(...)` and related helpers so `MediaSource.Type=="connector"` resolves via host-owned connector policy
- [Phase 2] pass plugin, instance, and auth material into the connector media fetch path so `MediaRef.Auth` can be injected
- [Phase 2] execute connector media fetches through `ConnectorHTTPClient(...)`, not `FetchPublicURL(...)`

### `db/plugins/importer/importer_test.go`

- [Phase 1] add tests that cover the new byte-fetch path instead of `NewFileFromURL(...)`
- [Phase 1] add tests for public URL acceptance and private URL rejection in the importer path
- [Phase 1] add tests that ensure expired media is still skipped
- [Phase 1] add tests that reject oversized media responses
- [Phase 1] add tests that media items beyond `DefaultPluginMaxImportMediaItems` are skipped or rejected
- [Phase 1] if aggregate byte accounting is implemented, add tests that the importer stops before exceeding `DefaultPluginMaxImportMediaBytes`
- [Phase 1] keep importer tests focused on importer behavior; low-level SSRF bypass cases belong in `db/util` helper tests
- [Phase 2] add tests for `MediaSource.Type=="connector"` once connector media is introduced
- [Phase 2] add an Immich-style test that fetches `/api/assets/{id}/original` with an injected `x-api-key`
- [Phase 2] add tests that connector media uses `basePath` plus media path correctly
- [Phase 2] add tests that connector media rejects missing, unknown, or unauthorized `MediaRef.Auth`

### `db/util/network.go`

- [Phase 1] add `FetchPublicURL(ctx, rawURL, maxBytes)` as the single public-only safe fetch helper
- [Phase 1] add `DefaultPluginMaxImportMediaItems`
- [Phase 1] optionally add `DefaultPluginMaxImportMediaBytes`
- [Phase 1] implement `FetchPublicURL(...)` with `github.com/doyensec/safeurl`
- [Phase 1] pin the exact `safeurl` version in `db/go.mod`
- [Phase 1] configure public fetches for `http`, `https`, ports `80` and `443`, no embedded credentials, IPv6 enabled, and bounded reads
- [Phase 1] make `FetchPublicURL(...)` return bytes, content type, and final URL
- [Phase 1] make `FetchPublicURL(...)` reject redirects from `https` to `http`
- [Phase 1] keep the helper API small enough to replace the underlying transport later if required tests fail
- [Phase 2] add a connector-scoped transport/client path for host-owned connector requests
- [Phase 2] support connector TLS modes `system` and `customCA`; reject `insecure`
- [Phase 2] ensure TLS settings are scoped to the concrete connector target or storage origin
- [Phase 2] allow private IPs only when `allowPrivate` is true for the resolved connector target
- [Phase 2] allow RFC1918 / IPv6 ULA targets when `allowPrivate=true`, while continuing to reject loopback, link-local, multicast, and unspecified addresses
- [Phase 2] decide the complete special-purpose IP range policy, including CGNAT, documentation ranges, benchmark ranges, IPv4-mapped IPv6, and other IANA special-purpose ranges
- [Phase 2] prefer a centralized `net/netip` prefix table for IP classification
- [Phase 2] preserve DNS pinning / anti-rebinding behavior for connector traffic
- [Phase 2] preserve hostname-based TLS verification while dialing a pinned IP
- [Phase 2] keep `FetchPublicURL(...)` as the public-only plugin media path rather than weakening it
- [Phase 2] when `safeurl` is used for connector traffic, layer Wanderer's connector `basePath` and normalized path-scope checks around it
- [Phase 2] allow connector media redirects to explicitly configured storage origins only when manifest and host config both permit them
- [Phase 2] strip connector auth headers and auth query parameters before following storage-origin redirects
- [Phase 2] do not support storage-origin auth contexts in the first implementation

### `db/util/network_test.go`

- [Phase 1] add focused `FetchPublicURL(...)` tests for allowed public `http` and `https` URLs
- [Phase 1] add tests for rejection of `ftp`, embedded credentials, localhost, loopback, RFC1918, link-local metadata, IPv6 loopback, and IPv6 ULA targets
- [Phase 1] add tests for rejection of non-standard public ports such as `:8080`
- [Phase 1] add tests for `https` to `http` downgrade redirects
- [Phase 1] add tests for redirects to private/special-purpose targets
- [Phase 1] add tests for exact-limit and limit-plus-one response bodies
- [Phase 1] add tests that oversized responses fail even when `Content-Length` is absent or misleading
- [Phase 2] add connector transport tests for `system`, `customCA`, and rejection of `insecure` TLS mode
- [Phase 2] add tests that custom TLS settings do not leak between connectors
- [Phase 2] add tests for allowed and rejected storage-origin redirects
- [Phase 2] add tests that storage-origin redirects are media-only unless a future provider explicitly expands that scope
- [Phase 2] add tests that `x-api-key`, `Authorization`, and auth query parameters are not forwarded to a storage-origin redirect target

### `db/pluginsystem/policy.go`

- [Phase 1] replace hostname-only authorization with URL-scope authorization for existing absolute-URL requests
- [Phase 1] include effective port in the authorization decision
- [Phase 1] stop reducing configured targets to `Hostname()` for enforcement
- [Phase 1] tighten redirect revalidation to the same scope model used for the initial request
- [Phase 2] introduce connector-aware policy structures in place of plain `UserConfiguredOrigins []string`
- [Phase 2] support both `public_api` and host-configured connector target resolution
- [Phase 2] define a normalized path-scope check based on URL semantics
- [Phase 2] use `path.Clean`-based normalization before prefix comparison
- [Phase 2] require canonical slash-delimited prefix boundaries
- [Phase 2] reject scope escapes caused by encoding, traversal, or alternate ports
- [Phase 2] reject or canonicalize connector `basePath` explicitly; never strip configured paths silently
- [Phase 2] include `basePath` in request and redirect scope validation when configured
- [Phase 2] carry `allowPrivate` through the resolved connector policy so request execution can choose the correct transport behavior
- [Phase 2] carry TLS mode and storage-origin policy through the resolved connector policy
- [Phase 2] validate storage-origin redirects separately from same-origin connector redirects
- [Phase 2] validate connector query parameters as ordered name/value pairs and reject raw query strings

### `db/pluginsystem/policy_test.go`

- [Phase 1] replace hostname-only tests with scope-aware tests for existing absolute-URL requests
- [Phase 1] add coverage for port mismatches such as `:8080` vs `:9090`
- [Phase 1] add redirect tests for same-host but different-path / different-port cases if they are implemented through policy helpers
- [Phase 2] add coverage for allowed and denied path prefixes
- [Phase 2] add coverage for encoded-path normalization
- [Phase 2] add coverage for encoded slash/backslash and double-encoded traversal rejection
- [Phase 2] add coverage for traversal attempts such as `../`
- [Phase 2] add coverage for connector `basePath` preservation and rejection of redirects outside the base path
- [Phase 2] add coverage for connector-target resolution once the policy object changes
- [Phase 2] add coverage for manifest-fixed `public_api` connector targets
- [Phase 2] add coverage for host-configured connector targets
- [Phase 2] add coverage that `public_api` connector targets cannot enable private access or custom TLS by plugin choice
- [Phase 2] add coverage for repeated connector query parameters and stable ordering
- [Phase 2] add coverage that query values cannot override connector scope

### `db/pluginsystem/host_http.go`

- [Phase 1] update redirect validation to use the same scope model as the initial request
- [Phase 1] ensure redirects cannot escape to a different port on the same host
- [Phase 1] reject `https` to `http` downgrade redirects for provider-authenticated requests
- [Phase 1] if needed, reduce or disable redirect support until scope-aware validation is in place
- [Phase 2] ensure redirects cannot escape normalized path scope on the same host
- [Phase 2] ensure redirects cannot escape the connector `basePath`
- [Phase 2] resolve connector-based targets instead of relying only on absolute `URL`
- [Phase 2] execute connector requests through the connector-scoped transport required for `allowPrivate`
- [Phase 2] share request construction and auth injection primitives with connector media fetches

### `db/pluginsystem/host_http_test.go`

- [Phase 1] add a test that rejects redirects to a different port on the same host
- [Phase 1] keep the existing undeclared-host redirect test
- [Phase 2] add a test that rejects redirects to a different path scope on the same host
- [Phase 2] add tests for connector-target execution once the ABI is added
- [Phase 2] add tests that verify private IPs are still blocked by default for connector traffic
- [Phase 2] add tests that verify private IPs are allowed only for connector targets with `allowPrivate=true`
- [Phase 2] add tests for public API connector requests without admin-configured base URLs

### `db/pluginsystem/auth_injection.go`

- [Phase 2] refactor auth injection so both `HostRequestSpec` and connector media requests can use it
- [Phase 2] expose a host-internal helper that mutates an `http.Request` or request builder from an auth context
- [Phase 2] keep validation against `manifest.Auth.Contexts` and `manifest.Permissions.Auth`
- [Phase 2] add API-key header support for Immich-style `x-api-key` media downloads through the shared path

### `db/pluginsystem/auth_injection_test.go`

- [Phase 2] add tests that API-key auth can be injected into a connector media request
- [Phase 2] add tests that unknown or unpermitted auth contexts are rejected for connector media

### `db/routes/plugin_system_policy.go`

- [Phase 2] replace origin-string extraction with host-owned connector target resolution
- [Phase 2] read connector trust config from `config.host.connectors`
- [Phase 2] stop collapsing configured targets to plain origin/hostname semantics
- [Phase 2] preserve configured connector `basePath` during resolution
- [Phase 2] resolve manifest-fixed `public_api` connectors without requiring host config
- [Phase 2] resolve host-configured connectors from `config.host.connectors`
- [Phase 2] resolve configured storage origins from `config.host.connectors.<configKey>.storageOrigins`
- [Phase 2] construct the runtime `RequestPolicyContext` from resolved connector entries rather than `UserConfiguredOrigins`
- [Phase 2] keep host-only ownership of `allowPrivate`
- [Phase 2] keep host-only ownership of TLS mode, custom CA bundles, and storage origins

### `db/routes/plugin_system_config.go`

- [Phase 2] keep the runtime split between `config.plugin` and `config.host`
- [Phase 2] ensure effective config merging preserves the resolved `config.host.connectors` structure
- [Phase 2] ensure connector `baseURL` and `basePath` normalization happens in one host-owned path
- [Phase 2] ensure connector TLS and storage-origin normalization happens in the same host-owned path
- [Phase 2] if host-owned connector trust fields are sourced differently than ordinary plugin-instance config, centralize that resolution here before policy construction

### `db/pluginsystem/protocol.go`

- [Phase 2] add connector-aware request types to `HostRequestSpec`
- [Phase 2] use ordered `[]QueryParam` for connector query parameters instead of `map[string]string`
- [Phase 2] remove plugin-supplied absolute request URLs from the stable host-request ABI
- [Phase 2] encode the first stable connector-only request shape for provider traffic, including `public_api` and host-configured connector targets

### `db/pluginsystem/import_types.go`

- [Phase 2] change the media import ABI types (`MediaSource`, `Photo`, related import structs) for connector-aware media
- [Phase 2] prefer a `Path`-only connector media model for the first cut
- [Phase 2] add `MediaRef.Auth` and validate it through manifest auth permissions
- [Phase 2] use ordered `[]QueryParam` for `MediaRef.Query`
- [Phase 2] if `assetId` remains present early, mark it as metadata-only rather than a resolution input

### `plugins/sdk/types.go`

- [Phase 2] mirror the request ABI changes made in `db/pluginsystem/protocol.go`
- [Phase 2] mirror the media import ABI changes made in `db/pluginsystem/import_types.go`
- [Phase 2] expose connector request types to plugin authors
- [Phase 2] expose connector media references with `Auth`, `Path`, and optional query fields
- [Phase 2] expose query parameters as ordered name/value pairs
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
- [Phase 2] validate connector target type: `public_api` or `configured`
- [Phase 2] require `fixedBaseURL` for `public_api` connectors and reject it for `configured` connectors
- [Phase 2] normalize `public_api.fixedBaseURL` into fixed origin plus canonical `basePath`
- [Phase 2] require `configKey` for `configured` connectors and reject or ignore it for `public_api` connectors
- [Phase 2] reject manifest attempts to select TLS mode or storage-origin concrete URLs
- [Phase 2] allow manifests to declare support for media auth, custom TLS, and storage redirects as capabilities only
- [Phase 2] validate path-prefix declarations into canonical URL-path form if they are stored in manifest metadata

### `db/pluginsystem/manifest_test.go`

- [Phase 2] add tests for any new connector-level manifest metadata
- [Phase 2] add tests that reject manifest attempts to grant private access
- [Phase 2] add tests for `public_api` vs `configured` connector validation
- [Phase 2] add tests for `configKey` rules on `public_api` and `configured` connectors
- [Phase 2] add tests that reject manifest-owned TLS trust decisions and concrete storage origins
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
- [Phase 2] surface or preserve the resolved `basePath` explicitly instead of hiding it inside `baseURL`
- [Phase 2] expose TLS mode only in host/admin-owned connector settings
- [Phase 2] make `insecure` TLS a deliberate admin action with clear warning state
- [Phase 2] expose storage-origin configuration only for connectors whose manifest declares storage redirect support
- [Phase 2] do not expose `allowPrivate` in normal user-editable plugin settings flows
- [Phase 2] keep plugin-owned fields under `config.plugin` and connector trust fields under `config.host`

### `plugins/README.md`

- [Phase 2] update the plugin-system flow documentation once connector targets replace origin-style policy
- [Phase 2] document that private access is host-owned
- [Phase 2] document that public media URLs and connector media use different trust models
- [Phase 2] document `public_api` vs host-configured connector target types

### `plugins/sdk/README.md`

- [Phase 2] document the first stable connector request ABI exposed to plugin authors
- [Phase 2] document the connector media ABI exposed to plugin authors
- [Phase 2] document connector media auth references
- [Phase 2] document which fields are metadata only versus actual resolution inputs
- [Phase 2] document explicitly that request targets are connector-based, while `MediaSource.Type=="url"` remains available only for public external assets

### First-party plugins under `plugins/*`

- [Phase 2] update bundled plugins to the first stable connector/media ABI once it exists
- [Phase 2] prefer connector `Path`-based media references over absolute provider URLs
- [Phase 2] remove assumptions that provider URLs can be emitted directly by the plugin
- [Phase 2] migrate Strava, Komoot, and Hammerhead API calls to manifest-fixed `public_api` connector targets
- [Phase 2] use Immich as the first self-hosted connector acceptance case before declaring the connector ABI complete

### Suggested first implementation slice

If the work is split into phases, the first phase should touch only the files needed for immediate hardening:

- `db/go.mod`
- `db/go.sum`
- `db/plugins/importer/importer.go`
- `db/plugins/importer/importer_test.go`
- `db/util/network.go`
- `db/util/safe_fetch.go` if the helper is split out of `network.go`
- `db/util/network_test.go`
- `db/pluginsystem/policy.go`
- `db/pluginsystem/policy_test.go`
- `db/pluginsystem/host_http.go`
- `db/pluginsystem/host_http_test.go`

Phase 1 should also define one concrete media-download limit policy:

- use `util.DefaultPluginMediaMaxBytes = 50 << 20` for the first implementation
- use `util.DefaultPluginMaxImportMediaItems = 100` for the first implementation
- optionally add `util.DefaultPluginMaxImportMediaBytes` if total byte accounting is cheap in the first slice
- keep the helper signature `FetchPublicURL(ctx, rawURL, maxBytes)` so a host-owned configurable limit can be added later without changing importer behavior

The important point is that plugin media downloads must not remain unbounded once the fetch path moves to `FetchPublicURL(...)`.

The importer should enforce the aggregate budget before or during media iteration:

- skip or reject media items beyond `DefaultPluginMaxImportMediaItems`
- keep the limit generic as "media items", not "trail photos"
- if aggregate byte accounting is implemented, stop fetching once the total imported media bytes would exceed `DefaultPluginMaxImportMediaBytes`
- log skipped media counts so large provider responses are visible during import troubleshooting

The second phase should introduce the connector ABI and runtime policy model:

- `db/pluginsystem/protocol.go`
- `db/pluginsystem/import_types.go`
- `db/pluginsystem/auth_injection.go`
- `db/pluginsystem/auth_injection_test.go`
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
