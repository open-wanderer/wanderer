package routes

import "testing"

func TestRemoteOriginBindingAllowsHTTPSUpgradeWithoutTrustingOtherHosts(t *testing.T) {
	for _, tc := range []struct {
		iri, origin string
		want        bool
	}{
		{"http://remote.example/api/v1/summit-log/one", "https://remote.example", true},
		{"https://remote.example/api/v1/summit-log/one", "http://remote.example", true},
		{"http://remote.example:444/api/v1/summit-log/one", "https://remote.example:444", true},
		{"https://foreign.example/api/v1/summit-log/one", "https://remote.example", false},
		{"https://remote.example:444/api/v1/summit-log/one", "https://remote.example", false},
		{"ftp://remote.example/api/v1/summit-log/one", "https://remote.example", false},
		{"https://user@remote.example/api/v1/summit-log/one", "https://remote.example", false},
	} {
		if got := remoteSyncIRIMatchesOrigin(tc.iri, tc.origin); got != tc.want {
			t.Errorf("%s against %s = %t; want %t", tc.iri, tc.origin, got, tc.want)
		}
	}
}
