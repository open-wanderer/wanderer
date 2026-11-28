package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"

	"pocketbase/geocoding"
)

const (
	geocodingTestSecret = "test-geocoding-proxy"
	geocodingCollection = `{"type":"FeatureCollection","features":[]}`
	geocodingErrorJSON  = `{"message":"Nominatim request failed"}`
)

func geocodingTestRouter(t *testing.T) http.Handler {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	drainTestAppStartupLogs(t, app)
	app.Settings().Logs.MaxDays = 0

	r, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	r.GET("/health", Health)
	RegisterGeocoding(&core.ServeEvent{App: app, Router: r})
	mux, err := r.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return mux
}

func geocodingRequest(t *testing.T, h http.Handler, path, secret string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	if secret != "" {
		req.Header.Set("X-Internal-Secret", secret)
	}
	req.Header.Set("Cookie", "test=caller-cookie")
	req.Header.Set("X-Forwarded-For", "192.0.2.1")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	return res
}

func assertGeocodingResponse(t *testing.T, res *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if got := strings.TrimSpace(res.Body.String()); res.Code != status || got != body {
		t.Fatalf("response = HTTP %d %q, want HTTP %d %q", res.Code, got, status, body)
	}
}

func TestGeocodingRequiresInternalProxy(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(geocodingCollection))
	}))
	t.Cleanup(upstream.Close)
	t.Setenv("NOMINATIM_URL", upstream.URL)
	t.Setenv("POCKETBASE_PROXY_SECRET", geocodingTestSecret)
	handler := geocodingTestRouter(t)
	for _, path := range []string{"/geocoding/search?q=place", "/geocoding/reverse?lat=1&lon=2"} {
		for _, secret := range []string{"", "wrong-proxy-secret"} {
			if res := geocodingRequest(t, handler, path, secret); res.Code != http.StatusUnauthorized {
				t.Fatalf("%s with no matching secret returned %d", path, res.Code)
			}
		}
	}
	t.Setenv("POCKETBASE_PROXY_SECRET", "")
	for _, path := range []string{"/geocoding/search", "/geocoding/reverse"} {
		if res := geocodingRequest(t, handler, path, geocodingTestSecret); res.Code != http.StatusUnauthorized {
			t.Fatalf("%s with no configured secret returned %d", path, res.Code)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("a request with no valid internal secret reached the upstream")
	}
}

func TestGeocodingFixedPathsAndQueryWhitelist(t *testing.T) {
	type request struct {
		method string
		path   string
		query  url.Values
		header http.Header
	}
	upstreamRequests := make(chan request, 3)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamRequests <- request{r.Method, r.URL.Path, r.URL.Query(), r.Header.Clone()}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(geocodingCollection))
	}))
	t.Cleanup(upstream.Close)
	t.Setenv("NOMINATIM_URL", upstream.URL+"/fixture/geo")
	t.Setenv("PUBLIC_NOMINATIM_URL", "ftp://invalid-legacy.test")
	t.Setenv("POCKETBASE_PROXY_SECRET", geocodingTestSecret)
	handler := geocodingTestRouter(t)

	for _, tc := range []struct {
		path   string
		target string
		params url.Values
	}{
		{"/geocoding/search?q=Berg+%26+湖&q=ignored&limit=0x10&limit=NaN&lat=2",
			"/fixture/geo/search", url.Values{"q": {"Berg & 湖"}, "limit": {"0x10"}}},
		{"/geocoding/search?q=+&limit=",
			"/fixture/geo/search", url.Values{"q": {" "}}},
		{"/geocoding/reverse?lat=%2BInfinity&lat=1&lon=0b10&lon=ignored&q=other",
			"/fixture/geo/reverse", url.Values{"lat": {"+Infinity"}, "lon": {"0b10"}}},
	} {
		t.Run(tc.path, func(t *testing.T) {
			path := tc.path + "&format=json&addressdetails=0&url=http://ignored.test/&fields=type"
			res := geocodingRequest(t, handler, path, geocodingTestSecret)
			assertGeocodingResponse(t, res, http.StatusCreated, geocodingCollection)
			var got request
			select {
			case got = <-upstreamRequests:
			default:
				t.Fatal("no upstream request was recorded")
			}
			tc.params.Set("format", "geojson")
			tc.params.Set("addressdetails", "1")
			if got.method != http.MethodGet || got.path != tc.target || !reflect.DeepEqual(got.query, tc.params) {
				t.Fatalf("unexpected upstream request: %s %s %v", got.method, got.path, got.query)
			}
			for _, header := range []string{"Cookie", "Authorization", "X-Internal-Secret", "X-Forwarded-For"} {
				if got.header.Get(header) != "" {
					t.Errorf("%s was sent to the upstream", header)
				}
			}
			if got := got.header.Get("User-Agent"); !strings.HasPrefix(got, "wanderer/") {
				t.Errorf("expected the Wanderer user agent, got %q", got)
			}
		})
	}
	assertGeocodingResponse(t, geocodingRequest(t, handler, "/geocoding/search?q=&q=valid", geocodingTestSecret),
		http.StatusBadRequest, `{"message":"Missing query parameter: q"}`)
	assertGeocodingResponse(t, geocodingRequest(t, handler, "/geocoding/reverse?lat=1", geocodingTestSecret),
		http.StatusBadRequest, `{"message":"Missing query parameter: lat or lon"}`)
	if len(upstreamRequests) != 0 {
		t.Fatal("missing required query values reached the upstream")
	}
}

func TestGeocodingUpstreamStatusAndRedaction(t *testing.T) {
	const noAddress = `{"error":"Unable to geocode"}`
	var redirectContacts atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect-target" {
			redirectContacts.Add(1)
			return
		}
		if r.URL.Path == "/reverse" {
			_, _ = w.Write([]byte(noAddress))
			return
		}
		switch r.URL.Query().Get("q") {
		case "missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"private upstream URL and credential"}`))
		case "unavailable":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("private upstream text"))
		case "rate-limited":
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusTooManyRequests)
		case "redirect":
			w.Header().Set("Location", "/redirect-target")
			w.WriteHeader(http.StatusFound)
		case "large":
			_, _ = w.Write([]byte(strings.Repeat("x", 2<<20)))
		case "disconnect":
			panic(http.ErrAbortHandler) // close the test connection without a response
		case "no-content":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(upstream.Close)
	t.Setenv("NOMINATIM_URL", upstream.URL)
	t.Setenv("POCKETBASE_PROXY_SECRET", geocodingTestSecret)
	handler := geocodingTestRouter(t)
	for _, tc := range []struct {
		query  string
		status int
		body   string
	}{
		{"missing", http.StatusNotFound, geocodingErrorJSON},
		{"unavailable", http.StatusServiceUnavailable, geocodingErrorJSON},
		{"rate-limited", http.StatusTooManyRequests, geocodingErrorJSON},
		{"redirect", http.StatusBadGateway, geocodingErrorJSON},
		{"large", http.StatusBadGateway, geocodingErrorJSON},
		{"disconnect", http.StatusBadGateway, geocodingErrorJSON},
		{"no-content", http.StatusNoContent, ""},
		{"empty", http.StatusOK, "{}"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			res := geocodingRequest(t, handler, "/geocoding/search?q="+tc.query, geocodingTestSecret)
			assertGeocodingResponse(t, res, tc.status, tc.body)
			if res.Header().Get("Location") != "" || res.Header().Get("Retry-After") != "" {
				t.Fatal("upstream redirect or retry headers were exposed")
			}
		})
	}
	if redirectContacts.Load() != 0 {
		t.Fatal("an upstream redirect was followed")
	}
	assertGeocodingResponse(t, geocodingRequest(t, handler, "/geocoding/reverse?lat=0&lon=0", geocodingTestSecret),
		http.StatusOK, noAddress)
}

func TestGeocodingRejectsInvalidExplicitConfigAndAllowsLegacyFallback(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "fixture-user" || password != "fixture-password" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(geocodingCollection))
	}))
	t.Cleanup(upstream.Close)
	legacy, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	legacy.User = url.UserPassword("fixture-user", "fixture-password")
	t.Setenv("PUBLIC_NOMINATIM_URL", legacy.String())
	t.Setenv("POCKETBASE_PROXY_SECRET", geocodingTestSecret)

	for _, config := range []string{"", " \t", "ftp://fixture-user:private-config@invalid.test"} {
		t.Setenv("NOMINATIM_URL", config)
		// Guard the network checks even if a regression selected the public default.
		if client, _ := configuredGeocodingClient(); client != nil {
			t.Fatal("invalid explicit configuration should disable geocoding")
		}
		handler := geocodingTestRouter(t)
		for _, path := range []string{"/geocoding/search?q=place", "/geocoding/reverse?lat=1&lon=2"} {
			assertGeocodingResponse(t, geocodingRequest(t, handler, path, geocodingTestSecret), http.StatusBadGateway, geocodingErrorJSON)
		}
		assertGeocodingResponse(t, geocodingRequest(t, handler, "/health", ""), http.StatusOK, `{"status":"ok"}`)
	}
	if err := os.Unsetenv("NOMINATIM_URL"); err != nil {
		t.Fatal(err)
	}
	handler := geocodingTestRouter(t)
	assertGeocodingResponse(t, geocodingRequest(t, handler, "/geocoding/reverse?lat=1&lon=2", geocodingTestSecret),
		http.StatusOK, geocodingCollection)
}

func TestGeocodingRequestContextCancelsUpstream(t *testing.T) {
	started := make(chan struct{}, 1)
	done := make(chan struct{}, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-r.Context().Done()
		select {
		case done <- struct{}{}:
		default:
		}
	}))
	t.Cleanup(upstream.Close)
	t.Setenv("NOMINATIM_URL", upstream.URL)
	t.Setenv("POCKETBASE_PROXY_SECRET", geocodingTestSecret)
	handler := geocodingTestRouter(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/geocoding/search?q=slow", nil)
	req.Header.Set("X-Internal-Secret", geocodingTestSecret)
	res := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		handler.ServeHTTP(res, req.WithContext(ctx))
	}()

	timeout := time.After(3 * time.Second)
	for _, step := range []struct {
		ch   <-chan struct{}
		name string
	}{
		{started, "start"},
		{done, "cancel upstream"},
		{finished, "finish handler"},
	} {
		select {
		case <-step.ch:
			if step.ch == started {
				cancel()
			}
		case <-timeout:
			t.Fatalf("timed out waiting for %s", step.name)
		}
	}
	assertGeocodingResponse(t, res, http.StatusBadGateway, geocodingErrorJSON)
}

type geocodingLogApp struct {
	core.App
	logger *slog.Logger
}

func (app geocodingLogApp) Logger() *slog.Logger { return app.logger }

func assertGeocodingLog(t *testing.T, output *bytes.Buffer, want map[string]any) {
	t.Helper()
	if want == nil {
		if output.Len() != 0 {
			t.Fatalf("unexpected log: %s", output)
		}
		return
	}
	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("expected exactly one JSON log entry, got %q: %v", output, err)
	}
	delete(got, slog.TimeKey)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("log = %v, want %v", got, want)
	}
}

func TestGeocodingStartupNoticeOnlyForImplicitPublicDefault(t *testing.T) {
	const configured = "https://fixture-user:fixture-password@private.fixture.test/prefix"
	for _, tc := range []struct {
		name     string
		settings map[string]string
		want     map[string]any
	}{
		{"both absent", nil, map[string]any{
			slog.LevelKey:   "INFO",
			slog.MessageKey: "Geocoding is using public Nominatim by default because no Nominatim URL is configured on db",
		}},
		{"canonical configured", map[string]string{"NOMINATIM_URL": configured}, nil},
		{"legacy configured", map[string]string{"PUBLIC_NOMINATIM_URL": configured}, nil},
		{"explicit public", map[string]string{"NOMINATIM_URL": geocoding.DefaultBaseURL}, nil},
		{"explicit blank", map[string]string{"NOMINATIM_URL": "", "PUBLIC_NOMINATIM_URL": configured}, map[string]any{
			slog.LevelKey: "WARN", slog.MessageKey: "Geocoding disabled: invalid Nominatim configuration",
		}},
		{"invalid legacy", map[string]string{"PUBLIC_NOMINATIM_URL": "ftp://fixture-user:fixture-password@private.fixture.test"}, map[string]any{
			slog.LevelKey: "WARN", slog.MessageKey: "Geocoding disabled: invalid Nominatim configuration",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{"NOMINATIM_URL", "PUBLIC_NOMINATIM_URL"} {
				value, present := tc.settings[name]
				t.Setenv(name, value)
				if !present {
					if err := os.Unsetenv(name); err != nil {
						t.Fatal(err)
					}
				}
			}
			var output bytes.Buffer
			app := geocodingLogApp{logger: slog.New(slog.NewJSONHandler(&output, nil))}
			RegisterGeocoding(&core.ServeEvent{App: app, Router: router.NewRouter[*core.RequestEvent](nil)})
			assertGeocodingLog(t, &output, tc.want)
		})
	}
}

func TestGeocodingFailureLogsOnlyFixedCategoriesAtDebug(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	expired, cancelDeadline := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancelDeadline()
	const sensitiveCause = "https://fixture-user:fixture-password@private.fixture.test/?q=visitor-place"
	for _, tc := range []struct {
		name     string
		ctx      context.Context
		level    slog.Level
		err      error
		category string
	}{
		{"upstream timeout", t.Context(), slog.LevelDebug, geocoding.ErrRequestTimeout, "timeout"},
		{"too large", t.Context(), slog.LevelDebug, geocoding.ErrResponseTooLarge, "response_too_large"},
		{"transport or read failure", t.Context(), slog.LevelDebug, fmt.Errorf("%s: %w", sensitiveCause, geocoding.ErrRequestFailed), "transport_read_failure"},
		{"unknown failure", t.Context(), slog.LevelDebug, errors.New(sensitiveCause), ""},
		{"default log level", t.Context(), slog.LevelInfo, geocoding.ErrRequestFailed, ""},
		{"caller cancellation", cancelled, slog.LevelDebug, context.Canceled, ""},
		{"caller deadline", expired, slog.LevelDebug, context.DeadlineExceeded, ""},
		{"caller deadline takes precedence", expired, slog.LevelDebug, geocoding.ErrRequestTimeout, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			app := geocodingLogApp{logger: slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: tc.level}))}
			req := httptest.NewRequestWithContext(tc.ctx, http.MethodGet, "/geocoding/search?q=visitor-place", nil)
			req.Header.Set("X-Internal-Secret", geocodingTestSecret)
			req.Header.Set("X-Forwarded-For", "192.0.2.1")
			req.Header.Set("Cookie", "test=caller-cookie")
			res := httptest.NewRecorder()

			event := &core.RequestEvent{App: app, Event: router.Event{Request: req, Response: res}}
			if err := writeGeocodingResponse(event, geocoding.Response{}, tc.err); err != nil {
				t.Fatal(err)
			}
			assertGeocodingResponse(t, res, http.StatusBadGateway, geocodingErrorJSON)
			if tc.category == "" {
				assertGeocodingLog(t, &output, nil)
				return
			}
			assertGeocodingLog(t, &output, map[string]any{
				slog.LevelKey: "DEBUG", slog.MessageKey: "Nominatim upstream request failed", "category": tc.category,
			})
		})
	}
}
