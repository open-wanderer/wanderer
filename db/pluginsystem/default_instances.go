package pluginsystem

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const defaultPluginUserBatchSize = 100

type defaultPluginUser struct {
	ID string `db:"id"`
}

type firstPartyPluginPolicy struct {
	PluginType         string
	DefaultEnabled     bool
	firstInstallConfig func(config map[string]any)
}

// This registry is owned by Wanderer. Community plugin manifests cannot opt
// themselves into automatic activation or environment-backed compatibility
// imports; IDs here are reserved for first-party plugins. firstInstallConfig is
// invoked once on the first successful discovery, including recovery from an
// initial setup-error placeholder.
var firstPartyPluginPolicies = map[string]firstPartyPluginPolicy{
	"valhalla": {
		PluginType:         PluginTypeRouting,
		DefaultEnabled:     true,
		firstInstallConfig: importLegacyValhallaFirstInstallConfig,
	},
}

// applyFirstPartyPluginFirstInstallConfig is the trusted compatibility boundary
// for configuration that cannot be sourced from an untrusted manifest. Before
// the first routing-plugin release Valhalla was not an installed plugin, so its
// legacy environment value is imported on its first successful discovery. A
// setup-error placeholder does not consume that one-time import. The hook does
// this without allowing a manifest to name or read arbitrary environment
// variables.
func applyFirstPartyPluginFirstInstallConfig(manifest Manifest, config map[string]any) {
	policy, ok := firstPartyPluginPolicies[manifest.ID]
	if !ok || policy.PluginType != manifest.Type || policy.firstInstallConfig == nil {
		return
	}
	policy.firstInstallConfig(config)
}

func importLegacyValhallaFirstInstallConfig(config map[string]any) {
	baseURL := strings.TrimSpace(os.Getenv("VALHALLA_URL"))
	if baseURL == "" {
		return
	}
	hostConfig := firstPartyConfigMap(config, "host")
	connectors := firstPartyConfigMap(hostConfig, "connectors")
	valhalla := firstPartyConfigMap(connectors, "valhalla")
	valhalla["baseURL"] = baseURL
}

func firstPartyConfigMap(config map[string]any, key string) map[string]any {
	if raw, ok := config[key].(map[string]any); ok && raw != nil {
		return raw
	}
	next := map[string]any{}
	config[key] = next
	return next
}

// ProvisionDefaultPluginForAllUsers creates missing instances for an available
// default-active first-party plugin without changing existing user choices.
// A missing row is treated as incomplete provisioning and is repaired; users
// persistently opt out by keeping an existing instance disabled.
func ProvisionDefaultPluginForAllUsers(app core.App, pluginID string) error {
	return provisionDefaultPluginForAllUsers(app, pluginID)
}

func provisionDefaultPluginForAllUsers(app core.App, pluginID string) error {
	installed, err := installedDefaultEnabledPlugin(app, pluginID)
	if err != nil || installed == nil {
		return err
	}
	for {
		users, err := missingDefaultPluginUsers(app, pluginID, defaultPluginUserBatchSize)
		if err != nil {
			return err
		}
		if len(users) == 0 {
			return nil
		}
		if err := app.RunInTransaction(func(txApp core.App) error {
			for _, user := range users {
				if err := ensureDefaultPluginEnabledForUser(txApp, installed, pluginID, user.ID); err != nil {
					return fmt.Errorf("enable %s for user %s: %w", pluginID, user.ID, err)
				}
			}
			return nil
		}); err != nil {
			return err
		}
		if len(users) < defaultPluginUserBatchSize {
			return nil
		}
	}
}

// missingDefaultPluginUsers keeps routine plugin discovery cheap: when all
// defaults are present it performs one indexed anti-join and no per-user
// lookups or writes. When repair is needed it returns only the next missing
// batch. The unique (user, plugin_id) index makes existing disabled instances
// count as present without inspecting or mutating their state.
func missingDefaultPluginUsers(app core.App, pluginID string, limit int) ([]defaultPluginUser, error) {
	users := []defaultPluginUser{}
	err := app.DB().
		Select("users.id AS id").
		From("users").
		LeftJoin(
			"plugin_instances",
			dbx.NewExp(
				"plugin_instances.user = users.id AND plugin_instances.plugin_id = {:plugin_id}",
				dbx.Params{"plugin_id": pluginID},
			),
		).
		Where(dbx.NewExp("plugin_instances.id IS NULL")).
		OrderBy("users.id ASC").
		Limit(int64(limit)).
		All(&users)
	return users, err
}

// EnableDefaultPluginsForUser provisions all installed default-active
// first-party plugins for a newly created user.
func EnableDefaultPluginsForUser(app core.App, userID string) error {
	if userID == "" {
		return nil
	}
	for _, pluginID := range defaultEnabledFirstPartyPluginIDs() {
		installed, err := installedDefaultEnabledPlugin(app, pluginID)
		if err != nil {
			return err
		}
		if installed == nil {
			continue
		}
		if err := ensureDefaultPluginEnabledForUser(app, installed, pluginID, userID); err != nil {
			return fmt.Errorf("enable %s for user %s: %w", pluginID, userID, err)
		}
	}
	return nil
}

func defaultEnabledFirstPartyPluginIDs() []string {
	pluginIDs := make([]string, 0, len(firstPartyPluginPolicies))
	for pluginID, policy := range firstPartyPluginPolicies {
		if policy.DefaultEnabled {
			pluginIDs = append(pluginIDs, pluginID)
		}
	}
	sort.Strings(pluginIDs)
	return pluginIDs
}

func defaultEnabledFirstPartyPlugin(pluginID string, pluginType string) bool {
	policy, ok := firstPartyPluginPolicies[pluginID]
	return ok && policy.DefaultEnabled && policy.PluginType == pluginType
}

func installedDefaultEnabledPlugin(app core.App, pluginID string) (*core.Record, error) {
	policy, ok := firstPartyPluginPolicies[pluginID]
	if !ok || !policy.DefaultEnabled {
		return nil, nil
	}
	record, err := app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id={:plugin_id} && type={:type} && status={:status}",
		dbx.Params{"plugin_id": pluginID, "type": policy.PluginType, "status": "available"},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return record, nil
}

func ensureDefaultPluginEnabledForUser(app core.App, installed *core.Record, pluginID string, userID string) error {
	instance, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id}",
		dbx.Params{"user": userID, "plugin_id": pluginID},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if instance != nil {
		return nil
	}
	if instance == nil {
		collection, err := app.FindCollectionByNameOrId("plugin_instances")
		if err != nil {
			return err
		}
		instance = core.NewRecord(collection)
		instance.Set("user", userID)
		instance.Set("plugin_id", pluginID)
		instance.Set("config", JSONMapFromRecord(installed, "config"))
	}
	instance.Set("enabled", true)
	instance.Set("status", "configured")
	return app.Save(instance)
}
