package util

import (
	"bytes"
	"context"
	"net"
	"testing"
)

func TestFetchPublicURLRejectsUnsafeInputs(t *testing.T) {
	tests := []string{
		"ftp://example.com/file.jpg",
		"http://user:pass@example.com/file.jpg",
		"http://127.0.0.1/file.jpg",
		"http://localhost/file.jpg",
		"http://10.0.0.1/file.jpg",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/file.jpg",
		"http://[fc00::1]/file.jpg",
		"http://example.com:8080/file.jpg",
	}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := FetchPublicURL(context.Background(), rawURL, 1024); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestReadBoundedForPlugin(t *testing.T) {
	if _, err := ReadBoundedForPlugin(bytes.NewReader([]byte("1234")), 4); err != nil {
		t.Fatalf("unexpected exact-limit error: %v", err)
	}
	if _, err := ReadBoundedForPlugin(bytes.NewReader([]byte("12345")), 4); err == nil {
		t.Fatal("expected oversized response error")
	}
}

func TestSafeFetchedFileName(t *testing.T) {
	tests := []struct {
		name        string
		fallback    string
		finalURL    string
		contentType string
		want        string
	}{
		{
			name:        "uses URL filename with extension",
			fallback:    "activitypub-photo",
			finalURL:    "https://example.com/photos/camera.jpg?token=ignored",
			contentType: "image/png",
			want:        "camera.jpg",
		},
		{
			name:        "uses fallback extension when URL has none",
			fallback:    "activitypub-trail.gpx",
			finalURL:    "https://example.com/download",
			contentType: "application/xml+gpx",
			want:        "activitypub-trail.gpx",
		},
		{
			name:        "adds content type extension",
			fallback:    "activitypub-photo",
			finalURL:    "https://example.com/download",
			contentType: "image/png; charset=binary",
			want:        "download.png",
		},
		{
			name:        "sanitizes path fallback",
			fallback:    "../../bad",
			finalURL:    "https://example.com/",
			contentType: "",
			want:        "bad.bin",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := safeFetchedFileName(test.fallback, test.finalURL, test.contentType); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestConnectorTLSConfigRejectsInsecureMode(t *testing.T) {
	if _, err := connectorTLSConfig("insecure", nil); err == nil {
		t.Fatal("expected insecure TLS mode to be rejected")
	}
}

func TestConnectorIPAllowed(t *testing.T) {
	tests := []struct {
		ip           string
		allowPrivate bool
		want         bool
	}{
		{ip: "8.8.8.8", want: true},
		{ip: "10.0.0.1", want: false},
		{ip: "10.0.0.1", allowPrivate: true, want: true},
		{ip: "fc00::1", allowPrivate: true, want: true},
		{ip: "127.0.0.1", allowPrivate: true, want: false},
		{ip: "169.254.1.1", allowPrivate: true, want: false},
		{ip: "100.64.0.1", allowPrivate: true, want: false},
		{ip: "192.0.2.1", allowPrivate: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.ip, func(t *testing.T) {
			if got := connectorIPAllowed(net.ParseIP(test.ip), test.allowPrivate); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}
