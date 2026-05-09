package trailmerge

import (
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func TryAutoMergeImportedTrail(
	app core.App,
	client meilisearch.ServiceManager,
	actor *core.Record,
	sourceTrailID string,
	settings IntegrationAutoMergeSettings,
) error {
	if actor == nil || sourceTrailID == "" || !settings.Enabled {
		return nil
	}

	response, err := Suggest(app, actor.Id, SuggestRequest{
		Mode:          SuggestModeAutoDiscovery,
		SourceTrailID: sourceTrailID,
	})
	if err != nil {
		return err
	}

	selectableCandidates := make([]SuggestCandidate, 0, len(response.Candidates))
	for _, candidate := range response.Candidates {
		if candidate.Selectable {
			selectableCandidates = append(selectableCandidates, candidate)
		}
	}

	if len(selectableCandidates) != 1 {
		return nil
	}

	targetTrailID := selectableCandidates[0].TrailID
	if targetTrailID == "" || targetTrailID == sourceTrailID {
		return nil
	}

	return Merge(app, client, actor, sourceTrailID, targetTrailID, DefaultIntegrationAutoMergeMergeSettings())
}
