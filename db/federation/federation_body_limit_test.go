package federation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"pocketbase/util"
	"strings"
	"testing"
)

type boundedFederationBody struct {
	io.Reader
	read   int64
	closed bool
}

func (b *boundedFederationBody) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.read += int64(n)
	return n, e
}
func (b *boundedFederationBody) Close() error { b.closed = true; return nil }

type boundedFederationTransport struct {
	body   *boundedFederationBody
	status int
}

func (t boundedFederationTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: t.status, ContentLength: -1, Header: make(http.Header), Body: t.body, Request: r}, nil
}

func TestFederationFetchersBoundCompleteBodies(t *testing.T) {
	for _, kind := range []string{"actor", "collection", "webfinger"} {
		for _, oversize := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/oversize=%v", kind, oversize), func(t *testing.T) {
				t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
				app := setupActorDeleteTestApp(t)
				iri := "https://remote.example/users/alice"
				limit := util.FederationActorBodyLimit
				document := fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"alice","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"origin-key"}}`, iri, iri+"/inbox", iri+"/outbox", iri+"/main-key", iri)
				if kind == "collection" {
					document = `{"type":"OrderedCollectionPage","orderedItems":[]}`
					limit = util.FederationObjectBodyLimit
				}
				if kind == "webfinger" {
					document = fmt.Sprintf(`{"links":[{"rel":"self","href":%q}]}`, iri)
					limit = util.WebfingerBodyLimit
				}
				if oversize {
					document += strings.Repeat(" ", int(limit))
				}
				body := &boundedFederationBody{Reader: strings.NewReader(document)}
				client := &http.Client{Transport: boundedFederationTransport{body: body, status: 200}}
				originalActor, originalCollection, originalWebfinger := newActorFetchHTTPClient, newHTTPClient, newWebfingerHTTPClient
				newActorFetchHTTPClient = func() *http.Client { return client }
				newHTTPClient = func() *http.Client { return client }
				newWebfingerHTTPClient = func() *http.Client { return client }
				t.Cleanup(func() {
					newActorFetchHTTPClient = originalActor
					newHTTPClient = originalCollection
					newWebfingerHTTPClient = originalWebfinger
				})
				var err error
				switch kind {
				case "actor":
					_, _, _, err = fetchRemoteActor(app, context.Background(), iri, false)
				case "collection":
					_, err = FetchCollection(app, context.Background(), iri+"/followers")
				case "webfinger":
					_, err = iriFromHandle(context.Background(), "remote.example", "alice")
				}
				if !body.closed {
					t.Fatal("response was not closed")
				}
				if oversize {
					if !errors.Is(err, util.ErrFederationBodyTooLarge) || body.read != limit+1 {
						t.Fatalf("oversize read=%d err=%v", body.read, err)
					}
				} else if err != nil {
					t.Fatalf("small body refused: %v", err)
				}
			})
		}
	}
}

func TestFederationFetchersCloseErrorResponseBodies(t *testing.T) {
	for _, kind := range []string{"actor", "collection", "webfinger"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
			app := setupActorDeleteTestApp(t)
			body := &boundedFederationBody{Reader: strings.NewReader("unavailable")}
			client := &http.Client{Transport: boundedFederationTransport{body: body, status: 503}}
			originalActor, originalCollection, originalWebfinger := newActorFetchHTTPClient, newHTTPClient, newWebfingerHTTPClient
			newActorFetchHTTPClient = func() *http.Client { return client }
			newHTTPClient = func() *http.Client { return client }
			newWebfingerHTTPClient = func() *http.Client { return client }
			t.Cleanup(func() {
				newActorFetchHTTPClient = originalActor
				newHTTPClient = originalCollection
				newWebfingerHTTPClient = originalWebfinger
			})
			var err error
			switch kind {
			case "actor":
				_, _, _, err = fetchRemoteActor(app, context.Background(), "https://remote.example/users/alice", false)
			case "collection":
				_, err = FetchCollection(app, context.Background(), "https://remote.example/followers")
			case "webfinger":
				_, err = iriFromHandle(context.Background(), "remote.example", "alice")
			}
			if err == nil || !body.closed {
				t.Fatalf("error response leaked: err=%v closed=%v", err, body.closed)
			}
		})
	}
}
