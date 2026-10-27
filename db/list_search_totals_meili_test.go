package main

import (
	"testing"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

func TestRealMeiliPublicListTotalsExcludeUnreadableTrails(t *testing.T) {
	client := startRealMeili(t)
	app := newRealSchemaApp(t)
	trails := seedRealisticTrails(t, app, 3)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	other := core.NewRecord(actors)
	other.Set("iri", "https://remote.example/api/v1/activitypub/user/other")
	other.Set("preferred_username", "other")
	other.Set("username", "Other")
	other.Set("inbox", other.GetString("iri")+"/inbox")
	other.Set("public_key", "test-key")
	if err := app.Save(other); err != nil {
		t.Fatal(err)
	}
	for i, trail := range trails {
		trail.Set("distance", []float64{100, 200, 900000}[i])
		trail.Set("elevation_gain", []float64{10, 20, 90000}[i])
		trail.Set("duration", []float64{1, 2, 9000}[i])
		trail.Set("public", i == 0)
		if i == 2 {
			trail.Set("author", other.Id)
		}
		if err := app.Save(trail); err != nil {
			t.Fatal(err)
		}
	}
	lists, err := app.FindAllRecords("lists")
	if err != nil || len(lists) != 1 {
		t.Fatalf("list fixture: %v %v", lists, err)
	}
	list := lists[0]
	initMeilisearchConfig(client)
	if _, err := initMeilisearchDocuments(app, client); err != nil {
		t.Fatal(err)
	}
	assertTotals := func() {
		t.Helper()
		waitForIdleMeili(t, client)
		var document map[string]any
		if err := client.Index("lists").GetDocument(list.Id, nil, &document); err != nil {
			t.Fatal(err)
		}
		if document["public"] != true || document["trails"] != float64(2) || document["distance"] != float64(300) || document["elevation_gain"] != float64(30) || document["duration"] != float64(3) {
			t.Fatalf("public search totals include an unreadable trail: %v", document)
		}
	}
	assertTotals()
	author, err := app.FindRecordById("activitypub_actors", list.GetString("author"))
	if err != nil {
		t.Fatal(err)
	}
	trails[2].Set("distance", 1800000)
	if err := app.Save(trails[2]); err != nil {
		t.Fatal(err)
	}
	if err := util.UpdateList(app, list, author, client); err != nil {
		t.Fatal(err)
	}
	assertTotals()
}
