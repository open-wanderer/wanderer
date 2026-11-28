package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"pocketbase/util"
	"slices"
	"strings"
	"testing"
	"time"
)

type directedShareFixture struct {
	app                                                  *core.BaseApp
	trail, remote, bob, carol, bobUser, carolUser, entry *core.Record
}

func newDirectedShareFixture(t *testing.T) *directedShareFixture {
	t.Helper()
	app := newRemoteRealSchemaApp(t)
	t.Setenv("ORIGIN", "https://a.example")
	save := func(c string, v map[string]any) *core.Record { return saveRemoteChildRecord(t, app, c, v) }
	f := &directedShareFixture{app: app}
	local := func(name string) (*core.Record, *core.Record) {
		user := save("users", map[string]any{"username": name, "email": name + "@a.example", "password": "test-password"})
		actor := save("activitypub_actors", map[string]any{"username": name, "preferred_username": name, "domain": "a.example", "is_local": true, "user": user.Id, "public_key": "key", "iri": "https://a.example/api/v1/activitypub/user/" + name, "inbox": "https://a.example/inbox/" + name})
		return actor, user
	}
	f.bob, f.bobUser = local("bob")
	f.carol, f.carolUser = local("carol")
	f.remote = save("activitypub_actors", map[string]any{"username": "owner@b.example", "preferred_username": "owner", "domain": "b.example", "public_key": "key", "iri": "https://b.example/api/v1/activitypub/user/owner", "inbox": "https://b.example/inbox/owner", "last_fetched": time.Now()})
	f.trail = save("trails", map[string]any{"name": "Stored trail", "author": f.remote.Id, "iri": "https://b.example/api/v1/trail/origintrail0001", "public": false, "full_sync_completed": true, "needs_full_sync": true})
	f.entry = save("summit_logs", map[string]any{"author": f.bob.Id, "trail": f.trail.Id, "text": "Local entry", "iri": "https://a.example/api/v1/summit-log/localentry00001"})
	return f
}
func (f *directedShareFixture) share(t *testing.T, id string, actor *core.Record, token string) *core.Record {
	t.Helper()
	return saveRemoteChildRecord(t, f.app, "trail_share", map[string]any{"id": id, "trail": f.trail.Id, "actor": actor.Id, "permission": "view", "remote_token": token})
}
func (f *directedShareFixture) payload() map[string]any {
	return map[string]any{"id": "origintrail0001", "iri": f.trail.GetString("iri"), "name": "Current private trail", "public": false, "expand": map[string]any{"trail_assets_via_trail": []any{}, "waypoints_via_trail": []any{}, "summit_logs_via_trail": []any{}}}
}
func (f *directedShareFixture) client(t *testing.T, fn func(*http.Request) (map[string]any, int)) {
	t.Helper()
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			data, status := fn(request)
			if status != http.StatusOK {
				body := io.ReadCloser(http.NoBody)
				if data != nil {
					encoded, err := json.Marshal(data)
					if err != nil {
						t.Fatal(err)
					}
					body = io.NopCloser(strings.NewReader(string(encoded)))
				}
				return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: body, Request: request}, nil
			}
			return richTextSyncTransport{data}.RoundTrip(request)
		}}}
	}
}
func (f *directedShareFixture) view(t *testing.T, user *core.Record, query string) int {
	t.Helper()
	event := &core.RequestEvent{App: f.app, Auth: user}
	event.Request = httptest.NewRequest(http.MethodGet, "/api/v1/trail/"+f.trail.Id+"?"+query, nil)
	event.Request.SetPathValue("id", f.trail.Id)
	response := httptest.NewRecorder()
	event.Response = response
	err := RemoteTrailGet(event)
	if err != nil {
		var apiError *router.ApiError
		if !errors.As(err, &apiError) {
			t.Fatal(err)
		}
		return apiError.Status
	}
	return response.Code
}
func (f *directedShareFixture) wait(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, busy := trailSyncing.Load(f.trail.GetString("iri")); !busy {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("background trail refresh did not finish")
}
func (f *directedShareFixture) keepsLocalRecords(t *testing.T) {
	t.Helper()
	for _, record := range []*core.Record{f.trail, f.entry} {
		if _, err := f.app.FindRecordById(record.Collection(), record.Id); err != nil {
			t.Fatalf("request removed stored %s: %v", record.Collection().Name, err)
		}
	}
}

func TestRemoteTrailRequesterShareKeepsStoredCredentialAndLocalEntry(t *testing.T) {
	f := newDirectedShareFixture(t)
	valid := strings.Repeat("v", 32)
	share := f.share(t, "aaaaaaaaaaaaaaa", f.bob, valid)
	attempts := []string{}
	f.client(t, func(request *http.Request) (map[string]any, int) {
		token := request.URL.Query().Get("share")
		attempts = append(attempts, token)
		if token == valid {
			return f.payload(), http.StatusOK
		}
		if token != "" {
			return nil, http.StatusNotFound
		}
		return nil, http.StatusForbidden
	})
	previous := remoteHandleGetActor
	t.Cleanup(func() { remoteHandleGetActor = previous })
	remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) { return f.remote, nil }
	status := f.view(t, nil, "handle=owner@b.example&share="+strings.Repeat("x", 32))
	f.wait(t)
	if status != http.StatusForbidden {
		t.Fatalf("private copy request status=%d want403", status)
	}
	f.keepsLocalRecords(t)
	stored, err := f.app.FindRecordById("trail_share", share.Id)
	if err != nil || stored.GetString("remote_token") != valid {
		t.Fatal("request replaced or removed stored directed share")
	}
	if !slices.Equal(attempts, []string{valid}) {
		t.Fatalf("requester share displaced stored credential: attempts=%d", len(attempts))
	}
}

func TestRemoteTrailRequesterDenialsNeverRetractCopyOrShare(t *testing.T) {
	for _, status := range []int{401, 403, 404, 410} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			f := newDirectedShareFixture(t)
			share := f.share(t, "aaaaaaaaaaaaaaa", f.bob, "")
			requester := strings.Repeat("x", 32)
			attempts := []string{}
			f.client(t, func(request *http.Request) (map[string]any, int) {
				token := request.URL.Query().Get("share")
				attempts = append(attempts, token)
				if token == requester {
					if status == http.StatusForbidden {
						return deniedFederatedSharePayload(), status
					}
					return nil, status
				}
				return nil, http.StatusForbidden
			})
			previous := remoteHandleGetActor
			t.Cleanup(func() { remoteHandleGetActor = previous })
			remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) { return f.remote, nil }
			f.view(t, nil, "handle=owner@b.example&share="+requester)
			f.wait(t)
			f.keepsLocalRecords(t)
			if _, err := f.app.FindRecordById("trail_share", share.Id); err != nil {
				t.Fatal("requester denial removed a directed share")
			}
			if !slices.Equal(attempts, []string{requester, ""}) {
				t.Fatalf("requester denial missed anonymous confirmation: attempts=%d", len(attempts))
			}
		})
	}
}

func deniedFederatedSharePayload() map[string]any {
	return map[string]any{"data": map[string]any{"share": map[string]any{"code": "federated_share_denied"}}}
}

func TestRemoteTrailRevocationRemovesOnlyDeniedRecipientsAccess(t *testing.T) {
	for _, activeFirst := range []bool{false, true} {
		name := "revoked_first"
		if activeFirst {
			name = "active_first"
		}
		t.Run(name, func(t *testing.T) {
			f := newDirectedShareFixture(t)
			revoked := strings.Repeat("r", 32)
			active := strings.Repeat("c", 32)
			deniedID, activeID := "aaaaaaaaaaaaaaa", "bbbbbbbbbbbbbbb"
			if activeFirst {
				deniedID, activeID = activeID, deniedID
			}
			removed := f.share(t, deniedID, f.bob, revoked)
			retained := f.share(t, activeID, f.carol, active)
			if f.view(t, f.bobUser, "") != 200 || f.view(t, f.carolUser, "") != 200 {
				t.Fatal("fixture recipients cannot view shared copy")
			}
			attempts := []string{}
			f.client(t, func(request *http.Request) (map[string]any, int) {
				token := request.URL.Query().Get("share")
				attempts = append(attempts, token)
				if token == active {
					return f.payload(), 200
				}
				return deniedFederatedSharePayload(), http.StatusForbidden
			})
			result, err := performFullSync(f.app, t.Context(), &url.URL{RawQuery: "share=" + strings.Repeat("x", 32)}, f.trail)
			if err != nil || result.GetString("name") != "Current private trail" {
				t.Fatalf("active recipient's sync failed: %v", err)
			}
			if _, err := f.app.FindRecordById("trail_share", removed.Id); err == nil {
				t.Fatal("revoked recipient's local share remained")
			}
			if _, err := f.app.FindRecordById("trail_share", retained.Id); err != nil {
				t.Fatal("active recipient's local share was removed")
			}
			if got := f.view(t, f.bobUser, ""); got != 403 {
				t.Fatalf("revoked user still views refreshed copy: status=%d", got)
			}
			if got := f.view(t, f.carolUser, ""); got != 200 {
				t.Fatalf("active user lost refreshed copy: status=%d", got)
			}
			expected := []string{revoked, active}
			if activeFirst {
				expected = []string{active, revoked}
			}
			if !slices.Equal(attempts, expected) {
				t.Fatal("stored validation was suppressed by requester credential or early success")
			}
			f.keepsLocalRecords(t)
		})
	}
}

func TestRemoteTrailOnlyRevokedRecipientKeepsCopyAndLocalEntry(t *testing.T) {
	f := newDirectedShareFixture(t)
	revoked := strings.Repeat("r", 32)
	share := f.share(t, "aaaaaaaaaaaaaaa", f.bob, revoked)
	attempts := []string{}
	f.client(t, func(request *http.Request) (map[string]any, int) {
		attempts = append(attempts, request.URL.Query().Get("share"))
		if request.URL.Query().Get("share") != "" {
			return deniedFederatedSharePayload(), 403
		}
		return nil, 403
	})
	_, err := performFullSync(f.app, t.Context(), nil, f.trail)
	var status *remoteFetchStatusError
	if !errors.As(err, &status) || status.StatusCode != 403 || errors.Is(err, errRemoteGone) {
		t.Fatalf("revoked share denial became object deletion: %v", err)
	}
	f.keepsLocalRecords(t)
	if _, err := f.app.FindRecordById("trail_share", share.Id); err == nil {
		t.Fatal("only revoked recipient retained local access")
	}
	if got := f.view(t, f.bobUser, ""); got != 403 {
		t.Fatalf("only revoked recipient still views copy: status=%d", got)
	}
	if !slices.Equal(attempts, []string{revoked, ""}) {
		t.Fatal("denied stored share did not try credential-free read")
	}
}

func TestRemoteTrailRevocationKeepsShareRenewedDuringFetch(t *testing.T) {
	f := newDirectedShareFixture(t)
	revoked := strings.Repeat("r", 32)
	active := strings.Repeat("c", 32)
	share := f.share(t, "aaaaaaaaaaaaaaa", f.bob, revoked)
	f.client(t, func(request *http.Request) (map[string]any, int) {
		if request.URL.Query().Get("share") == revoked {
			current, err := f.app.FindRecordById("trail_share", share.Id)
			if err != nil {
				t.Fatal(err)
			}
			current.Set("remote_token", active)
			if err := f.app.Save(current); err != nil {
				t.Fatal(err)
			}
		}
		if request.URL.Query().Get("share") != "" {
			return deniedFederatedSharePayload(), 403
		}
		return nil, 403
	})
	_, err := performFullSync(f.app, t.Context(), nil, f.trail)
	if err == nil {
		t.Fatal("fixture should deny both fetch attempts")
	}
	current, err := f.app.FindRecordById("trail_share", share.Id)
	if err != nil || current.GetString("remote_token") != active {
		t.Fatal("denied old token removed a renewed directed share")
	}
	f.keepsLocalRecords(t)
}

func TestRemoteTrailRequesterShareFollowsDeniedStoredCredential(t *testing.T) {
	f := newDirectedShareFixture(t)
	revoked := strings.Repeat("r", 32)
	requester := strings.Repeat("c", 32)
	share := f.share(t, "aaaaaaaaaaaaaaa", f.bob, revoked)
	attempts := []string{}
	f.client(t, func(request *http.Request) (map[string]any, int) {
		token := request.URL.Query().Get("share")
		attempts = append(attempts, token)
		if token == requester {
			return f.payload(), 200
		}
		if token == revoked {
			return deniedFederatedSharePayload(), 403
		}
		return nil, 403
	})
	result, err := performFullSync(f.app, t.Context(), &url.URL{RawQuery: "share=" + requester}, f.trail)
	if err != nil || result.GetString("name") != "Current private trail" || !slices.Equal(attempts, []string{revoked, requester}) {
		t.Fatalf("requester credential was not tried after stored denial: requests=%d error=%v", len(attempts), err)
	}
	if _, err := f.app.FindRecordById("trail_share", share.Id); err == nil {
		t.Fatal("denied stored token retained access after requester fetch")
	}
	f.keepsLocalRecords(t)
}

type trackedShareProbeBody struct {
	closed bool
	read   bool
}

func (b *trackedShareProbeBody) Read([]byte) (int, error) { b.read = true; return 0, io.EOF }
func (b *trackedShareProbeBody) Close() error             { b.closed = true; return nil }

func TestRemoteTrailLaterRecipientProbeKeepsFirstDocumentAndClosesBodies(t *testing.T) {
	for _, failure := range []string{"available", "network", "server"} {
		t.Run(failure, func(t *testing.T) {
			f := newDirectedShareFixture(t)
			first := strings.Repeat("a", 32)
			later := strings.Repeat("b", 32)
			f.share(t, "aaaaaaaaaaaaaaa", f.bob, first)
			share := f.share(t, "bbbbbbbbbbbbbbb", f.carol, later)
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			probe := &trackedShareProbeBody{}
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Query().Get("share") == first {
						return richTextSyncTransport{f.payload()}.RoundTrip(request)
					}
					if request.URL.Query().Get("share") != later || request.URL.Query().Get("fields") != "id" || request.URL.Query().Get("expand") != "" {
						t.Fatal("later recipient did not use a lightweight access probe")
					}
					if failure == "network" {
						return nil, fmt.Errorf("fetch failed for %s", request.URL)
					}
					status := 200
					if failure == "server" {
						status = 503
					}
					return &http.Response{StatusCode: status, Header: make(http.Header), Body: probe, Request: request}, nil
				}}}
			}
			result, err := performFullSync(f.app, t.Context(), nil, f.trail)
			if err != nil || result.GetString("name") != "Current private trail" {
				t.Fatalf("later recipient probe replaced the selected document: %v", err)
			}
			if failure != "network" && (!probe.closed || probe.read) {
				t.Fatal("recipient probe retained or read an unnecessary body")
			}
			if _, err := f.app.FindRecordById("trail_share", share.Id); err != nil {
				t.Fatal("unconfirmed recipient was revoked after a transient probe")
			}
		})
	}
}

func TestRemoteTrailAnonymousConfirmationClearsInheritedCredential(t *testing.T) {
	f := newDirectedShareFixture(t)
	requester := strings.Repeat("r", 32)
	attempts := []string{}
	f.client(t, func(request *http.Request) (map[string]any, int) {
		token := request.URL.Query().Get("share")
		attempts = append(attempts, token)
		if token == "" {
			return nil, 403
		}
		return nil, 404
	})
	ctx := util.WithFederatedShareToken(t.Context(), f.trail.GetString("iri"), strings.Repeat("i", 32))
	_, err := performFullSync(f.app, ctx, &url.URL{RawQuery: "share=" + requester}, f.trail)
	if errors.Is(err, errRemoteGone) || !slices.Equal(attempts, []string{requester, ""}) {
		t.Fatal("anonymous confirmation inherited a credential")
	}
	f.keepsLocalRecords(t)
}
