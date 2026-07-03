package pluginhost

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	"pocketbase/util"
)

var ErrMissingEncryptionKey = errors.New("POCKETBASE_ENCRYPTION_KEY not set")

func LocalPlugin(app core.App, pluginID string) (pluginsystem.LocalPlugin, error) {
	plugin, err := pluginsystem.LoadInstalledPlugin(app, "", pluginID)
	if err != nil {
		return pluginsystem.LocalPlugin{}, fmt.Errorf("unknown plugin: %w", err)
	}
	return plugin, nil
}

func EffectiveConfig(app core.App, pluginID string, instance *core.Record) map[string]any {
	config := InstalledConfig(app, pluginID)
	if instance != nil {
		pluginsystem.MergePluginConfig(config, pluginsystem.JSONMapFromRecord(instance, "config"))
	}
	return config
}

func RuntimeConfig(config map[string]any) map[string]any {
	return configSection(config, "plugin")
}

func HostConfig(config map[string]any) map[string]any {
	return configSection(config, "host")
}

func InstalledConfig(app core.App, pluginID string) map[string]any {
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

func InstancePolicy(plugin pluginsystem.LocalPlugin, config map[string]any) pluginsystem.RequestPolicyContext {
	connectors := map[string]pluginsystem.ResolvedConnectorTarget{}
	hostConfig := HostConfig(config)
	hostConnectors := util.ConfigMap(hostConfig, "connectors")

	for _, manifestConnector := range plugin.Manifest.Permissions.Network.Connectors {
		target, err := resolveConnectorTarget(manifestConnector, hostConnectors)
		if err != nil {
			continue
		}
		connectors[manifestConnector.Name] = target
	}

	return pluginsystem.RequestPolicyContext{Connectors: connectors}
}

func AssetConnectorConfig(plugin pluginsystem.LocalPlugin, config map[string]any) map[string]any {
	pluginConfig := RuntimeConfig(config)
	urlValue, _ := pluginConfig["url"].(string)
	if urlValue == "" {
		return config
	}
	hostConfig := HostConfig(config)
	hostConnectors, _ := hostConfig["connectors"].(map[string]any)
	if hostConnectors == nil {
		hostConnectors = map[string]any{}
		hostConfig["connectors"] = hostConnectors
	}
	for _, connector := range plugin.Manifest.Permissions.Network.Connectors {
		if connector.Type != pluginsystem.ConnectorTypeConfigured || connector.ConfigKey == "" {
			continue
		}
		rawConnector, _ := hostConnectors[connector.ConfigKey].(map[string]any)
		if rawConnector == nil {
			rawConnector = map[string]any{}
			hostConnectors[connector.ConfigKey] = rawConnector
		}
		if existing, _ := rawConnector["baseURL"].(string); existing == "" {
			rawConnector["baseURL"] = urlValue
		}
	}
	config["host"] = hostConfig
	return config
}

func DecryptedInstanceAuth(instance *core.Record) (map[string]any, error) {
	auth := pluginsystem.JSONMapFromRecord(instance, "auth")
	if len(auth) == 0 {
		return map[string]any{}, nil
	}
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if encryptionKey == "" {
		return nil, ErrMissingEncryptionKey
	}
	for key, value := range auth {
		secret, ok := value.(string)
		if !ok || secret == "" || !util.CanDecryptSecret(secret) {
			continue
		}
		decrypted, err := security.Decrypt(secret, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", key, err)
		}
		auth[key] = string(decrypted)
	}
	return auth, nil
}

func PhotoImportLimits(hostConfig map[string]any) importer.PhotoImportLimits {
	return importer.PhotoImportLimits{
		MaxPhotosPerTrail:     util.PositiveConfigInt(hostConfig, "maxPhotosPerTrail", util.DefaultPluginMaxPhotosPerTrail),
		MaxPhotosPerWaypoint:  util.PositiveConfigInt(hostConfig, "maxPhotosPerWaypoint", util.DefaultPluginMaxPhotosPerWaypoint),
		MaxPhotosPerSummitLog: util.PositiveConfigInt(hostConfig, "maxPhotosPerSummitLog", util.DefaultPluginMaxPhotosPerSummitLog),
	}
}

func configSection(config map[string]any, key string) map[string]any {
	raw, ok := config[key].(map[string]any)
	if !ok || raw == nil {
		return map[string]any{}
	}
	return raw
}

func resolveConnectorTarget(manifest pluginsystem.ConnectorTargetPermission, hostConnectors map[string]any) (pluginsystem.ResolvedConnectorTarget, error) {
	target := pluginsystem.ResolvedConnectorTarget{
		Name:                     manifest.Name,
		Type:                     manifest.Type,
		AllowedPathPrefixes:      manifest.AllowedPathPrefixes,
		Auth:                     manifest.Auth,
		SupportsMediaAuth:        manifest.SupportsMediaAuth,
		SupportsStorageRedirects: manifest.SupportsStorageRedirects,
		SupportsCustomTLS:        manifest.SupportsCustomTLS,
		TLS:                      pluginsystem.ConnectorTLSConfig{Mode: pluginsystem.TLSModeSystem},
		StorageOrigins:           map[string]pluginsystem.ResolvedConnectorOrigin{},
	}

	switch manifest.Type {
	case pluginsystem.ConnectorTypePublicAPI:
		baseURL, basePath, err := pluginsystem.NormalizeConnectorBase(manifest.FixedBaseURL, "")
		if err != nil {
			return target, err
		}
		target.BaseURL = baseURL
		target.BasePath = basePath
		target.AllowPrivate = false
	case pluginsystem.ConnectorTypeConfigured:
		rawConfig := util.ConfigMap(hostConnectors, manifest.ConfigKey)
		if len(rawConfig) == 0 {
			return target, fmt.Errorf("configured connector %q has no host config", manifest.Name)
		}
		baseURL := util.ConfigString(rawConfig, "baseURL")
		basePath := util.ConfigString(rawConfig, "basePath")
		normalizedBaseURL, normalizedBasePath, err := pluginsystem.NormalizeConnectorBase(baseURL, basePath)
		if err != nil {
			return target, err
		}
		target.BaseURL = normalizedBaseURL
		target.BasePath = normalizedBasePath
		target.AllowPrivate = util.ConfigBool(rawConfig, "allowPrivate", false)
		target.TLS = tlsConfig(rawConfig, manifest.SupportsCustomTLS)
		if manifest.SupportsStorageRedirects {
			target.StorageOrigins = storageOrigins(rawConfig)
		}
	default:
		return target, fmt.Errorf("unsupported connector type %q", manifest.Type)
	}
	return target, nil
}

func storageOrigins(rawConfig map[string]any) map[string]pluginsystem.ResolvedConnectorOrigin {
	rawOrigins := util.ConfigMap(rawConfig, "storageOrigins")
	origins := map[string]pluginsystem.ResolvedConnectorOrigin{}
	for name, raw := range rawOrigins {
		originMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		baseURL, basePath, err := pluginsystem.NormalizeConnectorBase(
			util.ConfigString(originMap, "baseURL"),
			util.ConfigString(originMap, "basePath"),
		)
		if err != nil {
			continue
		}
		origins[name] = pluginsystem.ResolvedConnectorOrigin{
			Name:         name,
			BaseURL:      baseURL,
			BasePath:     basePath,
			AllowPrivate: util.ConfigBool(originMap, "allowPrivate", false),
			TLS:          tlsConfig(originMap, true),
		}
	}
	return origins
}

func tlsConfig(raw map[string]any, customAllowed bool) pluginsystem.ConnectorTLSConfig {
	rawTLS := util.ConfigMap(raw, "tls")
	mode := util.ConfigString(rawTLS, "mode")
	if mode == "" {
		mode = pluginsystem.TLSModeSystem
	}
	if mode != pluginsystem.TLSModeSystem && mode != pluginsystem.TLSModeCustomCA {
		mode = pluginsystem.TLSModeSystem
	}
	if !customAllowed && mode != pluginsystem.TLSModeSystem {
		mode = pluginsystem.TLSModeSystem
	}
	cfg := pluginsystem.ConnectorTLSConfig{Mode: mode}
	if mode == pluginsystem.TLSModeCustomCA {
		ca := util.ConfigString(rawTLS, "caBundle")
		if decoded, err := base64.StdEncoding.DecodeString(ca); err == nil {
			cfg.CABundle = decoded
		} else {
			cfg.CABundle = []byte(ca)
		}
	}
	return cfg
}
