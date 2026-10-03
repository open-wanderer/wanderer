package federation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestCheckActorIDHost(t *testing.T) {
	const requested = "https://peer.example.com/users/x"

	actorWithID := func(id string) *pub.Actor {
		a := &pub.Actor{}
		a.ID = pub.IRI(id)
		return a
	}

	cases := []struct {
		name    string
		actor   *pub.Actor
		wantErr string
	}{
		{"same id", actorWithID("https://peer.example.com/users/x"), ""},
		{"same host different path", actorWithID("https://peer.example.com/@x"), ""},
		{"foreign host", actorWithID("https://other.example.com/users/x"), "host mismatch"},
		{"case-insensitive host", actorWithID("https://PEER.example.com/users/x"), ""},
		{"different port", actorWithID("https://peer.example.com:8443/users/x"), "host"},
		{"nil actor", nil, "actor"},
		{"empty id", actorWithID(""), "actor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkActorIDHost(requested, tc.actor)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestFetchRemoteActorRejectsForeignID(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	defaultClient := newHTTPClient
	newHTTPClient = func() *http.Client { return http.DefaultClient }
	t.Cleanup(func() { newHTTPClient = defaultClient })

	app := newInboxTestApp(t)

	serve := func(t *testing.T, idFor func(srvURL string) string) string {
		var srv *httptest.Server
		srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := idFor(srv.URL)
			w.Header().Set("Content-Type", "application/activity+json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"@context":          "https://www.w3.org/ns/activitystreams",
				"type":              "Application",
				"id":                id,
				"preferredUsername": "instance",
				"inbox":             srv.URL + "/inbox",
				"outbox":            srv.URL + "/outbox",
				"publicKey": map[string]any{
					"id":           id + "#main-key",
					"owner":        id,
					"publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----\n",
				},
			})
		}))
		t.Cleanup(srv.Close)
		return srv.URL
	}

	t.Run("foreign id", func(t *testing.T) {
		url := serve(t, func(string) string { return "https://peer.example.com/users/x" })
		_, _, _, err := fetchRemoteActor(app, context.Background(), url+"/users/x", false)
		if err == nil || !strings.Contains(err.Error(), "host mismatch") {
			t.Fatalf("err = %v, want host mismatch", err)
		}
	})

	t.Run("same host", func(t *testing.T) {
		url := serve(t, func(srv string) string { return srv + "/users/x" })
		actor, _, _, err := fetchRemoteActor(app, context.Background(), url+"/users/x", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if actor == nil {
			t.Fatal("expected actor")
		}
	})
}

func TestFetchRemoteActorRejectsCrossHostRedirect(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	defaultClient := newHTTPClient
	newHTTPClient = func() *http.Client { return http.DefaultClient }
	t.Cleanup(func() { newHTTPClient = defaultClient })

	app := newInboxTestApp(t)

	actorDoc := func(w http.ResponseWriter, srvURL, id string) {
		w.Header().Set("Content-Type", "application/activity+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"@context":          "https://www.w3.org/ns/activitystreams",
			"type":              "Application",
			"id":                id,
			"preferredUsername": "instance",
			"inbox":             srvURL + "/inbox",
			"outbox":            srvURL + "/outbox",
			"publicKey": map[string]any{
				"id":           id + "#main-key",
				"owner":        id,
				"publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----\n",
			},
		})
	}

	t.Run("cross-host redirect", func(t *testing.T) {
		var a, b *httptest.Server
		// B claims A's host in its actor id.
		b = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actorDoc(w, a.URL, a.URL+"/actor")
		}))
		t.Cleanup(b.Close)
		a = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, b.URL+"/actor", http.StatusFound)
		}))
		t.Cleanup(a.Close)

		actor, _, _, err := fetchRemoteActor(app, context.Background(), a.URL+"/redirect", false)
		if err == nil || !strings.Contains(err.Error(), "redirect") {
			t.Fatalf("err = %v, want redirect error", err)
		}
		if actor != nil {
			t.Fatal("no actor must be returned")
		}
	})

	t.Run("same-host redirect", func(t *testing.T) {
		var a *httptest.Server
		a = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/old" {
				http.Redirect(w, r, a.URL+"/actor", http.StatusFound)
				return
			}
			actorDoc(w, a.URL, a.URL+"/actor")
		}))
		t.Cleanup(a.Close)

		actor, _, _, err := fetchRemoteActor(app, context.Background(), a.URL+"/old", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if actor == nil {
			t.Fatal("expected actor")
		}
	})
}
