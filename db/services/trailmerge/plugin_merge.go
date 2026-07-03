package trailmerge

import (
	"context"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

// TryAutoMergeImportedTrail merges a freshly imported trail into a single
// matching candidate when auto-merge is enabled. It returns the effective trail
// ID that callers should continue to work with: the merge target when a merge
// happened (the source trail is then deleted), otherwise the unchanged source
// trail ID.
func TryAutoMergeImportedTrail(
	app core.App,
	client meilisearch.ServiceManager,
	ctx context.Context,
	actor *core.Record,
	sourceTrailID string,
	settings PluginAutoMergeSettings,
) (string, error) {
	if actor == nil || sourceTrailID == "" || !settings.Enabled {
		return sourceTrailID, nil
	}

	response, err := Suggest(app, actor.Id, SuggestRequest{
		Mode:          SuggestModeAutoDiscovery,
		SourceTrailID: sourceTrailID,
	})
	if err != nil {
		return sourceTrailID, err
	}

	selectableCandidates := make([]SuggestCandidate, 0, len(response.Candidates))
	for _, candidate := range response.Candidates {
		if candidate.Selectable {
			selectableCandidates = append(selectableCandidates, candidate)
		}
	}

	if len(selectableCandidates) != 1 {
		return sourceTrailID, nil
	}

	targetTrailID := selectableCandidates[0].TrailID
	if targetTrailID == "" || targetTrailID == sourceTrailID {
		return sourceTrailID, nil
	}

	if err := Merge(app, client, ctx, actor, sourceTrailID, targetTrailID, DefaultPluginAutoMergeMergeSettings()); err != nil {
		return sourceTrailID, err
	}
	return targetTrailID, nil
}
