package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
)

const (
	defaultPluginSyncBatchLimit = 10
	defaultPluginSyncMaxBatches = 100
)

var syncCapabilityDescriptors = []syncCapabilityDescriptor{
	{
		OptionKey:      "planned",
		CapabilityName: "list_routes",
		Version:        "v1",
		StateKey:       "list_routes.v1",
	},
	{
		OptionKey:      "completed",
		CapabilityName: "list_activities",
		Version:        "v1",
		StateKey:       "list_activities.v1",
	},
}

type syncCapabilityDescriptor struct {
	OptionKey      string
	CapabilityName string
	Version        string
	StateKey       string
}

type pluginSystemSyncRequest struct {
	PluginID string `json:"pluginId"`
}

type pluginSystemListInput struct {
	Instance          pluginsystem.InstanceRef `json:"instance"`
	Auth              map[string]any           `json:"auth,omitempty"`
	State             map[string]any           `json:"state,omitempty"`
	Options           map[string]any           `json:"options,omitempty"`
	Limits            pluginSystemSyncLimits   `json:"limits,omitempty"`
	RecentExternalIDs []string                 `json:"recentExternalIds,omitempty"`
}

type pluginSystemSyncLimits struct {
	MaxItems int `json:"maxItems,omitempty"`
}

type pluginSystemListOutput struct {
	Items   []pluginsystem.TrailImport `json:"items"`
	State   map[string]any             `json:"state,omitempty"`
	HasMore bool                       `json:"hasMore"`
	Error   *pluginsystem.PluginError  `json:"error,omitempty"`
}

type pluginSystemSyncResult struct {
	PluginID string `json:"pluginId"`
	Imported int    `json:"imported"`
	Skipped  int    `json:"skipped"`
}

// PluginSystemSync runs a manual sync for the authenticated user's enabled
// instance of one plugin.
func PluginSystemSync(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginSystemSyncRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.PluginID == "" {
		return apis.NewBadRequestError("pluginId is required", nil)
	}

	instance, err := e.App.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id} && enabled=true",
		dbx.Params{"user": e.Auth.Id, "plugin_id": data.PluginID},
	)
	if err != nil {
		return apis.NewBadRequestError("no enabled plugin instance configured for this plugin", nil)
	}

	plugin, err := localPlugin(e.App, data.PluginID)
	if err != nil {
		return err
	}
	result, err := syncPluginInstance(e.Request.Context(), e.App, plugin, instance)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, result)
}

// PluginSystemSyncConfigured is the cron entrypoint. It refreshes plugin
// metadata, finds enabled instances, skips instances in backoff, and syncs each
// configured import capability.
func PluginSystemSyncConfigured(ctx context.Context, app core.App) error {
	manager := pluginsystem.NewManager(app, "")
	if err := manager.SyncInstalledPlugins(ctx); err != nil {
		return err
	}
	plugins, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err != nil {
		return err
	}

	var syncErr error
	for _, plugin := range plugins {
		if !pluginHasAnySyncCapability(plugin) {
			continue
		}
		instances, err := pluginInstances(app, plugin.Manifest.ID)
		if err != nil {
			return err
		}
		for _, instance := range instances {
			if err := ctx.Err(); err != nil {
				return err
			}
			if shouldSkipPluginInstance(instance) {
				continue
			}
			result, err := syncPluginInstance(ctx, app, plugin, instance)
			if err != nil {
				app.Logger().Warn("plugin instance sync failed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "error", err)
				syncErr = err
				continue
			}
			app.Logger().Info("plugin instance sync completed", "plugin", result.PluginID, "instance", instance.Id, "imported", result.Imported, "skipped", result.Skipped)
		}
	}
	return syncErr
}

func pluginInstances(app core.App, pluginID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter(
		"plugin_instances",
		"plugin_id={:plugin_id} && enabled=true",
		"",
		-1,
		0,
		dbx.Params{"plugin_id": pluginID},
	)
}

// syncPluginInstance prepares one plugin instance for import: it resolves the
// actor, creates the runtime, decrypts/refreshes auth, and dispatches every
// enabled sync capability.
func syncPluginInstance(ctx context.Context, app core.App, plugin pluginsystem.LocalPlugin, instance *core.Record) (*pluginSystemSyncResult, error) {
	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", instance.GetString("user"))
	if err != nil {
		setPluginInstanceStatus(app, instance, "error", "invalid_request", "activitypub actor not found")
		return nil, err
	}

	runtime, err := pluginsystem.NewRuntimeRegistry().RuntimeFor(plugin)
	if err != nil {
		return nil, err
	}
	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		setPluginInstanceStatus(app, instance, "needs_reauth", "auth_failed", err.Error())
		return nil, err
	}
	auth, err = pluginsystem.RefreshOAuthAuthIfNeeded(ctx, app, plugin, instance, auth)
	if err != nil {
		setPluginInstanceStatus(app, instance, "needs_reauth", "auth_failed", err.Error())
		return nil, err
	}
	config := jsonMapFromRecord(instance, "config")
	state := jsonMapFromRecord(instance, "state")
	defaultPublic := userDefaultPublic(app, instance.GetString("user"))
	createSummitLog := boolOption(config, "createSummitLogForCompleted", true)

	instance.Set("status", "syncing")
	if err := app.Save(instance); err != nil {
		return nil, err
	}

	result := &pluginSystemSyncResult{PluginID: plugin.Manifest.ID}
	for _, descriptor := range syncCapabilityDescriptors {
		if !boolOption(config, descriptor.OptionKey, true) || !pluginHasCapability(plugin, descriptor.CapabilityName, descriptor.Version) {
			continue
		}
		capability, err := pluginCapability(plugin, descriptor.CapabilityName, descriptor.Version)
		if err != nil {
			return nil, err
		}
		capResult, err := syncPluginCapability(ctx, app, runtime, plugin, capability, instance, actor, auth, config, stateForCapability(state, descriptor.StateKey), defaultPublic, createSummitLog)
		if err != nil {
			setPluginInstanceStatusForError(app, instance, err)
			return nil, err
		}
		result.Imported += capResult.Imported
		result.Skipped += capResult.Skipped
		state[descriptor.StateKey] = capResult.State
	}

	instance.Set("state", state)
	instance.Set("last_sync_at", time.Now())
	instance.Set("last_error", map[string]any{})
	instance.Set("next_retry_at", "")
	instance.Set("status", "configured")
	if err := app.Save(instance); err != nil {
		return nil, err
	}
	return result, nil
}

// shouldSkipPluginInstance applies provider backoff from the last sync error.
func shouldSkipPluginInstance(instance *core.Record) bool {
	nextRetry := instance.GetDateTime("next_retry_at")
	return !nextRetry.IsZero() && nextRetry.Time().After(time.Now())
}

type capabilitySyncResult struct {
	Imported int
	Skipped  int
	State    map[string]any
}

// syncPluginCapability calls one plugin export such as list_routes_v1, imports
// the returned trail items, and carries the capability cursor state forward.
func syncPluginCapability(ctx context.Context, app core.App, runtime pluginsystem.Runtime, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, actor *core.Record, auth map[string]any, config map[string]any, state map[string]any, defaultPublic bool, createSummitLog bool) (*capabilitySyncResult, error) {
	result := &capabilitySyncResult{State: state}
	recentIDs := recentExternalIDs(app, instance.GetString("plugin_id"), instance.GetString("user"))
	hasMore := true
	for batch := 0; hasMore && batch < defaultPluginSyncMaxBatches; batch++ {
		input := pluginSystemListInput{
			Instance: pluginsystem.InstanceRef{
				ID:       instance.Id,
				PluginID: instance.GetString("plugin_id"),
			},
			Auth:              pluginsystem.PluginInputAuth(plugin, auth),
			State:             result.State,
			Options:           config,
			Limits:            pluginSystemSyncLimits{MaxItems: defaultPluginSyncBatchLimit},
			RecentExternalIDs: recentIDs,
		}
		inputBytes, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		outputBytes, err := runtime.Call(ctx, plugin, capability.Export, inputBytes, pluginInstancePolicy(plugin, config))
		if err != nil {
			return nil, err
		}
		var output pluginSystemListOutput
		if err := json.Unmarshal(outputBytes, &output); err != nil {
			return nil, fmt.Errorf("plugin returned invalid %s output: %w", capability.Export, err)
		}
		if output.Error != nil {
			return nil, pluginsystem.PluginCapabilityError{Err: output.Error}
		}

		for _, item := range output.Items {
			applyImportPolicy(&item, config)
			imported, err := importer.ImportTrail(ctx, app, item, importer.Options{
				UserID:                      instance.GetString("user"),
				ActorID:                     actor.Id,
				DefaultPublic:               defaultPublic,
				CreateSummitLogForCompleted: createSummitLog,
			})
			if err != nil {
				return nil, err
			}
			if imported.Created {
				result.Imported++
				app.Logger().Info("imported plugin trail", "provider", item.Source.Provider, "external_id", item.Source.ExternalID, "trail", imported.TrailID)
			}
			if imported.Skipped {
				result.Skipped++
			}
		}

		result.State = output.State
		hasMore = output.HasMore
	}
	if hasMore {
		return nil, fmt.Errorf("sync stopped after %d batches", defaultPluginSyncMaxBatches)
	}
	if result.State == nil {
		result.State = map[string]any{}
	}
	return result, nil
}

// pluginCapability returns the manifest entry for a concrete capability/version
// pair so the host can call the export declared by the plugin.
func pluginCapability(plugin pluginsystem.LocalPlugin, name string, version string) (pluginsystem.CapabilityManifest, error) {
	for _, capability := range plugin.Manifest.Capabilities {
		if capability.Name == name && capability.Version == version {
			return capability, nil
		}
	}
	return pluginsystem.CapabilityManifest{}, apis.NewBadRequestError("plugin capability is not available", map[string]string{
		"name":    name,
		"version": version,
	})
}

func pluginHasCapability(plugin pluginsystem.LocalPlugin, name string, version string) bool {
	for _, capability := range plugin.Manifest.Capabilities {
		if capability.Name == name && capability.Version == version {
			return true
		}
	}
	return false
}

func pluginHasAnySyncCapability(plugin pluginsystem.LocalPlugin) bool {
	for _, descriptor := range syncCapabilityDescriptors {
		if pluginHasCapability(plugin, descriptor.CapabilityName, descriptor.Version) {
			return true
		}
	}
	return false
}

// localPlugin resolves an installed plugin from the cached installed_plugins
// record, with disk manifest fallback handled inside pluginsystem.
func localPlugin(app core.App, pluginID string) (pluginsystem.LocalPlugin, error) {
	plugin, err := pluginsystem.LoadInstalledPlugin(app, "", pluginID)
	if err != nil {
		return pluginsystem.LocalPlugin{}, apis.NewBadRequestError("unknown plugin", err)
	}
	return plugin, nil
}

func stateForCapability(state map[string]any, capability string) map[string]any {
	raw, ok := state[capability]
	if !ok {
		return map[string]any{}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}

func setPluginInstanceStatus(app core.App, instance *core.Record, status string, code string, message string) {
	instance.Set("status", status)
	instance.Set("last_error", map[string]any{
		"code":    code,
		"message": message,
	})
	if err := app.Save(instance); err != nil {
		app.Logger().Warn("failed to update plugin instance status", "instance", instance.Id, "error", err)
	}
}

func setPluginInstanceStatusForError(app core.App, instance *core.Record, err error) {
	update := pluginsystem.InstanceStatusForError(err, time.Now())

	instance.Set("status", update.Status)
	instance.Set("last_error", map[string]any{
		"code":    update.Code,
		"message": update.Message,
	})
	if update.NextRetryAt != nil {
		instance.Set("next_retry_at", *update.NextRetryAt)
	} else {
		instance.Set("next_retry_at", "")
	}
	if saveErr := app.Save(instance); saveErr != nil {
		app.Logger().Warn("failed to update plugin instance status", "instance", instance.Id, "error", saveErr)
	}
}

func applyImportPolicy(item *pluginsystem.TrailImport, config map[string]any) {
	if stringOption(config, "privacy", "original") != "original" {
		item.Privacy = nil
	}
}

func boolOption(config map[string]any, key string, fallback bool) bool {
	value, ok := config[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func stringOption(config map[string]any, key string, fallback string) string {
	value, ok := config[key].(string)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func userDefaultPublic(app core.App, userID string) bool {
	settings, err := app.FindFirstRecordByData("settings", "user", userID)
	if err != nil || settings == nil {
		return false
	}

	privacySettings := struct {
		Trails string `json:"trails"`
	}{}
	if err := settings.UnmarshalJSONField("privacy", &privacySettings); err != nil {
		return false
	}

	return privacySettings.Trails == "public"
}

func recentExternalIDs(app core.App, provider string, userID string) []string {
	refs, err := app.FindRecordsByFilter(
		"trail_external_reference",
		"provider={:provider} && trail.author.user={:user}",
		"-created",
		100,
		0,
		dbx.Params{"provider": provider, "user": userID},
	)
	if err != nil {
		return nil
	}

	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		if externalID := ref.GetString("external_id"); externalID != "" {
			ids = append(ids, externalID)
		}
	}
	return ids
}
