package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	"pocketbase/services/trailmerge"
	"pocketbase/util"
)

const (
	defaultPluginSyncBatchLimit                = 50
	defaultPluginSyncMaxBatches                = 100
	defaultPluginProviderCategoryBackfillLimit = 10
)

var syncCapabilityDescriptors = []syncCapabilityDescriptor{
	{
		OptionKey:      "planned",
		CapabilityName: "list_routes",
		DetailName:     "get_route_detail",
		Version:        "v1",
	},
	{
		OptionKey:      "completed",
		CapabilityName: "list_activities",
		DetailName:     "get_activity_detail",
		Version:        "v1",
	},
}

type syncCapabilityDescriptor struct {
	OptionKey      string
	CapabilityName string
	DetailName     string
	Version        string
}

type pluginSystemListInput struct {
	Instance pluginsystem.InstanceRef `json:"instance"`
	Auth     map[string]any           `json:"auth,omitempty"`
	State    map[string]any           `json:"state,omitempty"`
	Options  map[string]any           `json:"options,omitempty"`
	Limits   pluginSystemSyncLimits   `json:"limits,omitempty"`
}

type pluginSystemSyncLimits struct {
	MaxItems int `json:"maxItems,omitempty"`
}

type pluginSystemListOutput struct {
	Items   []pluginsystem.TrailSummary `json:"items"`
	State   map[string]any              `json:"state,omitempty"`
	HasMore bool                        `json:"hasMore"`
	Error   *pluginsystem.PluginError   `json:"error,omitempty"`
}

type pluginSystemDetailInput struct {
	Instance pluginsystem.InstanceRef  `json:"instance"`
	Auth     map[string]any            `json:"auth,omitempty"`
	Options  map[string]any            `json:"options,omitempty"`
	Summary  pluginsystem.TrailSummary `json:"summary"`
}

type pluginSystemDetailOutput struct {
	Item  pluginsystem.TrailImport  `json:"item"`
	Error *pluginsystem.PluginError `json:"error,omitempty"`
}

type pluginSystemSyncResult struct {
	PluginID string `json:"pluginId"`
	Imported int    `json:"imported"`
	Skipped  int    `json:"skipped"`
}

// PluginSystemSyncConfigured is the cron entrypoint. It refreshes plugin
// metadata, finds enabled instances, skips instances in backoff, and syncs each
// configured import capability.
func PluginSystemSyncConfigured(ctx context.Context, app core.App, client meilisearch.ServiceManager) error {
	app.Logger().Info("plugin sync cron started")
	manager := pluginsystem.NewManager(app, "")
	if err := manager.SyncInstalledPlugins(ctx); err != nil {
		return err
	}
	plugins, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err != nil {
		return err
	}
	app.Logger().Info("plugin sync discovered installed plugins", "count", len(plugins))

	var syncErr error
	for _, plugin := range plugins {
		if !pluginHasAnySyncCapability(plugin) {
			app.Logger().Info("plugin sync skipping plugin without sync capability", "plugin", plugin.Manifest.ID)
			continue
		}
		instances, err := pluginInstances(app, plugin.Manifest.ID)
		if err != nil {
			return err
		}
		app.Logger().Info("plugin sync found enabled instances", "plugin", plugin.Manifest.ID, "count", len(instances))
		for _, instance := range instances {
			if err := ctx.Err(); err != nil {
				return err
			}
			if shouldSkipPluginInstance(instance) {
				app.Logger().Info("plugin sync skipping instance due to retry delay", "plugin", plugin.Manifest.ID, "instance", instance.Id, "retry_not_before", instance.GetString("retry_not_before"))
				continue
			}
			app.Logger().Info("plugin instance sync started", "plugin", plugin.Manifest.ID, "instance", instance.Id)
			result, err := syncPluginInstance(ctx, app, client, plugin, instance)
			if err != nil {
				app.Logger().Warn("plugin instance sync failed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "error", err)
				syncErr = err
				continue
			}
			app.Logger().Info("plugin instance sync completed", "plugin", result.PluginID, "instance", instance.Id, "imported", result.Imported, "skipped", result.Skipped)
		}
	}
	app.Logger().Info("plugin sync cron completed")
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
func syncPluginInstance(ctx context.Context, app core.App, client meilisearch.ServiceManager, plugin pluginsystem.LocalPlugin, instance *core.Record) (*pluginSystemSyncResult, error) {
	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", instance.GetString("user"))
	if err != nil {
		setPluginInstanceStatus(app, instance, "error", "invalid_request", "activitypub actor not found")
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
	config := effectivePluginConfig(app, plugin.Manifest.ID, instance)
	pluginConfig := pluginRuntimeConfig(config)
	hostConfig := pluginHostConfig(config)
	defaultPublic := userDefaultPublic(app, instance.GetString("user"))
	createSummitLog := boolOption(hostConfig, "createSummitLogForCompleted", true)
	runtime, err := pluginsystem.NewRuntimeRegistry().RuntimeFor(plugin)
	if err != nil {
		setPluginInstanceStatusForError(app, instance, err)
		return nil, err
	}
	sessions := &pluginSyncRuntimeSession{
		runtime: runtime,
		plugin:  plugin,
		policy:  pluginInstancePolicy(plugin, config).WithHostAuth(auth),
	}
	if err := sessions.open(ctx); err != nil {
		setPluginInstanceStatusForError(app, instance, err)
		return nil, err
	}
	defer func() {
		_ = sessions.close(context.Background())
	}()

	instance.Set("status", "syncing")
	if err := app.Save(instance); err != nil {
		return nil, err
	}

	result := &pluginSystemSyncResult{PluginID: plugin.Manifest.ID}
	for _, descriptor := range syncCapabilityDescriptors {
		if !boolOption(hostConfig, descriptor.OptionKey, true) {
			app.Logger().Info("plugin sync skipping disabled capability", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", descriptor.CapabilityName, "option", descriptor.OptionKey)
			continue
		}
		if !pluginHasCapability(plugin, descriptor.CapabilityName, descriptor.Version) {
			app.Logger().Info("plugin sync skipping unavailable capability", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", descriptor.CapabilityName, "version", descriptor.Version)
			continue
		}
		if !pluginHasCapability(plugin, descriptor.DetailName, descriptor.Version) {
			app.Logger().Warn("plugin sync skipping list capability because matching detail capability is unavailable", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", descriptor.CapabilityName, "detail_capability", descriptor.DetailName, "version", descriptor.Version)
			continue
		}
		capability, err := pluginCapability(plugin, descriptor.CapabilityName, descriptor.Version)
		if err != nil {
			return nil, err
		}
		detailCapability, err := pluginCapability(plugin, descriptor.DetailName, descriptor.Version)
		if err != nil {
			return nil, err
		}
		app.Logger().Info("plugin capability sync started", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", capability.Name, "version", capability.Version, "export", capability.Export)
		capResult, err := syncPluginCapability(ctx, app, client, sessions, plugin, capability, detailCapability, instance, actor, auth, pluginConfig, hostConfig, defaultPublic, createSummitLog)
		if err != nil {
			setPluginInstanceStatusForError(app, instance, err)
			return nil, err
		}
		app.Logger().Info("plugin capability sync completed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", capability.Name, "imported", capResult.Imported, "skipped", capResult.Skipped)
		result.Imported += capResult.Imported
		result.Skipped += capResult.Skipped
	}

	instance.Set("state", map[string]any{})
	instance.Set("last_sync_at", time.Now())
	instance.Set("last_error", map[string]any{})
	instance.Set("retry_not_before", "")
	instance.Set("status", "configured")
	if err := app.Save(instance); err != nil {
		return nil, err
	}
	return result, nil
}

// shouldSkipPluginInstance applies retry delay from the last sync error.
func shouldSkipPluginInstance(instance *core.Record) bool {
	retryNotBefore := instance.GetDateTime("retry_not_before")
	return !retryNotBefore.IsZero() && retryNotBefore.Time().After(time.Now())
}

type capabilitySyncResult struct {
	Imported int
	Skipped  int
}

type pluginSyncRuntimeSession struct {
	runtime pluginsystem.Runtime
	plugin  pluginsystem.LocalPlugin
	policy  pluginsystem.RequestPolicyContext
	session pluginsystem.RuntimeSession
}

func (s *pluginSyncRuntimeSession) open(ctx context.Context) error {
	session, err := s.runtime.OpenSession(ctx, s.plugin, s.policy)
	if err != nil {
		return err
	}
	s.session = session
	return nil
}

func (s *pluginSyncRuntimeSession) reopen(ctx context.Context) error {
	_ = s.close(context.Background())
	return s.open(ctx)
}

func (s *pluginSyncRuntimeSession) close(ctx context.Context) error {
	if s.session == nil {
		return nil
	}
	err := s.session.Close(ctx)
	s.session = nil
	return err
}

// syncPluginCapability calls one plugin export such as list_routes_v1, imports
// the returned trail items, and carries transient page state only within this
// sync run. The page cursor is intentionally not persisted across runs.
func syncPluginCapability(ctx context.Context, app core.App, client meilisearch.ServiceManager, sessions *pluginSyncRuntimeSession, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, detailCapability pluginsystem.CapabilityManifest, instance *core.Record, actor *core.Record, auth map[string]any, pluginConfig map[string]any, hostConfig map[string]any, defaultPublic bool, createSummitLog bool) (*capabilitySyncResult, error) {
	result := &capabilitySyncResult{}
	state := map[string]any{}
	hasMore := true
	policy := sessions.policy
	providerCategoryBackfillsRemaining := 0
	if hasUsableCategoryMapping(categoryMapping(hostConfig)) {
		providerCategoryBackfillsRemaining = defaultPluginProviderCategoryBackfillLimit
	}
	for batch := 0; hasMore && batch < defaultPluginSyncMaxBatches; batch++ {
		input := pluginSystemListInput{
			Instance: pluginsystem.InstanceRef{
				ID:       instance.Id,
				PluginID: instance.GetString("plugin_id"),
			},
			Auth:    pluginsystem.PluginInputAuth(plugin, auth),
			State:   state,
			Options: pluginConfig,
			Limits:  pluginSystemSyncLimits{MaxItems: defaultPluginSyncBatchLimit},
		}
		inputBytes, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		outputBytes, err := sessions.session.Call(ctx, capability.Export, inputBytes)
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
		app.Logger().Info("plugin capability batch returned items", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", capability.Name, "batch", batch, "items", len(output.Items), "has_more", output.HasMore)

		summaries := output.Items
		externalIDsByProvider := map[string][]string{}
		for i := range summaries {
			if summaries[i].Source.Provider == "" {
				summaries[i].Source.Provider = plugin.Manifest.ID
			}
			if summaries[i].Source.ExternalID == "" {
				continue
			}
			externalIDsByProvider[summaries[i].Source.Provider] = append(externalIDsByProvider[summaries[i].Source.Provider], summaries[i].Source.ExternalID)
		}
		existingIDsByProvider := map[string]map[string]bool{}
		providerCategoryBackfillCandidatesByProvider := map[string]map[string]*core.Record{}
		for provider, externalIDs := range externalIDsByProvider {
			existingIDs, err := util.FindExistingExternalReferenceIDsForUser(app, instance.GetString("user"), provider, externalIDs)
			if err != nil {
				return nil, err
			}
			existingIDsByProvider[provider] = existingIDs
			if providerCategoryBackfillsRemaining > 0 && len(existingIDs) > 0 {
				candidates, err := providerCategoryBackfillCandidatesForSync(app, instance.GetString("user"), provider, externalIDs, providerCategoryBackfillsRemaining)
				if err != nil {
					return nil, err
				}
				providerCategoryBackfillCandidatesByProvider[provider] = candidates
			}
		}

		for _, summary := range summaries {
			if summary.Source.ExternalID == "" {
				continue
			}
			if existingIDsByProvider[summary.Source.Provider][summary.Source.ExternalID] {
				result.Skipped++
				if providerCategoryBackfillsRemaining > 0 {
					ref := providerCategoryBackfillCandidatesByProvider[summary.Source.Provider][summary.Source.ExternalID]
					attempted, err := backfillProviderCategoryDuringSync(ctx, app, sessions, plugin, detailCapability, instance, auth, pluginConfig, summary, ref)
					if err != nil {
						return nil, err
					}
					if attempted {
						providerCategoryBackfillsRemaining--
					}
				}
				continue
			}
			item, err := pluginDetail(ctx, sessions.session, plugin, detailCapability, instance, auth, pluginConfig, summary)
			if err != nil {
				result.Skipped++
				app.Logger().Warn("skipping plugin item after detail fetch failed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", capability.Name, "provider", summary.Source.Provider, "external_id", summary.Source.ExternalID, "error", err)
				if pluginsystem.IsRuntimeSessionFatalError(err) {
					if reopenErr := sessions.reopen(ctx); reopenErr != nil {
						return nil, reopenErr
					}
				}
				continue
			}
			applyHostPolicy(&item, hostConfig)
			imported, err := importer.ImportTrail(ctx, app, item, importer.Options{
				UserID:                      instance.GetString("user"),
				ActorID:                     actor.Id,
				DefaultPublic:               defaultPublic,
				CreateSummitLogForCompleted: createSummitLog,
				CategoryMapping:             categoryMapping(hostConfig),
				Manifest:                    plugin.Manifest,
				Policy:                      policy,
				Auth:                        auth,
			})
			if err != nil {
				return nil, err
			}
			if imported.Created {
				result.Imported++
				app.Logger().Info("imported plugin trail", "provider", item.Source.Provider, "external_id", item.Source.ExternalID, "trail", imported.TrailID)
				if autoMergeEnabled(hostConfig) {
					settings := trailmerge.DefaultPluginAutoMergeSettings()
					settings.Enabled = true
					if err := trailmerge.TryAutoMergeImportedTrail(app, client, ctx, actor, imported.TrailID, settings); err != nil {
						app.Logger().Warn("unable to auto-merge imported plugin trail", "provider", item.Source.Provider, "external_id", item.Source.ExternalID, "trail", imported.TrailID, "error", err)
					}
				}
			}
			if imported.Skipped {
				result.Skipped++
			}
		}

		state = output.State
		if state == nil {
			state = map[string]any{}
		}
		hasMore = output.HasMore
	}
	if hasMore {
		return nil, fmt.Errorf("sync stopped after %d batches", defaultPluginSyncMaxBatches)
	}
	return result, nil
}

func providerCategoryBackfillCandidatesForSync(app core.App, userID string, provider string, externalIDs []string, limit int) (map[string]*core.Record, error) {
	candidates := map[string]*core.Record{}
	if userID == "" || provider == "" || len(externalIDs) == 0 || limit <= 0 {
		return candidates, nil
	}

	params := dbx.Params{
		"user":     userID,
		"provider": provider,
	}
	seenExternalIDs := map[string]bool{}
	idFilters := make([]string, 0, len(externalIDs))
	for _, externalID := range externalIDs {
		if externalID == "" || seenExternalIDs[externalID] {
			continue
		}
		seenExternalIDs[externalID] = true
		paramName := fmt.Sprintf("external_id_%d", len(idFilters))
		params[paramName] = externalID
		idFilters = append(idFilters, "external_id={:"+paramName+"}")
	}
	if len(idFilters) == 0 {
		return candidates, nil
	}

	filter := "user={:user} && provider={:provider} && (" + strings.Join(idFilters, " || ") + ")"
	refs, err := app.FindRecordsByFilter("trail_external_reference", filter, "", len(idFilters), 0, params)
	if err != nil || len(refs) == 0 {
		return candidates, err
	}

	for _, ref := range refs {
		if len(candidates) >= limit {
			break
		}
		if ref.GetString("provider_category") != "" || !ref.GetDateTime("provider_category_checked_at").IsZero() {
			continue
		}
		candidates[ref.GetString("external_id")] = ref
	}
	return candidates, nil
}

func backfillProviderCategoryDuringSync(ctx context.Context, app core.App, sessions *pluginSyncRuntimeSession, plugin pluginsystem.LocalPlugin, detailCapability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, pluginConfig map[string]any, summary pluginsystem.TrailSummary, ref *core.Record) (bool, error) {
	if ref == nil {
		return false, nil
	}

	item, err := pluginDetail(ctx, sessions.session, plugin, detailCapability, instance, auth, pluginConfig, summary)
	if err != nil {
		app.Logger().Warn("skipping provider category backfill after detail fetch failed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "provider", summary.Source.Provider, "external_id", summary.Source.ExternalID, "error", err)
		if pluginsystem.IsRuntimeSessionFatalError(err) {
			if reopenErr := sessions.reopen(ctx); reopenErr != nil {
				return true, reopenErr
			}
		}
		return true, nil
	}

	ref.Set("provider_category", importer.ProviderCategoryFromImport(item))
	ref.Set("provider_category_checked_at", time.Now())
	if err := app.Save(ref); err != nil {
		return false, err
	}
	return true, nil
}

func pluginDetail(ctx context.Context, session pluginsystem.RuntimeSession, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, pluginConfig map[string]any, summary pluginsystem.TrailSummary) (pluginsystem.TrailImport, error) {
	input := pluginSystemDetailInput{
		Instance: pluginsystem.InstanceRef{
			ID:       instance.Id,
			PluginID: instance.GetString("plugin_id"),
		},
		Auth:    pluginsystem.PluginInputAuth(plugin, auth),
		Options: pluginConfig,
		Summary: summary,
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return pluginsystem.TrailImport{}, err
	}
	outputBytes, err := session.Call(ctx, capability.Export, inputBytes)
	if err != nil {
		return pluginsystem.TrailImport{}, err
	}
	var output pluginSystemDetailOutput
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return pluginsystem.TrailImport{}, fmt.Errorf("plugin returned invalid %s output: %w", capability.Export, err)
	}
	if output.Error != nil {
		return pluginsystem.TrailImport{}, pluginsystem.PluginCapabilityError{Err: output.Error}
	}
	return output.Item, nil
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
	if update.RetryNotBefore != nil {
		instance.Set("retry_not_before", *update.RetryNotBefore)
	} else {
		instance.Set("retry_not_before", "")
	}
	if saveErr := app.Save(instance); saveErr != nil {
		app.Logger().Warn("failed to update plugin instance status", "instance", instance.Id, "error", saveErr)
	}
}

func applyHostPolicy(item *pluginsystem.TrailImport, config map[string]any) {
	privacyMode, ok := config["privacy"].(string)
	if !ok || privacyMode == "" {
		privacyMode = "original"
	}
	if privacyMode != "original" {
		item.Privacy = nil
	}
}

func autoMergeEnabled(config map[string]any) bool {
	merge, ok := config["merge"].(map[string]any)
	return ok && boolOption(merge, "available", true) && boolOption(merge, "enabled", false)
}

func boolOption(config map[string]any, key string, fallback bool) bool {
	value, ok := config[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func categoryMapping(config map[string]any) map[string]string {
	raw, ok := config["categoryMapping"].(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]string, len(raw))
	for key, value := range raw {
		category, ok := value.(string)
		if ok {
			result[key] = category
		}
	}
	return result
}

func hasUsableCategoryMapping(mapping map[string]string) bool {
	for _, category := range mapping {
		if strings.TrimSpace(category) != "" {
			return true
		}
	}
	return false
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
