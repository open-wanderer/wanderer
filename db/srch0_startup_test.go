package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"pocketbase/internal/srch0"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestSRCH0Startup(t *testing.T) {
	// initPlugins resolves its directory relative to the working directory.
	// Keep startup discovery inside a synthetic directory as well as a temp DB.
	t.Chdir(t.TempDir())
	for _, c := range srch0.Cases(t, "mutation", "go-startup") {
		t.Run(c.ID, func(t *testing.T) {
			d := srch0.Data(t)
			// Take a closed relational subset of the shared dataset, or its empty
			// boundary. No developer database or remote origin is consulted.
			for _, collection := range []string{"trails", "lists", "trail_share", "trail_like", "list_share"} {
				kept := []srch0.Object{}
				for _, r := range d.Records[collection] {
					if collection == "trails" && (r["id"] == "public-alpine" || r["id"] == "public-lake") || collection == "lists" && r["id"] == "list-local" {
						kept = append(kept, r)
					}
				}
				d.Records[collection] = kept
			}
			if c.Input["empty_source"] == true {
				for _, collection := range []string{"trails", "lists", "activitypub_actors"} {
					d.Records[collection] = nil
				}
			}
			app := srch0.App(t, d)
			m := srch0.NewMeili(t)
			m.Missing = c.Input["indexes"] == "missing"
			m.FailDelete = c.Input["fail_delete"] == true
			if c.Input["indexes"] == "existing" {
				for _, index := range []string{"trails", "lists", "actors"} {
					m.Documents[index]["prior"] = srch0.Object{"id": "prior"}
				}
			}
			type phase struct {
				request srch0.Request
				resume  chan struct{}
			}
			phases := make(chan phase)
			cancel := make(chan struct{})
			m.Before = func(req srch0.Request) {
				if !strings.Contains(req.Path, "/documents") {
					return
				}
				p := phase{req, make(chan struct{})}
				select {
				case phases <- p:
				case <-cancel:
					return
				}
				select {
				case <-p.resume:
				case <-cancel:
					return
				}
			}
			var jobs sync.WaitGroup
			background := func(task func()) {
				jobs.Add(1)
				go func() { defer jobs.Done(); task() }()
			}
			// The injected launcher preserves the real goroutine but gives this
			// test an explicit completion boundary before the temporary DB closes.
			defer func() {
				close(cancel)
				if !awaitSRCH0Background(&jobs) {
					t.Error("startup jobs did not stop before database cleanup")
				}
			}()
			r := router.NewRouter(func(w http.ResponseWriter, r *http.Request) (*core.RequestEvent, router.EventCleanupFunc) {
				e := &core.RequestEvent{App: app}
				e.Request = r
				e.Response = w
				return e, nil
			})
			se := &core.ServeEvent{App: app, Router: r}
			opened := false
			h := hook.Hook[*core.ServeEvent]{}
			h.BindFunc(onBeforeServeHandlerWithBackground(m.Client, background))
			if err := h.Trigger(se, func(*core.ServeEvent) error { opened = true; return nil }); err != nil {
				t.Fatal(err)
			}
			mux, err := r.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			availability := []any{}
			wantPhases := c.Observed["mutation_state"].(map[string]any)["availability"].([]any)
			for range wantPhases {
				var p phase
				select {
				case p = <-phases:
				case <-time.After(5 * time.Second):
					t.Fatal("startup stopped before declared phase")
				}
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, httptest.NewRequest("GET", "/search/token", nil))
				counts := map[string]int{}
				for index, v := range m.Snapshot() {
					counts[index] = len(v.(map[string]any))
				}
				availability = append(availability, map[string]any{"phase": p.request.Method + " " + p.request.Path, "search_token_status": response.Code, "document_counts": counts})
				close(p.resume)
			}
			if !awaitSRCH0Background(&jobs) {
				t.Fatal("startup produced an unobserved phase or did not complete")
			}
			// Settings iteration is a Go map: only per-index configuration and the
			// ordered document rebuild are stable. Task UIDs are never a golden.
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
			got := map[string]any{"serve_next_called": opened, "settings_match_profiles": len(settings) == 3, "terminal_task_reads": waits, "rebuild_requests": rebuild, "availability": availability}
			srch0.Assert(t, c, got, c.Observed["mutation_state"])
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
