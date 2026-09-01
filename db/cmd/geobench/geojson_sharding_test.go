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
	"time"

	"github.com/meilisearch/meilisearch-go"
)

func TestGeoJSONShardForIDIsStable(t *testing.T) {
	const id = "trail-000001"
	if got := geoJSONShardForID(id, 4); got != 1 {
		t.Fatalf("shard for %q = %d, want pinned FNV-1a shard 1", id, got)
	}
	for attempt := 0; attempt < 10; attempt++ {
		if got := geoJSONShardForID(id, 4); got != 1 {
			t.Fatalf("attempt %d shard = %d, want 1", attempt, got)
		}
	}
	if got := geoJSONShardForID(id, 1); got != 0 {
		t.Fatalf("single-index shard = %d, want 0", got)
	}
}

func TestSplitGeoJSONDocumentsByShardIsDisjointAndOrderIndependent(t *testing.T) {
	documents := []map[string]any{
		{"id": "trail-a"}, {"id": "trail-b"}, {"id": "trail-c"},
		{"id": "trail-d"}, {"id": "trail-e"},
	}
	forward, err := splitGeoJSONDocumentsByShard(documents, 4)
	if err != nil {
		t.Fatal(err)
	}
	reversedDocuments := append([]map[string]any(nil), documents...)
	for left, right := 0, len(reversedDocuments)-1; left < right; left, right = left+1, right-1 {
		reversedDocuments[left], reversedDocuments[right] = reversedDocuments[right], reversedDocuments[left]
	}
	reversed, err := splitGeoJSONDocumentsByShard(reversedDocuments, 4)
	if err != nil {
		t.Fatal(err)
	}

	seen := make(map[string]int, len(documents))
	for shard, values := range forward {
		for _, document := range values {
			id := document["id"].(string)
			if previous, duplicate := seen[id]; duplicate {
				t.Fatalf("document %q appears in shards %d and %d", id, previous, shard)
			}
			seen[id] = shard
		}
	}
	if len(seen) != len(documents) {
		t.Fatalf("partition contains %d documents, want %d", len(seen), len(documents))
	}
	for shard, values := range reversed {
		for _, document := range values {
			id := document["id"].(string)
			if seen[id] != shard {
				t.Fatalf("reordering moved %q from shard %d to %d", id, seen[id], shard)
			}
		}
	}
}

func TestSplitGeoJSONDocumentsByShardValidatesInputAndKeepsEmptyShards(t *testing.T) {
	shards, err := splitGeoJSONDocumentsByShard([]map[string]any{{"id": "only"}}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(shards) != 8 {
		t.Fatalf("shard count = %d, want 8", len(shards))
	}
	total := 0
	empty := 0
	for _, shard := range shards {
		total += len(shard)
		if len(shard) == 0 {
			empty++
		}
	}
	if total != 1 || empty != 7 {
		t.Fatalf("partition total/empty = %d/%d, want 1/7", total, empty)
	}
	if _, err := splitGeoJSONDocumentsByShard(nil, 0); err == nil {
		t.Fatal("zero shards unexpectedly accepted")
	}
	if _, err := splitGeoJSONDocumentsByShard([]map[string]any{{"id": 12}}, 2); err == nil || !strings.Contains(err.Error(), "string id") {
		t.Fatalf("invalid ID error = %v", err)
	}
}

func TestSplitGeoJSONDocumentsCapacityCreatedUsesPlanThenIDAndIgnoresPayloadDates(t *testing.T) {
	config := GeoJSONIndexConfig{
		ShardMode:         geoJSONShardModeCapacityCreated,
		MaxTrailsPerShard: 2,
	}
	documents := []map[string]any{
		{"id": "d", "created": int64(2), "origin_created": int64(500)},
		{"id": "b", "created": int64(40), "origin_created": int64(400)},
		{"id": "c", "created": int64(3), "origin_created": int64(300)},
		{"id": "a", "created": int64(30), "origin_created": int64(200)},
		{"id": "e", "created": int64(1), "origin_created": int64(100)},
	}

	localCreated := map[string]int64{"d": 20, "b": 10, "c": 20, "a": 10, "e": 30}
	shards, err := splitGeoJSONDocuments(documents, config, localCreated)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if got := geoJSONShardIDs(shards); !reflect.DeepEqual(got, want) {
		t.Fatalf("capacity-created shards = %v, want %v", got, want)
	}
	if len(shards) != 3 {
		t.Fatalf("actual shard count = %d, want ceil(5/2) = 3", len(shards))
	}
	for index, document := range documents {
		document["created"] = int64(10_000 - index)
		document["origin_created"] = int64(-10_000 + index)
		document["published"] = "1900-01-01T00:00:00Z"
	}
	mutated, err := splitGeoJSONDocuments(documents, config, localCreated)
	if err != nil {
		t.Fatal(err)
	}
	if got := geoJSONShardIDs(mutated); !reflect.DeepEqual(got, want) {
		t.Fatalf("mutating product/origin dates changed assignment: %v, want %v", got, want)
	}
}

func TestSplitGeoJSONDocumentsCapacityCreatedIsInputOrderIndependent(t *testing.T) {
	config := GeoJSONIndexConfig{ShardMode: geoJSONShardModeCapacityCreated, MaxTrailsPerShard: 2, CapacityArrival: geoJSONCapacityArrivalMixed}
	documents := []map[string]any{
		{"id": "c", "created": int64(2)},
		{"id": "a", "created": int64(1)},
		{"id": "d", "created": int64(2)},
		{"id": "b", "created": int64(1)},
	}
	localCreated := map[string]int64{"c": 2, "a": 1, "d": 2, "b": 1}
	forward, err := splitGeoJSONDocuments(documents, config, localCreated)
	if err != nil {
		t.Fatal(err)
	}
	reversed := append([]map[string]any(nil), documents...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	backward, err := splitGeoJSONDocuments(reversed, config, localCreated)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(geoJSONShardIDs(forward), geoJSONShardIDs(backward)) {
		t.Fatalf("input order changed capacity assignment: %v / %v", geoJSONShardIDs(forward), geoJSONShardIDs(backward))
	}
}

func TestCapacityCreatedShardValidation(t *testing.T) {
	valid := GeoJSONIndexConfig{ShardMode: geoJSONShardModeCapacityCreated, MaxTrailsPerShard: 2}
	for _, test := range []struct {
		name      string
		documents []map[string]any
		config    GeoJSONIndexConfig
		plan      map[string]int64
		want      string
	}{
		{name: "missing plan entry", documents: []map[string]any{{"id": "a", "created": 1, "origin_created": 1}}, config: valid, plan: map[string]int64{}, want: "missing from"},
		{name: "extra plan entry", documents: []map[string]any{{"id": "a"}}, config: valid, plan: map[string]int64{"a": 1, "b": 2}, want: "contains 2 trails"},
		{name: "duplicate id", documents: []map[string]any{{"id": "a"}, {"id": "a"}}, config: valid, plan: map[string]int64{"a": 1}, want: "duplicate id"},
		{name: "zero capacity", documents: []map[string]any{{"id": "a"}}, config: GeoJSONIndexConfig{ShardMode: geoJSONShardModeCapacityCreated}, plan: map[string]int64{"a": 1}, want: "must be positive"},
		{name: "unknown mode", documents: []map[string]any{{"id": "a"}}, config: GeoJSONIndexConfig{ShardMode: "calendar"}, plan: map[string]int64{"a": 1}, want: "unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := splitGeoJSONDocuments(test.documents, test.config, test.plan)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}

	if count, err := geoJSONShardCount(valid, 5); err != nil || count != 3 {
		t.Fatalf("capacity shard count = %d, %v; want 3, nil", count, err)
	}
	if shards, err := validateGeoJSONShardRun(valid, 3, false); err != nil || shards != 2 {
		t.Fatalf("multi-shard search validation = %d, %v; want 2, nil", shards, err)
	}
	if _, err := validateGeoJSONShardRun(valid, 2, false); err == nil || !strings.Contains(err.Error(), "at least two") {
		t.Fatalf("single capacity shard without index-only error = %v", err)
	}
	if _, err := geoJSONShardCount(GeoJSONIndexConfig{ShardMode: geoJSONShardModeCapacityCreated, MaxTrailsPerShard: 781}, 50_000); err == nil || !strings.Contains(err.Error(), "maximum is 64") || !strings.Contains(err.Error(), ">= 782") {
		t.Fatalf("derived shard limit error = %v", err)
	}
}

func TestGeoJSONShardAssignmentSHA256IsPinnedAndInputOrderIndependent(t *testing.T) {
	first := [][]map[string]any{
		{{"id": "b"}, {"id": "a"}},
		{{"id": "c"}},
	}
	reordered := [][]map[string]any{
		{{"id": "a"}, {"id": "b"}},
		{{"id": "c"}},
	}
	firstDigest, err := geoJSONShardAssignmentSHA256(first)
	if err != nil {
		t.Fatal(err)
	}
	reorderedDigest, err := geoJSONShardAssignmentSHA256(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if firstDigest != reorderedDigest {
		t.Fatalf("assignment digest depends on document order: %s / %s", firstDigest, reorderedDigest)
	}
	const want = "059005385ce4a34a8fea01c803d715c95e38f296299702f5f862fda925f29e60"
	if firstDigest != want {
		t.Fatalf("assignment digest = %s, want pinned %s", firstDigest, want)
	}
}

func TestCapacityArrivalPlans(t *testing.T) {
	trails := []Trail{{ID: "trail-c"}, {ID: "trail-a"}, {ID: "trail-b"}, {ID: "trail-d"}}
	clustered, err := geoJSONCapacityLocalCreatedPlan(trails, 42, geoJSONCapacityArrivalClustered)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]int64{"trail-c": 0, "trail-a": 1, "trail-b": 2, "trail-d": 3}; !reflect.DeepEqual(clustered, want) {
		t.Fatalf("clustered arrival = %v, want dataset order %v", clustered, want)
	}
	mixed, err := geoJSONCapacityLocalCreatedPlan(trails, 42, geoJSONCapacityArrivalMixed)
	if err != nil {
		t.Fatal(err)
	}
	reversed := append([]Trail(nil), trails...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	mixedReversed, err := geoJSONCapacityLocalCreatedPlan(reversed, 42, geoJSONCapacityArrivalMixed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mixed, mixedReversed) {
		t.Fatalf("mixed arrival depends on dataset order: %v / %v", mixed, mixedReversed)
	}
	if reflect.DeepEqual(mixed, clustered) {
		t.Fatalf("mixed seed+ID permutation unexpectedly equals clustered order: %v", mixed)
	}
	if _, err := geoJSONCapacityLocalCreatedPlan(trails, 42, "remote-first"); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("invalid arrival error = %v", err)
	}
}

func TestPopulateCapacityCreatedShardReportDistribution(t *testing.T) {
	source := []Trail{
		{ID: "a", Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}, {Lat: 0, Lon: 2}}}},
		{ID: "b", Parts: [][]Coordinate{{{Lat: 1, Lon: 0}, {Lat: 1, Lon: 1}}}},
		{ID: "c", Parts: [][]Coordinate{{{Lat: 2, Lon: 0}, {Lat: 2, Lon: 1}}}},
	}
	indexed := []Trail{
		{ID: "a", Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 2}}}},
		{ID: "b", Parts: source[1].Parts},
		{ID: "c", Parts: source[2].Parts},
	}
	config := GeoJSONIndexConfig{ShardMode: geoJSONShardModeCapacityCreated, MaxTrailsPerShard: 2, CapacityArrival: geoJSONCapacityArrivalMixed}
	documents := []map[string]any{
		{"id": "c", "created": int64(3)},
		{"id": "a", "created": int64(1)},
		{"id": "b", "created": int64(2)},
	}
	shards, err := splitGeoJSONDocuments(documents, config, map[string]int64{"a": 1, "b": 2, "c": 3})
	if err != nil {
		t.Fatal(err)
	}
	report := &GeoJSONIndexReport{}
	if err := populateGeoJSONShardReport(report, source, indexed, shards, config); err != nil {
		t.Fatal(err)
	}
	if report.ShardMode != geoJSONShardModeCapacityCreated || report.MaxTrailsPerShard != 2 || report.CapacityArrival != geoJSONCapacityArrivalMixed || report.AssignmentScope != geoJSONCapacitySnapshotScope || report.Shards != 2 || report.ShardAlgorithm != geoJSONCapacityShardAlgorithm || len(report.ShardAssignmentSHA256) != 64 {
		t.Fatalf("capacity report metadata = %+v", report)
	}
	if !reflect.DeepEqual(report.ShardDocumentCounts, []int{2, 1}) ||
		!reflect.DeepEqual(report.ShardSourceVertices, []int{5, 2}) ||
		!reflect.DeepEqual(report.ShardIndexedVertices, []int{4, 2}) {
		t.Fatalf("capacity report distribution = %+v", report)
	}
}

func geoJSONShardIDs(shards [][]map[string]any) [][]string {
	result := make([][]string, len(shards))
	for shard, documents := range shards {
		for _, document := range documents {
			result[shard] = append(result[shard], document["id"].(string))
		}
	}
	return result
}

func TestGeoJSONProjectionReportsPerShardGeometry(t *testing.T) {
	trails := []Trail{
		{ID: "a", Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}, {Lat: 0, Lon: 2}}}},
		{ID: "b", Parts: [][]Coordinate{{{Lat: 1, Lon: 0}, {Lat: 1, Lon: 1}}}},
		{ID: "c", Parts: [][]Coordinate{{{Lat: 2, Lon: 0}, {Lat: 2, Lon: 1}}, {{Lat: 3, Lon: 0}, {Lat: 3, Lon: 1}}}},
	}
	_, report, err := projectMeiliIndexTrails(context.Background(), "geojson", trails, GeoJSONIndexConfig{Shards: 4})
	if err != nil {
		t.Fatal(err)
	}
	if report.Shards != 4 || report.ShardAlgorithm != geoJSONShardAlgorithm {
		t.Fatalf("shard metadata = %+v", report)
	}
	if sumInts(report.ShardDocumentCounts) != len(trails) ||
		sumInts(report.ShardSourceVertices) != report.SourceVertices ||
		sumInts(report.ShardIndexedVertices) != report.IndexedVertices ||
		sumInts(report.ShardSourceSegments) != report.SourceSegments ||
		sumInts(report.ShardIndexedSegments) != report.IndexedSegments {
		t.Fatalf("per-shard geometry does not sum to projection: %+v", report)
	}
	if report.PreNormalizationVertices != report.SourceVertices ||
		report.PreNormalizationSegments != report.SourceSegments {
		t.Fatalf("zero-tolerance RDP changed geometry: %+v", report)
	}
	if report.MaxSegmentLengthMeters != defaultGeoJSONMaxSegmentLengthMeters ||
		report.DensifiedVertices == 0 || report.IndexedVertices <= report.SourceVertices ||
		report.AntimeridianCuts != 0 {
		t.Fatalf("default segment normalization was not reported: %+v", report)
	}
}

func TestIndexGeoJSONShardsEnqueuesEveryIndexBeforeBarrierAndAggregates(t *testing.T) {
	indexUIDs := geoJSONIndexUIDs(2)
	documentShards := [][]map[string]any{
		{{"id": "a"}, {"id": "b"}},
		{{"id": "c"}, {"id": "d"}},
	}
	var mutex sync.Mutex
	posted := make(map[string]int)
	taskIndex := make(map[int64]string)
	nextTask := int64(1)
	taskPolledBeforeAllSubmissions := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/documents"):
			var documents []map[string]any
			if err := json.NewDecoder(request.Body).Decode(&documents); err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
			indexUID := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/indexes/"), "/documents")
			mutex.Lock()
			uid := nextTask
			nextTask++
			posted[indexUID] += len(documents)
			taskIndex[uid] = indexUID
			mutex.Unlock()
			writer.WriteHeader(http.StatusAccepted)
			_, _ = writer.Write([]byte(`{"taskUid":` + strconv.FormatInt(uid, 10) + `,"indexUid":"` + indexUID + `","status":"enqueued","type":"documentAdditionOrUpdate","enqueuedAt":"2024-01-01T00:00:00Z"}`))
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/tasks/"):
			uid, _ := strconv.ParseInt(strings.TrimPrefix(request.URL.Path, "/tasks/"), 10, 64)
			mutex.Lock()
			if len(posted) != len(indexUIDs) {
				taskPolledBeforeAllSubmissions = true
			}
			indexUID := taskIndex[uid]
			mutex.Unlock()
			_, _ = writer.Write([]byte(`{"uid":` + strconv.FormatInt(uid, 10) + `,"indexUid":"` + indexUID + `","status":"succeeded","type":"documentAdditionOrUpdate","enqueuedAt":"2024-01-01T00:00:00Z","finishedAt":"` + time.Now().UTC().Format(time.RFC3339Nano) + `"}`))
		case request.Method == http.MethodGet && request.URL.Path == "/stats":
			_, _ = writer.Write([]byte(`{"databaseSize":4096,"usedDatabaseSize":2048,"indexes":{"` + indexUIDs[0] + `":{"numberOfDocuments":2},"` + indexUIDs[1] + `":{"numberOfDocuments":2}}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/batches":
			indexUID := request.URL.Query().Get("indexUids")
			uid := 10
			if indexUID == indexUIDs[1] {
				uid = 11
			}
			_, _ = writer.Write([]byte(`{"results":[{"uid":` + strconv.Itoa(uid) + `,"duration":"PT1S","batchStrategy":"test","stats":{"totalNbTasks":1}}],"total":1,"limit":1}`))
		default:
			http.Error(writer, request.Method+" "+request.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	container := &meiliContainer{
		name:    "fake-sharded-meili",
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
	report, builds, err := indexGeoJSONShards(
		context.Background(), client, container, indexUIDs, documentShards,
		100, meiliMaxBatchBytes, 0,
	)
	if err != nil {
		t.Fatal(err)
	}

	mutex.Lock()
	defer mutex.Unlock()
	if taskPolledBeforeAllSubmissions {
		t.Fatal("a task was polled before every shard request was submitted")
	}
	if !reflect.DeepEqual(posted, map[string]int{indexUIDs[0]: 2, indexUIDs[1]: 2}) {
		t.Fatalf("submitted documents by index = %#v", posted)
	}
	if report.DocumentCount != 4 || report.SubmittedDocuments != 4 || report.TaskCount != 2 || report.BarrierCount != 1 || report.SubmittedBytes == 0 {
		t.Fatalf("aggregate sharded phase = %+v", report)
	}
	if report.DiskBytes != 4096 || report.UsedDiskBytes != 2048 || report.LastBatch != nil {
		t.Fatalf("aggregate server gauges = %+v", report)
	}
	if len(builds) != 2 {
		t.Fatalf("shard builds = %d, want 2", len(builds))
	}
	for shard, build := range builds {
		if build.IndexUID != indexUIDs[shard] || build.Phase.DocumentCount != 2 ||
			build.Phase.SubmittedDocuments != 2 || build.Phase.TaskCount != 1 ||
			build.Phase.BarrierCount != 1 || build.Phase.LastBatch == nil || build.Phase.LastBatch.UID != 10+shard {
			t.Fatalf("shard %d build = %+v", shard, build)
		}
	}
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
