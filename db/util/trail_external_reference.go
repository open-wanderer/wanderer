package util

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func FindTrailByExternalReferenceForUser(app core.App, userID string, provider string, externalID string) (*core.Record, error) {
	if userID == "" || provider == "" || externalID == "" {
		return nil, nil
	}

	refs, err := app.FindRecordsByFilter(
		"trail_external_reference",
		"user={:user} && provider={:provider} && external_id={:external_id}",
		"+created",
		1,
		0,
		dbx.Params{
			"user":        userID,
			"provider":    provider,
			"external_id": externalID,
		},
	)
	if err != nil || len(refs) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	trailID := refs[0].GetString("trail")
	if trailID == "" {
		return nil, nil
	}

	trail, err := app.FindRecordById("trails", trailID)
	if err == nil {
		return trail, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if deleteErr := app.Delete(refs[0]); deleteErr != nil {
		return nil, fmt.Errorf("delete orphaned trail external reference: %w", deleteErr)
	}
	app.Logger().Warn("deleted orphaned trail external reference", "provider", provider, "external_id", externalID, "trail", trailID)
	return nil, nil
}

func FindExistingExternalReferenceIDsForUser(app core.App, userID string, provider string, externalIDs []string) (map[string]bool, error) {
	existingIDs := map[string]bool{}
	if userID == "" || provider == "" || len(externalIDs) == 0 {
		return existingIDs, nil
	}

	params := dbx.Params{
		"user":     userID,
		"provider": provider,
	}
	seen := map[string]bool{}
	idFilters := make([]string, 0, len(externalIDs))
	for _, externalID := range externalIDs {
		if externalID == "" || seen[externalID] {
			continue
		}
		seen[externalID] = true
		paramName := fmt.Sprintf("external_id_%d", len(idFilters))
		params[paramName] = externalID
		idFilters = append(idFilters, "external_id={:"+paramName+"}")
	}
	if len(idFilters) == 0 {
		return existingIDs, nil
	}

	filter := "user={:user} && provider={:provider} && (" + strings.Join(idFilters, " || ") + ")"
	refs, err := app.FindRecordsByFilter("trail_external_reference", filter, "", len(idFilters), 0, params)
	if err != nil || len(refs) == 0 {
		return existingIDs, err
	}

	trailIDs := make([]string, 0, len(refs))
	for _, ref := range refs {
		if trailID := ref.GetString("trail"); trailID != "" {
			trailIDs = append(trailIDs, trailID)
		}
	}
	var trails []*core.Record
	if len(trailIDs) > 0 {
		trails, err = app.FindRecordsByIds("trails", trailIDs)
		if err != nil {
			return nil, err
		}
	}
	trailsByID := make(map[string]bool, len(trails))
	for _, trail := range trails {
		trailsByID[trail.Id] = true
	}

	for _, ref := range refs {
		trailID := ref.GetString("trail")
		if trailID != "" && trailsByID[trailID] {
			existingIDs[ref.GetString("external_id")] = true
			continue
		}
		if deleteErr := app.Delete(ref); deleteErr != nil {
			return nil, fmt.Errorf("delete orphaned trail external reference: %w", deleteErr)
		}
		app.Logger().Warn("deleted orphaned trail external reference", "provider", provider, "external_id", ref.GetString("external_id"), "trail", trailID)
	}
	return existingIDs, nil
}

func EnsureTrailExternalReference(app core.App, trailID string, provider string, externalID string, pluginID string, providerCategory string) error {
	if trailID == "" || provider == "" || externalID == "" {
		return nil
	}
	userID, err := externalReferenceUserID(app, trailID)
	if err != nil {
		return err
	}
	if userID == "" {
		app.Logger().Warn("skipping trail external reference without local user", "provider", provider, "external_id", externalID, "trail", trailID)
		return nil
	}

	refs, err := app.FindRecordsByFilter(
		"trail_external_reference",
		"user={:user} && provider={:provider} && external_id={:external_id}",
		"",
		1,
		0,
		dbx.Params{
			"user":        userID,
			"provider":    provider,
			"external_id": externalID,
		},
	)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		if refs[0].GetString("trail") == trailID {
			changed := false
			if pluginID != "" && refs[0].GetString("plugin_id") == "" {
				refs[0].Set("plugin_id", pluginID)
				changed = true
			}
			if refs[0].GetDateTime("provider_category_checked_at").IsZero() {
				refs[0].Set("provider_category", providerCategory)
				refs[0].Set("provider_category_checked_at", time.Now())
				changed = true
			}
			if changed {
				return app.Save(refs[0])
			}
			return nil
		}
		return fmt.Errorf("trail external reference already exists for another trail")
	}

	collection, err := app.FindCollectionByNameOrId("trail_external_reference")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Load(map[string]any{
		"trail":                        trailID,
		"user":                         userID,
		"provider":                     provider,
		"external_id":                  externalID,
		"plugin_id":                    pluginID,
		"provider_category":            providerCategory,
		"provider_category_checked_at": time.Now(),
	})

	return app.Save(record)
}

func externalReferenceUserID(app core.App, trailID string) (string, error) {
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return "", err
	}
	actor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return "", err
	}
	return actor.GetString("user"), nil
}

func ReassignTrailExternalReferences(app core.App, sourceTrailID string, targetTrailID string) error {
	if sourceTrailID == "" || targetTrailID == "" || sourceTrailID == targetTrailID {
		return nil
	}

	refs, err := app.FindRecordsByFilter(
		"trail_external_reference",
		"trail={:trail}",
		"",
		-1,
		0,
		dbx.Params{"trail": sourceTrailID},
	)
	if err != nil {
		return err
	}

	for _, ref := range refs {
		provider := ref.GetString("provider")
		externalID := ref.GetString("external_id")

		existing, err := app.FindRecordsByFilter(
			"trail_external_reference",
			"trail={:trail} && provider={:provider} && external_id={:external_id}",
			"",
			1,
			0,
			dbx.Params{
				"trail":       targetTrailID,
				"provider":    provider,
				"external_id": externalID,
			},
		)
		if err != nil {
			return err
		}

		if len(existing) > 0 {
			if err := app.Delete(ref); err != nil {
				return err
			}
			continue
		}

		ref.Set("trail", targetTrailID)
		if err := app.Save(ref); err != nil {
			return err
		}
	}

	return nil
}
