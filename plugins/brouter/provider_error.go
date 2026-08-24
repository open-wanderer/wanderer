package main

import "strings"

func brouterProviderErrorCode(status int, body []byte, clientCode string) string {
	message := strings.ToLower(string(body))
	if strings.Contains(message, "thread-priority-watchdog") ||
		strings.Contains(message, " timeout after ") {
		return "provider_timeout"
	}
	if status >= 400 && status < 500 {
		return clientCode
	}
	return "provider_unavailable"
}
