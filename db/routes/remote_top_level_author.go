package routes

import (
	"context"
	"fmt"
	"net/url"
	"pocketbase/federation"
	"pocketbase/util"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

var remoteSyncGetActorByIRI = federation.GetActorByIRI

// A direct fetch from the canonical origin can correct the handle used to
// create an older copy. Both authors must remain remote actors of that
// origin, and the response must identify the exact object being refreshed.
// Older releases also stored copies under a remote actor of another host;
// an identified response replaces that author with the canonical one.
func resolveRemoteTopLevelAuthor(app core.App, ctx context.Context, record *core.Record, data map[string]any) (string, error) {
	iri := record.GetString("iri")
	if util.RecordHasLocalIdentity(app, record) || remoteRecordHasLocalAuthor(app, record) {
		return "", fmt.Errorf("remote sync cannot update a locally authored object")
	}
	stored, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
	if err != nil || stored.GetBool("is_local") {
		return "", fmt.Errorf("remote object stored author does not match its origin")
	}
	storedOnOrigin := remoteSyncIRIMatchesOrigin(stored.GetString("iri"), iri)
	responseIRI, identified := data["iri"]
	canonicalIRI, validIRI := responseIRI.(string)
	// v0.19.x origins answer "iri": "" for their own objects. The exact record
	// id then identifies the requested object.
	if identified && !(validIRI && (sameRemoteSyncObjectIRI(iri, canonicalIRI) ||
		canonicalIRI == "" && remoteResponseIDMatchesObjectIRI(record.Collection().Name, iri, data))) {
		return "", fmt.Errorf("remote response does not identify the requested object")
	}
	expand, _ := data["expand"].(map[string]any)
	expandedAuthor, authorPresent := expand["author"]
	if !authorPresent {
		if !storedOnOrigin {
			return "", fmt.Errorf("remote object stored author does not match its origin")
		}
		if record.IsNew() {
			return "", fmt.Errorf("remote object response is missing its expanded author")
		}
		return stored.Id, nil
	}
	authorMap, ok := expandedAuthor.(map[string]any)
	authorIRI, _ := authorMap["iri"].(string)
	if !ok || !remoteSyncIRIMatchesOrigin(authorIRI, iri) || util.IsLocalIRI(authorIRI) {
		return "", fmt.Errorf("remote object author does not match its origin")
	}
	if !storedOnOrigin && !identified {
		return "", fmt.Errorf("remote object stored author does not match its origin")
	}
	if (record.IsNew() || authorIRI != stored.GetString("iri")) && !identified {
		return "", fmt.Errorf("remote author attribution requires the canonical object identity")
	}
	// The origin confirms the stored author: refreshing that actor adds
	// nothing to the binding and must not decide whether the sync succeeds.
	if storedOnOrigin && authorIRI == stored.GetString("iri") && federation.CachedActorIdentityMatchesOrigin(stored) {
		return stored.Id, nil
	}
	author, err := resolveRemoteActorForSync(app, ctx, authorIRI)
	if err != nil {
		return "", err
	}
	if author.GetBool("is_local") || !remoteSyncIRIMatchesOrigin(author.GetString("iri"), iri) {
		return "", fmt.Errorf("remote object author does not match its origin")
	}
	if !storedOnOrigin {
		app.Logger().Warn("replacing a remote author from another host with the canonical author",
			"iri", iri, "previous_author", stored.GetString("iri"), "author", authorIRI)
	}
	return author.Id, nil
}

// resolveRemoteActorForSync refreshes a remote actor named by an origin. A
// failed refresh falls back to the cached row returned with the error when
// that row is exactly the requested remote actor and its stored identity is
// consistent with its host.
func resolveRemoteActorForSync(app core.App, ctx context.Context, actorIRI string) (*core.Record, error) {
	releaseFetch, err := acquireRemoteSyncFetch(ctx, actorIRI)
	if err != nil {
		return nil, err
	}
	defer releaseFetch()
	actor, err := remoteSyncGetActorByIRI(app, federation.WithActorIdentityLookup(ctx), actorIRI, false)
	if err == nil {
		return actor, nil
	}
	// Another concurrent fetch may have won the unique actor-IRI write.
	// Re-read the committed row before deciding whether the refresh failed.
	if stored, lookupErr := app.FindFirstRecordByData("activitypub_actors", "iri", actorIRI); lookupErr == nil {
		actor = stored
	}
	if actor == nil || actor.Id == "" || actor.GetString("iri") != actorIRI || actor.GetBool("is_local") ||
		!federation.CachedActorIdentityMatchesOrigin(actor) {
		return nil, err
	}
	app.Logger().Warn("using the cached remote actor after a failed refresh", "actor", actorIRI, "error", err)
	return actor, nil
}

// remoteResponseIDMatchesObjectIRI accepts only the canonical
// /api/v1/<kind>/<id> path of the requested object.
func remoteResponseIDMatchesObjectIRI(collection, requested string, data map[string]any) bool {
	kind := map[string]string{"trails": "trail", "lists": "list", "comments": "comment", "summit_logs": "summit-log"}[collection]
	id, _ := data["id"].(string)
	if kind == "" || strings.TrimSpace(id) == "" || strings.ContainsAny(id, "/?#%") {
		return false
	}
	object, err := url.Parse(requested)
	return err == nil && remoteSyncIRIMatchesOrigin(requested, requested) && object.User == nil && object.RawQuery == "" && !object.ForceQuery && object.Fragment == "" &&
		object.EscapedPath() == object.Path && object.Path == "/api/v1/"+kind+"/"+id
}

func sameRemoteSyncObjectIRI(requested, response string) bool {
	if !remoteSyncIRIMatchesOrigin(response, requested) {
		return false
	}
	object, err := url.Parse(requested)
	if err != nil {
		return false
	}
	canonical, err := url.Parse(response)
	return err == nil && object.RawQuery == "" && canonical.RawQuery == "" &&
		object.Fragment == "" && canonical.Fragment == "" &&
		object.EscapedPath() == object.Path && canonical.EscapedPath() == canonical.Path &&
		object.Path == canonical.Path && strings.TrimSpace(object.Path) != ""
}
