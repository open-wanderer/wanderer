package routes

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

func TestRemoteListReferencesUseThePublicHTTPClient(t *testing.T) {
	f := newListViewFixture(t)
	var connections atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("private destination was contacted") }))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	defer server.Close()
	previous := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = util.SafeHTTPClient
	defer func() { newRemoteSyncHTTPClient = previous }()
	collection, err := f.app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	list := core.NewRecord(collection)
	list.Set("iri", "https://origin.example/api/v1/list/list-id")
	refs := []any{map[string]any{"iri": server.URL + "/api/v1/trail/trail-one"},
		map[string]any{"iri": "http://10.0.0.1/api/v1/trail/trail-one"},
		map[string]any{"iri": "http://172.16.0.1/api/v1/trail/trail-one"},
		map[string]any{"iri": "http://192.168.1.1/api/v1/trail/trail-one"},
		map[string]any{"iri": "http://[::1]/api/v1/trail/trail-one"}}
	t.Setenv("ORIGIN", "https://www.local.example:8443")
	for _, host := range []string{"local.example", "LOCAL.example", "local.example.", "LOCAL.example.:8443", "local.example:8080", "www.local.example", "WWW.LOCAL.example.", "www.local.example.:9080"} {
		for _, scheme := range []string{"http", "https"} {
			iri := scheme + "://" + host + "/api/v1/trail/trail-one"
			if !remoteListReferenceIsLocal(iri) {
				t.Fatalf("own host not recognized: %s", iri)
			}
			refs = append(refs, map[string]any{"iri": iri})
		}
	}
	ctx := prepareRemoteListReferences(f.app, t.Context(), list, map[string]any{"expand": map[string]any{"trails": refs}}, "https://origin.example")
	state := remoteListReferenceStateOf(ctx)
	if len(state.confirmed) != 0 || len(state.refreshed) != 0 || connections.Load() != 0 {
		t.Fatalf("private references loaded: confirmed=%d connections=%d", len(state.confirmed), connections.Load())
	}
	count, err := f.app.CountRecords("trails")
	if err != nil || count != 0 {
		t.Fatalf("trail records=%d err=%v", count, err)
	}
	for _, raw := range refs {
		iri := raw.(map[string]any)["iri"].(string)
		if strings.Contains(iri, "local.example") && remoteListReferenceFailed(f.app, iri) {
			t.Fatalf("own reference fetched: %s", iri)
		}
	}
}

func TestRemoteListReferencesCloseIdleObjectConnections(t *testing.T) {
	f := newListReferenceFixture(t)
	actor := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
	const iri = "http://third.example/api/v1/trail/trail-id"
	var open, requests, dials atomic.Int32
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if err := json.NewEncoder(w).Encode(map[string]any{"id": "trail-id", "iri": iri, "name": "Origin", "public": true,
			"expand": map[string]any{"author": map[string]any{"iri": actor.GetString("iri")}}}); err != nil {
			t.Error(err)
		}
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		switch state {
		case http.StateNew:
			open.Add(1)
		case http.StateClosed, http.StateHijacked:
			open.Add(-1)
		}
	}
	server.Start()
	defer server.Close()
	previous := newRemoteSyncHTTPClient
	defer func() { newRemoteSyncHTTPClient = previous }()
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Timeout: time.Second, Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				dials.Add(1)
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			},
		}}
	}
	// Each sync owns a real transport without an idle timeout, as in production.
	data := map[string]any{"expand": map[string]any{"trails": []any{map[string]any{"iri": iri}}}}
	for index := range 10 {
		ctx := prepareRemoteListReferences(f.app, t.Context(), f.list, data, "https://remote.example")
		if len(remoteListReferenceStateOf(ctx).confirmed) != 1 {
			t.Fatalf("sync %d did not confirm its reference", index)
		}
		for deadline := time.Now().Add(time.Second); open.Load() != 0; time.Sleep(time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("sync %d left %d idle object connections open", index, open.Load())
			}
		}
	}
	if requests.Load() != 10 || dials.Load() != 10 {
		t.Fatalf("requests=%d dials=%d", requests.Load(), dials.Load())
	}
}
