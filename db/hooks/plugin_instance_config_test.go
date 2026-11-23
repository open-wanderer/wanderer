package hooks

import (
	"reflect"
	"testing"

	"pocketbase/pluginsystem"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestPluginInstanceSavesUseAdministrativeConnectorSettings(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "01234567890123456789012345678901")
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app.Settings().Logs.MaxDays = 0
	t.Cleanup(app.Cleanup)
	installed := core.NewBaseCollection("installed_plugins")
	installed.Fields.Add(&core.TextField{Name: "plugin_id"}, &core.JSONField{Name: "config"})
	instances := core.NewBaseCollection("plugin_instances")
	instances.Fields.Add(&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "status"}, &core.JSONField{Name: "auth"}, &core.JSONField{Name: "config"})
	for _, collection := range []*core.Collection{installed, instances} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	adminConnectors := map[string]any{"api": map[string]any{"baseURL": "https://a.example", "allowPrivate": true, "tls": map[string]any{"mode": "custom_ca", "caBundle": "admin-ca"}}}
	defaults := core.NewRecord(installed)
	defaults.Set("plugin_id", "review-trails")
	defaults.Set("config", map[string]any{"plugin": map[string]any{"after": "2026-01-01"}, "host": map[string]any{"planned": false, "completed": true, "privacy": "settings", "connectors": adminConnectors}})
	if err := app.Save(defaults); err != nil {
		t.Fatal(err)
	}
	app.OnRecordCreate("plugin_instances").BindFunc(CreatePluginInstanceHandler())
	app.OnRecordUpdate("plugin_instances").BindFunc(UpdatePluginInstanceHandler())
	userConnector := map[string]any{"api": map[string]any{"baseURL": "https://b.example", "allowPrivate": false, "tls": map[string]any{"mode": "system"}}}
	record := core.NewRecord(instances)
	record.Set("plugin_id", "review-trails")
	record.Set("config", map[string]any{"plugin": map[string]any{"after": "2026-02-01"}, "host": map[string]any{"planned": true, "completed": false, "categoryRemapDismissedAt": "2026-02-01T00:00:00Z", "connectors": userConnector}})
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	assertSaved := func(wantPlanned bool) *core.Record {
		t.Helper()
		stored, err := app.FindRecordById("plugin_instances", record.Id)
		if err != nil {
			t.Fatal(err)
		}
		config := pluginsystem.JSONMapFromRecord(stored, "config")
		host := config["host"].(map[string]any)
		if host["connectors"] != nil {
			t.Fatalf("instance connector settings persisted: %#v", host["connectors"])
		}
		if host["planned"] != wantPlanned || host["completed"] != false || host["privacy"] != "settings" || host["categoryRemapDismissedAt"] != "2026-02-01T00:00:00Z" {
			t.Fatalf("ordinary instance preferences changed: %#v", host)
		}
		if config["plugin"].(map[string]any)["after"] != "2026-02-01" {
			t.Fatalf("plugin setting changed: %#v", config)
		}
		admin, err := app.FindRecordById("installed_plugins", defaults.Id)
		if err != nil {
			t.Fatal(err)
		}
		adminConfig := pluginsystem.JSONMapFromRecord(admin, "config")
		if !reflect.DeepEqual(adminConfig["host"].(map[string]any)["connectors"], adminConnectors) {
			t.Fatalf("administrative connector settings changed: %#v", adminConfig)
		}
		return stored
	}
	record = assertSaved(true)
	updated := pluginsystem.JSONMapFromRecord(record, "config")
	host := updated["host"].(map[string]any)
	host["connectors"] = userConnector
	host["planned"] = false
	record.Set("config", updated)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	assertSaved(false)
}
