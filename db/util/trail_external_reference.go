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

const (
	ExternalReferenceKindPlanned   = "planned"
	ExternalReferenceKindCompleted = "completed"

	// TrackSourceMoved marks a reference a merge moved onto a trail whose own
	// track was kept; it never describes that trail's track.
	TrackSourceMoved = "moved"
	// TrackSourceLegacy marks a reference from before merges were tracked
	// that looks like its trail's own: whether a merge moved it cannot be
	// told, so a resync asks the user to double-check.
	TrackSourceLegacy = "legacy"
)

// Windows used to classify references that predate track_source.
const (
	// An import creates the reference right after its trail; a reference
	// created farther from its trail's creation was attached later, which
	// only a merge does.
	legacyReferenceImportWindow = 5 * time.Minute
	// Migration 1772400001 created references for trails imported before
	// the reference table existed, all at once and long after those trails.
	legacyReferenceBackfillWindow = time.Hour
)

// ClassifyExternalReferencesBeforeTracking classifies references created
// before track_source existed, using only the records' own timestamps: a
// reference an import created follows its trail within seconds, while one
// created long before or after its trail reached it through a merge, which
// moves references but keeps the target's track. References the migration
// 1772400001 backfilled at backfilledAt belong to their trail although they
// are much younger; pass the zero time when that moment is unknown. Every
// other reference from before tracking is marked legacy: it is treated as
// the trail's own, but a merge after the backfill or within the import
// window cannot be ruled out. Returns how many were marked moved and legacy.
func ClassifyExternalReferencesBeforeTracking(app core.App, backfilledAt time.Time) (moved int64, legacy int64, err error) {
	params := dbx.Params{
		"moved":        TrackSourceMoved,
		"window":       int64(legacyReferenceImportWindow.Seconds()),
		"backfillFrom": int64(-1),
		"backfillTo":   int64(-1),
	}
	if !backfilledAt.IsZero() {
		params["backfillFrom"] = backfilledAt.Add(-legacyReferenceBackfillWindow).Unix()
		params["backfillTo"] = backfilledAt.Add(legacyReferenceBackfillWindow).Unix()
	}
	result, err := app.DB().NewQuery(`
		UPDATE trail_external_reference
		SET track_source = {:moved}
		WHERE COALESCE(track_source, '') = ''
		  AND CAST(strftime('%s', created) AS INTEGER) NOT BETWEEN {:backfillFrom} AND {:backfillTo}
		  AND EXISTS (
		    SELECT 1 FROM trails
		    WHERE trails.id = trail_external_reference.trail
		      AND abs(CAST(strftime('%s', trail_external_reference.created) AS INTEGER) - CAST(strftime('%s', trails.created) AS INTEGER)) > {:window}
		  )`).Bind(params).Execute()
	if err != nil {
		return 0, 0, err
	}
	if moved, err = result.RowsAffected(); err != nil {
		return 0, 0, err
	}
	result, err = app.DB().NewQuery(`
		UPDATE trail_external_reference
		SET track_source = {:legacy}
		WHERE COALESCE(track_source, '') = ''`).Bind(dbx.Params{"legacy": TrackSourceLegacy}).Execute()
	if err != nil {
		return moved, 0, err
	}
	if legacy, err = result.RowsAffected(); err != nil {
		return moved, 0, err
	}
	return moved, legacy, nil
}

func EnsureTrailExternalReference(app core.App, trailID string, provider string, externalID string, pluginID string, providerCategory string, kind string) error {
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
			if kind != "" && refs[0].GetString("kind") == "" {
				refs[0].Set("kind", kind)
				changed = true
			}
			if refs[0].GetDateTime("provider_category_checked_at").IsZero() {
				refs[0].Set("provider_category", providerCategory)
				refs[0].Set("provider_category_checked_at", time.Now())
				changed = true
			}
			if changed {
				refs[0].IgnoreUnchangedFields(true)
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
		"kind":                         kind,
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

// ReassignTrailExternalReferences moves the import references of a merged
// source trail onto the target so the provider items are not imported again.
// The target keeps its own track, so moved references are marked as such and
// excluded from track resyncs.
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
		ref.Set("track_source", TrackSourceMoved)
		ref.IgnoreUnchangedFields(true)
		if err := app.Save(ref); err != nil {
			return err
		}
	}

	return nil
}
