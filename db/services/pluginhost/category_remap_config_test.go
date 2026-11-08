package pluginhost

import (
	"encoding/json"
	"testing"
)

func TestCategoryRemapDismissalSurvivesRepeatedSaves(t *testing.T) {
	for _, dismissedAt := range []string{"2026-10-12T08:00:00Z", ""} {
		defaults := administrativeConfig("https://a.example", false)
		defaults["host"].(map[string]any)["categoryRemapDismissedAt"] = "2026-10-01T08:00:00Z"
		current := map[string]any{"host": map[string]any{
			"categoryRemapDismissedAt": dismissedAt,
			"connectors":               instanceConnectorConfig(),
		}}
		stored := MergeInstanceConfigDefaults(defaults, current)
		encoded, err := json.Marshal(stored)
		if err != nil {
			t.Fatal(err)
		}
		var reloaded map[string]any
		if err := json.Unmarshal(encoded, &reloaded); err != nil {
			t.Fatal(err)
		}
		updated := MergeInstanceConfigDefaults(defaults, reloaded)
		if HostConfig(updated)["categoryRemapDismissedAt"] != dismissedAt {
			t.Fatalf("dismissal changed after repeated saves: %#v", updated)
		}
		if HostConfig(updated)["connectors"] != nil {
			t.Fatal("preserving the dismissal restored administrative connector snapshots")
		}
	}
}
