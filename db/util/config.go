package util

import (
	"encoding/json"
	"strconv"
	"strings"
)

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

func ConfigInt(raw map[string]any, key string, fallback int) int {
	switch value := raw[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case int32:
		return int(value)
	case float64:
		return int(value)
	case float32:
		return int(value)
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func PositiveConfigInt(raw map[string]any, key string, fallback int) int {
	value := ConfigInt(raw, key, fallback)
	if value <= 0 {
		return fallback
	}
	return value
}
