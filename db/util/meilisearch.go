package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/twpayne/go-polyline"
)

func documentFromTrailRecord(app core.App, r *core.Record, author *core.Record, includeShares bool) (map[string]interface{}, error) {
	quadNodeIds, err := TrailQuadNodeIDs(app, r.Id)
	if err != nil {
		return nil, err
	}
	timeBucketIds, err := TrailTimeBucketIDs(app, r.Id)
	if err != nil {
		return nil, err
	}
	return documentFromTrailRecordWithShards(app, r, author, includeShares, quadNodeIds, timeBucketIds)
}

func documentFromTrailRecordWithShards(app core.App, r *core.Record, author *core.Record, includeShares bool, quadNodeIds, timeBucketIds []string) (map[string]interface{}, error) {
	photos := r.GetStringSlice("photos")
	thumbnail := ""
	if len(photos) > 0 {
		thumbnailIndex := r.GetInt("thumbnail")
		if thumbnailIndex >= len(photos) {
			thumbnailIndex = 0
		}
		thumbnail = photos[thumbnailIndex]
	}

	tagRecords := r.ExpandedAll("tags")
	tags := make([]string, len(tagRecords))

	for i, v := range tagRecords {
		tags[i] = v.GetString("name")
	}

	category := ""
	trailCategory := r.ExpandedOne("category")
	if trailCategory != nil {
		category = trailCategory.GetString("name")
	}

	polyline, err := getPolyline(app, r)
	if err != nil {
		app.Logger().Warn(fmt.Sprintf("failed to parse gpx for trail %s: %v", r.Id, err))
		polyline = ""
	}

	domain := ""
	if !author.GetBool("isLocal") {
		domain = author.GetString("domain")
	}

	logCount, err := app.CountRecords("summit_logs", dbx.NewExp("trail={:id}", dbx.Params{"id": r.Id}))
	if err != nil {
		return nil, err
	}

	document := map[string]any{
		"id":              r.Id,
		"author":          author.Id,
		"author_name":     author.GetString("preferred_username"),
		"author_avatar":   author.GetString("icon"),
		"name":            r.GetString("name"),
		"description":     r.GetString("description"),
		"location":        r.GetString("location"),
		"distance":        r.GetFloat("distance"),
		"elevation_gain":  r.GetFloat("elevation_gain"),
		"elevation_loss":  r.GetFloat("elevation_loss"),
		"duration":        r.GetFloat("duration"),
		"difficulty":      difficultyToNumber(r.GetString("difficulty")),
		"category":        category,
		"completed":       logCount > 0,
		"date":            r.GetDateTime("date").Time().Unix(),
		"created":         r.GetDateTime("created").Time().Unix(),
		"public":          r.GetBool("public"),
		"thumbnail":       thumbnail,
		"gpx":             r.GetString("gpx"),
		"tags":            tags,
		"polyline":        polyline,
		"domain":          domain,
		"iri":             r.GetString("iri"),
		"min_lat":         r.GetFloat("min_lat"),
		"min_lon":         r.GetFloat("min_lon"),
		"max_lat":         r.GetFloat("max_lat"),
		"max_lon":         r.GetFloat("max_lon"),
		"time_bucket_ids": timeBucketIds,
		"quad_node_ids":   quadNodeIds,
		"_geo": map[string]float64{
			"lat": r.GetFloat("lat"),
			"lng": r.GetFloat("lon"),
		},
	}

	if includeShares {
		trailShares := r.ExpandedAll("trail_share_via_trail")
		if trailShares != nil {
			sharedIDs := make([]string, len(trailShares))
			for i, v := range trailShares {
				sharedIDs[i] = v.GetString("actor")
			}

			document["shares"] = sharedIDs

		} else {
			document["shares"] = []string{}
		}

		trailLikes := r.ExpandedAll("trail_like_via_trail")
		if trailLikes != nil {
			likeIDs := make([]string, len(trailLikes))
			for i, v := range trailLikes {
				likeIDs[i] = v.GetString("actor")
			}

			document["likes"] = likeIDs
			document["like_count"] = len(trailLikes)

		} else {
			document["likes"] = []string{}
			document["like_count"] = 0
		}

	}

	return document, nil
}

func difficultyToNumber(difficulty string) int32 {
	switch difficulty {
	case "easy":
		return 0
	case "moderate":
		return 1
	case "difficult":
		return 2
	}

	return 0
}

func trailBoundsFromRecord(r *core.Record) (float64, float64, float64, float64, bool) {
	if r.Get("min_lat") != nil &&
		r.Get("min_lon") != nil &&
		r.Get("max_lat") != nil &&
		r.Get("max_lon") != nil {
		return r.GetFloat("min_lat"),
			r.GetFloat("min_lon"),
			r.GetFloat("max_lat"),
			r.GetFloat("max_lon"),
			true
	}

	if r.Get("lat") != nil && r.Get("lon") != nil {
		lat := r.GetFloat("lat")
		lon := r.GetFloat("lon")
		return lat, lon, lat, lon, true
	}

	return 0, 0, 0, 0, false
}

func getPolyline(app core.App, r *core.Record) (string, error) {
	gpxPath := r.GetString("gpx")
	if len(gpxPath) == 0 {
		return "", nil
	}
	avatarKey := r.BaseFilesPath() + "/" + gpxPath
	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", err
	}
	defer fsys.Close()

	gpxFile, err := fsys.GetReader(avatarKey)
	if err != nil {
		return "", err
	}
	defer gpxFile.Close()

	content := new(bytes.Buffer)
	_, err = io.Copy(content, gpxFile)
	if err != nil {
		return "", err
	}
	gpxData, err := gpx.Parse(content)
	if err != nil {
		return "", err
	}

	gpxData.SimplifyTracks(50)
	coordinates := make([][]float64, 0)
	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				coordinates = append(coordinates, []float64{pt.Latitude, pt.Longitude})
			}
		}
	}

	if len(coordinates) == 0 {
		return "", nil
	}

	return string(polyline.EncodeCoords(coordinates)), nil
}

func documentFromListRecord(r *core.Record, author *core.Record, includeShares bool) (map[string]interface{}, error) {

	totalElevationGain := 0.0
	totalElevationLoss := 0.0
	totalDistance := 0.0
	totalDuration := 0.0
	trails := len(r.GetStringSlice("trails"))

	if r.GetString("iri") != "" {
		doc, err := documentFromRemoteRecord(r, "lists")
		if err == nil {
			totalElevationGain = doc["elevation_gain"].(float64)
			totalElevationLoss = doc["elevation_loss"].(float64)
			totalDistance = doc["distance"].(float64)
			totalDuration = doc["duration"].(float64)

			trails = int(doc["trails"].(float64))
		}

	} else {
		allTrails := r.ExpandedAll("trails")

		for _, t := range allTrails {
			totalElevationGain += t.GetFloat("elevation_gain")
			totalElevationLoss += t.GetFloat("elevation_loss")
			totalDistance += t.GetFloat("distance")
			totalDuration += t.GetFloat("duration")

		}
	}

	domain := ""
	if !author.GetBool("isLocal") {
		domain = author.GetString("domain")
	}

	document := map[string]any{
		"id":             r.Id,
		"author":         author.Id,
		"author_name":    author.GetString("preferred_username"),
		"author_avatar":  author.GetString("icon"),
		"avatar":         r.GetString("avatar"),
		"name":           r.GetString("name"),
		"description":    r.GetString("description"),
		"elevation_gain": totalElevationGain,
		"elevation_loss": totalElevationLoss,
		"distance":       totalDistance,
		"duration":       totalDuration,
		"domain":         domain,
		"public":         r.GetBool("public"),
		"created":        r.GetDateTime("created").Time().Unix(),
		"trails":         trails,
		"iri":            r.GetString("iri"),
	}

	if includeShares {
		listShares := r.ExpandedAll("list_share_via_list")
		if listShares != nil {
			sharedIDs := make([]string, len(listShares))
			for i, v := range listShares {
				sharedIDs[i] = v.GetString("actor")
			}

			document["shares"] = sharedIDs

		} else {
			document["shares"] = []string{}
		}
	}

	return document, nil
}

func documentFromRemoteRecord(r *core.Record, index string) (map[string]interface{}, error) {
	client := &http.Client{}

	if r.GetString("iri") == "" {
		return nil, fmt.Errorf("record has no iri")
	}

	iri := r.GetString("iri")

	url, err := url.Parse(iri)
	if err != nil {
		return nil, err
	}

	remoteRecordId := path.Base(url.Path)

	searchURL := fmt.Sprintf("%s://%s/api/v1/search/%s", url.Scheme, url.Host, index)
	body := []byte(fmt.Sprintf(`{"q": "%s"}`, remoteRecordId))

	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch remote record: received status %d", resp.StatusCode)
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var searchResponse meilisearch.SearchResponse
	json.Unmarshal(respBytes, &searchResponse)

	if len(searchResponse.Hits) == 0 {
		return nil, fmt.Errorf("no documents in result set")
	}

	document, ok := searchResponse.Hits[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected document format")
	}
	return document, nil
}

func IndexTrails(app core.App, trails []*core.Record, client meilisearch.ServiceManager) error {
	documents := make([]map[string]any, len(trails))

	for i, r := range trails {
		errs := app.ExpandRecord(r, []string{"tags"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand tags: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"category"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand category: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"trail_share_via_trail"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand trail_share_via_trail: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"trail_like_via_trail"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand trail_like_via_trail: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"author"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand author: %v", errs)
		}

		author := r.ExpandedOne("author")

		doc, err := documentFromTrailRecord(app, r, author, true)
		if err != nil {
			return err
		}

		documents[i] = doc
	}

	if _, err := client.Index("trails").AddDocuments(documents); err != nil {
		return err
	}

	return nil
}

// IndexTrailsWithPreloadedShards indexes trails with pre-loaded shard mappings for better performance
func IndexTrailsWithPreloadedShards(app core.App, trails []*core.Record, client meilisearch.ServiceManager, quadNodeMap map[string][]string, timeBucketMap map[string][]string) error {
	documents := make([]map[string]any, len(trails))

	for i, r := range trails {
		errs := app.ExpandRecord(r, []string{"tags"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand tags: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"category"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand category: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"trail_share_via_trail"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand trail_share_via_trail: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"trail_like_via_trail"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand trail_like_via_trail: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"author"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand author: %v", errs)
		}

		author := r.ExpandedOne("author")

		// Use preloaded shard IDs instead of querying per trail
		quadNodeIds := quadNodeMap[r.Id]
		if quadNodeIds == nil {
			quadNodeIds = []string{}
		}
		timeBucketIds := timeBucketMap[r.Id]
		if timeBucketIds == nil {
			timeBucketIds = []string{}
		}

		doc, err := documentFromTrailRecordWithShards(app, r, author, true, quadNodeIds, timeBucketIds)
		if err != nil {
			return err
		}

		documents[i] = doc
	}

	if _, err := client.Index("trails").AddDocuments(documents); err != nil {
		return err
	}

	return nil
}

// BulkLoadShardMappings loads all trail-to-shard mappings in two queries
func BulkLoadShardMappings(app core.App) (quadNodeMap map[string][]string, timeBucketMap map[string][]string, err error) {
	quadNodeMap = make(map[string][]string)
	timeBucketMap = make(map[string][]string)

	if !BucketsEnabled() {
		return quadNodeMap, timeBucketMap, nil
	}

	// Load all quad node mappings
	var quadRows []struct {
		Trail    string `db:"trail"`
		QuadNode string `db:"quad_node"`
	}
	if err := app.DB().NewQuery(`SELECT trail, quad_node FROM trail_quad_nodes`).All(&quadRows); err != nil {
		return nil, nil, err
	}
	for _, row := range quadRows {
		quadNodeMap[row.Trail] = append(quadNodeMap[row.Trail], row.QuadNode)
	}

	// Load all time bucket mappings
	var bucketRows []struct {
		Trail  string `db:"trail"`
		Bucket string `db:"bucket"`
	}
	if err := app.DB().NewQuery(`SELECT trail, bucket FROM trail_time_bucket_entries`).All(&bucketRows); err != nil {
		return nil, nil, err
	}
	for _, row := range bucketRows {
		timeBucketMap[row.Trail] = append(timeBucketMap[row.Trail], row.Bucket)
	}

	return quadNodeMap, timeBucketMap, nil
}

func UpdateTrail(app core.App, r *core.Record, author *core.Record, client meilisearch.ServiceManager) error {
	errs := app.ExpandRecord(r, []string{"tags"}, nil)
	if len(errs) > 0 {
		return fmt.Errorf("failed to expand tags: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"category"}, nil)
	if len(errs) > 0 {
		return fmt.Errorf("failed to expand category: %v", errs)
	}

	doc, err := documentFromTrailRecord(app, r, author, false)
	if err != nil {
		return err
	}
	documents := []map[string]interface{}{doc}

	if _, err := client.Index("trails").UpdateDocuments(documents); err != nil {
		return err
	}

	return nil
}

func UpdateTrailShares(trailId string, shares []string, client meilisearch.ServiceManager) error {
	documents := []map[string]interface{}{
		{
			"id":     trailId,
			"shares": shares,
		},
	}
	if _, err := client.Index("trails").UpdateDocuments(documents); err != nil {
		return err
	}
	return nil
}

func UpdateTrailLikes(trailId string, likes []string, client meilisearch.ServiceManager) error {
	documents := []map[string]interface{}{
		{
			"id":         trailId,
			"like_count": len(likes),
			"likes":      likes,
		},
	}
	if _, err := client.Index("trails").UpdateDocuments(documents); err != nil {
		return err
	}
	return nil
}

// BulkUpdateTrailShardIds updates only the shard IDs for trails in Meilisearch
// This is much faster than full reindexing as it skips GPX parsing and record expansion
func BulkUpdateTrailShardIds(client meilisearch.ServiceManager, quadNodeMap map[string][]string, timeBucketMap map[string][]string) error {
	// Collect all trail IDs from both maps
	trailIds := make(map[string]bool)
	for trailId := range quadNodeMap {
		trailIds[trailId] = true
	}
	for trailId := range timeBucketMap {
		trailIds[trailId] = true
	}

	// Build partial documents with only id and shard fields
	documents := make([]map[string]any, 0, len(trailIds))
	for trailId := range trailIds {
		quadNodeIds := quadNodeMap[trailId]
		if quadNodeIds == nil {
			quadNodeIds = []string{}
		}
		timeBucketIds := timeBucketMap[trailId]
		if timeBucketIds == nil {
			timeBucketIds = []string{}
		}
		documents = append(documents, map[string]any{
			"id":              trailId,
			"quad_node_ids":   quadNodeIds,
			"time_bucket_ids": timeBucketIds,
		})
	}

	// Update in batches
	const batchSize = 500
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}
		batch := documents[i:end]
		if _, err := client.Index("trails").UpdateDocuments(batch); err != nil {
			return err
		}
	}

	return nil
}

func IndexLists(app core.App, lists []*core.Record, client meilisearch.ServiceManager) error {
	documents := make([]map[string]any, len(lists))

	for i, r := range lists {
		errs := app.ExpandRecord(r, []string{"trails"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand trails: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"list_share_via_list"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand list_share_via_list: %v", errs)
		}
		errs = app.ExpandRecord(r, []string{"author"}, nil)
		if len(errs) > 0 {
			return fmt.Errorf("failed to expand author: %v", errs)
		}

		author := r.ExpandedOne("author")

		doc, err := documentFromListRecord(r, author, true)
		if err != nil {
			return err
		}
		documents[i] = doc
	}
	if _, err := client.Index("lists").AddDocuments(documents); err != nil {
		return err
	}

	return nil
}

func UpdateList(app core.App, r *core.Record, author *core.Record, client meilisearch.ServiceManager) error {
	errs := app.ExpandRecord(r, []string{"trails"}, nil)
	if len(errs) > 0 {
		return fmt.Errorf("failed to expand trails: %v", errs)
	}

	documents, err := documentFromListRecord(r, author, false)
	if err != nil {
		return err
	}

	if _, err = client.Index("lists").UpdateDocuments(documents); err != nil {
		return err
	}

	return nil
}

func UpdateListShares(listId string, shares []string, client meilisearch.ServiceManager) error {
	documents := []map[string]interface{}{
		{
			"id":     listId,
			"shares": shares,
		},
	}
	if _, err := client.Index("lists").UpdateDocuments(documents); err != nil {
		return err
	}
	return nil
}

func GenerateMeilisearchToken(rules map[string]interface{}, client meilisearch.ServiceManager) (resp string, err error) {
	apiKeyUid := ""
	apiKey := ""

	if keys, err := client.GetKeys(nil); err != nil {
		log.Fatal(err)
	} else {
		for _, k := range keys.Results {
			if k.Name == "Default Search API Key" {
				apiKeyUid = k.UID
				apiKey = k.Key
			}
		}
	}

	if len(apiKey) == 0 || len(apiKeyUid) == 0 {
		return "", errors.New("unable to locate meilisearch API key")
	}

	options := &meilisearch.TenantTokenOptions{
		APIKey: apiKey,
	}

	return client.GenerateTenantToken(apiKeyUid, rules, options)
}
