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

	// Cleanup may leave duplicate, empty and space-only historical names.
	for _, id := range []string{"zzzzzzzzzzzzzzz", "aaaaaaaaaaaaaaa"} {
		record := core.NewRecord(collection)
		record.Id = id
		record.Set("name", "Berg Tour")
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	empty := core.NewRecord(collection)
	if err := app.Save(empty); err != nil {
		t.Fatal(err)
	}
	spaces := core.NewRecord(collection)
	spaces.Set("name", "  ")
	if err := app.Save(spaces); err != nil {
		t.Fatal(err)
	}

	record, err := ResolveFederatedTag(app, "Ber\x00g\tTou\x7fr")
	if err != nil || record == nil || record.Id != "aaaaaaaaaaaaaaa" {
		t.Fatalf("collision resolution = %v, %v; want smallest existing ID", record, err)
	}
	for _, name := range []string{"", "\x00\x01\x1f\x7f", "\t\n\v\f\r\x7f", "  "} {
		if record, err := ResolveFederatedTag(app, name); err != nil || record != nil {
			t.Fatalf("blank-name resolution = %v, %v; want intentional omission", record, err)
		}
	}

	const literal = `🥾 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; ' || id != ''`
	const spacedLiteral = " " + literal + " "
	record, err = ResolveFederatedTag(app, "\n"+literal+"\t")
	if err != nil || record == nil || record.GetString("name") != spacedLiteral {
		t.Fatalf("plain text resolution = %v, %v", record, err)
	}
	again, err := ResolveFederatedTag(app, spacedLiteral)
	if err != nil || again.Id != record.Id {
		t.Fatalf("literal lookup did not reuse the same tag: %v, %v", again, err)
	}
	if count, err := app.CountRecords("tags"); err != nil || count != 5 {
		t.Fatalf("tag count = %d, %v; want all historical IDs plus one new tag", count, err)
	}
	for _, id := range []string{"zzzzzzzzzzzzzzz", "aaaaaaaaaaaaaaa", empty.Id, spaces.Id} {
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
