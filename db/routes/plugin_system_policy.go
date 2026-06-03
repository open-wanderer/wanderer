package routes

import (
	"encoding/base64"
	"fmt"
	"strings"

	"pocketbase/pluginsystem"
)

func pluginInstancePolicy(plugin pluginsystem.LocalPlugin, config map[string]any) pluginsystem.RequestPolicyContext {
	connectors := map[string]pluginsystem.ResolvedConnectorTarget{}
	hostConfig := pluginHostConfig(config)
	hostConnectors := configMap(configMap(hostConfig, "connectors"), "")

	for _, manifestConnector := range plugin.Manifest.Permissions.Network.Connectors {
		target, err := resolveConnectorTarget(manifestConnector, hostConnectors)
		if err != nil {
			continue
		}
		connectors[manifestConnector.Name] = target
	}

	return pluginsystem.RequestPolicyContext{Connectors: connectors}
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
		rawConfig := configMap(hostConnectors, manifest.ConfigKey)
		if len(rawConfig) == 0 {
			return target, fmt.Errorf("configured connector %q has no host config", manifest.Name)
		}
		baseURL := stringConfig(rawConfig, "baseURL")
		basePath := stringConfig(rawConfig, "basePath")
		normalizedBaseURL, normalizedBasePath, err := pluginsystem.NormalizeConnectorBase(baseURL, basePath)
		if err != nil {
			return target, err
		}
		target.BaseURL = normalizedBaseURL
		target.BasePath = normalizedBasePath
		target.AllowPrivate = boolConfig(rawConfig, "allowPrivate")
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
	rawOrigins := configMap(rawConfig, "storageOrigins")
	origins := map[string]pluginsystem.ResolvedConnectorOrigin{}
	for name, raw := range rawOrigins {
		originMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		baseURL, basePath, err := pluginsystem.NormalizeConnectorBase(
			stringConfig(originMap, "baseURL"),
			stringConfig(originMap, "basePath"),
		)
		if err != nil {
			continue
		}
		origins[name] = pluginsystem.ResolvedConnectorOrigin{
			Name:         name,
			BaseURL:      baseURL,
			BasePath:     basePath,
			AllowPrivate: boolConfig(originMap, "allowPrivate"),
			TLS:          tlsConfig(originMap, true),
		}
	}
	return origins
}

func tlsConfig(raw map[string]any, customAllowed bool) pluginsystem.ConnectorTLSConfig {
	rawTLS := configMap(raw, "tls")
	mode := stringConfig(rawTLS, "mode")
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
		ca := stringConfig(rawTLS, "caBundle")
		if decoded, err := base64.StdEncoding.DecodeString(ca); err == nil {
			cfg.CABundle = decoded
		} else {
			cfg.CABundle = []byte(ca)
		}
	}
	return cfg
}

func configMap(raw map[string]any, key string) map[string]any {
	if key == "" {
		return raw
	}
	value, ok := raw[key]
	if !ok {
		return map[string]any{}
	}
	switch typed := value.(type) {
	case map[string]any:
		return typed
	default:
		return map[string]any{}
	}
}

func stringConfig(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return strings.TrimSpace(value)
}

func boolConfig(raw map[string]any, key string) bool {
	value, _ := raw[key].(bool)
	return value
}
