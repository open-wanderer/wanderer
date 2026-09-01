package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// productStore is a small stand-in for PocketBase's SQLite source of truth.
// Documents are persisted as encoded records so the timed product workload
// includes SQL, row scanning, allocation, and JSON/geometry decoding.
type productStore struct {
	db   *sql.DB
	path string
}

type productDocumentLoad struct {
	Documents    []ProductQueryDocument
	Rows         int
	EncodedBytes int64
}

func newProductStore(path string, readers int) (*productStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("product store path is empty")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open product store: %w", err)
	}
	if readers < 1 {
		readers = 1
	}
	db.SetMaxOpenConns(readers)
	store := &productStore{db: db, path: sqliteFilePath(path)}
	if err := store.initialize(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *productStore) initialize() error {
	if err := s.db.Ping(); err != nil {
		return fmt.Errorf("connect to product store: %w", err)
	}
	for _, statement := range []string{
		`PRAGMA busy_timeout = 5000`,
		`CREATE TABLE IF NOT EXISTS product_documents (
			id TEXT PRIMARY KEY NOT NULL,
			document BLOB NOT NULL
		) WITHOUT ROWID`,
	} {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("initialize product store: %w", err)
		}
	}
	return nil
}

func (s *productStore) replaceAll(ctx context.Context, documents []ProductQueryDocument) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin product store transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM product_documents`); err != nil {
		return fmt.Errorf("clear product store: %w", err)
	}
	insert, err := tx.PrepareContext(ctx, `INSERT INTO product_documents (id, document) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare product document insert: %w", err)
	}
	defer insert.Close()
	for _, document := range documents {
		if err := ctx.Err(); err != nil {
			return err
		}
		encoded, err := json.Marshal(document)
		if err != nil {
			return fmt.Errorf("encode product document %q: %w", document.Trail.ID, err)
		}
		if _, err := insert.ExecContext(ctx, document.Trail.ID, encoded); err != nil {
			return fmt.Errorf("store product document %q: %w", document.Trail.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit product store: %w", err)
	}
	return nil
}

func (s *productStore) load(ctx context.Context, ids map[string]struct{}) (productDocumentLoad, error) {
	result := productDocumentLoad{Documents: make([]ProductQueryDocument, 0, len(ids))}
	if err := ctx.Err(); err != nil {
		return result, err
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
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, document FROM product_documents WHERE id IN (`+placeholders+`) ORDER BY id`,
			arguments...,
		)
		if err != nil {
			return result, fmt.Errorf("load product candidates: %w", err)
		}
		for rows.Next() {
			var id string
			var encoded []byte
			if err := rows.Scan(&id, &encoded); err != nil {
				_ = rows.Close()
				return result, fmt.Errorf("scan product candidate: %w", err)
			}
			var document ProductQueryDocument
			if err := json.Unmarshal(encoded, &document); err != nil {
				_ = rows.Close()
				return result, fmt.Errorf("decode product candidate %q: %w", id, err)
			}
			if document.Trail.ID != id {
				_ = rows.Close()
				return result, fmt.Errorf("product candidate key %q contains trail %q", id, document.Trail.ID)
			}
			result.Documents = append(result.Documents, document)
			result.Rows++
			result.EncodedBytes += int64(len(encoded))
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return result, fmt.Errorf("iterate product candidates: %w", err)
		}
		if err := rows.Close(); err != nil {
			return result, fmt.Errorf("close product candidate rows: %w", err)
		}
	}
	return result, nil
}

func (s *productStore) loadGeometries(ctx context.Context, ids map[string]struct{}) (geometryLoad, error) {
	loaded, err := s.load(ctx, ids)
	result := geometryLoad{
		Trails:       make(map[string]Trail, len(loaded.Documents)),
		Rows:         loaded.Rows,
		EncodedBytes: loaded.EncodedBytes,
	}
	if err != nil {
		return result, err
	}
	for _, document := range loaded.Documents {
		result.Trails[document.Trail.ID] = document.Trail
	}
	return result, nil
}

func (s *productStore) diskBytes() (int64, error) {
	if s.path == "" {
		return 0, nil
	}
	return sqliteDatabaseFilesSize(s.path)
}

func (s *productStore) close() error {
	return s.db.Close()
}

func generateProductDocuments(dataset Dataset) []ProductQueryDocument {
	documents := make([]ProductQueryDocument, 0, len(dataset.Trails))
	categories := [...]string{"hike", "bike", "run"}
	categoryNames := [...]string{"Hiking", "Cycling", "Running"}
	categoryIcons := [...]string{"hiking.svg", "cycling.svg", "running.svg"}
	subcategories := [...]string{"day", "multi_day", "alpine", "accessible"}
	locations := [...]string{"Bernese Alps", "Valais", "Graubuenden", "Jura", "Remote world"}
	for index, trail := range dataset.Trails {
		distance := productTrailLengthMeters(trail)
		scenarioWords := strings.ReplaceAll(trail.Scenario, "_", " ")
		author := fmt.Sprintf("actor-%02d", index%23)
		federated := index%7 == 0
		public := index%5 != 0
		shares := []string{}
		if !public {
			shares = []string{fmt.Sprintf("actor-%02d", (index+7)%23)}
		}
		categoryIndex := index % len(categories)
		subcategory := subcategories[index%len(subcategories)]
		createdUnix := int64(1_609_459_200 + int64(index%1825)*86_400)
		start := trail.Start()
		domain := ""
		iri := ""
		federatedCategoryName := ""
		federatedSubcategoryName := ""
		if federated {
			domain = fmt.Sprintf("remote-%d.example", index%5)
			iri = fmt.Sprintf("https://%s/trails/%s", domain, trail.ID)
			federatedCategoryName = categoryNames[categoryIndex]
			federatedSubcategoryName = strings.ReplaceAll(subcategory, "_", " ")
		}
		documents = append(documents, ProductQueryDocument{
			Trail:                    trail,
			Polyline:                 compactTrailPolyline(trail),
			Name:                     fmt.Sprintf("%s trail %06d", scenarioWords, index),
			Description:              fmt.Sprintf("Recorded route for %s terrain with lakes forest and summit passages", scenarioWords),
			Location:                 locations[index%len(locations)],
			Tags:                     []string{trail.Scenario, categories[categoryIndex], fmt.Sprintf("season-%d", index%4)},
			AuthorID:                 author,
			AuthorName:               fmt.Sprintf("wanderer %02d", index%23),
			AuthorAvatar:             fmt.Sprintf("avatar-%02d.webp", index%23),
			Public:                   public,
			SharedWithActorIDs:       shares,
			Federated:                federated,
			CategoryID:               categories[categoryIndex],
			CategoryName:             categoryNames[categoryIndex],
			CategoryIcon:             categoryIcons[categoryIndex],
			SubcategoryID:            subcategory,
			FederatedCategoryName:    federatedCategoryName,
			FederatedSubcategoryName: federatedSubcategoryName,
			DistanceMeters:           distance,
			DurationSeconds:          distance / (0.9 + float64(index%5)*0.15),
			ElevationGainMeters:      math.Mod(distance*0.073+float64(index*17), 2600),
			ElevationLossMeters:      math.Mod(distance*0.069+float64(index*11), 2400),
			Difficulty:               float64(index % 3),
			Completed:                index%4 == 0,
			DateUnix:                 createdUnix - int64(index%31)*86_400,
			CreatedUnix:              createdUnix,
			Thumbnail:                fmt.Sprintf("trail-%06d.webp", index),
			Domain:                   domain,
			GPX:                      fmt.Sprintf("trail-%06d.gpx", index),
			LikeCount:                index % 37,
			IRI:                      iri,
			BoundingBoxDiagonal:      productTrailBoundingBoxDiagonalMeters(trail),
			Geo:                      ProductGeoPoint{Lat: start.Lat, Lng: start.Lon},
		})
	}
	return documents
}

func productTrailBoundingBoxDiagonalMeters(trail Trail) float64 {
	initialized := false
	var minLat, maxLat, minLon, maxLon float64
	for _, part := range trail.Parts {
		for _, coordinate := range part {
			if !initialized {
				minLat, maxLat = coordinate.Lat, coordinate.Lat
				minLon, maxLon = coordinate.Lon, coordinate.Lon
				initialized = true
				continue
			}
			minLat = math.Min(minLat, coordinate.Lat)
			maxLat = math.Max(maxLat, coordinate.Lat)
			minLon = math.Min(minLon, coordinate.Lon)
			maxLon = math.Max(maxLon, coordinate.Lon)
		}
	}
	if !initialized {
		return 0
	}
	return angularDistance(Coordinate{Lat: minLat, Lon: minLon}, Coordinate{Lat: maxLat, Lon: maxLon}) * earthRadiusMeters
}

func productTrailLengthMeters(trail Trail) float64 {
	var distance float64
	for _, part := range trail.Parts {
		for index := 1; index < len(part); index++ {
			distance += angularDistance(part[index-1], part[index]) * earthRadiusMeters
		}
	}
	return distance
}

func productDocumentsByID(documents []ProductQueryDocument) map[string]ProductQueryDocument {
	result := make(map[string]ProductQueryDocument, len(documents))
	for _, document := range documents {
		result[document.Trail.ID] = document
	}
	return result
}

func sortedProductDocuments(values map[string]ProductQueryDocument) []ProductQueryDocument {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]ProductQueryDocument, 0, len(ids))
	for _, id := range ids {
		result = append(result, values[id])
	}
	return result
}
