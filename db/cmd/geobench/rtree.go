package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

type rtreeBackend struct {
	db        *sql.DB
	path      string
	closeOnce sync.Once
	closeErr  error
}

type rtreePreparedTrail struct {
	id       string
	scenario string
	geometry []byte
	bounds   []rtreeSegmentBounds
}

type rtreeSegmentBounds struct {
	minLon float64
	maxLon float64
	minLat float64
	maxLat float64
}

type rtreeLongitudeRange struct {
	min float64
	max float64
}

func newRTreeBackend(path string) (*rtreeBackend, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("rtree database path is empty")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open rtree database: %w", err)
	}

	// Besides making :memory: databases reliable, one connection ensures the
	// connection-local SQLite pragmas below apply during indexing and the
	// serial engine workload. The runner temporarily raises this limit for its
	// read-only concurrent product workload, then restores it before mutations.
	db.SetMaxOpenConns(1)

	backend := &rtreeBackend{db: db, path: sqliteFilePath(path)}
	if err := backend.initialize(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return backend, nil
}

func (b *rtreeBackend) initialize() error {
	if err := b.db.Ping(); err != nil {
		return fmt.Errorf("connect to rtree database: %w", err)
	}

	for _, statement := range []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
	} {
		if _, err := b.db.Exec(statement); err != nil {
			return fmt.Errorf("configure rtree database: %w", err)
		}
	}

	tx, err := b.db.Begin()
	if err != nil {
		return fmt.Errorf("begin rtree schema transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS trails (
			id TEXT PRIMARY KEY NOT NULL,
			scenario TEXT NOT NULL,
			geometry BLOB NOT NULL
		) WITHOUT ROWID`,
		`CREATE TABLE IF NOT EXISTS segments (
			id INTEGER PRIMARY KEY,
			trail_id TEXT NOT NULL REFERENCES trails(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS segments_trail_id ON segments(trail_id)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS segment_bounds USING rtree(
			id,
			min_lon, max_lon,
			min_lat, max_lat
		)`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement); err != nil {
			return fmt.Errorf("initialize rtree schema: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rtree schema: %w", err)
	}
	return nil
}

func (b *rtreeBackend) fullIndex(ctx context.Context, trails []Trail) error {
	prepared, err := prepareRTreeTrailsContext(ctx, trails)
	if err != nil {
		return err
	}
	return b.writeTrails(ctx, prepared, true)
}

func (b *rtreeBackend) update(ctx context.Context, trails []Trail) error {
	prepared, err := prepareRTreeTrailsContext(ctx, trails)
	if err != nil {
		return err
	}
	return b.writeTrails(ctx, prepared, false)
}

// deleteTrails removes route metadata, geometry, segments, and RTree bounds in
// one transaction. segment_bounds is a virtual table and therefore cannot use
// a foreign key cascade; its rows must be removed before their segment rows.
// Unknown IDs are deliberately idempotent.
func (b *rtreeBackend) deleteTrails(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	for index, id := range ids {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("delete rtree trail at index %d: trail ID is empty", index)
		}
	}

	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rtree delete transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	deleteBounds, err := tx.PrepareContext(ctx,
		`DELETE FROM segment_bounds WHERE id IN (SELECT id FROM segments WHERE trail_id = ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare rtree bounds deletion: %w", err)
	}
	defer deleteBounds.Close()

	deleteSegments, err := tx.PrepareContext(ctx, `DELETE FROM segments WHERE trail_id = ?`)
	if err != nil {
		return fmt.Errorf("prepare rtree segment deletion: %w", err)
	}
	defer deleteSegments.Close()

	deleteTrail, err := tx.PrepareContext(ctx, `DELETE FROM trails WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("prepare rtree trail deletion: %w", err)
	}
	defer deleteTrail.Close()

	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := deleteBounds.ExecContext(ctx, id); err != nil {
			return fmt.Errorf("delete rtree bounds for trail %q: %w", id, err)
		}
		if _, err := deleteSegments.ExecContext(ctx, id); err != nil {
			return fmt.Errorf("delete rtree segments for trail %q: %w", id, err)
		}
		if _, err := deleteTrail.ExecContext(ctx, id); err != nil {
			return fmt.Errorf("delete rtree trail %q: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rtree delete transaction: %w", err)
	}
	return nil
}

func prepareRTreeTrails(trails []Trail) ([]rtreePreparedTrail, error) {
	return prepareRTreeTrailsContext(context.Background(), trails)
}

func prepareRTreeTrailsContext(ctx context.Context, trails []Trail) ([]rtreePreparedTrail, error) {
	prepared := make([]rtreePreparedTrail, 0, len(trails))
	seen := make(map[string]struct{}, len(trails))

	for _, trail := range trails {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if trail.ID == "" {
			return nil, errors.New("prepare rtree trail: trail ID is empty")
		}
		if _, ok := seen[trail.ID]; ok {
			return nil, fmt.Errorf("prepare rtree trail %q: duplicate trail ID", trail.ID)
		}
		seen[trail.ID] = struct{}{}

		bounds, err := rtreeBoundsForTrail(trail)
		if err != nil {
			return nil, fmt.Errorf("prepare rtree trail %q: %w", trail.ID, err)
		}
		geometry, err := json.Marshal(trail.Parts)
		if err != nil {
			return nil, fmt.Errorf("prepare rtree trail %q geometry: %w", trail.ID, err)
		}

		prepared = append(prepared, rtreePreparedTrail{
			id:       trail.ID,
			scenario: trail.Scenario,
			geometry: geometry,
			bounds:   bounds,
		})
	}

	return prepared, nil
}

func rtreeBoundsForTrail(trail Trail) ([]rtreeSegmentBounds, error) {
	var bounds []rtreeSegmentBounds
	for partIndex, part := range trail.Parts {
		for coordinateIndex, coordinate := range part {
			if err := validateRTreeCoordinate(coordinate); err != nil {
				return nil, fmt.Errorf("part %d coordinate %d: %w", partIndex, coordinateIndex, err)
			}
		}

		switch len(part) {
		case 0:
			continue
		case 1:
			coordinate := part[0]
			bounds = append(bounds, rtreeSegmentBounds{
				minLon: coordinate.Lon,
				maxLon: coordinate.Lon,
				minLat: coordinate.Lat,
				maxLat: coordinate.Lat,
			})
		default:
			for index := 1; index < len(part); index++ {
				bounds = append(bounds, rtreeBoundsForSegment(part[index-1], part[index])...)
			}
		}
	}
	return bounds, nil
}

func validateRTreeCoordinate(coordinate Coordinate) error {
	if math.IsNaN(coordinate.Lat) || math.IsInf(coordinate.Lat, 0) || coordinate.Lat < -90 || coordinate.Lat > 90 {
		return fmt.Errorf("invalid latitude %v", coordinate.Lat)
	}
	if math.IsNaN(coordinate.Lon) || math.IsInf(coordinate.Lon, 0) || coordinate.Lon < -180 || coordinate.Lon > 180 {
		return fmt.Errorf("invalid longitude %v", coordinate.Lon)
	}
	return nil
}

func rtreeBoundsForSegment(a, c Coordinate) []rtreeSegmentBounds {
	minLat, maxLat := rtreeGreatCircleLatitudeBounds(a, c)
	minLon, maxLon := orderedFloats(a.Lon, c.Lon)
	if maxLon-minLon <= 180 {
		return []rtreeSegmentBounds{{
			minLon: minLon,
			maxLon: maxLon,
			minLat: minLat,
			maxLat: maxLat,
		}}
	}

	// A segment whose endpoints differ by more than 180 degrees crosses the
	// antimeridian along the short path. Storing two boxes avoids turning it
	// into an almost-worldwide candidate box.
	return []rtreeSegmentBounds{
		{minLon: maxLon, maxLon: 180, minLat: minLat, maxLat: maxLat},
		{minLon: -180, maxLon: minLon, minLat: minLat, maxLat: maxLat},
	}
}

// rtreeGreatCircleLatitudeBounds includes latitude extrema reached between
// the endpoints of the minor great-circle arc. Endpoint-only latitude bounds
// can otherwise produce false negatives for long east-west segments that bow
// toward a pole.
func rtreeGreatCircleLatitudeBounds(a, c Coordinate) (float64, float64) {
	minLat, maxLat := orderedFloats(a.Lat, c.Lat)
	start := rtreeCoordinateVector(a)
	end := rtreeCoordinateVector(c)
	normal := start.cross(end)
	normalLength := normal.norm()
	arcLength := rtreeVectorAngle(start, end)
	if normalLength < 1e-14 || arcLength < 1e-14 {
		return minLat, maxLat
	}

	unitNormal := normal.scale(1 / normalLength)
	// Project the north pole onto the great-circle plane. Its normalized
	// projection is the northernmost point on the complete great circle.
	northPole := rtreeVector{z: 1}
	northProjection := northPole.subtract(unitNormal.scale(northPole.dot(unitNormal)))
	projectionLength := northProjection.norm()
	if projectionLength < 1e-14 {
		return minLat, maxLat
	}

	north := northProjection.scale(1 / projectionLength)
	if rtreeVectorOnArc(start, end, north, arcLength) {
		maxLat = math.Max(maxLat, math.Asin(clampRTreeUnit(north.z))*180/math.Pi)
	}
	south := north.scale(-1)
	if rtreeVectorOnArc(start, end, south, arcLength) {
		minLat = math.Min(minLat, math.Asin(clampRTreeUnit(south.z))*180/math.Pi)
	}
	return minLat, maxLat
}

type rtreeVector struct {
	x float64
	y float64
	z float64
}

func rtreeCoordinateVector(coordinate Coordinate) rtreeVector {
	latitude := coordinate.Lat * math.Pi / 180
	longitude := coordinate.Lon * math.Pi / 180
	sinLat, cosLat := math.Sincos(latitude)
	sinLon, cosLon := math.Sincos(longitude)
	return rtreeVector{x: cosLat * cosLon, y: cosLat * sinLon, z: sinLat}
}

func (v rtreeVector) dot(other rtreeVector) float64 {
	return v.x*other.x + v.y*other.y + v.z*other.z
}

func (v rtreeVector) cross(other rtreeVector) rtreeVector {
	return rtreeVector{
		x: v.y*other.z - v.z*other.y,
		y: v.z*other.x - v.x*other.z,
		z: v.x*other.y - v.y*other.x,
	}
}

func (v rtreeVector) subtract(other rtreeVector) rtreeVector {
	return rtreeVector{x: v.x - other.x, y: v.y - other.y, z: v.z - other.z}
}

func (v rtreeVector) scale(factor float64) rtreeVector {
	return rtreeVector{x: v.x * factor, y: v.y * factor, z: v.z * factor}
}

func (v rtreeVector) norm() float64 {
	return math.Sqrt(v.dot(v))
}

func rtreeVectorAngle(a, c rtreeVector) float64 {
	return math.Atan2(a.cross(c).norm(), clampRTreeUnit(a.dot(c)))
}

func rtreeVectorOnArc(start, end, candidate rtreeVector, arcLength float64) bool {
	throughCandidate := rtreeVectorAngle(start, candidate) + rtreeVectorAngle(candidate, end)
	return math.Abs(throughCandidate-arcLength) <= 1e-10
}

func clampRTreeUnit(value float64) float64 {
	return math.Max(-1, math.Min(1, value))
}

func orderedFloats(a, c float64) (float64, float64) {
	if a <= c {
		return a, c
	}
	return c, a
}

func (b *rtreeBackend) writeTrails(ctx context.Context, trails []rtreePreparedTrail, replaceAll bool) error {
	tx, err := b.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rtree index transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if replaceAll {
		for _, statement := range []string{
			`DELETE FROM segment_bounds`,
			`DELETE FROM segments`,
			`DELETE FROM trails`,
		} {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("clear rtree index: %w", err)
			}
		}
	}

	deleteBounds, err := tx.PrepareContext(ctx,
		`DELETE FROM segment_bounds WHERE id IN (SELECT id FROM segments WHERE trail_id = ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare rtree bounds replacement: %w", err)
	}
	defer deleteBounds.Close()

	deleteSegments, err := tx.PrepareContext(ctx, `DELETE FROM segments WHERE trail_id = ?`)
	if err != nil {
		return fmt.Errorf("prepare rtree segment replacement: %w", err)
	}
	defer deleteSegments.Close()

	upsertTrail, err := tx.PrepareContext(ctx, `
		INSERT INTO trails (id, scenario, geometry) VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			scenario = excluded.scenario,
			geometry = excluded.geometry
	`)
	if err != nil {
		return fmt.Errorf("prepare rtree trail upsert: %w", err)
	}
	defer upsertTrail.Close()

	insertSegment, err := tx.PrepareContext(ctx, `INSERT INTO segments (trail_id) VALUES (?)`)
	if err != nil {
		return fmt.Errorf("prepare rtree segment insert: %w", err)
	}
	defer insertSegment.Close()

	insertBounds, err := tx.PrepareContext(ctx, `
		INSERT INTO segment_bounds (id, min_lon, max_lon, min_lat, max_lat)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare rtree bounds insert: %w", err)
	}
	defer insertBounds.Close()

	for _, trail := range trails {
		if !replaceAll {
			if _, err := deleteBounds.ExecContext(ctx, trail.id); err != nil {
				return fmt.Errorf("delete old rtree bounds for trail %q: %w", trail.id, err)
			}
			if _, err := deleteSegments.ExecContext(ctx, trail.id); err != nil {
				return fmt.Errorf("delete old rtree segments for trail %q: %w", trail.id, err)
			}
		}

		if _, err := upsertTrail.ExecContext(ctx, trail.id, trail.scenario, trail.geometry); err != nil {
			return fmt.Errorf("store rtree geometry for trail %q: %w", trail.id, err)
		}

		for _, bounds := range trail.bounds {
			result, err := insertSegment.ExecContext(ctx, trail.id)
			if err != nil {
				return fmt.Errorf("store rtree segment for trail %q: %w", trail.id, err)
			}
			segmentID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("read rtree segment ID for trail %q: %w", trail.id, err)
			}
			if _, err := insertBounds.ExecContext(
				ctx,
				segmentID,
				bounds.minLon,
				bounds.maxLon,
				bounds.minLat,
				bounds.maxLat,
			); err != nil {
				return fmt.Errorf("index rtree segment bounds for trail %q: %w", trail.id, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rtree index transaction: %w", err)
	}
	return nil
}

func (b *rtreeBackend) queryCandidates(ctx context.Context, point Coordinate, radiusM float64) (map[string]struct{}, error) {
	minLat, maxLat, longitudeRanges, err := rtreeQueryBounds(point, radiusM)
	if err != nil {
		return nil, err
	}

	var query strings.Builder
	query.WriteString(`
		SELECT DISTINCT segments.trail_id
		FROM segment_bounds
		JOIN segments ON segments.id = segment_bounds.id
		WHERE segment_bounds.max_lat >= ?
		  AND segment_bounds.min_lat <= ?
		  AND (
	`)
	arguments := []any{minLat, maxLat}
	for index, longitudeRange := range longitudeRanges {
		if index > 0 {
			query.WriteString(" OR ")
		}
		query.WriteString("(segment_bounds.max_lon >= ? AND segment_bounds.min_lon <= ?)")
		arguments = append(arguments, longitudeRange.min, longitudeRange.max)
	}
	query.WriteString(")")

	rows, err := b.db.QueryContext(ctx, query.String(), arguments...)
	if err != nil {
		return nil, fmt.Errorf("query rtree candidates: %w", err)
	}
	defer rows.Close()

	ids := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan rtree candidate: %w", err)
		}
		ids[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rtree candidates: %w", err)
	}
	return ids, nil
}

func rtreeQueryBounds(point Coordinate, radiusM float64) (float64, float64, []rtreeLongitudeRange, error) {
	if err := validateRTreeCoordinate(point); err != nil {
		return 0, 0, nil, fmt.Errorf("invalid rtree query point: %w", err)
	}
	if math.IsNaN(radiusM) || math.IsInf(radiusM, 0) || radiusM < 0 {
		return 0, 0, nil, fmt.Errorf("invalid rtree query radius %v", radiusM)
	}

	// A tiny outward margin prevents double-precision trigonometry and the
	// RTree module's float32 coordinate storage from excluding an exact
	// endpoint at a nominally zero-radius query. Exact verification still uses
	// the caller's unmodified radius.
	angularRadius := (radiusM + 0.01) / earthRadiusMeters
	if angularRadius >= math.Pi {
		return -90, 90, []rtreeLongitudeRange{{min: -180, max: 180}}, nil
	}

	latitude := point.Lat * math.Pi / 180
	minLatitude := latitude - angularRadius
	maxLatitude := latitude + angularRadius
	clampedMinLatitude := math.Max(minLatitude, -math.Pi/2)
	clampedMaxLatitude := math.Min(maxLatitude, math.Pi/2)
	minLat := clampedMinLatitude * 180 / math.Pi
	maxLat := clampedMaxLatitude * 180 / math.Pi

	if minLatitude <= -math.Pi/2 || maxLatitude >= math.Pi/2 {
		return minLat, maxLat, []rtreeLongitudeRange{{min: -180, max: 180}}, nil
	}

	ratio := math.Sin(angularRadius) / math.Cos(latitude)
	ratio = math.Max(-1, math.Min(1, ratio))
	longitudeDelta := math.Asin(ratio) * 180 / math.Pi
	minLon := point.Lon - longitudeDelta
	maxLon := point.Lon + longitudeDelta

	switch {
	case minLon < -180:
		return minLat, maxLat, []rtreeLongitudeRange{
			{min: -180, max: maxLon},
			{min: minLon + 360, max: 180},
		}, nil
	case maxLon > 180:
		return minLat, maxLat, []rtreeLongitudeRange{
			{min: minLon, max: 180},
			{min: -180, max: maxLon - 360},
		}, nil
	case minLon == maxLon && (minLon == -180 || minLon == 180):
		// -180 and +180 denote the same meridian. Include both spellings for
		// zero-radius queries as well.
		return minLat, maxLat, []rtreeLongitudeRange{
			{min: -180, max: -180},
			{min: 180, max: 180},
		}, nil
	default:
		return minLat, maxLat, []rtreeLongitudeRange{{min: minLon, max: maxLon}}, nil
	}
}

func (b *rtreeBackend) diskBytes() (int64, error) {
	if b.path == "" {
		return 0, nil
	}

	var total int64
	foundDatabase := false
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		info, err := os.Stat(b.path + suffix)
		if err != nil {
			if os.IsNotExist(err) && suffix != "" {
				continue
			}
			return 0, fmt.Errorf("stat rtree database file %q: %w", b.path+suffix, err)
		}
		if suffix == "" {
			foundDatabase = true
		}
		total += info.Size()
	}
	if !foundDatabase {
		return 0, fmt.Errorf("rtree database file %q does not exist", b.path)
	}
	return total, nil
}

func (b *rtreeBackend) usedDatabaseBytes(ctx context.Context) (int64, error) {
	var pageSize, pageCount, freelistCount int64
	for _, query := range []struct {
		statement string
		value     *int64
	}{
		{statement: `PRAGMA page_size`, value: &pageSize},
		{statement: `PRAGMA page_count`, value: &pageCount},
		{statement: `PRAGMA freelist_count`, value: &freelistCount},
	} {
		if err := b.db.QueryRowContext(ctx, query.statement).Scan(query.value); err != nil {
			return 0, fmt.Errorf("read SQLite storage stats: %w", err)
		}
	}
	if pageSize <= 0 || pageCount < 0 || freelistCount < 0 || freelistCount > pageCount {
		return 0, fmt.Errorf("invalid SQLite storage stats: page_size=%d page_count=%d freelist_count=%d", pageSize, pageCount, freelistCount)
	}
	return pageSize * (pageCount - freelistCount), nil
}

func (b *rtreeBackend) storedTrail(ctx context.Context, id string) (Trail, error) {
	var scenario string
	var geometry []byte
	if err := b.db.QueryRowContext(ctx, `SELECT scenario, geometry FROM trails WHERE id = ?`, id).Scan(&scenario, &geometry); err != nil {
		return Trail{}, fmt.Errorf("read SQLite trail %q: %w", id, err)
	}
	var parts [][]Coordinate
	if err := json.Unmarshal(geometry, &parts); err != nil {
		return Trail{}, fmt.Errorf("decode SQLite trail %q: %w", id, err)
	}
	return Trail{ID: id, Scenario: scenario, Parts: parts}, nil
}

// loadCandidateTrails deliberately reads and decodes the stored SQLite BLOBs.
// Query benchmarks use this path for exact refinement so the RTree result is
// not "made exact" with geometry already resident in the harness.
func (b *rtreeBackend) loadCandidateTrails(ctx context.Context, ids map[string]struct{}) (geometryLoad, error) {
	return loadEncodedTrailRows(ctx, b.db, "trails", ids)
}

func sqliteFilePath(dataSourceName string) string {
	lowerDataSourceName := strings.ToLower(dataSourceName)
	if dataSourceName == ":memory:" || strings.HasPrefix(lowerDataSourceName, "file::memory:") || strings.Contains(lowerDataSourceName, "mode=memory") {
		return ""
	}
	if !strings.HasPrefix(dataSourceName, "file:") {
		return dataSourceName
	}

	fileName := strings.TrimPrefix(dataSourceName, "file:")
	if queryIndex := strings.IndexByte(fileName, '?'); queryIndex >= 0 {
		fileName = fileName[:queryIndex]
	}
	decoded, err := url.PathUnescape(fileName)
	if err != nil {
		return fileName
	}
	return decoded
}

func (b *rtreeBackend) documentCount(ctx context.Context) (int64, error) {
	var count int64
	if err := b.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trails`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count rtree documents: %w", err)
	}
	return count, nil
}

func (b *rtreeBackend) close() error {
	b.closeOnce.Do(func() {
		b.closeErr = b.db.Close()
	})
	return b.closeErr
}
