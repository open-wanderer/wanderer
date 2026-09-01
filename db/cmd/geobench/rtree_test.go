package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrepareRTreeTrailsHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := prepareRTreeTrailsContext(ctx, []Trail{{ID: "route", Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}}}}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("prepare cancellation error = %v", err)
	}
}

func TestRTreeBackendQueryCandidates(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	trails := []Trail{
		{
			ID: "crossing",
			Parts: [][]Coordinate{{
				{Lat: 47.3769, Lon: 8.5217},
				{Lat: 47.3769, Lon: 8.5617},
			}},
		},
		{
			ID: "far",
			Parts: [][]Coordinate{{
				{Lat: 47.5, Lon: 8.52},
				{Lat: 47.5, Lon: 8.56},
			}},
		},
	}
	if err := backend.fullIndex(ctx, trails); err != nil {
		t.Fatal(err)
	}

	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 47.3769, Lon: 8.5417}, 50)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids, "crossing")
}

func TestRTreeBackendLongSegmentReturnsConservativeCandidate(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	// This point is hundreds of kilometres from the diagonal itself, but lies
	// inside its segment bounding box. RTree is deliberately a candidate
	// stage: exact point-to-polyline verification removes this false positive.
	trail := Trail{
		ID: "long-diagonal",
		Parts: [][]Coordinate{{
			{Lat: 0, Lon: 0},
			{Lat: 10, Lon: 10},
		}},
	}
	if err := backend.fullIndex(ctx, []Trail{trail}); err != nil {
		t.Fatal(err)
	}

	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 0, Lon: 10}, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids, "long-diagonal")
}

func TestRTreeBackendUpdateReplacesSegmentsAndGeometry(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	original := Trail{
		ID:       "moving",
		Scenario: "original",
		Parts: [][]Coordinate{{
			{Lat: 47, Lon: 8},
			{Lat: 47, Lon: 8.1},
		}},
	}
	if err := backend.fullIndex(ctx, []Trail{original}); err != nil {
		t.Fatal(err)
	}

	replacement := Trail{
		ID:       "moving",
		Scenario: "replacement",
		Parts: [][]Coordinate{{
			{Lat: 46, Lon: 7},
			{Lat: 46, Lon: 7.1},
			{Lat: 46.1, Lon: 7.1},
		}},
	}
	if err := backend.update(ctx, []Trail{replacement}); err != nil {
		t.Fatal(err)
	}

	oldIDs, err := backend.queryCandidates(ctx, Coordinate{Lat: 47, Lon: 8.05}, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, oldIDs)

	newIDs, err := backend.queryCandidates(ctx, Coordinate{Lat: 46, Lon: 7.05}, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, newIDs, "moving")

	count, err := backend.documentCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("document count = %d, want 1", count)
	}

	var scenario string
	var storedGeometry []byte
	if err := backend.db.QueryRowContext(ctx,
		`SELECT scenario, geometry FROM trails WHERE id = ?`, replacement.ID,
	).Scan(&scenario, &storedGeometry); err != nil {
		t.Fatal(err)
	}
	if scenario != replacement.Scenario {
		t.Fatalf("stored scenario = %q, want %q", scenario, replacement.Scenario)
	}
	var parts [][]Coordinate
	if err := json.Unmarshal(storedGeometry, &parts); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parts, replacement.Parts) {
		t.Fatalf("stored geometry = %#v, want %#v", parts, replacement.Parts)
	}

	var segmentCount int
	if err := backend.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM segments WHERE trail_id = ?`, replacement.ID,
	).Scan(&segmentCount); err != nil {
		t.Fatal(err)
	}
	if segmentCount != 2 {
		t.Fatalf("stored segment bounds = %d, want 2", segmentCount)
	}
}

func TestRTreeBackendUpdateIsAtomic(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	original := Trail{
		ID: "original",
		Parts: [][]Coordinate{{
			{Lat: 47, Lon: 8},
			{Lat: 47, Lon: 8.1},
		}},
	}
	if err := backend.fullIndex(ctx, []Trail{original}); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.db.ExecContext(ctx, `
		CREATE TRIGGER fail_invalid_rtree_trail
		BEFORE INSERT ON trails
		WHEN NEW.id = 'invalid'
		BEGIN
			SELECT RAISE(ABORT, 'forced rtree test failure');
		END
	`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = backend.db.Exec(`DROP TRIGGER IF EXISTS fail_invalid_rtree_trail`)
	})

	err := backend.update(ctx, []Trail{
		{
			ID: "original",
			Parts: [][]Coordinate{{
				{Lat: 46, Lon: 7},
				{Lat: 46, Lon: 7.1},
			}},
		},
		{
			ID: "invalid",
			Parts: [][]Coordinate{{
				{Lat: 45, Lon: 6},
			}},
		},
	})
	if err == nil {
		t.Fatal("update with invalid coordinate succeeded")
	}

	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 47, Lon: 8.05}, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids, "original")

	ids, err = backend.queryCandidates(ctx, Coordinate{Lat: 46, Lon: 7.05}, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids)
}

func TestRTreeBackendDeleteTrailsRemovesGeometryAndBounds(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	trails := []Trail{
		{ID: "deleted", Parts: [][]Coordinate{{{Lat: 47, Lon: 8}, {Lat: 47, Lon: 8.1}}}},
		{ID: "kept", Parts: [][]Coordinate{{{Lat: 46, Lon: 7}, {Lat: 46, Lon: 7.1}}}},
	}
	if err := backend.fullIndex(ctx, trails); err != nil {
		t.Fatal(err)
	}
	if err := backend.deleteTrails(ctx, []string{"deleted", "missing", "deleted"}); err != nil {
		t.Fatal(err)
	}

	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 47, Lon: 8.05}, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids)
	ids, err = backend.queryCandidates(ctx, Coordinate{Lat: 46, Lon: 7.05}, 100)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids, "kept")

	count, err := backend.documentCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("document count after delete = %d, want 1", count)
	}
	for _, table := range []string{"segments", "segment_bounds"} {
		var deletedRows int
		query := `SELECT COUNT(*) FROM ` + table + ` WHERE id NOT IN (SELECT id FROM segments)`
		if table == "segments" {
			query = `SELECT COUNT(*) FROM segments WHERE trail_id = 'deleted'`
		}
		if err := backend.db.QueryRowContext(ctx, query).Scan(&deletedRows); err != nil {
			t.Fatalf("count stale %s rows: %v", table, err)
		}
		if deletedRows != 0 {
			t.Fatalf("stale %s rows after delete = %d, want 0", table, deletedRows)
		}
	}
}

func TestRTreeBackendDeleteTrailsIsAtomic(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	trails := []Trail{
		{ID: "first", Parts: [][]Coordinate{{{Lat: 47, Lon: 8}, {Lat: 47, Lon: 8.1}}}},
		{ID: "blocked", Parts: [][]Coordinate{{{Lat: 46, Lon: 7}, {Lat: 46, Lon: 7.1}}}},
	}
	if err := backend.fullIndex(ctx, trails); err != nil {
		t.Fatal(err)
	}
	if _, err := backend.db.ExecContext(ctx, `
		CREATE TRIGGER fail_rtree_delete
		BEFORE DELETE ON trails
		WHEN OLD.id = 'blocked'
		BEGIN
			SELECT RAISE(ABORT, 'forced rtree delete failure');
		END
	`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = backend.db.Exec(`DROP TRIGGER IF EXISTS fail_rtree_delete`)
	})

	if err := backend.deleteTrails(ctx, []string{"first", "blocked"}); err == nil {
		t.Fatal("delete batch with forced failure succeeded")
	}
	count, err := backend.documentCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("document count after rolled-back delete = %d, want 2", count)
	}
	for _, trail := range trails {
		ids, err := backend.queryCandidates(ctx, trail.Start(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if _, found := ids[trail.ID]; !found {
			t.Fatalf("trail %q bounds were not restored by rollback", trail.ID)
		}
	}
}

func TestRTreeBackendDeleteTrailsValidatesIDsAndCancellation(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()
	if err := backend.fullIndex(ctx, []Trail{{ID: "kept", Parts: [][]Coordinate{{{Lat: 47, Lon: 8}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := backend.deleteTrails(ctx, []string{" "}); err == nil {
		t.Fatal("empty delete ID succeeded")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := backend.deleteTrails(cancelled, []string{"kept"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled delete error = %v, want context.Canceled", err)
	}
	if count, err := backend.documentCount(ctx); err != nil || count != 1 {
		t.Fatalf("document count after rejected deletes = %d, err %v; want 1", count, err)
	}
}

func TestRTreeBackendMultiLineDoesNotBridgeGap(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	trail := Trail{
		ID: "two-parts",
		Parts: [][]Coordinate{
			{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 0.1}},
			{{Lat: 0, Lon: 9.9}, {Lat: 0, Lon: 10}},
		},
	}
	if err := backend.fullIndex(ctx, []Trail{trail}); err != nil {
		t.Fatal(err)
	}

	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 0, Lon: 5}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids)
}

func TestRTreeBackendAntimeridian(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	trail := Trail{
		ID: "dateline",
		Parts: [][]Coordinate{{
			{Lat: 0, Lon: 179.8},
			{Lat: 0, Lon: -179.8},
		}},
	}
	if err := backend.fullIndex(ctx, []Trail{trail}); err != nil {
		t.Fatal(err)
	}

	for _, query := range []Coordinate{
		{Lat: 0, Lon: 179.95},
		{Lat: 0, Lon: -179.95},
		{Lat: 0, Lon: 180},
	} {
		ids, err := backend.queryCandidates(ctx, query, 20_000)
		if err != nil {
			t.Fatalf("query at %#v: %v", query, err)
		}
		assertRTreeCandidateIDs(t, ids, "dateline")
	}

	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 0, Lon: 0}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids)

	var boxes int
	if err := backend.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM segment_bounds`).Scan(&boxes); err != nil {
		t.Fatal(err)
	}
	if boxes != 2 {
		t.Fatalf("antimeridian segment boxes = %d, want 2", boxes)
	}
}

func TestRTreeBackendPoleQueryUsesAllLongitudes(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	trail := Trail{
		ID:    "near-pole",
		Parts: [][]Coordinate{{{Lat: 89.95, Lon: 120}}},
	}
	if err := backend.fullIndex(ctx, []Trail{trail}); err != nil {
		t.Fatal(err)
	}

	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 90, Lon: 0}, 6000)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids, "near-pole")
}

func TestRTreeBackendSegmentBoundsIncludeGreatCircleLatitudeExtremum(t *testing.T) {
	backend, _ := newTestRTreeBackend(t)
	ctx := context.Background()

	trail := Trail{
		ID: "polar-arc",
		Parts: [][]Coordinate{{
			{Lat: 80, Lon: -45},
			{Lat: 80, Lon: 45},
		}},
	}
	if err := backend.fullIndex(ctx, []Trail{trail}); err != nil {
		t.Fatal(err)
	}

	// The minor great-circle arc reaches roughly 82.89 degrees north even
	// though both endpoints are at 80 degrees.
	ids, err := backend.queryCandidates(ctx, Coordinate{Lat: 82.89, Lon: 0}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	assertRTreeCandidateIDs(t, ids, "polar-arc")
}

func TestRTreeBackendSegmentBoundsContainGreatCircleArc(t *testing.T) {
	segments := []struct {
		name  string
		start Coordinate
		end   Coordinate
	}{
		{name: "alpine", start: Coordinate{Lat: 46, Lon: 7}, end: Coordinate{Lat: 48, Lon: 11}},
		{name: "north", start: Coordinate{Lat: 80, Lon: -45}, end: Coordinate{Lat: 80, Lon: 45}},
		{name: "south", start: Coordinate{Lat: -75, Lon: -160}, end: Coordinate{Lat: -70, Lon: 170}},
		{name: "antimeridian", start: Coordinate{Lat: 10, Lon: 170}, end: Coordinate{Lat: 15, Lon: -165}},
		{name: "over-pole", start: Coordinate{Lat: 89, Lon: -90}, end: Coordinate{Lat: 89, Lon: 90}},
	}

	for _, segment := range segments {
		t.Run(segment.name, func(t *testing.T) {
			backend, _ := newTestRTreeBackend(t)
			ctx := context.Background()
			trail := Trail{ID: segment.name, Parts: [][]Coordinate{{segment.start, segment.end}}}
			if err := backend.fullIndex(ctx, []Trail{trail}); err != nil {
				t.Fatal(err)
			}

			for _, fraction := range []float64{0, 0.25, 0.5, 0.75, 1} {
				point := interpolateRTreeGreatCircle(segment.start, segment.end, fraction)
				ids, err := backend.queryCandidates(ctx, point, 0.01)
				if err != nil {
					t.Fatalf("query fraction %.2f at %#v: %v", fraction, point, err)
				}
				if _, ok := ids[segment.name]; !ok || len(ids) != 1 {
					t.Fatalf("query fraction %.2f at %#v returned %#v, want %q", fraction, point, ids, segment.name)
				}
			}
		})
	}
}

func interpolateRTreeGreatCircle(start, end Coordinate, fraction float64) Coordinate {
	if fraction <= 0 {
		return start
	}
	if fraction >= 1 {
		return end
	}
	a := rtreeCoordinateVector(start)
	b := rtreeCoordinateVector(end)
	angle := rtreeVectorAngle(a, b)
	if angle < 1e-14 {
		return start
	}
	sinAngle := math.Sin(angle)
	interpolated := a.scale(math.Sin((1-fraction)*angle) / sinAngle)
	interpolated = rtreeVector{
		x: interpolated.x + b.x*math.Sin(fraction*angle)/sinAngle,
		y: interpolated.y + b.y*math.Sin(fraction*angle)/sinAngle,
		z: interpolated.z + b.z*math.Sin(fraction*angle)/sinAngle,
	}
	interpolated = interpolated.scale(1 / interpolated.norm())
	return Coordinate{
		Lat: math.Atan2(interpolated.z, math.Hypot(interpolated.x, interpolated.y)) * 180 / math.Pi,
		Lon: math.Atan2(interpolated.y, interpolated.x) * 180 / math.Pi,
	}
}

func TestRTreeBackendDiskBytesAndDocumentCount(t *testing.T) {
	backend, path := newTestRTreeBackend(t)
	ctx := context.Background()

	trails := []Trail{
		{ID: "one", Parts: [][]Coordinate{{{Lat: 47, Lon: 8}, {Lat: 47.1, Lon: 8.1}}}},
		{ID: "two", Parts: [][]Coordinate{{{Lat: 46, Lon: 7}, {Lat: 46.1, Lon: 7.1}}}},
	}
	if err := backend.fullIndex(ctx, trails); err != nil {
		t.Fatal(err)
	}

	count, err := backend.documentCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("document count = %d, want 2", count)
	}

	bytes, err := backend.diskBytes()
	if err != nil {
		t.Fatal(err)
	}
	if bytes <= 0 {
		t.Fatalf("disk bytes = %d for %q, want > 0", bytes, path)
	}
	usedBytes, err := backend.usedDatabaseBytes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usedBytes <= 0 || usedBytes > bytes {
		t.Fatalf("used database bytes = %d, total disk bytes = %d", usedBytes, bytes)
	}

	if err := backend.fullIndex(ctx, trails[:1]); err != nil {
		t.Fatal(err)
	}
	count, err = backend.documentCount(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("document count after replacement = %d, want 1", count)
	}
}

func TestSQLiteFilePathTreatsMemoryDataSourcesAsNonFiles(t *testing.T) {
	for _, dataSourceName := range []string{
		":memory:",
		"file::memory:?cache=shared",
		"file:benchmark?mode=memory&cache=shared",
	} {
		if path := sqliteFilePath(dataSourceName); path != "" {
			t.Errorf("sqliteFilePath(%q) = %q, want empty", dataSourceName, path)
		}
	}
}

func newTestRTreeBackend(t *testing.T) (*rtreeBackend, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rtree.sqlite")
	backend, err := newRTreeBackend(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := backend.close(); err != nil {
			t.Errorf("close rtree backend: %v", err)
		}
	})
	return backend, path
}

func assertRTreeCandidateIDs(t *testing.T, got map[string]struct{}, want ...string) {
	t.Helper()
	wantSet := make(map[string]struct{}, len(want))
	for _, id := range want {
		wantSet[id] = struct{}{}
	}
	if !reflect.DeepEqual(got, wantSet) {
		t.Fatalf("candidate IDs = %#v, want %#v", got, wantSet)
	}
}
