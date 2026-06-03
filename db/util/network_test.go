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
