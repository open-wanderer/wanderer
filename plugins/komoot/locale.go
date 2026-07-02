package main

import "strings"

func acceptLanguage(locale string) string {
	locale = strings.TrimSpace(locale)
	if locale == "" {
		return ""
	}
	primary := locale
	if index := strings.IndexAny(primary, "_-"); index >= 0 {
		primary = primary[:index]
	}
	locale = strings.ReplaceAll(locale, "_", "-")
	if primary == "" || primary == locale {
		return locale
	}
	return locale + "," + primary + ";q=0.9"
}
