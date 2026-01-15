package util

import (
	"database/sql"
	"errors"
	"fmt"
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

		for _, trail := range trailsPage {
			if _, err := UpdateTrailBoundingBox(app, trail); err != nil {
				app.Logger().Warn(fmt.Sprintf("Unable to update trail bbox for %s: %v", trail.Id, err))
			}
			if err := AssignTrailShardsForced(app, trail); err != nil {
				return fmt.Errorf("failed to assign shards for trail %s: %w", trail.Id, err)
			}
		}

		if err := IndexTrails(app, trailsPage, client); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to index trails page %d: %v", page, err))
		}

		page++
	}

	app.Logger().Info(fmt.Sprintf("Successfully rebuilt trail buckets for %d pages", page))
	return nil
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
