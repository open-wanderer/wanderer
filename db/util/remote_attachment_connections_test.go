package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/doyensec/safeurl"
)

type remoteFileClosingClient struct {
	HTTPDoer
	closed int
}

func (client *remoteFileClosingClient) CloseIdleConnections() { client.closed++ }

func TestRemoteFileFetchersCloseClientsOnErrors(t *testing.T) {
	for _, fetcher := range []string{"attachment-default", "attachment-installed", "public"} {
		for _, failure := range []string{"request", "status", "size"} {
			t.Run(fetcher+"/"+failure, func(t *testing.T) {
				client := &remoteFileClosingClient{HTTPDoer: &http.Client{Transport: attachmentRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					if failure == "request" {
						return nil, errors.New("request failed")
					}
					response := attachmentBodyResponse(req, []byte("12345"))
					if failure == "status" {
						response.StatusCode = http.StatusServiceUnavailable
					}
					return response, nil
				})}}
				var err error
				if fetcher != "public" {
					if fetcher == "attachment-default" {
						t.Cleanup(SetRemoteAttachmentClientForTesting(nil))
						previous := newRemoteAttachmentClient
						newRemoteAttachmentClient = func(time.Duration, func(*http.Request, []*http.Request) error) HTTPDoer { return client }
						t.Cleanup(func() { newRemoteAttachmentClient = previous })
					} else {
						t.Cleanup(SetRemoteAttachmentClientForTesting(client))
					}
					_, cleanup, fetchErr := DownloadRemoteFile(context.Background(), "http://example.com/file", 4)
					defer cleanup()
					err = fetchErr
				} else {
					previous := newPublicURLClient
					newPublicURLClient = func(time.Duration, func(*http.Request, []*http.Request) error) HTTPDoer { return client }
					t.Cleanup(func() { newPublicURLClient = previous })
					_, err = FetchPublicURL(context.Background(), "http://example.com/file", 4)
				}
				if err == nil || client.closed != 1 {
					t.Fatalf("failed fetch left its client open: error=%v closes=%d", err, client.closed)
				}
			})
		}
	}
}

func TestRemoteFileFetchersReleaseIdleConnectionsAfterRepeatedCalls(t *testing.T) {
	for _, fetcher := range []string{"attachment-default", "attachment-installed", "public"} {
		t.Run(fetcher, func(t *testing.T) {
			var active, opened atomic.Int32
			changed := make(chan struct{}, 1)
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("file-body"))
			}))
			server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				switch state {
				case http.StateNew:
					active.Add(1)
					opened.Add(1)
				case http.StateClosed:
					active.Add(-1)
				default:
					return
				}
				select {
				case changed <- struct{}{}:
				default:
				}
			}
			server.Start()
			t.Cleanup(server.Close)
			newClient := func(timeout time.Duration, redirect func(*http.Request, []*http.Request) error) HTTPDoer {
				client := NewSafeURLClient(timeout, redirect).(*safeurl.WrappedClient)
				// Only the fixture's dialer reaches the local server. The separate
				// IP-policy test uses the production dialer unchanged.
				client.Client.Transport.(*http.Transport).DialContext = (&net.Dialer{}).DialContext
				t.Cleanup(client.CloseIdleConnections)
				return client
			}
			if fetcher == "public" {
				previous := newPublicURLClient
				newPublicURLClient = newClient
				t.Cleanup(func() { newPublicURLClient = previous })
			} else {
				t.Cleanup(SetRemoteAttachmentClientForTesting(nil))
				if fetcher == "attachment-default" {
					previous := newRemoteAttachmentClient
					newRemoteAttachmentClient = newClient
					t.Cleanup(func() { newRemoteAttachmentClient = previous })
				}
			}
			const calls = 100
			for index := range calls {
				rawURL := fmt.Sprintf("%s/file-%d", server.URL, index)
				if fetcher != "public" {
					if fetcher == "attachment-installed" {
						SetRemoteAttachmentClientForTesting(newClient(RemoteAttachmentTimeout, publicMediaRedirectPolicy))
					}
					file, cleanup, err := DownloadRemoteFile(context.Background(), rawURL, 16)
					cleanup()
					if err != nil || file == nil || file.Size != 9 {
						t.Fatalf("download %d failed: file=%v error=%v", index, file, err)
					}
				} else {
					result, err := FetchPublicURL(context.Background(), rawURL, 16)
					if err != nil || result == nil || string(result.Body) != "file-body" {
						t.Fatalf("fetch %d failed: result=%v error=%v", index, result, err)
					}
				}
			}
			if opened.Load() != calls {
				t.Fatalf("expected %d separate client connections, got %d", calls, opened.Load())
			}
			deadline := time.NewTimer(time.Second)
			defer deadline.Stop()
			for active.Load() != 0 {
				select {
				case <-changed:
				case <-deadline.C:
					t.Fatalf("%d idle connections remain after %d completed calls", active.Load(), calls)
				}
			}
		})
	}
}
