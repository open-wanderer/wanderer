//go:build tinygo

package main

import (
	"strconv"
	"strings"
	"time"
)

func stringField(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func intState(state map[string]any, key string, fallback int) int {
	switch value := state[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func knownIDs(ids []string) map[string]bool {
	known := make(map[string]bool, len(ids))
	for _, id := range ids {
		known[id] = true
	}
	return known
}

func limit(input listInput) int {
	if input.Limits.MaxItems > 0 {
		return input.Limits.MaxItems
	}
	return 10
}

func dateOption(options map[string]any, key string) string {
	value, _ := options[key].(string)
	return strings.TrimSpace(value)
}

func unixAfter(options map[string]any) int64 {
	after := dateOption(options, "after")
	if after == "" {
		return 0
	}
	parsed, err := time.Parse("2006-01-02", after)
	if err != nil {
		return 0
	}
	return parsed.UTC().Unix()
}

func timeAfterDate(value string, after string) bool {
	if after == "" {
		return true
	}
	limit, err := time.Parse("2006-01-02", after)
	if err != nil {
		return true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return true
	}
	return !parsed.Before(limit)
}
