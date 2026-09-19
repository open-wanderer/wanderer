package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

func TestBRouterProviderErrorCodeRecognizesWatchdogAsTimeout(t *testing.T) {
	message := []byte("operation killed by thread-priority-watchdog after 0 seconds")
	if got := brouterProviderErrorCode(400, message, "provider_unavailable"); got != "provider_timeout" {
		t.Fatalf("watchdog error code = %q, want provider_timeout", got)
	}
}

func TestBRouterProviderErrorCodeKeepsOtherHTTPClassification(t *testing.T) {
	if got := brouterProviderErrorCode(404, []byte("no route"), "no_route"); got != "no_route" {
		t.Fatalf("client error code = %q, want no_route", got)
	}
	if got := brouterProviderErrorCode(429, []byte("slow down"), "provider_unavailable"); got != "rate_limited" {
		t.Fatalf("rate-limit error code = %q, want rate_limited", got)
	}
	if got := brouterProviderErrorCode(503, []byte("busy"), "no_route"); got != "provider_unavailable" {
		t.Fatalf("server error code = %q, want provider_unavailable", got)
	}
}

func TestBRouterProviderErrorCodeMarksRejectedPreparedProfileForRefresh(t *testing.T) {
	if got := brouterProviderErrorCode(400, []byte("unknown custom profile"), "unsupported_profile"); got != "unsupported_profile" {
		t.Fatalf("prepared profile error code = %q, want unsupported_profile", got)
	}
}

func TestBRouterRetryAfterSecondsRequiresReliableDelta(t *testing.T) {
	tests := []struct {
		name     string
		response sdk.HostResponse
		want     int
		present  bool
	}{
		{
			name:     "rate limited",
			response: sdk.HostResponse{Status: 429, HeaderValues: map[string][]string{"Retry-After": {" 120 "}}},
			want:     120,
			present:  true,
		},
		{
			name:     "service unavailable case insensitive",
			response: sdk.HostResponse{Status: 503, HeaderValues: map[string][]string{"retry-after": {"30"}}},
			want:     30,
			present:  true,
		},
		{
			name:     "unrelated status",
			response: sdk.HostResponse{Status: 500, HeaderValues: map[string][]string{"Retry-After": {"30"}}},
		},
		{
			name:     "absolute date",
			response: sdk.HostResponse{Status: 429, HeaderValues: map[string][]string{"Retry-After": {"Wed, 21 Oct 2037 07:28:00 GMT"}}},
		},
		{
			name:     "duplicate values",
			response: sdk.HostResponse{Status: 429, HeaderValues: map[string][]string{"Retry-After": {"30", "60"}}},
		},
		{
			name:     "zero",
			response: sdk.HostResponse{Status: 429, HeaderValues: map[string][]string{"Retry-After": {"0"}}},
		},
		{
			name:     "negative",
			response: sdk.HostResponse{Status: 429, HeaderValues: map[string][]string{"Retry-After": {"-1"}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := brouterRetryAfterSeconds(test.response)
			if !test.present {
				if got != nil {
					t.Fatalf("retry hint = %d, want omitted", *got)
				}
				return
			}
			if got == nil || *got != test.want {
				t.Fatalf("retry hint = %v, want %d", got, test.want)
			}
		})
	}
}

func TestBRouterPluginErrorPropagatesReliableRetryHint(t *testing.T) {
	err := brouterProviderError(sdk.HostResponse{
		Status:       429,
		HeaderValues: map[string][]string{"Retry-After": {"90"}},
	}, []byte(`{"message":"rate limit reached"}`), errProviderUnavailable)
	pluginErr := brouterPluginError(err)
	if pluginErr.Code != "rate_limited" || pluginErr.Message != "BRouter request failed (429): rate limit reached" {
		t.Fatalf("plugin error = %#v", pluginErr)
	}
	if pluginErr.RetryAfterSeconds == nil || *pluginErr.RetryAfterSeconds != 90 {
		t.Fatalf("plugin retry hint = %#v", pluginErr.RetryAfterSeconds)
	}
	wrappedPluginErr := brouterPluginError(fmt.Errorf("route request: %w", err))
	if wrappedPluginErr.Code != "rate_limited" || wrappedPluginErr.RetryAfterSeconds == nil || *wrappedPluginErr.RetryAfterSeconds != 90 {
		t.Fatalf("wrapped plugin error lost code or retry hint: %#v", wrappedPluginErr)
	}

	err = brouterProviderError(sdk.HostResponse{
		Status:       429,
		HeaderValues: map[string][]string{"Retry-After": {"tomorrow"}},
	}, []byte("slow down"), errProviderUnavailable)
	if pluginErr := brouterPluginError(err); pluginErr.RetryAfterSeconds != nil {
		t.Fatalf("unreliable retry hint was propagated: %#v", pluginErr)
	}

	pluginErr = brouterPluginError(errors.New("connector failed"))
	if pluginErr.Code != errProviderUnavailable || pluginErr.Message != "connector failed" || pluginErr.RetryAfterSeconds != nil {
		t.Fatalf("fallback plugin error = %#v", pluginErr)
	}
}
