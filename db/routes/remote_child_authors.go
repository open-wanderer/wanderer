package routes

import (
	"context"
	"fmt"
	"time"

	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

type preparedRemoteChildAuthorsKey struct{}

// Origin-owned contributions share a bounded step while each actor may use
// the ordinary response time of the document's own host.
var sameOriginAuthorTimeout = 10 * time.Second

var sameOriginAuthorBudget = 10 * time.Second

// Confirm and persist origin-owned child authors before the record transaction.
// A temporarily unavailable contribution can be skipped without rolling back
// confirmed keys and the remaining readable children.
func prepareRemoteChildAuthors(app core.App, ctx context.Context, data map[string]any, origin, relation, kind string) context.Context {
	expanded, _ := data["expand"].(map[string]any)
	children, _ := expanded[relation].([]any)
	references := []string{}
	for _, child := range children {
		raw, _ := child.(map[string]any)
		id, _ := raw["id"].(string)
		if id == "" {
			continue
		}
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/%s/%s", origin, kind, id)
		}
		if util.IsLocalIRI(iri) || !remoteSyncIRIMatchesOrigin(iri, origin) {
			continue
		}
		expanded, _ := raw["expand"].(map[string]any)
		actor, _ := expanded["author"].(map[string]any)
		authorIRI, _ := actor["iri"].(string)
		if util.IsLocalIRI(authorIRI) || !remoteSyncIRIMatchesOrigin(authorIRI, iri) {
			continue
		}
		references = append(references, authorIRI)
	}
	// Look up contributors without a confirmed stored identity first. Known
	// contributors follow in document order and can use the fallback below.
	unknown, known := []string{}, []string{}
	for _, iri := range references {
		if storedConfirmedChildAuthor(app, iri) == nil {
			unknown = append(unknown, iri)
		} else {
			known = append(known, iri)
		}
	}
	authors := resolveRemoteAuthorsWithin(app, ctx, append(unknown, known...), sameOriginAuthorTimeout, sameOriginAuthorBudget)
	// Include confirmed contributors even when the step ended before their
	// lookup started. A stored remote identity needs no freshness pre-pass.
	for _, iri := range references {
		if authors[iri] != nil {
			continue
		}
		if stored := storedConfirmedChildAuthor(app, iri); stored != nil {
			authors[iri] = stored
		}
	}
	return context.WithValue(ctx, preparedRemoteChildAuthorsKey{}, authors)
}

func storedConfirmedChildAuthor(app core.App, iri string) *core.Record {
	stored, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
	if err == nil && !stored.GetBool("is_local") && stored.GetString("iri") == iri && federation.CachedActorIdentityMatchesOrigin(stored) {
		return stored
	}
	return nil
}
