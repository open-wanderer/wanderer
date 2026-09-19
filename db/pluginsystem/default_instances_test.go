package pluginsystem

import (
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestFirstPartyPluginFirstInstallConfigIsTrustedAndScoped(t *testing.T) {
	t.Setenv("VALHALLA_URL", " https://legacy-routing.example/base ")

	config := map[string]any{
		"host": map[string]any{
			"connectors": map[string]any{
				"valhalla": map[string]any{"allowPrivate": false},
			},
		},
	}
	applyFirstPartyPluginFirstInstallConfig(Manifest{ID: "valhalla", Type: PluginTypeRouting}, config)
	valhalla := config["host"].(map[string]any)["connectors"].(map[string]any)["valhalla"].(map[string]any)
	if got := valhalla["baseURL"]; got != "https://legacy-routing.example/base" {
		t.Fatalf("legacy Valhalla baseURL = %#v", got)
	}
	if got := valhalla["allowPrivate"]; got != false {
		t.Fatalf("unrelated connector config changed: %#v", got)
	}

	for _, manifest := range []Manifest{
		{ID: "community-routing", Type: PluginTypeRouting},
		{ID: "brouter", Type: PluginTypeRouting},
		{ID: "valhalla", Type: PluginTypeTrails},
	} {
		candidate := map[string]any{"host": map[string]any{"marker": "unchanged"}}
		before := CloneJSONMap(candidate)
		applyFirstPartyPluginFirstInstallConfig(manifest, candidate)
		if !reflect.DeepEqual(candidate, before) {
			t.Fatalf("manifest %s/%s received trusted first-install config: %#v", manifest.ID, manifest.Type, candidate)
		}
	}
}

func TestFirstPartyPluginFirstInstallConfigIgnoresEmptyLegacyValue(t *testing.T) {
	t.Setenv("VALHALLA_URL", " \t ")
	config := map[string]any{"host": map[string]any{"marker": "unchanged"}}
	before := CloneJSONMap(config)
	applyFirstPartyPluginFirstInstallConfig(Manifest{ID: "valhalla", Type: PluginTypeRouting}, config)
	if !reflect.DeepEqual(config, before) {
		t.Fatalf("empty legacy value changed config: %#v", config)
	}
}

func TestDefaultPluginQueriesPropagateDatabaseErrors(t *testing.T) {
	t.Run("installed plugin lookup", func(t *testing.T) {
		app := newDefaultInstanceTestApp(t)
		collection := core.NewBaseCollection("installed_plugins")
		collection.Fields.Add(
			&core.TextField{Name: "plugin_id"},
			&core.TextField{Name: "type"},
			&core.TextField{Name: "status"},
		)
		if err := app.Save(collection); err != nil {
			t.Fatalf("save installed plugins collection: %v", err)
		}
		if _, err := app.NonconcurrentDB().NewQuery("DROP TABLE installed_plugins").Execute(); err != nil {
			t.Fatalf("drop installed plugins table: %v", err)
		}
		if _, err := installedDefaultEnabledPlugin(app, "valhalla"); err == nil {
			t.Fatal("installed plugin query error was swallowed")
		}
	})

	t.Run("plugin instance lookup", func(t *testing.T) {
		app := newDefaultInstanceTestApp(t)
		instanceCollection := core.NewBaseCollection("plugin_instances")
		instanceCollection.Fields.Add(
			&core.TextField{Name: "user"},
			&core.TextField{Name: "plugin_id"},
		)
		if err := app.Save(instanceCollection); err != nil {
			t.Fatalf("save plugin instances collection: %v", err)
		}
		if _, err := app.NonconcurrentDB().NewQuery("DROP TABLE plugin_instances").Execute(); err != nil {
			t.Fatalf("drop plugin instances table: %v", err)
		}
		installedCollection := core.NewBaseCollection("installed_plugins_stub")
		installedCollection.Fields.Add(&core.JSONField{Name: "config"})
		installed := core.NewRecord(installedCollection)
		if err := ensureDefaultPluginEnabledForUser(app, installed, "valhalla", "user-id"); err == nil {
			t.Fatal("plugin instance query error was swallowed")
		}
	})
}

func newDefaultInstanceTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	return app
}
