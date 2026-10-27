package routes

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

// A self-deletion can be reconciled without a cached key when the actor's
// own exact endpoint independently confirms that it is permanently gone.
// wanderer answers removed accounts with 404. Such a self-deletion is only
// accepted when legacySigned authenticates it with the key stored before the
// cache refresh; nil means only 410 counts.
func reconcileGoneInboxActor(app core.App, ctx context.Context, actor *core.Record, activity pub.Activity, legacySigned func(*core.Record) bool) (bool, error) {
	if actor == nil || actor.GetBool("is_local") || activity.Type != pub.DeleteType || activity.Actor == nil || activity.Object == nil {
		return false, nil
	}
	iri := actor.GetString("iri")
	if util.IsLocalIRI(iri) || activity.Actor.GetID().String() != iri || activity.Object.GetID().String() != iri {
		return false, nil
	}
	target, err := url.Parse(iri)
	if err != nil || !remoteSyncIRIMatchesOrigin(iri, iri) || target.User != nil || target.RawQuery != "" || target.ForceQuery || target.Fragment != "" || target.Path == "" || target.EscapedPath() != target.Path {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	client := *newRemoteSyncHTTPClient()
	client.CheckRedirect = canonicalRemoteRedirect
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, iri, nil)
	if err != nil {
		return false, nil
	}
	request.Header.Set("Accept", "application/activity+json")
	response, err := client.Do(request)
	if err != nil {
		return false, nil
	}
	defer response.Body.Close()
	legacy := response.StatusCode == http.StatusNotFound
	if legacy && (legacySigned == nil || util.ObjectKindFromIRI(iri) != util.ObjectKindActor || !legacySigned(actor)) {
		return false, nil
	}
	if !legacy && response.StatusCode != http.StatusGone {
		return false, nil
	}
	// Recheck the current stored identity before deleting; the refresh or
	// another request may have replaced this row in the meantime.
	stored, err := app.FindRecordById("activitypub_actors", actor.Id)
	if err != nil || stored.GetBool("is_local") || stored.GetString("iri") != iri || util.IsLocalIRI(iri) {
		return false, nil
	}
	if legacy && (strings.TrimSpace(stored.GetString("public_key")) != "" || stored.GetString("legacy_public_key") != actor.GetString("legacy_public_key") || !legacyKeyFetchedAtActorHost(stored)) {
		return false, nil
	}
	return true, util.DeleteRemoteActor(app, stored)
}

// legacySelfDeletionSigned verifies a request with the key an actor had
// before the cache refresh, under the key id wanderer signs with. A key
// confirmed at the origin since then replaces it.
func legacySelfDeletionSigned(app core.App, request *http.Request, body []byte, actor *core.Record) bool {
	key, iri := actor.GetString("legacy_public_key"), actor.GetString("iri")
	if strings.TrimSpace(key) == "" || strings.TrimSpace(actor.GetString("public_key")) != "" || !legacyKeyFetchedAtActorHost(actor) {
		return false
	}
	signature, err := util.CheckRequestSignature(request, body, time.Now())
	if err != nil || signature.KeyID != iri+"#main-key" {
		return false
	}
	verified, err := util.VerifySignature(app, request, key, body, iri)
	return err == nil && verified
}

// Before the cache refresh, the stored domain named the host the document
// was requested from (without "www."), while the IRI was the document's id.
// A key served for another host's identity is never used.
func legacyKeyFetchedAtActorHost(actor *core.Record) bool {
	parsed, err := url.Parse(actor.GetString("iri"))
	domain := strings.TrimSpace(actor.GetString("domain"))
	return err == nil && domain != "" && strings.EqualFold(strings.TrimPrefix(parsed.Hostname(), "www."), domain)
}
