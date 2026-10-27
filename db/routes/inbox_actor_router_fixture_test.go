package routes

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/apis"
)

func knownKeyInboxRouter(t *testing.T, w *knownKeyWorld) http.Handler {
	t.Helper()
	r, err := apis.NewRouter(w.app)
	if err != nil {
		t.Fatal(err)
	}
	r.POST("/inbox", ActivitypubActivityProcess)
	mux, err := r.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return mux
}

func (w *knownKeyWorld) inboxRouterRequest(t *testing.T, iri string, key *rsa.PrivateKey, ctx context.Context) *http.Request {
	t.Helper()
	body, err := json.Marshal(pub.Activity{ID: pub.IRI(fmt.Sprintf("https://remote.example/activities/%d", w.requests.Add(1))), Type: pub.DeleteType, Actor: pub.IRI(iri), Object: pub.IRI(iri)})
	if err != nil {
		t.Fatal(err)
	}
	path := strings.TrimPrefix(w.recipient.GetString("inbox"), "https://example.com")
	request := httptest.NewRequest(http.MethodPost, "https://example.com"+path, bytes.NewReader(body))
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	request.Header.Set("X-Internal-Secret", "known-key-test-secret")
	request.Header.Set("X-Forwarded-Path", path)
	request.Header.Set("Content-Type", "application/activity+json")
	request.Header.Set("Date", strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"))
	request.Header.Set("Host", "example.com")
	if key != nil {
		signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}, httpsig.Signature, 60)
		if err != nil {
			t.Fatal(err)
		}
		if err := signer.SignRequest(key, iri+"#main-key", request, body); err != nil {
			t.Fatal(err)
		}
	}
	// The signature is checked against the forwarded public inbox path.
	// PocketBase receives the same body and headers at its internal route.
	request.URL.Path = "/inbox"
	return request
}

func knownKeyRouterStatus(handler http.Handler, request *http.Request) int {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func waitForInboxActorReloads(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if countInboxActorReloadWaiters() >= count {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d deliveries did not reach the actor refresh", count)
		}
		time.Sleep(time.Millisecond)
	}
}

// A caller waiting in the refresh select has registered with the shared call.
func countInboxActorReloadWaiters() int {
	buffer := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buffer, true)
		if n < len(buffer) {
			buffer = buffer[:n]
			break
		}
		buffer = make([]byte, 2*len(buffer))
	}
	callers := 0
	for _, stack := range strings.Split(string(buffer), "\n\n") {
		lines := strings.SplitN(stack, "\n", 3)
		if len(lines) >= 2 && strings.HasSuffix(lines[0], " [select]:") &&
			strings.HasPrefix(lines[1], "pocketbase/routes.reloadKnownInboxActor(") {
			callers++
		}
	}
	return callers
}
