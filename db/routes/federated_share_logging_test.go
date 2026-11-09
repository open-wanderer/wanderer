package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
)

func TestFederatedShareRequestLogsKeepCredentialsOutOfPersistedRequests(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError, http.StatusBadRequest} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			app, err := pbtests.NewTestApp(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(app.Cleanup)
			app.Settings().Logs.MaxDays = 1
			app.Settings().Logs.MinLevel = 0
			router, err := apis.NewRouter(app)
			if err != nil {
				t.Fatal(err)
			}
			router.Bind(RedactFederatedShareRequestLogs())
			token := "abcdefghijklmnopqrstuvwxyz012345"
			handled := false
			router.GET("/request-log/photo", func(e *core.RequestEvent) error {
				handled = true
				if e.Request.URL.Query().Get("share") != token {
					t.Fatal("handler lost its share credential")
				}
				switch status {
				case http.StatusNotFound:
					return e.NotFoundError("Photo not found", nil)
				case http.StatusInternalServerError:
					failure := e.InternalServerError("Photo fetch failed", fmt.Errorf("file fetch failed for %s", e.Request.URL.String()))
					failure.Data = map[string]any{"operation": "photo-read"}
					return failure
				case http.StatusBadRequest:
					return fmt.Errorf("file fetch failed for %s", e.Request.URL.String())
				default:
					return e.JSON(http.StatusOK, map[string]bool{"ok": true})
				}
			})
			mux, err := router.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, "/request-log/photo?expand=asset&share="+token, nil)
			request.Header.Set("Referer", "https://a.example/trail?share="+token+"&tab=photos")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if request.URL.Query().Get("share") != token {
				t.Fatal("logging changed a request URL already handed to route work")
			}
			if !handled || response.Code != status {
				t.Fatalf("handled=%t status=%d want=%d", handled, response.Code, status)
			}
			if strings.Contains(response.Body.String(), token) {
				t.Fatal("response contains share credential")
			}
			if status == http.StatusInternalServerError {
				var body struct {
					Status  int            `json:"status"`
					Message string         `json:"message"`
					Data    map[string]any `json:"data"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.Status != status || body.Message != "Photo fetch failed." || body.Data["operation"] != "photo-read" {
					t.Fatal("request logging changed the public error response")
				}
			}
			// PocketBase queues request logs asynchronously, then persists its
			// actual batch handler to the auxiliary database.
			handler, ok := app.Logger().Handler().(*logger.BatchHandler)
			if !ok {
				t.Fatalf("unexpected app log handler %T", app.Logger().Handler())
			}
			var logs []*core.Log
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				if err := handler.WriteAll(t.Context()); err != nil {
					t.Fatal(err)
				}
				if err := app.LogQuery().All(&logs); err != nil {
					t.Fatal(err)
				}
				if len(logs) > 0 {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if len(logs) != 1 {
				t.Fatalf("persisted request logs=%d want=1", len(logs))
			}
			data, err := json.Marshal(logs[0])
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), token) {
				t.Fatal("persisted request log contains share credential")
			}
			if logs[0].Data["url"] != "/request-log/photo?expand=asset" || logs[0].Data["referer"] != "https://a.example/trail?tab=photos" {
				t.Fatalf("request log lost ordinary URL context: %s", data)
			}
		})
	}
}
