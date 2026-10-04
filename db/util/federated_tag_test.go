package util

import (
	"bytes"
	"log/slog"
	"pocketbase/tagname"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

type federatedTagLogApp struct {
	core.App
	log *slog.Logger
}

func (a federatedTagLogApp) Logger() *slog.Logger { return a.log }

func TestResolveFederatedTag(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	collection := core.NewBaseCollection("tags")
	collection.Fields.Add(&core.TextField{Name: "name", Max: tagname.MaxLength, Pattern: tagname.Pattern})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}

	// Cleanup may leave duplicate and empty names with distinct historical IDs.
	for _, id := range []string{"zzzzzzzzzzzzzzz", "aaaaaaaaaaaaaaa"} {
		record := core.NewRecord(collection)
		record.Id = id
		record.Set("name", "ab")
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	empty := core.NewRecord(collection)
	if err := app.Save(empty); err != nil {
		t.Fatal(err)
	}

	record, err := ResolveFederatedTag(app, "a\t\x00b\x7f")
	if err != nil || record == nil || record.Id != "aaaaaaaaaaaaaaa" {
		t.Fatalf("collision resolution = %v, %v; want smallest existing ID", record, err)
	}
	if record, err := ResolveFederatedTag(app, "\n\t\x7f"); err != nil || record != nil {
		t.Fatalf("normalized-empty resolution = %v, %v; want intentional omission", record, err)
	}

	const literal = `🥾 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; ' || id != ''`
	record, err = ResolveFederatedTag(app, "\n"+literal+"\t")
	if err != nil || record == nil || record.GetString("name") != literal {
		t.Fatalf("plain text resolution = %v, %v", record, err)
	}
	again, err := ResolveFederatedTag(app, literal)
	if err != nil || again.Id != record.Id {
		t.Fatalf("literal lookup did not reuse the same tag: %v, %v", again, err)
	}
	if count, err := app.CountRecords("tags"); err != nil || count != 4 {
		t.Fatalf("tag count = %d, %v; want all historical IDs plus one new tag", count, err)
	}
	for _, id := range []string{"zzzzzzzzzzzzzzz", "aaaaaaaaaaaaaaa", empty.Id} {
		if _, err := app.FindRecordById("tags", id); err != nil {
			t.Fatalf("historical ID removed: %s: %v", id, err)
		}
	}
}

func TestResolveFederatedTagFailureDiagnostics(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	var logs bytes.Buffer
	loggedApp := federatedTagLogApp{App: app, log: slog.New(slog.NewTextHandler(&logs, nil))}
	const privateInput = "remote-value-must-not-appear-in-logs"
	if _, err := ResolveFederatedTag(loggedApp, privateInput); err == nil {
		t.Fatal("missing tags collection must report a lookup failure")
	}
	if !strings.Contains(logs.String(), "tag_collection_unavailable") || strings.Contains(logs.String(), privateInput) {
		t.Fatalf("unsafe or missing lookup diagnostic: %s", &logs)
	}
	collection := core.NewBaseCollection("tags")
	collection.Fields.Add(&core.TextField{Name: "name", Max: tagname.MaxLength, Pattern: tagname.Pattern})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	logs.Reset()
	if _, err := ResolveFederatedTag(loggedApp, privateInput+strings.Repeat("a", tagname.MaxLength)); err == nil {
		t.Fatal("overlong remote name must report a validation failure")
	}
	if !strings.Contains(logs.String(), "tag_save_failed") || strings.Contains(logs.String(), privateInput) || strings.Contains(logs.String(), "validation_") {
		t.Fatalf("unsafe or missing save diagnostic: %s", &logs)
	}
}
