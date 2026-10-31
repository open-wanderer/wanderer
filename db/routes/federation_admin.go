package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"pocketbase/federation"
	"pocketbase/util"

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
// /federation/peers response.
type peerEntry struct {
	FollowID string `json:"follow_id"`
	// InboundFollowID is the inbound record of a "mutual" entry; empty otherwise.
	InboundFollowID string `json:"inbound_follow_id,omitempty"`
	Direction       string `json:"direction"` // "outbound" | "inbound" | "mutual"
	Status          string `json:"status"`
	Domain          string `json:"domain"`
}

// ---------------------------------------------------------------------------
// NodeInfo helpers
// ---------------------------------------------------------------------------

// pickNodeInfo21Href returns the Href of the link whose Rel equals the
// NodeInfo 2.1 schema URI. Returns an error containing "not a Wanderer
// instance" if no such link is present.
func pickNodeInfo21Href(links []nodeInfoLink) (string, error) {
	const rel21 = "http://nodeinfo.diaspora.software/ns/schema/2.1"
	for _, link := range links {
		if link.Rel == rel21 {
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("not a Wanderer instance: no NodeInfo 2.1 endpoint found")
}

// fetchNodeInfo21URL fetches /.well-known/nodeinfo from rawURL's host and returns
// the NodeInfo 2.1 href. Bodies are limited to 64 KiB; network errors contain
// "unreachable".
func fetchNodeInfo21URL(client util.HTTPDoer, rawURL string) (string, error) {
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

	// A non-200 response is not a valid JRD document.
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unreachable: JRD returned HTTP %d", resp.StatusCode)
	}

	var jrd struct {
		Links []nodeInfoLink `json:"links"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&jrd); err != nil {
		return "", fmt.Errorf("not a Wanderer instance: invalid discovery document")
	}

	href, err := pickNodeInfo21Href(jrd.Links)
	if err != nil {
		return "", err
	}

	// The href must stay on the requested host.
	hrefParsed, err := url.Parse(href)
	if err != nil || !strings.EqualFold(hrefParsed.Host, u.Host) {
		return "", fmt.Errorf("not a Wanderer instance: NodeInfo href host mismatch")
	}

	return href, nil
}

// ---------------------------------------------------------------------------
// Local instance actor lookup
// ---------------------------------------------------------------------------

// findLocalInstanceActor returns the local instance actor.
func findLocalInstanceActor(app core.App) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		"activitypub_actors",
		"actor_type={:t} && is_local={:l}",
		dbx.Params{"t": "instance", "l": true},
	)
}

// ---------------------------------------------------------------------------
// FederationFollow handler and DB helper
// ---------------------------------------------------------------------------

// createOutboundFollow creates a pending follows record from localID to remoteID.
// The follows hooks deliver the Follow.
func createOutboundFollow(app core.App, localID, remoteID string) (*core.Record, error) {
	// Remote actor must have an activitypub_actors record.
	if _, err := app.FindRecordById("activitypub_actors", remoteID); err != nil {
		return nil, fmt.Errorf("unknown actor; run discover first")
	}

	// A rejected follow is replaced so the admin can retry; pending and accepted
	// follows block a new one.
	existing, checkErr := app.FindFirstRecordByFilter(
		"follows",
		"follower={:f} && followee={:e}",
		dbx.Params{"f": localID, "e": remoteID},
	)
	if checkErr == nil && existing != nil {
		if existing.GetString("status") == "rejected" {
			if err := app.Delete(existing); err != nil {
				return nil, fmt.Errorf("clear rejected follow: %w", err)
			}
		} else {
			return nil, fmt.Errorf("follow already exists")
		}
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

// FederationFollow handles POST /federation/follow with { "actor_id": "<id>" }
// and creates a pending outbound follow. Returns { "follow_id", "status" }.
func FederationFollow(e *core.RequestEvent) error {
	// 1. Auth guard — must be first.
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

	// 3. Verify remote actor exists.
	remoteActor, err := e.App.FindRecordById("activitypub_actors", body.ActorID)
	if err != nil || remoteActor == nil {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "unknown actor; run discover first"})
	}

	// 4. Look up local instance actor.
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	// 4a. Refuse following the local instance itself.
	if localActor.Id == remoteActor.Id {
		return e.BadRequestError("cannot follow local instance actor", nil)
	}

	// 5. Create the outbound follows record via the testable helper.
	rec, err := createOutboundFollow(e.App, localActor.Id, remoteActor.Id)
	if err != nil {
		// An existing follow is a 409, not a 500.
		if err.Error() == "follow already exists" {
			return e.JSON(http.StatusConflict, map[string]any{"error": "already following this instance"})
		}
		return fmt.Errorf("createOutboundFollow: %w", err)
	}

	// 6. Return the follow_id and pending status.
	return e.JSON(http.StatusOK, map[string]any{
		"follow_id": rec.Id,
		"status":    "pending",
	})
}

// ---------------------------------------------------------------------------
// FederationDisconnect handler and pure routing helper
// ---------------------------------------------------------------------------

// disconnectAction returns "delete" for an outbound follow (followerID ==
// localID) and "reject" for an inbound one.
func disconnectAction(followerID, localID string) string {
	if followerID == localID {
		return "delete"
	}
	return "reject"
}

// errNotInstanceFollow is returned by disconnectPeer when the follows record
// does not involve the local instance actor.
var errNotInstanceFollow = errors.New("follow does not involve the local instance actor")

// findFollowBetween returns the follows record from followerID to followeeID,
// or (nil, nil) when none exists.
func findFollowBetween(app core.App, followerID, followeeID string) (*core.Record, error) {
	rec, err := app.FindFirstRecordByFilter(
		"follows",
		"follower={:follower} && followee={:followee}",
		dbx.Params{"follower": followerID, "followee": followeeID},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rec, nil
}

// disconnectPeer ends the relationship with the peer of follow, a follows record
// involving the local instance actor. It only writes follows records; the
// follows hooks deliver the Undo and Reject.
//
//   - A non-accepted outbound request is deleted, a pending inbound request is
//     rejected, and an already rejected inbound request is left alone.
//   - An accepted record ends every accepted relationship with that peer: the
//     accepted outbound row is deleted and the accepted inbound row is rejected.
//
// Inbound rows are never deleted. Steps that are already done are skipped, and
// received content is kept.
func disconnectPeer(app core.App, localID string, follow *core.Record) error {
	if follow.GetString("follower") != localID && follow.GetString("followee") != localID {
		return errNotInstanceFollow
	}
	outbound := disconnectAction(follow.GetString("follower"), localID) == "delete"
	remoteID := follow.GetString("follower")
	if outbound {
		remoteID = follow.GetString("followee")
	}

	status := follow.GetString("status")

	if status != "accepted" {
		if outbound {
			if err := app.Delete(follow); err != nil {
				return fmt.Errorf("delete follow: %w", err)
			}
			return nil
		}
		if status == "rejected" {
			return nil
		}
		follow.Set("status", "rejected")
		if err := app.Save(follow); err != nil {
			return fmt.Errorf("save follow rejected: %w", err)
		}
		return nil
	}

	outRec, err := findFollowBetween(app, localID, remoteID)
	if err != nil {
		return fmt.Errorf("find outbound follow: %w", err)
	}
	inRec, err := findFollowBetween(app, remoteID, localID)
	if err != nil {
		return fmt.Errorf("find inbound follow: %w", err)
	}

	if outRec != nil && outRec.GetString("status") == "accepted" {
		if err := app.Delete(outRec); err != nil {
			return fmt.Errorf("delete outbound follow: %w", err)
		}
	}
	if inRec != nil && inRec.GetString("status") == "accepted" {
		inRec.Set("status", "rejected")
		if err := app.Save(inRec); err != nil {
			return fmt.Errorf("save inbound follow rejected: %w", err)
		}
	}
	return nil
}

// FederationDisconnect handles POST /federation/disconnect/:id and delegates to
// disconnectPeer. Records that do not involve the local instance actor are
// refused with 400.
func FederationDisconnect(e *core.RequestEvent) error {
	// 1. Auth guard — must be first.
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

	// 5. End the relationship(s).
	if err := disconnectPeer(e.App, localActor.Id, follow); err != nil {
		if errors.Is(err, errNotInstanceFollow) {
			return e.BadRequestError("follow does not involve the local instance actor", err)
		}
		return fmt.Errorf("disconnect peer: %w", err)
	}

	return e.JSON(http.StatusOK, map[string]any{"status": "ok"})
}

// ---------------------------------------------------------------------------
// FederationPeers handler and pure folding helper
// ---------------------------------------------------------------------------

// followInput is the subset of a follows record buildPeerEntries needs.
type followInput struct {
	ID       string
	Follower string
	Followee string
	Status   string
}

// buildPeerEntries returns one peerEntry per follows record. An outbound and an
// inbound record for the same remote actor are merged into one "mutual" entry
// only when both are accepted; its FollowID is the outbound and its
// InboundFollowID the inbound record id.
//
// domainOf resolves an actor id to its domain. The result is sorted by domain,
// direction and follow id.
func buildPeerEntries(
	outbound []followInput,
	inbound []followInput,
	localID string,
	domainOf func(actorID string) string,
) []peerEntry {
	display := func(remoteID string) string {
		if d := domainOf(remoteID); d != "" {
			return d
		}
		return remoteID // fallback
	}

	// Index inbound records by remote actor id (the follower).
	inboundByRemote := make(map[string][]int, len(inbound))
	for i, f := range inbound {
		inboundByRemote[f.Follower] = append(inboundByRemote[f.Follower], i)
	}
	consumed := make(map[int]bool)

	result := make([]peerEntry, 0, len(outbound)+len(inbound))

	for _, f := range outbound {
		remoteID := f.Followee
		entry := peerEntry{
			FollowID:  f.ID,
			Direction: "outbound",
			Status:    f.Status,
			Domain:    display(remoteID),
		}
		if f.Status == "accepted" {
			for _, idx := range inboundByRemote[remoteID] {
				if consumed[idx] || inbound[idx].Status != "accepted" {
					continue
				}
				consumed[idx] = true
				entry.Direction = "mutual"
				entry.InboundFollowID = inbound[idx].ID
				break
			}
		}
		result = append(result, entry)
	}

	for i, f := range inbound {
		if consumed[i] {
			continue
		}
		result = append(result, peerEntry{
			FollowID:  f.ID,
			Direction: "inbound",
			Status:    f.Status,
			Domain:    display(f.Follower),
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Domain != result[j].Domain {
			return result[i].Domain < result[j].Domain
		}
		if result[i].Direction != result[j].Direction {
			return result[i].Direction < result[j].Direction
		}
		return result[i].FollowID < result[j].FollowID
	})
	return result
}

// FederationPeers handles GET /federation/peers.
//
// Returns the peer entries built by buildPeerEntries.
func FederationPeers(e *core.RequestEvent) error {
	// 1. Auth guard — must be first.
	if !e.HasSuperuserAuth() {
		return e.UnauthorizedError("superuser authentication required", nil)
	}

	// 2. Look up local instance actor.
	localActor, err := findLocalInstanceActor(e.App)
	if err != nil {
		return fmt.Errorf("local instance actor not found: %w", err)
	}

	// 3. Query outbound follows (local is follower).
	// A query error is a real failure, not an empty list.
	outboundRecords, err := e.App.FindRecordsByFilter(
		"follows",
		"follower={:local}",
		"-created",
		-1,
		0,
		dbx.Params{"local": localActor.Id},
	)
	if err != nil {
		return fmt.Errorf("query outbound follows: %w", err)
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
		return fmt.Errorf("query inbound follows: %w", err)
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

// setFollowStatus sets the status of an inbound follows record (the local
// instance is the followee). The follows hooks deliver the Accept or Reject.
func setFollowStatus(app core.App, follow *core.Record, status, localID string) error {
	// Approve and reject only apply to inbound follows.
	if follow.GetString("followee") != localID {
		return fmt.Errorf("not an inbound follow")
	}

	follow.Set("status", status)
	return app.Save(follow)
}

// FederationApprove handles POST /federation/approve/:id and accepts an inbound
// follow.
func FederationApprove(e *core.RequestEvent) error {
	// 1. Auth guard — must be first.
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
	// Direction errors are 400, save errors 500.
	if err := setFollowStatus(e.App, follow, "accepted", localActor.Id); err != nil {
		if err.Error() == "not an inbound follow" {
			return e.BadRequestError("not an inbound follow", nil)
		}
		return fmt.Errorf("save follow status: %w", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"follow_id": follow.Id,
		"status":    "accepted",
	})
}

// FederationReject handles POST /federation/reject/:id and rejects an inbound
// follow.
func FederationReject(e *core.RequestEvent) error {
	// 1. Auth guard — must be first.
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
	// Direction errors are 400, save errors 500.
	if err := setFollowStatus(e.App, follow, "rejected", localActor.Id); err != nil {
		if err.Error() == "not an inbound follow" {
			return e.BadRequestError("not an inbound follow", nil)
		}
		return fmt.Errorf("save follow status: %w", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"follow_id": follow.Id,
		"status":    "rejected",
	})
}

// ---------------------------------------------------------------------------
// FederationDiscover handler
// ---------------------------------------------------------------------------

// FederationDiscover handles POST /federation/discover with { "url": "<remote>" }.
// It verifies via NodeInfo that the URL is a Wanderer instance, refuses the local
// instance and already connected peers, refreshes the remote instance actor and
// returns { actor_id, domain, version, user_count, trail_count }.
func FederationDiscover(e *core.RequestEvent) error {
	// 1. Auth guard — must be first.
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

	// 3. Fetch NodeInfo 2.1 URL via the JRD discovery document.
	client := util.NewSafeURLClient(10*time.Second, nil)
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

	// A non-200 response is not a valid NodeInfo document.
	if niResp.StatusCode != http.StatusOK {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "unreachable"})
	}

	// 4b. Decode and verify Wanderer identity.
	var ni nodeInfo21
	if err := json.NewDecoder(io.LimitReader(niResp.Body, 64*1024)).Decode(&ni); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "not a Wanderer instance"})
	}
	if ni.Software.Name != "wanderer" {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "not a Wanderer instance"})
	}

	// 5. Derive actor IRI and run self-follow guard.
	parsedURL, err := url.Parse(body.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "unreachable"})
	}
	actorIRI := fmt.Sprintf("%s://%s/api/v1/activitypub/instance", parsedURL.Scheme, parsedURL.Host)

	if util.IsLocalIRI(actorIRI) {
		return e.JSON(http.StatusBadRequest, map[string]any{"error": "resolves to local instance"})
	}

	// 6. Already-connected check.
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
		// Check for a follow in either direction; rejected follows don't count.
		_, followErr := e.App.FindFirstRecordByFilter(
			"follows",
			"(follower={:l} && followee={:r} && status!='rejected') || (follower={:r} && followee={:l} && status!='rejected')",
			dbx.Params{"l": localActor.Id, "r": existingActor.Id},
		)
		if followErr == nil {
			// A follow record was found → already connected.
			return e.JSON(http.StatusBadRequest, map[string]any{"error": "already connected"})
		}

		// 7. Cache bypass: clear last_fetched so GetActorByIRI re-fetches.
		existingActor.Set("last_fetched", time.Time{})
		_ = e.App.Save(existingActor)
	}

	// 7b. Fetch or create the remote actor record via GetActorByIRI.
	actor, err := federation.GetActorByIRI(e.App, context.Background(), actorIRI, false)
	if err != nil {
		return e.JSON(http.StatusBadGateway, map[string]any{"error": "unreachable"})
	}

	// 8. Build the preview card response.
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
