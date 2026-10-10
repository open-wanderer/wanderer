package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/doyensec/safeurl"
)

// Attachment URLs may use the origin's own port on the origin's hostname; any
// other URL is limited to ports 80 and 443. Only public addresses are allowed.

// attachmentAllowLoopbackIPs adds loopback to the attachment client's IP allow
// list so tests reach httptest servers through the real port and redirect
// policy.
func attachmentAllowLoopbackIPs(t *testing.T) {
	t.Helper()
	t.Cleanup(SetRemoteAttachmentClientForTesting(nil))
	remoteAttachmentAllowedIPsForTest = []string{"127.0.0.1"}
	t.Cleanup(func() { remoteAttachmentAllowedIPsForTest = nil })
}

// attachmentProductionClient makes sure neither an override nor the loopback
// knob is active.
func attachmentProductionClient(t *testing.T) {
	t.Helper()
	t.Cleanup(SetRemoteAttachmentClientForTesting(nil))
	if remoteAttachmentAllowedIPsForTest != nil {
		t.Fatal("loopback knob leaked into a production-client test")
	}
}

type portServer struct {
	*httptest.Server
	hits int32
	port int
}

func (s *portServer) total() int { return int(atomic.LoadInt32(&s.hits)) }

func newPortServer(t *testing.T, handler http.HandlerFunc) *portServer {
	t.Helper()
	s := &portServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&s.hits, 1)
		handler(w, r)
	}))
	t.Cleanup(s.Close)
	s.port = s.Listener.Addr().(*net.TCPAddr).Port
	return s
}

func serveGPX(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write([]byte("<gpx/>"))
}

func originOn(host string, port int) string {
	return fmt.Sprintf("http://%s:%d/api/v1/activitypub/user/porty", host, port)
}

func TestDownloadRemoteFileAllowsOriginPort(t *testing.T) {
	attachmentAllowLoopbackIPs(t)
	attachmentIsolatedTempDir(t)
	s := newPortServer(t, serveGPX)

	f, cleanup, err := DownloadRemoteFile(context.Background(), s.URL+"/t.gpx", RemoteGPXMaxBytes, originOn("127.0.0.1", s.port))
	defer cleanup()
	if err != nil {
		t.Fatalf("attachment on the origin's own port was refused: %v", err)
	}
	if f == nil || f.Size == 0 {
		t.Fatalf("no file returned: %v", f)
	}
	if s.total() != 1 {
		t.Errorf("server hits = %d, want 1", s.total())
	}
}

func TestDownloadRemoteFileFollowsRedirectOnOriginPort(t *testing.T) {
	attachmentAllowLoopbackIPs(t)
	attachmentIsolatedTempDir(t)
	s := newPortServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start.gpx" {
			http.Redirect(w, r, "/final.gpx", http.StatusFound)
			return
		}
		serveGPX(w, r)
	})

	f, cleanup, err := DownloadRemoteFile(context.Background(), s.URL+"/start.gpx", RemoteGPXMaxBytes, originOn("127.0.0.1", s.port))
	defer cleanup()
	if err != nil {
		t.Fatalf("redirect on the origin's own port was refused: %v", err)
	}
	if f == nil || f.OriginalName != "final.gpx" {
		t.Errorf("file = %v, want final.gpx", f)
	}
	if s.total() != 2 {
		t.Errorf("server hits = %d, want 2", s.total())
	}
}

func TestDownloadRemoteFileOriginPortStillRefusesLoopback(t *testing.T) {
	attachmentProductionClient(t)
	attachmentIsolatedTempDir(t)
	s := newPortServer(t, serveGPX)

	_, cleanup, err := DownloadRemoteFile(context.Background(), s.URL+"/t.gpx", RemoteGPXMaxBytes, originOn("127.0.0.1", s.port))
	defer cleanup()
	if err == nil {
		t.Fatal("loopback address on the origin's port was downloaded")
	}
	var ipErr *safeurl.AllowedIPError
	if !errors.As(err, &ipErr) {
		t.Errorf("error = %v, want the IP policy (*safeurl.AllowedIPError)", err)
	}
	var portErr *safeurl.AllowedPortError
	if errors.As(err, &portErr) {
		t.Errorf("error = %v, refused by the port policy, want the IP policy", err)
	}
	if s.total() != 0 {
		t.Errorf("server hits = %d, want 0", s.total())
	}
}

func TestDownloadRemoteFileRefusesOtherNonDefaultPort(t *testing.T) {
	attachmentAllowLoopbackIPs(t)
	attachmentIsolatedTempDir(t)
	s := newPortServer(t, serveGPX)
	other := newPortServer(t, serveGPX)

	_, cleanup, err := DownloadRemoteFile(context.Background(), s.URL+"/t.gpx", RemoteGPXMaxBytes, originOn("127.0.0.1", other.port))
	defer cleanup()
	if err == nil {
		t.Fatal("a non-default port other than the origin's was downloaded")
	}
	if s.total() != 0 {
		t.Errorf("server hits = %d, want 0", s.total())
	}
}

func TestDownloadRemoteFileRefusesOriginPortOnOtherHost(t *testing.T) {
	attachmentAllowLoopbackIPs(t)
	attachmentIsolatedTempDir(t)
	s := newPortServer(t, serveGPX)

	_, cleanup, err := DownloadRemoteFile(context.Background(), s.URL+"/t.gpx", RemoteGPXMaxBytes, originOn("peer.example", s.port))
	defer cleanup()
	if err == nil {
		t.Fatal("the origin's port on another hostname was downloaded")
	}
	if s.total() != 0 {
		t.Errorf("server hits = %d, want 0", s.total())
	}
}

func TestDownloadRemoteFileRefusesRedirectToOtherPort(t *testing.T) {
	attachmentAllowLoopbackIPs(t)
	attachmentIsolatedTempDir(t)
	second := newPortServer(t, serveGPX)
	first := newPortServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, second.URL+"/t.gpx", http.StatusFound)
	})

	_, cleanup, err := DownloadRemoteFile(context.Background(), first.URL+"/t.gpx", RemoteGPXMaxBytes, originOn("127.0.0.1", first.port))
	defer cleanup()
	if err == nil {
		t.Fatal("a redirect to another non-default port was followed")
	}
	if second.total() != 0 {
		t.Errorf("redirect target hits = %d, want 0", second.total())
	}
}

func TestDownloadRemoteFileDefaultPortOriginAddsNoPort(t *testing.T) {
	attachmentAllowLoopbackIPs(t)
	attachmentIsolatedTempDir(t)
	s := newPortServer(t, serveGPX)

	_, cleanup, err := DownloadRemoteFile(context.Background(), s.URL+"/t.gpx", RemoteGPXMaxBytes, "https://peer.example/api/v1/activitypub/user/porty")
	defer cleanup()
	if err == nil {
		t.Fatal("an origin on the default port opened a non-default port")
	}
	if s.total() != 0 {
		t.Errorf("server hits = %d, want 0", s.total())
	}
}

func TestFetchPublicURLStillRefusesNonDefaultPort(t *testing.T) {
	s := newPortServer(t, serveGPX)

	_, err := FetchPublicURL(context.Background(), s.URL+"/x", 1024)
	if err == nil {
		t.Fatal("FetchPublicURL fetched a non-default port")
	}
	var portErr *safeurl.AllowedPortError
	if !errors.As(err, &portErr) {
		t.Errorf("error = %v, want *safeurl.AllowedPortError", err)
	}
	if s.total() != 0 {
		t.Errorf("server hits = %d, want 0", s.total())
	}
}

func TestCheckAttachmentURLPort(t *testing.T) {
	origin := "https://Peer.Example:8443/api/v1/activitypub/user/porty"
	cases := []struct {
		name, url, origin string
		allowed           bool
	}{
		{"default https", "https://other.example/a", origin, true},
		{"default http", "http://other.example/a", "", true},
		{"explicit 443", "https://other.example:443/a", "", true},
		{"origin port same host", "https://peer.example:8443/a", origin, true},
		{"origin port host case-insensitive", "https://PEER.example:8443/a", origin, true},
		{"origin port other host", "https://other.example:8443/a", origin, false},
		{"other port same host", "https://peer.example:9000/a", origin, false},
		{"non-default without origin", "https://peer.example:8443/a", "", false},
		{"origin on default port adds nothing", "https://peer.example:8443/a", "https://peer.example/u", false},
		{"unparseable port", "https://peer.example:99999/a", origin, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, err := url.Parse(c.url)
			if err != nil {
				t.Fatal(err)
			}
			err = checkAttachmentURLPort(u, c.origin)
			if c.allowed && err != nil {
				t.Errorf("refused: %v", err)
			}
			if !c.allowed && !errors.Is(err, ErrAttachmentPortNotAllowed) {
				t.Errorf("error = %v, want ErrAttachmentPortNotAllowed", err)
			}
		})
	}
}
