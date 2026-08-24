package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

const (
	errProviderUnavailable = "provider_unavailable"
	errNoRoute             = "no_route"
)

func brouterPluginError(err error) *sdk.PluginError {
	if err == nil {
		return nil
	}
	pluginErr := &sdk.PluginError{Code: errProviderUnavailable, Message: err.Error()}
	var coded codedError
	if errors.As(err, &coded) {
		if coded.code != "" {
			pluginErr.Code = coded.code
		}
		pluginErr.RetryAfterSeconds = coded.retryAfterSeconds
	}
	return pluginErr
}

func brouterProviderError(response sdk.HostResponse, body []byte, clientCode string) error {
	return codedError{
		code:              brouterProviderErrorCode(response.Status, body, clientCode),
		message:           fmt.Sprintf("BRouter request failed (%d): %s", response.Status, brouterProviderMessage(body)),
		retryAfterSeconds: brouterRetryAfterSeconds(response),
	}
}

func brouterProviderErrorCode(status int, body []byte, clientCode string) string {
	if status == 429 {
		return "rate_limited"
	}
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

// brouterRetryAfterSeconds accepts only one positive delta-seconds value on a
// rate-limit or service-unavailable response. Absolute HTTP dates depend on the
// plugin clock, and duplicate values are ambiguous, so neither produces a hint.
func brouterRetryAfterSeconds(response sdk.HostResponse) *int {
	if response.Status != 429 && response.Status != 503 {
		return nil
	}
	values := []string{}
	for name, headerValues := range response.HeaderValues {
		if strings.EqualFold(name, "Retry-After") {
			values = append(values, headerValues...)
		}
	}
	if len(values) != 1 {
		return nil
	}
	value := strings.TrimSpace(values[0])
	if value == "" {
		return nil
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return nil
		}
	}
	seconds64, err := strconv.ParseUint(value, 10, 31)
	if err != nil || seconds64 == 0 {
		return nil
	}
	seconds := int(seconds64)
	return &seconds
}

func brouterProviderMessage(body []byte) string {
	message := strings.TrimSpace(string(body))
	if message == "" {
		return "empty provider response"
	}
	var parsed map[string]any
	if json.Unmarshal(body, &parsed) != nil {
		return message
	}
	if msg := stringValue(parsed["message"]); msg != "" {
		return msg
	}
	if errValue, ok := parsed["error"]; ok {
		switch errValue := errValue.(type) {
		case string:
			if errValue != "" {
				return errValue
			}
		case map[string]any:
			if msg := stringValue(errValue["message"]); msg != "" {
				return msg
			}
			if code := stringValue(errValue["code"]); code != "" {
				return code
			}
		}
	}
	if code := stringValue(parsed["code"]); code != "" {
		return code
	}
	return message
}
