package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type remoteTrailAccessErrorApp struct{ core.App }

func (app *remoteTrailAccessErrorApp) CanAccessRecord(*core.Record, *core.RequestInfo, *string) (bool, error) {
	return false, errors.New("access lookup failed")
}

func TestRemoteTrailShareDenialMarkerRequiresCompletedAccessCheck(t *testing.T) {
	for _, test := range []struct {
		name        string
		share       string
		accessError bool
		status      int
		marked      bool
	}{
		{"valid_share", strings.Repeat("s", 32), false, http.StatusForbidden, true},
		{"no_share", "", false, http.StatusForbidden, false},
		{"malformed_share", "short", false, http.StatusForbidden, false},
		{"access_error_with_share", strings.Repeat("s", 32), true, http.StatusInternalServerError, false},
		{"access_error_without_share", "", true, http.StatusInternalServerError, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newDirectedShareFixture(t)
			var app core.App = f.app
			if test.accessError {
				app = &remoteTrailAccessErrorApp{App: app}
			}
			router, err := apis.NewRouter(app)
			if err != nil {
				t.Fatal(err)
			}
			router.GET("/remote/trail/{id}", RemoteTrailGet)
			mux, err := router.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+f.trail.Id+"?share="+test.share, nil)
			mux.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("trail access response status=%d want=%d", response.Code, test.status)
			}
			var body struct {
				Data map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if test.marked {
				var share struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(body.Data["share"], &share); err != nil || share.Code != "federated_share_denied" {
					t.Fatalf("confirmed share denial marker missing: %s", response.Body.String())
				}
			} else if _, present := body.Data["share"]; present {
				t.Fatalf("unconfirmed access response carries a share denial marker: %s", response.Body.String())
			}
		})
	}
}
