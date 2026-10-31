package hooks

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

const wr10ChildEnv = "WANDERER_WR10_CHILD"

// TestDeleteActorHandlerSurvivesTaskWaitFailure checks that a failing
// Meilisearch task wait during actor deletion is logged instead of exiting. It
// runs in a child process because the failure mode is os.Exit(1).
func TestDeleteActorHandlerSurvivesTaskWaitFailure(t *testing.T) {
	if os.Getenv(wr10ChildEnv) != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestDeleteActorHandlerSurvivesTaskWaitFailure$", "-test.count=1")
		cmd.Env = append(os.Environ(), wr10ChildEnv+"=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child process did not survive the task wait failure: %v\n%s", err, out)
		}
		return
	}

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/indexes/actors/documents/"):
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"taskUid":1,"indexUid":"actors","status":"enqueued","type":"documentDeletion","enqueuedAt":"2026-10-01T00:00:00Z"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/tasks/1":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom","code":"internal","type":"internal","link":""}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer fake.Close()

	app := setupActorDeleteHooksTestApp(t)
	app.OnRecordAfterDeleteSuccess("activitypub_actors").BindFunc(DeleteActorHandler(meilisearch.New(fake.URL)))

	col, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(col)
	actor.Set("iri", "https://remote.example.com/api/v1/activitypub/user/gone")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}

	if err := app.Delete(actor); err != nil {
		t.Fatalf("delete actor: %v", err)
	}
	if _, err := app.FindRecordById("activitypub_actors", actor.Id); err == nil {
		t.Fatal("actor record still exists after delete")
	}
}
