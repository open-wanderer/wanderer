package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

// These tests run against a real Meilisearch, because the fakes can only
// repeat what their author believed about it: how it round-trips floats, when
// a cancel takes effect, and how it creates a missing index. They use the
// binary at WANDERER_TEST_MEILISEARCH, or ../search/meilisearch, and are
// skipped when neither exists.

const realMeiliKey = "real-meili-test-master-key-0123456789"

// startRealMeili starts a Meilisearch with an empty database on a free port.
func startRealMeili(t *testing.T) meilisearch.ServiceManager {
	t.Helper()
	binary := os.Getenv("WANDERER_TEST_MEILISEARCH")
	if binary == "" {
		binary = filepath.Join("..", "search", "meilisearch")
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skipf("no Meilisearch binary at %s; set WANDERER_TEST_MEILISEARCH", binary)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := listener.Addr().String()
	listener.Close()

	cmd := exec.Command(binary,
		"--master-key", realMeiliKey,
		"--http-addr", addr,
		"--db-path", filepath.Join(t.TempDir(), "data.ms"),
		"--no-analytics",
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	url := "http://" + addr
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(url + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("Meilisearch at %s did not come up", url)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return meilisearch.New(url, meilisearch.WithAPIKey(realMeiliKey))
}

// newRealSchemaApp builds the full schema from the migrations, skipping only
// those that configure Meilisearch.
func newRealSchemaApp(t *testing.T) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].File < migrations[j].File
	})
	for _, migration := range migrations {
		switch migration.File {
		case "1742167033_init_meilisearch.go", "1744651602_add_polyline.go", "1749831369_update_sortable_attributes.go":
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatalf("apply %s: %v", migration.File, err)
		}
	}
	return app
}

// seedRealisticTrails stores an author, trails whose numbers look like ones
// computed from GPX tracks, and lists summing them.
func seedRealisticTrails(t *testing.T, app core.App, count int) []*core.Record {
	t.Helper()
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	author := core.NewRecord(actors)
	author.Set("preferred_username", "gpxhiker")
	author.Set("iri", "https://example.com/api/v1/activitypub/user/gpxhiker")
	author.Set("username", "GPX Hiker")
	author.Set("inbox", "https://example.com/api/v1/activitypub/user/gpxhiker/inbox")
	author.Set("public_key", "-----BEGIN PUBLIC KEY-----\ntest\n-----END PUBLIC KEY-----")
	author.Set("is_local", true)
	if err := app.Save(author); err != nil {
		t.Fatal(err)
	}

	trailsCollection, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	random := rand.New(rand.NewPCG(1, 2))
	var trails []*core.Record
	for i := 0; i < count; i++ {
		trail := core.NewRecord(trailsCollection)
		lat := 46 + random.Float64()*2
		lon := 10 + random.Float64()*3
		trail.Set("name", fmt.Sprintf("Ridge %03d", i))
		trail.Set("author", author.Id)
		trail.Set("public", true)
		trail.Set("distance", random.Float64()*25000)
		trail.Set("elevation_gain", random.Float64()*1800)
		trail.Set("elevation_loss", random.Float64()*1800)
		trail.Set("duration", random.Float64()*36000)
		trail.Set("lat", lat)
		trail.Set("lon", lon)
		trail.Set("min_lat", lat-random.Float64()/50)
		trail.Set("max_lat", lat+random.Float64()/50)
		trail.Set("min_lon", lon-random.Float64()/50)
		trail.Set("max_lon", lon+random.Float64()/50)
		trail.Set("bounding_box_diagonal", random.Float64()*9000)
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
		trails = append(trails, trail)
	}

	listsCollection, err := app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i+3 <= len(trails); i += 3 {
		list := core.NewRecord(listsCollection)
		list.Set("name", fmt.Sprintf("Loops %d", i))
		list.Set("author", author.Id)
		list.Set("public", true)
		list.Set("trails", []string{trails[i].Id, trails[i+1].Id, trails[i+2].Id})
		if err := app.Save(list); err != nil {
			t.Fatal(err)
		}
	}
	return trails
}

// waitForIdleMeili waits until Meilisearch has no queued or running task.
func waitForIdleMeili(t *testing.T, client meilisearch.ServiceManager) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		tasks, err := client.GetTasks(&meilisearch.TasksQuery{Statuses: pendingSearchTaskStatuses, Limit: 1})
		if err != nil {
			t.Fatal(err)
		}
		if tasks.Total == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("Meilisearch did not go idle")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func captureLog(app core.App) (core.App, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return &searchInitLogApp{App: app, logger: logger}, &buf
}

func TestRealMeiliRepairLeavesIndexedNumbersAlone(t *testing.T) {
	client := startRealMeili(t)
	app := newRealSchemaApp(t)
	trails := seedRealisticTrails(t, app, 60)
	initMeilisearchConfig(client)
	logged, buf := captureLog(app)
	if err := initMeilisearchDocuments(logged, client); err != nil {
		t.Fatalf("initMeilisearchDocuments: %v\n%s", err, buf)
	}
	waitForIdleMeili(t, client)

	buf.Reset()
	if err := repairSearchIndexes(logged, client); err != nil {
		t.Fatalf("repairSearchIndexes: %v\n%s", err, buf)
	}
	for _, index := range []string{"trails", "lists", "actors"} {
		if !strings.Contains(buf.String(), "Search repair "+index+": 0 documents rewritten") {
			t.Fatalf("log = %s; want nothing rewritten in %s right after a rebuild", buf, index)
		}
	}

	// The tolerance must not hide a real change.
	trails[0].Set("distance", trails[0].GetFloat("distance")+0.001)
	if err := app.UnsafeWithoutHooks().Save(trails[0]); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := repairSearchIndexes(logged, client); err != nil {
		t.Fatalf("repairSearchIndexes: %v\n%s", err, buf)
	}
	if !strings.Contains(buf.String(), "Search repair trails: 1 documents rewritten") {
		t.Fatalf("log = %s; want the changed trail rewritten", buf)
	}
}

func TestRealMeiliTimedOutCancelKeepsLiveWrites(t *testing.T) {
	client := startRealMeili(t)
	app := newRealSchemaApp(t)
	trails := seedRealisticTrails(t, app, 6)
	initMeilisearchConfig(client)
	logged, buf := captureLog(app)
	if err := initMeilisearchDocuments(logged, client); err != nil {
		t.Fatalf("initMeilisearchDocuments: %v\n%s", err, buf)
	}
	waitForIdleMeili(t, client)

	// A long batch on another index holds up the queue, as an old rebuild
	// backlog does on the first start after the upgrade.
	backlog := make([]map[string]any, 300000)
	for i := range backlog {
		backlog[i] = map[string]any{"id": fmt.Sprintf("b%d", i), "text": fmt.Sprintf("backlog document %d", i)}
	}
	if _, err := client.Index("backlog").AddDocuments(backlog, &meilisearch.DocumentOptions{PrimaryKey: meilisearch.StringPtr("id")}); err != nil {
		t.Fatal(err)
	}

	// A trail is made private while the queue is held up: the record is
	// saved and its hook's update waits in the queue.
	private := trails[0]
	private.Set("public", false)
	if err := app.Save(private); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Index("trails").UpdateDocuments([]map[string]any{{"id": private.Id, "public": false}}, nil); err != nil {
		t.Fatal(err)
	}

	// The next start rebuilds trails, and its cancel cannot finish in time.
	versions := readSearchIndexVersions(app)
	state := versions["trails"]
	state.Version--
	versions["trails"] = state
	if err := writeSearchIndexVersions(app, versions); err != nil {
		t.Fatal(err)
	}
	timeout := searchTaskCancelTimeout
	searchTaskCancelTimeout = 10 * time.Millisecond
	t.Cleanup(func() { searchTaskCancelTimeout = timeout })

	buf.Reset()
	_ = initMeilisearchDocuments(logged, client)
	waitForIdleMeili(t, client)

	var document map[string]any
	if err := client.Index("trails").GetDocument(private.Id, nil, &document); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Cancel of queued trails tasks still pending") {
		t.Fatalf("log = %s; want the cancel to have timed out behind the backlog", buf)
	}
	if document["public"] != false {
		t.Fatalf("stored public = %v; want the trail made private to stay private\n%s", document["public"], buf)
	}
}

func TestRealMeiliRepairRecreatesMissingIndex(t *testing.T) {
	client := startRealMeili(t)
	app := newRealSchemaApp(t)
	trails := seedRealisticTrails(t, app, 12)
	initMeilisearchConfig(client)
	logged, buf := captureLog(app)
	if err := initMeilisearchDocuments(logged, client); err != nil {
		t.Fatalf("initMeilisearchDocuments: %v\n%s", err, buf)
	}
	waitForIdleMeili(t, client)

	// Meilisearch's data is wiped while the backend keeps running.
	task, err := client.DeleteIndex("trails")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.WaitForTask(task.TaskUID, 0); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- repairSearchIndexes(logged, client) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("repairSearchIndexes: %v\n%s", err, buf)
		}
	case <-ctx.Done():
		t.Fatalf("repair into a missing index did not finish\n%s", buf)
	}
	waitForIdleMeili(t, client)

	index, err := client.GetIndex("trails")
	if err != nil {
		t.Fatalf("trails index: %v", err)
	}
	if index.PrimaryKey != "id" {
		t.Fatalf("primary key = %q; want id", index.PrimaryKey)
	}
	stats, err := client.Index("trails").GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.NumberOfDocuments != int64(len(trails)) {
		t.Fatalf("documents = %d; want %d", stats.NumberOfDocuments, len(trails))
	}
	filterable, err := client.Index("trails").GetFilterableAttributes()
	if err != nil {
		t.Fatal(err)
	}
	if filterable == nil || !slices.Contains(*filterable, "public") {
		t.Fatalf("filterable attributes = %v; want the index settings restored", filterable)
	}
}

func TestSearchIndexStateIsLeftOutOfBackups(t *testing.T) {
	app := newRealSchemaApp(t)
	registerSearchBackupHooks(app)
	if err := writeSearchIndexVersions(app, map[string]searchIndexState{"trails": {Version: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := app.CreateBackup(context.Background(), "search.zip"); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(filepath.Join(app.DataDir(), core.LocalBackupsDirName, "search.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if strings.Contains(file.Name, searchIndexVersionsFile) {
			t.Fatalf("backup contains %s, so a restore would keep the indexes from before it", file.Name)
		}
	}
}

func TestRealMeiliRepairCorrectsChangesQueuedAheadOfItsBatch(t *testing.T) {
	client := startRealMeili(t)
	app := newRealSchemaApp(t)
	trails := seedRealisticTrails(t, app, 12)
	initMeilisearchConfig(client)
	logged, buf := captureLog(app)
	if err := initMeilisearchDocuments(logged, client); err != nil {
		t.Fatalf("initMeilisearchDocuments: %v\n%s", err, buf)
	}
	waitForIdleMeili(t, client)

	// Every trail document is stale, so the repair rewrites the whole page.
	stale := make([]map[string]any, len(trails))
	for i, trail := range trails {
		stale[i] = map[string]any{"id": trail.Id, "name": "stale"}
	}
	if _, err := client.Index("trails").UpdateDocuments(stale, nil); err != nil {
		t.Fatal(err)
	}
	waitForIdleMeili(t, client)

	// Right before the batch is queued, one trail is made private and another
	// deleted, and their hooks queue their writes ahead of the batch.
	private, deleted := trails[0], trails[1]
	beforeSearchRepairWrite = func() {
		beforeSearchRepairWrite = func() {}
		private.Set("public", false)
		if err := app.Save(private); err != nil {
			t.Error(err)
		}
		if _, err := client.Index("trails").UpdateDocuments([]map[string]any{{"id": private.Id, "public": false}}, nil); err != nil {
			t.Error(err)
		}
		if err := app.Delete(deleted); err != nil {
			t.Error(err)
		}
		if _, err := client.Index("trails").DeleteDocument(deleted.Id, nil); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { beforeSearchRepairWrite = func() {} })

	buf.Reset()
	if err := repairSearchIndexes(logged, client); err != nil {
		t.Fatalf("repairSearchIndexes: %v\n%s", err, buf)
	}
	waitForIdleMeili(t, client)

	var document map[string]any
	if err := client.Index("trails").GetDocument(private.Id, nil, &document); err != nil {
		t.Fatal(err)
	}
	if document["public"] != false {
		t.Fatalf("stored public = %v; want the trail made private to stay private\n%s", document["public"], buf)
	}
	if err := client.Index("trails").GetDocument(deleted.Id, nil, &document); !searchIndexNotFound(err) && !strings.Contains(fmt.Sprint(err), "document_not_found") {
		t.Fatalf("deleted trail document = %v, %v; want it gone", document, err)
	}
	if err := client.Index("trails").GetDocument(trails[2].Id, nil, &document); err != nil || document["name"] == "stale" {
		t.Fatalf("trail %s = %v, %v; want it repaired", trails[2].Id, document, err)
	}
}

func TestRealMeiliRepairRestoresSettingsOfIndexRecreatedByLiveWrite(t *testing.T) {
	client := startRealMeili(t)
	app := newRealSchemaApp(t)
	trails := seedRealisticTrails(t, app, 12)
	initMeilisearchConfig(client)
	logged, buf := captureLog(app)
	if err := initMeilisearchDocuments(logged, client); err != nil {
		t.Fatalf("initMeilisearchDocuments: %v\n%s", err, buf)
	}
	waitForIdleMeili(t, client)

	// Meilisearch's data is wiped while the backend keeps running, and a
	// live edit recreates the index before the repair runs.
	task, err := client.DeleteIndex("trails")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.WaitForTask(task.TaskUID, 0); err != nil {
		t.Fatal(err)
	}
	edit, err := client.Index("trails").UpdateDocuments([]map[string]any{{"id": trails[0].Id, "name": trails[0].GetString("name")}}, util.SearchWriteOptions)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.WaitForTask(edit.TaskUID, 0); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	if err := repairSearchIndexes(logged, client); err != nil {
		t.Fatalf("repairSearchIndexes: %v\n%s", err, buf)
	}
	waitForIdleMeili(t, client)

	filterable, err := client.Index("trails").GetFilterableAttributes()
	if err != nil {
		t.Fatal(err)
	}
	if filterable == nil || !slices.Contains(*filterable, "public") {
		t.Fatalf("filterable attributes = %v; want the settings applied by the repair", filterable)
	}
	stats, err := client.Index("trails").GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.NumberOfDocuments != int64(len(trails)) {
		t.Fatalf("documents = %d; want %d", stats.NumberOfDocuments, len(trails))
	}

	// The repair filled the recreated index, so the next start keeps it.
	createdAt, err := searchIndexCreatedAt(client, "trails")
	if err != nil {
		t.Fatal(err)
	}
	if recorded := readSearchIndexVersions(app)["trails"].CreatedAt; recorded != createdAt {
		t.Fatalf("recorded creation time = %q; want the recreated index's %q", recorded, createdAt)
	}
}
