package federation

import "testing"

func TestIsLocalHandleDomain(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		domain string
		want   bool
	}{
		{"origin host", "https://www.local.example", "www.local.example", true},
		{"stored host", "https://www.local.example", "local.example", true},
		{"origin host case", "https://www.local.example", "WWW.LOCAL.EXAMPLE", true},
		{"stored host case", "https://www.local.example", "LOCAL.EXAMPLE", true},
		{"origin host case retained", "https://WWW.Local.Example", "www.local.example", true},
		{"only stored prefix is removed", "https://WWW.Local.Example", "local.example", false},
		{"bare origin host", "https://local.example", "local.example", true},
		{"no added www alias", "https://local.example", "www.local.example", false},
		{"https default port", "https://www.local.example", "www.local.example:443", true},
		{"stored host https default port", "https://www.local.example", "local.example:443", true},
		{"http default port", "http://local.example", "local.example:80", true},
		{"https different port", "https://local.example", "local.example:80", false},
		{"http different port", "http://local.example", "local.example:443", false},
		{"explicit origin port", "https://www.local.example:8443", "www.local.example:8443", true},
		{"stored host explicit origin port", "https://www.local.example:8443", "local.example:8443", true},
		{"portless origin host with custom port", "https://www.local.example:8443", "www.local.example", true},
		{"portless stored host with custom port", "https://www.local.example:8443", "local.example", true},
		{"custom origin different port", "https://local.example:8443", "local.example:443", false},
		{"configured default port", "https://local.example:443", "local.example:443", true},
		{"unknown scheme explicit port", "other://local.example", "local.example:443", false},
		{"foreign host", "https://www.local.example", "remote.example", false},
		{"earlier origin host", "https://www.local.example", "previous.example", false},
		{"host suffix", "https://local.example", "local.example.remote.example", false},
		{"host prefix", "https://local.example", "otherlocal.example", false},
		{"www host suffix", "https://www.local.example", "www.local.example.remote.example", false},
		{"empty handle domain", "https://local.example", "", false},
		{"empty origin", "", "local.example", false},
		{"origin without host", "https://", "local.example", false},
		{"origin without scheme", "local.example", "local.example", false},
		{"invalid origin", "https://%local.example", "local.example", false},
		{"scheme in handle domain", "https://local.example", "https://local.example", false},
		{"userinfo", "https://local.example", "account@local.example", false},
		{"path", "https://local.example", "local.example/profile", false},
		{"empty path segment", "https://local.example", "local.example/", false},
		{"query", "https://local.example", "local.example?profile=1", false},
		{"empty query", "https://local.example", "local.example?", false},
		{"fragment", "https://local.example", "local.example#profile", false},
		{"empty fragment", "https://local.example", "local.example#", false},
		{"explicit empty port", "https://local.example", "local.example:", false},
		{"invalid port", "https://local.example", "local.example:https", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ORIGIN", tt.origin)
			if got := isLocalHandleDomain(tt.domain); got != tt.want {
				t.Fatalf("isLocalHandleDomain(%q) with ORIGIN=%q = %v, want %v", tt.domain, tt.origin, got, tt.want)
			}
		})
	}
}
