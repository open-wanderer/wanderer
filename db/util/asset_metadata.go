package util

import "strings"

func AssetMetadataString(metadata map[string]any, path ...string) string {
	var current any = metadata
	for _, part := range path {
		values, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = values[part]
	}
	value, _ := current.(string)
	return strings.TrimSpace(value)
}
