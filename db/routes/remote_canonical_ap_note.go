package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

func canonicalAPNoteURL(iri string) (*url.URL, error) {
	parsed, err := url.Parse(iri)
	if err != nil || len(iri) > 2048 || !remoteSyncIRIMatchesOrigin(iri, iri) || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.EscapedPath() != parsed.Path || parsed.Path == "" || strings.HasPrefix(parsed.Path, "/api/v1/comment") {
		return nil, fmt.Errorf("invalid ActivityPub note reference")
	}
	return parsed, nil
}

// Public replies from other ActivityPub implementations have opaque object
// paths. Their object, author and reply target must be confirmed by the host
// serving that exact Note; the trail origin's cached text is never imported.
func fetchCanonicalAPNoteComment(app core.App, ctx context.Context, client *http.Client, trail *core.Record, job canonicalCommentRequest) (canonicalRemoteComment, bool) {
	reject := func() (canonicalRemoteComment, bool) {
		canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
		return canonicalRemoteComment{}, false
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, job.url.String(), nil)
	if err != nil {
		return reject()
	}
	request.Header.Set("Accept", "application/activity+json, application/ld+json; profile=\"https://www.w3.org/ns/activitystreams\"")
	response, err := client.Do(request)
	if errors.Is(err, util.ErrRateLimited) {
		return canonicalRemoteComment{}, false
	}
	if err != nil {
		return reject()
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
			return canonicalGoneComment(job)
		}
		return reject()
	}
	var note map[string]any
	if err := decodeRemoteFederationJSON(ctx, response.Body, canonicalRemoteBodyLimit, &note); err != nil {
		return reject()
	}
	identity, _ := note["id"].(string)
	kind, _ := note["type"].(string)
	parentIRI := singleActivityPubIRI(note["inReplyTo"])
	authorIRI := singleActivityPubIRI(note["attributedTo"])
	content, _ := note["content"].(string)
	if kind != "Note" || !sameRemoteSyncObjectIRI(job.iri, identity) || !sameRemoteSyncObjectIRI(trail.GetString("iri"), parentIRI) ||
		content == "" || !activityPubNoteIsPublic(note) || util.IsLocalIRI(authorIRI) || !remoteSyncIRIMatchesOrigin(authorIRI, job.iri) {
		return reject()
	}
	return canonicalRemoteComment{data: map[string]any{"text": content}, authorIRI: authorIRI, cacheKey: job.cacheKey, updatedHint: job.updated}, true
}

func singleActivityPubIRI(value any) string {
	if values, ok := value.([]any); ok {
		if len(values) != 1 {
			return ""
		}
		value = values[0]
	}
	if iri, ok := value.(string); ok {
		return iri
	}
	if object, ok := value.(map[string]any); ok {
		iri, _ := object["id"].(string)
		return iri
	}
	return ""
}

func activityPubNoteIsPublic(note map[string]any) bool {
	isPublic := func(value any) bool {
		switch singleActivityPubIRI(value) {
		case "https://www.w3.org/ns/activitystreams#Public", "as:Public", "Public":
			return true
		}
		return false
	}
	for _, field := range []string{"to", "cc"} {
		if values, ok := note[field].([]any); ok {
			for _, recipient := range values {
				if isPublic(recipient) {
					return true
				}
			}
		} else if isPublic(note[field]) {
			return true
		}
	}
	return false
}
