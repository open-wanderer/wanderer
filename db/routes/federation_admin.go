package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"pocketbase/federation"
	"pocketbase/util"

	"github.com/doyensec/safeurl"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// nodeInfoLink represents a single entry in the JRD /.well-known/nodeinfo
// discovery document links array.
type nodeInfoLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// nodeInfo21 is the decoded NodeInfo 2.1 payload shape. Only the fields
// needed by the discovery handler are represented.
type nodeInfo21 struct {
	Software struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"software"`
	Usage struct {
		Users struct {
			Total int64 `json:"total"`
		} `json:"users"`
		LocalPosts int64 `json:"localPosts"`
	} `json:"usage"`
}

// peerEntry is the JSON shape for one peer connection in the GET
// /federation/peers response (used by FederationPeers in Plan 03).
type peerEntry struct {
	FollowID  string `json:"follow_id"`
	Direction string `json:"direction"` // "outbound" | "inbound" | "mutual"
	Status    string `json:"status"`
	Domain    string `json:"domain"`
}

// httpDoer is satisfied by *safeurl.WrappedClient and *http.Client, allowing
// the helpers to accept either. This avoids forcing callers to depend on the
// concrete safeurl type.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ---------------------------------------------------------------------------
// SSRF-safe discovery client (SAFE-06: 10-second timeout, not 60s)
// ---------------------------------------------------------------------------

// newDiscoveryClient builds a safeurl.WrappedClient with a 10-second timeout
// for all outbound NodeInfo fetches triggered by admin-supplied URLs (SAFE-06).
// The existing FetchPublicURL utility uses a 60s timeout and must NOT be used
// here — this dedicated client satisfies the ≤10s requirement.
//
// The builder mirrors the pattern in db/util/safe_fetch.go:48-56 with the
// timeout reduced to 10 seconds per SAFE-06.
func newDiscoveryClient() *safeurl.WrappedClient {
	config := safeurl.GetConfigBuilder().
		SetTimeout(10 * time.Second).
		SetAllowedSchemes("http", "https").
		SetAllowedPorts(80, 443).
		EnableIPv6(true).
		AllowSendingCredentials(false).
		Build()
	return safeurl.Client(config)
}

// ---------------------------------------------------------------------------
// NodeInfo helpers
// ---------------------------------------------------------------------------

// pickNodeInfo21Href returns the Href of the link whose Rel equals the
// NodeInfo 2.1 schema URI. Returns an error containing "not a Wanderer
// instance" if no such link is present (D-08, DISC-02, Pitfall 6).
func pickNodeInfo21Href(links []nodeInfoLink) (string, error) {
	const rel21 = "http://nodeinfo.diaspora.software/ns/schema/2.1"
	for _, link := range links {
		if link.Rel == rel21 {
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("not a Wanderer instance: no NodeInfo 2.1 endpoint found")
}

// fetchNodeInfo21URL performs the two-step NodeInfo discovery for rawURL:
//  1. GETs /.well-known/nodeinfo on the remote host (the JRD discovery doc).
//  2. Calls pickNodeInfo21Href to locate the 2.1 href by rel value (D-08).
//
// Both requests use the provided SSRF-safe 10s client (SAFE-06).
// Body size is bounded to 64 KiB (T-05-03).
// On network failure the returned error contains "unreachable" (DISC-02).
func fetchNodeInfo21URL(client httpDoer, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("unreachable: invalid URL")
	}

	jrdURL := fmt.Sprintf("%s://%s/.well-known/nodeinfo", u.Scheme, u.Host)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, jrdURL, nil)
	if err != nil {
		return "", fmt.Errorf("unreachable: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unreachable: %w", err)
	}
	defer resp.Body.Close()

	var jrd struct {
		Links []nodeInfoLink `json:"links"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&jrd); err != nil {
		return "", fmt.Errorf("not a Wanderer instance: invalid discovery document")
	}

	return pickNodeInfo21Href(jrd.Links)
}

// ---------------------------------------------------------------------------
// Local instance actor lookup
// ---------------------------------------------------------------------------

// findLocalInstanceActor looks up the record in activitypub_actors that has
// actor_type="instance" and is_local=true. This is the Application-type actor
// that represents this Wanderer instance in the ActivityPub network.
//
// This helper is reused by FederationFollow (Plan 02), FederationPeers, and
// FederationDisconnect (Plan 03).
func findLocalInstanceActor(app core.App) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		"activitypub_actors",
		"actor_type={:t} && is_local={:l}",
		dbx.Params{"t": "instance", "l": true},
	)
}

// ---------------------------------------------------------------------------
// FederationDiscover handler
// ---------------------------------------------------------------------------

// FederationDiscover handles POST /federation/discover.
//
// It is the required first step of the connect flow (D-01). The admin
// supplies a remote Wanderer instance URL; this handler:
//  1. Enforces superuser auth (D-10, T-05-04).
//  2. Fetches /.well-known/nodeinfo and follows the 2.1 link (D-08, SAFE-06).
//  3. Decodes NodeInfo 2.1 and verifies software.name == "wanderer" (D-06).
//  4. Derives the remote instance actor IRI and runs the SAFE-05 self-follow
//     guard via util.IsLocalIRI.
//  5. Checks that no follow record already exists between the two actors
//     (Pitfall 5, T-05-05).
//  6. Bypasses the 2-hour actor cache by clearing last_fetched (D-03).
//  7. Calls federation.GetActorByIRI to create or refresh the remote actor.
//  8. Returns { actor_id, domain, version, user_count, trail_count } (DISC-01).
//
// ---------------------------------------------------------------------------
// FederationFollow handler and DB helper
// ---------------------------------------------------------------------------

// createOutboundFollow inserts a follows record where localID is the follower
// and remoteID is the followee with status="pending". It first confirms the
// remote actor exists in activitypub_actors (D-02). The after-create hook
// (InstanceFollowCreateHandler) fires the outbound Follow delivery — this
// helper must NOT call federation delivery functions directly (SAFE-07).
func createOutboundFollow(app core.App, localID, remoteID string) (*core.Record, error) {
	// D-02: remote actor must have an activitypub_actors record.
	if _, err := app.FindRecordById("activitypub_actors", remoteID); err != nil {
		return nil, fmt.Errorf("unknown actor; run discover first")
	}

	followCollection, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		return nil, fmt.Errorf("follows collection not found: %w", err)
	}
	rec := core.NewRecord(followCollection)
	rec.Set("follower", localID)
	rec.Set("followee", remoteID)
	rec.Set("status", "pending")
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("save follow record: %w", err)
	}
	return rec, nil
}

// FederationFollow handles POST /federation/follow.
//
// Accepts { "actor_id": "<activitypub_actors record id>" } and creates an
// outbound follows record with the local instance as the follower and status
// "pending" (CONN-01). The after-create hook fires the Follow activity delivery
// — this handler must NOT call federation delivery functions directly (SAFE-07).
//
// Returns { "follow_id": "<id>", "status": "pending" } on success.
// Route registration happens in Plan 03.
func FederationFollow(e *core.RequestEvent) error {
	// 1. Auth guard — must be first (D-10, T-05-06).
	if !e.HasSuperuserAuth() {
		return e.UnauthorizedError("superuser authentication required", nil)
	}

	// 2. Decode body { "actor_id": "<id>" }.
	var body struct {
		ActorID string `json:"actor_id"`
	}
	if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || body.ActorID == "" {
		return e.BadRequestError("actor_id is required", nil)
	}

	// 3. Verify remote actor exists (D-02).
	remoteActor, err := e.App.FindRecordById("activitypub_actors", body.ActorID)
	if err != nil || remoteActor == nil {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "unknown actor; run discover first"})
	}

	// 4. Look up local instance actor.
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	// 5. Create the outbound follows record via the testable helper.
	rec, err := createOutboundFollow(e.App, localActor.Id, remoteActor.Id)
	if err != nil {
		return fmt.Errorf("createOutboundFollow: %w", err)
	}

	// 6. Return the follow_id and pending status.
	return e.JSON(http.StatusOK, map[string]any{
		"follow_id": rec.Id,
		"status":    "pending",
	})
}

// ---------------------------------------------------------------------------
// FederationApprove + FederationReject handlers and shared DB helper
// ---------------------------------------------------------------------------

// setFollowStatus updates a follows record to the given status, enforcing that
// the local instance (identified by localID) is the followee — i.e., this is
// an inbound follow that the admin is approving or rejecting (T-05-07).
//
// The after-update hook (InstanceFollowUpdateHandler) fires the Accept or
// Reject delivery — this helper must NOT call federation delivery functions
// directly (SAFE-07).
func setFollowStatus(app core.App, followID, status, localID string) error {
	follow, err := app.FindRecordById("follows", followID)
	if err != nil {
		return fmt.Errorf("follow not found: %w", err)
	}

	// Direction guard (T-05-07): approve/reject only apply to inbound follows
	// where the local instance is the followee. This mirrors the hook guard in
	// db/hooks/follow.go:146 so the Accept/Reject delivery will actually fire.
	if follow.GetString("followee") != localID {
		return fmt.Errorf("not an inbound follow")
	}

	follow.Set("status", status)
	return app.Save(follow)
}

// FederationApprove handles POST /federation/approve/:id.
//
// Moves an inbound pending follows record to status "accepted" (CONN-02).
// The after-update hook fires the Accept delivery — this handler must NOT
// call federation delivery functions directly (SAFE-07).
//
// Route registration happens in Plan 03.
func FederationApprove(e *core.RequestEvent) error {
	// 1. Auth guard — must be first (D-10, T-05-06).
	if !e.HasSuperuserAuth() {
		return e.UnauthorizedError("superuser authentication required", nil)
	}

	// 2. Extract follow ID from path.
	id := e.Request.PathValue("id")

	// 3. Verify follow exists.
	follow, err := e.App.FindRecordById("follows", id)
	if err != nil {
		return e.NotFoundError("follow not found", err)
	}

	// 4. Look up local instance actor for direction guard.
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	// 5. Apply direction guard and status update via the testable helper.
	if err := setFollowStatus(e.App, follow.Id, "accepted", localActor.Id); err != nil {
		return e.BadRequestError("not an inbound follow", nil)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"follow_id": follow.Id,
		"status":    "accepted",
	})
}

// FederationReject handles POST /federation/reject/:id.
//
// Moves an inbound pending follows record to status "rejected" (CONN-03).
// The after-update hook fires the Reject delivery — this handler must NOT
// call federation delivery functions directly (SAFE-07).
//
// Route registration happens in Plan 03.
func FederationReject(e *core.RequestEvent) error {
	// 1. Auth guard — must be first (D-10, T-05-06).
	if !e.HasSuperuserAuth() {
		return e.UnauthorizedError("superuser authentication required", nil)
	}

	// 2. Extract follow ID from path.
	id := e.Request.PathValue("id")

	// 3. Verify follow exists.
	follow, err := e.App.FindRecordById("follows", id)
	if err != nil {
		return e.NotFoundError("follow not found", err)
	}

	// 4. Look up local instance actor for direction guard.
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	// 5. Apply direction guard and status update via the testable helper.
	if err := setFollowStatus(e.App, follow.Id, "rejected", localActor.Id); err != nil {
		return e.BadRequestError("not an inbound follow", nil)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"follow_id": follow.Id,
		"status":    "rejected",
	})
}

// ---------------------------------------------------------------------------
// FederationDiscover handler
// ---------------------------------------------------------------------------

// Route registration happens in Plan 03 alongside the other five handlers.
// SAFE-07: this handler does not call federation delivery functions directly.
func FederationDiscover(e *core.RequestEvent) error {
	// 1. Auth guard — must be first (D-10, T-05-04).
	if !e.HasSuperuserAuth() {
		return e.UnauthorizedError("superuser authentication required", nil)
	}

	// 2. Decode request body { "url": "<remote>" }.
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || body.URL == "" {
		return e.BadRequestError("url is required", nil)
	}

	// 3. Fetch NodeInfo 2.1 URL via the JRD discovery document (SAFE-06).
	client := newDiscoveryClient()
	nodeInfoURL, err := fetchNodeInfo21URL(client, body.URL)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}

	// 4a. Fetch the NodeInfo 2.1 payload.
	niReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, nodeInfoURL, nil)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "unreachable"})
	}
	niResp, err := client.Do(niReq)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "unreachable"})
	}
	defer niResp.Body.Close()

	// 4b. Decode and verify Wanderer identity (D-06, DISC-02).
	var ni nodeInfo21
	if err := json.NewDecoder(io.LimitReader(niResp.Body, 64*1024)).Decode(&ni); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "not a Wanderer instance"})
	}
	if ni.Software.Name != "wanderer" {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "not a Wanderer instance"})
	}

	// 5. Derive actor IRI and run SAFE-05 self-follow guard (T-05-02).
	parsedURL, err := url.Parse(body.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "unreachable"})
	}
	actorIRI := fmt.Sprintf("%s://%s/api/v1/activitypub/instance", parsedURL.Scheme, parsedURL.Host)

	if util.IsLocalIRI(actorIRI) {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "resolves to local instance"})
	}

	// 6. Already-connected check (Pitfall 5, T-05-05).
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	existingActor, remoteActorErr := e.App.FindFirstRecordByFilter(
		"activitypub_actors",
		"iri={:iri}",
		dbx.Params{"iri": actorIRI},
	)
	if remoteActorErr == nil && existingActor != nil {
		// Remote actor record exists — check for an existing follow in either direction.
		_, followErr := e.App.FindFirstRecordByFilter(
			"follows",
			"(follower={:l} && followee={:r}) || (follower={:r} && followee={:l})",
			dbx.Params{"l": localActor.Id, "r": existingActor.Id},
		)
		if followErr == nil {
			// A follow record was found → already connected.
			return e.JSON(http.StatusBadRequest, map[string]any{"error": "already connected"})
		}

		// 7. Cache bypass (D-03): clear last_fetched so GetActorByIRI re-fetches.
		existingActor.Set("last_fetched", time.Time{})
		_ = e.App.Save(existingActor)
	}

	// 7b. Fetch or create the remote actor record via GetActorByIRI.
	actor, err := federation.GetActorByIRI(e.App, context.Background(), actorIRI, false)
	if err != nil {
		return e.JSON(http.StatusBadGateway, map[string]any{"error": "unreachable"})
	}

	// 8. Build the preview card response (DISC-01, D-07).
	domain := actor.GetString("domain")
	if domain == "" {
		domain = parsedURL.Hostname()
	}

	return e.JSON(http.StatusOK, map[string]any{
		"actor_id":    actor.Id,
		"domain":      domain,
		"version":     ni.Software.Version,
		"user_count":  ni.Usage.Users.Total,
		"trail_count": ni.Usage.LocalPosts,
	})
}
