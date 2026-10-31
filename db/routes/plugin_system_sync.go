package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
		if err := syncPluginInstancesOneByOne(ctx, app, client, plugin, instances); err != nil {
			if ctx.Err() != nil {
				return err
			}
			syncErr = err
		}
	}
	app.Logger().Info("plugin sync cron completed")
	return syncErr
}

// syncPluginInstancesOneByOne is the cron's loop over one plugin's instances.
// It returns the last sync error, or the context's error once it is cancelled.
func syncPluginInstancesOneByOne(ctx context.Context, app core.App, client meilisearch.ServiceManager, plugin pluginsystem.LocalPlugin, instances []*core.Record) error {
	var syncErr error
	for _, instance := range instances {
		if err := ctx.Err(); err != nil {
			return err
		}
		result, skipped, err := syncFreshPluginInstance(ctx, app, client, plugin, instance.Id)
		if skipped || errors.Is(err, errPluginInstanceSwitchedOff) {
			app.Logger().Info("plugin sync skipping instance that is switched off, deleted or waiting for its retry time", "plugin", plugin.Manifest.ID, "instance", instance.Id)
			continue
		}
		if err != nil {
			app.Logger().Warn("plugin instance sync failed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "error", err)
			syncErr = err
			continue
		}
		app.Logger().Info("plugin instance sync completed", "plugin", result.PluginID, "instance", instance.Id, "imported", result.Imported, "skipped", result.Skipped)
	}
	return syncErr
}

// ResetInterruptedPluginSyncs clears the "syncing" status a restart left on
// instances whose sync it cut off, so they do not show as syncing until their
// next sync.
func ResetInterruptedPluginSyncs(app core.App) error {
	_, err := app.DB().NewQuery("UPDATE plugin_instances SET status = CASE WHEN enabled THEN 'configured' ELSE 'disabled' END WHERE status = 'syncing'").Execute()
	return err
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

// runPluginInstanceSync is syncPluginInstance. Tests replace it to check the
// status handling around a sync without a plugin runtime.
var runPluginInstanceSync = syncPluginInstance

// pluginRuntimeFor picks the runtime for a plugin. Tests replace it to run
// syncPluginInstance without a worker process.
var pluginRuntimeFor = func(plugin pluginsystem.LocalPlugin) (pluginsystem.Runtime, error) {
	return pluginsystem.NewRuntimeRegistry().RuntimeFor(plugin)
}

// syncFreshPluginInstance re-reads the instance before syncing it, since the
// caller's copy may predate a change the user or another sync made. skipped is
// true when the instance is gone, switched off or waiting for its retry time.
// A panic during the sync becomes an error on the instance instead of taking
// the server down.
func syncFreshPluginInstance(ctx context.Context, app core.App, client meilisearch.ServiceManager, plugin pluginsystem.LocalPlugin, instanceID string) (result *pluginSystemSyncResult, skipped bool, err error) {
	instance, err := app.FindRecordById("plugin_instances", instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !instance.GetBool("enabled") || shouldSkipPluginInstance(instance) {
		return nil, true, nil
	}
	app.Logger().Info("plugin instance sync started", "plugin", plugin.Manifest.ID, "instance", instanceID)
	defer func() {
		// The panic value and stack can hold plugin data, so neither is logged.
		if recover() != nil {
			app.Logger().Error("plugin instance sync panicked", "plugin", plugin.Manifest.ID, "instance", instanceID)
			setPluginInstanceStatus(app, instance, "error", "internal_error", "the sync stopped on an internal error")
			result, skipped, err = nil, false, errors.New("plugin sync panicked")
		}
	}()
	result, err = runPluginInstanceSync(ctx, app, client, plugin, instance)
	return result, false, err
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
	runtime, err := pluginRuntimeFor(plugin)
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

	if err := savePluginInstanceSyncState(app, instance, map[string]any{"status": "syncing"}); err != nil {
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
		if errors.Is(err, errPluginInstanceSwitchedOff) {
			// The user's own change set the status; leave it as it is.
			return nil, err
		}
		if err != nil {
			setPluginInstanceStatusForError(app, instance, err)
			return nil, err
		}
		app.Logger().Info("plugin capability sync completed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "capability", capability.Name, "imported", capResult.Imported, "skipped", capResult.Skipped)
		result.Imported += capResult.Imported
		result.Skipped += capResult.Skipped
	}

	if err := savePluginInstanceSyncState(app, instance, map[string]any{
		"state":            map[string]any{},
		"last_sync_at":     time.Now(),
		"last_error":       map[string]any{},
		"retry_not_before": "",
		"status":           "configured",
	}); err != nil {
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
	limits := pluginSystemSyncLimits{MaxItems: defaultPluginSyncBatchLimit}
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
			Limits:  limits,
		}
		inputBytes, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		outputBytes, err := sessions.session.Call(ctx, capability.Export, inputBytes)
		if err != nil && pluginsystem.IsRuntimeSessionFatalError(err) && ctx.Err() == nil {
			// The worker ends a session after WANDERER_PLUGIN_WORKER_SESSION_TIMEOUT,
			// which a long sync can outlast. Reopen it once, as for detail calls.
			if reopenErr := sessions.reopen(ctx); reopenErr != nil {
				return nil, reopenErr
			}
			outputBytes, err = sessions.session.Call(ctx, capability.Export, inputBytes)
		}
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
		if len(output.Items) > limits.MaxItems {
			return nil, pluginsystem.SyncItemLimitError{Returned: len(output.Items), MaxItems: limits.MaxItems}
		}
		if pluginInstanceSwitchedOff(app, instance.Id) {
			return nil, errPluginInstanceSwitchedOff
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
			// A batch can take minutes, so check again before each item's calls.
			if pluginInstanceSwitchedOff(app, instance.Id) {
				return nil, errPluginInstanceSwitchedOff
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

	refs, err := util.FindTrailExternalReferencesForUser(app, userID, provider, externalIDs)
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

var errPluginInstanceSwitchedOff = errors.New("plugin instance was switched off or deleted during the sync")

// pluginInstanceSwitchedOff reports whether the user switched the instance off
// or deleted it since the sync started. A failed lookup for any other reason
// lets the sync go on.
func pluginInstanceSwitchedOff(app core.App, instanceID string) bool {
	fresh, err := app.FindRecordById("plugin_instances", instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	return err == nil && !fresh.GetBool("enabled")
}

// savePluginInstanceSyncState writes the fields a sync owns onto a fresh copy
// of the instance. A sync can run for many minutes, and saving the copy it
// started with would write back every other field too, undoing changes the
// user made in the meantime, such as switching the plugin off or disconnecting
// it. When the user has switched the plugin off, their status is kept.
func savePluginInstanceSyncState(app core.App, instance *core.Record, fields map[string]any) error {
	fresh, err := app.FindRecordById("plugin_instances", instance.Id)
	if err != nil {
		return err
	}
	for key, value := range fields {
		if key == "status" && !fresh.GetBool("enabled") {
			continue
		}
		fresh.Set(key, value)
	}
	return app.Save(fresh)
}

func setPluginInstanceStatus(app core.App, instance *core.Record, status string, code string, message string) {
	err := savePluginInstanceSyncState(app, instance, map[string]any{
		"status":     status,
		"last_error": map[string]any{"code": code, "message": message},
	})
	if err != nil {
		app.Logger().Warn("failed to update plugin instance status", "instance", instance.Id, "error", err)
	}
}

func setPluginInstanceStatusForError(app core.App, instance *core.Record, err error) {
	update := pluginsystem.InstanceStatusForError(err, time.Now())
	fields := map[string]any{
		"status":           update.Status,
		"last_error":       map[string]any{"code": update.Code, "message": update.Message},
		"retry_not_before": "",
	}
	if update.RetryNotBefore != nil {
		fields["retry_not_before"] = *update.RetryNotBefore
	}
	if saveErr := savePluginInstanceSyncState(app, instance, fields); saveErr != nil {
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

func categoryMapping(config map[string]any) map[string]importer.CategoryMappingValue {
	raw, ok := config["categoryMapping"].(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string]importer.CategoryMappingValue, len(raw))
	for key, value := range raw {
		switch typed := value.(type) {
		case string:
			result[key] = importer.CategoryMappingValue{Category: strings.TrimSpace(typed)}
		case map[string]any:
			category, _ := typed["category"].(string)
			subcategory, _ := typed["subcategory"].(string)
			result[key] = importer.CategoryMappingValue{
				Category:    strings.TrimSpace(category),
				Subcategory: strings.TrimSpace(subcategory),
			}
		}
	}
	return result
}

func hasUsableCategoryMapping(mapping map[string]importer.CategoryMappingValue) bool {
	for _, target := range mapping {
		if strings.TrimSpace(target.Category) != "" || strings.TrimSpace(target.Subcategory) != "" {
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
