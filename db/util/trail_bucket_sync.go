package util

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func SyncTrailBuckets(app core.App, client meilisearch.ServiceManager, force bool) error {
	if !BucketsEnabled() {
		return clearTrailBucketData(app)
	}

	return rebuildTrailBuckets(app, client, force)
}

func ensureBucketCollections(app core.App) error {
	collections := []string{
		"quad_nodes",
		"trail_quad_nodes",
		"trail_time_buckets",
		"trail_time_bucket_entries",
	}
	for _, name := range collections {
		if _, err := app.FindCollectionByNameOrId(name); err != nil {
			return fmt.Errorf("missing bucket collection %s: %w", name, err)
		}
	}
	return nil
}

func rebuildTrailBuckets(app core.App, client meilisearch.ServiceManager, force bool) error {
	if err := ensureBucketCollections(app); err != nil {
		return err
	}

	empty, err := bucketTablesEmpty(app)
	if err != nil {
		return err
	}
	if !force && !empty {
		return nil
	}

	if err := clearTrailBucketData(app); err != nil {
		return err
	}

	if err := bulkRebuildShards(app); err != nil {
		return err
	}

	// Pre-load all shard mappings in 2 queries
	quadNodeMap, timeBucketMap, err := BulkLoadShardMappings(app)
	if err != nil {
		return fmt.Errorf("failed to load shard mappings: %w", err)
	}

	if _, err := client.GetIndex("trails"); err != nil {
		app.Logger().Warn("Meilisearch trails index missing; rebuilding full trails index")
		if _, err := client.CreateIndex(&meilisearch.IndexConfig{
			Uid:        "trails",
			PrimaryKey: "id",
		}); err != nil {
			return fmt.Errorf("failed to create trails index: %w", err)
		}
		if err := rebuildTrailIndexWithShards(app, client, quadNodeMap, timeBucketMap); err != nil {
			return err
		}

		return nil
	}

	// Update only shard IDs in Meilisearch (partial update - much faster than full reindex)
	if err := BulkUpdateTrailShardIds(client, quadNodeMap, timeBucketMap); err != nil {
		return fmt.Errorf("failed to update trail shard IDs in Meilisearch: %w", err)
	}

	app.Logger().Info(fmt.Sprintf("Successfully updated shard IDs for %d trails", len(quadNodeMap)))
	return nil
}

func rebuildTrailIndexWithShards(app core.App, client meilisearch.ServiceManager, quadNodeMap map[string][]string, timeBucketMap map[string][]string) error {
	const pageSize int64 = 100
	var page int64

	for {
		trailsPage := []*core.Record{}
		if err := app.RecordQuery("trails").
			Limit(pageSize).
			Offset(page * pageSize).
			All(&trailsPage); err != nil {
			return fmt.Errorf("failed to query trails: %w", err)
		}
		if len(trailsPage) == 0 {
			break
		}

		if err := IndexTrailsWithPreloadedShards(app, trailsPage, client, quadNodeMap, timeBucketMap); err != nil {
			return fmt.Errorf("failed to index trails page %d: %w", page, err)
		}

		page++
	}

	app.Logger().Info(fmt.Sprintf("Successfully rebuilt trails index for %d pages", page))
	return nil
}

// bulkTrailData holds trail info needed for bulk sharding
type bulkTrailData struct {
	Id        string
	MinLat    float64
	MinLon    float64
	MaxLat    float64
	MaxLon    float64
	CreatedAt int64
}

// bulkRebuildShards builds all sharding structures in bulk - much faster than trail-by-trail
func bulkRebuildShards(app core.App) error {

	// Load all trails with their bounding boxes in one query
	var trails []bulkTrailData
	if err := app.DB().NewQuery(`
		SELECT id, min_lat, min_lon, max_lat, max_lon,
			CAST(strftime('%s', created) AS INTEGER) AS created_at
		FROM trails
		WHERE min_lat IS NOT NULL AND max_lat IS NOT NULL
		ORDER BY created
	`).All(&trails); err != nil {
		return fmt.Errorf("failed to load trails: %w", err)
	}

	if len(trails) == 0 {
		app.Logger().Info("No trails to shard")
		return nil
	}

	app.Logger().Info(fmt.Sprintf("Building shards for %d trails...", len(trails)))

	// Build time buckets first (simpler structure)
	if err := bulkBuildTimeBuckets(app, trails); err != nil {
		return fmt.Errorf("failed to build time buckets: %w", err)
	}

	// Build quad tree
	if err := bulkBuildQuadTree(app, trails); err != nil {
		return fmt.Errorf("failed to build quad tree: %w", err)
	}

	return nil
}

// bulkBuildTimeBuckets creates all time buckets and entries in bulk
func bulkBuildTimeBuckets(app core.App, trails []bulkTrailData) error {
	// Trails are already sorted by created time
	// Split into chunks of TrailBucketMax
	var allEntries []struct {
		TrailId  string
		BucketId string
	}

	for i := 0; i < len(trails); i += TrailBucketMax {
		end := i + TrailBucketMax
		if end > len(trails) {
			end = len(trails)
		}

		chunk := trails[i:end]
		createdFrom := chunk[0].CreatedAt
		var createdTo int64
		if end < len(trails) {
			createdTo = trails[end].CreatedAt
		}

		bucketId, err := createTimeBucket(app, createdFrom, createdTo, len(chunk))
		if err != nil {
			return err
		}

		for _, trail := range chunk {
			allEntries = append(allEntries, struct {
				TrailId  string
				BucketId string
			}{TrailId: trail.Id, BucketId: bucketId})
		}
	}

	// Bulk insert all entries
	if _, err := bulkInsertTrailTimeBucketEntries(app, allEntries); err != nil {
		return err
	}

	app.Logger().Info(fmt.Sprintf("Created %d time buckets", (len(trails)+TrailBucketMax-1)/TrailBucketMax))
	return nil
}

// quadBuildNode represents a node being built in the quad tree
type quadBuildNode struct {
	id       string
	minLat   float64
	minLon   float64
	maxLat   float64
	maxLon   float64
	depth    int
	path     string
	parentId string
	trails   []bulkTrailData
}

// bulkBuildQuadTree builds the entire quad tree structure in bulk
func bulkBuildQuadTree(app core.App, trails []bulkTrailData) error {
	// Create root node
	rootId, err := createQuadNode(app, "", 0, -90.0, -180.0, 90.0, 180.0, 0, true, "spatial", nil, nil, "root")
	if err != nil {
		return err
	}

	// Start with all trails in root
	root := &quadBuildNode{
		id:     rootId,
		minLat: -90.0,
		minLon: -180.0,
		maxLat: 90.0,
		maxLon: 180.0,
		depth:  0,
		path:   "root",
		trails: trails,
	}

	// Process nodes using a queue (breadth-first)
	queue := []*quadBuildNode{root}
	var allEntries []struct {
		TrailId string
		NodeId  string
	}
	var leafNodes []*quadBuildNode

	nodesCreated := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		if len(node.trails) <= TrailBucketMax {
			// This is a leaf node - collect entries
			leafNodes = append(leafNodes, node)
			for _, trail := range node.trails {
				allEntries = append(allEntries, struct {
					TrailId string
					NodeId  string
				}{TrailId: trail.Id, NodeId: node.id})
			}
			continue
		}

		// Need to split
		if node.depth >= QuadTreeMaxDepth {
			// Logical split by time
			children, err := bulkLogicalSplit(app, node)
			if err != nil {
				return err
			}
			nodesCreated += len(children)
			for _, child := range children {
				for _, trail := range child.trails {
					allEntries = append(allEntries, struct {
						TrailId string
						NodeId  string
					}{TrailId: trail.Id, NodeId: child.id})
				}
			}
		} else {
			// Spatial split
			if err := updateNodeLeafState(app, node.id, false, "spatial", 0); err != nil {
				return err
			}

			children, err := bulkSpatialSplit(app, node)
			if err != nil {
				return err
			}
			nodesCreated += len(children)
			queue = append(queue, children...)
		}
	}

	// Bulk insert all trail-node mappings
	if _, err := bulkInsertTrailQuadNodes(app, allEntries); err != nil {
		return err
	}

	// Update trail counts for leaf nodes
	for _, leaf := range leafNodes {
		if _, err := app.DB().NewQuery(`UPDATE quad_nodes SET trail_count = {:count} WHERE id = {:id}`).
			Bind(map[string]any{"count": len(leaf.trails), "id": leaf.id}).
			Execute(); err != nil {
			return err
		}
	}

	app.Logger().Info(fmt.Sprintf("Created quad tree with %d leaf nodes", len(leafNodes)))
	return nil
}

// bulkSpatialSplit splits a node into 4 quadrants
func bulkSpatialSplit(app core.App, node *quadBuildNode) ([]*quadBuildNode, error) {
	latMid := (node.minLat + node.maxLat) / 2.0
	lonMid := (node.minLon + node.maxLon) / 2.0
	if node.depth == 0 {
		latMid = RootSplitLat
		lonMid = RootSplitLon
	}

	quadrants := []struct {
		suffix string
		minLat float64
		minLon float64
		maxLat float64
		maxLon float64
	}{
		{"sw", node.minLat, node.minLon, latMid, lonMid},
		{"se", node.minLat, lonMid, latMid, node.maxLon},
		{"nw", latMid, node.minLon, node.maxLat, lonMid},
		{"ne", latMid, lonMid, node.maxLat, node.maxLon},
	}

	var children []*quadBuildNode
	for _, q := range quadrants {
		childId, err := createQuadNode(app, node.id, node.depth+1, q.minLat, q.minLon, q.maxLat, q.maxLon, 0, true, "spatial", nil, nil, node.path+"/"+q.suffix)
		if err != nil {
			return nil, err
		}

		child := &quadBuildNode{
			id:       childId,
			minLat:   q.minLat,
			minLon:   q.minLon,
			maxLat:   q.maxLat,
			maxLon:   q.maxLon,
			depth:    node.depth + 1,
			path:     node.path + "/" + q.suffix,
			parentId: node.id,
			trails:   make([]bulkTrailData, 0),
		}

		// Assign trails that intersect this quadrant
		for _, trail := range node.trails {
			bbox := normalizeTrailBoundingBox(TrailBoundingBox{
				MinLat: trail.MinLat,
				MinLon: trail.MinLon,
				MaxLat: trail.MaxLat,
				MaxLon: trail.MaxLon,
			})
			if bboxIntersects(q.minLat, q.minLon, q.maxLat, q.maxLon, bbox) {
				child.trails = append(child.trails, trail)
			}
		}

		if len(child.trails) > 0 {
			children = append(children, child)
		}
	}

	return children, nil
}

// bulkLogicalSplit splits a node by time when max depth is reached
func bulkLogicalSplit(app core.App, node *quadBuildNode) ([]*quadBuildNode, error) {
	if err := updateNodeLeafState(app, node.id, false, "logical", 0); err != nil {
		return nil, err
	}

	// Sort by creation time
	sort.Slice(node.trails, func(i, j int) bool {
		return node.trails[i].CreatedAt < node.trails[j].CreatedAt
	})

	var children []*quadBuildNode
	for i := 0; i < len(node.trails); i += TrailBucketMax {
		end := i + TrailBucketMax
		if end > len(node.trails) {
			end = len(node.trails)
		}

		chunk := node.trails[i:end]
		createdFrom := chunk[0].CreatedAt
		createdTo := chunk[len(chunk)-1].CreatedAt
		suffix := fmt.Sprintf("t%04d", i/TrailBucketMax)

		childId, err := createQuadNode(app, node.id, node.depth+1, node.minLat, node.minLon, node.maxLat, node.maxLon, len(chunk), true, "logical", &createdFrom, &createdTo, node.path+"/"+suffix)
		if err != nil {
			return nil, err
		}

		children = append(children, &quadBuildNode{
			id:     childId,
			trails: chunk,
		})
	}

	return children, nil
}

// bboxIntersects checks if a quad node intersects a trail bbox
func bboxIntersects(nodeMinLat, nodeMinLon, nodeMaxLat, nodeMaxLon float64, bbox TrailBoundingBox) bool {
	return nodeMaxLat >= bbox.MinLat &&
		nodeMinLat <= bbox.MaxLat &&
		nodeMaxLon >= bbox.MinLon &&
		nodeMinLon <= bbox.MaxLon
}
func bucketTablesEmpty(app core.App) (bool, error) {
	type countRow struct {
		Total int `db:"total"`
	}

	var quadNodes countRow
	if err := app.DB().NewQuery("SELECT COUNT(1) AS total FROM quad_nodes").
		One(&quadNodes); err != nil {
		return false, err
	}
	if quadNodes.Total > 0 {
		return false, nil
	}

	var buckets countRow
	if err := app.DB().NewQuery("SELECT COUNT(1) AS total FROM trail_time_buckets").
		One(&buckets); err != nil {
		return false, err
	}

	return buckets.Total == 0, nil
}

func clearTrailBucketData(app core.App) error {
	tables := []string{
		"trail_quad_nodes",
		"quad_nodes",
		"trail_time_bucket_entries",
		"trail_time_buckets",
	}

	for _, table := range tables {
		if err := ensureBucketCollectionExists(app, table); err != nil {
			if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
				continue
			}
			return err
		}
		if err := deleteBucketTable(app, table); err != nil {
			return err
		}
	}

	return nil
}

func deleteBucketTable(app core.App, table string) error {
	var query string
	switch table {
	case "trail_quad_nodes":
		query = "DELETE FROM trail_quad_nodes"
	case "quad_nodes":
		query = "DELETE FROM quad_nodes"
	case "trail_time_bucket_entries":
		query = "DELETE FROM trail_time_bucket_entries"
	case "trail_time_buckets":
		query = "DELETE FROM trail_time_buckets"
	default:
		return fmt.Errorf("unsupported bucket table: %s", table)
	}

	_, err := app.DB().NewQuery(query).Execute()
	return err
}

func ensureBucketCollectionExists(app core.App, name string) error {
	_, err := app.FindCollectionByNameOrId(name)
	return err
}
