package hooks

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

type assetFileStorageErrorApp struct{ core.App }

func (app assetFileStorageErrorApp) NewFilesystem() (*filesystem.System, error) {
	return nil, errors.New("photo storage unavailable")
}

func TestAssetFileCacheOnlyKeepsSuccessfulNativeResponses(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	collection := core.NewBaseCollection("assets")
	collection.ViewRule = types.Pointer("readable=true")
	collection.Fields.Add(&core.BoolField{Name: "readable"}, &core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1 << 20, Thumbs: []string{"100x100"}})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(raw, "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := core.NewRecord(collection)
	asset.Set("readable", true)
	asset.Set("file", file)
	if err := app.Save(asset); err != nil {
		t.Fatal(err)
	}
	app.OnFileDownloadRequest("assets").BindFunc(AuthorizeAssetFileDownload())
	routerFor := func(application core.App) http.Handler {
		t.Helper()
		router, err := apis.NewRouter(application)
		if err != nil {
			t.Fatal(err)
		}
		router.BindFunc(PrepareAssetFileResponse)
		mux, err := router.BuildMux()
		if err != nil {
			t.Fatal(err)
		}
		return mux
	}
	mux := routerFor(app)
	path := "/api/files/assets/" + asset.Id + "/" + asset.GetString("file")
	request := func(handler http.Handler, method, path, body string, headers map[string]string, want int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("photo response status=%d want=%d: %s", response.Code, want, response.Body)
		}
		cache := "no-store"
		if want == 200 || want == 206 || want == 304 {
			cache = "private, max-age=3600"
		}
		if response.Result().Header.Get("Cache-Control") != cache {
			t.Fatalf("photo response%d Cache-Control=%q want=%q", want, response.Result().Header.Get("Cache-Control"), cache)
		}
		return response
	}
	success := request(mux, http.MethodGet, path, "", nil, 200)
	if !strings.Contains(strings.Join(success.Header().Values("Vary"), ","), "Authorization") {
		t.Fatal("photo Vary lost")
	}
	modified := success.Header().Get("Last-Modified")
	if modified == "" {
		t.Fatal("photo validator missing")
	}
	request(mux, http.MethodGet, path, "", map[string]string{"If-Modified-Since": modified}, 304)
	request(mux, http.MethodGet, path, "", map[string]string{"Range": "bytes=0-7"}, 206)
	request(mux, http.MethodGet, path, "", map[string]string{"Range": "bytes=999999-"}, 416)
	request(mux, http.MethodGet, path, "", map[string]string{"If-Match": `"another-photo"`}, 412)
	asset.Set("readable", false)
	if err := app.Save(asset); err != nil {
		t.Fatal(err)
	}
	request(mux, http.MethodGet, path, "", nil, 404)
	asset.Set("readable", true)
	if err := app.Save(asset); err != nil {
		t.Fatal(err)
	}
	request(mux, http.MethodGet, path, "", nil, 200)
	request(mux, http.MethodGet, "/api/files/assets/nonexistent0000/photo.png", "", nil, 404)
	request(mux, http.MethodGet, "/api/files/assets/"+asset.Id+"/missing.png", "", nil, 404)
	request(mux, http.MethodGet, path, "{", map[string]string{"Content-Type": "application/json"}, 500)
	request(routerFor(assetFileStorageErrorApp{app}), http.MethodGet, path, "", nil, 500)
	failHook := app.OnFileDownloadRequest("assets").BindFunc(func(e *core.FileDownloadRequestEvent) error {
		return e.InternalServerError("Photo serving failed", nil)
	})
	request(mux, http.MethodGet, path, "", nil, 500)
	app.OnFileDownloadRequest("assets").Unbind(failHook)
	streamHook := app.OnFileDownloadRequest("assets").BindFunc(func(e *core.FileDownloadRequestEvent) error {
		if err := http.NewResponseController(e.Response).Flush(); err != nil {
			return err
		}
		_, err := e.Response.Write([]byte("photo stream"))
		return err
	})
	stream := request(mux, http.MethodGet, path, "", nil, 200)
	if !stream.Flushed || stream.Body.String() != "photo stream" {
		t.Fatal("photo response lost controller flushing or streaming writes")
	}
	app.OnFileDownloadRequest("assets").Unbind(streamHook)
	fs, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	if err := fs.Delete(asset.BaseFilesPath() + "/" + asset.GetString("file")); err != nil {
		t.Fatal(err)
	}
	fs.Close()
	request(mux, http.MethodGet, path, "", nil, 404)
	request(mux, http.MethodGet, path+"?thumb=100x100", "", nil, 404)
}
