package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"pocketbase/internal/srch0"
	"pocketbase/util"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/spf13/cobra"
)

func srch0StartupApp(t *testing.T, empty bool) core.App {
	t.Helper()
	d := srch0.Data(t)
	for _, collection := range []string{"trails", "lists", "trail_share", "trail_like", "list_share"} {
		kept := []srch0.Object{}
		for _, record := range d.Records[collection] {
			if collection == "trails" && (record["id"] == "public-alpine" || record["id"] == "public-lake") || collection == "lists" && record["id"] == "list-local" {
				kept = append(kept, record)
			}
		}
		d.Records[collection] = kept
	}
	if empty {
		for _, collection := range []string{"trails", "lists", "activitypub_actors"} {
			d.Records[collection] = nil
		}
	}
	return srch0.App(t, d)
}

func srch0StartupRouter(app core.App) *router.Router[*core.RequestEvent] {
	return router.NewRouter(func(w http.ResponseWriter, r *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
		event := &core.RequestEvent{App: app}
		event.Request, event.Response = r, w
		return event, nil
	})
}

func TestSRCH0Startup(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, c := range srch0.Cases(t, "mutation", "go-startup") {
		t.Run(c.ID, func(t *testing.T) {
			app := srch0StartupApp(t, c.Input["empty_source"] == true)
			m := srch0.NewMeili(t)
			m.Missing = c.Input["indexes"] == "missing"
			m.FailDelete = c.Input["fail_delete"] == true
			if c.Input["indexes"] == "existing" {
				for _, index := range []string{"trails", "lists", "actors"} {
					m.Documents[index]["prior"] = srch0.Object{"id": "prior"}
				}
			}
			before := m.Snapshot()
			var opened atomic.Bool
			availability := []any{}
			m.Before = func(req srch0.Request) {
				if !strings.Contains(req.Path, "/documents") {
					return
				}
				if req.Method == "DELETE" {
					t.Errorf("startup must not delete existing search data: %s", req.Path)
				}
				if opened.Load() {
					t.Errorf("search became ready before startup document task: %s %s", req.Method, req.Path)
				}
				counts := map[string]int{}
				for index, documents := range m.Snapshot() {
					counts[index] = len(documents.(map[string]any))
				}
				availability = append(availability, map[string]any{"phase": req.Method + " " + req.Path, "search_ready": opened.Load(), "document_counts": counts})
			}
			var jobs sync.WaitGroup
			background := func(task func()) {
				jobs.Add(1)
				go func() { defer jobs.Done(); task() }()
			}
			routes := srch0StartupRouter(app)
			handler := hook.Hook[*core.ServeEvent]{}
			handler.BindFunc(onBeforeServeHandlerWithBackground(m.Client, background))
			err := handler.Trigger(&core.ServeEvent{App: app, Router: routes}, func(*core.ServeEvent) error { opened.Store(true); return nil })
			if !awaitSRCH0Background(&jobs) {
				t.Fatal("startup job did not finish")
			}
			if err != nil {
				t.Fatal(err)
			}
			after := m.Snapshot()
			if !m.Missing && !reflect.DeepEqual(before, after) {
				t.Error("normal startup changed existing index documents")
			}
			if m.Missing {
				for index, count := range map[string]int{"trails": 2, "lists": 1, "actors": 3} {
					if len(after[index].(map[string]any)) != count {
						t.Errorf("%s is incomplete at readiness", index)
					}
				}
			}
			mux, err := routes.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest("GET", "/search/token", nil))
			if response.Code != http.StatusOK {
				t.Errorf("ready search token status=%d", response.Code)
			}

			settings := map[string]any{}
			rebuild := []string{}
			waits := 0
			for _, req := range m.Calls() {
				if strings.HasSuffix(req.Path, "/settings") {
					settings[strings.Split(req.Path, "/")[2]] = req.Body
				}
				if strings.Contains(req.Path, "/documents") {
					rebuild = append(rebuild, req.Method+" "+req.Path)
				}
				if strings.HasPrefix(req.Path, "/tasks/") {
					waits++
				}
			}
			wantWaits := 3
			if m.Missing {
				wantWaits = 9
			}
			if waits != wantWaits {
				t.Errorf("startup must await every creation/settings/document task: got %d, want %d", waits, wantWaits)
			}
			for index, name := range map[string]string{"trails": "trails-legacy-v0", "lists": "lists-legacy-v0", "actors": "legacy-actor-v0"} {
				var profile struct {
					Settings map[string]any `json:"settings"`
				}
				srch0.Read(t, "profiles/"+name+".json", &profile)
				want, err := srch0StartupSettings(profile.Settings, c.Observed, index)
				if err != nil {
					t.Fatal(err)
				}
				var actual meilisearch.Settings
				bytes, _ := json.Marshal(settings[index])
				if err := json.Unmarshal(bytes, &actual); err != nil {
					t.Fatal(err)
				}
				srch0.Assert(t, c, actual, want)
			}
			readyCounts := map[string]int{}
			for index, documents := range after {
				readyCounts[index] = len(documents.(map[string]any))
			}
			got := map[string]any{"serve_next_called": opened.Load(), "search_token_status": response.Code, "document_counts": readyCounts, "settings_match_profiles": len(settings) == 3, "terminal_task_reads": waits, "rebuild_requests": rebuild, "availability": availability}
			srch0.Assert(t, c, got, c.Observed["mutation_state"])
		})
	}
}

type srch0StartupFault struct {
	path   string
	status int
	body   string
	once   sync.Once
}

func (fault *srch0StartupFault) RoundTrip(request *http.Request) (*http.Response, error) {
	reject := false
	if request.URL.Path == fault.path {
		fault.once.Do(func() { reject = true })
	}
	if !reject {
		return http.DefaultTransport.RoundTrip(request)
	}
	return &http.Response{StatusCode: fault.status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(fault.body)), Request: request}, nil
}

func TestSRCH0StartupRejectsFailedInitialization(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, fault := range []struct {
		name, path string
		status     int
		body       string
	}{
		{"lookup-permission", "/indexes/trails", 403, `{"message":"denied","code":"invalid_api_key","type":"auth"}`},
		{"settings-submission", "/indexes/trails/settings", 400, `{"message":"invalid settings","code":"invalid_settings_ranking_rules","type":"invalid_request"}`},
		{"terminal-task-failure", "/tasks/1", 200, `{"uid":1,"status":"failed","type":"indexCreation","error":{"message":"rejected","code":"invalid_document_id"}}`},
		{"document-submission", "/indexes/trails/documents", 400, `{"message":"invalid document","code":"invalid_document_id","type":"invalid_request"}`},
	} {
		t.Run(fault.name, func(t *testing.T) {
			app := srch0StartupApp(t, false)
			m := srch0.NewMeili(t)
			m.Missing = true
			client := meilisearch.New(m.Server.URL, meilisearch.WithCustomClient(&http.Client{Transport: &srch0StartupFault{path: fault.path, status: fault.status, body: fault.body}}))
			var jobs sync.WaitGroup
			background := func(task func()) { jobs.Add(1); go func() { defer jobs.Done(); task() }() }
			opened := false
			handler := hook.Hook[*core.ServeEvent]{}
			handler.BindFunc(onBeforeServeHandlerWithBackground(client, background))
			err := handler.Trigger(&core.ServeEvent{App: app, Router: srch0StartupRouter(app)}, func(*core.ServeEvent) error { opened = true; return nil })
			if !awaitSRCH0Background(&jobs) {
				t.Fatal("failed startup kept retrying in background")
			}
			if err == nil || opened {
				t.Errorf("initialization failure must prevent serve: err=%v, opened=%v", err, opened)
			}
		})
	}
}

func TestSRCH0StartupResumesInterruptedInitialization(t *testing.T) {
	t.Chdir(t.TempDir())
	app := srch0StartupApp(t, false)
	m := srch0.NewMeili(t)
	m.Missing = true
	client := meilisearch.New(m.Server.URL, meilisearch.WithCustomClient(&http.Client{Transport: &srch0StartupFault{
		path: "/indexes/trails/documents", status: 400,
		body: `{"message":"interrupted first fill","code":"invalid_document_id","type":"invalid_request"}`,
	}}))
	run := func() (bool, error) {
		var jobs sync.WaitGroup
		background := func(task func()) { jobs.Add(1); go func() { defer jobs.Done(); task() }() }
		opened := false
		handler := hook.Hook[*core.ServeEvent]{}
		handler.BindFunc(onBeforeServeHandlerWithBackground(client, background))
		err := handler.Trigger(&core.ServeEvent{App: app, Router: srch0StartupRouter(app)}, func(*core.ServeEvent) error { opened = true; return nil })
		if !awaitSRCH0Background(&jobs) {
			t.Fatal("startup job did not finish")
		}
		return opened, err
	}
	if opened, err := run(); err == nil || opened {
		t.Errorf("interrupted initialization became ready: err=%v, opened=%v", err, opened)
	}
	// The indexes now exist, but their failed first fill is not a ready state.
	// A new startup must finish them, rather than accepting mere index existence.
	m.Missing = false
	if opened, err := run(); err != nil || !opened {
		t.Fatalf("retry did not become ready: err=%v, opened=%v", err, opened)
	}
	for index, count := range map[string]int{"trails": 2, "lists": 1, "actors": 3} {
		if got := len(m.Snapshot()[index].(map[string]any)); got != count {
			t.Errorf("retry left %s incomplete: got %d, want %d", index, got, count)
		}
		if _, err := os.Stat(filepath.Join(app.DataDir(), ".search-initializing-"+index)); !os.IsNotExist(err) {
			t.Errorf("completed %s initialization retained its pending marker: %v", index, err)
		}
	}
}

func TestSRCH0SearchIndexRepairCommand(t *testing.T) {
	for _, scenario := range []struct{ name, refusal string }{
		{"existing", ""},
		{"missing", "existing trails index"},
		{"unfinished", "unfinished initialization"},
		{"invalid-timeout", "timeout must be positive"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			app := srch0StartupApp(t, false)
			m := srch0.NewMeili(t)
			m.Missing = scenario.name == "missing"
			if scenario.name == "existing" {
				trails, _ := app.FindAllRecords("trails")
				lists, _ := app.FindAllRecords("lists")
				actors, _ := app.FindAllRecords("activitypub_actors")
				for _, err := range []error{util.IndexTrails(app, trails, m.Client), util.IndexLists(app, lists, m.Client), util.IndexActors(actors, m.Client)} {
					if err != nil {
						t.Fatal(err)
					}
				}
				m.ClearCalls()
			}
			t.Setenv("MEILI_URL", m.Server.URL)
			t.Setenv("MEILI_MASTER_KEY", "")
			if scenario.name == "unfinished" {
				if err := os.WriteFile(filepath.Join(app.DataDir(), ".search-initializing-trails"), []byte("trails\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var output bytes.Buffer
			root := &cobra.Command{Use: "wanderer", SilenceUsage: true, SilenceErrors: true}
			root.SetOut(&output)
			root.SetErr(&output)
			args := []string{"search-index", "repair"}
			if scenario.name == "invalid-timeout" {
				args = append(args, "--timeout=0s")
			}
			root.SetArgs(args)
			setupCommands(&pocketbase.PocketBase{App: app, RootCmd: root})
			err := root.Execute()
			if scenario.refusal != "" {
				if err == nil || !strings.Contains(err.Error(), scenario.refusal) {
					t.Fatalf("repair should refuse %s: %v", scenario.name, err)
				}
				for _, request := range m.Calls() {
					if strings.Contains(request.Path, "/documents") {
						t.Error("refused repair submitted a document mutation")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for index, want := range map[string]int{"trails": 2, "lists": 1, "actors": 3} {
				if got := len(m.Snapshot()[index].(map[string]any)); got != want {
					t.Errorf("repair command did not process %s: got %d, want %d", index, got, want)
				}
			}
			if !strings.Contains(output.String(), "erfolgreich repariert") {
				t.Error("repair did not report successful completion")
			}
			mutations := 0
			for _, request := range m.Calls() {
				if strings.Contains(request.Path, "/documents") && (request.Method == "PUT" || request.Method == "POST") {
					mutations++
				}
			}
			if mutations == 0 {
				t.Error("repair command did not invoke metadata repair")
			}
		})
	}
}

// A successor may change active startup settings through an observed override;
// the historical profile itself remains available to the baseline engine suite.
func srch0StartupSettings(profile, observed map[string]any, index string) (meilisearch.Settings, error) {
	settings := map[string]any{}
	for key, value := range profile {
		settings[key] = value
	}
	if overrides, exists := observed["settings_overrides"]; exists {
		byIndex, ok := overrides.(map[string]any)
		if !ok {
			return meilisearch.Settings{}, fmt.Errorf("settings_overrides muss ein Objekt sein")
		}
		if override, exists := byIndex[index]; exists {
			values, ok := override.(map[string]any)
			if !ok {
				return meilisearch.Settings{}, fmt.Errorf("settings_overrides.%s muss ein Objekt sein", index)
			}
			for key, value := range values {
				settings[key] = value
			}
		}
	}
	var result meilisearch.Settings
	data, err := json.Marshal(settings)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)
	return result, err
}

func TestSRCH0StartupSettingsChange(t *testing.T) {
	profile := map[string]any{"filterableAttributes": []any{"author"}, "rankingRules": []any{"words", "sort"}}
	observed := map[string]any{"settings_overrides": map[string]any{"trails": map[string]any{"filterableAttributes": []any{"author", "new_field"}, "sortableAttributes": []any{"name"}}}}
	got, err := srch0StartupSettings(profile, observed, "trails")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(got.FilterableAttributes) != "[author new_field]" || fmt.Sprint(got.SortableAttributes) != "[name]" || fmt.Sprint(got.RankingRules) != "[words sort]" {
		t.Fatalf("overrides did not replace/add settings: %+v", got)
	}
	unchanged, err := srch0StartupSettings(profile, observed, "actors")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(unchanged.FilterableAttributes) != "[author]" || unchanged.SortableAttributes != nil {
		t.Fatalf("override leaked to another index: %+v", unchanged)
	}
	if len(profile["filterableAttributes"].([]any)) != 1 {
		t.Fatal("baseline profile mutated")
	}
}

func awaitSRCH0Background(jobs *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() { jobs.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(5 * time.Second):
		return false
	}
}
