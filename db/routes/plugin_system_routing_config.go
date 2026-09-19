package routes

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	routingScopeBuiltin           = "builtin"
	routingScopeAdmin             = "admin"
	routingScopeUser              = "user"
	routingProfileContentMaxBytes = 256 * 1024
	routingConfigMaxBytes         = 64 * 1024
	routingConfigMaxDepth         = 8
	routingConfigMaxEntries       = 256
	routingPreferenceMaxEntries   = 64
	routingConfigMaxKeyBytes      = 128
	routingConfigMaxStringBytes   = 4096
)

var routingFeatureKeys = map[string]struct{}{
	"routing":                {},
	"navigation":             {},
	"standardControls":       {},
	"nativeAdvancedControls": {},
	"profileUpload":          {},
	"parallelRouting":        {},
	"variants":               {},
	"routeCandidates":        {},
}

var routingSettingKeys = map[string]struct{}{
	"primaryRoutePluginId": {},
	"elevationPluginId":    {},
	"maneuverPluginId":     {},
	"defaultVariantCount":  {},
	"defaultRoutingMode":   {},
	"defaultPreferences":   {},
	"exposedFeatures":      {},
}

var routingEffectiveControlsRateLimiter = util.NewRateLimiter(60, time.Minute)
var routingInvalidPersistedPreferencesWarning sync.Once

type routingSettings struct {
	PrimaryRoutePluginID string          `json:"primaryRoutePluginId,omitempty"`
	ElevationPluginID    string          `json:"elevationPluginId,omitempty"`
	ManeuverPluginID     string          `json:"maneuverPluginId,omitempty"`
	DefaultVariantCount  int             `json:"defaultVariantCount,omitempty"`
	DefaultRoutingMode   string          `json:"defaultRoutingMode,omitempty"`
	DefaultPreferences   map[string]any  `json:"defaultPreferences,omitempty"`
	ExposedFeatures      map[string]bool `json:"exposedFeatures,omitempty"`
}

type routingProfileMappingView struct {
	ID               string         `json:"id,omitempty"`
	Scope            string         `json:"scope"`
	Category         string         `json:"category"`
	Subcategory      string         `json:"subcategory,omitempty"`
	PluginID         string         `json:"pluginId"`
	InstanceID       string         `json:"instanceId,omitempty"`
	NativeProfileKey string         `json:"nativeProfileKey,omitempty"`
	ProfileID        string         `json:"profileId,omitempty"`
	Preferences      map[string]any `json:"preferences,omitempty"`
	NativeConfig     map[string]any `json:"nativeConfig,omitempty"`
}

type routingProfileView struct {
	ID            string         `json:"id,omitempty"`
	Scope         string         `json:"scope"`
	PluginID      string         `json:"pluginId"`
	Key           string         `json:"key"`
	Name          string         `json:"name"`
	Kind          string         `json:"kind"`
	Mode          string         `json:"mode"`
	Source        string         `json:"source"`
	ContentBase64 string         `json:"contentBase64,omitempty"`
	ContentType   string         `json:"contentType,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	NativeConfig  map[string]any `json:"nativeConfig,omitempty"`
	Enabled       bool           `json:"enabled"`
}

type routingEngineView struct {
	PluginID      string         `json:"pluginId"`
	InstanceID    string         `json:"instanceId"`
	Name          string         `json:"name"`
	Enabled       bool           `json:"enabled"`
	Roles         []string       `json:"roles,omitempty"`
	Modes         []string       `json:"modes,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	discoveryRank int
}

type routingEngineSelection struct {
	PluginID   string `json:"pluginId"`
	InstanceID string `json:"instanceId,omitempty"`
}

type routingEffectiveControlsInput struct {
	Category    string `json:"category"`
	Subcategory string `json:"subcategory,omitempty"`
	Routing     struct {
		Mode    string                   `json:"mode,omitempty"`
		Engines []routingEngineSelection `json:"engines,omitempty"`
	} `json:"routing"`
}

type routingEffectiveControlsOutput struct {
	Category                    string                      `json:"category"`
	Subcategory                 string                      `json:"subcategory,omitempty"`
	Mode                        string                      `json:"mode,omitempty"`
	ProfileRevisions            map[string]string           `json:"profileRevisions,omitempty"`
	ProfileUploadRequired       map[string]bool             `json:"profileUploadRequired,omitempty"`
	ProfilePreparationSupported map[string]bool             `json:"profilePreparationSupported,omitempty"`
	Controls                    []routingControlView        `json:"controls"`
	NativeControlGroups         []routingNativeControlGroup `json:"nativeControlGroups,omitempty"`
	HiddenControls              []routingHiddenControl      `json:"hiddenControls,omitempty"`
	Warnings                    []string                    `json:"warnings,omitempty"`
}

type routingNativeControlsInput struct {
	PluginID         string         `json:"pluginId"`
	InstanceID       string         `json:"instanceId,omitempty"`
	ProfileID        string         `json:"profileId,omitempty"`
	NativeProfileKey string         `json:"nativeProfileKey,omitempty"`
	NativeConfig     map[string]any `json:"nativeConfig,omitempty"`
}

type routingNativeControlsOutput struct {
	PluginID   string                      `json:"pluginId"`
	InstanceID string                      `json:"instanceId,omitempty"`
	ProfileID  string                      `json:"profileId,omitempty"`
	Groups     []routingNativeControlGroup `json:"groups"`
	Warnings   []string                    `json:"warnings,omitempty"`
}

type routingNativeControlGroup struct {
	Key      string               `json:"key"`
	Label    string               `json:"label"`
	Labels   map[string]string    `json:"labels,omitempty"`
	Controls []routingControlView `json:"controls"`
}

type routingControlOption struct {
	Value  string            `json:"value"`
	Label  string            `json:"label,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
}

type routingControlView struct {
	Key        string                 `json:"key"`
	Label      string                 `json:"label,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Unit       string                 `json:"unit,omitempty"`
	Type       string                 `json:"type"`
	UI         string                 `json:"ui,omitempty"`
	ValueType  string                 `json:"valueType,omitempty"`
	Min        *float64               `json:"min,omitempty"`
	Max        *float64               `json:"max,omitempty"`
	Step       *float64               `json:"step,omitempty"`
	Default    any                    `json:"default,omitempty"`
	Current    any                    `json:"current,omitempty"`
	Support    string                 `json:"support,omitempty"`
	Comparable bool                   `json:"comparable,omitempty"`
	Target     string                 `json:"target,omitempty"`
	Path       []string               `json:"path,omitempty"`
	Options    []routingControlOption `json:"options,omitempty"`
}

type routingHiddenControl struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

func PluginSystemRoutingSettingsGet(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	settings, err := ResolveRoutingSettings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, settings)
}

func PluginSystemRoutingSettingsPatch(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var submitted map[string]any
	if err := e.BindBody(&submitted); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if routingEnableRequested(submitted) {
		available, err := enabledRoutingRoutePluginAvailable(e.App, e.Auth.Id)
		if err != nil {
			return err
		}
		if !available {
			return routingJSONError(
				e,
				routingErrorFromCode("routing_plugin_required", "an enabled routing plugin is required"),
				nil,
			)
		}
	}
	if err := patchRoutingSettingsConfig(e.App, routingScopeUser, e.Auth.Id, submitted); err != nil {
		return err
	}
	settings, err := ResolveRoutingSettings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, settings)
}

func routingEnableRequested(submitted map[string]any) bool {
	features, ok := submitted["exposedFeatures"].(map[string]any)
	if !ok {
		return false
	}
	enabled, _ := features["routing"].(bool)
	return enabled
}

func enabledRoutingRoutePluginAvailable(app core.App, userID string) (bool, error) {
	return enabledRoutingCapabilityAvailable(app, userID, "route")
}

func enabledRoutingCapabilityAvailable(app core.App, userID string, capabilityName string) (bool, error) {
	plugins, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err != nil {
		return false, err
	}
	instances, err := app.FindRecordsByFilter(
		"plugin_instances",
		"user={:user} && enabled=true",
		"",
		-1,
		0,
		dbx.Params{"user": userID},
	)
	if err != nil {
		return false, err
	}
	enabledPluginIDs := make(map[string]struct{}, len(instances))
	for _, instance := range instances {
		enabledPluginIDs[instance.GetString("plugin_id")] = struct{}{}
	}
	for _, plugin := range plugins {
		if plugin.Manifest.Type != pluginsystem.PluginTypeRouting {
			continue
		}
		if _, enabled := enabledPluginIDs[plugin.Manifest.ID]; !enabled {
			continue
		}
		if routingManifestHasCapability(plugin.Manifest, capabilityName, "v1") {
			return true, nil
		}
	}
	return false, nil
}

func PluginSystemRoutingAdminSettingsGet(e *core.RequestEvent) error {
	if !e.HasSuperuserAuth() {
		return apis.NewUnauthorizedError("superuser authentication required", nil)
	}
	settings, err := routingSettingsForScope(e.App, routingScopeAdmin, "")
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, settings)
}

func PluginSystemRoutingAdminSettingsPatch(e *core.RequestEvent) error {
	if !e.HasSuperuserAuth() {
		return apis.NewUnauthorizedError("superuser authentication required", nil)
	}
	var submitted map[string]any
	if err := e.BindBody(&submitted); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if err := patchRoutingSettingsConfig(e.App, routingScopeAdmin, "", submitted); err != nil {
		return err
	}
	settings, err := routingSettingsForScope(e.App, routingScopeAdmin, "")
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, settings)
}

func PluginSystemRoutingMappingsGet(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	mappings, err := ResolveRoutingProfileMappings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]any{"items": mappings})
}

func PluginSystemRoutingMappingsPut(e *core.RequestEvent) error {
	var input routingProfileMappingView
	if err := e.BindBody(&input); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	scope, userID, err := routingWritableScope(e, input.Scope)
	if err != nil {
		return err
	}
	if err := validateRoutingProfileMappingInput(input); err != nil {
		return err
	}
	collection, err := e.App.FindCollectionByNameOrId("routing_profile_mappings")
	if err != nil {
		return err
	}
	record, err := routingProfileMappingUpsertRecord(e.App, collection, scope, userID, input)
	if err != nil {
		return err
	}
	setRoutingProfileMappingRecord(record, input)
	if err := e.App.Save(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, routingProfileMappingFromRecord(record))
}

func PluginSystemRoutingMappingsPatch(e *core.RequestEvent) error {
	record, err := e.App.FindRecordById("routing_profile_mappings", e.Request.PathValue("id"))
	if err != nil {
		return apis.NewNotFoundError("routing profile mapping not found", err)
	}
	if err := authorizeRoutingScopedRecordWrite(e, record); err != nil {
		return err
	}
	var input routingProfileMappingView
	if err := e.BindBody(&input); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if err := validateRoutingProfileMappingInput(input); err != nil {
		return err
	}
	setRoutingProfileMappingRecord(record, input)
	if err := e.App.Save(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, routingProfileMappingFromRecord(record))
}

func PluginSystemRoutingProfilesGet(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	profiles, err := RoutingProfiles(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]any{"profiles": profiles})
}

func PluginSystemRoutingProfilesPut(e *core.RequestEvent) error {
	var input routingProfileView
	if err := e.BindBody(&input); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	scope, userID, err := routingWritableScope(e, input.Scope)
	if err != nil {
		return err
	}
	if err := validateRoutingProfileInput(input); err != nil {
		return err
	}
	if input.Kind == "custom_file" {
		if err := routingEffectiveControlsRateLimiter.CheckRateLimit(userID, "profile-introspection"); err != nil {
			return e.TooManyRequestsError("too many routing profile introspections", err)
		}
		if err := enrichCustomRoutingProfileContext(e.Request.Context(), e.App, userID, &input); err != nil {
			return routingJSONError(e, err, nil)
		}
	}
	collection, err := e.App.FindCollectionByNameOrId("routing_profiles")
	if err != nil {
		return err
	}
	record, err := routingProfileUpsertRecord(e.App, collection, scope, userID, input)
	if err != nil {
		return err
	}
	if err := setRoutingProfileRecord(record, input); err != nil {
		return err
	}
	if err := e.App.Save(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, routingProfileFromRecord(record))
}

func PluginSystemRoutingProfilesPatch(e *core.RequestEvent) error {
	record, err := e.App.FindRecordById("routing_profiles", e.Request.PathValue("id"))
	if err != nil {
		return apis.NewNotFoundError("routing profile not found", err)
	}
	if err := authorizeRoutingScopedRecordWrite(e, record); err != nil {
		return err
	}
	var input routingProfileView
	if err := e.BindBody(&input); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if err := validateRoutingProfileInput(input); err != nil {
		return err
	}
	if input.Kind == "custom_file" {
		if err := routingEffectiveControlsRateLimiter.CheckRateLimit(e.Auth.Id, "profile-introspection"); err != nil {
			return e.TooManyRequestsError("too many routing profile introspections", err)
		}
		if err := enrichCustomRoutingProfileContext(e.Request.Context(), e.App, e.Auth.Id, &input); err != nil {
			return routingJSONError(e, err, nil)
		}
	}
	if err := setRoutingProfileRecord(record, input); err != nil {
		return err
	}
	if err := e.App.Save(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, routingProfileFromRecord(record))
}

func PluginSystemRoutingProfilesDelete(e *core.RequestEvent) error {
	record, err := e.App.FindRecordById("routing_profiles", e.Request.PathValue("id"))
	if err != nil {
		return apis.NewNotFoundError("routing profile not found", err)
	}
	if err := authorizeRoutingScopedRecordWrite(e, record); err != nil {
		return err
	}
	inUse, err := routingProfileInUse(e.App, record.Id)
	if err != nil {
		return err
	}
	if inUse {
		return apis.NewBadRequestError("routing profile is in use", nil)
	}
	if err := e.App.Delete(record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]any{"id": record.Id})
}

func PluginSystemRoutingEnginesGet(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	engines, err := routingEngines(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]any{"engines": engines})
}

func PluginSystemRoutingEffectiveControls(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var input routingEffectiveControlsInput
	if err := e.BindBody(&input); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if err := routingEffectiveControlsRateLimiter.CheckRateLimit(e.Auth.Id, "effective-controls"); err != nil {
		return e.TooManyRequestsError("too many routing effective-control requests", err)
	}
	engines, validationErr := normalizeRoutingEngineSelections(input.Routing.Engines, routingMaxEngines)
	if validationErr != nil {
		return routingJSONError(e, validationErr, nil)
	}
	input.Routing.Engines = engines
	output, err := ResolveRoutingEffectiveControlsContext(e.Request.Context(), e.App, e.Auth.Id, input)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, output)
}

func normalizeRoutingEngineSelections(selections []routingEngineSelection, maximum int) ([]routingEngineSelection, *routingError) {
	if len(selections) > maximum {
		return nil, routingErrorFromCode("engine_limit_exceeded", "too many routing engines requested")
	}
	result := make([]routingEngineSelection, 0, len(selections))
	seen := make(map[string]bool, len(selections))
	for _, selection := range selections {
		selection.PluginID = strings.TrimSpace(selection.PluginID)
		selection.InstanceID = strings.TrimSpace(selection.InstanceID)
		if selection.PluginID == "" {
			continue
		}
		key := selection.PluginID + ":" + selection.InstanceID
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, selection)
	}
	return result, nil
}

func PluginSystemRoutingNativeControls(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var input routingNativeControlsInput
	if err := e.BindBody(&input); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if err := routingEffectiveControlsRateLimiter.CheckRateLimit(e.Auth.Id, "native-controls"); err != nil {
		return e.TooManyRequestsError("too many routing native-control requests", err)
	}
	if err := validateRoutingConfig(input.NativeConfig, "nativeConfig"); err != nil {
		return apis.NewBadRequestError("invalid nativeConfig", err)
	}
	output, err := ResolveRoutingNativeControlsContext(e.Request.Context(), e.App, e.Auth.Id, input)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, output)
}

func InitRoutingDefaults(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("routing_settings"); err != nil {
		return nil
	}
	return seedRoutingBuiltinSettings(app)
}

func ResolveRoutingSettings(app core.App, userID string) (routingSettings, error) {
	layers, err := pluginsystem.ResolveRoutingSettingsLayers(app, userID)
	if err != nil {
		return routingSettings{}, err
	}
	discardedPreferences := sanitizePersistedRoutingDefaultPreferences(layers.Builtin)
	discardedPreferences += sanitizePersistedRoutingDefaultPreferences(layers.Admin)
	discardedPreferences += sanitizePersistedRoutingDefaultPreferences(layers.User)
	config := pluginsystem.MergeRoutingSettingsLayers(layers)
	discardedPreferences += limitResolvedRoutingDefaultPreferences(config, layers)
	if discardedPreferences > 0 {
		warnInvalidPersistedRoutingDefaultPreferences(app, "effective settings", userID, discardedPreferences)
	}
	resolved := routingSettingsFromConfig(config)
	if resolved.DefaultVariantCount < 1 {
		resolved.DefaultVariantCount = 1
	}
	if resolved.DefaultVariantCount > routingMaxVariants {
		resolved.DefaultVariantCount = routingMaxVariants
	}
	// Built-in settings deliberately contain no provider IDs. Complete only
	// missing selections from plugins that are both available and enabled for
	// this user; explicit admin and user selections have already won the scoped
	// merge above and are therefore preserved.
	if userID != "" && (resolved.PrimaryRoutePluginID == "" || resolved.ElevationPluginID == "" || resolved.ManeuverPluginID == "") {
		// Discovery is optional defaulting. A temporarily unavailable plugin
		// registry must not make otherwise valid explicit settings unreadable.
		if engines, discoveryErr := routingEngines(app, userID); discoveryErr == nil {
			applyRoutingEngineDefaults(&resolved, engines)
		}
	}
	return resolved, nil
}

func ResolveRoutingProfileMappings(app core.App, userID string) ([]routingProfileMappingView, error) {
	plugins, err := loadRoutingPluginsWithFallback(app)
	if err != nil {
		return nil, err
	}
	return ResolveRoutingProfileMappingsFromPlugins(app, userID, plugins)
}

func ResolveRoutingProfileMappingsFromPlugins(app core.App, userID string, plugins []pluginsystem.LocalPlugin) ([]routingProfileMappingView, error) {
	defaults := builtinRoutingProfileMappingsFromPlugins(plugins)
	records, err := scopedRoutingRecords(app, "routing_profile_mappings", userID)
	if err != nil {
		return nil, err
	}
	byKey := map[string]routingProfileMappingView{}
	for _, mapping := range defaults {
		byKey[routingProfileMappingResolutionKey(mapping)] = mapping
	}
	for _, record := range records {
		mapping := routingProfileMappingFromRecord(record)
		byKey[routingProfileMappingResolutionKey(mapping)] = mapping
	}
	mappings := make([]routingProfileMappingView, 0, len(byKey))
	for _, mapping := range byKey {
		mappings = append(mappings, mapping)
	}
	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].PluginID == mappings[j].PluginID {
			if mappings[i].InstanceID == mappings[j].InstanceID {
				if mappings[i].Category == mappings[j].Category {
					return mappings[i].Subcategory < mappings[j].Subcategory
				}
				return mappings[i].Category < mappings[j].Category
			}
			return mappings[i].InstanceID < mappings[j].InstanceID
		}
		return mappings[i].PluginID < mappings[j].PluginID
	})
	return mappings, nil
}

func ResolveRoutingProfileMapping(app core.App, userID string, pluginID string, instanceID string, category string, subcategory string) (*routingProfileMappingView, error) {
	mappings, err := ResolveRoutingProfileMappings(app, userID)
	if err != nil {
		return nil, err
	}
	return resolveRoutingProfileMappingFromMappings(mappings, pluginID, instanceID, category, subcategory), nil
}

func resolveRoutingProfileMappingFromMappings(mappings []routingProfileMappingView, pluginID string, instanceID string, category string, subcategory string) *routingProfileMappingView {
	matches := []routingProfileMappingView{}
	for _, mapping := range mappings {
		if mapping.PluginID != pluginID || mapping.Category != category {
			continue
		}
		if mapping.InstanceID != "" && mapping.InstanceID != instanceID {
			continue
		}
		if mapping.Subcategory != "" && mapping.Subcategory != subcategory {
			continue
		}
		matches = append(matches, mapping)
	}
	return selectRoutingProfileMapping(matches)
}

func applyRoutingCategoryMapping(app core.App, userID string, pluginID string, instanceID string, request *pluginRoutingRouteRequest) error {
	return applyRoutingCategoryMappingContext(context.Background(), app, userID, pluginID, instanceID, request, nil)
}

func applyRoutingCategoryMappingContext(ctx context.Context, app core.App, userID string, pluginID string, instanceID string, request *pluginRoutingRouteRequest, settings *routingSettings) error {
	request.Profile.PluginID = pluginID
	clientNativeConfig := cloneRoutingMap(request.Profile.NativeConfig)
	resolvedSettings := routingSettings{}
	if settings == nil {
		var err error
		resolvedSettings, err = ResolveRoutingSettings(app, userID)
		if err != nil {
			return err
		}
	} else {
		resolvedSettings = *settings
	}
	var err error
	var mapping *routingProfileMappingView
	if request.Category != "" {
		mapping, err = ResolveRoutingProfileMapping(app, userID, pluginID, instanceID, request.Category, request.Subcategory)
		if err != nil {
			return err
		}
		if mapping == nil {
			return &routingError{
				Code:       "mapping_missing",
				Message:    "no routing profile mapping exists for the selected category",
				HTTPStatus: http.StatusUnprocessableEntity,
			}
		}
	}
	if mapping != nil && mapping.NativeProfileKey != "" {
		plugin, err := localPlugin(app, pluginID)
		if err != nil {
			return err
		}
		profile := routingManifestNativeProfile(plugin.Manifest, mapping.NativeProfileKey)
		if len(profile) == 0 {
			return &routingError{Code: "unsupported_profile", Message: "mapped routing profile does not exist", HTTPStatus: http.StatusUnprocessableEntity}
		}
		profileMode := stringValue(profile["mode"])
		request.Mode = profileMode
		request.Profile.Key = mapping.NativeProfileKey
		request.Profile.Kind = "builtin"
		request.Profile.Mode = profileMode
		request.Profile.Metadata = profile
		allowedNativeConfig, err := allowlistedRoutingNativeConfig(mergeRoutingConfigLayers(
			mapValue(profile["nativeConfig"]),
			mapping.NativeConfig,
			clientNativeConfig,
		), anySlice(profile["nativeControlGroups"]))
		if err != nil {
			return apis.NewBadRequestError("invalid profile.nativeConfig", err)
		}
		request.Profile.NativeConfig = allowedNativeConfig
	}
	if mapping != nil && mapping.ProfileID != "" {
		profile, err := app.FindRecordById("routing_profiles", mapping.ProfileID)
		if err != nil {
			return err
		}
		if profile.GetString("plugin_id") != pluginID {
			return apis.NewBadRequestError("routing profile belongs to a different plugin", map[string]any{
				"profileId": mapping.ProfileID,
				"pluginId":  pluginID,
			})
		}
		if !profile.GetBool("enabled") {
			return apis.NewBadRequestError("routing profile is disabled", map[string]any{
				"profileId": mapping.ProfileID,
			})
		}
		request.Profile.ID = profile.Id
		request.Profile.Key = profile.GetString("key")
		request.Profile.Kind = profile.GetString("kind")
		profileMode := profile.GetString("mode")
		request.Mode = profileMode
		request.Profile.Mode = profileMode
		request.Profile.ContentBase64 = profile.GetString("content_base64")
		request.Profile.ContentType = profile.GetString("content_type")
		request.Profile.Metadata = pluginsystem.JSONMapFromRecord(profile, "metadata")
		allowedNativeConfig, err := allowlistedRoutingNativeConfig(mergeRoutingConfigLayers(
			pluginsystem.JSONMapFromRecord(profile, "native_config"),
			mapping.NativeConfig,
			clientNativeConfig,
		), anySlice(request.Profile.Metadata["nativeControlGroups"]))
		if err != nil {
			return apis.NewBadRequestError("invalid profile.nativeConfig", err)
		}
		request.Profile.NativeConfig = allowedNativeConfig
	}
	if mapping == nil {
		request.Profile.NativeConfig = nil
	}
	preferences := map[string]any{}
	mergeRoutingPreferenceLayer(preferences, routingPreferencesApplicableToMode(resolvedSettings.DefaultPreferences, request.Mode))
	if mapping != nil {
		mergeRoutingPreferenceLayer(preferences, mapping.Preferences)
	}
	mergeRoutingPreferenceLayer(preferences, request.Preferences)
	if mapping != nil {
		supported, restricted, err := routingProfileMappingSupportedPreferencesContext(ctx, app, userID, *mapping)
		if err != nil {
			return err
		}
		if restricted {
			restrictRoutingPreferences(preferences, supported)
		}
	}
	clampRoutingPreferencesToMode(preferences, request.Mode)
	request.Preferences = preferences
	return nil
}

func mergeRoutingConfigLayers(layers ...map[string]any) map[string]any {
	merged := map[string]any{}
	for _, layer := range layers {
		if len(layer) > 0 {
			pluginsystem.DeepMergeConfig(merged, cloneRoutingMap(layer))
		}
	}
	return merged
}

func routingProfileMappingResolutionKey(mapping routingProfileMappingView) string {
	return mapping.PluginID + "|" + mapping.InstanceID + "|" + mapping.Category + "|" + mapping.Subcategory
}

func routingProfileMappingPrecedenceLess(left routingProfileMappingView, right routingProfileMappingView) bool {
	leftScope := routingScopeRank(left.Scope)
	rightScope := routingScopeRank(right.Scope)
	if leftScope != rightScope {
		return leftScope < rightScope
	}
	if left.PluginID != right.PluginID {
		return left.PluginID < right.PluginID
	}
	if left.Category != right.Category {
		return left.Category < right.Category
	}
	if left.InstanceID != right.InstanceID {
		return left.InstanceID == "" && right.InstanceID != ""
	}
	return left.Subcategory == "" && right.Subcategory != ""
}

func selectRoutingProfileMapping(matches []routingProfileMappingView) *routingProfileMappingView {
	sort.SliceStable(matches, func(i, j int) bool {
		return routingProfileMappingPrecedenceLess(matches[i], matches[j])
	})
	if len(matches) == 0 {
		return nil
	}
	selected := matches[len(matches)-1]
	return &selected
}

func mergeRoutingPreferenceLayer(base map[string]any, layer map[string]any) {
	if len(layer) == 0 {
		return
	}
	pluginsystem.DeepMergeConfig(base, cloneRoutingMap(layer))
}

func RoutingProfiles(app core.App, userID string) ([]routingProfileView, error) {
	profiles := routingDiscoveryProfiles(app)
	records, err := scopedRoutingRecords(app, "routing_profiles", userID)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		profiles = append(profiles, routingProfileFromRecord(record))
	}
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].PluginID == profiles[j].PluginID {
			return profiles[i].Key < profiles[j].Key
		}
		return profiles[i].PluginID < profiles[j].PluginID
	})
	return profiles, nil
}

func ResolveRoutingEffectiveControls(app core.App, userID string, input routingEffectiveControlsInput) (routingEffectiveControlsOutput, error) {
	return ResolveRoutingEffectiveControlsContext(context.Background(), app, userID, input)
}

func ResolveRoutingEffectiveControlsContext(ctx context.Context, app core.App, userID string, input routingEffectiveControlsInput) (routingEffectiveControlsOutput, error) {
	output := routingEffectiveControlsOutput{
		Category:                    input.Category,
		Subcategory:                 input.Subcategory,
		Controls:                    []routingControlView{},
		ProfileRevisions:            map[string]string{},
		ProfileUploadRequired:       map[string]bool{},
		ProfilePreparationSupported: map[string]bool{},
	}
	preferences := map[string]any{}
	settings, err := ResolveRoutingSettings(app, userID)
	if err != nil {
		return output, err
	}
	if len(input.Routing.Engines) == 0 && settings.PrimaryRoutePluginID != "" {
		input.Routing.Engines = []routingEngineSelection{{PluginID: settings.PrimaryRoutePluginID}}
	}
	engines, validationErr := normalizeRoutingEngineSelections(input.Routing.Engines, routingMaxEngines)
	if validationErr != nil {
		return output, validationErr
	}
	input.Routing.Engines = engines
	installed, err := loadRoutingPluginsWithFallback(app)
	if err != nil {
		return output, err
	}
	pluginByID := make(map[string]pluginsystem.LocalPlugin, len(installed))
	for _, plugin := range installed {
		pluginByID[plugin.Manifest.ID] = plugin
	}
	mappings, err := ResolveRoutingProfileMappingsFromPlugins(app, userID, installed)
	if err != nil {
		return output, err
	}
	mappingPreferenceLayers := make([]map[string]any, 0, len(input.Routing.Engines))
	profilePreferenceDefaults := make([]map[string]any, 0, len(input.Routing.Engines))
	var profileSupportedPreferences map[string]bool
	for _, engine := range input.Routing.Engines {
		plugin, found := pluginByID[engine.PluginID]
		if !found || plugin.Manifest.Type != pluginsystem.PluginTypeRouting {
			return output, apis.NewBadRequestError("unknown routing plugin", map[string]any{"pluginId": engine.PluginID})
		}
		if _, supported := routingProfilePrepareCapability(plugin); supported {
			output.ProfilePreparationSupported[engine.PluginID] = true
		}
		mapping := resolveRoutingProfileMappingFromMappings(mappings, engine.PluginID, engine.InstanceID, input.Category, input.Subcategory)
		if mapping == nil {
			profilePreferenceDefaults = append(profilePreferenceDefaults, nil)
			continue
		}
		metadata, metadataErr := resolveRoutingEffectiveMappingMetadata(ctx, app, userID, plugin, *mapping)
		if metadataErr != nil {
			return output, metadataErr
		}
		if metadata.Revision != "" {
			output.ProfileRevisions[engine.PluginID] = metadata.Revision
		}
		if metadata.RequiresUpload {
			output.ProfileUploadRequired[engine.PluginID] = true
		}
		if output.Mode == "" {
			output.Mode = metadata.Mode
		}
		if len(input.Routing.Engines) == 1 {
			output.NativeControlGroups = metadata.NativeControlGroups
		}
		if metadata.RestrictedPreferences {
			if profileSupportedPreferences == nil {
				profileSupportedPreferences = metadata.SupportedPreferences
			} else {
				intersectRoutingPreferences(profileSupportedPreferences, metadata.SupportedPreferences)
			}
		}
		mappingPreferenceLayers = append(mappingPreferenceLayers, mapping.Preferences)
		profilePreferenceDefaults = append(profilePreferenceDefaults, metadata.PreferenceDefaults)
	}
	// A settings preference can only be checked against its mode after the
	// selected profile has supplied that mode. Keep the documented precedence:
	// settings first, then the selected category mappings.
	mergeRoutingPreferenceLayer(preferences, routingPreferencesApplicableToMode(settings.DefaultPreferences, output.Mode))
	for _, layer := range mappingPreferenceLayers {
		mergeRoutingPreferenceLayer(preferences, layer)
	}
	supportedPreferences := routingSupportedPreferencesFromPlugins(pluginByID, input.Routing.Engines, output.Mode)
	if profileSupportedPreferences != nil {
		intersectRoutingPreferences(supportedPreferences, profileSupportedPreferences)
	}
	for key := range preferences {
		if !routingPreferenceEnabled(key, supportedPreferences) {
			output.HiddenControls = append(output.HiddenControls, routingHiddenControl{Key: key, Reason: "unsupported_for_selection"})
		}
	}
	for key := range supportedPreferences {
		fallback := defaultRoutingPreferenceValue(key, output.Mode)
		if profileDefault, shared := sharedRoutingProfilePreferenceDefault(profilePreferenceDefaults, key); shared {
			fallback = profileDefault
		}
		control := routingControlForPreference(key, fallback, output.Mode)
		if value, explicitlyConfigured := preferences[key]; explicitlyConfigured {
			control.Current = value
		}
		output.Controls = append(output.Controls, control)
	}
	sort.Slice(output.Controls, func(i, j int) bool { return output.Controls[i].Key < output.Controls[j].Key })
	sort.Slice(output.HiddenControls, func(i, j int) bool { return output.HiddenControls[i].Key < output.HiddenControls[j].Key })
	return output, nil
}

func loadRoutingPluginsWithFallback(app core.App) ([]pluginsystem.LocalPlugin, error) {
	installed, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err == nil && len(installed) > 0 {
		return installed, nil
	}
	local, localErr := pluginsystem.LoadLocalPlugins(pluginsystem.PluginDir())
	if localErr != nil {
		if err != nil {
			return nil, err
		}
		return nil, localErr
	}
	return local, nil
}

type routingEffectiveMappingMetadata struct {
	Revision              string
	RequiresUpload        bool
	Mode                  string
	SupportedPreferences  map[string]bool
	PreferenceDefaults    map[string]any
	RestrictedPreferences bool
	NativeControlGroups   []routingNativeControlGroup
}

func resolveRoutingEffectiveMappingMetadata(
	ctx context.Context,
	app core.App,
	userID string,
	plugin pluginsystem.LocalPlugin,
	mapping routingProfileMappingView,
) (routingEffectiveMappingMetadata, error) {
	metadata := routingEffectiveMappingMetadata{}
	if mapping.ProfileID == "" {
		if mapping.NativeProfileKey == "" {
			return metadata, nil
		}
		profile := routingManifestNativeProfile(plugin.Manifest, mapping.NativeProfileKey)
		if len(profile) == 0 {
			return metadata, nil
		}
		metadata.Mode = stringValue(profile["mode"])
		metadata.PreferenceDefaults = routingManifestProfilePreferenceDefaults(profile, metadata.Mode)
		metadata.Revision = routingResolvedProfileRevision(pluginRoutingProfile{
			PluginID: plugin.Manifest.ID,
			Key:      mapping.NativeProfileKey,
			Kind:     "builtin",
			Mode:     metadata.Mode,
			Metadata: profile,
		})
		metadata.NativeControlGroups = routingNativeControlGroupsFromMetadata(
			anySlice(profile["nativeControlGroups"]),
			mergeRoutingConfigLayers(mapValue(profile["nativeConfig"]), mapping.NativeConfig),
		)
		if supported, declared := routingManifestProfileSupportedPreferences(profile); declared {
			metadata.RestrictedPreferences = true
			metadata.SupportedPreferences = supported
		}
		return metadata, nil
	}

	record, err := app.FindRecordById("routing_profiles", mapping.ProfileID)
	if err != nil {
		return metadata, err
	}
	if err := authorizeRoutingProfileRead(userID, record); err != nil {
		return metadata, err
	}
	profileMetadata := pluginsystem.JSONMapFromRecord(record, "metadata")
	profile := pluginRoutingProfile{
		ID:            record.Id,
		PluginID:      record.GetString("plugin_id"),
		Key:           record.GetString("key"),
		Kind:          record.GetString("kind"),
		Mode:          record.GetString("mode"),
		ContentBase64: record.GetString("content_base64"),
		ContentType:   record.GetString("content_type"),
		Metadata:      profileMetadata,
		NativeConfig:  pluginsystem.JSONMapFromRecord(record, "native_config"),
	}
	metadata.Revision = routingResolvedProfileRevision(profile)
	metadata.RequiresUpload = profile.ContentBase64 != ""
	metadata.Mode = profile.Mode
	rawGroups := anySlice(profileMetadata["nativeControlGroups"])
	metadata.NativeControlGroups = routingNativeControlGroupsFromMetadata(
		rawGroups,
		mergeRoutingConfigLayers(profile.NativeConfig, mapping.NativeConfig),
	)
	if profile.Kind != "custom_file" {
		return metadata, nil
	}
	metadata.RestrictedPreferences = true
	keys := stringSlice(profileMetadata["supportedPreferences"])
	_, preferencesFound := profileMetadata["supportedPreferences"]
	if !preferencesFound || len(rawGroups) == 0 {
		output, introspectionErr := introspectRoutingProfileContext(ctx, app, userID, mapping.InstanceID, profile)
		if introspectionErr != nil {
			return metadata, introspectionErr
		}
		if !preferencesFound {
			keys = output.SupportedPreferences
		}
		if len(rawGroups) == 0 {
			metadata.NativeControlGroups = output.NativeControlGroups
		}
	}
	metadata.SupportedPreferences = make(map[string]bool, len(keys))
	for _, key := range keys {
		metadata.SupportedPreferences[key] = true
	}
	return metadata, nil
}

func routingProfileRevisionForMapping(app core.App, mapping routingProfileMappingView) (string, error) {
	if mapping.ProfileID != "" {
		record, err := app.FindRecordById("routing_profiles", mapping.ProfileID)
		if err != nil {
			return "", err
		}
		return routingResolvedProfileRevision(pluginRoutingProfile{
			ID:            record.Id,
			PluginID:      record.GetString("plugin_id"),
			Key:           record.GetString("key"),
			Kind:          record.GetString("kind"),
			Mode:          record.GetString("mode"),
			ContentBase64: record.GetString("content_base64"),
			ContentType:   record.GetString("content_type"),
			Metadata:      pluginsystem.JSONMapFromRecord(record, "metadata"),
		}), nil
	}
	if mapping.NativeProfileKey == "" {
		return "", nil
	}
	plugin, err := localPlugin(app, mapping.PluginID)
	if err != nil {
		return "", err
	}
	profile := routingManifestNativeProfile(plugin.Manifest, mapping.NativeProfileKey)
	if len(profile) == 0 {
		return "", nil
	}
	return routingResolvedProfileRevision(pluginRoutingProfile{
		PluginID: mapping.PluginID,
		Key:      mapping.NativeProfileKey,
		Kind:     "builtin",
		Mode:     stringValue(profile["mode"]),
		Metadata: profile,
	}), nil
}

func routingProfileMappingRequiresUpload(app core.App, mapping routingProfileMappingView) (bool, error) {
	if mapping.ProfileID == "" {
		return false, nil
	}
	profile, err := app.FindRecordById("routing_profiles", mapping.ProfileID)
	if err != nil {
		return false, err
	}
	return profile.GetString("content_base64") != "", nil
}

func routingResolvedProfileRevision(profile pluginRoutingProfile) string {
	if profile.ID == "" && profile.Key == "" && profile.ContentBase64 == "" && len(profile.Metadata) == 0 {
		return ""
	}
	payload, err := json.Marshal(map[string]any{
		"id": profile.ID, "pluginId": profile.PluginID, "key": profile.Key,
		"kind": profile.Kind, "mode": profile.Mode, "contentBase64": profile.ContentBase64,
		"contentType": profile.ContentType, "metadata": profile.Metadata,
	})
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func routingProfileMappingSupportedPreferences(app core.App, userID string, mapping routingProfileMappingView) (map[string]bool, bool, error) {
	return routingProfileMappingSupportedPreferencesContext(context.Background(), app, userID, mapping)
}

func routingProfileMappingSupportedPreferencesContext(ctx context.Context, app core.App, userID string, mapping routingProfileMappingView) (map[string]bool, bool, error) {
	if mapping.ProfileID == "" {
		if mapping.NativeProfileKey == "" {
			return nil, false, nil
		}
		plugin, err := localPlugin(app, mapping.PluginID)
		if err != nil {
			return nil, false, err
		}
		supported, declared := routingManifestProfileSupportedPreferences(
			routingManifestNativeProfile(plugin.Manifest, mapping.NativeProfileKey),
		)
		return supported, declared, nil
	}
	record, err := app.FindRecordById("routing_profiles", mapping.ProfileID)
	if err != nil {
		return nil, false, err
	}
	if err := authorizeRoutingProfileRead(userID, record); err != nil {
		return nil, false, err
	}
	if record.GetString("kind") != "custom_file" {
		return nil, false, nil
	}

	metadata := pluginsystem.JSONMapFromRecord(record, "metadata")
	rawSupported, found := metadata["supportedPreferences"]
	keys := stringSlice(rawSupported)
	if !found {
		output, err := introspectRoutingProfileContext(ctx, app, userID, mapping.InstanceID, pluginRoutingProfile{
			ID:            record.Id,
			PluginID:      record.GetString("plugin_id"),
			Key:           record.GetString("key"),
			Kind:          record.GetString("kind"),
			Mode:          record.GetString("mode"),
			ContentBase64: record.GetString("content_base64"),
			ContentType:   record.GetString("content_type"),
			Metadata:      metadata,
			NativeConfig:  pluginsystem.JSONMapFromRecord(record, "native_config"),
		})
		if err != nil {
			return nil, true, err
		}
		keys = output.SupportedPreferences
	}

	supported := make(map[string]bool, len(keys))
	for _, key := range keys {
		supported[key] = true
	}
	return supported, true, nil
}

// routingManifestProfileSupportedPreferences reads the standard preferences a
// curated profile can actually honor. Bases differ in what they expose, so a
// plugin-wide declaration is only the union; a profile that declares an empty
// list supports none of them.
func routingManifestProfileSupportedPreferences(profile map[string]any) (map[string]bool, bool) {
	raw, found := profile["supportedPreferences"]
	if !found {
		return nil, false
	}
	keys := stringSlice(raw)
	supported := make(map[string]bool, len(keys))
	for _, key := range keys {
		supported[key] = true
	}
	return supported, true
}

// routingManifestProfilePreferenceDefaults reads provider-declared neutral
// values for standard controls. They are presentation fallbacks only: route
// requests continue to contain just settings, mapping, and client values, so
// an untouched control cannot rewrite the selected profile.
func routingManifestProfilePreferenceDefaults(profile map[string]any, mode string) map[string]any {
	raw, found := profile["preferenceDefaults"]
	if !found {
		return nil
	}
	supported, declared := routingManifestProfileSupportedPreferences(profile)
	if !declared {
		return nil
	}
	defaults := map[string]any{}
	for key, value := range mapValue(raw) {
		if !supported[key] || !isCanonicalRoutingPreference(key) {
			continue
		}
		number, ok := routingFiniteNumber(value)
		if !ok || validateCanonicalRoutingPreferenceRange(key, number) != nil {
			continue
		}
		min, max, _ := routingPreferenceRange(key, mode)
		if number < min || number > max {
			continue
		}
		defaults[key] = number
	}
	if len(defaults) == 0 {
		return nil
	}
	return defaults
}

// sharedRoutingProfilePreferenceDefault prevents one engine's profile default
// from being presented as though it applied to all selected engines. A
// provider-specific fallback is used only when every selected profile declares
// the same numeric value.
func sharedRoutingProfilePreferenceDefault(defaults []map[string]any, key string) (any, bool) {
	if len(defaults) == 0 {
		return nil, false
	}
	var shared any
	var sharedNumber float64
	for index, profileDefaults := range defaults {
		value, found := profileDefaults[key]
		number, numeric := routingFiniteNumber(value)
		if !found || !numeric {
			return nil, false
		}
		if index == 0 {
			shared, sharedNumber = value, number
			continue
		}
		if number != sharedNumber {
			return nil, false
		}
	}
	return shared, true
}

func intersectRoutingPreferences(target map[string]bool, supported map[string]bool) {
	for key := range target {
		if !supported[key] {
			delete(target, key)
		}
	}
}

// clampRoutingPreferencesToMode keeps merged preference layers inside the range
// the host advertises for the resolved mode. The settings-wide defaults are
// mode-agnostic, so without this a walking speed could reach a car engine.
func clampRoutingPreferencesToMode(preferences map[string]any, mode string) {
	for key, value := range preferences {
		if !isCanonicalRoutingPreference(key) {
			continue
		}
		number, ok := routingFiniteNumber(value)
		if !ok {
			continue
		}
		min, max, _ := routingPreferenceRange(key, mode)
		if number < min {
			preferences[key] = min
			continue
		}
		if number > max {
			preferences[key] = max
		}
	}
}

func restrictRoutingPreferences(preferences map[string]any, supported map[string]bool) {
	for key := range preferences {
		if isCanonicalRoutingPreference(key) && !supported[key] {
			delete(preferences, key)
		}
	}
}

// restrictRoutingPluginPreferences applies the plugin manifest's declared
// standard and provider-specific preference keys as a complete allowlist.
// Undeclared client-controlled values must never reach a provider process.
func restrictRoutingPluginPreferences(plugin pluginsystem.LocalPlugin, request *pluginRoutingRouteRequest) {
	allowed := routingPluginSupportedPreferences(plugin, request.Mode)
	routingMetadata := mapValue(plugin.Manifest.Metadata["routing"])
	for _, key := range stringSlice(routingMetadata["providerPreferences"]) {
		key = strings.TrimSpace(key)
		if key != "" {
			allowed[key] = true
		}
	}
	for key := range request.Preferences {
		if !allowed[key] {
			delete(request.Preferences, key)
		}
	}
}

func isCanonicalRoutingPreference(key string) bool {
	switch key {
	case "speedPreference", "hillPreference", "maxHikingDifficulty", "roadPreference", "avoidBadSurfaces", "vehicleWidth", "vehicleHeight":
		return true
	default:
		return false
	}
}

func validateRoutingPreferences(preferences map[string]any, canonicalOnly bool) error {
	if len(preferences) > routingPreferenceMaxEntries {
		return fmt.Errorf("too many preference entries (maximum %d)", routingPreferenceMaxEntries)
	}
	for rawKey, value := range preferences {
		key := strings.TrimSpace(rawKey)
		if key == "" || key != rawKey || len(key) > routingConfigMaxKeyBytes {
			return fmt.Errorf("preference key is empty, padded, or too long")
		}
		if canonicalOnly && !isCanonicalRoutingPreference(key) {
			return fmt.Errorf("unknown standard preference %q", key)
		}
		switch typed := value.(type) {
		case bool:
			if isCanonicalRoutingPreference(key) {
				return fmt.Errorf("preference %q must be numeric", key)
			}
		case string:
			if len(typed) > routingConfigMaxStringBytes {
				return fmt.Errorf("preference %q is too long", key)
			}
			if isCanonicalRoutingPreference(key) {
				return fmt.Errorf("preference %q must be numeric", key)
			}
		default:
			number, ok := routingFiniteNumber(typed)
			if !ok {
				return fmt.Errorf("preference %q must be a finite scalar", key)
			}
			if err := validateCanonicalRoutingPreferenceRange(key, number); err != nil {
				return err
			}
		}
	}
	return nil
}

func routingFiniteNumber(value any) (float64, bool) {
	var number float64
	switch typed := value.(type) {
	case int:
		number = float64(typed)
	case int8:
		number = float64(typed)
	case int16:
		number = float64(typed)
	case int32:
		number = float64(typed)
	case int64:
		number = float64(typed)
	case uint:
		number = float64(typed)
	case uint8:
		number = float64(typed)
	case uint16:
		number = float64(typed)
	case uint32:
		number = float64(typed)
	case uint64:
		number = float64(typed)
	case float32:
		number = float64(typed)
	case float64:
		number = typed
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		number = parsed
	default:
		return 0, false
	}
	return number, !math.IsNaN(number) && !math.IsInf(number, 0)
}

func validateCanonicalRoutingPreferenceRange(key string, value float64) error {
	minimum, maximum, constrained := 0.0, 0.0, true
	switch key {
	case "speedPreference":
		minimum, maximum = 0.1, 300
	case "hillPreference", "roadPreference", "avoidBadSurfaces":
		minimum, maximum = 0, 1
	case "maxHikingDifficulty":
		minimum, maximum = 0, 6
	case "vehicleWidth", "vehicleHeight":
		minimum, maximum = 0.5, 10
	default:
		constrained = false
	}
	if constrained && (value < minimum || value > maximum) {
		return fmt.Errorf("preference %q must be between %g and %g", key, minimum, maximum)
	}
	return nil
}

func validateRoutingConfig(value map[string]any, field string) error {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("%s must contain JSON-compatible values: %w", field, err)
	}
	if len(encoded) > routingConfigMaxBytes {
		return fmt.Errorf("%s exceeds %d bytes", field, routingConfigMaxBytes)
	}
	entries := 0
	if err := validateRoutingConfigValue(value, 0, &entries); err != nil {
		return fmt.Errorf("invalid %s: %w", field, err)
	}
	return nil
}

func validateRoutingConfigValue(value any, depth int, entries *int) error {
	if depth > routingConfigMaxDepth {
		return fmt.Errorf("nesting exceeds %d levels", routingConfigMaxDepth)
	}
	switch typed := value.(type) {
	case map[string]any:
		*entries += len(typed)
		if *entries > routingConfigMaxEntries {
			return fmt.Errorf("configuration exceeds %d entries", routingConfigMaxEntries)
		}
		for key, nested := range typed {
			if strings.TrimSpace(key) == "" || len(key) > routingConfigMaxKeyBytes {
				return fmt.Errorf("configuration key is empty or too long")
			}
			if err := validateRoutingConfigValue(nested, depth+1, entries); err != nil {
				return err
			}
		}
	case []any:
		*entries += len(typed)
		if *entries > routingConfigMaxEntries {
			return fmt.Errorf("configuration exceeds %d entries", routingConfigMaxEntries)
		}
		for _, nested := range typed {
			if err := validateRoutingConfigValue(nested, depth+1, entries); err != nil {
				return err
			}
		}
	case string:
		if len(typed) > routingConfigMaxStringBytes {
			return fmt.Errorf("configuration string exceeds %d bytes", routingConfigMaxStringBytes)
		}
	case nil, bool:
		return nil
	default:
		if _, ok := routingFiniteNumber(typed); !ok {
			return fmt.Errorf("configuration contains a non-finite or unsupported value")
		}
	}
	return nil
}

func routingProfileMappingMode(app core.App, mapping routingProfileMappingView) (string, error) {
	if mapping.ProfileID != "" {
		profile, err := app.FindRecordById("routing_profiles", mapping.ProfileID)
		if err != nil {
			return "", err
		}
		return profile.GetString("mode"), nil
	}
	if mapping.NativeProfileKey == "" {
		return "", nil
	}
	plugin, err := localPlugin(app, mapping.PluginID)
	if err != nil {
		return "", err
	}
	profile := routingManifestNativeProfile(plugin.Manifest, mapping.NativeProfileKey)
	return stringValue(profile["mode"]), nil
}

func ResolveRoutingNativeControls(app core.App, userID string, input routingNativeControlsInput) (routingNativeControlsOutput, error) {
	return ResolveRoutingNativeControlsContext(context.Background(), app, userID, input)
}

func ResolveRoutingNativeControlsContext(ctx context.Context, app core.App, userID string, input routingNativeControlsInput) (routingNativeControlsOutput, error) {
	if input.ProfileID != "" && input.PluginID == "" {
		record, err := app.FindRecordById("routing_profiles", input.ProfileID)
		if err != nil {
			return routingNativeControlsOutput{}, apis.NewNotFoundError("routing profile not found", err)
		}
		if err := authorizeRoutingProfileRead(userID, record); err != nil {
			return routingNativeControlsOutput{}, err
		}
		input.PluginID = record.GetString("plugin_id")
	}
	output := routingNativeControlsOutput{
		PluginID:   input.PluginID,
		InstanceID: input.InstanceID,
		ProfileID:  input.ProfileID,
		Groups:     []routingNativeControlGroup{},
	}
	if output.PluginID == "" {
		settings, err := ResolveRoutingSettings(app, userID)
		if err != nil {
			return output, err
		}
		if settings.PrimaryRoutePluginID == "" {
			return output, apis.NewBadRequestError("pluginId is required when no primary routing engine is configured", nil)
		}
		output.PluginID = settings.PrimaryRoutePluginID
		input.PluginID = settings.PrimaryRoutePluginID
	}
	groups, err := manifestRoutingNativeControlGroups(ctx, app, userID, input)
	if err != nil {
		return output, err
	}
	output.Groups = groups
	return output, nil
}

func routingSettingsForScope(app core.App, scope string, userID string) (routingSettings, error) {
	record, err := pluginsystem.RoutingSettingsRecordForScope(app, scope, userID)
	if err != nil || record == nil {
		return routingSettings{}, err
	}
	config := pluginsystem.JSONMapFromRecord(record, "config")
	if discarded := sanitizePersistedRoutingDefaultPreferences(config); discarded > 0 {
		warnInvalidPersistedRoutingDefaultPreferences(app, scope+" settings", userID, discarded)
	}
	return routingSettingsFromConfig(config), nil
}

func saveRoutingSettingsConfig(app core.App, scope string, userID string, config map[string]any) error {
	collection, err := app.FindCollectionByNameOrId("routing_settings")
	if err != nil {
		return err
	}
	record, err := pluginsystem.RoutingSettingsRecordForScope(app, scope, userID)
	if err != nil {
		return err
	}
	if record == nil {
		record = core.NewRecord(collection)
		record.Set("scope", scope)
		if userID != "" {
			record.Set("user", userID)
		}
	}
	record.Set("config", config)
	return app.Save(record)
}

func patchRoutingSettingsConfig(app core.App, scope string, userID string, patch map[string]any) error {
	for key := range patch {
		if _, supported := routingSettingKeys[key]; !supported {
			return apis.NewBadRequestError("unknown routing setting: "+key, nil)
		}
	}
	record, err := pluginsystem.RoutingSettingsRecordForScope(app, scope, userID)
	if err != nil {
		return err
	}
	config := map[string]any{}
	if record != nil {
		config = pluginsystem.JSONMapFromRecord(record, "config")
	}
	if discarded := sanitizePersistedRoutingDefaultPreferences(config); discarded > 0 {
		warnInvalidPersistedRoutingDefaultPreferences(app, scope+" settings update", userID, discarded)
	}
	if raw, exists := config["exposedFeatures"]; exists {
		config["exposedFeatures"] = knownRoutingFeatures(raw)
	}
	for key, value := range patch {
		if key == "exposedFeatures" {
			featurePatch, err := validatedRoutingFeatures(value)
			if err != nil {
				return err
			}
			if scope == routingScopeUser {
				if _, exists := featurePatch["navigation"]; exists {
					return apis.NewBadRequestError("navigation is controlled by administrators", nil)
				}
			}
			features := cloneRoutingMap(mapValue(config[key]))
			for feature, enabled := range featurePatch {
				features[feature] = enabled
			}
			config[key] = features
			continue
		}
		config[key] = value
	}
	if err := normalizeRoutingSettingsConfig(config); err != nil {
		return err
	}
	return saveRoutingSettingsConfig(app, scope, userID, config)
}

// sanitizePersistedRoutingDefaultPreferences makes legacy settings safe to
// consume without weakening validation for newly submitted values. It mutates
// only the already-persisted config passed by the caller and returns the number
// of discarded entries so callers can emit one aggregate warning.
func sanitizePersistedRoutingDefaultPreferences(config map[string]any) int {
	raw, exists := config["defaultPreferences"]
	if !exists {
		return 0
	}
	preferences, ok := raw.(map[string]any)
	if !ok || preferences == nil {
		delete(config, "defaultPreferences")
		return 1
	}

	keys := make([]string, 0, len(preferences))
	for key := range preferences {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	sanitized := make(map[string]any, min(len(preferences), routingPreferenceMaxEntries))
	discarded := 0
	for _, key := range keys {
		value := preferences[key]
		if len(sanitized) >= routingPreferenceMaxEntries || validateRoutingPreferences(map[string]any{key: value}, false) != nil {
			discarded++
			continue
		}
		sanitized[key] = value
	}
	config["defaultPreferences"] = sanitized
	return discarded
}

// limitResolvedRoutingDefaultPreferences applies the effective preference
// bound without allowing lexicographic map order to erase a nearer scope. A
// key defined by the user wins admission before admin and builtin keys; its
// value still comes from the normal merged result.
func limitResolvedRoutingDefaultPreferences(config map[string]any, layers pluginsystem.RoutingSettingsLayers) int {
	preferences := mapValue(config["defaultPreferences"])
	if len(preferences) <= routingPreferenceMaxEntries {
		return 0
	}

	limited := make(map[string]any, routingPreferenceMaxEntries)
	for _, layer := range []map[string]any{layers.User, layers.Admin, layers.Builtin} {
		layerPreferences := mapValue(layer["defaultPreferences"])
		keys := make([]string, 0, len(layerPreferences))
		for key := range layerPreferences {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if len(limited) == routingPreferenceMaxEntries {
				break
			}
			if _, exists := limited[key]; exists {
				continue
			}
			if value, exists := preferences[key]; exists {
				limited[key] = value
			}
		}
	}
	config["defaultPreferences"] = limited
	return len(preferences) - len(limited)
}

func warnInvalidPersistedRoutingDefaultPreferences(app core.App, source string, userID string, discarded int) {
	routingInvalidPersistedPreferencesWarning.Do(func() {
		app.Logger().Warn(
			"discarded invalid persisted routing default preferences; further warnings are suppressed",
			"source", source,
			"user", userID,
			"discarded", discarded,
		)
	})
}

func normalizeRoutingSettingsConfig(config map[string]any) error {
	for _, key := range []string{"primaryRoutePluginId", "elevationPluginId", "maneuverPluginId"} {
		if raw, exists := config[key]; exists {
			value, ok := raw.(string)
			if !ok {
				return apis.NewBadRequestError(key+" must be a string", nil)
			}
			config[key] = strings.TrimSpace(value)
		}
	}
	if raw, exists := config["defaultVariantCount"]; exists {
		count, valid := routingInteger(raw)
		if !valid {
			return apis.NewBadRequestError("defaultVariantCount must be a number", nil)
		}
		if count < 1 {
			count = 1
		}
		if count > routingMaxVariants {
			count = routingMaxVariants
		}
		config["defaultVariantCount"] = count
	}
	if raw, exists := config["defaultRoutingMode"]; exists {
		mode := strings.TrimSpace(stringValue(raw))
		if mode != "segment" && mode != "via" {
			return apis.NewBadRequestError("defaultRoutingMode must be segment or via", nil)
		}
		config["defaultRoutingMode"] = mode
	}
	if raw, exists := config["defaultPreferences"]; exists {
		preferences, ok := raw.(map[string]any)
		if !ok || preferences == nil {
			return apis.NewBadRequestError("defaultPreferences must be an object", nil)
		}
		if err := validateRoutingPreferences(preferences, false); err != nil {
			return apis.NewBadRequestError("invalid defaultPreferences", err)
		}
		config["defaultPreferences"] = cloneRoutingMap(preferences)
	}
	if raw, exists := config["exposedFeatures"]; exists {
		features, err := validatedRoutingFeatures(raw)
		if err != nil {
			return err
		}
		config["exposedFeatures"] = features
	}
	return nil
}

func knownRoutingFeatures(value any) map[string]any {
	raw := mapValue(value)
	result := map[string]any{}
	for key, value := range raw {
		if _, allowed := routingFeatureKeys[key]; !allowed {
			continue
		}
		if enabled, ok := value.(bool); ok {
			result[key] = enabled
		}
	}
	return result
}

func validatedRoutingFeatures(value any) (map[string]any, error) {
	raw, ok := value.(map[string]any)
	if !ok || raw == nil {
		return nil, apis.NewBadRequestError("exposedFeatures must be an object", nil)
	}
	result := make(map[string]any, len(raw))
	for key, value := range raw {
		if _, allowed := routingFeatureKeys[key]; !allowed {
			return nil, apis.NewBadRequestError("unknown routing feature", map[string]any{
				"feature": key,
			})
		}
		enabled, ok := value.(bool)
		if !ok {
			return nil, apis.NewBadRequestError("routing feature values must be boolean", map[string]any{
				"feature": key,
			})
		}
		result[key] = enabled
	}
	return result, nil
}

func routingInteger(value any) (int, bool) {
	switch value := value.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), value == float64(int(value))
	case float32:
		return int(value), value == float32(int(value))
	default:
		return 0, false
	}
}

func routingSettingsFromConfig(config map[string]any) routingSettings {
	settings := routingSettings{
		PrimaryRoutePluginID: stringValue(config["primaryRoutePluginId"]),
		ElevationPluginID:    stringValue(config["elevationPluginId"]),
		ManeuverPluginID:     stringValue(config["maneuverPluginId"]),
		DefaultVariantCount:  intValue(config["defaultVariantCount"]),
		DefaultRoutingMode:   stringValue(config["defaultRoutingMode"]),
	}
	if value, exists := config["defaultPreferences"]; exists {
		settings.DefaultPreferences = mapValue(value)
	}
	if value, exists := config["exposedFeatures"]; exists {
		settings.ExposedFeatures = boolMap(value)
	}
	return settings
}

func routingProfileMappingUpsertRecord(app core.App, collection *core.Collection, scope string, userID string, input routingProfileMappingView) (*core.Record, error) {
	filter := "scope={:scope} && plugin_id={:plugin} && instance_id={:instance} && category={:category}"
	params := dbx.Params{
		"scope":    scope,
		"plugin":   input.PluginID,
		"instance": input.InstanceID,
		"category": input.Category,
	}
	if scope == routingScopeUser {
		filter += " && user={:user}"
		params["user"] = userID
	}
	records, err := app.FindRecordsByFilter("routing_profile_mappings", filter, "", -1, 0, params)
	if err != nil {
		return nil, err
	}
	var record *core.Record
	for _, candidate := range records {
		if candidate.GetString("subcategory") == input.Subcategory {
			record = candidate
			break
		}
	}
	if record == nil {
		record = core.NewRecord(collection)
		record.Set("scope", scope)
		if userID != "" {
			record.Set("user", userID)
		}
	}
	return record, nil
}

func routingProfileMappingFromRecord(record *core.Record) routingProfileMappingView {
	return routingProfileMappingView{
		ID:               record.Id,
		Scope:            record.GetString("scope"),
		Category:         record.GetString("category"),
		Subcategory:      record.GetString("subcategory"),
		PluginID:         record.GetString("plugin_id"),
		InstanceID:       record.GetString("instance_id"),
		NativeProfileKey: record.GetString("native_profile_key"),
		ProfileID:        record.GetString("profile"),
		Preferences:      pluginsystem.JSONMapFromRecord(record, "preferences"),
		NativeConfig:     pluginsystem.JSONMapFromRecord(record, "native_config"),
	}
}

func setRoutingProfileMappingRecord(record *core.Record, input routingProfileMappingView) {
	record.Set("category", input.Category)
	record.Set("subcategory", input.Subcategory)
	record.Set("plugin_id", input.PluginID)
	record.Set("instance_id", input.InstanceID)
	record.Set("native_profile_key", input.NativeProfileKey)
	record.Set("profile", input.ProfileID)
	record.Set("preferences", input.Preferences)
	record.Set("native_config", input.NativeConfig)
}

func validateRoutingProfileMappingInput(input routingProfileMappingView) error {
	if strings.TrimSpace(input.Category) == "" || strings.TrimSpace(input.PluginID) == "" {
		return apis.NewBadRequestError("routing profile mapping requires category and pluginId", nil)
	}
	if input.NativeProfileKey == "" && input.ProfileID == "" {
		return apis.NewBadRequestError("routing profile mapping requires a profile", nil)
	}
	if err := validateRoutingPreferences(input.Preferences, false); err != nil {
		return apis.NewBadRequestError("invalid routing profile mapping preferences", err)
	}
	if err := validateRoutingConfig(input.NativeConfig, "nativeConfig"); err != nil {
		return apis.NewBadRequestError("invalid routing profile mapping nativeConfig", err)
	}
	return nil
}

func routingProfileUpsertRecord(app core.App, collection *core.Collection, scope string, userID string, input routingProfileView) (*core.Record, error) {
	filter := "scope={:scope} && plugin_id={:plugin} && key={:key}"
	params := dbx.Params{
		"scope":  scope,
		"plugin": input.PluginID,
		"key":    input.Key,
	}
	if scope == routingScopeUser {
		filter += " && user={:user}"
		params["user"] = userID
	}
	record, err := findOptionalRoutingRecord(app, "routing_profiles", filter, params)
	if err != nil {
		return nil, err
	}
	if record == nil {
		record = core.NewRecord(collection)
		record.Set("scope", scope)
		if userID != "" {
			record.Set("user", userID)
		}
	}
	return record, nil
}

func routingProfileFromRecord(record *core.Record) routingProfileView {
	return routingProfileView{
		ID:            record.Id,
		Scope:         record.GetString("scope"),
		PluginID:      record.GetString("plugin_id"),
		Key:           record.GetString("key"),
		Name:          record.GetString("name"),
		Kind:          record.GetString("kind"),
		Mode:          record.GetString("mode"),
		Source:        record.GetString("scope"),
		ContentBase64: record.GetString("content_base64"),
		ContentType:   record.GetString("content_type"),
		Metadata:      pluginsystem.JSONMapFromRecord(record, "metadata"),
		NativeConfig:  pluginsystem.JSONMapFromRecord(record, "native_config"),
		Enabled:       record.GetBool("enabled"),
	}
}

func setRoutingProfileRecord(record *core.Record, input routingProfileView) error {
	if input.Kind == "builtin" {
		return apis.NewBadRequestError("builtin routing profiles are discovered and cannot be materialized", nil)
	}
	if err := validateRoutingProfileInput(input); err != nil {
		return err
	}
	record.Set("plugin_id", input.PluginID)
	record.Set("key", input.Key)
	record.Set("name", input.Name)
	record.Set("kind", input.Kind)
	record.Set("mode", input.Mode)
	record.Set("content_base64", input.ContentBase64)
	record.Set("content_type", input.ContentType)
	record.Set("metadata", input.Metadata)
	record.Set("native_config", input.NativeConfig)
	record.Set("enabled", input.Enabled)
	return nil
}

func validateRoutingProfileInput(input routingProfileView) error {
	if input.PluginID == "" || input.Key == "" || input.Name == "" || input.Kind == "" || input.Mode == "" {
		return apis.NewBadRequestError("routing profile requires pluginId, key, name, kind, and mode", nil)
	}
	if err := validateRoutingConfig(input.Metadata, "metadata"); err != nil {
		return apis.NewBadRequestError("invalid routing profile metadata", err)
	}
	if err := validateRoutingConfig(input.NativeConfig, "nativeConfig"); err != nil {
		return apis.NewBadRequestError("invalid routing profile nativeConfig", err)
	}
	switch input.Kind {
	case "custom_file":
		if input.ContentBase64 == "" {
			return apis.NewBadRequestError("routing profile file content is required", nil)
		}
		if err := validateRoutingProfileContentSize(input.ContentBase64); err != nil {
			return err
		}
		if input.ContentType == "" {
			return apis.NewBadRequestError("routing profile file content type is required", nil)
		}
		return nil
	case "generated":
		if len(input.Metadata) == 0 && len(input.NativeConfig) == 0 {
			return apis.NewBadRequestError("generated routing profiles require metadata or native config", nil)
		}
		return nil
	case "native_config":
		return nil
	default:
		return apis.NewBadRequestError("unsupported routing profile kind", map[string]any{"kind": input.Kind})
	}
}

func validateRoutingProfileContentSize(contentBase64 string) error {
	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return apis.NewBadRequestError("routing profile file content is not valid base64", err)
	}
	if len(content) == 0 || len(content) > routingProfileContentMaxBytes {
		return apis.NewBadRequestError("routing profile file content is empty or too large", map[string]any{"maxBytes": routingProfileContentMaxBytes})
	}
	return nil
}

func routingWritableScope(e *core.RequestEvent, requested string) (string, string, error) {
	if requested == "" || requested == routingScopeUser {
		if e.Auth == nil {
			return "", "", apis.NewUnauthorizedError("authentication required", nil)
		}
		return routingScopeUser, e.Auth.Id, nil
	}
	if requested == routingScopeAdmin {
		if !e.HasSuperuserAuth() {
			return "", "", apis.NewUnauthorizedError("superuser authentication required", nil)
		}
		return routingScopeAdmin, "", nil
	}
	return "", "", apis.NewBadRequestError("builtin routing records are read-only", nil)
}

func authorizeRoutingScopedRecordWrite(e *core.RequestEvent, record *core.Record) error {
	switch record.GetString("scope") {
	case routingScopeAdmin:
		if !e.HasSuperuserAuth() {
			return apis.NewUnauthorizedError("superuser authentication required", nil)
		}
	case routingScopeUser:
		if e.Auth == nil || record.GetString("user") != e.Auth.Id {
			return apis.NewUnauthorizedError("authentication required", nil)
		}
	default:
		return apis.NewBadRequestError("builtin routing records are read-only", nil)
	}
	return nil
}

func scopedRoutingRecords(app core.App, collection string, userID string) ([]*core.Record, error) {
	records, err := app.FindRecordsByFilter(collection, "scope='builtin' || scope='admin' || (scope='user' && user={:user})", "", -1, 0, dbx.Params{"user": userID})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(records, func(i, j int) bool {
		return routingScopeRank(records[i].GetString("scope")) < routingScopeRank(records[j].GetString("scope"))
	})
	return records, nil
}

func findOptionalRoutingRecord(app core.App, collection string, filter string, params dbx.Params) (*core.Record, error) {
	record, err := app.FindFirstRecordByFilter(collection, filter, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return record, nil
}

func routingEngines(app core.App, userID string) ([]routingEngineView, error) {
	plugins, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err != nil {
		return nil, err
	}
	instances, err := app.FindRecordsByFilter("plugin_instances", "user={:user} && enabled=true", "+created,+id", -1, 0, dbx.Params{"user": userID})
	if err != nil {
		return nil, err
	}
	enabled := map[string]*core.Record{}
	discoveryRanks := map[string]int{}
	for index, instance := range instances {
		pluginID := instance.GetString("plugin_id")
		if _, exists := enabled[pluginID]; exists {
			continue
		}
		enabled[pluginID] = instance
		discoveryRanks[pluginID] = index
	}
	engines := []routingEngineView{}
	for _, plugin := range plugins {
		if plugin.Manifest.Type != pluginsystem.PluginTypeRouting {
			continue
		}
		instance := enabled[plugin.Manifest.ID]
		metadata := plugin.Manifest.Metadata
		routingMetadata := cloneRoutingMap(mapValue(metadata["routing"]))
		if routingMetadata["supportsRoundTrip"] == true && !routingManifestHasCapability(plugin.Manifest, "round_trip", "v1") {
			routingMetadata["supportsRoundTrip"] = false
		}
		roles := canonicalRoutingRoles(plugin.Manifest)
		routingMetadata["roles"] = roles
		engines = append(engines, routingEngineView{
			PluginID:      plugin.Manifest.ID,
			InstanceID:    instanceID(instance),
			Name:          plugin.Manifest.Name,
			Enabled:       instance != nil,
			Roles:         stringSlice(routingMetadata["roles"]),
			Modes:         stringSlice(routingMetadata["modes"]),
			Metadata:      map[string]any{"routing": routingMetadata},
			discoveryRank: discoveryRanks[plugin.Manifest.ID],
		})
	}
	sort.Slice(engines, func(i, j int) bool {
		return engines[i].PluginID < engines[j].PluginID
	})
	return engines, nil
}

// applyRoutingEngineDefaults fills unconfigured roles from loadable plugins
// with an enabled user instance. The user's stable instance setup order is the
// provider-neutral tie-breaker. Elevation and maneuver selection prefer the
// primary route engine when it also implements the required capability, which
// keeps the normal single-provider path coherent without requiring it.
func applyRoutingEngineDefaults(settings *routingSettings, engines []routingEngineView) {
	if settings == nil {
		return
	}
	if settings.PrimaryRoutePluginID == "" {
		settings.PrimaryRoutePluginID = defaultRoutingEngineForRole(engines, "route", "")
	}
	if settings.ElevationPluginID == "" {
		settings.ElevationPluginID = defaultRoutingEngineForRole(engines, "elevation", settings.PrimaryRoutePluginID)
	}
	if settings.ManeuverPluginID == "" {
		settings.ManeuverPluginID = defaultRoutingEngineForRole(engines, "maneuvers", settings.PrimaryRoutePluginID)
	}
}

func defaultRoutingEngineForRole(engines []routingEngineView, role string, preferredPluginID string) string {
	preferredPluginID = strings.TrimSpace(preferredPluginID)
	bestIndex := -1
	for index := range engines {
		engine := engines[index]
		if !engine.Enabled || !containsRoutingString(engine.Roles, role) {
			continue
		}
		if preferredPluginID != "" && engine.PluginID == preferredPluginID {
			return engine.PluginID
		}
		if bestIndex < 0 || engine.discoveryRank < engines[bestIndex].discoveryRank ||
			(engine.discoveryRank == engines[bestIndex].discoveryRank && engine.PluginID < engines[bestIndex].PluginID) {
			bestIndex = index
		}
	}
	if bestIndex < 0 {
		return ""
	}
	return engines[bestIndex].PluginID
}

func routingManifestHasRole(manifest pluginsystem.Manifest, role string) bool {
	for _, candidate := range canonicalRoutingRoles(manifest) {
		if candidate == role {
			return true
		}
	}
	return false
}

func routingManifestSupportsManeuvers(manifest pluginsystem.Manifest) bool {
	return routingManifestHasRole(manifest, "maneuvers")
}

func canonicalRoutingRoles(manifest pluginsystem.Manifest) []string {
	roles := make([]string, 0, 4)
	for _, role := range []string{"route", "elevation", "maneuvers", "round_trip"} {
		if routingManifestHasCapability(manifest, role, "v1") {
			roles = append(roles, role)
		}
	}
	return roles
}

func containsRoutingString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func routingManifestHasCapability(manifest pluginsystem.Manifest, name string, version string) bool {
	for _, capability := range manifest.Capabilities {
		if capability.Name == name && capability.Version == version {
			return true
		}
	}
	return false
}

func routingSupportedPreferences(app core.App, engines []routingEngineSelection, mode string) (map[string]bool, error) {
	if len(engines) == 0 {
		return map[string]bool{}, nil
	}
	plugins, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err != nil {
		return nil, err
	}
	pluginByID := map[string]pluginsystem.LocalPlugin{}
	for _, plugin := range plugins {
		pluginByID[plugin.Manifest.ID] = plugin
	}
	return routingSupportedPreferencesFromPlugins(pluginByID, engines, mode), nil
}

func routingSupportedPreferencesFromPlugins(pluginByID map[string]pluginsystem.LocalPlugin, engines []routingEngineSelection, mode string) map[string]bool {
	var supported map[string]bool
	for _, engine := range engines {
		plugin, ok := pluginByID[engine.PluginID]
		if !ok {
			return map[string]bool{}
		}
		engineSupported := routingPluginSupportedPreferences(plugin, mode)
		if supported == nil {
			supported = engineSupported
			continue
		}
		for key := range supported {
			if !engineSupported[key] {
				delete(supported, key)
			}
		}
	}
	if supported == nil {
		return map[string]bool{}
	}
	return supported
}

func routingPluginSupportedPreferences(plugin pluginsystem.LocalPlugin, mode string) map[string]bool {
	routingMetadata := mapValue(plugin.Manifest.Metadata["routing"])
	supported := map[string]bool{}
	for _, raw := range anySlice(routingMetadata["standardPreferences"]) {
		preference := mapValue(raw)
		key := stringValue(preference["key"])
		if key == "" {
			continue
		}
		modes := stringSlice(preference["modes"])
		if len(modes) > 0 && mode != "" && !containsString(modes, mode) {
			continue
		}
		supported[key] = true
	}
	return supported
}

func routingDiscoveryProfiles(app core.App) []routingProfileView {
	plugins, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err != nil {
		return nil
	}
	var profiles []routingProfileView
	for _, plugin := range plugins {
		if plugin.Manifest.Type != pluginsystem.PluginTypeRouting {
			continue
		}
		routingMetadata := mapValue(plugin.Manifest.Metadata["routing"])
		for _, raw := range anySlice(routingMetadata["nativeProfiles"]) {
			profile := mapValue(raw)
			key := stringValue(profile["key"])
			if key == "" {
				continue
			}
			profiles = append(profiles, routingProfileView{
				Scope:    routingScopeBuiltin,
				PluginID: plugin.Manifest.ID,
				Key:      key,
				Name:     fallbackString(stringValue(profile["label"]), key),
				Kind:     "builtin",
				Mode:     stringValue(profile["mode"]),
				Source:   "discovery",
				Enabled:  true,
				Metadata: profile,
			})
		}
	}
	return profiles
}

func seedRoutingBuiltinSettings(app core.App) error {
	config := map[string]any{
		"defaultVariantCount": 3,
		"defaultRoutingMode":  "segment",
		"defaultPreferences":  defaultRoutingPreferences(),
		"exposedFeatures": map[string]any{
			"routing":                true,
			"navigation":             true,
			"standardControls":       true,
			"nativeAdvancedControls": true,
			"profileUpload":          true,
			"parallelRouting":        true,
			"variants":               true,
			"routeCandidates":        false,
		},
	}
	return saveRoutingSettingsConfig(app, routingScopeBuiltin, "", config)
}

func builtinRoutingProfileMappings(app core.App) ([]routingProfileMappingView, error) {
	plugins, err := pluginsystem.LoadInstalledPlugins(app, "")
	if err != nil || len(plugins) == 0 {
		plugins, err = pluginsystem.LoadLocalPlugins(pluginsystem.PluginDir())
		if err != nil {
			return nil, err
		}
	}
	return builtinRoutingProfileMappingsFromPlugins(plugins), nil
}

func builtinRoutingProfileMappingsFromPlugins(plugins []pluginsystem.LocalPlugin) []routingProfileMappingView {
	mappings := []routingProfileMappingView{}
	for _, plugin := range plugins {
		if plugin.Manifest.Type != pluginsystem.PluginTypeRouting {
			continue
		}
		routingMetadata := mapValue(plugin.Manifest.Metadata["routing"])
		for _, raw := range anySlice(routingMetadata["categoryMappings"]) {
			mapping := mapValue(raw)
			category := stringValue(mapping["category"])
			profileKey := stringValue(mapping["profile"])
			if category == "" || profileKey == "" {
				continue
			}
			profile := routingManifestNativeProfile(plugin.Manifest, profileKey)
			if len(profile) == 0 {
				continue
			}
			mappings = append(mappings, routingProfileMappingView{
				Scope:            routingScopeBuiltin,
				Category:         category,
				Subcategory:      stringValue(mapping["subcategory"]),
				PluginID:         plugin.Manifest.ID,
				NativeProfileKey: profileKey,
				Preferences:      cloneRoutingMap(mapValue(mapping["preferences"])),
				NativeConfig: mergeRoutingConfigLayers(
					mapValue(profile["nativeConfig"]),
					mapValue(mapping["nativeConfig"]),
				),
			})
		}
	}
	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].PluginID == mappings[j].PluginID {
			if mappings[i].Category == mappings[j].Category {
				return mappings[i].Subcategory < mappings[j].Subcategory
			}
			return mappings[i].Category < mappings[j].Category
		}
		return mappings[i].PluginID < mappings[j].PluginID
	})
	return mappings
}

// defaultRoutingPreferences seeds only the preferences that mean the same thing
// in every travel mode. A speed does not: 5 km/h is a brisk walk and a stalled
// car, so it belongs to the category mapping or to the profile itself.
func defaultRoutingPreferences() map[string]any {
	return map[string]any{
		"hillPreference": 0.5,
	}
}

// routingPreferencesApplicableToMode drops values a layer carries that cannot
// apply to the resolved travel mode. The settings-wide defaults are written
// before any mode is known, and a speed is the one preference whose scale
// differs per mode: a walking speed reaching a car profile would overwrite the
// target speed that profile defines for itself. Values that do fit the mode are
// kept, so a deliberately configured default still applies.
func routingPreferencesApplicableToMode(preferences map[string]any, mode string) map[string]any {
	if len(preferences) == 0 {
		return preferences
	}
	filtered := make(map[string]any, len(preferences))
	for key, value := range preferences {
		if routingPreferenceIsModeScaled(key) {
			number, ok := routingFiniteNumber(value)
			min, max, _ := routingPreferenceRange(key, mode)
			if !ok || number < min || number > max {
				continue
			}
		}
		filtered[key] = value
	}
	return filtered
}

// routingPreferenceIsModeScaled reports whether a preference is expressed in a
// unit whose plausible range depends on the travel mode.
func routingPreferenceIsModeScaled(key string) bool {
	return key == "speedPreference"
}

func defaultRoutingPreferenceValue(key string, mode string) any {
	switch key {
	case "speedPreference":
		switch mode {
		case "bike":
			return 20.0
		case "motor":
			return 100.0
		default:
			return 5.1
		}
	case "hillPreference":
		return 0.5
	case "maxHikingDifficulty":
		return 6
	case "roadPreference":
		return 0.5
	case "avoidBadSurfaces":
		return 0.25
	case "vehicleWidth":
		return 1.8
	case "vehicleHeight":
		return 1.9
	default:
		return nil
	}
}

func routingPreferenceEnabled(key string, supportedPreferences map[string]bool) bool {
	return key != "" && supportedPreferences[key]
}

// routingPreferenceRange is the advertised range of a standard preference. A
// value outside it is meaningless for that mode, most notably a speed: the same
// key means km/h on foot and km/h in a car, and the two have no common scale.
func routingPreferenceRange(key string, mode string) (float64, float64, float64) {
	switch key {
	case "speedPreference":
		switch mode {
		case "foot":
			return 0.5, 25, 0.1
		case "bike":
			return 5, 50, 0.5
		case "motor":
			return 10, 300, 1
		default:
			return 0.5, 300, 0.1
		}
	case "maxHikingDifficulty":
		return 0, 6, 1
	case "vehicleWidth":
		return 1, 3, 0.1
	case "vehicleHeight":
		return 1, 3.5, 0.1
	default:
		return 0, 1, 0.05
	}
}

func routingControlForPreference(key string, value any, mode string) routingControlView {
	min, max, step := routingPreferenceRange(key, mode)
	label := key
	unit := ""
	switch key {
	case "speedPreference":
		label = "Speed"
		unit = "km/h"
	case "hillPreference":
		label = "Hill preference"
	case "maxHikingDifficulty":
		label = "Maximum hiking difficulty"
	case "roadPreference":
		label = "Road preference"
	case "avoidBadSurfaces":
		label = "Avoid bad surfaces"
	case "vehicleWidth":
		label = "Vehicle width"
		unit = "m"
	case "vehicleHeight":
		label = "Vehicle height"
		unit = "m"
	}
	return routingControlView{
		Key: key, Label: label, Unit: unit,
		Type: "number", UI: "slider", ValueType: "number",
		Min: &min, Max: &max, Step: &step, Default: value,
		Support: "partial", Comparable: true, Target: "preference", Path: []string{key},
	}
}

func manifestRoutingNativeControlGroups(ctx context.Context, app core.App, userID string, input routingNativeControlsInput) ([]routingNativeControlGroup, error) {
	if input.ProfileID != "" {
		record, err := app.FindRecordById("routing_profiles", input.ProfileID)
		if err != nil {
			return nil, apis.NewNotFoundError("routing profile not found", err)
		}
		if err := authorizeRoutingProfileRead(userID, record); err != nil {
			return nil, err
		}
		metadata := pluginsystem.JSONMapFromRecord(record, "metadata")
		nativeConfig := pluginsystem.JSONMapFromRecord(record, "native_config")
		if input.NativeConfig != nil {
			pluginsystem.DeepMergeConfig(nativeConfig, input.NativeConfig)
		}
		if rawGroups := anySlice(metadata["nativeControlGroups"]); len(rawGroups) > 0 {
			return routingNativeControlGroupsFromMetadata(rawGroups, nativeConfig), nil
		}
		if groups, err := routingProfilePluginIntrospection(ctx, app, userID, record, input); err != nil {
			return nil, err
		} else if len(groups) > 0 {
			return groups, nil
		}
		return routingNativeControlGroupsFromMetadata(anySlice(metadata["nativeControlGroups"]), pluginsystem.JSONMapFromRecord(record, "native_config")), nil
	}
	plugin, err := localPlugin(app, input.PluginID)
	if err != nil {
		return nil, err
	}
	profile := routingManifestNativeProfile(plugin.Manifest, input.NativeProfileKey)
	if len(profile) == 0 {
		return nil, nil
	}
	return routingNativeControlGroupsFromMetadata(anySlice(profile["nativeControlGroups"]), input.NativeConfig), nil
}

func routingProfilePluginIntrospection(ctx context.Context, app core.App, userID string, record *core.Record, input routingNativeControlsInput) ([]routingNativeControlGroup, error) {
	output, err := introspectRoutingProfileContext(ctx, app, userID, input.InstanceID, pluginRoutingProfile{
		ID:            record.Id,
		PluginID:      record.GetString("plugin_id"),
		Key:           record.GetString("key"),
		Kind:          record.GetString("kind"),
		Mode:          record.GetString("mode"),
		ContentBase64: record.GetString("content_base64"),
		ContentType:   record.GetString("content_type"),
		Metadata:      pluginsystem.JSONMapFromRecord(record, "metadata"),
		// Legacy discovery runs without client-supplied native values. The
		// returned control contract must establish the allowlist first.
		NativeConfig: nil,
	})
	if err != nil {
		return nil, err
	}
	return output.NativeControlGroups, nil
}

func enrichCustomRoutingProfile(app core.App, userID string, input *routingProfileView) error {
	return enrichCustomRoutingProfileContext(context.Background(), app, userID, input)
}

func enrichCustomRoutingProfileContext(ctx context.Context, app core.App, userID string, input *routingProfileView) error {
	output, err := introspectRoutingProfileContext(ctx, app, userID, "", pluginRoutingProfile{
		PluginID:      input.PluginID,
		Key:           input.Key,
		Kind:          input.Kind,
		Mode:          input.Mode,
		ContentBase64: input.ContentBase64,
		ContentType:   input.ContentType,
		Metadata:      input.Metadata,
		// Introspection establishes the provider-owned native-control allowlist;
		// untrusted native values are applied only after that contract exists.
		NativeConfig: nil,
	})
	if err != nil {
		return err
	}
	filteredNativeConfig, err := allowlistedRoutingNativeConfigFromGroups(input.NativeConfig, output.NativeControlGroups)
	if err != nil {
		return apis.NewBadRequestError("invalid routing profile nativeConfig", err)
	}
	input.NativeConfig = filteredNativeConfig
	metadata := cloneRoutingMap(input.Metadata)
	pluginsystem.DeepMergeConfig(metadata, output.Metadata)
	metadata["supportedPreferences"] = append([]string{}, output.SupportedPreferences...)
	if len(output.NativeControlGroups) > 0 {
		metadata["nativeControlGroups"] = output.NativeControlGroups
	} else {
		delete(metadata, "nativeControlGroups")
	}

	switch output.Mode {
	case "foot", "bike", "motor", "mixed":
		input.Mode = output.Mode
		metadata["modeDetection"] = "automatic"
	case "":
		if stringValue(metadata["modeDetection"]) == "manual" && isAssignableRoutingProfileMode(input.Mode) {
			metadata["modeDetection"] = "manual"
		} else {
			input.Mode = "other"
			metadata["modeDetection"] = "unresolved"
		}
	default:
		return &routingError{
			Code:       "invalid_plugin_response",
			Message:    "routing plugin returned an unsupported profile mode",
			HTTPStatus: http.StatusBadGateway,
		}
	}
	input.Metadata = metadata
	return nil
}

func allowlistedRoutingNativeConfigFromGroups(input map[string]any, groups []routingNativeControlGroup) (map[string]any, error) {
	result := map[string]any{}
	for _, group := range groups {
		for _, control := range group.Controls {
			if control.Target != "" && control.Target != "native_config" {
				continue
			}
			path := control.Path
			if len(path) == 0 {
				path = []string{control.Key}
			}
			value, found := nestedValue(input, path)
			if !found {
				continue
			}
			normalized, err := validateRoutingNativeControlViewValue(control, value)
			if err != nil {
				return nil, fmt.Errorf("control %q: %w", control.Key, err)
			}
			setNestedRoutingValue(result, path, normalized)
		}
	}
	return result, nil
}

// normalizeRoutingNativeControlValue validates a control value and returns it in
// the type the control declares. A numeric control that carries options is
// rendered as a select, so its value comes back as the option's string form;
// storing that verbatim would later fail validation and reach providers as text.
func normalizeRoutingNativeControlValue(valueType string, options []routingControlOption, min *float64, max *float64, value any) (any, error) {
	if len(options) > 0 && valueType == "" {
		valueType = "string"
	}
	switch valueType {
	case "boolean":
		if typed, ok := value.(bool); ok {
			return typed, nil
		}
		if text, ok := value.(string); ok {
			switch text {
			case "true":
				return true, nil
			case "false":
				return false, nil
			}
		}
		return nil, fmt.Errorf("must be boolean")
	case "string", "select":
		text, ok := value.(string)
		if !ok {
			if number, isNumber := routingFiniteNumber(value); isNumber {
				text = strconv.FormatFloat(number, 'f', -1, 64)
			} else {
				return nil, fmt.Errorf("must be a bounded string")
			}
		}
		if len(text) > routingConfigMaxStringBytes {
			return nil, fmt.Errorf("must be a bounded string")
		}
		if len(options) > 0 && !routingControlOptionsContainExact(options, text) {
			return nil, fmt.Errorf("is not one of the declared options")
		}
		return text, nil
	default:
		number, ok := routingFiniteNumber(value)
		if !ok {
			text, isText := value.(string)
			if !isText {
				return nil, fmt.Errorf("must be a finite number")
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
			if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
				return nil, fmt.Errorf("must be a finite number")
			}
			number = parsed
		}
		if len(options) > 0 && !routingControlOptionsContain(options, strconv.FormatFloat(number, 'f', -1, 64)) {
			return nil, fmt.Errorf("is not one of the declared options")
		}
		if min != nil && number < *min {
			return nil, fmt.Errorf("must be at least %g", *min)
		}
		if max != nil && number > *max {
			return nil, fmt.Errorf("must be at most %g", *max)
		}
		return number, nil
	}
}

func routingControlOptionsContainExact(options []routingControlOption, value string) bool {
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

// routingControlOptionsContain compares numerically when both sides are numbers,
// so "3" and "3.0" match the same option.
func routingControlOptionsContain(options []routingControlOption, value string) bool {
	candidate, candidateIsNumber := routingFiniteNumber(value)
	if !candidateIsNumber {
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			candidate, candidateIsNumber = parsed, true
		}
	}
	for _, option := range options {
		if option.Value == value {
			return true
		}
		if !candidateIsNumber {
			continue
		}
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(option.Value), 64); err == nil && parsed == candidate {
			return true
		}
	}
	return false
}

func validateRoutingNativeControlViewValue(control routingControlView, value any) (any, error) {
	return normalizeRoutingNativeControlValue(
		fallbackString(control.ValueType, control.Type), control.Options, control.Min, control.Max, value,
	)
}

func isAssignableRoutingProfileMode(mode string) bool {
	switch mode {
	case "foot", "bike", "motor", "mixed":
		return true
	default:
		return false
	}
}

func introspectRoutingProfile(app core.App, userID string, profile pluginRoutingProfile) (pluginRoutingProfileIntrospectOutput, error) {
	return introspectRoutingProfileContext(context.Background(), app, userID, "", profile)
}

func introspectRoutingProfileContext(ctx context.Context, app core.App, userID string, requestedInstanceID string, profile pluginRoutingProfile) (pluginRoutingProfileIntrospectOutput, error) {
	plugin, err := localPlugin(app, profile.PluginID)
	if err != nil {
		return pluginRoutingProfileIntrospectOutput{}, err
	}
	capability, err := pluginCapability(plugin, "profile_introspect", "v1")
	if err != nil {
		return pluginRoutingProfileIntrospectOutput{}, nil
	}
	instance, err := routingEnabledPluginInstance(app, userID, plugin.Manifest.ID, requestedInstanceID)
	if err != nil {
		return pluginRoutingProfileIntrospectOutput{}, err
	}
	auth := map[string]any{}
	if instance != nil {
		auth, err = decryptedInstanceAuth(instance)
		if err != nil {
			return pluginRoutingProfileIntrospectOutput{}, err
		}
	}
	config := effectiveRoutingPluginConfig(app, plugin, instance)
	output, err := callRoutingProfileIntrospectPlugin(ctx, plugin, capability, instance, auth, config, pluginRoutingProfileIntrospectRequest{
		Profile: profile,
	})
	if err != nil {
		return pluginRoutingProfileIntrospectOutput{}, err
	}
	if output.Error != nil {
		return pluginRoutingProfileIntrospectOutput{}, routingErrorFromPluginError(*output.Error)
	}
	return output, nil
}

func authorizeRoutingProfileRead(userID string, record *core.Record) error {
	if record.GetString("scope") == routingScopeUser && record.GetString("user") != userID {
		return apis.NewNotFoundError("routing profile not found", nil)
	}
	return nil
}

func routingProfileInUse(app core.App, profileID string) (bool, error) {
	records, err := app.FindRecordsByFilter(
		"routing_profile_mappings",
		"profile={:profile}",
		"",
		1,
		0,
		dbx.Params{"profile": profileID},
	)
	if err != nil {
		return false, err
	}
	return len(records) > 0, nil
}

func routingManifestNativeProfile(manifest pluginsystem.Manifest, profileKey string) map[string]any {
	routingMetadata := mapValue(manifest.Metadata["routing"])
	for _, raw := range anySlice(routingMetadata["nativeProfiles"]) {
		profile := mapValue(raw)
		if stringValue(profile["key"]) == profileKey {
			return profile
		}
	}
	return nil
}

func routingNativeControlGroupsFromMetadata(raw []any, nativeConfig map[string]any) []routingNativeControlGroup {
	if len(raw) == 0 {
		return nil
	}
	groups := make([]routingNativeControlGroup, 0, len(raw))
	for _, rawGroup := range raw {
		groupMetadata := mapValue(rawGroup)
		key := stringValue(groupMetadata["key"])
		if key == "" {
			continue
		}
		controls := routingControlsFromMetadata(anySlice(groupMetadata["controls"]), nativeConfig)
		if len(controls) == 0 {
			continue
		}
		groups = append(groups, routingNativeControlGroup{
			Key:      key,
			Label:    fallbackString(stringValue(groupMetadata["label"]), key),
			Labels:   stringMap(groupMetadata["labels"]),
			Controls: controls,
		})
	}
	return groups
}

func routingControlsFromMetadata(raw []any, nativeConfig map[string]any) []routingControlView {
	if len(raw) == 0 {
		return nil
	}
	controls := make([]routingControlView, 0, len(raw))
	for _, rawControl := range raw {
		metadata := mapValue(rawControl)
		key := stringValue(metadata["key"])
		if key == "" {
			continue
		}
		path := stringSlice(metadata["path"])
		if len(path) == 0 {
			path = []string{key}
		}
		current, ok := nestedValue(nativeConfig, path)
		if !ok {
			current = metadata["default"]
		}
		control := routingControlView{
			Key:       key,
			Label:     fallbackString(stringValue(metadata["label"]), key),
			Labels:    stringMap(metadata["labels"]),
			Unit:      strings.TrimSpace(stringValue(metadata["unit"])),
			Type:      fallbackString(stringValue(metadata["type"]), fallbackString(stringValue(metadata["valueType"]), "number")),
			UI:        stringValue(metadata["ui"]),
			ValueType: fallbackString(stringValue(metadata["valueType"]), stringValue(metadata["type"])),
			Default:   metadata["default"],
			Current:   current,
			Target:    fallbackString(stringValue(metadata["target"]), "native_config"),
			Path:      path,
			Options:   routingControlOptions(metadata["options"]),
		}
		if control.ValueType == "" {
			control.ValueType = control.Type
		}
		numericOptions := len(control.Options) > 0 && (control.Type == "number" || control.ValueType == "number")
		if control.Type == "boolean" || control.ValueType == "boolean" {
			control.Type = "boolean"
			control.ValueType = "boolean"
		} else if control.Type == "string" || control.ValueType == "string" || len(control.Options) > 0 {
			control.Type = "string"
			control.ValueType = "string"
			control.UI = fallbackString(control.UI, "select")
			if numericOptions {
				control.Default = routingControlPresentationOptionValue(control.Options, control.Default)
				control.Current = routingControlPresentationOptionValue(control.Options, control.Current)
			}
		} else {
			control.Type = fallbackString(control.Type, "number")
			control.UI = fallbackString(control.UI, "slider")
			if min, ok := numericMetadata(metadata["min"]); ok {
				control.Min = &min
			}
			if max, ok := numericMetadata(metadata["max"]); ok {
				control.Max = &max
			}
			if step, ok := numericMetadata(metadata["step"]); ok {
				control.Step = &step
			}
		}
		controls = append(controls, control)
	}
	return controls
}

// routingControlPresentationOptionValue returns the declared option spelling
// used by select widgets. Provider metadata may declare a numeric control while
// option values are necessarily strings; leaving default/current numeric makes
// an otherwise valid selection appear empty in the browser. Validation still
// uses the provider's raw valueType and converts the selected string back to a
// number before it reaches the provider.
func routingControlPresentationOptionValue(options []routingControlOption, value any) any {
	if value == nil || len(options) == 0 {
		return value
	}
	if text, ok := value.(string); ok {
		for _, option := range options {
			if option.Value == text {
				return option.Value
			}
		}
	}
	number, isNumber := routingFiniteNumber(value)
	if !isNumber {
		if text, ok := value.(string); ok {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
			if err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
				number, isNumber = parsed, true
			}
		}
	}
	if isNumber {
		for _, option := range options {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(option.Value), 64)
			if err == nil && parsed == number {
				return option.Value
			}
		}
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return value
}

func nestedValue(values map[string]any, path []string) (any, bool) {
	var current any = values
	for _, key := range path {
		mapped := mapValue(current)
		if len(mapped) == 0 {
			return nil, false
		}
		next, ok := mapped[key]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}

func allowlistedRoutingNativeConfig(input map[string]any, rawGroups []any) (map[string]any, error) {
	result := map[string]any{}
	if len(input) == 0 || len(rawGroups) == 0 {
		return result, nil
	}
	for _, rawGroup := range rawGroups {
		for _, rawControl := range anySlice(mapValue(rawGroup)["controls"]) {
			metadata := mapValue(rawControl)
			if fallbackString(stringValue(metadata["target"]), "native_config") != "native_config" {
				continue
			}
			path := stringSlice(metadata["path"])
			if len(path) == 0 {
				if key := strings.TrimSpace(stringValue(metadata["key"])); key != "" {
					path = []string{key}
				}
			}
			if len(path) == 0 || len(path) > routingConfigMaxDepth {
				continue
			}
			value, found := nestedValue(input, path)
			if !found {
				continue
			}
			normalized, err := validateRoutingNativeControlValue(metadata, value)
			if err != nil {
				return nil, fmt.Errorf("control %q: %w", stringValue(metadata["key"]), err)
			}
			setNestedRoutingValue(result, path, normalized)
		}
	}
	return result, nil
}

func validateRoutingNativeControlValue(metadata map[string]any, value any) (any, error) {
	var min, max *float64
	if number, found := numericMetadata(metadata["min"]); found {
		min = &number
	}
	if number, found := numericMetadata(metadata["max"]); found {
		max = &number
	}
	return normalizeRoutingNativeControlValue(
		fallbackString(stringValue(metadata["valueType"]), stringValue(metadata["type"])),
		routingControlOptions(metadata["options"]), min, max, value,
	)
}

func setNestedRoutingValue(target map[string]any, path []string, value any) {
	current := target
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	current[path[len(path)-1]] = value
}

func numericMetadata(metadata any) (float64, bool) {
	switch value := metadata.(type) {
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case float64:
		return value, true
	case float32:
		return float64(value), true
	default:
		return 0, false
	}
}

func mapValue(value any) map[string]any {
	if value, ok := value.(map[string]any); ok && value != nil {
		return value
	}
	return map[string]any{}
}

func cloneRoutingMap(input map[string]any) map[string]any {
	output := map[string]any{}
	for key, value := range input {
		if nested, ok := value.(map[string]any); ok {
			output[key] = cloneRoutingMap(nested)
			continue
		}
		output[key] = value
	}
	return output
}

func stringValue(value any) string {
	if value, ok := value.(string); ok {
		return value
	}
	return ""
}

func intValue(value any) int {
	switch value := value.(type) {
	case int:
		return value
	case float64:
		return int(value)
	case float32:
		return int(value)
	default:
		return 0
	}
}

func floatValue(value any) float64 {
	switch value := value.(type) {
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case float64:
		return value
	case float32:
		return float64(value)
	default:
		return 0
	}
}

func stringSlice(value any) []string {
	items := anySlice(value)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if s := stringValue(item); s != "" {
			result = append(result, s)
		}
	}
	return result
}

func anySlice(value any) []any {
	switch value := value.(type) {
	case []any:
		return value
	case []map[string]any:
		result := make([]any, len(value))
		for i, item := range value {
			result[i] = item
		}
		return result
	case []string:
		result := make([]any, len(value))
		for i, item := range value {
			result[i] = item
		}
		return result
	default:
		return nil
	}
}

func stringMap(value any) map[string]string {
	raw := mapValue(value)
	result := map[string]string{}
	for key, item := range raw {
		if s := stringValue(item); s != "" {
			result[key] = s
		}
	}
	return result
}

func stringMapSlice(value any) []map[string]string {
	if typed, ok := value.([]map[string]string); ok {
		return typed
	}
	if typed, ok := value.([]map[string]any); ok {
		result := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			mapped := stringMap(item)
			if len(mapped) > 0 {
				result = append(result, mapped)
			}
		}
		return result
	}
	items := anySlice(value)
	result := make([]map[string]string, 0, len(items))
	for _, item := range items {
		mapped := stringMap(item)
		if len(mapped) > 0 {
			result = append(result, mapped)
		}
	}
	return result
}

func routingControlOptions(value any) []routingControlOption {
	items := anySlice(value)
	result := make([]routingControlOption, 0, len(items))
	for _, item := range items {
		metadata := mapValue(item)
		optionValue := stringValue(metadata["value"])
		if optionValue == "" {
			continue
		}
		result = append(result, routingControlOption{
			Value:  optionValue,
			Label:  fallbackString(stringValue(metadata["label"]), optionValue),
			Labels: stringMap(metadata["labels"]),
		})
	}
	return result
}

func boolMap(value any) map[string]bool {
	raw := mapValue(value)
	result := map[string]bool{}
	for key, item := range raw {
		if b, ok := item.(bool); ok {
			result[key] = b
		}
	}
	return result
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func titleWords(value string) string {
	parts := strings.Fields(value)
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func routingScopeRank(scope string) int {
	switch scope {
	case routingScopeBuiltin:
		return 0
	case routingScopeAdmin:
		return 1
	case routingScopeUser:
		return 2
	default:
		return 3
	}
}
