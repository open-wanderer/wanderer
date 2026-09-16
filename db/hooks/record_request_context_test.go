package hooks

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestTrailAPIUpdatePropagatesCancellationToMediaAndPreventsPublication(t *testing.T) {
	for _, formData := range []bool{false, true} {
		t.Run(map[bool]string{false: "JSON", true: "multipart"}[formData], func(t *testing.T) {
			app, trail, mux := newTrailRequestContextTestApp(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			mediaCancelled := make(chan struct{})
			media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cancel()
				<-r.Context().Done()
				close(mediaCancelled)
			}))
			defer media.Close()

			// Simulate the publication hook's synchronous HTTP download through
			// the actual PocketBase collection upsert, not a direct wrapper call.
			var mediaErr error
			app.OnRecordUpdate("trails").BindFunc(func(e *core.RecordEvent) error {
				if e.Context != ctx {
					return errors.New("record hook did not receive request context")
				}
				request, err := http.NewRequestWithContext(e.Context, http.MethodGet, media.URL, nil)
				if err != nil {
					return err
				}
				client := &http.Client{Timeout: time.Second}
				response, err := client.Do(request)
				if response != nil {
					response.Body.Close()
				}
				mediaErr = err
				if err != nil {
					return err
				}
				return e.Next()
			})
			published := false
			app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
				published = true
				return e.Next()
			})

			request := trailPublishRequest(t, trail.Id, formData).WithContext(ctx)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			if recorder.Code < 400 || !errors.Is(mediaErr, context.Canceled) {
				t.Fatalf("expected canceled publication, got HTTP %d, media error %v: %s", recorder.Code, mediaErr, recorder.Body.String())
			}
			select {
			case <-mediaCancelled:
			case <-time.After(time.Second):
				t.Fatal("upstream media request was not canceled")
			}
			stored, err := app.FindRecordById("trails", trail.Id)
			if err != nil || stored.GetBool("public") || published {
				t.Fatalf("canceled update persisted or announced publication: record=%v, published=%t, error=%v", stored, published, err)
			}
		})
	}
}

func TestTrailAPIUpdateChecksCancellationAtFinalDatabaseWrite(t *testing.T) {
	app, trail, mux := newTrailRequestContextTestApp(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.OnRecordUpdate("trails").BindFunc(func(e *core.RecordEvent) error {
		cancel()
		// Even if a record hook doesn't explicitly return ctx.Err(), the
		// database update must still use the canceled request context.
		return e.Next()
	})
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, trailPublishRequest(t, trail.Id, false).WithContext(ctx))
	stored, err := app.FindRecordById("trails", trail.Id)
	if recorder.Code < 400 || err != nil || stored.GetBool("public") {
		t.Fatalf("canceled write succeeded: HTTP %d, record=%v, error=%v", recorder.Code, stored, err)
	}
}

func TestTrailAPIUpdateStillPublishesWithActiveRequest(t *testing.T) {
	app, trail, mux := newTrailRequestContextTestApp(t)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, trailPublishRequest(t, trail.Id, false))
	stored, err := app.FindRecordById("trails", trail.Id)
	if recorder.Code != http.StatusOK || err != nil || !stored.GetBool("public") {
		t.Fatalf("active update failed: HTTP %d, record=%v, error=%v: %s", recorder.Code, stored, err, recorder.Body.String())
	}
}

func newTrailRequestContextTestApp(t *testing.T) (*pbtests.TestApp, *core.Record, http.Handler) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.BoolField{Name: "public"})
	openRule := ""
	trails.UpdateRule = &openRule
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	app.OnRecordUpdateRequest("trails").BindFunc(UseTrailRequestContext)
	pbRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := pbRouter.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return app, trail, mux
}

func trailPublishRequest(t *testing.T, trailID string, formData bool) *http.Request {
	t.Helper()
	path := "/api/collections/trails/records/" + trailID
	request := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(`{"public":true}`))
	request.Header.Set("Content-Type", "application/json")
	if formData {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		if err := writer.WriteField("public", "true"); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request = httptest.NewRequest(http.MethodPatch, path, &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
	}
	return request
}
