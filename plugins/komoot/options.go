package main

import (
	"time"
)

func tourDateAfter(tourDate string, after string) bool {
	if after == "" {
		return true
	}
	limit, err := time.Parse("2006-01-02", after)
	if err != nil {
		return true
	}
	date, err := parseKomootDate(tourDate)
	if err != nil {
		return true
	}
	return !date.Before(limit)
}

func parseKomootDate(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02", value)
}
