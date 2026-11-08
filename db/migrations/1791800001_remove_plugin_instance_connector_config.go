package migrations

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		records, err := app.FindAllRecords("plugin_instances")
		if err != nil {
			return err
		}
		for _, record := range records {
			var config map[string]any
			if err := json.Unmarshal([]byte(record.GetString("config")), &config); err != nil {
				config = nil
			}
			projected := map[string]any{}
			if plugin, ok := config["plugin"].(map[string]any); ok {
				projected["plugin"] = plugin
			}
			if host, ok := config["host"].(map[string]any); ok {
				projectedHost := map[string]any{}
				for _, key := range []string{
					"planned",
					"completed",
					"privacy",
					"createSummitLogForCompleted",
					"categoryMapping",
					"categoryMappingUpdatedAt",
					"categoryRemapDismissedAt",
				} {
					if value, exists := host[key]; exists {
						projectedHost[key] = value
					}
				}
				if merge, ok := host["merge"].(map[string]any); ok {
					if enabled, exists := merge["enabled"]; exists {
						projectedHost["merge"] = map[string]any{"enabled": enabled}
					}
				}
				if len(projectedHost) > 0 {
					projected["host"] = projectedHost
				}
			}
			record.Set("config", projected)
			if err := app.UnsafeWithoutHooks().Save(record); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		// Removed connector snapshots are administrative config and must not be
		// restored into user-owned plugin instance records.
		return nil
	})
}
