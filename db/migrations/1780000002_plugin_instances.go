package migrations

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		// Create plugin_instances collection
		jsonData := `{
			"createRule": "@request.auth.id = user.id",
			"deleteRule": "@request.auth.id = user.id",
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text430001001",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": true,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": true,
					"collectionId": "_pb_users_auth_",
					"hidden": false,
					"id": "relation430001002",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "user",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text430001003",
					"max": 64,
					"min": 1,
					"name": "plugin_id",
					"pattern": "^[a-z0-9][a-z0-9_-]*$",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "bool430001004",
					"name": "enabled",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"hidden": false,
					"id": "json430001005",
					"maxSize": 2000000,
					"name": "auth",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json430001006",
					"maxSize": 2000000,
					"name": "config",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "json430001007",
					"maxSize": 2000000,
					"name": "state",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "select430001008",
					"maxSelect": 1,
					"name": "status",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "select",
					"values": [
						"configured",
						"needs_auth",
						"needs_reauth",
						"syncing",
						"rate_limited",
						"unavailable",
						"unsupported_protocol",
						"error",
						"disabled"
					]
				},
				{
					"hidden": false,
					"id": "json430001009",
					"maxSize": 2000000,
					"name": "last_error",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "json"
				},
				{
					"hidden": false,
					"id": "date430001010",
					"max": "",
					"min": "",
					"name": "last_sync_at",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"hidden": false,
					"id": "date430001011",
					"max": "",
					"min": "",
					"name": "retry_not_before",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "date"
				},
				{
					"hidden": false,
					"id": "autodate430001012",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate430001013",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_430001000",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_plugin_instances_user_plugin_id` + "`" + ` ON ` + "`" + `plugin_instances` + "`" + ` (` + "`" + `user` + "`" + `, ` + "`" + `plugin_id` + "`" + `)"
			],
			"listRule": "@request.auth.id = user.id",
			"name": "plugin_instances",
			"system": false,
			"type": "base",
			"updateRule": "@request.auth.id = user.id",
			"viewRule": "@request.auth.id = user.id"
		}`

		if _, err := app.FindCollectionByNameOrId("pbc_430001000"); err != nil {
			collection := &core.Collection{}
			if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
				return err
			}
			if err := app.Save(collection); err != nil {
				return err
			}
		}

		if err := migrateLegacyIntegrationsToPluginInstances(app); err != nil {
			return err
		}

		// Remove the previous hard-coded provider settings collection after
		// migrating its configuration into plugin_instances. The migration is
		// data-only and does not require the corresponding plugin bundles to be
		// installed.
		if legacyCollection, err := app.FindCollectionByNameOrId("integrations"); err == nil {
			if err := app.Delete(legacyCollection); err != nil {
				return err
			}
		}

		// Add user field to trail_external_reference and update index to be user-scoped
		refCollection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}

		if refCollection.Fields.GetByName("user") == nil {
			if err := refCollection.Fields.AddMarshaledJSONAt(2, []byte(`{
				"cascadeDelete": true,
				"collectionId": "_pb_users_auth_",
				"hidden": false,
				"id": "relation430002001",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "user",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "relation"
			}`)); err != nil {
				return err
			}

			// Replace the global unique (provider, external_id) index with a
			// user-scoped one so the same external trail can be imported by
			// multiple users. This must be managed via the collection metadata
			// (not a raw DROP INDEX), otherwise app.Save would recreate the old
			// index from the still-present metadata entry.
			keptIndexes := refCollection.Indexes[:0]
			for _, idx := range refCollection.Indexes {
				if strings.Contains(idx, "idx_trail_external_reference_provider_external_id") {
					continue
				}
				keptIndexes = append(keptIndexes, idx)
			}
			refCollection.Indexes = append(keptIndexes,
				"CREATE UNIQUE INDEX `idx_trail_external_reference_user_provider_external_id` ON `trail_external_reference` (`user`, `provider`, `external_id`)",
			)

			if err := app.Save(refCollection); err != nil {
				return err
			}

			refs, err := app.FindAllRecords("trail_external_reference")
			if err != nil {
				return err
			}
			for _, ref := range refs {
				trailID := ref.GetString("trail")
				if trailID == "" {
					continue
				}
				trail, err := app.FindRecordById("trails", trailID)
				if err != nil {
					continue
				}
				actor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
				if err != nil {
					continue
				}
				userID := actor.GetString("user")
				if userID == "" {
					continue
				}
				ref.Set("user", userID)
				if err := app.Save(ref); err != nil {
					return err
				}
			}
		}

		return nil
	}, nil)
}

func migrateLegacyIntegrationsToPluginInstances(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("integrations"); err != nil {
		return nil
	}

	records, err := app.FindAllRecords("integrations")
	if err != nil {
		return err
	}
	for _, record := range records {
		userID := record.GetString("user")
		if userID == "" {
			continue
		}

		if raw := legacyJSONObject(record.GetString("strava")); legacyHasValue(raw["clientId"]) {
			auth := legacyPick(raw, "clientId", "clientSecret", "accessToken", "refreshToken", "expiresAt", "tokenType", "scope")
			legacyNormalizeStravaAuth(auth)
			hostConfig := legacyPick(raw, "privacy", "merge")
			hostConfig["planned"] = legacyBool(raw["routes"])
			hostConfig["completed"] = legacyBool(raw["activities"])
			config := legacyNamespacedPluginConfig(
				legacyPick(raw, "after"),
				hostConfig,
			)
			if err := saveLegacyMappedPluginInstance(app, userID, "strava", auth, config, raw); err != nil {
				return err
			}
		}

		if raw := legacyJSONObject(record.GetString("komoot")); legacyHasValue(raw["email"]) {
			auth := legacyPick(raw, "email", "password")
			config := legacyNamespacedPluginConfig(
				legacyPick(raw, "after"),
				legacyPick(raw, "planned", "completed", "privacy", "merge"),
			)
			if err := saveLegacyMappedPluginInstance(app, userID, "komoot", auth, config, raw); err != nil {
				return err
			}
		}

		if raw := legacyJSONObject(record.GetString("hammerhead")); legacyHasValue(raw["email"]) {
			auth := legacyPick(raw, "email", "password")
			config := legacyNamespacedPluginConfig(
				legacyPick(raw, "after"),
				legacyPick(raw, "planned", "completed", "privacy", "merge"),
			)
			if err := saveLegacyMappedPluginInstance(app, userID, "hammerhead", auth, config, raw); err != nil {
				return err
			}
		}
	}
	return nil
}

func legacyNamespacedPluginConfig(pluginConfig map[string]any, hostConfig map[string]any) map[string]any {
	return map[string]any{
		"plugin": nilMap(pluginConfig),
		"host":   nilMap(hostConfig),
	}
}

func saveLegacyMappedPluginInstance(app core.App, userID string, pluginID string, auth map[string]any, config map[string]any, raw map[string]any) error {
	enabled := legacyBool(raw["active"]) && legacyPluginAuthComplete(pluginID, auth)
	return saveLegacyPluginInstance(app, legacyPluginInstance{
		UserID:    userID,
		PluginID:  pluginID,
		Enabled:   enabled,
		Auth:      auth,
		Config:    config,
		State:     map[string]any{},
		Status:    legacyPluginInstanceStatus(pluginID, auth, enabled, ""),
		LastError: map[string]any{},
	})
}

type legacyPluginInstance struct {
	UserID         string
	PluginID       string
	Enabled        bool
	Auth           map[string]any
	Config         map[string]any
	State          map[string]any
	Status         string
	LastError      map[string]any
	LastSyncAt     string
	RetryNotBefore string
}

func saveLegacyPluginInstance(app core.App, instance legacyPluginInstance) error {
	if instance.UserID == "" || instance.PluginID == "" {
		return nil
	}
	existing, _ := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id}",
		dbx.Params{"user": instance.UserID, "plugin_id": instance.PluginID},
	)
	if existing != nil {
		return nil
	}

	authJSON, err := json.Marshal(nilMap(instance.Auth))
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(nilMap(instance.Config))
	if err != nil {
		return err
	}
	stateJSON, err := json.Marshal(nilMap(instance.State))
	if err != nil {
		return err
	}
	lastErrorJSON, err := json.Marshal(nilMap(instance.LastError))
	if err != nil {
		return err
	}
	status := instance.Status
	if status == "" {
		status = legacyPluginInstanceStatus(instance.PluginID, instance.Auth, instance.Enabled, "")
	}

	now := types.NowDateTime().String()
	_, err = app.DB().Insert("plugin_instances", dbx.Params{
		"id":               security.RandomStringWithAlphabet(15, "abcdefghijklmnopqrstuvwxyz0123456789"),
		"user":             instance.UserID,
		"plugin_id":        instance.PluginID,
		"enabled":          instance.Enabled,
		"auth":             string(authJSON),
		"config":           string(configJSON),
		"state":            string(stateJSON),
		"status":           status,
		"last_error":       string(lastErrorJSON),
		"last_sync_at":     instance.LastSyncAt,
		"retry_not_before": instance.RetryNotBefore,
		"created":          now,
		"updated":          now,
	}).Execute()
	return err
}

func legacyPluginInstanceStatus(pluginID string, auth map[string]any, enabled bool, previous string) string {
	if !legacyPluginAuthComplete(pluginID, auth) {
		return "needs_auth"
	}
	if !enabled {
		return "disabled"
	}
	switch previous {
	case "configured", "needs_reauth", "syncing", "rate_limited", "unavailable", "unsupported_protocol", "error":
		return previous
	default:
		return "configured"
	}
}

func legacyPluginAuthComplete(pluginID string, auth map[string]any) bool {
	switch pluginID {
	case "strava":
		return legacyHasValue(auth["clientId"]) && legacyHasValue(auth["clientSecret"]) && legacyHasValue(auth["refreshToken"])
	case "komoot", "hammerhead":
		return legacyHasValue(auth["email"]) && legacyHasValue(auth["password"])
	default:
		return false
	}
}

func legacyNormalizeStravaAuth(auth map[string]any) {
	legacyStringAuthFields(auth, "clientId", "clientSecret", "accessToken", "refreshToken", "tokenType", "scope")
	switch value := auth["expiresAt"].(type) {
	case float64:
		if value > 0 {
			auth["expiresAt"] = time.Unix(int64(value), 0).UTC().Format(time.RFC3339)
		}
	case int64:
		if value > 0 {
			auth["expiresAt"] = time.Unix(value, 0).UTC().Format(time.RFC3339)
		}
	case int:
		if value > 0 {
			auth["expiresAt"] = time.Unix(int64(value), 0).UTC().Format(time.RFC3339)
		}
	}
}

func legacyStringAuthFields(auth map[string]any, keys ...string) {
	for _, key := range keys {
		switch value := auth[key].(type) {
		case string:
			// already normalized
		case float64:
			auth[key] = strconv.FormatFloat(value, 'f', -1, 64)
		case int64:
			auth[key] = strconv.FormatInt(value, 10)
		case int:
			auth[key] = strconv.Itoa(value)
		case nil:
			// leave absent/null values untouched so completeness checks still fail
		default:
			auth[key] = fmt.Sprint(value)
		}
	}
}

func legacyJSONObject(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil || data == nil {
		return map[string]any{}
	}
	return data
}

func legacyPick(src map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, key := range keys {
		if value, ok := src[key]; ok && value != nil {
			out[key] = value
		}
	}
	return out
}

func legacyHasValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return true
	}
}

func legacyBool(value any) bool {
	b, _ := value.(bool)
	return b
}

func nilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
