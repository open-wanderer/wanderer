package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/meilisearch/meilisearch-go"
)

func TestDeriveAddedTrailsIsDeterministicAndDoesNotModifyInput(t *testing.T) {
	dataset, err := generateDataset(30, 9, 17)
	if err != nil {
		t.Fatal(err)
	}
	original := cloneTrailsForTest(dataset.Trails)

	first := deriveAddedTrails(dataset, 6)
	second := deriveAddedTrails(dataset, 6)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("added trails differ between deterministic runs")
	}
	if !reflect.DeepEqual(dataset.Trails, original) {
		t.Fatal("deriveAddedTrails modified its input")
	}
	if len(first) != 6 {
		t.Fatalf("added trail count = %d, want 6", len(first))
	}

	existing := make(map[string]struct{}, len(original))
	for _, trail := range original {
		existing[trail.ID] = struct{}{}
	}
	generated := make(map[string]struct{}, len(first))
	for index, trail := range first {
		if _, collision := existing[trail.ID]; collision {
			t.Fatalf("added trail %d reuses existing ID %q", index, trail.ID)
		}
		if _, duplicate := generated[trail.ID]; duplicate {
			t.Fatalf("added trail ID %q is duplicated", trail.ID)
		}
		generated[trail.ID] = struct{}{}

		sourceIndex := mutationSourceIndex(len(original), len(first), index)
		if trail.Scenario != original[sourceIndex].Scenario {
			t.Fatalf("added trail %d scenario = %q, want %q", index, trail.Scenario, original[sourceIndex].Scenario)
		}
		if reflect.DeepEqual(trail.Parts, original[sourceIndex].Parts) {
			t.Fatalf("added trail %d has unchanged source geometry", index)
		}
	}

	// Mutation batches intentionally include the corpus tail, where the rare
	// long-route scenario is generated.
	if first[len(first)-1].Scenario != scenarioLongOutlier {
		t.Fatalf("last added trail scenario = %q, want %q", first[len(first)-1].Scenario, scenarioLongOutlier)
	}
	if single := deriveAddedTrails(dataset, 1); len(single) != 1 || single[0].Scenario != scenarioLongOutlier {
		t.Fatalf("single added trail = %+v, want one long-outlier trail", single)
	}
}

func TestDeriveAddedTrailsCanExceedSourceCountAndAvoidIDCollisions(t *testing.T) {
	dataset := Dataset{
		Seed: 4,
		Trails: []Trail{
			{ID: "trail-added-4-000000", Scenario: "one", Parts: [][]Coordinate{{{Lat: 1, Lon: 2}}}},
			{ID: "source-two", Scenario: "two", Parts: [][]Coordinate{{{Lat: 3, Lon: 4}}}},
		},
	}
	added := deriveAddedTrails(dataset, 5)
	if len(added) != 5 {
		t.Fatalf("added trail count = %d, want 5", len(added))
	}
	if added[0].ID != "trail-added-4-000000-1" {
		t.Fatalf("collision-safe ID = %q, want trail-added-4-000000-1", added[0].ID)
	}
	for index, wantScenario := range []string{"one", "two", "one", "two", "one"} {
		if added[index].Scenario != wantScenario {
			t.Errorf("added trail %d scenario = %q, want %q", index, added[index].Scenario, wantScenario)
		}
	}
}

func TestSelectDeletionIDsSamplesCorpusWithoutMutation(t *testing.T) {
	trails := []Trail{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}, {ID: "e"}}
	original := append([]Trail(nil), trails...)
	if got, want := selectDeletionIDs(trails, 3), []string{"a", "c", "e"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("selected deletion IDs = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(trails, original) {
		t.Fatal("selectDeletionIDs modified its input")
	}
	if got := selectDeletionIDs(trails, 1); !reflect.DeepEqual(got, []string{"e"}) {
		t.Fatalf("single deletion ID = %#v, want corpus tail", got)
	}
	if got := selectDeletionIDs(trails, 99); !reflect.DeepEqual(got, []string{"a", "b", "c", "d", "e"}) {
		t.Fatalf("capped deletion IDs = %#v", got)
	}
	if got := selectDeletionIDs(trails, 0); got != nil {
		t.Fatalf("zero deletion IDs = %#v, want nil", got)
	}
}

func TestDeleteMeiliDocumentsBatchesWaitsAndCapturesPhaseStats(t *testing.T) {
	var mutex sync.Mutex
	var batches [][]string
	taskRequests := make(map[string]int)
	nextTask := int64(1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/indexes/"+benchmarkIndexUID+"/documents/delete-batch":
			var ids []string
			if err := json.NewDecoder(request.Body).Decode(&ids); err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
			mutex.Lock()
			uid := nextTask
			nextTask++
			batches = append(batches, ids)
			mutex.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusAccepted)
			_, _ = writer.Write([]byte(`{"taskUid":` + strconv.FormatInt(uid, 10) + `,"indexUid":"` + benchmarkIndexUID + `","status":"enqueued","type":"documentDeletion","enqueuedAt":"2024-01-01T00:00:00Z"}`))
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/tasks/"):
			uid := strings.TrimPrefix(request.URL.Path, "/tasks/")
			mutex.Lock()
			taskRequests[uid]++
			mutex.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"uid":` + uid + `,"indexUid":"` + benchmarkIndexUID + `","status":"succeeded","type":"documentDeletion","enqueuedAt":"2024-01-01T00:00:00Z"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/stats":
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"databaseSize":4096,"usedDatabaseSize":2048,"indexes":{"` + benchmarkIndexUID + `":{"numberOfDocuments":7,"isIndexing":false,"fieldDistribution":{}}}}`))
		default:
			http.Error(writer, request.Method+" "+request.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	container := &meiliContainer{
		name:    "fake-meili",
		dataDir: t.TempDir(),
		statsClient: &http.Client{Transport: meiliDeleteStatsRoundTripper(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"memory_stats":{"usage":8192,"stats":{"inactive_file":4096}}}`)),
			}, nil
		})},
	}
	client := meilisearch.New(server.URL)
	report, err := deleteMeiliDocuments(
		context.Background(),
		client,
		container,
		[]string{"a", "b", "c", "d", "e"},
		2,
	)
	if err != nil {
		t.Fatal(err)
	}

	mutex.Lock()
	defer mutex.Unlock()
	if want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}; !reflect.DeepEqual(batches, want) {
		t.Fatalf("delete batches = %#v, want %#v", batches, want)
	}
	for _, uid := range []string{"1", "2", "3"} {
		if taskRequests[uid] != 1 {
			t.Errorf("task %s audit requests = %d, want 1", uid, taskRequests[uid])
		}
	}
	if report.DocumentCount != 7 || report.DiskBytes != 4096 || report.UsedDiskBytes != 2048 {
		t.Fatalf("delete phase storage stats = %+v", report)
	}
	if report.EngineBaselineBytes != 4096 || report.EnginePeakBytes != 4096 {
		t.Fatalf("delete phase engine RSS = baseline %d peak %d, want 4096/4096", report.EngineBaselineBytes, report.EnginePeakBytes)
	}
}

func TestDeleteMeiliDocumentsRejectsEmptyIDWithoutStartingPhase(t *testing.T) {
	if _, err := deleteMeiliDocuments(context.Background(), nil, nil, []string{""}, 100); err == nil {
		t.Fatal("empty Meilisearch delete ID succeeded")
	}
	if report, err := deleteMeiliDocuments(context.Background(), nil, nil, nil, 100); err != nil || !reflect.DeepEqual(report, PhaseReport{}) {
		t.Fatalf("empty Meilisearch delete batch = %+v, err %v", report, err)
	}
}

type meiliDeleteStatsRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip meiliDeleteStatsRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}
