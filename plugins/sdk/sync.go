package sdk

import (
	"encoding/json"
	"strconv"
	"strings"
)

func StringField(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func StringOption(options map[string]any, key string) string {
	return StringField(options, key)
}

func IntState(state map[string]any, key string, fallback int) int {
	switch value := state[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func KnownIDs(ids []string) map[string]bool {
	known := make(map[string]bool, len(ids))
	for _, id := range ids {
		known[id] = true
	}
	return known
}

func SyncLimit(input ListInput) int {
	if input.Limits.MaxItems > 0 {
		return input.Limits.MaxItems
	}
	return 10
}

func NextPageState(nextPage int, hasMore bool) map[string]any {
	if !hasMore {
		return nil
	}
	return map[string]any{"page": nextPage}
}
