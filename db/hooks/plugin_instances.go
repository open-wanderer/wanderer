package hooks

import (
	"encoding/json"
	"os"

	"github.com/pocketbase/dbx"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"

	"pocketbase/pluginsystem"
)

// ListPluginInstanceHandler censors auth values before plugin instances leave
// the API. The database keeps encrypted secrets, but normal users never receive
// the encrypted payload either.
func ListPluginInstanceHandler() func(e *core.RecordsListRequestEvent) error {
	return func(e *core.RecordsListRequestEvent) error {
		if e.HasSuperuserAuth() {
			return e.Next()
		}
		for _, r := range e.Records {
			censorPluginInstanceAuth(e.App, r)
		}

		return e.Next()
	}
}

// ViewPluginInstanceHandler applies the same auth censoring for single-record
// reads that ListPluginInstanceHandler applies for list reads.
func ViewPluginInstanceHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if e.HasSuperuserAuth() {
			return e.Next()
		}
		censorPluginInstanceAuth(e.App, e.Record)

		return e.Next()
	}
}

// CreatePluginInstanceHandler normalizes initial status and encrypts submitted
// auth fields before a plugin instance is persisted.
func CreatePluginInstanceHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		ensurePluginInstanceStatus(e.Record)
		mergePluginInstanceDefaultConfig(e.App, e.Record)
		if err := encryptPluginInstanceAuth(e.App, e.Record); err != nil {
			return err
		}

		return e.Next()
	}
}

// CreateUpdatePluginInstanceSuccessHandler censors auth values in the response
// body after PocketBase has stored the encrypted values.
func CreateUpdatePluginInstanceSuccessHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		censorPluginInstanceAuth(e.App, e.Record)
		return e.Next()
	}
}

// UpdatePluginInstanceHandler re-applies status defaults and encrypts any
// changed auth fields before the update is persisted.
func UpdatePluginInstanceHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		ensurePluginInstanceStatus(e.Record)
		mergePluginInstanceDefaultConfig(e.App, e.Record)
		if err := encryptPluginInstanceAuth(e.App, e.Record); err != nil {
			return err
		}

		return e.Next()
	}
}

func mergePluginInstanceDefaultConfig(app core.App, r *core.Record) {
	defaults := installedPluginDefaultConfig(app, r.GetString("plugin_id"))
	if len(defaults) == 0 {
		return
	}
	merged := pluginsystem.CloneJSONMap(defaults)
	pluginsystem.MergePluginConfig(merged, pluginsystem.JSONMapFromRecord(r, "config"))
	r.Set("config", merged)
}

func installedPluginDefaultConfig(app core.App, pluginID string) map[string]any {
	if pluginID == "" {
		return map[string]any{}
	}
	record, _ := app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id={:plugin_id}",
		dbx.Params{"plugin_id": pluginID},
	)
	if record == nil {
		return map[string]any{}
	}
	return pluginsystem.JSONMapFromRecord(record, "config")
}

func censorPluginInstanceAuth(app core.App, r *core.Record) {
	if authString := r.GetString("auth"); authString != "" {
		var auth map[string]any
		if err := json.Unmarshal([]byte(authString), &auth); err != nil {
			r.Set("auth", "{}")
			return
		}

		secretFields := pluginInstanceSecretFields(app, r.GetString("plugin_id"))
		encryptAll := len(secretFields) == 0
		for key := range auth {
			if encryptAll || secretFields[key] {
				auth[key] = ""
			}
		}

		b, err := json.Marshal(auth)
		if err != nil {
			r.Set("auth", "{}")
			return
		}
		r.Set("auth", string(b))
	}
}

func ensurePluginInstanceStatus(r *core.Record) {
	if r.GetString("status") != "" {
		return
	}
	if r.GetString("auth") == "" {
		r.Set("status", "needs_auth")
		return
	}
	if r.GetBool("enabled") {
		r.Set("status", "configured")
		return
	}
	r.Set("status", "disabled")
}

func encryptPluginInstanceAuth(app core.App, r *core.Record) error {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
	}

	authString := r.GetString("auth")
	if authString == "" {
		return nil
	}

	var auth map[string]any
	if err := json.Unmarshal([]byte(authString), &auth); err != nil {
		return err
	}
	if auth == nil {
		return nil
	}

	var originalAuth map[string]any
	if original, _ := app.FindRecordById("plugin_instances", r.Id); original != nil {
		originalString := original.GetString("auth")
		if originalString != "" {
			_ = json.Unmarshal([]byte(originalString), &originalAuth)
		}
	}

	secretFields := pluginInstanceSecretFields(app, r.GetString("plugin_id"))
	encryptAll := len(secretFields) == 0
	if originalAuth != nil {
		for key, value := range originalAuth {
			if _, ok := auth[key]; ok {
				continue
			}
			if encryptAll || secretFields[key] {
				auth[key] = value
			}
		}
	}

	for key, value := range auth {
		secret, ok := value.(string)
		if !ok {
			continue
		}
		if secret == "" {
			if originalAuth != nil {
				if restored, ok := originalAuth[key].(string); ok && restored != "" {
					secret = restored
				}
			}
			if secret == "" {
				continue
			}
		}
		if !encryptAll && !secretFields[key] {
			auth[key] = secret
			continue
		}
		if util.CanDecryptSecret(secret) {
			auth[key] = secret
			continue
		}
		encryptedSecret, err := security.Encrypt([]byte(secret), encryptionKey)
		if err != nil {
			return err
		}
		auth[key] = encryptedSecret
	}

	b, err := json.Marshal(auth)
	if err != nil {
		return err
	}
	r.Set("auth", string(b))

	return nil
}

func pluginInstanceSecretFields(app core.App, pluginID string) map[string]bool {
	manifest, ok := pluginInstancePluginManifest(app, pluginID)
	if !ok {
		return nil
	}

	fields := map[string]bool{}
	for _, field := range pluginsystem.InternalAuthSecretFields() {
		fields[field] = true
	}
	for _, context := range manifest.Auth.Contexts {
		if context.SecretField != "" {
			fields[context.SecretField] = true
		}
		for _, field := range context.SecretFields {
			fields[field] = true
		}
	}
	return fields
}

func pluginInstancePluginManifest(app core.App, pluginID string) (pluginsystem.Manifest, bool) {
	record, _ := app.FindFirstRecordByFilter(
		"installed_plugins",
		"plugin_id={:plugin_id}",
		dbx.Params{"plugin_id": pluginID},
	)
	if record != nil {
		var manifest pluginsystem.Manifest
		if err := record.UnmarshalJSONField("manifest", &manifest); err == nil && manifest.ID != "" {
			return manifest, true
		}
	}

	plugins, err := pluginsystem.LoadLocalPlugins("")
	if err != nil {
		return pluginsystem.Manifest{}, false
	}
	for _, plugin := range plugins {
		if plugin.Manifest.ID == pluginID {
			return plugin.Manifest, true
		}
	}
	return pluginsystem.Manifest{}, false
}
