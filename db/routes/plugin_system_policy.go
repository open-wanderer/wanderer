package routes

import (
	"net/url"
	"strings"

	"pocketbase/pluginsystem"
)

func pluginInstancePolicy(plugin pluginsystem.LocalPlugin, config map[string]any) pluginsystem.RequestPolicyContext {
	origins := make([]string, 0, len(plugin.Manifest.Permissions.Network.UserConfiguredOrigins))
	for _, key := range plugin.Manifest.Permissions.Network.UserConfiguredOrigins {
		value, ok := config[key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		origin := normalizePluginOrigin(value)
		if origin != "" {
			origins = append(origins, origin)
		}
	}
	return pluginsystem.RequestPolicyContext{UserConfiguredOrigins: origins}
}

func normalizePluginOrigin(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
