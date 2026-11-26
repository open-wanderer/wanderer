package routes

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestRemoteLogConfirmationEnrichesOnlyListedLogs(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, scenario := range []string{"listed", "omitted", "omitted-legacy-author"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteCleanupFixture(t, "summit_log")
			f.children.Fields.Add(&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1024}, &core.FileField{Name: "photos", MaxSelect: 2, MaxSize: 1024})
			if err := f.app.Save(f.children); err != nil {
				t.Fatal(err)
			}
			author := f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/carol", "last_fetched": time.Now().Add(-2 * time.Hour)})
			storedAuthor := author.Id
			if scenario == "omitted-legacy-author" {
				storedAuthor = f.remoteActor.Id
			}
			iri := "https://a.example:8443/api/v1/summit-log/original"
			log := f.child(iri, storedAuthor, f.trail.Id)
			listed := scenario == "listed"
			entries := []any{}
			if listed {
				entries = append(entries, map[string]any{"id": "copy", "iri": iri})
			}
			previousClient, previousActor, previousMedia := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI, fetchRemoteSyncMedia
			t.Cleanup(func() {
				newRemoteSyncHTTPClient, remoteSyncGetActorByIRI, fetchRemoteSyncMedia = previousClient, previousActor, previousMedia
			})
			var confirmations, actors atomic.Int32
			media := map[string]int{}
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "a.example" {
						return richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{"summit_logs_via_trail": entries}}}.RoundTrip(request)
					}
					confirmations.Add(1)
					return richTextSyncTransport{map[string]any{"id": "original", "iri": iri, "text": "Confirmed", "gpx": "track.gpx", "photos": []any{"photo.jpg"},
						"expand": map[string]any{"trail": map[string]any{"iri": f.trail.GetString("iri")}, "author": map[string]any{"iri": author.GetString("iri")}}}}.RoundTrip(request)
				}}}
			}
			remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, actorIRI string, includeFollows bool) (*core.Record, error) {
				actors.Add(1)
				if actorIRI != author.GetString("iri") || includeFollows {
					t.Errorf("unexpected actor lookup: %s includeFollows=%t", actorIRI, includeFollows)
				}
				return author, nil
			}
			fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, filename string, maxBytes int64) (*filesystem.File, func(), error) {
				media[filename]++
				file, err := filesystem.NewFileFromBytes([]byte("Canonical file"), filename)
				return file, func() {}, err
			}
			for round := 0; round < 2; round++ {
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatal(err)
				}
			}
			stored, err := f.app.FindRecordById(f.children, log.Id)
			if err != nil {
				t.Fatal(err)
			}
			if confirmations.Load() != 1 {
				t.Errorf("confirmation requests=%d, want 1 within cache TTL", confirmations.Load())
			}
			if listed {
				if actors.Load() != 1 || media["track.gpx"] != 1 || media["photo.jpg"] != 1 || len(media) != 2 {
					t.Errorf("listed enrichment changed: actors=%d media=%v", actors.Load(), media)
				}
				if stored.GetString("text") != "Confirmed" || stored.GetString("gpx") == "" || len(stored.GetStringSlice("photos")) != 1 {
					t.Fatalf("listed canonical data/media not saved: %v", stored)
				}
			} else {
				if actors.Load() != 0 || len(media) != 0 {
					t.Errorf("existence-only confirmation enriched unused data: actors=%d media=%v", actors.Load(), media)
				}
				if stored.GetString("text") != "Original" || stored.GetString("author") != storedAuthor || stored.GetString("gpx") != "" || len(stored.GetStringSlice("photos")) != 0 {
					t.Fatalf("omitted stored data changed: %v", stored)
				}
			}
		})
	}
}

func TestRemoteMediaCollectorSkipsOmittedCanonicalData(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	previous := fetchRemoteSyncMedia
	t.Cleanup(func() { fetchRemoteSyncMedia = previous })
	calls := 0
	fetchRemoteSyncMedia = func(context.Context, string, string, string, int64) (*filesystem.File, func(), error) {
		calls++
		return nil, func() {}, nil
	}
	canonical := map[string]canonicalRemoteSummitLog{"https://third.example/api/v1/summit-log/original": {
		omitted: true, remoteID: "original", origin: "https://third.example", data: map[string]any{"gpx": "track.gpx", "photos": []any{"photo.jpg"}},
	}}
	_, cleanup, err := prepareCollectedRemoteTrailMedia(f.app, t.Context(), remoteMediaCollector{seen: map[string]bool{}}, canonical)
	defer cleanup()
	if err != nil || calls != 0 {
		t.Fatalf("omitted data reached media prefetch: calls=%d error=%v", calls, err)
	}
}

func TestCanonicalOmittedLogConfirmsOnlyValidatedExistence(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, scenario := range []string{"valid", "wrong-id", "wrong-iri", "wrong-parent", "wrong-author-origin"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRemoteCleanupFixture(t, "summit_log")
			author := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
			iri := "https://third.example/api/v1/summit-log/original"
			f.child(iri, author.Id, f.trail.Id)
			authorMap, parent := map[string]any{"iri": author.GetString("iri")}, map[string]any{"iri": f.trail.GetString("iri")}
			payload := map[string]any{"id": "original", "iri": iri, "text": "Unused", "photos": []any{"unused.jpg"}, "expand": map[string]any{"author": authorMap, "trail": parent}}
			switch scenario {
			case "wrong-id":
				payload["id"] = "different"
			case "wrong-iri":
				payload["iri"] = "https://third.example/api/v1/summit-log/different"
			case "wrong-parent":
				parent["iri"] = "https://a.example/api/v1/trail/different"
			case "wrong-author-origin":
				authorMap["iri"] = "https://fourth.example/users/carol"
			}
			previousClient, previousActor := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI
			t.Cleanup(func() { newRemoteSyncHTTPClient, remoteSyncGetActorByIRI = previousClient, previousActor })
			newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
			var actors atomic.Int32
			remoteSyncGetActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) {
				actors.Add(1)
				return author, nil
			}
			canonical := fetchCanonicalRemoteSummitLogs(f.app, t.Context(), map[string]any{"expand": map[string]any{"summit_logs_via_trail": []any{}}}, f.trail, "https://a.example")
			confirmed, found := canonical[iri]
			if found != (scenario == "valid") || actors.Load() != 0 {
				t.Fatalf("existence validation: accepted=%t actors=%d", found, actors.Load())
			}
			if found && (!confirmed.omitted || confirmed.removed || confirmed.author != author.Id || confirmed.data != nil || confirmed.authorIRI != "") {
				t.Fatalf("existence probe retained mutable enrichment: %#v", confirmed)
			}
		})
	}
}

func TestOmittedLogConfirmationCacheFollowsTransaction(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	iri := "https://third.example/api/v1/summit-log/original"
	log := f.child(iri, f.remoteActor.Id, f.trail.Id)
	key := canonicalLogCacheKey(f.app, f.trail, iri)
	canonical := map[string]canonicalRemoteSummitLog{iri: {omitted: true, author: f.remoteActor.Id, cacheKey: key, updatedHint: "canonical-absence"}}
	rollback := errors.New("rollback confirmation")
	for _, commit := range []bool{false, true} {
		err := f.app.RunInTransaction(func(tx core.App) error {
			if err := applyOmittedRemoteSummitLogs(tx, f.trail, canonical, nil); err != nil {
				return err
			}
			if !commit {
				return rollback
			}
			return nil
		})
		if commit && err != nil || !commit && !errors.Is(err, rollback) {
			t.Fatal(err)
		}
		if cached := canonicalRemoteLogCache.skip(key, "canonical-absence", log.GetString("updated"), true); cached != commit {
			t.Fatalf("commit=%t cached=%t", commit, cached)
		}
	}
}
