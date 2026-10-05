package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"

	"pocketbase/util"
)

// Exercise PocketBase's actual collection API and field validator. The minimal
// comments schema isolates sanitization from federation and search services.
func TestRichTextAPIRejectsOverlongWritesWithoutSilentTextLoss(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	comments := core.NewBaseCollection("comments")
	comments.CreateRule = types.Pointer("")
	comments.UpdateRule = types.Pointer("")
	comments.Fields.Add(&core.TextField{Name: "text"}, &core.TextField{Name: "name"})
	if err := app.Save(comments); err != nil {
		t.Fatal(err)
	}
	app.OnRecordValidate().BindFunc(util.SanitizeHTML())
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(t *testing.T, method, path string, data map[string]string) *httptest.ResponseRecorder {
		t.Helper()
		body, err := json.Marshal(data)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	assertLengthError := func(t *testing.T, response *httptest.ResponseRecorder) {
		t.Helper()
		if response.Code != http.StatusBadRequest {
			t.Fatalf("overlong write returned %d, want 400: %s", response.Code, response.Body.String())
		}
		var body struct {
			Data map[string]struct {
				Code string `json:"code"`
			} `json:"data"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Data["text"].Code != "validation_max_text_constraint" {
			t.Fatalf("response did not report the rich-text length limit: %s", response.Body.String())
		}
	}
	const path = "/api/collections/comments/records"
	inputs := []struct{ name, value string }{
		{"6000 plain characters", strings.Repeat("x", 6000)},
		{"valid raw length requiring oversized escaping", strings.Repeat("x", 4997) + "&&"},
	}
	for _, input := range inputs {
		t.Run("create/"+input.name, func(t *testing.T) {
			assertLengthError(t, request(t, http.MethodPost, path, map[string]string{"text": input.value}))
			records, err := app.FindAllRecords(comments)
			if err != nil || len(records) != 0 {
				t.Fatalf("rejected create persisted data: records=%d, error=%v", len(records), err)
			}
		})
	}
	quotes := strings.Repeat(`'"`, 2028) + "x"
	response := request(t, http.MethodPost, path, map[string]string{"text": quotes, "name": "unchanged"})
	if response.Code != http.StatusOK {
		t.Fatalf("valid quote-heavy create failed: %d %s", response.Code, response.Body.String())
	}
	var created struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Text != quotes {
		t.Fatal("valid quote-heavy text changed in the create response")
	}
	for _, input := range inputs {
		t.Run("update/"+input.name, func(t *testing.T) {
			assertLengthError(t, request(t, http.MethodPatch, path+"/"+created.ID, map[string]string{"text": input.value, "name": "changed"}))
			stored, err := app.FindRecordById(comments, created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.GetString("text") != quotes || stored.GetString("name") != "unchanged" {
				t.Fatal("rejected update changed the existing comment or another submitted field")
			}
		})
	}
	updatedQuotes := strings.Repeat(`'"`, 2028) + "y"
	response = request(t, http.MethodPatch, path+"/"+created.ID, map[string]string{"text": updatedQuotes})
	if response.Code != http.StatusOK {
		t.Fatalf("valid quote-heavy update failed: %d %s", response.Code, response.Body.String())
	}
	stored, err := app.FindRecordById(comments, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("text") != updatedQuotes {
		t.Fatal("valid quote-heavy text changed in the stored comment")
	}
}
