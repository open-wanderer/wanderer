package util

import (
	"bytes"
	"context"
	"net"
	"net/netip"
	"strings"
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

// A blocked address must be rejected by policy, not merely fail to connect --
// dialling e.g. ::7f00:1 errors out on most hosts regardless, which would make a
// plain "expected an error" assertion pass vacuously.
func TestFetchPublicURLBlocksTransitionAddresses(t *testing.T) {
	for _, rawURL := range []string{
		"http://[64:ff9b::7f00:1]/file.jpg",            // NAT64 -> 127.0.0.1
		"http://[2002:c0a8:101::]/file.jpg",            // 6to4 -> 192.168.1.1
		"http://[2001:0:53aa:64c::3f57:fefe]/file.jpg", // Teredo -> 192.168.1.1
		"http://[::7f00:1]/file.jpg",                   // IPv4-compatible -> 127.0.0.1
		"http://[::a9fe:a9fe]/file.jpg",                // IPv4-compatible -> 169.254.169.254
	} {
		t.Run(rawURL, func(t *testing.T) {
			_, err := FetchPublicURL(context.Background(), rawURL, 1024)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "not found in allowlist") {
				t.Fatalf("expected the address to be blocked by policy, got: %v", err)
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

func TestIsPrivateOrReservedIP(t *testing.T) {
	blocked := []string{
		// IPv6 transition addresses embedding an internal IPv4 (GHSA-8j93-xvwx-jjg3)
		"64:ff9b::7f00:1",            // NAT64 -> 127.0.0.1
		"64:ff9b::a9fe:a9fe",         // NAT64 -> 169.254.169.254 (cloud metadata)
		"64:ff9b::a00:1",             // NAT64 -> 10.0.0.1
		"2002:c0a8:101::",            // 6to4 -> 192.168.1.1
		"2002:7f00:1::",              // 6to4 -> 127.0.0.1
		"2001:0:53aa:64c::3f57:fefe", // Teredo -> 192.168.1.1
		"::7f00:1",                   // IPv4-compatible -> 127.0.0.1
		// Already covered before the fix
		"127.0.0.1",
		"::ffff:127.0.0.1",
		"10.0.0.1",
		"192.168.1.1",
		"172.16.0.1",
		"169.254.169.254",
		"::1",
		"fc00::1",
		"0.0.0.0",
		"224.0.0.1",
		"ff02::1",
		// Reserved ranges the old implementation missed entirely
		"100.64.0.1", // CGNAT
		"240.0.0.1",  // reserved
		"0.1.2.3",    // 0.0.0.0/8
		"192.0.0.1",  // IETF protocol assignments
		"192.0.2.1",  // TEST-NET-1
	}
	for _, ip := range blocked {
		t.Run("block/"+ip, func(t *testing.T) {
			if !isPrivateOrReservedIP(net.ParseIP(ip)) {
				t.Fatalf("expected %s to be blocked", ip)
			}
		})
	}

	allowed := []string{
		"8.8.8.8",
		"1.1.1.1",
		"93.184.216.34",
		"2606:4700:4700::1111",
	}
	for _, ip := range allowed {
		t.Run("allow/"+ip, func(t *testing.T) {
			if isPrivateOrReservedIP(net.ParseIP(ip)) {
				t.Fatalf("expected %s to be allowed", ip)
			}
		})
	}

	t.Run("unparseable fails closed", func(t *testing.T) {
		if !isPrivateOrReservedIP(net.IP{1, 2, 3}) {
			t.Fatal("expected malformed IP to be blocked")
		}
	})
}

func TestUnwrapTransitionIP(t *testing.T) {
	tests := map[string]string{
		"64:ff9b::7f00:1":            "127.0.0.1",
		"2002:c0a8:101::":            "192.168.1.1",
		"2001:0:53aa:64c::3f57:fefe": "192.168.1.1",
		"::7f00:1":                   "127.0.0.1",
		"::ffff:8.8.8.8":             "8.8.8.8",
		"2606:4700:4700::1111":       "2606:4700:4700::1111", // untouched
		"8.8.8.8":                    "8.8.8.8",
	}
	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			addr, err := netip.ParseAddr(input)
			if err != nil {
				t.Fatalf("bad test input: %v", err)
			}
			if got := unwrapTransitionIP(addr).String(); got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
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
		// IPv6 transition addresses embedding an internal IPv4 (GHSA-8j93-xvwx-jjg3).
		// allowPrivate must not reopen these.
		{ip: "64:ff9b::7f00:1", allowPrivate: true, want: false},
		{ip: "64:ff9b::a9fe:a9fe", allowPrivate: true, want: false},
		{ip: "2002:c0a8:101::", allowPrivate: true, want: false},
		{ip: "2001:0:53aa:64c::3f57:fefe", allowPrivate: true, want: false},
		{ip: "::7f00:1", allowPrivate: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.ip, func(t *testing.T) {
			if got := connectorIPAllowed(net.ParseIP(test.ip), test.allowPrivate); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}
