package util

import "strings"

func ConfigMap(raw map[string]any, key string) map[string]any {
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

func ConfigString(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return strings.TrimSpace(value)
}

func ConfigBool(raw map[string]any, key string, fallback bool) bool {
	value, ok := raw[key].(bool)
	if !ok {
		return fallback
	}
	return value
}
