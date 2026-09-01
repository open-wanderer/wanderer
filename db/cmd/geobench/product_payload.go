package main

import (
	"encoding/json"
	"fmt"
)

// productPageAttributeNames is the common trail-list projection for both the
// exact hybrid pipeline and a direct Meilisearch product page. Keep the list in
// response order so the two paths also have a comparable serialized payload.
var productPageAttributeNames = []string{
	"id",
	"author",
	"author_name",
	"author_avatar",
	"name",
	"description",
	"location",
	"distance",
	"elevation_gain",
	"elevation_loss",
	"duration",
	"difficulty",
	"category",
	"category_id",
	"category_icon",
	"subcategory_id",
	"is_federated",
	"federated_category_name",
	"federated_subcategory_name",
	"completed",
	"date",
	"created",
	"public",
	"thumbnail",
	"domain",
	"gpx",
	"tags",
	"like_count",
	"shares",
	"iri",
	"bounding_box_diagonal",
	"_geo",
}

func productPageAttributes() []string {
	return append([]string(nil), productPageAttributeNames...)
}

type productPageHitPayload struct {
	ID                       string          `json:"id"`
	AuthorID                 string          `json:"author"`
	AuthorName               string          `json:"author_name"`
	AuthorAvatar             string          `json:"author_avatar"`
	Name                     string          `json:"name"`
	Description              string          `json:"description"`
	Location                 string          `json:"location"`
	DistanceMeters           float64         `json:"distance"`
	ElevationGain            float64         `json:"elevation_gain"`
	ElevationLoss            float64         `json:"elevation_loss"`
	DurationSeconds          float64         `json:"duration"`
	Difficulty               float64         `json:"difficulty"`
	Category                 string          `json:"category"`
	CategoryID               string          `json:"category_id"`
	CategoryIcon             string          `json:"category_icon"`
	SubcategoryID            *string         `json:"subcategory_id"`
	Federated                bool            `json:"is_federated"`
	FederatedCategoryName    string          `json:"federated_category_name"`
	FederatedSubcategoryName string          `json:"federated_subcategory_name"`
	Completed                bool            `json:"completed"`
	DateUnix                 int64           `json:"date"`
	CreatedUnix              int64           `json:"created"`
	Public                   bool            `json:"public"`
	Thumbnail                string          `json:"thumbnail"`
	Domain                   string          `json:"domain"`
	GPX                      string          `json:"gpx"`
	Tags                     []string        `json:"tags"`
	LikeCount                int             `json:"like_count"`
	SharedWithActorIDs       []string        `json:"shares"`
	IRI                      string          `json:"iri"`
	BoundingBoxDiagonal      float64         `json:"bounding_box_diagonal"`
	Geo                      ProductGeoPoint `json:"_geo"`
}

type productPagePayload struct {
	Hits        []productPageHitPayload `json:"hits"`
	TotalHits   int                     `json:"totalHits"`
	TotalPages  int                     `json:"totalPages"`
	Page        int                     `json:"page"`
	HitsPerPage int                     `json:"hitsPerPage"`
}

func materializeProductPagePayload(result ProductQueryResult, documents []ProductQueryDocument) (productPagePayload, error) {
	payload := productPagePayload{
		Hits:        make([]productPageHitPayload, 0, len(result.Hits)),
		TotalHits:   result.TotalItems,
		TotalPages:  result.TotalPages,
		Page:        result.Page,
		HitsPerPage: result.PerPage,
	}
	documentsByID := make(map[string]ProductQueryDocument, len(documents))
	for _, document := range documents {
		id := document.Trail.ID
		if id == "" {
			return productPagePayload{}, fmt.Errorf("materialize product page: document has no trail id")
		}
		if _, duplicate := documentsByID[id]; duplicate {
			return productPagePayload{}, fmt.Errorf("materialize product page: duplicate trail id %q", id)
		}
		documentsByID[id] = document
	}
	for _, hit := range result.Hits {
		document, found := documentsByID[hit.ID]
		if !found {
			return productPagePayload{}, fmt.Errorf("materialize product page: result trail %q was not loaded", hit.ID)
		}
		payload.Hits = append(payload.Hits, productPageHitPayloadFromDocument(document))
	}
	return payload, nil
}

func marshalProductPagePayload(result ProductQueryResult, documents []ProductQueryDocument) ([]byte, error) {
	payload, err := materializeProductPagePayload(result, documents)
	if err != nil {
		return nil, err
	}
	return marshalMaterializedProductPagePayload(payload)
}

// marshalMeiliProductPagePayload normalizes RawMessage hits before encoding so
// direct Meilisearch and exact hybrid pages measure the same response envelope,
// field ordering, and number representation.
func marshalMeiliProductPagePayload(
	hits []map[string]json.RawMessage,
	totalHits, totalPages, page, hitsPerPage int,
) ([]byte, error) {
	payload := productPagePayload{
		Hits:        make([]productPageHitPayload, 0, len(hits)),
		TotalHits:   totalHits,
		TotalPages:  totalPages,
		Page:        page,
		HitsPerPage: hitsPerPage,
	}
	seen := make(map[string]struct{}, len(hits))
	attributeSet := make(map[string]struct{}, len(productPageAttributeNames))
	for _, attribute := range productPageAttributeNames {
		attributeSet[attribute] = struct{}{}
	}
	for index, rawHit := range hits {
		for _, attribute := range productPageAttributeNames {
			if _, found := rawHit[attribute]; !found {
				return nil, fmt.Errorf("materialize Meilisearch product page: hit %d has no %q field", index, attribute)
			}
		}
		for attribute := range rawHit {
			if _, expected := attributeSet[attribute]; !expected {
				return nil, fmt.Errorf("materialize Meilisearch product page: hit %d has unexpected %q field", index, attribute)
			}
		}
		encodedHit, err := json.Marshal(rawHit)
		if err != nil {
			return nil, fmt.Errorf("materialize Meilisearch product page hit %d: %w", index, err)
		}
		var hit productPageHitPayload
		if err := json.Unmarshal(encodedHit, &hit); err != nil {
			return nil, fmt.Errorf("materialize Meilisearch product page hit %d: %w", index, err)
		}
		if hit.ID == "" {
			return nil, fmt.Errorf("materialize Meilisearch product page: hit %d has an empty id", index)
		}
		if _, duplicate := seen[hit.ID]; duplicate {
			return nil, fmt.Errorf("materialize Meilisearch product page: duplicate trail id %q", hit.ID)
		}
		seen[hit.ID] = struct{}{}
		payload.Hits = append(payload.Hits, hit)
	}
	return marshalMaterializedProductPagePayload(payload)
}

func marshalMaterializedProductPagePayload(payload productPagePayload) ([]byte, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("serialize product page: %w", err)
	}
	return encoded, nil
}

func productPageHitPayloadFromDocument(document ProductQueryDocument) productPageHitPayload {
	var subcategoryID *string
	if document.SubcategoryID != "" {
		value := document.SubcategoryID
		subcategoryID = &value
	}
	return productPageHitPayload{
		ID:                       document.Trail.ID,
		AuthorID:                 document.AuthorID,
		AuthorName:               document.AuthorName,
		AuthorAvatar:             document.AuthorAvatar,
		Name:                     document.Name,
		Description:              document.Description,
		Location:                 document.Location,
		DistanceMeters:           document.DistanceMeters,
		ElevationGain:            document.ElevationGainMeters,
		ElevationLoss:            document.ElevationLossMeters,
		DurationSeconds:          document.DurationSeconds,
		Difficulty:               document.Difficulty,
		Category:                 document.CategoryName,
		CategoryID:               document.CategoryID,
		CategoryIcon:             document.CategoryIcon,
		SubcategoryID:            subcategoryID,
		Federated:                document.Federated,
		FederatedCategoryName:    document.FederatedCategoryName,
		FederatedSubcategoryName: document.FederatedSubcategoryName,
		Completed:                document.Completed,
		DateUnix:                 document.DateUnix,
		CreatedUnix:              document.CreatedUnix,
		Public:                   document.Public,
		Thumbnail:                document.Thumbnail,
		Domain:                   document.Domain,
		GPX:                      document.GPX,
		Tags:                     document.Tags,
		LikeCount:                document.LikeCount,
		SharedWithActorIDs:       document.SharedWithActorIDs,
		IRI:                      document.IRI,
		BoundingBoxDiagonal:      document.BoundingBoxDiagonal,
		Geo:                      document.Geo,
	}
}

func productDocumentPolyline(document ProductQueryDocument) string {
	if document.Polyline != "" {
		return document.Polyline
	}
	return compactTrailPolyline(document.Trail)
}
