package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"pocketbase/pluginsystem"
	"pocketbase/services/pluginhost"
)

func init() {
	m.Register(func(app core.App) error {
		records, err := app.FindAllRecords("plugin_instances")
		if err != nil {
			return err
		}
		for _, record := range records {
			config := pluginsystem.JSONMapFromRecord(record, "config")
			record.Set("config", pluginhost.InstanceConfigOverrides(config))
			if err := app.Save(record); err != nil {
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
