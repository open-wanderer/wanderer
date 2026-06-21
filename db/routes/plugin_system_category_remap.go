package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/plugins/importer"
)

type pluginCategoryRemapRequest struct {
	InstanceID string         `json:"instanceId"`
	Config     map[string]any `json:"config,omitempty"`
}

type pluginCategoryRemapResponse struct {
	Count                  int `json:"count"`
	BackfilledSinceMapping int `json:"backfilledSinceMapping,omitempty"`
	Remapped               int `json:"remapped,omitempty"`
}

type pluginCategoryRemapCandidate struct {
	Trail      *core.Record
	CategoryID string
}

type pluginCategoryTrailReference struct {
	Ref        *core.Record
	Trail      *core.Record
	ExternalID string
}

// PluginSystemCategoryRemapPreview counts imported trails whose stored provider
// category can be mapped with the current plugin instance configuration.
func PluginSystemCategoryRemapPreview(e *core.RequestEvent) error {
	instance, mapping, err := pluginCategoryRemapInput(e)
	if err != nil {
		return err
	}
	refs, err := pluginCategoryTrailReferences(e.App, e.Auth.Id, instance.GetString("plugin_id"))
	if err != nil {
		return err
	}
	candidates := pluginCategoryRemapCandidatesFromRefs(e.App, refs, mapping)
	backfilledSinceMapping := pluginCategoryBackfilledSinceMappingCountFromRefs(e.App, instance, refs, mapping)
	return e.JSON(http.StatusOK, pluginCategoryRemapResponse{
		Count:                  len(candidates),
		BackfilledSinceMapping: backfilledSinceMapping,
	})
}

// PluginSystemCategoryRemapApply updates the local category of imported trails
// whose stored provider category matches the current plugin instance mapping.
func PluginSystemCategoryRemapApply(e *core.RequestEvent) error {
	instance, mapping, err := pluginCategoryRemapInput(e)
	if err != nil {
		return err
	}
	candidates, err := pluginCategoryRemapCandidates(e.App, e.Auth.Id, instance.GetString("plugin_id"), mapping)
	if err != nil {
		return err
	}
	remapped := 0
	if err := e.App.RunInTransaction(func(txApp core.App) error {
		for _, candidate := range candidates {
			trail, err := txApp.FindRecordById("trails", candidate.Trail.Id)
			if err != nil {
				return err
			}
			trail.Set("category", candidate.CategoryID)
			if err := txApp.Save(trail); err != nil {
				return err
			}
			remapped++
		}
		return nil
	}); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, pluginCategoryRemapResponse{Count: len(candidates), Remapped: remapped})
}

func pluginCategoryRemapInput(e *core.RequestEvent) (*core.Record, map[string]string, error) {
	if e.Auth == nil {
		return nil, nil, apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginCategoryRemapRequest
	if err := e.BindBody(&data); err != nil {
		return nil, nil, apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.InstanceID == "" {
		return nil, nil, apis.NewBadRequestError("instanceId is required", nil)
	}

	instance, err := e.App.FindRecordById("plugin_instances", data.InstanceID)
	if err != nil || instance.GetString("user") != e.Auth.Id {
		return nil, nil, apis.NewNotFoundError("plugin instance not found", err)
	}

	config := effectivePluginConfig(e.App, instance.GetString("plugin_id"), instance)
	if data.Config != nil {
		config = data.Config
	}
	return instance, categoryMapping(pluginHostConfig(config)), nil
}

func pluginCategoryRemapCandidates(app core.App, userID string, pluginID string, mapping map[string]string) ([]pluginCategoryRemapCandidate, error) {
	if userID == "" || pluginID == "" || len(mapping) == 0 {
		return nil, nil
	}

	refs, err := pluginCategoryTrailReferences(app, userID, pluginID)
	if err != nil || len(refs) == 0 {
		return nil, err
	}

	return pluginCategoryRemapCandidatesFromRefs(app, refs, mapping), nil
}

func pluginCategoryRemapCandidatesFromRefs(app core.App, refs []pluginCategoryTrailReference, mapping map[string]string) []pluginCategoryRemapCandidate {
	if len(refs) == 0 || len(mapping) == 0 {
		return nil
	}

	candidates := make([]pluginCategoryRemapCandidate, 0, len(refs))
	for _, ref := range refs {
		providerCategory := strings.TrimSpace(ref.Ref.GetString("provider_category"))
		categoryID, matched := importer.CategoryFromProviderMapping(app, providerCategory, mapping)
		if !matched || categoryID == "" || ref.Trail.GetString("category") == categoryID {
			continue
		}
		candidates = append(candidates, pluginCategoryRemapCandidate{
			Trail:      ref.Trail,
			CategoryID: categoryID,
		})
	}
	return candidates
}

func pluginCategoryBackfilledSinceMappingCountFromRefs(app core.App, instance *core.Record, refs []pluginCategoryTrailReference, mapping map[string]string) int {
	mappingUpdatedAt := categoryMappingUpdatedAt(app, instance)
	if mappingUpdatedAt.IsZero() || len(refs) == 0 || len(mapping) == 0 {
		return 0
	}

	count := 0
	for _, ref := range refs {
		checkedAt := ref.Ref.GetDateTime("provider_category_checked_at")
		if checkedAt.IsZero() || !checkedAt.Time().After(mappingUpdatedAt) {
			continue
		}
		providerCategory := strings.TrimSpace(ref.Ref.GetString("provider_category"))
		categoryID, matched := importer.CategoryFromProviderMapping(app, providerCategory, mapping)
		if matched && categoryID != "" && ref.Trail.GetString("category") != categoryID {
			count++
		}
	}
	return count
}

func categoryMappingUpdatedAt(app core.App, instance *core.Record) time.Time {
	if instance == nil {
		return time.Time{}
	}
	config := effectivePluginConfig(app, instance.GetString("plugin_id"), instance)
	raw, _ := pluginHostConfig(config)["categoryMappingUpdatedAt"].(string)
	if raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func pluginCategoryTrailReferences(app core.App, userID string, pluginID string) ([]pluginCategoryTrailReference, error) {
	if userID == "" || pluginID == "" {
		return nil, nil
	}

	refs, err := app.FindRecordsByFilter(
		"trail_external_reference",
		"user={:user} && plugin_id={:plugin_id}",
		"",
		-1,
		0,
		dbx.Params{"user": userID, "plugin_id": pluginID},
	)
	if err != nil || len(refs) == 0 {
		return nil, err
	}

	trailIDs := make([]string, 0, len(refs))
	seen := map[string]bool{}
	for _, ref := range refs {
		trailID := ref.GetString("trail")
		if trailID == "" || seen[trailID] {
			continue
		}
		seen[trailID] = true
		trailIDs = append(trailIDs, trailID)
	}
	if len(trailIDs) == 0 {
		return nil, nil
	}

	trails, err := app.FindRecordsByIds("trails", trailIDs)
	if err != nil {
		return nil, err
	}

	trailsByID := make(map[string]*core.Record, len(trails))
	for _, trail := range trails {
		trailsByID[trail.Id] = trail
	}

	result := make([]pluginCategoryTrailReference, 0, len(trails))
	seen = map[string]bool{}
	for _, ref := range refs {
		trailID := ref.GetString("trail")
		if trailID == "" || seen[trailID] {
			continue
		}
		trail := trailsByID[trailID]
		if trail == nil {
			continue
		}
		seen[trailID] = true
		result = append(result, pluginCategoryTrailReference{
			Ref:        ref,
			Trail:      trail,
			ExternalID: ref.GetString("external_id"),
		})
	}
	return result, nil
}
