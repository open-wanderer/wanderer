package routes

import (
	"context"
	"encoding/json"
	"testing"

	"pocketbase/pluginsystem"

	"github.com/pocketbase/pocketbase/core"
)

func TestPluginDetailAllowsKomootFallbackRequestCount(t *testing.T) {
	instance := core.NewRecord(core.NewBaseCollection("plugin_instances"))
	instance.Set("plugin_id", "komoot")
	session := assetImportBatchTestSession{call: func(_ context.Context, _ string, _ []byte, options pluginsystem.RuntimeCallOptions) ([]byte, error) {
		// Login, language, tour, two share-token calls, timeline, twenty tips
		// and cover must all fit in the same detail export.
		for request := 1; request <= 27; request++ {
			if request > pluginsystem.EffectiveMaxHostRequests(options) {
				return nil, pluginsystem.HostRequestBudgetError{Limit: options.MaxHostRequests}
			}
		}
		return json.Marshal(pluginSystemDetailOutput{Item: pluginsystem.TrailImport{
			Source: pluginsystem.TrailImportSource{Provider: "komoot", ExternalID: "fallback-tour"},
		}})
	}}
	item, err := pluginDetail(context.Background(), session, pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{Export: "get_route_detail_v1"}, instance, nil, nil, pluginsystem.TrailSummary{})
	if err != nil || item.Source.ExternalID != "fallback-tour" {
		t.Fatalf("Komoot fallback was skipped: item=%#v error=%v", item, err)
	}
}
