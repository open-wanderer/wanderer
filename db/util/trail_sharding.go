package util

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type quadNodeRecord struct {
	Id          string  `db:"id"`
	Parent      *string `db:"parent"`
	Depth       int     `db:"depth"`
	MinLat      float64 `db:"min_lat"`
	MinLon      float64 `db:"min_lon"`
	MaxLat      float64 `db:"max_lat"`
	MaxLon      float64 `db:"max_lon"`
	TrailCount  int     `db:"trail_count"`
	IsLeaf      bool    `db:"is_leaf"`
	SplitMode   string  `db:"split_mode"`
	CreatedFrom *int64  `db:"created_from"`
	CreatedTo   *int64  `db:"created_to"`
	Path        string  `db:"path"`
}

type trailNodeEntry struct {
	Id        string  `db:"id"`
	MinLat    float64 `db:"min_lat"`
	MinLon    float64 `db:"min_lon"`
	MaxLat    float64 `db:"max_lat"`
	MaxLon    float64 `db:"max_lon"`
	CreatedAt int64   `db:"created_unix"`
}

type timeBucketRecord struct {
	Id          string `db:"id"`
	CreatedFrom int64  `db:"created_from"`
	CreatedTo   int64  `db:"created_to"`
	TrailCount  int    `db:"trail_count"`
}

func AssignTrailShards(app core.App, trail *core.Record) error {
	return assignTrailShards(app, trail, BucketsEnabled())
}

func AssignTrailShardsForced(app core.App, trail *core.Record) error {
	return assignTrailShards(app, trail, true)
}

func assignTrailShards(app core.App, trail *core.Record, enabled bool) error {
	if !enabled {
		return nil
	}
	minLat, minLon, maxLat, maxLon, ok := trailBoundsFromRecord(trail)
	var bbox TrailBoundingBox
	if ok {
		bbox = normalizeTrailBoundingBox(TrailBoundingBox{
			MinLat: minLat,
			MinLon: minLon,
			MaxLat: maxLat,
			MaxLon: maxLon,
		})
	}

	if _, err := AssignTrailTimeBucket(app, trail); err != nil {
		return err
	}
	if ok {
		if _, err := AssignTrailToQuadNodes(app, trail.Id, bbox, trail.GetDateTime("created").Time()); err != nil {
			return err
		}
	}

	return nil
}

func ReassignTrailShards(app core.App, trail *core.Record) error {
	if !BucketsEnabled() {
		return nil
	}
	if err := RemoveTrailFromTimeBuckets(app, trail.Id); err != nil {
		return err
	}
	if err := RemoveTrailFromQuadNodes(app, trail.Id); err != nil {
		return err
	}
	return AssignTrailShards(app, trail)
}

func RemoveTrailShards(app core.App, trail *core.Record) error {
	if !BucketsEnabled() {
		return nil
	}
	if err := RemoveTrailFromTimeBuckets(app, trail.Id); err != nil {
		return err
	}
	return RemoveTrailFromQuadNodes(app, trail.Id)
}

func AssignTrailToQuadNodes(app core.App, trailId string, bbox TrailBoundingBox, createdAt time.Time) ([]string, error) {
	root, err := ensureRootQuadNode(app)
	if err != nil {
		return nil, err
	}
	return assignTrailToNode(app, root.Id, trailId, bbox, createdAt)
}

func RemoveTrailFromQuadNodes(app core.App, trailId string) error {
	var rows []struct {
		QuadNode string `db:"quad_node"`
	}
	if err := app.DB().NewQuery(`SELECT quad_node FROM trail_quad_nodes WHERE trail = {:trail}`).
		Bind(dbx.Params{"trail": trailId}).
		All(&rows); err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	if _, err := app.DB().NewQuery(`DELETE FROM trail_quad_nodes WHERE trail = {:trail}`).
		Bind(dbx.Params{"trail": trailId}).
		Execute(); err != nil {
		return err
	}

	for _, row := range rows {
		if err := incrementNodeTrailCount(app, row.QuadNode, -1); err != nil {
			return err
		}
	}

	return nil
}

func TrailQuadNodeIDs(app core.App, trailId string) ([]string, error) {
	if !BucketsEnabled() {
		return []string{}, nil
	}
	var rows []struct {
		QuadNode string `db:"quad_node"`
	}
	if err := app.DB().NewQuery(`SELECT quad_node FROM trail_quad_nodes WHERE trail = {:trail}`).
		Bind(dbx.Params{"trail": trailId}).
		All(&rows); err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.QuadNode)
	}
	return ids, nil
}

func TrailTimeBucketIDs(app core.App, trailId string) ([]string, error) {
	if !BucketsEnabled() {
		return []string{}, nil
	}
	var rows []struct {
		Bucket string `db:"bucket"`
	}
	if err := app.DB().NewQuery(`SELECT bucket FROM trail_time_bucket_entries WHERE trail = {:trail}`).
		Bind(dbx.Params{"trail": trailId}).
		All(&rows); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Bucket)
	}
	return ids, nil
}

func ensureRootQuadNode(app core.App) (*quadNodeRecord, error) {
	nodes, err := loadQuadNodes(app, "depth = 0 AND path = 'root'", nil)
	if err != nil {
		return nil, err
	}
	if len(nodes) > 0 {
		return &nodes[0], nil
	}

	collection, err := app.FindCollectionByNameOrId("quad_nodes")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("depth", 0)
	record.Set("min_lat", -90.0)
	record.Set("min_lon", -180.0)
	record.Set("max_lat", 90.0)
	record.Set("max_lon", 180.0)
	record.Set("trail_count", 0)
	record.Set("is_leaf", true)
	record.Set("split_mode", "spatial")
	record.Set("path", "root")

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return &quadNodeRecord{
		Id:         record.Id,
		Depth:      0,
		MinLat:     -90.0,
		MinLon:     -180.0,
		MaxLat:     90.0,
		MaxLon:     180.0,
		TrailCount: 0,
		IsLeaf:     true,
		SplitMode:  "spatial",
		Path:       "root",
	}, nil
}

func assignTrailToNode(app core.App, nodeId, trailId string, bbox TrailBoundingBox, createdAt time.Time) ([]string, error) {
	node, err := loadQuadNode(app, nodeId)
	if err != nil {
		return nil, err
	}

	if !nodeIntersects(node, bbox) {
		return nil, nil
	}

	if node.IsLeaf {
		inserted, err := insertTrailQuadNode(app, trailId, node.Id)
		if err != nil {
			return nil, err
		}
		if inserted {
			if err := incrementNodeTrailCount(app, node.Id, 1); err != nil {
				return nil, err
			}
			node.TrailCount++
		}

		if node.TrailCount > QuadTreeBucketMax {
			if node.Depth < QuadTreeMaxDepth {
				if err := spatialSplitNode(app, node); err != nil {
					return nil, err
				}
			} else {
				if err := logicalSplitNode(app, node); err != nil {
					return nil, err
				}
			}
			return TrailQuadNodeIDs(app, trailId)
		}

		return []string{node.Id}, nil
	}

	children, err := loadQuadNodes(app, "parent = {:parent}", dbx.Params{"parent": node.Id})
	if err != nil {
		return nil, err
	}

	var leafIds []string
	for _, child := range children {
		ids, err := assignTrailToNode(app, child.Id, trailId, bbox, createdAt)
		if err != nil {
			return nil, err
		}
		leafIds = append(leafIds, ids...)
	}

	return leafIds, nil
}

func spatialSplitNode(app core.App, node *quadNodeRecord) error {
	if err := updateNodeLeafState(app, node.Id, false, "spatial", 0); err != nil {
		return err
	}

	latMid := (node.MinLat + node.MaxLat) / 2.0
	lonMid := (node.MinLon + node.MaxLon) / 2.0
	if node.Depth == 0 {
		latMid = RootSplitLat
		lonMid = RootSplitLon
	}

	children := []struct {
		Suffix string
		MinLat float64
		MinLon float64
		MaxLat float64
		MaxLon float64
	}{
		{"sw", node.MinLat, node.MinLon, latMid, lonMid},
		{"se", node.MinLat, lonMid, latMid, node.MaxLon},
		{"nw", latMid, node.MinLon, node.MaxLat, lonMid},
		{"ne", latMid, lonMid, node.MaxLat, node.MaxLon},
	}

	for _, child := range children {
		if _, err := createQuadNode(app, node.Id, node.Depth+1, child.MinLat, child.MinLon, child.MaxLat, child.MaxLon, 0, true, "spatial", nil, nil, node.Path+"/"+child.Suffix); err != nil {
			return err
		}
	}

	trails, err := loadTrailsForQuadNode(app, node.Id)
	if err != nil {
		return err
	}

	if _, err := app.DB().NewQuery(`DELETE FROM trail_quad_nodes WHERE quad_node = {:node}`).
		Bind(dbx.Params{"node": node.Id}).
		Execute(); err != nil {
		return err
	}

	for _, trail := range trails {
		trailBbox := TrailBoundingBox{
			MinLat: trail.MinLat,
			MinLon: trail.MinLon,
			MaxLat: trail.MaxLat,
			MaxLon: trail.MaxLon,
		}
		createdAt := time.Unix(trail.CreatedAt, 0)
		if _, err := assignTrailToNode(app, node.Id, trail.Id, trailBbox, createdAt); err != nil {
			return err
		}
	}

	return nil
}

func logicalSplitNode(app core.App, node *quadNodeRecord) error {
	if err := updateNodeLeafState(app, node.Id, false, "logical", 0); err != nil {
		return err
	}

	trails, err := loadTrailsForQuadNode(app, node.Id)
	if err != nil {
		return err
	}

	if _, err := app.DB().NewQuery(`DELETE FROM trail_quad_nodes WHERE quad_node = {:node}`).
		Bind(dbx.Params{"node": node.Id}).
		Execute(); err != nil {
		return err
	}

	sort.Slice(trails, func(i, j int) bool {
		return trails[i].CreatedAt < trails[j].CreatedAt
	})

	for i := 0; i < len(trails); i += QuadTreeBucketMax {
		end := i + QuadTreeBucketMax
		if end > len(trails) {
			end = len(trails)
		}

		chunk := trails[i:end]
		createdFrom := chunk[0].CreatedAt
		createdTo := chunk[len(chunk)-1].CreatedAt
		suffix := fmt.Sprintf("t%04d", i/QuadTreeBucketMax)
		childId, err := createQuadNode(app, node.Id, node.Depth+1, node.MinLat, node.MinLon, node.MaxLat, node.MaxLon, len(chunk), true, "logical", &createdFrom, &createdTo, node.Path+"/"+suffix)
		if err != nil {
			return err
		}

		for _, trail := range chunk {
			if _, err := insertTrailQuadNode(app, trail.Id, childId); err != nil {
				return err
			}
		}
	}

	return nil
}

func loadQuadNode(app core.App, nodeId string) (*quadNodeRecord, error) {
	nodes, err := loadQuadNodes(app, "id = {:id}", dbx.Params{"id": nodeId})
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("quad_node not found: %s", nodeId)
	}
	return &nodes[0], nil
}

func loadQuadNodes(app core.App, filter string, params dbx.Params) ([]quadNodeRecord, error) {
	var nodes []quadNodeRecord
	query := `SELECT id, parent, depth, min_lat, min_lon, max_lat, max_lon, trail_count, is_leaf, split_mode, created_from, created_to, path
		FROM quad_nodes
		WHERE ` + filter
	if params == nil {
		params = dbx.Params{}
	}
	if err := app.DB().NewQuery(query).Bind(params).All(&nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

func loadTrailsForQuadNode(app core.App, nodeId string) ([]trailNodeEntry, error) {
	var trails []trailNodeEntry
	query := `SELECT t.id,
			t.min_lat,
			t.min_lon,
			t.max_lat,
			t.max_lon,
			CAST(strftime('%s', t.created) AS INTEGER) AS created_unix
		FROM trail_quad_nodes tqn
		JOIN trails t ON t.id = tqn.trail
		WHERE tqn.quad_node = {:node}`
	if err := app.DB().NewQuery(query).Bind(dbx.Params{"node": nodeId}).All(&trails); err != nil {
		return nil, err
	}
	return trails, nil
}

func createQuadNode(app core.App, parentId string, depth int, minLat, minLon, maxLat, maxLon float64, trailCount int, isLeaf bool, splitMode string, createdFrom, createdTo *int64, path string) (string, error) {
	collection, err := app.FindCollectionByNameOrId("quad_nodes")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)
	if parentId != "" {
		record.Set("parent", parentId)
	}
	record.Set("depth", depth)
	record.Set("min_lat", minLat)
	record.Set("min_lon", minLon)
	record.Set("max_lat", maxLat)
	record.Set("max_lon", maxLon)
	record.Set("trail_count", trailCount)
	record.Set("is_leaf", isLeaf)
	record.Set("split_mode", splitMode)
	if createdFrom != nil {
		record.Set("created_from", *createdFrom)
	}
	if createdTo != nil {
		record.Set("created_to", *createdTo)
	}
	record.Set("path", path)

	if err := app.Save(record); err != nil {
		return "", err
	}

	return record.Id, nil
}

func insertTrailQuadNode(app core.App, trailId, nodeId string) (bool, error) {
	result, err := app.DB().NewQuery(`INSERT OR IGNORE INTO trail_quad_nodes (trail, quad_node) VALUES ({:trail}, {:node})`).
		Bind(dbx.Params{"trail": trailId, "node": nodeId}).
		Execute()
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func insertTrailTimeBucketEntry(app core.App, trailId, bucketId string) (bool, error) {
	result, err := app.DB().NewQuery(`INSERT OR IGNORE INTO trail_time_bucket_entries (trail, bucket) VALUES ({:trail}, {:bucket})`).
		Bind(dbx.Params{"trail": trailId, "bucket": bucketId}).
		Execute()
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func incrementNodeTrailCount(app core.App, nodeId string, delta int) error {
	_, err := app.DB().NewQuery(`UPDATE quad_nodes
		SET trail_count = CASE
			WHEN trail_count + {:delta} < 0 THEN 0
			ELSE trail_count + {:delta}
		END
		WHERE id = {:id}`).
		Bind(dbx.Params{"delta": delta, "id": nodeId}).
		Execute()
	return err
}

func updateNodeLeafState(app core.App, nodeId string, isLeaf bool, splitMode string, trailCount int) error {
	_, err := app.DB().NewQuery(`UPDATE quad_nodes
		SET is_leaf = {:is_leaf},
			split_mode = {:split_mode},
			trail_count = {:trail_count}
		WHERE id = {:id}`).
		Bind(dbx.Params{
			"is_leaf":     isLeaf,
			"split_mode":  splitMode,
			"trail_count": trailCount,
			"id":          nodeId,
		}).Execute()
	return err
}

func nodeIntersects(node *quadNodeRecord, bbox TrailBoundingBox) bool {
	return node.MaxLat >= bbox.MinLat &&
		node.MinLat <= bbox.MaxLat &&
		node.MaxLon >= bbox.MinLon &&
		node.MinLon <= bbox.MaxLon
}

func normalizeLongitude(lon float64) float64 {
	for lon < -180.0 {
		lon += 360.0
	}
	for lon >= 180.0 {
		lon -= 360.0
	}
	return lon
}

func normalizeTrailBoundingBox(bbox TrailBoundingBox) TrailBoundingBox {
	bbox.MinLon = normalizeLongitude(bbox.MinLon)
	bbox.MaxLon = normalizeLongitude(bbox.MaxLon)
	return bbox
}

func AssignTrailTimeBucket(app core.App, trail *core.Record) (string, error) {
	createdAt := trail.GetDateTime("created").Time().Unix()

	bucket, err := findTimeBucketForTimestamp(app, createdAt)
	if err != nil {
		return "", err
	}
	if bucket == nil {
		bucketId, err := createTimeBucket(app, 0, 0, 0)
		if err != nil {
			return "", err
		}
		bucket = &timeBucketRecord{
			Id:          bucketId,
			CreatedFrom: 0,
			CreatedTo:   0,
			TrailCount:  0,
		}
	}

	if bucket.TrailCount >= QuadTreeBucketMax {
		if bucket.CreatedTo == 0 {
			updated, err := setTimeBucketEndIfOpen(app, bucket.Id, createdAt)
			if err != nil {
				return "", err
			}
			if updated {
				bucketId, err := createTimeBucket(app, createdAt, 0, 0)
				if err != nil {
					return "", err
				}
				bucket = &timeBucketRecord{
					Id:          bucketId,
					CreatedFrom: createdAt,
					CreatedTo:   0,
					TrailCount:  0,
				}
			} else {
				refreshed, err := findTimeBucketForTimestamp(app, createdAt)
				if err != nil {
					return "", err
				}
				if refreshed == nil {
					bucketId, err := createTimeBucket(app, createdAt, 0, 0)
					if err != nil {
						return "", err
					}
					refreshed = &timeBucketRecord{
						Id:          bucketId,
						CreatedFrom: createdAt,
						CreatedTo:   0,
						TrailCount:  0,
					}
				}
				bucket = refreshed
			}
		}
	}

	inserted, err := insertTrailTimeBucketEntry(app, trail.Id, bucket.Id)
	if err != nil {
		return "", err
	}
	if inserted {
		if err := incrementTimeBucketCount(app, bucket.Id, 1); err != nil {
			return "", err
		}
	}

	newId, err := splitTimeBucketIfNeeded(app, bucket.Id, trail.Id)
	if err != nil {
		return "", err
	}

	return newId, nil
}

func RemoveTrailFromTimeBuckets(app core.App, trailId string) error {
	var rows []struct {
		Bucket string `db:"bucket"`
	}
	if err := app.DB().NewQuery(`SELECT bucket FROM trail_time_bucket_entries WHERE trail = {:trail}`).
		Bind(dbx.Params{"trail": trailId}).
		All(&rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	if _, err := app.DB().NewQuery(`DELETE FROM trail_time_bucket_entries WHERE trail = {:trail}`).
		Bind(dbx.Params{"trail": trailId}).
		Execute(); err != nil {
		return err
	}

	for _, row := range rows {
		if err := incrementTimeBucketCount(app, row.Bucket, -1); err != nil {
			return err
		}
	}

	return nil
}

func findTimeBucketForTimestamp(app core.App, createdAt int64) (*timeBucketRecord, error) {
	var buckets []timeBucketRecord
	query := `SELECT id, created_from, created_to, trail_count
		FROM trail_time_buckets
		WHERE (created_from = 0 OR created_from <= {:created})
			AND (created_to = 0 OR created_to > {:created})
		ORDER BY created_from = 0 DESC, created_from
		LIMIT 1`
	if err := app.DB().NewQuery(query).
		Bind(dbx.Params{"created": createdAt}).
		All(&buckets); err != nil {
		return nil, err
	}
	if len(buckets) == 0 {
		return nil, nil
	}
	return &buckets[0], nil
}

func createTimeBucket(app core.App, createdFrom, createdTo int64, trailCount int) (string, error) {
	collection, err := app.FindCollectionByNameOrId("trail_time_buckets")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)
	record.Set("created_from", createdFrom)
	record.Set("created_to", createdTo)
	record.Set("trail_count", trailCount)

	if err := app.Save(record); err != nil {
		return "", err
	}
	return record.Id, nil
}

func incrementTimeBucketCount(app core.App, bucketId string, delta int) error {
	_, err := app.DB().NewQuery(`UPDATE trail_time_buckets
		SET trail_count = CASE
			WHEN trail_count + {:delta} < 0 THEN 0
			ELSE trail_count + {:delta}
		END
		WHERE id = {:id}`).
		Bind(dbx.Params{"delta": delta, "id": bucketId}).
		Execute()
	return err
}

func splitTimeBucketIfNeeded(app core.App, bucketId, focusTrailId string) (string, error) {
	var buckets []timeBucketRecord
	if err := app.DB().NewQuery(`SELECT id, created_from, created_to, trail_count
		FROM trail_time_buckets
		WHERE id = {:id}`).
		Bind(dbx.Params{"id": bucketId}).
		All(&buckets); err != nil {
		return "", err
	}
	if len(buckets) == 0 || buckets[0].TrailCount <= QuadTreeBucketMax {
		return bucketId, nil
	}

	var trails []struct {
		Id        string `db:"id"`
		CreatedAt int64  `db:"created_unix"`
	}
	if err := app.DB().NewQuery(`SELECT t.id, CAST(strftime('%s', t.created) AS INTEGER) AS created_unix
		FROM trail_time_bucket_entries e
		JOIN trails t ON t.id = e.trail
		WHERE e.bucket = {:bucket}
		ORDER BY t.created`).
		Bind(dbx.Params{"bucket": bucketId}).
		All(&trails); err != nil {
		return "", err
	}

	if len(trails) == 0 {
		return bucketId, nil
	}

	newBucketId := bucketId
	chunkStarts := []int{0}
	for i := 0; i < len(trails); {
		end := i + QuadTreeBucketMax
		if end > len(trails) {
			end = len(trails)
		}
		for end < len(trails) && trails[end-1].CreatedAt == trails[end].CreatedAt {
			end++
		}
		i = end
		if i < len(trails) {
			chunkStarts = append(chunkStarts, i)
		}
	}

	for idx, start := range chunkStarts {
		end := len(trails)
		if idx+1 < len(chunkStarts) {
			end = chunkStarts[idx+1]
		}
		chunk := trails[start:end]
		createdFrom := chunk[0].CreatedAt
		var createdTo int64
		if idx+1 < len(chunkStarts) {
			nextStart := chunkStarts[idx+1]
			nextFrom := trails[nextStart].CreatedAt
			createdTo = nextFrom
		}
		chunkBucketId, err := createTimeBucket(app, createdFrom, createdTo, len(chunk))
		if err != nil {
			return "", err
		}

		ids := make([]string, 0, len(chunk))
		for _, trail := range chunk {
			ids = append(ids, trail.Id)
			if trail.Id == focusTrailId {
				newBucketId = chunkBucketId
			}
		}

		inClause, inParams := buildInClause("trail", ids)
		params := dbx.Params{"bucket": chunkBucketId, "old": bucketId}
		for key, value := range inParams {
			params[key] = value
		}
		query := fmt.Sprintf(`UPDATE trail_time_bucket_entries
			SET bucket = {:bucket}
			WHERE %s AND bucket = {:old}`, inClause)
		if _, err := app.DB().NewQuery(query).
			Bind(params).
			Execute(); err != nil {
			return "", err
		}
	}

	if _, err := app.DB().NewQuery(`DELETE FROM trail_time_buckets WHERE id = {:id}`).
		Bind(dbx.Params{"id": bucketId}).
		Execute(); err != nil {
		return "", err
	}

	return newBucketId, nil
}

func setTimeBucketEnd(app core.App, bucketId string, createdTo int64) error {
	_, err := app.DB().NewQuery(`UPDATE trail_time_buckets
		SET created_to = {:created}
		WHERE id = {:id}`).
		Bind(dbx.Params{"created": createdTo, "id": bucketId}).
		Execute()
	return err
}

func setTimeBucketEndIfOpen(app core.App, bucketId string, createdTo int64) (bool, error) {
	result, err := app.DB().NewQuery(`UPDATE trail_time_buckets
		SET created_to = {:created}
		WHERE id = {:id} AND created_to = 0`).
		Bind(dbx.Params{"created": createdTo, "id": bucketId}).
		Execute()
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func buildInClause(field string, values []string) (string, dbx.Params) {
	placeholders := make([]string, 0, len(values))
	params := dbx.Params{}
	for i, value := range values {
		key := fmt.Sprintf("in_%s_%d", field, i)
		placeholders = append(placeholders, "{:"+key+"}")
		params[key] = value
	}
	return fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ",")), params
}
