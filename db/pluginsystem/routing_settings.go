package pluginsystem

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	routingSettingsScopeBuiltin = "builtin"
	routingSettingsScopeAdmin   = "admin"
	routingSettingsScopeUser    = "user"
)

// RoutingSettingsLayers contains the single deterministic settings record for
// each scope in builtin-to-user precedence order.
type RoutingSettingsLayers struct {
	Builtin map[string]any
	Admin   map[string]any
	User    map[string]any
}

// RoutingSettingsLayer keeps the policy scope attached to its config while it
// is merged. Scope-specific rules must not depend on a layer's list position.
type RoutingSettingsLayer struct {
	Scope  string
	Config map[string]any
}

// Configs returns the scoped layers in effective merge order.
func (l RoutingSettingsLayers) Configs() []RoutingSettingsLayer {
	return []RoutingSettingsLayer{
		{Scope: routingSettingsScopeBuiltin, Config: l.Builtin},
		{Scope: routingSettingsScopeAdmin, Config: l.Admin},
		{Scope: routingSettingsScopeUser, Config: l.User},
	}
}

// RoutingPluginIsSelected reports whether pluginID is referenced by one of the
// user's active editor-routing features. Editor routing protects the primary
// route and elevation selections. Additional variant engines and maneuver
// engines are resolved from enabled instances and therefore do not need
// selection protection.
func RoutingPluginIsSelected(app core.App, userID string, pluginID string) (bool, error) {
	userID = strings.TrimSpace(userID)
	pluginID = strings.TrimSpace(pluginID)
	if userID == "" || pluginID == "" {
		return false, nil
	}
	settings, err := ResolveEffectiveRoutingSettings(app, userID)
	if err != nil {
		return false, err
	}
	features, _ := settings["exposedFeatures"].(map[string]any)
	routingEnabled := true
	if enabled, defined := features["routing"].(bool); defined {
		routingEnabled = enabled
	}
	if routingEnabled {
		if strings.TrimSpace(stringFromJSON(settings["primaryRoutePluginId"])) == pluginID ||
			strings.TrimSpace(stringFromJSON(settings["elevationPluginId"])) == pluginID {
			return true, nil
		}
	}
	return false, nil
}

// ResolveEffectiveRoutingSettings loads and merges routing settings once for
// all consumers, including HTTP handlers and plugin lifecycle guards.
func ResolveEffectiveRoutingSettings(app core.App, userID string) (map[string]any, error) {
	layers, err := ResolveRoutingSettingsLayers(app, userID)
	if err != nil {
		return nil, err
	}
	return MergeRoutingSettingsLayers(layers), nil
}

// MergeRoutingSettingsLayers applies the routing settings precedence and merge
// policy. Scalars use the nearest defined value, default preferences
// deep-merge, and feature flags are upper bounds across scopes. Navigation is
// a host policy and therefore ignores user-scoped values.
func MergeRoutingSettingsLayers(layers RoutingSettingsLayers) map[string]any {
	effective := map[string]any{}
	features := map[string]any{}
	for _, layer := range layers.Configs() {
		config := layer.Config
		for _, key := range []string{"primaryRoutePluginId", "elevationPluginId", "maneuverPluginId", "defaultRoutingMode"} {
			if value := stringFromJSON(config[key]); value != "" {
				effective[key] = value
			}
		}
		if raw, exists := config["defaultVariantCount"]; exists && routingJSONNumberNonZero(raw) {
			effective["defaultVariantCount"] = CloneJSONValue(raw)
		}
		if preferences, ok := config["defaultPreferences"].(map[string]any); ok {
			effectivePreferences, _ := effective["defaultPreferences"].(map[string]any)
			if effectivePreferences == nil {
				effectivePreferences = map[string]any{}
				effective["defaultPreferences"] = effectivePreferences
			}
			DeepMergeConfig(effectivePreferences, preferences)
		}
		if featureLayer, ok := config["exposedFeatures"].(map[string]any); ok {
			for key, raw := range featureLayer {
				if layer.Scope == routingSettingsScopeUser && key == "navigation" {
					continue
				}
				enabled, ok := raw.(bool)
				if !ok {
					continue
				}
				if current, defined := features[key].(bool); defined {
					features[key] = current && enabled
				} else {
					features[key] = enabled
				}
			}
		}
	}
	if len(features) > 0 {
		effective["exposedFeatures"] = features
	}
	return effective
}

func routingJSONNumberNonZero(value any) bool {
	switch value := value.(type) {
	case int:
		return value != 0
	case int64:
		return value != 0
	case float32:
		return value != 0
	case float64:
		return value != 0
	default:
		return false
	}
}

// ResolveRoutingSettingsLayers is the shared scope resolver for routing policy
// and HTTP settings. Duplicate records within a scope are resolved
// deterministically to the lexicographically first record ID.
func ResolveRoutingSettingsLayers(app core.App, userID string) (RoutingSettingsLayers, error) {
	builtin, err := RoutingSettingsRecordForScope(app, routingSettingsScopeBuiltin, "")
	if err != nil {
		return RoutingSettingsLayers{}, err
	}
	admin, err := RoutingSettingsRecordForScope(app, routingSettingsScopeAdmin, "")
	if err != nil {
		return RoutingSettingsLayers{}, err
	}
	user, err := RoutingSettingsRecordForScope(app, routingSettingsScopeUser, userID)
	if err != nil {
		return RoutingSettingsLayers{}, err
	}
	return RoutingSettingsLayers{
		Builtin: routingSettingsRecordConfig(builtin),
		Admin:   routingSettingsRecordConfig(admin),
		User:    routingSettingsRecordConfig(user),
	}, nil
}

// RoutingSettingsRecordForScope returns the same deterministic record used by
// ResolveRoutingSettingsLayers so reads and writes cannot diverge on duplicate
// legacy rows.
func RoutingSettingsRecordForScope(app core.App, scope string, userID string) (*core.Record, error) {
	filter := "scope={:scope}"
	params := dbx.Params{"scope": scope}
	if scope == routingSettingsScopeUser {
		filter += " && user={:user}"
		params["user"] = userID
	}
	records, err := app.FindRecordsByFilter("routing_settings", filter, "+id", 1, 0, params)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func routingSettingsRecordConfig(record *core.Record) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	return JSONMapFromRecord(record, "config")
}

func stringFromJSON(value any) string {
	valueString, _ := value.(string)
	return valueString
}
