package routes

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"pocketbase/federation"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"pocketbase/hooks"
)

func startRemoteListMeili(t *testing.T) meilisearch.ServiceManager {
	t.Helper()
	binary := os.Getenv("WANDERER_TEST_MEILISEARCH")
	if binary == "" {
		binary = filepath.Join("..", "..", "search", "meilisearch")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skipf("no Meilisearch binary at %s; set WANDERER_TEST_MEILISEARCH", binary)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()
	const key = "remote-list-test-master-key-0123456789"
	cmd := exec.Command(binary, "--master-key", key, "--http-addr", addr, "--db-path", filepath.Join(t.TempDir(), "data.ms"), "--no-analytics")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	host := "http://" + addr
	for deadline := time.Now().Add(30 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		response, err := http.Get(host + "/health")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 200 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("Meilisearch did not become ready")
		}
	}
	return meilisearch.New(host, meilisearch.WithAPIKey(key))
}

func remoteListIndexedDocument(t *testing.T, client meilisearch.ServiceManager, index, id string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	for {
		tasks, err := client.GetTasks(&meilisearch.TasksQuery{Statuses: []meilisearch.TaskStatus{meilisearch.TaskStatusEnqueued, meilisearch.TaskStatusProcessing}, Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if tasks.Total == 0 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("search indexing did not finish")
		case <-time.After(20 * time.Millisecond):
		}
	}
	document := map[string]any{}
	if err := client.Index(index).GetDocument(id, nil, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func TestRemoteListReferencesUpdateTheRealSearchIndex(t *testing.T) {
	client := startRemoteListMeili(t)
	f := newListViewFixture(t)
	f.app.OnRecordAfterCreateSuccess("trails").BindFunc(hooks.CreateTrailHandler(client))
	f.app.OnRecordAfterUpdateSuccess("trails").BindFunc(hooks.UpdateTrailHandler(client))
	f.app.OnRecordAfterCreateSuccess("lists").BindFunc(hooks.CreateListHandler(client))
	f.app.OnRecordAfterUpdateSuccess("lists").BindFunc(hooks.UpdateListHandler(client))
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	f.actorKey = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	t.Cleanup(federation.SetActorFetchClientForTesting(func() *http.Client { return &http.Client{Transport: remoteSyncTransport{roundTrip: f.serve}} }))
	ben, err := f.app.FindFirstRecordByData("activitypub_actors", "iri", "https://trail.example/api/v1/activitypub/user/ben")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.app.Delete(ben); err != nil {
		t.Fatal(err)
	}
	own := func(i int) any {
		return map[string]any{"id": fmt.Sprintf("own-%d", i), "iri": fmt.Sprintf("https://origin.example/api/v1/trail/own-%d", i), "name": fmt.Sprintf("Own %d", i), "public": true, "distance": 9700.0, "expand": map[string]any{"author": map[string]any{"iri": "https://origin.example/api/v1/activitypub/user/anna"}}}
	}
	first, second := f.trail("trail-one", "Origin", "Forest"), f.trail("trail-two", "Other", "Meadow")
	f.trails["trail.example/api/v1/trail/trail-one"]["distance"] = 12000.0
	f.trails["trail.example/api/v1/trail/trail-two"]["distance"] = 8000.0
	f.list(own(1), first, own(2), second, own(3))
	trails := f.view()
	if len(trails) != 5 {
		t.Fatalf("list trails=%d", len(trails))
	}
	trail := trails[1]
	list, err := f.app.FindFirstRecordByData("lists", "iri", "https://origin.example/api/v1/list/list-id")
	if err != nil {
		t.Fatal(err)
	}
	check := func(name, tag, line string, distance, total float64) {
		t.Helper()
		document := remoteListIndexedDocument(t, client, "trails", trail.Id)
		tags := []string{}
		for _, v := range document["tags"].([]any) {
			tags = append(tags, v.(string))
		}
		if document["name"] != name || !slices.Equal(tags, []string{tag}) || document["distance"] != distance || document["polyline"] != line || document["completed"] != true {
			t.Fatalf("trail search document=%v", document)
		}
		document = remoteListIndexedDocument(t, client, "lists", list.Id)
		if document["distance"] != total || document["trails"] != float64(5) {
			t.Fatalf("list search document=%v", document)
		}
	}
	if !slices.Equal(listTrailNames(trails), []string{"Own 1", "Origin", "Own 2", "Other", "Own 3"}) {
		t.Fatalf("list order=%v", listTrailNames(trails))
	}
	objectRequests, authorRequests := 0, 0
	f.lock.Lock()
	for _, request := range f.requests {
		if strings.HasPrefix(request, "trail.example/api/v1/trail/") {
			objectRequests++
		}
		if strings.HasPrefix(request, "trail.example/api/v1/activitypub/user/") {
			authorRequests++
		}
	}
	f.lock.Unlock()
	if objectRequests != 2 || authorRequests != 1 {
		t.Fatalf("origin requests: objects=%d authors=%d", objectRequests, authorRequests)
	}
	check("Origin", "Forest", "", 12000, 49100)
	f.lock.Lock()
	raw := f.trails["trail.example/api/v1/trail/trail-one"]
	raw["name"] = "Renamed"
	raw["distance"] = 2000.0
	raw["polyline"] = "refreshed-line"
	raw["expand"].(map[string]any)["tags"] = []any{map[string]any{"name": "Ridge"}}
	f.lock.Unlock()
	f.age()
	f.view()
	check("Renamed", "Ridge", "refreshed-line", 2000, 39100)
}
