package federation

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var (
	errNotAcceptedPeer         = errors.New("signer host has no accepted peer relationship")
	errLifecycleNotForInstance = errors.New("lifecycle activity does not concern the instance actor")
	errFollowerNotInstance     = errors.New("instance follows must come from an instance actor")
)

// peerActor is a remote actor with an accepted follow relationship to or from
// the local instance actor.
type peerActor struct {
	ID    string
	IRI   string
	Inbox string
}

// acceptedPeerActors returns every remote instance actor with an accepted
// follow to or from the local instance actor. It defines "connected peer" for
// both outgoing fan-out and the inbound gate. Returns (nil, nil) when the local
// instance actor has not been seeded.
func acceptedPeerActors(app core.App) ([]peerActor, error) {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return nil, fmt.Errorf("ORIGIN not set")
	}
	instanceActor, err := app.FindFirstRecordByData("activitypub_actors", "iri", origin+"/api/v1/activitypub/instance")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	queries := []struct{ joinOn, where string }{
		{"f.follower = aa.id", "f.followee = {:instance} AND f.status = 'accepted' AND aa.is_local = false AND aa.actor_type = 'instance'"},
		{"f.followee = aa.id", "f.follower = {:instance} AND f.status = 'accepted' AND aa.is_local = false AND aa.actor_type = 'instance'"},
	}

	var peers []peerActor
	seen := map[string]struct{}{}
	for _, q := range queries {
		rows, err := app.DB().
			Select("aa.id", "aa.iri", "aa.inbox").
			From("follows f").
			InnerJoin("activitypub_actors aa", dbx.NewExp(q.joinOn)).
			Where(dbx.NewExp(q.where, dbx.Params{"instance": instanceActor.Id})).
			Rows()
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var p peerActor
			if err := rows.Scan(&p.ID, &p.IRI, &p.Inbox); err != nil {
				rows.Close()
				return nil, err
			}
			if _, dup := seen[p.ID]; dup {
				continue
			}
			seen[p.ID] = struct{}{}
			peers = append(peers, p)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return peers, nil
}

// IsAcceptedPeerHost reports whether the signer IRI's host (case-insensitive,
// port included) equals the host of an accepted peer actor.
func IsAcceptedPeerHost(app core.App, signerIRI string) (bool, error) {
	u, err := url.Parse(signerIRI)
	if err != nil || u.Host == "" {
		return false, nil
	}
	peers, err := acceptedPeerActors(app)
	if err != nil {
		return false, err
	}
	for _, p := range peers {
		if sameHost(signerIRI, p.IRI) {
			return true, nil
		}
	}
	return false, nil
}

// isLifecycleActivity is true for the activities that establish or end a
// connection: Follow, Accept, Reject and Undo{Follow}.
func isLifecycleActivity(activity pub.Activity) bool {
	switch activity.Type {
	case pub.FollowType, pub.AcceptType, pub.RejectType:
		return true
	case pub.UndoType:
		return activity.Object != nil && activity.Object.GetType() == pub.FollowType
	}
	return false
}

// lifecycleTargetsInstance reports whether a lifecycle activity concerns the
// local instance actor.
func lifecycleTargetsInstance(activity pub.Activity, instanceIRI string) bool {
	switch activity.Type {
	case pub.FollowType:
		return activity.Object != nil && activity.Object.GetID().String() == instanceIRI
	case pub.AcceptType, pub.RejectType:
		follow, ok := activity.Object.(*pub.Activity)
		if !ok || follow == nil || follow.Type != pub.FollowType || follow.Actor == nil {
			return false
		}
		return follow.Actor.GetID().String() == instanceIRI
	case pub.UndoType:
		follow, ok := activity.Object.(*pub.Activity)
		if !ok || follow == nil || follow.Type != pub.FollowType || follow.Object == nil {
			return false
		}
		return follow.Object.GetID().String() == instanceIRI
	}
	return false
}

// authorizeInstanceActivity decides whether a verified activity may be
// processed by the instance inbox. Lifecycle activities must concern the
// instance actor; everything else requires the signer's host to be an accepted
// peer. signer must be the record whose key verified the signature.
func authorizeInstanceActivity(app core.App, signer *core.Record, activity pub.Activity, instanceIRI string) error {
	if isLifecycleActivity(activity) {
		if lifecycleTargetsInstance(activity, instanceIRI) {
			return nil
		}
		return errLifecycleNotForInstance
	}
	ok, err := IsAcceptedPeerHost(app, signer.GetString("iri"))
	if err != nil {
		return fmt.Errorf("peer gate: %w", err)
	}
	if !ok {
		return errNotAcceptedPeer
	}
	return nil
}
