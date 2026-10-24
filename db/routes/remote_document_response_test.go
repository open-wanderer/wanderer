package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/router"
	"pocketbase/util"
)

func TestRemoteDocumentLimitResponseIncludesOriginalView(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, boundary := range []string{"bytes", "tokens"} {
			t.Run(collection+"/"+boundary, func(t *testing.T) {
				f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
				actor := f.remoteActor(t, "alice", "remote.example")
				copy := f.object(t, collection, actor.Id, "https://remote.example", "remoteid0000001", true, false)
				copy.Set("full_sync_completed", false)
				if err := f.app.Save(copy); err != nil {
					t.Fatal(err)
				}
				f.transport.body = `{"id":"remoteid0000001","name":"Unconfirmed"}` + strings.Repeat(" ", int(util.FederationSyncBodyLimit))
				if boundary == "tokens" {
					f.transport.body = remoteCapacityJSON(remoteSyncJSONTokenLimit + 1)
				}
				router, err := apis.NewRouter(f.app)
				if err != nil {
					t.Fatal(err)
				}
				kind, viewPath, handler := "trail", "/trail/view/@alice/remoteid0000001", RemoteTrailGet
				if collection == "lists" {
					kind, viewPath, handler = "list", "/lists/@alice/remoteid0000001", RemoteListGet
				}
				router.GET("/remote/"+kind+"/{id}", handler)
				mux, err := router.BuildMux()
				if err != nil {
					t.Fatal(err)
				}
				request := httptest.NewRequest(http.MethodGet, "/remote/"+kind+"/"+copy.Id+"?handle="+url.QueryEscape("@alice@remote.example"), nil)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				var body struct {
					Message string `json:"message"`
					Data    struct {
						Code      string `json:"code"`
						OriginURL string `json:"origin_url"`
					} `json:"data"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if response.Code != http.StatusUnprocessableEntity || body.Data.Code != "remote_document_limit" || body.Data.OriginURL != "https://remote.example"+viewPath || !strings.Contains(body.Message, "too large") {
					t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
				}
				stored, err := f.app.FindRecordById(collection, copy.Id)
				if err != nil || stored.GetBool("full_sync_completed") || stored.GetString("name") != "object" {
					t.Fatalf("rejected document rewrote copy: %v %v", stored, err)
				}
			})
		}
	}
}

func TestRemoteDocumentOriginalViewKeepsOriginAndEscapesPath(t *testing.T) {
	for _, origin := range []string{"https://www.remote.example:8443", "http://remote.example:8080"} {
		f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
		actor := f.remoteActor(t, "alice", "remote.example")
		for _, collection := range []string{"trails", "lists"} {
			copy := f.object(t, collection, actor.Id, origin, "remoteid0000001", true, false)
			err := remoteDocumentLimitError(f.app, copy, collection)
			apiError := err.(*router.ApiError)
			raw, ok := apiError.Data["origin_url"].(string)
			parsed, parseErr := url.Parse(raw)
			if !ok || parseErr != nil || parsed.Scheme+"://"+parsed.Host != origin || parsed.RawQuery != "" || parsed.Fragment != "" {
				t.Fatalf("original view=%q error=%v", raw, parseErr)
			}
			for _, iri := range []string{"javascript:alert(1)", "https://user:pass@remote.example/api/v1/trail/id", "//remote.example/api/v1/trail/id"} {
				copy.Set("iri", iri)
				if apiError := remoteDocumentLimitError(f.app, copy, collection).(*router.ApiError); apiError.Data["origin_url"] != nil {
					t.Fatalf("unexpected original view for %q", iri)
				}
			}
		}
	}
}

func TestRemoteDocumentTokenLimitKeepsCompletedCopiesReadable(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		t.Run(collection, func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
			actor := f.remoteActor(t, "alice", "remote.example")
			copy := f.object(t, collection, actor.Id, "https://remote.example", "remoteid0000001", true, false)
			f.transport.body = remoteCapacityJSON(remoteSyncJSONTokenLimit + 1)
			status, body := f.get(t, collection, copy.Id, "@alice@remote.example")
			if status != http.StatusOK || !strings.Contains(body, `"name":"object"`) {
				t.Fatalf("completed copy status=%d body=%s", status, body)
			}
			stored, err := f.app.FindRecordById(collection, copy.Id)
			if err != nil || !stored.GetBool("full_sync_completed") || stored.GetString("name") != "object" {
				t.Fatalf("stored copy=%v error=%v", stored, err)
			}
		})
	}
}
