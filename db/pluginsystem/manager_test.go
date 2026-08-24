package pluginsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestPluginIssueRecordID(t *testing.T) {
	valid := pluginIssueRecordID(LocalPluginIssue{ID: "komoot", Dir: "/plugins/komoot"})
	if valid != "komoot" {
		t.Fatalf("pluginIssueRecordID(valid) = %q, want komoot", valid)
	}

	first := pluginIssueRecordID(LocalPluginIssue{ID: "@@@", Dir: "/plugins/@@@"})
	second := pluginIssueRecordID(LocalPluginIssue{ID: "***", Dir: "/plugins/***"})
	if first == second {
		t.Fatalf("invalid plugin issue ids collided: %q", first)
	}
	for _, got := range []string{first, second} {
		if !pluginIDPattern.MatchString(got) {
			t.Fatalf("pluginIssueRecordID() = %q, not a valid plugin id", got)
		}
	}
}

func TestDefaultEnabledPluginPolicyIsFirstPartyOnly(t *testing.T) {
	if !defaultEnabledFirstPartyPlugin("valhalla", PluginTypeRouting) {
		t.Fatal("expected Valhalla to be default-enabled by host policy")
	}
	if _, exists := firstPartyPluginPolicies["brouter"]; exists {
		t.Fatal("opt-in BRouter must not occupy the trusted host policy registry")
	}
	if defaultEnabledFirstPartyPlugin("brouter", PluginTypeRouting) {
		t.Fatal("expected first-party BRouter to remain opt-in")
	}
	if defaultEnabledFirstPartyPlugin("community-routing", PluginTypeRouting) {
		t.Fatal("community plugin must not become default-enabled")
	}
}

func TestNewValhallaInstallationEnablesInstancesForAllUsers(t *testing.T) {
	t.Setenv("VALHALLA_URL", "https://legacy-initial.example/routing")
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	defer app.ResetBootstrapState()

	installedCollection := core.NewBaseCollection("installed_plugins")
	installedCollection.Fields.Add(
		&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "name"},
		&core.TextField{Name: "type"}, &core.TextField{Name: "version"},
		&core.TextField{Name: "runtime"}, &core.TextField{Name: "path"},
		&core.JSONField{Name: "manifest"}, &core.JSONField{Name: "config"},
		&core.TextField{Name: "status"}, &core.TextField{Name: "error"},
	)
	instanceCollection := core.NewBaseCollection("plugin_instances")
	instanceCollection.Fields.Add(
		&core.TextField{Name: "user"}, &core.TextField{Name: "plugin_id"},
		&core.BoolField{Name: "enabled"}, &core.JSONField{Name: "config"},
		&core.TextField{Name: "status"},
	)
	for _, collection := range []*core.Collection{installedCollection, instanceCollection} {
		if err := app.Save(collection); err != nil {
			t.Fatalf("save %s collection: %v", collection.Name, err)
		}
	}
	userCollection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	users := make([]*core.Record, 2)
	for index := range users {
		users[index] = core.NewRecord(userCollection)
		users[index].SetEmail(fmt.Sprintf("routing-user-%d@example.com", index))
		users[index].SetPassword("test-password-123")
		if err := app.Save(users[index]); err != nil {
			t.Fatalf("save user %d: %v", index, err)
		}
	}

	pluginRoot := t.TempDir()
	manifest := Manifest{
		ManifestVersion: ManifestVersion,
		ID:              "valhalla", Type: PluginTypeRouting,
		Name: "Valhalla", Version: "1.0.0",
		Runtime: RuntimeManifest{Type: RuntimeWASM, Entrypoint: "plugin.wasm"},
		Capabilities: []CapabilityManifest{
			{Name: "route", Version: "v1", Export: "route_v1"},
			{Name: "elevation", Version: "v1", Export: "elevation_v1"},
		},
	}
	pluginDir := filepath.Join(pluginRoot, "valhalla")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatalf("create Valhalla plugin directory: %v", err)
	}
	manifestPath := filepath.Join(pluginDir, "plugin.json")
	if err := os.WriteFile(manifestPath, []byte(`{"manifestVersion":`), 0o600); err != nil {
		t.Fatalf("write invalid Valhalla manifest: %v", err)
	}

	manager := NewManager(app, pluginRoot)
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("sync Valhalla setup error: %v", err)
	}
	placeholder, err := app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find Valhalla setup-error placeholder: %v", err)
	}
	if placeholder.GetString("status") != "error" {
		t.Fatalf("Valhalla placeholder status = %q, want error", placeholder.GetString("status"))
	}
	if pluginRecordHadSuccessfulDiscovery(placeholder, manifest) {
		t.Fatal("initial setup-error placeholder was marked as previously successful")
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("encode Valhalla manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestJSON, 0o600); err != nil {
		t.Fatalf("write Valhalla manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.wasm"), []byte("wasm"), 0o600); err != nil {
		t.Fatalf("write Valhalla entrypoint: %v", err)
	}

	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("sync first successful Valhalla discovery: %v", err)
	}
	installed, err := app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find newly installed Valhalla record: %v", err)
	}
	assertValhallaConnectorBaseURL(t, installed, "https://legacy-initial.example/routing")
	// The environment value is a one-time compatibility import. Routine syncs
	// must preserve the already stored administrator-owned connector config.
	t.Setenv("VALHALLA_URL", "https://must-not-be-reimported.example/routing")
	for _, user := range users {
		instance, err := app.FindFirstRecordByFilter(
			"plugin_instances",
			"user='"+user.Id+"' && plugin_id='valhalla'",
			nil,
		)
		if err != nil || instance == nil || !instance.GetBool("enabled") {
			t.Fatalf("Valhalla instance for user %s was not enabled: record=%v err=%v", user.Id, instance, err)
		}
	}

	// A later broken bundle must retain the fact that first-install compatibility
	// was already applied, while its cache snapshot remains invalid and therefore
	// unusable by request paths. Recovery must not import a changed environment
	// value a second time.
	if err := os.WriteFile(manifestPath, []byte(`{"manifestVersion":`), 0o600); err != nil {
		t.Fatalf("break Valhalla manifest after successful discovery: %v", err)
	}
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("sync later Valhalla setup error: %v", err)
	}
	placeholder, err = app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find recovered Valhalla setup-error placeholder: %v", err)
	}
	if !pluginRecordHadSuccessfulDiscovery(placeholder, manifest) {
		t.Fatal("later setup-error placeholder forgot the earlier successful discovery")
	}
	if placeholder.GetString("type") != PluginTypeRouting {
		t.Fatalf("later setup-error placeholder type = %q, want routing", placeholder.GetString("type"))
	}
	if pluginRecordHadSuccessfulDiscovery(placeholder, Manifest{ID: "replacement", Type: PluginTypeRouting}) {
		t.Fatal("setup-error success marker leaked across plugin identities")
	}
	if _, err := localPluginFromRecord(placeholder); err == nil {
		t.Fatal("setup-error placeholder unexpectedly exposed stale executable metadata")
	}
	if err := os.WriteFile(manifestPath, manifestJSON, 0o600); err != nil {
		t.Fatalf("restore Valhalla manifest: %v", err)
	}
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("sync recovered Valhalla bundle: %v", err)
	}
	installed, err = app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find recovered installed Valhalla record: %v", err)
	}
	assertValhallaConnectorBaseURL(t, installed, "https://legacy-initial.example/routing")

	// Removing and later reinstalling the bundle must neither expose its stale
	// cached manifest nor consume first-install compatibility a second time. The
	// retained invalid cache record preserves both that history and admin config.
	installedID := installed.Id
	detachedDir := filepath.Join(t.TempDir(), "valhalla")
	if err := os.Rename(pluginDir, detachedDir); err != nil {
		t.Fatalf("temporarily remove Valhalla bundle: %v", err)
	}
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("sync removed Valhalla bundle: %v", err)
	}
	retained, err := app.FindRecordById("installed_plugins", installedID)
	if err != nil {
		t.Fatalf("find retained Valhalla cache record: %v", err)
	}
	if retained.GetString("status") != "error" || retained.GetString("error") != missingDefaultPluginBundleError || retained.GetString("path") != "" {
		t.Fatalf("retained Valhalla cache state: status=%q error=%q path=%q", retained.GetString("status"), retained.GetString("error"), retained.GetString("path"))
	}
	if _, err := localPluginFromRecord(retained); err == nil {
		t.Fatal("removed Valhalla bundle remained executable through its cache record")
	}
	assertValhallaConnectorBaseURL(t, retained, "https://legacy-initial.example/routing")
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("repeat sync while Valhalla bundle is absent: %v", err)
	}
	if err := os.Rename(detachedDir, pluginDir); err != nil {
		t.Fatalf("reinstall Valhalla bundle: %v", err)
	}
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("sync reinstalled Valhalla bundle: %v", err)
	}
	installed, err = app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find reinstalled Valhalla record: %v", err)
	}
	if installed.Id != installedID || installed.GetString("status") != "available" {
		t.Fatalf("reinstalled Valhalla cache record: id=%q status=%q, want id=%q status=available", installed.Id, installed.GetString("status"), installedID)
	}
	assertValhallaConnectorBaseURL(t, installed, "https://legacy-initial.example/routing")

	first, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user='"+users[0].Id+"' && plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find first Valhalla instance: %v", err)
	}
	firstID := first.Id
	first.Set("enabled", false)
	first.Set("status", "disabled")
	first.Set("config", map[string]any{"userChoice": "disabled-instance"})
	if err := app.Save(first); err != nil {
		t.Fatalf("disable first Valhalla instance: %v", err)
	}
	second, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user='"+users[1].Id+"' && plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find second Valhalla instance: %v", err)
	}
	secondID := second.Id
	second.Set("config", map[string]any{"userChoice": "enabled-instance"})
	if err := app.Save(second); err != nil {
		t.Fatalf("customize second Valhalla instance: %v", err)
	}
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("resync installed plugins: %v", err)
	}
	first, err = app.FindRecordById("plugin_instances", firstID)
	if err != nil {
		t.Fatalf("reload first Valhalla instance: %v", err)
	}
	if first.GetBool("enabled") || first.GetString("status") != "disabled" {
		t.Fatalf("routine plugin sync changed disabled instance: enabled=%v status=%q", first.GetBool("enabled"), first.GetString("status"))
	}
	if got := JSONMapFromRecord(first, "config")["userChoice"]; got != "disabled-instance" {
		t.Fatalf("routine plugin sync changed disabled instance config: %#v", got)
	}
	second, err = app.FindRecordById("plugin_instances", secondID)
	if err != nil {
		t.Fatalf("reload second Valhalla instance: %v", err)
	}
	if !second.GetBool("enabled") || second.GetString("status") != "configured" {
		t.Fatalf("routine plugin sync changed enabled instance: enabled=%v status=%q", second.GetBool("enabled"), second.GetString("status"))
	}
	if got := JSONMapFromRecord(second, "config")["userChoice"]; got != "enabled-instance" {
		t.Fatalf("routine plugin sync changed enabled instance config: %#v", got)
	}

	// Absence represents interrupted provisioning, not opt-out. Deleting the row
	// is therefore repaired on the next sync; keeping it disabled above is the
	// durable user choice and remains untouched.
	if err := app.Delete(second); err != nil {
		t.Fatalf("delete second Valhalla instance: %v", err)
	}
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("repair missing Valhalla instance: %v", err)
	}
	repaired, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user='"+users[1].Id+"' && plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find repaired Valhalla instance: %v", err)
	}
	if repaired.Id == secondID {
		t.Fatal("missing Valhalla instance was not recreated")
	}
	if !repaired.GetBool("enabled") || repaired.GetString("status") != "configured" {
		t.Fatalf("repaired Valhalla instance: enabled=%v status=%q", repaired.GetBool("enabled"), repaired.GetString("status"))
	}
	first, err = app.FindRecordById("plugin_instances", firstID)
	if err != nil {
		t.Fatalf("reload disabled Valhalla instance after repair: %v", err)
	}
	if first.GetBool("enabled") || JSONMapFromRecord(first, "config")["userChoice"] != "disabled-instance" {
		t.Fatal("repair of a missing peer changed the disabled Valhalla instance")
	}
	installed, err = app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id='valhalla'",
		nil,
	)
	if err != nil {
		t.Fatalf("find installed Valhalla record: %v", err)
	}
	assertValhallaConnectorBaseURL(t, installed, "https://legacy-initial.example/routing")
	installed.Set("status", "error")
	if err := app.Save(installed); err != nil {
		t.Fatalf("mark installed Valhalla unavailable: %v", err)
	}
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		t.Fatalf("resync recovered Valhalla plugin: %v", err)
	}
	first, err = app.FindRecordById("plugin_instances", first.Id)
	if err != nil {
		t.Fatalf("reload Valhalla after availability transition: %v", err)
	}
	if first.GetBool("enabled") {
		t.Fatal("availability transition re-enabled an explicitly disabled existing instance")
	}

	newUser := core.NewRecord(userCollection)
	newUser.SetEmail("new-routing-user@example.com")
	newUser.SetPassword("test-password-123")
	if err := app.Save(newUser); err != nil {
		t.Fatalf("save new user: %v", err)
	}
	if err := EnableDefaultPluginsForUser(app, newUser.Id); err != nil {
		t.Fatalf("enable Valhalla for new user: %v", err)
	}
	newInstance, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user='"+newUser.Id+"' && plugin_id='valhalla'",
		nil,
	)
	if err != nil || newInstance == nil || !newInstance.GetBool("enabled") {
		t.Fatalf("Valhalla instance for new user was not enabled: record=%v err=%v", newInstance, err)
	}
}

func assertValhallaConnectorBaseURL(t *testing.T, record *core.Record, expected string) {
	t.Helper()
	config := JSONMapFromRecord(record, "config")
	host, ok := config["host"].(map[string]any)
	if !ok {
		t.Fatalf("installed plugin host config missing: %#v", config)
	}
	connectors, ok := host["connectors"].(map[string]any)
	if !ok {
		t.Fatalf("installed plugin connectors missing: %#v", host)
	}
	valhalla, ok := connectors["valhalla"].(map[string]any)
	if !ok {
		t.Fatalf("installed Valhalla connector missing: %#v", connectors)
	}
	if got := valhalla["baseURL"]; got != expected {
		t.Fatalf("installed Valhalla baseURL = %#v, want %q", got, expected)
	}
}
