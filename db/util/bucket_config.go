package util

import (
	"os"
	"strings"
)

func BucketsEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_TRAIL_BUCKETS")))
	return value == "true" || value == "1" || value == "yes"
}
