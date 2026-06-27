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
// FederationDisconnect handler and pure routing helper (CONN-04, SAFE-07)
// ---------------------------------------------------------------------------

// disconnectAction returns "delete" when followerID == localID (outbound follow,
// i.e. the local instance issued the Follow and should send Undo) and "reject"
// otherwise (inbound-only follow; deleting would fire a wrong-direction Undo per
// db/hooks/follow.go:172-184, so we set status=rejected instead to trigger the
// Reject delivery via the update hook at follow.go:136-167).
//
// This pure helper is extracted for unit-testability (T-05-10).
func disconnectAction(followerID, localID string) string {
	if followerID == localID {
		return "delete"
	}
	return "reject"
}

// FederationDisconnect handles POST /federation/disconnect/:id (CONN-04, SAFE-07).
//
// Direction-aware disconnect:
//   - Outbound follow (local is follower): calls e.App.Delete(follow). The
//     after-delete hook fires the Undo{Follow} delivery.
//   - Inbound-only follow (local is followee): sets status="rejected" and calls
//     e.App.Save(follow). The after-update hook fires the Reject{Follow} delivery.
//
// Deleting an inbound-only record would trigger the hook's Undo unconditionally
// (db/hooks/follow.go:172-184), sending a wrong-direction Undo to the remote
// instance — which is why inbound-only records must NOT be deleted (T-05-10).
//
// SAFE-07: this handler does not call federation delivery functions directly.
// Route registration happens in Plan 03.
func FederationDisconnect(e *core.RequestEvent) error {
	// 1. Auth guard — must be first (D-10, T-05-11).
	if !e.HasSuperuserAuth() {
		return e.UnauthorizedError("superuser authentication required", nil)
	}

	// 2. Extract follow ID from path.
	id := e.Request.PathValue("id")

	// 3. Load the follow record.
	follow, err := e.App.FindRecordById("follows", id)
	if err != nil {
		return e.NotFoundError("follow not found", err)
	}

	// 4. Look up local instance actor for direction check.
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	// 5. Direction decision via the pure helper.
	action := disconnectAction(follow.GetString("follower"), localActor.Id)

	if action == "delete" {
		// Outbound: delete so the after-delete hook fires the Undo{Follow} delivery.
		if err := e.App.Delete(follow); err != nil {
			return fmt.Errorf("delete follow: %w", err)
		}
	} else {
		// Inbound-only: set rejected so the after-update hook fires Reject{Follow}.
		follow.Set("status", "rejected")
		if err := e.App.Save(follow); err != nil {
			return fmt.Errorf("save follow rejected: %w", err)
		}
	}

	return e.JSON(http.StatusOK, map[string]any{"status": "ok"})
}

// ---------------------------------------------------------------------------
// FederationPeers handler and pure folding helper (DISC-01, D-04, D-05)
// ---------------------------------------------------------------------------

// followInput is a lightweight value type used by buildPeerEntries so the
// folding logic stays pure and unit-testable without requiring core.Record
// construction. The handler converts core.Record slices to []followInput.
type followInput struct {
	ID       string
	Follower string
	Followee string
	Status   string
}

// buildPeerEntries folds outbound and inbound follow slices into a deduplicated
// []peerEntry keyed by remote domain (D-04). When an accepted outbound record and
// an accepted inbound record share the same remote domain, they collapse to a
// single "mutual" entry whose FollowID is the outbound record id (D-05).
//
// domainOf resolves an actor id to its domain string; the FederationPeers handler
// supplies a closure backed by FindRecordById so this helper requires no DB access.
//
// The returned slice is sorted by domain for deterministic output.
func buildPeerEntries(
	outbound []followInput,
	inbound []followInput,
	localID string,
	domainOf func(actorID string) string,
) []peerEntry {
	// Map from remote domain → entry pointer so we can merge in a second pass.
	type mergeEntry struct {
		entry      peerEntry
		hasOutbound bool
	}
	byDomain := make(map[string]*mergeEntry)

	// First pass: outbound follows (local is follower; remote is followee).
	for _, f := range outbound {
		remoteID := f.Followee
		domain := domainOf(remoteID)
		if domain == "" {
			domain = remoteID // fallback
		}
		byDomain[domain] = &mergeEntry{
			entry: peerEntry{
				FollowID:  f.ID,
				Direction: "outbound",
				Status:    f.Status,
				Domain:    domain,
			},
			hasOutbound: true,
		}
	}

	// Second pass: inbound follows (remote is follower; local is followee).
	for _, f := range inbound {
		remoteID := f.Follower
		domain := domainOf(remoteID)
		if domain == "" {
			domain = remoteID // fallback
		}
		if existing, ok := byDomain[domain]; ok && existing.hasOutbound &&
			existing.entry.Status == "accepted" && f.Status == "accepted" {
			// Both sides accepted → mutual. Keep the outbound follow_id (D-05).
			existing.entry.Direction = "mutual"
		} else if !ok {
			byDomain[domain] = &mergeEntry{
				entry: peerEntry{
					FollowID:  f.ID,
					Direction: "inbound",
					Status:    f.Status,
					Domain:    domain,
				},
			}
		}
		// If existing outbound is not accepted, leave direction as-is.
	}

	// Collect and sort by domain for deterministic output.
	result := make([]peerEntry, 0, len(byDomain))
	// Gather domains for sorting.
	domains := make([]string, 0, len(byDomain))
	for d := range byDomain {
		domains = append(domains, d)
	}
	// Simple insertion sort (peer lists are typically small).
	for i := 1; i < len(domains); i++ {
		for j := i; j > 0 && domains[j] < domains[j-1]; j-- {
			domains[j], domains[j-1] = domains[j-1], domains[j]
		}
	}
	for _, d := range domains {
		result = append(result, byDomain[d].entry)
	}
	return result
}

// FederationPeers handles GET /federation/peers (DISC-01, D-04, D-05).
//
// Returns a JSON array of peerEntry objects, one per unique remote domain, with
// mutual pairs collapsed (buildPeerEntries). Read-only handler (SAFE-07).
// Route registration happens in Plan 03.
func FederationPeers(e *core.RequestEvent) error {
	// 1. Auth guard — must be first (D-10, T-05-12).
	if !e.HasSuperuserAuth() {
		return e.UnauthorizedError("superuser authentication required", nil)
	}

	// 2. Look up local instance actor.
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	// 3. Query outbound follows (local is follower).
	outboundRecords, err := e.App.FindRecordsByFilter(
		"follows",
		"follower={:local}",
		"-created",
		-1,
		0,
		dbx.Params{"local": localActor.Id},
	)
	if err != nil {
		// Treat "no records" as empty rather than an error.
		outboundRecords = nil
	}

	// 4. Query inbound follows (local is followee).
	inboundRecords, err := e.App.FindRecordsByFilter(
		"follows",
		"followee={:local}",
		"-created",
		-1,
		0,
		dbx.Params{"local": localActor.Id},
	)
	if err != nil {
		inboundRecords = nil
	}

	// 5. Convert records to followInput for the pure helper.
	toInputs := func(records []*core.Record) []followInput {
		out := make([]followInput, 0, len(records))
		for _, r := range records {
			out = append(out, followInput{
				ID:       r.Id,
				Follower: r.GetString("follower"),
				Followee: r.GetString("followee"),
				Status:   r.GetString("status"),
			})
		}
		return out
	}

	// 6. domainOf closure: resolves actor id → domain field via DB lookup.
	domainOf := func(actorID string) string {
		actor, err := e.App.FindRecordById("activitypub_actors", actorID)
		if err != nil || actor == nil {
			return ""
		}
		return actor.GetString("domain")
	}

	// 7. Fold into peer entries.
	entries := buildPeerEntries(
		toInputs(outboundRecords),
		toInputs(inboundRecords),
		localActor.Id,
		domainOf,
	)

	return e.JSON(http.StatusOK, entries)
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
