package routes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

type canonicalAssetBodyTransport func(*http.Request) (*http.Response, error)

func (transport canonicalAssetBodyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type canonicalAssetCountedBody struct {
	io.Reader
	read    *int64
	closed  *int
	onClose func()
}

func (body canonicalAssetCountedBody) Read(buffer []byte) (int, error) {
	n, err := body.Reader.Read(buffer)
	*body.read += int64(n)
	return n, err
}

func (body canonicalAssetCountedBody) Close() error {
	*body.closed++
	if body.onClose != nil {
		body.onClose()
	}
	return nil
}

// Only bare schema/record values and in-memory HTTP bodies: no App, TestApp,
// Bootstrap, database, fixture storage, listener, actor lookup or media fetch.
func canonicalAssetBodyJob(t *testing.T) (*core.Record, canonicalLogRequest, string, string) {
	t.Helper()
	collection := core.NewBaseCollection("trails")
	collection.Fields.Add(&core.TextField{Name: "iri"})
	trail := core.NewRecord(collection)
	trail.Set("iri", "https://trail.example/api/v1/trail/parent")
	iri := "https://origin.example/api/v1/summit-log/entry"
	target, err := url.Parse(iri)
	if err != nil {
		t.Fatal(err)
	}
	key := "r1-pure:" + t.Name()
	t.Cleanup(func() { canonicalRemoteLogCache.invalidate(key) })
	coreData := map[string]any{
		"id": "entry", "iri": iri, "text": "Core entry",
		"expand": map[string]any{
			"author": map[string]any{"iri": "https://origin.example/api/v1/actor/author"},
			"trail":  map[string]any{"iri": trail.GetString("iri")},
		},
	}
	plain, err := json.Marshal(coreData)
	if err != nil {
		t.Fatal(err)
	}
	coreData["expand"].(map[string]any)["summit_log_assets_via_summit_log"] = []any{
		map[string]any{"expand": map[string]any{"asset": map[string]any{
			"id": "photo", "metadata": map[string]any{"description": strings.Repeat("x", int(canonicalRemoteBodyLimit))},
		}}},
	}
	expanded, err := json.Marshal(coreData)
	if err != nil {
		t.Fatal(err)
	}
	return trail, canonicalLogRequest{iri: iri, url: target, cacheKey: key, updated: "hint"}, string(plain), string(expanded)
}

func TestCanonicalAssetBodyLimitFallbackKeepsCoreWithinOriginalBudget(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "staged"}[staged], func(t *testing.T) {
			trail, job, plain, expanded := canonicalAssetBodyJob(t)
			ctx := withCanonicalRemoteBodyBudget(t.Context())
			if staged {
				ctx = context.WithValue(ctx, remoteSyncStagingKey{}, true)
			}
			token := strings.Repeat("a", 32)
			ctx = util.WithFederatedShareToken(ctx, "https://origin.example", token)
			budget := ctx.Value(canonicalRemoteBodyBudgetKey{}).(*canonicalRemoteBodyBudget)
			var requests []*http.Request
			var read int64
			closed := 0
			client := &http.Client{Transport: canonicalAssetBodyTransport(func(request *http.Request) (*http.Response, error) {
				requests = append(requests, request)
				want := "author,trail,summit_log_assets_via_summit_log.asset"
				wire := expanded
				if len(requests) == 2 {
					want, wire = "author,trail", plain
				}
				query := request.URL.Query()
				if query.Get("expand") != want || query.Get("share") != token || len(query) != 2 {
					t.Errorf("request %d query=%v", len(requests), query)
				}
				if request.Context().Value(canonicalRemoteBodyBudgetKey{}) != budget {
					t.Error("retry replaced the shared budget")
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request,
					Body: canonicalAssetCountedBody{Reader: strings.NewReader(wire), read: &read, closed: &closed}}, nil
			})}
			result, ok := fetchCanonicalRemoteSummitLog(nil, ctx, client, trail, job)
			if !ok || result.removed || result.omitted || result.assetsRequested || result.data["text"] != "Core entry" {
				t.Fatalf("core fallback failed or claimed assets: %#v accepted=%t", result, ok)
			}
			if len(requests) != 2 || closed != 2 || requests[0].Context() != requests[1].Context() {
				t.Fatalf("attempt count, closure or context changed: requests=%d closed=%d", len(requests), closed)
			}
			firstDeadline, _ := requests[0].Context().Deadline()
			secondDeadline, _ := requests[1].Context().Deadline()
			if firstDeadline.IsZero() || firstDeadline != secondDeadline {
				t.Fatal("retry reset the canonical timeout")
			}
			wantRead := canonicalRemoteBodyLimit + 1 + int64(len(plain))
			if read != wantRead || budget.remaining.Load() != canonicalRemoteByteBudget-wantRead {
				t.Fatalf("body cap or aggregate accounting changed: read=%d remaining=%d", read, budget.remaining.Load())
			}
			if _, authoritative := syncRecordPhotoList("summit_log", result.remoteID, result.origin, result.data, result.assetsRequested); authoritative {
				t.Fatal("missing asset projection became an authoritative empty photo list")
			}
			if canonicalRemoteLogCache.skip(job.cacheKey, job.updated, "", true) {
				t.Fatal("successful fallback retained a failure-cache entry")
			}
			if shouldRetryRemoteFetchWithoutAssetExpands(util.ErrFederationBodyTooLarge) {
				t.Fatal("global trail/share status-only fallback semantics changed")
			}
		})
	}
}

func TestCanonicalAssetBodyLimitFallbackRejectsUnsafeRetries(t *testing.T) {
	for _, scenario := range []string{
		"shared-exhausted", "shared-exhausted-at-response-boundary",
		"fallback-exhausts-shared", "fallback-still-oversized", "omitted",
		"invalid-json", "network-error", "denied", "not-found", "rate-limited", "server-error",
		"canceled-after-response",
	} {
		t.Run(scenario, func(t *testing.T) {
			trail, job, plain, expanded := canonicalAssetBodyJob(t)
			ctx, cancel := context.WithCancel(withCanonicalRemoteBodyBudget(t.Context()))
			defer cancel()
			budget := ctx.Value(canonicalRemoteBodyBudgetKey{}).(*canonicalRemoteBodyBudget)
			initialBudget := canonicalRemoteByteBudget
			switch scenario {
			case "shared-exhausted":
				initialBudget = 32
			case "shared-exhausted-at-response-boundary":
				initialBudget = canonicalRemoteBodyLimit + 1
			case "fallback-exhausts-shared":
				initialBudget = canonicalRemoteBodyLimit + 1 + 5
			case "omitted":
				job.omitted = true
			}
			budget.remaining.Store(initialBudget)
			requests, closed := 0, 0
			var read int64
			client := &http.Client{Transport: canonicalAssetBodyTransport(func(request *http.Request) (*http.Response, error) {
				requests++
				if job.omitted && request.URL.Query().Get("expand") != "author,trail" {
					t.Error("omitted entry requested assets")
				}
				if scenario == "network-error" {
					return nil, errors.New("in-memory transport error")
				}
				status, wire := http.StatusOK, expanded
				if requests == 2 && scenario != "fallback-still-oversized" {
					wire = plain
				}
				switch scenario {
				case "invalid-json":
					wire = "{invalid"
				case "denied":
					status = http.StatusForbidden
				case "not-found":
					status = http.StatusNotFound
				case "rate-limited":
					status = http.StatusTooManyRequests
				case "server-error":
					status = http.StatusInternalServerError
				}
				body := canonicalAssetCountedBody{Reader: strings.NewReader(wire), read: &read, closed: &closed}
				if scenario == "canceled-after-response" {
					body.onClose = cancel
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Request: request, Body: body}, nil
			})}
			if result, ok := fetchCanonicalRemoteSummitLog(nil, ctx, client, trail, job); ok {
				t.Fatalf("unsafe/failed core response accepted: %#v", result)
			}
			wantRequests := 1
			if scenario == "fallback-exhausts-shared" || scenario == "fallback-still-oversized" {
				wantRequests = 2
			}
			wantClosed := wantRequests
			if scenario == "network-error" {
				wantClosed = 0
			}
			if requests != wantRequests || closed != wantClosed {
				t.Fatalf("unexpected retry/response leak: requests=%d closed=%d want=%d/%d", requests, closed, wantRequests, wantClosed)
			}
			if read > initialBudget+1 || budget.remaining.Load() > initialBudget {
				t.Fatalf("shared budget increased or was exceeded: read=%d initial=%d remaining=%d", read, initialBudget, budget.remaining.Load())
			}
		})
	}
}
