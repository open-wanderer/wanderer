package routes

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"net/http"
	"testing"
	"time"
)

func TestCanonicalLogFetchCannotTakeOverAnotherOriginAuthorsStoredCopy(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteChildFixture(t, "summit_log")
	owner := f.save(f.actors, map[string]any{"iri": "https://fourth.example/users/owner", "last_fetched": time.Now()})
	canonical := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
	iri := "https://third.example/api/v1/summit-log/copy"
	f.children.Fields.Add(&core.FileField{Name: "photos", MaxSelect: 99, MaxSize: 1024})
	if err := f.app.Save(f.children); err != nil {
		t.Fatal(err)
	}
	stored := f.child(iri, owner.Id, f.trail.Id)
	photo, err := filesystem.NewFileFromBytes([]byte("Original photo"), "preserved.jpg")
	if err != nil {
		t.Fatal(err)
	}
	stored.Set("photos", []*filesystem.File{photo})
	if err := f.app.Save(stored); err != nil {
		t.Fatal(err)
	}
	filenames := stored.GetStringSlice("photos")
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	canonicalFetches := 0
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if request.URL.Host == "remote.example" {
				return richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{"summit_logs_via_trail": []any{map[string]any{"id": "copy", "iri": iri}}}}}.RoundTrip(request)
			}
			canonicalFetches++
			return richTextSyncTransport{map[string]any{"id": "copy", "iri": iri, "text": "Replacement", "expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": canonical.GetString("iri")}}}}.RoundTrip(request)
		}}}
	}
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	current, err := f.app.FindRecordById("summit_logs", stored.Id)
	if err != nil || current.GetString("author") != owner.Id || current.GetString("text") != "Original" || canonicalFetches != 0 {
		t.Fatalf("stored author gate did not preserve the independent copy: %v %v requests=%d", current, err, canonicalFetches)
	}
	storage, err := f.app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()
	if len(filenames) != 1 {
		t.Fatalf("photo fixture: %v", filenames)
	}
	exists, err := storage.Exists(stored.BaseFilesPath() + "/" + filenames[0])
	if err != nil || !exists {
		t.Fatalf("stored photo changed: exists=%t error=%v", exists, err)
	}
}
