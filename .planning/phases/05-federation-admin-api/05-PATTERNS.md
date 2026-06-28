# Phase 5: Federation Admin API - Pattern Map

**Mapped:** 2026-06-27
**Files analyzed:** 2 (1 new route file + 1 registration change)
**Analogs found:** 2 / 2

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `db/routes/federation_admin.go` | controller (6 handlers) | request-response + CRUD | `db/routes/nodeinfo.go`, `db/routes/plugin_system.go`, `db/routes/activitypub.go` | exact (role + data flow) |
| `db/main.go` (registerRoutes only) | config/registration | — | `db/main.go:172–209` | exact |

---

## Pattern Assignments

### `db/routes/federation_admin.go` (controller, request-response + CRUD)

**Primary analogs:** `db/routes/nodeinfo.go`, `db/routes/plugin_system.go`, `db/routes/activitypub.go`

---

#### Imports pattern

Copy from `db/routes/nodeinfo.go` lines 1–11 and `db/routes/activitypub.go` lines 1–17, combined:

```go
package routes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"pocketbase/federation"
	"pocketbase/util"
	"time"

	"github.com/doyensec/safeurl"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)
```

---

#### Auth guard pattern (superuser check)

**Source:** `db/routes/plugin_system.go` lines 14–17

Apply this as the **first statement** in every one of the six handlers:

```go
// db/routes/plugin_system.go:14-17
func PluginSystemPluginsList(e *core.RequestEvent) error {
	if e.Auth == nil && !e.HasSuperuserAuth() {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	// ...
}
```

For the federation admin handlers, the condition is stricter (superuser only, not any auth):

```go
if !e.HasSuperuserAuth() {
    return e.UnauthorizedError("superuser authentication required", nil)
}
```

`e.HasSuperuserAuth()` is confirmed at `db/routes/plugin_system.go:15,27`.

---

#### SSRF-safe discovery client (10s timeout, SAFE-06)

**Source:** `db/util/safe_fetch.go` lines 48–56 (the `FetchPublicURL` safeurl config block)

`FetchPublicURL` itself uses 60s — do NOT call it for discovery. Instead, build a local 10s client:

```go
// Adapted from db/util/safe_fetch.go:48-56 with timeout reduced to 10s (SAFE-06)
func newDiscoveryClient() *http.Client {
	config := safeurl.GetConfigBuilder().
		SetTimeout(10 * time.Second).
		SetAllowedSchemes("http", "https").
		SetAllowedPorts(80, 443).
		EnableIPv6(true).
		AllowSendingCredentials(false).
		Build()
	return safeurl.Client(config)
}
```

Use this client for both the JRD `/.well-known/nodeinfo` fetch and the NodeInfo 2.1 payload fetch.

---

#### NodeInfo two-step fetch pattern

**Source:** `db/routes/nodeinfo.go` lines 23–36 (shows JSON encode/decode flow); `db/routes/activitypub.go` lines 70–77 (body read + JSON decode pattern)

NodeInfo JRD fetch and link selection:

```go
// Pattern: fetch /.well-known/nodeinfo, find the 2.1 href
func fetchNodeInfo21URL(client *http.Client, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("unreachable: invalid URL")
	}
	jrdURL := fmt.Sprintf("%s://%s/.well-known/nodeinfo", u.Scheme, u.Host)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, jrdURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unreachable")
	}
	defer resp.Body.Close()

	var jrd struct {
		Links []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&jrd); err != nil {
		return "", fmt.Errorf("invalid discovery document")
	}
	// D-08: select by rel, not by index (may have 2.0 and 2.1 links)
	for _, link := range jrd.Links {
		if link.Rel == "http://nodeinfo.diaspora.software/ns/schema/2.1" {
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("not a Wanderer instance")
}
```

`io.LimitReader` pattern sourced from `db/federation/actor.go:353` (noted in RESEARCH.md security domain).

---

#### SAFE-05 self-follow guard

**Source:** `db/util/activitypub.go:124` (confirmed in RESEARCH.md)

```go
// After extracting actorIRI from NodeInfo response:
if util.IsLocalIRI(actorIRI) {
    return e.JSON(http.StatusBadRequest, map[string]any{
        "error": "resolves to local instance",
    })
}
```

---

#### Actor cache bypass for discovery (D-03)

**Source:** `db/federation/actor.go:221` (cache check: `last_fetched` within 2 hours short-circuits)

```go
// Clear last_fetched to force GetActorByIRI to re-fetch from remote (D-03)
if existingActor, err := e.App.FindFirstRecordByData(
    "activitypub_actors", "iri", actorIRI,
); err == nil {
    existingActor.Set("last_fetched", time.Time{}) // zero time → older than any 2-hour window
    _ = e.App.Save(existingActor)
}
actor, err := federation.GetActorByIRI(e.App, context.Background(), actorIRI, false)
```

---

#### Local instance actor lookup

**Source:** `db/federation/instance.go` lines 44–45 (`FindFirstRecordByData` pattern) and RESEARCH.md Pattern 3

```go
// db/federation/instance.go:44 pattern — find local instance actor
localActor, err := e.App.FindFirstRecordByFilter(
    "activitypub_actors",
    "actor_type={:t} && is_local={:l}",
    dbx.Params{"t": "instance", "l": true},
)
if err != nil {
    return err
}
```

Used in: `POST /federation/follow`, `GET /federation/peers`, `POST /federation/disconnect/:id`.

---

#### follows record create pattern (outbound follow)

**Source:** `db/federation/follow.go` lines 97–113 (`ProcessFollowActivity` inbound path — adapted for outbound admin-initiated follow)

```go
// Adapted from db/federation/follow.go:97-113
followCollection, err := e.App.FindCollectionByNameOrId("follows")
if err != nil {
    return err
}
followRecord := core.NewRecord(followCollection)
followRecord.Set("follower", localActor.Id)  // local instance actor
followRecord.Set("followee", remoteActor.Id) // actor returned by /discover
followRecord.Set("status", "pending")
if err := e.App.Save(followRecord); err != nil {
    return err
}
// Hook InstanceFollowCreateHandler fires → CreateFollowActivity — do NOT call directly (SAFE-07)
return e.JSON(http.StatusOK, map[string]any{"follow_id": followRecord.Id, "status": "pending"})
```

---

#### follows record status update pattern (approve / reject)

**Source:** `db/routes/activitypub.go` lines 113–130 (switch/dispatch + `app.Save` pattern); `db/federation/follow.go` lines 80–113

```go
// Pattern for POST /federation/approve/:id and POST /federation/reject/:id
id := e.Request.PathValue("id")
follow, err := e.App.FindRecordById("follows", id)
if err != nil {
    return e.NotFoundError("follow not found", err)
}
// Direction guard: must be inbound (followee = local instance actor)
if follow.GetString("followee") != localActor.Id {
    return e.BadRequestError("not an inbound follow", nil)
}
follow.Set("status", "accepted") // or "rejected"
if err := e.App.Save(follow); err != nil {
    return err
}
// Hook InstanceFollowUpdateHandler fires → CreateAcceptFollowActivity / CreateRejectFollowActivity
return e.JSON(http.StatusOK, map[string]any{"follow_id": follow.Id, "status": "accepted"})
```

---

#### Direction-aware disconnect pattern (D-11, CONN-04)

**Source:** `db/federation/follow.go` lines 24–62 (shows follower/followee field access)

```go
// POST /federation/disconnect/:id — direction-aware per CONN-04
id := e.Request.PathValue("id")
follow, err := e.App.FindRecordById("follows", id)
if err != nil {
    return e.NotFoundError("follow not found", err)
}
if follow.GetString("follower") == localActor.Id {
    // Outbound follow: delete triggers InstanceFollowDeleteHandler → CreateUnfollowActivity (Undo)
    if err := e.App.Delete(follow); err != nil {
        return err
    }
} else {
    // Inbound-only: update to rejected triggers InstanceFollowUpdateHandler → CreateRejectFollowActivity
    // Do NOT delete — Undo verb is invalid when local instance was never the follower
    follow.Set("status", "rejected")
    if err := e.App.Save(follow); err != nil {
        return err
    }
}
return e.JSON(http.StatusOK, map[string]any{"status": "ok"})
```

---

#### Peer list query and mutual collapse pattern (D-04, D-05)

**Source:** `db/federation/follow.go` lines 85–96 (`FindFirstRecordByFilter` + `dbx.Params` pattern); RESEARCH.md code example for peer list

```go
// GET /federation/peers — two queries + in-memory grouping (D-04, D-05)
// FindRecordsByFilter (plural) preferred; fallback: FindAllRecords + Go filter
outbound, err := e.App.FindRecordsByFilter(
    "follows",
    "follower={:local}",
    "-created", -1, 0,
    dbx.Params{"local": localActor.Id},
)
// err is non-fatal if zero results

inbound, err := e.App.FindRecordsByFilter(
    "follows",
    "followee={:local}",
    "-created", -1, 0,
    dbx.Params{"local": localActor.Id},
)

// Build map keyed by remote actor ID; collapse mutual pairs in Go
// domain extracted from remote actor's "domain" field
type peerEntry struct {
    FollowID  string `json:"follow_id"`
    Direction string `json:"direction"` // "outbound" | "inbound" | "mutual"
    Status    string `json:"status"`
    Domain    string `json:"domain"`
}
// ... grouping logic ...
return e.JSON(http.StatusOK, entries)
```

---

#### JSON response format

**Source:** `db/routes/activitypub.go` lines 54–63; `db/federation/instance.go` line 192

```go
// Success:
return e.JSON(http.StatusOK, map[string]any{"key": value})

// 400 Bad Request:
return e.BadRequestError("message", optionalWrappedErr)

// 401 Unauthorized:
return e.UnauthorizedError("superuser authentication required", nil)

// 404 Not Found:
return e.NotFoundError("follow not found", err)

// Custom error with status:
return e.JSON(http.StatusBadRequest, map[string]any{"error": "not a Wanderer instance"})
```

---

#### Error handling pattern

**Source:** `db/routes/activitypub.go` lines 45–63; `db/routes/nodeinfo.go` lines 14–20

```go
// Propagate DB errors directly (PocketBase maps to 500):
result, err := e.App.FindRecordById("follows", id)
if err != nil {
    return err  // PocketBase wraps in 404 for sql.ErrNoRows automatically
}

// Application-level 400:
return e.BadRequestError("already connected", nil)
```

---

### `db/main.go` — registerRoutes() addition

**Source:** `db/main.go` lines 172–209 (existing route registration block)

Add six entries at the end of `registerRoutes()`, following the existing `se.Router.METHOD(path, handler)` pattern:

```go
// db/main.go:registerRoutes — add after existing routes (lines 172-209)
se.Router.POST("/federation/discover", routes.FederationDiscover)
se.Router.POST("/federation/follow", routes.FederationFollow)
se.Router.POST("/federation/approve/{id}", routes.FederationApprove)
se.Router.POST("/federation/reject/{id}", routes.FederationReject)
se.Router.POST("/federation/disconnect/{id}", routes.FederationDisconnect)
se.Router.GET("/federation/peers", routes.FederationPeers)
```

Path value syntax `{id}` matches existing pattern at `db/main.go:199` (`/activitypub/actor/{id}/{follow}`).

---

## Shared Patterns

### Superuser Auth Guard
**Source:** `db/routes/plugin_system.go` lines 14–17
**Apply to:** All six handlers — first statement before any logic

```go
if !e.HasSuperuserAuth() {
    return e.UnauthorizedError("superuser authentication required", nil)
}
```

### SSRF-Safe HTTP Client
**Source:** `db/util/safe_fetch.go` lines 48–56 (builder pattern)
**Apply to:** `FederationDiscover` only — use local `newDiscoveryClient()` (10s timeout), NOT `util.FetchPublicURL` (60s)

### dbx.Params Parameterized Queries
**Source:** `db/federation/follow.go` lines 85–88
**Apply to:** All DB filter calls in `federation_admin.go`

```go
dbx.Params{"key": value}  // always use; never string interpolation
```

### PathValue for URL Parameters
**Source:** `db/routes/activitypub.go` line 133 (`e.Request.PathValue("id")`)
**Apply to:** `FederationApprove`, `FederationReject`, `FederationDisconnect`

```go
id := e.Request.PathValue("id")
```

### SAFE-07: No Direct Federation Function Calls
**Apply to:** All six handlers — handlers call `e.App.Save()` / `e.App.Delete()` only. Never call `federation.CreateFollowActivity`, `federation.CreateAcceptFollowActivity`, `federation.CreateRejectFollowActivity`, or `federation.CreateUnfollowActivity` from route handlers.

---

## No Analog Found

None. All patterns are covered by existing codebase files.

---

## Metadata

**Analog search scope:** `db/routes/`, `db/federation/`, `db/util/`, `db/main.go`
**Files read:** `db/routes/nodeinfo.go`, `db/routes/activitypub.go`, `db/routes/plugin_system.go`, `db/util/safe_fetch.go`, `db/federation/follow.go` (lines 1–120), `db/federation/instance.go`, `db/main.go` (lines 125–224)
**Pattern extraction date:** 2026-06-27
