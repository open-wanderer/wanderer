package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type geometryLoad struct {
	Trails       map[string]Trail
	Rows         int
	EncodedBytes int64
}

// loadEncodedTrailRows crosses the database/sql boundary and decodes stored
// geometry BLOBs for exact refinement. Only fixed internal table names are
// accepted because identifiers cannot be SQL parameters.
func loadEncodedTrailRows(ctx context.Context, db *sql.DB, table string, ids map[string]struct{}) (geometryLoad, error) {
	result := geometryLoad{Trails: make(map[string]Trail, len(ids))}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if table != "trails" {
		return result, fmt.Errorf("unsupported encoded trail table %q", table)
	}
	ordered := sortedIDs(ids)
	const batchSize = 900
	for start := 0; start < len(ordered); start += batchSize {
		end := min(start+batchSize, len(ordered))
		batch := ordered[start:end]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		arguments := make([]any, len(batch))
		for index, id := range batch {
			arguments[index] = id
		}
		rows, err := db.QueryContext(ctx,
			`SELECT id, scenario, geometry FROM `+table+` WHERE id IN (`+placeholders+`) ORDER BY id`,
			arguments...,
		)
		if err != nil {
			return result, fmt.Errorf("load candidate geometries: %w", err)
		}
		for rows.Next() {
			var trail Trail
			var geometry []byte
			if err := rows.Scan(&trail.ID, &trail.Scenario, &geometry); err != nil {
				_ = rows.Close()
				return result, fmt.Errorf("scan candidate geometry: %w", err)
			}
			result.EncodedBytes += int64(len(geometry))
			if err := json.Unmarshal(geometry, &trail.Parts); err != nil {
				_ = rows.Close()
				return result, fmt.Errorf("decode candidate geometry for trail %q: %w", trail.ID, err)
			}
			result.Trails[trail.ID] = trail
			result.Rows++
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return result, fmt.Errorf("iterate candidate geometries: %w", err)
		}
		if err := rows.Close(); err != nil {
			return result, fmt.Errorf("close candidate geometry rows: %w", err)
		}
	}
	return result, nil
}

func sqliteDatabaseFilesSize(path string) (int64, error) {
	var total int64
	for _, suffix := range []string{"", "-wal", "-shm", "-journal"} {
		info, err := os.Stat(path + suffix)
		if err != nil {
			if os.IsNotExist(err) && suffix != "" {
				continue
			}
			return 0, fmt.Errorf("stat SQLite database file %q: %w", path+suffix, err)
		}
		total += info.Size()
	}
	return total, nil
}
