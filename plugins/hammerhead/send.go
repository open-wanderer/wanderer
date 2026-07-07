package main

import "strings"

func trailGPXFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "trail.gpx"
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '/' || r == '\\' {
			return '-'
		}
		return r
	}, name)
	name = strings.Trim(name, ". -")
	if name == "" {
		return "trail.gpx"
	}
	if strings.HasSuffix(strings.ToLower(name), ".gpx") {
		return name
	}
	return name + ".gpx"
}
