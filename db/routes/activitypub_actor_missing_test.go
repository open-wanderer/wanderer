package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestUnknownLocalActorAPIKeepsNotFoundStatus(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	drainTestAppStartupLogs(t, app)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "preferred_username"},
		&core.TextField{Name: "domain"},
		&core.BoolField{Name: "is_local"},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	event := &core.RequestEvent{App: app}
	event.Request = httptest.NewRequest(http.MethodGet, "/activitypub/actor?resource=nobody", nil)
	event.Response = httptest.NewRecorder()
	var apiError *router.ApiError
	if err := ActivitypubActor(event); !errors.As(err, &apiError) || apiError.Status != http.StatusNotFound {
		t.Fatalf("unknown local actor status = %v", err)
	}
}
