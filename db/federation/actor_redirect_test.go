package federation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type actorRedirectTransport struct {
	document  string
	redirects map[string]string
	requests  int
}

func (t *actorRedirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.requests++
	header := make(http.Header)
	status, body := http.StatusOK, t.document
	if redirect, exists := t.redirects[r.URL.String()]; exists {
		status, body = http.StatusFound, ""
		header.Set("Location", redirect)
	}
	return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
}

func TestActorFetchKeepsRedirectsOnOriginalHost(t *testing.T) {
	for _, tc := range []struct {
		name, redirect           string
		accepted, previousReject bool
	}{
		{name: "same-host", redirect: "https://origin.example/actor.json", accepted: true},
		{name: "same-host-case", redirect: "https://ORIGIN.example/actor.json", accepted: true},
		{name: "https-to-http", redirect: "http://origin.example/actor.json"},
		{name: "cross-host", redirect: "https://other.example/actor.json"},
		{name: "other-port", redirect: "https://origin.example:8443/actor.json"},
		{name: "userinfo", redirect: "https://user@origin.example/actor.json"},
		{name: "previous-policy", redirect: "https://origin.example/actor.json", previousReject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
			app := setupActorDeleteTestApp(t)
			requested := "https://origin.example/redirect"
			transport := &actorRedirectTransport{redirects: map[string]string{requested: tc.redirect}, document: fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"carol","inbox":"https://origin.example/inbox","outbox":"https://origin.example/outbox","publicKey":{"id":%q,"owner":%q,"publicKeyPem":"origin-key"}}`, requested, requested+"/main-key", requested)}
			client := &http.Client{Transport: transport}
			policyCalls := 0
			if tc.previousReject {
				client.CheckRedirect = func(*http.Request, []*http.Request) error { policyCalls++; return errors.New("custom redirect policy") }
			}
			original := newActorFetchHTTPClient
			newActorFetchHTTPClient = func() *http.Client { return client }
			t.Cleanup(func() { newActorFetchHTTPClient = original })
			actor, err := GetActorByIRI(app, context.Background(), requested, false)
			count, countErr := app.CountRecords("activitypub_actors")
			if countErr != nil {
				t.Fatal(countErr)
			}
			if tc.accepted {
				if err != nil || actor == nil || count != 1 || transport.requests != 2 {
					t.Fatalf("same-origin redirect refused: actor=%v err=%v count=%d requests=%d", actor, err, count, transport.requests)
				}
			} else if err == nil || count != 0 || transport.requests != 1 {
				t.Fatalf("redirect cached foreign identity: err=%v count=%d requests=%d", err, count, transport.requests)
			}
			if tc.previousReject && policyCalls != 1 {
				t.Fatalf("caller redirect policy was replaced: calls=%d", policyCalls)
			}
			if !tc.previousReject && client.CheckRedirect != nil {
				t.Fatal("shared HTTP client was mutated")
			}
		})
	}
}

func TestActorFetchKeepsHTTPSAfterUpgradeRedirect(t *testing.T) {
	for _, downgrade := range []bool{false, true} {
		t.Run(fmt.Sprintf("downgrade=%v", downgrade), func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
			app := setupActorDeleteTestApp(t)
			requested := "http://remote.example/actor"
			upgraded := "https://remote.example/actor.json"
			document := "https://remote.example/actor"
			redirects := map[string]string{requested: upgraded}
			if downgrade {
				redirects[upgraded] = "http://remote.example/document.json"
			}
			transport := &actorRedirectTransport{
				redirects: redirects,
				document:  fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"alice","inbox":"https://remote.example/inbox","outbox":"https://remote.example/outbox","publicKey":{"id":%q,"owner":%q,"publicKeyPem":"origin-key"}}`, document, document+"#main-key", document),
			}
			original := newActorFetchHTTPClient
			newActorFetchHTTPClient = func() *http.Client { return &http.Client{Transport: transport} }
			t.Cleanup(func() { newActorFetchHTTPClient = original })
			actor, err := GetActorByIRI(app, context.Background(), requested, false)
			count, countErr := app.CountRecords("activitypub_actors")
			if countErr != nil {
				t.Fatal(countErr)
			}
			if transport.requests != 2 {
				t.Fatalf("redirect requests = %d, want 2", transport.requests)
			}
			if downgrade {
				if err == nil || actor != nil || count != 0 {
					t.Fatalf("HTTP redirect cached actor: actor=%v err=%v count=%d", actor, err, count)
				}
			} else if err != nil || actor == nil || count != 1 || actor.GetString("iri") != requested {
				t.Fatalf("HTTPS upgrade failed: actor=%v err=%v count=%d", actor, err, count)
			}
		})
	}
}
