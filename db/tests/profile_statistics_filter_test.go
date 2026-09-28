package util

import (
	"fmt"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestPocketBaseBackRelationInequalityUsesAllMatchSemantics(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	actors := core.NewBaseCollection("actors")
	actors.Fields.Add(&core.TextField{Name: "name", Required: true})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "name", Required: true})
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	summitLogs := core.NewBaseCollection("summit_logs")
	summitLogs.Fields.Add(
		&core.RelationField{
			Name:         "author",
			CollectionId: actors.Id,
			MaxSelect:    1,
			Required:     true,
		},
		&core.RelationField{
			Name:         "trail",
			CollectionId: trails.Id,
			MaxSelect:    1,
			Required:     true,
		},
	)
	if err := app.Save(summitLogs); err != nil {
		t.Fatal(err)
	}

	createActor := func(name string) *core.Record {
		t.Helper()
		record := core.NewRecord(actors)
		record.Set("name", name)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record
	}
	owner := createActor("owner")
	otherUser := createActor("other-user")

	createTrail := func(name string) *core.Record {
		t.Helper()
		record := core.NewRecord(trails)
		record.Set("name", name)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record
	}
	withoutLogs := createTrail("without-logs")
	withForeignLog := createTrail("with-foreign-log")
	withOwnerLog := createTrail("with-owner-log")
	withMixedLogs := createTrail("with-mixed-logs")

	createSummitLog := func(trail *core.Record, author *core.Record) {
		t.Helper()
		record := core.NewRecord(summitLogs)
		record.Set("trail", trail.Id)
		record.Set("author", author.Id)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	createSummitLog(withForeignLog, otherUser)
	createSummitLog(withOwnerLog, owner)
	createSummitLog(withMixedLogs, otherUser)
	createSummitLog(withMixedLogs, owner)

	// The fallback filter built in web/src/lib/server/profile_statistics.ts
	// relies on regular != being an all-match for a multi-valued back relation
	// and also matching an empty relation.
	records, err := app.FindRecordsByFilter(
		trails,
		fmt.Sprintf("summit_logs_via_trail.author!='%s'", owner.Id),
		"name",
		0,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}

	got := make(map[string]bool, len(records))
	for _, record := range records {
		got[record.Id] = true
	}
	for _, test := range []struct {
		name   string
		trail  *core.Record
		wanted bool
	}{
		{name: "empty relation", trail: withoutLogs, wanted: true},
		{name: "only another author's log", trail: withForeignLog, wanted: true},
		{name: "only the current author's log", trail: withOwnerLog, wanted: false},
		{name: "mixed authors", trail: withMixedLogs, wanted: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got[test.trail.Id] != test.wanted {
				t.Errorf("included = %t, want %t", got[test.trail.Id], test.wanted)
			}
		})
	}

	if len(records) != 2 {
		names := make([]string, 0, len(records))
		for _, record := range records {
			names = append(names, record.GetString("name"))
		}
		t.Errorf(
			"back-relation filter returned %v; want without-logs and with-foreign-log",
			names,
		)
	}
}
