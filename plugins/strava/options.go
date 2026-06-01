//go:build tinygo

package main

import (
	"strings"
	"time"
)

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
