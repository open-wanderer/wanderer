//go:build tinygo

package main

import (
	"time"
)

func changedAfter(changedAt string, after string) bool {
	if after == "" {
		return true
	}
	limit, err := time.Parse("2006-01-02", after)
	if err != nil {
		return true
	}
	changed, err := time.Parse(time.RFC3339, changedAt)
	if err != nil {
		return true
	}
	return !changed.Before(limit)
}
