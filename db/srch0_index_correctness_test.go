package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"pocketbase/internal/srch0"
	"pocketbase/util"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Meilisearch treats metadata updates as upserts. A new search hit must carry
// source identity, visibility and content as well as the changed metadata.
func assertSRCH0NewMetadataDocumentsComplete(t *testing.T, m *srch0.Meili, before srch0.Object) {
	t.Helper()
	for _, index := range []string{"trails", "lists"} {
		for id, value := range m.Snapshot()[index].(map[string]any) {
			if _, existed := before[index].(map[string]any)[id]; existed {
				continue
			}
			doc := value.(map[string]any)
			fields := []string{"id", "author", "author_name", "author_avatar", "name", "description", "public", "shares", "distance", "duration", "elevation_gain", "elevation_loss"}
			if index == "trails" {
				fields = append(fields, "difficulty", "category", "tags", "likes", "like_count", "_geo")
			} else {
				fields = append(fields, "trails")
			}
			for _, field := range fields {
				if _, exists := doc[field]; !exists {
					t.Fatalf("Metadatenänderung erzeugt unvollständigen Treffer %s/%s: %s fehlt", index, id, field)
				}
			}
		}
	}
}

// These properties come from saved source relations, independently of fixture
// observations and production projector helpers. A historical golden cannot
// authorize losing a remaining grant or displaying stale metadata.
func assertSRCH0CurrentIndex(t *testing.T, app core.App, m *srch0.Meili, before srch0.Object) {
	t.Helper()
	after := m.Snapshot()
	for _, index := range []string{"trails", "lists"} {
		for id := range before[index].(map[string]any) {
			source, err := app.FindRecordById(index, id)
			if err != nil { // deletion is checked by the mutation's full observation
				continue
			}
			doc, ok := after[index].(map[string]any)[id].(map[string]any)
			if !ok {
				t.Fatalf("Gespeichertes %s/%s fehlt im Index", index, id)
			}
			author := srch0.Record(t, app, "activitypub_actors", source.GetString("author"))
			assertSRCH0Field(t, index, id, "author_name", doc["author_name"], author.GetString("preferred_username"))
			assertSRCH0Field(t, index, id, "author_avatar", doc["author_avatar"], author.GetString("icon"))
			if index == "trails" {
				categoryName, categoryIcon := "", ""
				if category := source.GetString("category"); category != "" {
					r := srch0.Record(t, app, "categories", category)
					categoryName, categoryIcon = r.GetString("name"), r.GetString("icon")
				}
				assertSRCH0Field(t, index, id, "category", doc["category"], categoryName)
				assertSRCH0Field(t, index, id, "category_icon", doc["category_icon"], categoryIcon)
				tags := []string{}
				for _, tag := range source.GetStringSlice("tags") {
					tags = append(tags, srch0.Record(t, app, "tags", tag).GetString("name"))
				}
				assertSRCH0Field(t, index, id, "tags", sortedSRCH0Strings(t, doc["tags"]), sortedSRCH0Strings(t, tags))
			}
			collection, relation := "trail_share", "trail"
			if index == "lists" {
				collection, relation = "list_share", "list"
			}
			shares, err := app.FindAllRecords(collection, dbx.HashExp{relation: id})
			if err != nil {
				t.Fatal(err)
			}
			actors := []string{}
			for _, share := range shares {
				if actor := share.GetString("actor"); actor != "" {
					actors = append(actors, actor)
				}
			}
			assertSRCH0Field(t, index, id, "shares", sortedSRCH0Strings(t, doc["shares"]), sortedSRCH0Strings(t, actors))
		}
	}
}

func sortedSRCH0Strings(t *testing.T, value any) []string {
	t.Helper()
	b, _ := json.Marshal(value)
	result := []string{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatal("Erwartetes Stringarray fehlt:", err)
	}
	sort.Strings(result)
	return result
}

func assertSRCH0Field(t *testing.T, index, id, field string, got, want any) {
	t.Helper()
	gb, _ := json.Marshal(got)
	wb, _ := json.Marshal(want)
	if string(gb) != string(wb) {
		t.Fatalf("Index muss aktuelle DB-Metadaten und verbleibende Freigaben erhalten: %s/%s.%s = %s, DB verlangt %s", index, id, field, gb, wb)
	}
}

func TestSRCH0ShareDeletePreservesRemainingGrants(t *testing.T) {
	d := srch0.Data(t)
	for _, index := range []string{"trails", "lists"} {
		t.Run(index, func(t *testing.T) {
			app := srch0.App(t, d)
			m := srch0.NewMeili(t)
			collection, relation, label, shareID := "trail_share", "trail", "shared-alice", "share001"
			if index == "lists" {
				collection, relation, label, shareID = "list_share", "list", "list-private", "listshare001"
			}
			coll, _ := app.FindCollectionByNameOrId(collection)
			remaining := core.NewRecord(coll)
			srch0.Assign(t, remaining, srch0.Object{relation: label, "actor": "bob", "permission": "view"})
			srch0.Save(t, app, remaining)
			r := srch0.Record(t, app, index, label)
			var err error
			if index == "trails" {
				err = util.IndexTrails(app, []*core.Record{r}, m.Client)
			} else {
				err = util.IndexLists(app, []*core.Record{r}, m.Client)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := m.Snapshot()
			setupEventHandlers(&pocketbase.PocketBase{App: app}, m.Client)
			blocked := app.OnRecordDelete(collection).BindFunc(func(*core.RecordEvent) error { return errors.New("delete rejected") })
			m.ClearCalls()
			if err := app.Delete(srch0.Record(t, app, collection, shareID)); err == nil {
				t.Fatal("Testgrenze muss den Delete ablehnen")
			}
			if len(m.Calls()) != 0 {
				t.Fatal("Abgelehnter Delete darf keine Indexfreigabe ändern")
			}
			app.OnRecordDelete(collection).Unbind(blocked)
			if err := app.Delete(srch0.Record(t, app, collection, shareID)); err != nil {
				t.Fatal(err)
			}
			assertSRCH0CurrentIndex(t, app, m, before)
			additional := core.NewRecord(coll)
			srch0.Assign(t, additional, srch0.Object{relation: label, "actor": "remote", "permission": "view"})
			srch0.Save(t, app, additional)
			assertSRCH0CurrentIndex(t, app, m, before)
			additional.Set("actor", srch0.ID("alice"))
			srch0.Save(t, app, additional)
			assertSRCH0CurrentIndex(t, app, m, before)
			if err := app.Delete(additional); err != nil {
				t.Fatal(err)
			}
			if err := app.Delete(remaining); err != nil {
				t.Fatal(err)
			}
			assertSRCH0CurrentIndex(t, app, m, before)
		})
	}
}

func TestSRCH0UserMetadataUsesActorFanOut(t *testing.T) {
	app := srch0.App(t, srch0.Data(t))
	m := srch0.NewMeili(t)
	actor := srch0.Record(t, app, "activitypub_actors", "alice")
	actor.Set("icon", "https://wanderer.invalid/previous-avatar.jpg")
	srch0.Save(t, app, actor)
	trail := srch0.Record(t, app, "trails", "public-alpine")
	list := srch0.Record(t, app, "lists", "list-local")
	if err := util.IndexTrails(app, []*core.Record{trail}, m.Client); err != nil {
		t.Fatal(err)
	}
	if err := util.IndexLists(app, []*core.Record{list}, m.Client); err != nil {
		t.Fatal(err)
	}
	before := m.Snapshot()
	setupEventHandlers(&pocketbase.PocketBase{App: app}, m.Client)
	m.ClearCalls()
	user := srch0.Record(t, app, "users", "useralice")
	// The actor now has an icon; the user has no avatar. Saving the user
	// removes the old icon through the real user -> actor -> index hook chain.
	srch0.Save(t, app, user)
	assertSRCH0CurrentIndex(t, app, m, before)
	for _, index := range []string{"trails", "lists"} {
		count := 0
		for _, request := range m.Calls() {
			if request.Path == "/indexes/"+index+"/documents" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("User- und Actorhook müssen %s genau einmal aktualisieren, got %d", index, count)
		}
	}
}

func TestSRCH0TaxonomyDeleteClearsProjectedMetadata(t *testing.T) {
	for _, collection := range []string{"tags", "categories"} {
		t.Run(collection, func(t *testing.T) {
			app := srch0.App(t, srch0.Data(t))
			m := srch0.NewMeili(t)
			r := srch0.Record(t, app, "trails", "public-alpine")
			if err := util.IndexTrails(app, []*core.Record{r}, m.Client); err != nil {
				t.Fatal(err)
			}
			before := m.Snapshot()
			setupEventHandlers(&pocketbase.PocketBase{App: app}, m.Client)
			label := "scenic"
			if collection == "categories" {
				label = "hiking"
			}
			if err := app.Delete(srch0.Record(t, app, collection, label)); err != nil {
				t.Fatal(err)
			}
			assertSRCH0CurrentIndex(t, app, m, before)
		})
	}
}

func TestSRCH0ActorMetadataUpdatesEveryBatch(t *testing.T) {
	app := srch0.App(t, srch0.Data(t))
	m := srch0.NewMeili(t)
	template := srch0.Record(t, app, "trails", "public-alpine")
	trails := make([]*core.Record, 201)
	for i := range trails {
		r := core.NewRecord(template.Collection())
		values := template.FieldsData()
		delete(values, "id")
		srch0.Assign(t, r, values)
		srch0.Save(t, app, r)
		trails[i] = r
	}
	if err := util.IndexTrails(app, trails, m.Client); err != nil {
		t.Fatal(err)
	}
	before := m.Snapshot()
	setupEventHandlers(&pocketbase.PocketBase{App: app}, m.Client)
	actor := srch0.Record(t, app, "activitypub_actors", "alice")
	actor.Set("preferred_username", "Updated across all batches")
	srch0.Save(t, app, actor)
	assertSRCH0CurrentIndex(t, app, m, before)
}

func TestSRCH0ActorMetadataUpdatesRemoteListWithoutFetching(t *testing.T) {
	app := srch0.App(t, srch0.Data(t))
	m := srch0.NewMeili(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	r := srch0.Record(t, app, "lists", "list-remote-stub")
	r.Set("iri", server.URL+"/lists/"+r.Id)
	srch0.Save(t, app, r)
	actor := srch0.Record(t, app, "activitypub_actors", "remote")
	doc := map[string]any{"id": r.Id, "author_name": actor.GetString("preferred_username"), "author_avatar": actor.GetString("icon"), "shares": []string{}, "distance": 12345, "trails": 7}
	if _, err := m.Client.Index("lists").AddDocuments([]map[string]any{doc}, nil); err != nil {
		t.Fatal(err)
	}
	before := m.Snapshot()
	setupEventHandlers(&pocketbase.PocketBase{App: app}, m.Client)
	actor.Set("preferred_username", "Updated remote actor")
	actor.Set("icon", "https://remote.invalid/new-avatar.png")
	srch0.Save(t, app, actor)
	assertSRCH0CurrentIndex(t, app, m, before)
	after := m.Snapshot()["lists"].(map[string]any)[r.Id].(map[string]any)
	assertSRCH0Field(t, "lists", r.Id, "distance", after["distance"], 12345)
	assertSRCH0Field(t, "lists", r.Id, "trails", after["trails"], 7)
	if requests.Load() != 0 {
		t.Fatalf("Reine Actor-Metadatenänderung hat %d Remote-Listenabrufe ausgelöst", requests.Load())
	}
}

func TestSRCH0MissingRemoteListMetadataFailsWithoutPartialHit(t *testing.T) {
	app := srch0.App(t, srch0.Data(t))
	m := srch0.NewMeili(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	r := srch0.Record(t, app, "lists", "list-remote-stub")
	r.Set("iri", server.URL+"/lists/"+r.Id)
	srch0.Save(t, app, r)
	setupEventHandlers(&pocketbase.PocketBase{App: app}, m.Client)
	actor := srch0.Record(t, app, "activitypub_actors", "remote")
	actor.Set("preferred_username", "Updated remote actor")
	err := app.Save(actor)
	if err == nil || !strings.Contains(err.Error(), "missing remote list index document "+r.Id) {
		t.Fatalf("Fehlende Remote-Liste muss die Metadatenaktualisierung gezielt ablehnen, got %v", err)
	}
	if _, exists := m.Snapshot()["lists"].(map[string]any)[r.Id]; exists {
		t.Fatal("Fehlende Remote-Liste darf nicht als Teil-Dokument entstehen")
	}
	if requests.Load() != 0 {
		t.Fatalf("Metadatenaktualisierung hat %d Remote-Abrufe ausgelöst", requests.Load())
	}
}
