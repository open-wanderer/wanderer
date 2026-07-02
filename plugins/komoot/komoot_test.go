package main

import (
	"testing"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

func TestAcceptLanguageFromLocale(t *testing.T) {
	tests := []struct {
		name   string
		locale string
		want   string
	}{
		{name: "empty", locale: "", want: ""},
		{name: "language only", locale: "de", want: "de"},
		{name: "underscore region", locale: "de_CH", want: "de-CH,de;q=0.9"},
		{name: "hyphen region", locale: "en-US", want: "en-US,en;q=0.9"},
		{name: "trim space", locale: " fr_FR ", want: "fr-FR,fr;q=0.9"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := acceptLanguage(test.locale); got != test.want {
				t.Fatalf("acceptLanguage(%q) = %q, want %q", test.locale, got, test.want)
			}
		})
	}
}

func TestRequestHeadersOnlySendAuthToAPIConnector(t *testing.T) {
	client := &komootClient{
		userID: "user",
		token:  "token",
		locale: "de_CH",
	}

	apiHeaders := client.requestHeaders("api")
	if apiHeaders[sdk.AuthHeaderAuthorization] == "" {
		t.Fatalf("expected api connector authorization header")
	}
	if apiHeaders["Accept-Language"] != "de-CH,de;q=0.9" {
		t.Fatalf("unexpected api accept language: %#v", apiHeaders)
	}

	webHeaders := client.requestHeaders("web")
	if webHeaders[sdk.AuthHeaderAuthorization] != "" {
		t.Fatalf("expected no web connector authorization header, got %#v", webHeaders)
	}
	if webHeaders["Accept-Language"] != "de-CH,de;q=0.9" {
		t.Fatalf("unexpected web accept language: %#v", webHeaders)
	}
}
