package util

import (
	"errors"
	"fmt"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// UpdateTagReferences refreshes copied tag names without replacing unrelated
// search fields, such as independently updated shares and likes.
func UpdateTagReferences(app core.App, tagID string, client meilisearch.ServiceManager) error {
	const batchSize = 200
	params := dbx.Params{"tag": tagID, "cursor": ""}
	index := client.Index("trails")
	for {
		trails, err := app.FindRecordsByFilter("trails", "tags.id ?= {:tag} && id > {:cursor}", "id", batchSize, 0, params)
		if err != nil {
			return err
		}
		if len(trails) == 0 {
			return nil
		}
		patches := make([]map[string]any, 0, len(trails))
		for _, trail := range trails {
			var existing map[string]any
			err := index.GetDocument(trail.Id, &meilisearch.DocumentQuery{Fields: []string{"id"}}, &existing)
			if err != nil {
				var apiError *meilisearch.Error
				if !errors.As(err, &apiError) || apiError.MeilisearchApiError.Code != "document_not_found" {
					return fmt.Errorf("read trail %s before tag update: %w", trail.Id, err)
				}
				// A partial update also creates missing documents. Use the full
				// projector so a new hit includes its content and access fields.
				if err := IndexTrails(app, []*core.Record{trail}, client); err != nil {
					return err
				}
				continue
			}
			if errs := app.ExpandRecord(trail, []string{"tags"}, nil); len(errs) != 0 {
				return fmt.Errorf("expand tags for trail %s: %v", trail.Id, errs)
			}
			tags := []string{}
			for _, tag := range trail.ExpandedAll("tags") {
				tags = append(tags, tag.GetString("name"))
			}
			patches = append(patches, map[string]any{"id": trail.Id, "tags": tags})
		}
		if len(patches) > 0 {
			if _, err := index.UpdateDocuments(patches, nil); err != nil {
				return err
			}
		}
		if len(trails) < batchSize {
			return nil
		}
		params["cursor"] = trails[len(trails)-1].Id
	}
}
