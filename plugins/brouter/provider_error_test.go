package main

import "testing"

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
	if got := brouterProviderErrorCode(503, []byte("busy"), "no_route"); got != "provider_unavailable" {
		t.Fatalf("server error code = %q, want provider_unavailable", got)
	}
}

func TestBRouterProviderErrorCodeMarksRejectedPreparedProfileForRefresh(t *testing.T) {
	if got := brouterProviderErrorCode(400, []byte("unknown custom profile"), "unsupported_profile"); got != "unsupported_profile" {
		t.Fatalf("prepared profile error code = %q, want unsupported_profile", got)
	}
}
