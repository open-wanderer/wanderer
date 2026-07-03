package main

import (
	"strings"
	"time"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

func dateOption(options map[string]any, key string) string {
	value, _ := options[key].(string)
	return strings.TrimSpace(value)
}

func getOldestDate(input listInput) string {
	// 1. Try to get from State
	if stateOldest := sdk.StringField(input.State, "oldest"); stateOldest != "" {
		return stateOldest
	}

	// 2. Try to get from Options ("after")
	if after := dateOption(input.Options, "after"); after != "" {
		return after
	}

	// 3. Fallback to a default long ago
	return "2000-01-01"
}

func timeAfterDate(value string, after string) bool {
	if after == "" {
		return true
	}
	limit, err := time.Parse("2006-01-02", after)
	if err != nil {
		return true
	}
	// Parse activity start date (which could be RFC3339 or similar)
	var parsed time.Time
	if strings.Contains(value, "T") {
		if strings.HasSuffix(value, "Z") {
			parsed, err = time.Parse(time.RFC3339, value)
		} else {
			// ISO-8601 local usually
			parsed, err = time.Parse("2006-01-02T15:04:05", value)
		}
	} else {
		parsed, err = time.Parse("2006-01-02", value)
	}
	if err != nil {
		return true
	}
	return !parsed.Before(limit)
}
