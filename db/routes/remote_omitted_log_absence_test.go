package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type omitted404Fixture struct {
	*remoteChildFixture
	log, author       *core.Record
	iri, key          string
	public            bool
	entries, points   []any
	listMode          string
	responseParentIRI string
	status, requests  atomic.Int32
	beforeProbe       func(int32) error
}

func newOmitted404Fixture(t *testing.T) *omitted404Fixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://b.example")
	f := &omitted404Fixture{remoteChildFixture: newRemoteCleanupFixture(t, "summit_log"), public: true, entries: []any{}}
	f.children.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
	for _, collection := range []*core.Collection{f.children, f.trails} {
		collection.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		if err := f.app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	trail, err := f.app.FindRecordById(f.trails, f.trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	f.trail = trail
	if err := f.app.Save(f.trail); err != nil {
		t.Fatal(err)
	}
	f.author = f.save(f.actors, map[string]any{"iri": "https://a.example:8443/users/carol", "last_fetched": time.Now()})
	f.iri = "https://a.example:8443/api/v1/summit-log/omitted"
	f.log = f.child(f.iri, f.author.Id, f.trail.Id)
	f.key = canonicalLogCacheKey(f.app, f.trail, f.iri)
	f.status.Store(http.StatusNotFound)
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() {
		newRemoteSyncHTTPClient = previous
		canonicalRemoteLogAbsenceCache.resetTrail(f.app, f.trail)
	})
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if request.URL.Host == "a.example" {
				return richTextSyncTransport{f.document()}.RoundTrip(request)
			}
			status := int(f.status.Load())
			index := f.requests.Add(1)
			if f.beforeProbe != nil {
				if err := f.beforeProbe(index); err != nil {
					return nil, err
				}
			}
			if status != http.StatusOK {
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
			}
			parentIRI := f.trail.GetString("iri")
			if f.responseParentIRI != "" {
				parentIRI = f.responseParentIRI
			}
			return richTextSyncTransport{map[string]any{"id": "omitted", "iri": f.iri, "text": "Existence only", "expand": map[string]any{
				"trail": map[string]any{"iri": parentIRI}, "author": map[string]any{"iri": f.author.GetString("iri")},
			}}}.RoundTrip(request)
		}}}
	}
	return f
}

func (f *omitted404Fixture) document() map[string]any {
	expand := map[string]any{}
	data := map[string]any{"id": "trail", "public": f.public, "expand": expand}
	entries := f.entries
	switch f.listMode {
	case "absent":
		return data
	case "malformed":
		expand["summit_logs_via_trail"] = []any{map[string]any{"text": "No identity"}}
		return data
	case "saturated", "capped":
		count := remoteChildExpandLimit
		if f.listMode == "capped" {
			count++
		}
		entries = make([]any, count)
		for i := range entries {
			entries[i] = map[string]any{"id": "local", "iri": "https://b.example/api/v1/summit-log/local"}
		}
	}
	expand["summit_logs_via_trail"] = entries
	if f.points != nil {
		expand["waypoints_via_trail"] = f.points
	}
	return data
}

func (f *omitted404Fixture) refresh() error {
	trail, err := performFullSync(f.app, f.t.Context(), nil, f.trail)
	if trail != nil {
		f.trail = trail
	}
	return err
}

func (f *omitted404Fixture) fetch() map[string]canonicalRemoteSummitLog {
	return fetchCanonicalRemoteSummitLogs(f.app, f.t.Context(), f.document(), f.trail, "https://a.example")
}

func (f *omitted404Fixture) apply(canonical map[string]canonicalRemoteSummitLog) error {
	return f.app.RunInTransaction(func(tx core.App) error { return applyOmittedRemoteSummitLogs(tx, f.trail, canonical, nil) })
}

func (f *omitted404Fixture) proof() canonicalLogAbsenceProbe {
	canonicalRemoteLogAbsenceCache.Lock()
	defer canonicalRemoteLogAbsenceCache.Unlock()
	return canonicalLogAbsenceProbe{key: f.key, entry: canonicalRemoteLogAbsenceCache.entries[f.key]}
}

func (f *omitted404Fixture) ageFirst(elapsed time.Duration) time.Time {
	f.t.Helper()
	canonicalRemoteLogAbsenceCache.Lock()
	defer canonicalRemoteLogAbsenceCache.Unlock()
	entry := canonicalRemoteLogAbsenceCache.entries[f.key]
	if entry == nil {
		f.t.Fatal("first observation was not retained")
	}
	entry.first404 = time.Now().Add(-elapsed)
	return entry.first404
}

func (f *omitted404Fixture) expireGeneric() {
	canonicalRemoteLogCache.Lock()
	defer canonicalRemoteLogCache.Unlock()
	if entry, found := canonicalRemoteLogCache.entries[f.key]; found {
		entry.checked = time.Now().Add(-max(remoteSyncThreshold, canonicalRemoteLogFailureTTL) - time.Second)
		canonicalRemoteLogCache.entries[f.key] = entry
	}
}

func (f *omitted404Fixture) assertExists(want bool) {
	f.t.Helper()
	_, err := f.app.FindRecordById(f.children, f.log.Id)
	if want && err != nil || !want && !errors.Is(err, sql.ErrNoRows) {
		f.t.Fatalf("cached log existence: want=%t error=%v", want, err)
	}
}

func saveOmitted404Newer(app core.App, record *core.Record) error {
	updated, err := types.ParseDateTime(record.GetDateTime("updated").Time().Add(time.Hour))
	if err != nil {
		return err
	}
	record.SetRaw("updated", updated)
	return app.Save(record)
}

func TestRemoteOmitted404RequiresSeparatedPublicObservations(t *testing.T) {
	for _, interval := range []time.Duration{time.Minute, 37 * time.Minute} {
		t.Run(interval.String(), func(t *testing.T) {
			previous := remoteSyncThreshold
			remoteSyncThreshold = interval
			t.Cleanup(func() { remoteSyncThreshold = previous })
			f := newOmitted404Fixture(t)
			for round := 0; round < 2; round++ {
				if err := f.refresh(); err != nil {
					t.Fatal(err)
				}
				f.assertExists(true)
			}
			if f.requests.Load() != 1 {
				t.Fatal("a negative-cache hit became another observation")
			}
			first := f.ageFirst(interval / 2)
			f.expireGeneric()
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
			canonicalRemoteLogAbsenceCache.Lock()
			unchanged := canonicalRemoteLogAbsenceCache.entries[f.key].first404 == first
			canonicalRemoteLogAbsenceCache.Unlock()
			if !unchanged || f.requests.Load() != 2 {
				t.Fatal("generic-cache expiration lost evidence or an early observation advanced it")
			}
			f.ageFirst(interval + time.Second)
			f.expireGeneric()
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(false)
			if f.requests.Load() != 3 || f.proof().entry != nil {
				t.Fatal("qualified deletion did not commit and forget its evidence")
			}
		})
	}
}

func TestRemoteOmitted404PreservesPrivateAndListedStatusPolicies(t *testing.T) {
	for _, scenario := range []string{"private-404", "public-410", "private-410", "listed-404"} {
		t.Run(scenario, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			f.public = scenario != "private-404" && scenario != "private-410"
			if scenario == "public-410" || scenario == "private-410" {
				f.status.Store(http.StatusGone)
			}
			if scenario == "listed-404" {
				f.entries = []any{map[string]any{"id": "copy", "iri": f.iri}}
			}
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(scenario == "private-404")
			if f.proof().entry != nil {
				t.Fatal("private/listed/410 policy retained public omitted-404 evidence")
			}
			if scenario == "private-404" {
				f.expireGeneric()
				if err := f.refresh(); err != nil {
					t.Fatal(err)
				}
				f.assertExists(true)
			}
		})
	}
}

func TestRemoteOmitted404CounterevidenceResetsBeforeNegativeCacheSkip(t *testing.T) {
	for _, scenario := range []string{"listed", "private", "changed-version", "local-author", "wrong-author-origin"} {
		t.Run(scenario, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.ageFirst(remoteSyncThreshold + time.Second)
			switch scenario {
			case "listed":
				f.entries = []any{map[string]any{"id": "copy", "iri": f.iri}}
			case "private":
				f.public = false
			case "changed-version":
				if err := saveOmitted404Newer(f.app, f.log); err != nil {
					t.Fatal(err)
				}
			case "local-author", "wrong-author-origin":
				if scenario == "local-author" {
					f.author.Set("is_local", true)
				} else {
					f.author.Set("iri", "https://fourth.example/users/carol")
				}
				if err := f.app.Save(f.author); err != nil {
					t.Fatal(err)
				}
			}
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
			if f.requests.Load() != 1 || f.proof().entry != nil {
				t.Fatal("counterevidence was hidden behind a fresh negative-cache entry")
			}
			f.public, f.entries = true, []any{}
			f.author.Set("is_local", false)
			f.author.Set("iri", "https://a.example:8443/users/carol")
			if err := f.app.Save(f.author); err != nil {
				t.Fatal(err)
			}
			f.expireGeneric()
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
			if f.requests.Load() != 2 {
				t.Fatal("fresh omission was not probed after counterevidence")
			}
		})
	}
}

func TestRemoteOmitted404Non404ResponsesResetEvidence(t *testing.T) {
	for _, scenario := range []struct {
		name      string
		status    int
		probeErr  error
		parentIRI string
	}{
		{name: "200", status: http.StatusOK},
		{name: "invalid-200", status: http.StatusOK, parentIRI: "https://a.example/api/v1/trail/wrong"},
		{name: "403", status: http.StatusForbidden},
		{name: "503", status: http.StatusServiceUnavailable},
		{name: "transport-gone-without-http-status", probeErr: errRemoteGone},
		{name: "fetch-queue", probeErr: errRemoteSyncFetchQueue},
		{name: "rate-limited", probeErr: util.ErrRateLimited},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.ageFirst(remoteSyncThreshold + time.Second)
			f.expireGeneric()
			f.status.Store(int32(scenario.status))
			f.responseParentIRI = scenario.parentIRI
			f.beforeProbe = func(int32) error { return scenario.probeErr }
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
			if f.proof().entry != nil {
				t.Fatal("a non-404 response retained absence evidence")
			}
			f.status.Store(http.StatusNotFound)
			f.responseParentIRI = ""
			f.beforeProbe = nil
			f.expireGeneric()
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
		})
	}
}

func TestRemoteOmitted404InconclusiveExpandsNeverAuthorizeDeletion(t *testing.T) {
	for _, mode := range []string{"absent", "malformed", "saturated", "capped"} {
		t.Run(mode, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.ageFirst(remoteSyncThreshold + time.Second)
			f.expireGeneric()
			f.listMode = mode
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
			if f.requests.Load() != 1 || f.proof().entry != nil {
				t.Fatal("an inconclusive expand counted as a fresh omission")
			}
			f.listMode = ""
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
		})
	}
}
