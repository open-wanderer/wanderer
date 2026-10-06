package util

import (
	"errors"
	"fmt"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

// SearchDocumentVersions identifies the shape of the documents each index is
// built from. Bump an index's version whenever its document builder changes,
// or a migration rewrites indexed fields without running hooks, so instances
// rebuild that index on their next start.
var SearchDocumentVersions = map[string]int{
	"trails": 1,
	"lists":  1,
	"actors": 1,
}

func documentFromTrailRecord(r *core.Record, author *core.Record, includeShares bool) (map[string]interface{}, error) {
	if author == nil {
		return nil, fmt.Errorf("trail %s has missing author reference %q", r.Id, r.GetString("author"))
	}

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

	categoryID := r.GetString("category")
	var categoryIDValue any
	if categoryID != "" {
		categoryIDValue = categoryID
	}

	subcategoryID := r.GetString("subcategory")
	var subcategoryIDValue any
	if subcategoryID != "" {
		subcategoryIDValue = subcategoryID
	}

	category := ""
	categoryIcon := ""
	trailCategory := r.ExpandedOne("category")
	if trailCategory != nil {
		category = trailCategory.GetString("name")
		categoryIcon = trailCategory.GetString("icon")
	}

	bounds := getStoredBounds(r)

	domain := ""
	if !author.GetBool("is_local") {
		domain = author.GetString("domain")
	}

	diagonal := r.GetFloat("bounding_box_diagonal")
	if diagonal == 0 && (bounds[0] != bounds[1] || bounds[2] != bounds[3]) {
		diagonal = HaversineDistance(bounds[0], bounds[2], bounds[1], bounds[3])
	}

	document := map[string]any{
		"id":                         r.Id,
		"author":                     author.Id,
		"author_name":                author.GetString("preferred_username"),
		"author_avatar":              author.GetString("icon"),
		"name":                       r.GetString("name"),
		"description":                r.GetString("description"),
		"location":                   r.GetString("location"),
		"distance":                   r.GetFloat("distance"),
		"elevation_gain":             r.GetFloat("elevation_gain"),
		"elevation_loss":             r.GetFloat("elevation_loss"),
		"duration":                   r.GetFloat("duration"),
		"difficulty":                 difficultyToNumber(r.GetString("difficulty")),
		"category":                   category,
		"category_id":                categoryIDValue,
		"category_icon":              categoryIcon,
		"subcategory_id":             subcategoryIDValue,
		"is_federated":               !author.GetBool("is_local"),
		"federated_category_name":    r.GetString("federated_category_name"),
		"federated_subcategory_name": r.GetString("federated_subcategory_name"),
		"completed":                  r.GetBool("completed"),
		"date":                       r.GetDateTime("date").Time().Unix(),
		"created":                    r.GetDateTime("created").Time().Unix(),
		"public":                     r.GetBool("public"),
		"thumbnail":                  thumbnail,
		"gpx":                        r.GetString("gpx"),
		"tags":                       tags,
		"polyline":                   r.GetString("polyline"),
		"domain":                     domain,
		"iri":                        r.GetString("iri"),
		"min_lat":                    bounds[0],
		"max_lat":                    bounds[1],
		"min_lon":                    bounds[2],
		"max_lon":                    bounds[3],
		"bounding_box_diagonal":      diagonal,
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

func getStoredBounds(r *core.Record) [4]float64 {
	lat := r.GetFloat("lat")
	lon := r.GetFloat("lon")
	defaultBounds := [4]float64{lat, lat, lon, lon}

	minLat := r.GetFloat("min_lat")
	maxLat := r.GetFloat("max_lat")
	minLon := r.GetFloat("min_lon")
	maxLon := r.GetFloat("max_lon")
	if minLat == 0 && maxLat == 0 && minLon == 0 && maxLon == 0 && (lat != 0 || lon != 0) {
		return defaultBounds
	}

	return [4]float64{minLat, maxLat, minLon, maxLon}
}

// documentFromListRecord builds a list's search document. Its totals are
// summed from the list's trails in this database; a remote list has local
// copies of its trails once it has been opened here and fully synced.
func documentFromListRecord(r *core.Record, author *core.Record, includeShares bool) (map[string]any, error) {
	if author == nil {
		return nil, fmt.Errorf("list %s has missing author reference %q", r.Id, r.GetString("author"))
	}

	totalElevationGain := 0.0
	totalElevationLoss := 0.0
	totalDistance := 0.0
	totalDuration := 0.0
	trails := len(r.GetStringSlice("trails"))

	for _, t := range r.ExpandedAll("trails") {
		totalElevationGain += t.GetFloat("elevation_gain")
		totalElevationLoss += t.GetFloat("elevation_loss")
		totalDistance += t.GetFloat("distance")
		totalDuration += t.GetFloat("duration")
	}

	domain := ""
	if !author.GetBool("is_local") {
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

// ActorSearchDocument builds the search document of an actor.
func ActorSearchDocument(r *core.Record) (map[string]any, error) {
	return documentFromActorRecord(r)
}

func documentFromActorRecord(r *core.Record) (map[string]any, error) {

	document := map[string]any{
		"id":                 r.Id,
		"username":           r.GetString("username"),
		"preferred_username": r.GetString("preferred_username"),
		"domain":             r.GetString("domain"),
		"iri":                r.GetString("iri"),
		"icon":               r.GetString("icon"),
		"is_local":           r.GetBool("is_local"),
	}

	return document, nil
}

// TrailSearchDocument builds the full search document of a trail, as a
// rebuild indexes it.
func TrailSearchDocument(app core.App, r *core.Record) (map[string]any, error) {
	errs := app.ExpandRecord(r, []string{"tags"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand tags: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"category"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand category: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"trail_share_via_trail"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand trail_share_via_trail: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"trail_like_via_trail"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand trail_like_via_trail: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"author"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand author: %v", errs)
	}

	return documentFromTrailRecord(r, r.ExpandedOne("author"), true)
}

func IndexTrails(app core.App, trails []*core.Record, client meilisearch.ServiceManager) error {
	documents := make([]map[string]any, len(trails))

	for i, r := range trails {
		doc, err := TrailSearchDocument(app, r)
		if err != nil {
			return err
		}

		documents[i] = doc
	}

	if _, err := client.Index("trails").AddDocuments(documents, nil); err != nil {
		return err
	}

	return nil
}

func UpdateTrail(app core.App, r *core.Record, author *core.Record, client meilisearch.ServiceManager) error {
	errs := app.ExpandRecord(r, []string{"tags"}, nil)
	if len(errs) > 0 {
		return fmt.Errorf("meilisearch update trail: failed to expand tags: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"category"}, nil)
	if len(errs) > 0 {
		return fmt.Errorf("meilisearch update trail: failed to expand category: %v", errs)
	}

	doc, err := documentFromTrailRecord(r, author, false)
	if err != nil {
		return err
	}
	documents := []map[string]interface{}{doc}

	if _, err = client.Index("trails").UpdateDocuments(documents, nil); err != nil {
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
	if _, err := client.Index("trails").UpdateDocuments(documents, nil); err != nil {
		return err
	}
	return nil
}

// ListSearchDocument builds the full search document of a list, as a
// rebuild indexes it.
func ListSearchDocument(app core.App, r *core.Record) (map[string]any, error) {
	errs := app.ExpandRecord(r, []string{"trails"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand trails: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"list_share_via_list"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand list_share_via_list: %v", errs)
	}
	errs = app.ExpandRecord(r, []string{"author"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand author: %v", errs)
	}

	return documentFromListRecord(r, r.ExpandedOne("author"), true)
}

func IndexLists(app core.App, lists []*core.Record, client meilisearch.ServiceManager) error {
	documents := make([]map[string]any, len(lists))

	for i, r := range lists {
		doc, err := ListSearchDocument(app, r)
		if err != nil {
			return err
		}
		documents[i] = doc
	}
	if _, err := client.Index("lists").AddDocuments(documents, nil); err != nil {
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

	if _, err = client.Index("lists").UpdateDocuments(documents, nil); err != nil {
		return err
	}

	return nil
}

func IndexActors(actors []*core.Record, client meilisearch.ServiceManager) error {
	documents := make([]map[string]any, len(actors))

	for i, r := range actors {

		doc, err := documentFromActorRecord(r)
		if err != nil {
			return err
		}
		documents[i] = doc
	}
	if _, err := client.Index("actors").AddDocuments(documents, nil); err != nil {
		return err
	}

	return nil
}

func UpdateActor(r *core.Record, client meilisearch.ServiceManager) error {
	documents, err := documentFromActorRecord(r)
	if err != nil {
		return err
	}

	if _, err = client.Index("actors").UpdateDocuments(documents, nil); err != nil {
		return err
	}

	return nil
}

func GenerateMeilisearchToken(rules map[string]interface{}, client meilisearch.ServiceManager) (string, error) {
	var apiKeyUid string
	var apiKey string

	keys, err := client.GetKeys(&meilisearch.KeysQuery{Limit: 20})
	if err != nil {
		return "", fmt.Errorf("meilisearch connection error: %w", err)
	}

	for _, k := range keys.Results {
		for _, action := range k.Actions {
			if action == "search" || k.Name == "Default Search API Key" {
				apiKeyUid = k.UID
				apiKey = k.Key
				break
			}
		}
		if apiKey != "" {
			break
		}
	}

	if apiKey == "" || apiKeyUid == "" {
		return "", errors.New("unable to locate a valid search API key")
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	options := &meilisearch.TenantTokenOptions{
		APIKey:    apiKey,
		ExpiresAt: expiresAt,
	}

	return client.GenerateTenantToken(apiKeyUid, rules, options)
}
