package util

import (
	"fmt"

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

	return app.FindRecordById("trails", trailID)
}

func EnsureTrailExternalReference(app core.App, trailID string, provider string, externalID string) error {
	if trailID == "" || provider == "" || externalID == "" {
		return nil
	}
	userID, err := externalReferenceUserID(app, trailID)
	if err != nil {
		return err
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
		"trail":       trailID,
		"user":        userID,
		"provider":    provider,
		"external_id": externalID,
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
