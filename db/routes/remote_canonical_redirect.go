package routes

import (
	"net/http"
	"strings"
)

// An origin may upgrade the same canonical object to TLS. Keep its host,
// port, exact path and locally chosen query; no other redirect is followed.
func canonicalRemoteRedirect(next *http.Request, via []*http.Request) error {
	if len(via) != 1 {
		return http.ErrUseLastResponse
	}
	previous := via[0].URL
	if previous.Scheme != "http" || next.URL.Scheme != "https" || next.URL.User != nil ||
		!strings.EqualFold(previous.Host, next.URL.Host) || previous.EscapedPath() != next.URL.EscapedPath() ||
		previous.RawQuery != next.URL.RawQuery || next.URL.Fragment != "" {
		return http.ErrUseLastResponse
	}
	return nil
}
