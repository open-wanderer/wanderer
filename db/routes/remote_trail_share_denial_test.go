package routes

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteTrailUnconfirmedDenialsKeepRecipientGrantsAfterRecovery(t *testing.T) {
	for _, outcome := range []struct {
		name   string
		status int
		body   string
		header http.Header
	}{
		{"challenge", 403, "<html>Just a moment...</html>", http.Header{"Content-Type": {"text/html"}, "Cf-Mitigated": {"challenge"}}},
		{"unauthorized", 401, `{"message":"Unauthorized"}`, http.Header{"Content-Type": {"application/json"}}},
		{"unmarked_forbidden", 403, `{"status":403,"message":"Forbidden","data":{}}`, http.Header{"Content-Type": {"application/json"}}},
		{"not_found", 404, `{"message":"Not found"}`, http.Header{"Content-Type": {"application/json"}}},
		{"gone", 410, `{"message":"Gone"}`, http.Header{"Content-Type": {"application/json"}}},
	} {
		for _, mode := range []string{"all_unavailable", "unconfirmed_first", "unconfirmed_probe"} {
			t.Run(outcome.name+"/"+mode, func(t *testing.T) {
				f := newDirectedShareFixture(t)
				bobToken, carolToken := strings.Repeat("b", 32), strings.Repeat("c", 32)
				bobID, carolID := "aaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbb"
				if mode == "unconfirmed_probe" {
					bobID, carolID = carolID, bobID
				}
				bobShare := f.share(t, bobID, f.bob, bobToken)
				carolShare := f.share(t, carolID, f.carol, carolToken)
				outage := false
				attempts := []string{}
				previous := newRemoteSyncHTTPClient
				t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
				newRemoteSyncHTTPClient = func() *http.Client {
					return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
						token := request.URL.Query().Get("share")
						attempts = append(attempts, token)
						unavailable := token != "" && (mode == "all_unavailable" || token == bobToken)
						if token == "" && mode == "all_unavailable" && (outcome.status == 401 || outcome.status == 403) {
							unavailable = true
						}
						if outage && unavailable {
							if mode == "unconfirmed_probe" && request.URL.Query().Get("fields") != "id" {
								t.Fatal("later stored token was not checked with an access probe")
							}
							return &http.Response{StatusCode: outcome.status, Header: outcome.header.Clone(), Body: io.NopCloser(strings.NewReader(outcome.body)), Request: request}, nil
						}
						if token == bobToken || token == carolToken {
							return richTextSyncTransport{f.payload()}.RoundTrip(request)
						}
						return &http.Response{StatusCode: 403, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
					}}}
				}
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatalf("initial sync failed: %v", err)
				}
				if f.view(t, f.bobUser, "") != 200 || f.view(t, f.carolUser, "") != 200 {
					t.Fatal("fixture recipients cannot view the initially synced copy")
				}
				outage = true
				attempts = nil
				_, err := performFullSync(f.app, t.Context(), nil, f.trail)
				if mode == "all_unavailable" {
					if err == nil || errors.Is(err, errRemoteGone) || len(attempts) != 3 || attempts[2] != "" {
						t.Fatalf("outage did not leave an unconfirmed copy: requests=%d error=%v", len(attempts), err)
					}
				} else if err != nil || len(attempts) != 2 {
					t.Fatalf("unconfirmed stored token suppressed the other recipient: requests=%d error=%v", len(attempts), err)
				}
				for _, share := range []*core.Record{bobShare, carolShare} {
					if _, err := f.app.FindRecordById("trail_share", share.Id); err != nil {
						t.Fatal("unconfirmed denial removed a recipient's grant")
					}
				}
				f.keepsLocalRecords(t)
				outage = false
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatalf("recovery failed: %v", err)
				}
				if f.view(t, f.bobUser, "") != 200 || f.view(t, f.carolUser, "") != 200 {
					t.Fatal("recipients lost access after the origin recovered")
				}
				f.keepsLocalRecords(t)
			})
		}
	}
}

type countedShareDenialBody struct {
	reader *strings.Reader
	read   int
	closed bool
	fail   error
}

func (b *countedShareDenialBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	b.read += n
	if err == io.EOF && b.fail != nil {
		return n, b.fail
	}
	return n, err
}
func (b *countedShareDenialBody) Close() error { b.closed = true; return nil }

func TestRemoteTrailShareDenialRequiresBoundedJSONMarker(t *testing.T) {
	marker := `{"data":{"share":{"code":"federated_share_denied"}}}`
	for _, outcome := range []struct {
		name   string
		status int
		body   string
		denied bool
		fail   bool
	}{
		{"marked", 403, marker, true, false},
		{"empty", 403, "", false, false},
		{"plain", 403, `{"message":"Forbidden","data":{}}`, false, false},
		{"html", 403, "<html>Forbidden</html>", false, false},
		{"message_only", 403, `{"message":"federated_share_denied"}`, false, false},
		{"wrong_structure", 403, `{"share":{"code":"federated_share_denied"}}`, false, false},
		{"wrong_code", 403, `{"data":{"share":{"code":"unavailable"}}}`, false, false},
		{"trailing_data", 403, marker + " invalid", false, false},
		{"truncated", 403, `{"data":{"share":{"code":"federated_share_denied"}`, false, false},
		{"marker_after_limit", 403, strings.Repeat(" ", 4096) + marker, false, false},
		{"marker_before_oversized_padding", 403, marker + strings.Repeat(" ", 8192), false, false},
		{"marker_at_limit", 403, marker + strings.Repeat(" ", 4096-len(marker)), false, false},
		{"marker_below_limit", 403, marker + strings.Repeat(" ", 4095-len(marker)), true, false},
		{"read_error", 403, marker, false, true},
		{"unauthorized_marker", 401, marker, false, false},
		{"not_found_marker", 404, marker, false, false},
		{"gone_marker", 410, marker, false, false},
		{"server_marker", 503, marker, false, false},
	} {
		for _, probe := range []bool{false, true} {
			name := outcome.name + "/document"
			if probe {
				name = outcome.name + "/probe"
			}
			t.Run(name, func(t *testing.T) {
				token := strings.Repeat("r", 32)
				remoteURL, _ := url.Parse("https://b.example/api/v1/trail/origintrail0001?share=" + token)
				body := &countedShareDenialBody{reader: strings.NewReader(outcome.body)}
				if outcome.fail {
					body.fail = errors.New("read failed for " + remoteURL.String())
				}
				client := &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					return &http.Response{StatusCode: outcome.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: body, Request: request}, nil
				}}}
				denied := false
				if probe {
					var err error
					denied, err = probeRemoteTrailShare(t.Context(), client, remoteURL)
					if err != nil && strings.Contains(err.Error(), token) {
						t.Fatal("probe error retained a share token")
					}
				} else {
					document, err := fetchRemoteSyncDocument(t.Context(), client, remoteURL, "trail")
					if document != nil {
						document.close()
						t.Fatal("failed document fetch returned a staged document")
					}
					var status *remoteFetchStatusError
					if !errors.As(err, &status) || status.StatusCode != outcome.status {
						t.Fatalf("document fetch lost its status: %v", err)
					}
					denied = status.ShareDenied
					if strings.Contains(err.Error(), token) {
						t.Fatal("document error retained a share token")
					}
				}
				if denied != outcome.denied {
					t.Fatalf("confirmed denial=%v want %v", denied, outcome.denied)
				}
				if !body.closed || body.read > 4096 {
					t.Fatalf("denial body read=%d closed=%v", body.read, body.closed)
				}
				if outcome.status != 403 && body.read != 0 {
					t.Fatalf("non-403 denial body was read: %d bytes", body.read)
				}
			})
		}
	}
}
