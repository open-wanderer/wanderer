package federation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	pub "github.com/go-ap/activitypub"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// collectionServer serves a followers collection of seven actors, paged by
// size in the style of one server implementation.
func collectionServer(t *testing.T, style string, size int) string {
	t.Helper()

	actors := make([]string, 7)
	for i := range actors {
		actors[i] = fmt.Sprintf("https://remote.example/users/u%d", i)
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/alice/followers" {
			http.NotFound(w, r)
			return
		}
		coll := srv.URL + r.URL.Path
		q := r.URL.Query()

		// start is the index of the page's first item, or -1 for the root.
		start := -1
		switch style {
		case "mastodon", "wanderer", "wanderer-old":
			if n, err := strconv.Atoi(q.Get("page")); err == nil {
				start = (n - 1) * size
			} else if style == "wanderer-old" {
				start = 0
			}
		case "gotosocial":
			if q.Has("max_id") {
				start, _ = strconv.Atoi(q.Get("max_id"))
			} else if q.Has("limit") {
				start = 0
			}
		}

		var body map[string]any
		if start < 0 {
			first := coll + "?page=1"
			if style == "gotosocial" {
				first = coll + "?limit=" + strconv.Itoa(size)
			}
			body = map[string]any{"type": "OrderedCollection", "id": coll, "totalItems": len(actors), "first": first}
		} else {
			end := min(start+size, len(actors))
			body = map[string]any{"type": "OrderedCollectionPage", "totalItems": len(actors), "orderedItems": actors[start:end]}
			if end < len(actors) {
				switch style {
				case "gotosocial":
					body["next"] = fmt.Sprintf("%s?limit=%d&max_id=%d", coll, size, end)
				case "wanderer-old":
					// Older wanderer versions pointed followers' next at the outbox.
					body["next"] = fmt.Sprintf("%s/users/alice/outbox?page=%d", srv.URL, end/size+1)
				default:
					body["next"] = fmt.Sprintf("%s?page=%d", coll, end/size+1)
				}
			}
		}
		w.Header().Set("Content-Type", "application/activity+json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)

	return srv.URL + "/users/alice/followers"
}

func TestFetchCollectionPage(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	defaultClient := newHTTPClient
	newHTTPClient = func() *http.Client { return http.DefaultClient }
	t.Cleanup(func() { newHTTPClient = defaultClient })

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	want := func(from, to int) []string {
		var out []string
		for i := from; i < to; i++ {
			out = append(out, fmt.Sprintf("https://remote.example/users/u%d", i))
		}
		return out
	}

	for _, style := range []string{"mastodon", "gotosocial", "wanderer", "wanderer-old"} {
		t.Run(style, func(t *testing.T) {
			url := collectionServer(t, style, 3)
			for page, items := range map[int][]string{1: want(0, 3), 2: want(3, 6), 3: want(6, 7), 4: nil} {
				got, err := FetchCollectionPage(app, context.Background(), url, page)
				if err != nil {
					t.Fatalf("page %d: %v", page, err)
				}
				var ids []string
				for _, it := range got.OrderedItems {
					ids = append(ids, it.GetLink().String())
				}
				if !reflect.DeepEqual(ids, items) {
					t.Errorf("page %d = %v, want %v", page, ids, items)
				}
				if got.TotalItems != 7 {
					t.Errorf("page %d: totalItems = %d, want 7", page, got.TotalItems)
				}
			}
		})
	}
}

func TestFetchCollectionPageStaysOnHost(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	defaultClient := newHTTPClient
	newHTTPClient = func() *http.Client { return http.DefaultClient }
	t.Cleanup(func() { newHTTPClient = defaultClient })

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "OrderedCollectionPage", "orderedItems": []string{"https://remote.example/users/u0"},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "OrderedCollection", "totalItems": 1, "first": "https://elsewhere.example/followers?page=1",
		})
	}))
	t.Cleanup(srv.Close)

	got, err := FetchCollectionPage(app, context.Background(), srv.URL+"/followers", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.OrderedItems) != 1 {
		t.Fatalf("items = %v, want the page from the collection's own host", got.OrderedItems)
	}
}

// countingTransport counts the requests sent through it.
type countingTransport struct{ n atomic.Int32 }

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return http.DefaultTransport.RoundTrip(r)
}

func countRequests(t *testing.T) *countingTransport {
	t.Helper()
	counter := &countingTransport{}
	defaultClient := newHTTPClient
	newHTTPClient = func() *http.Client { return &http.Client{Transport: counter} }
	t.Cleanup(func() { newHTTPClient = defaultClient })
	return counter
}

func itemIDs(page *pub.OrderedCollectionPage) []string {
	var ids []string
	for _, it := range page.OrderedItems {
		ids = append(ids, it.GetLink().String())
	}
	return ids
}

func TestFetchCollectionCursor(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	counter := countRequests(t)

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	ctx := context.Background()

	for _, style := range []string{"mastodon", "gotosocial", "wanderer", "wanderer-old"} {
		t.Run(style, func(t *testing.T) {
			url := collectionServer(t, style, 3)

			page, err := FetchCollectionPage(app, ctx, url, 1)
			if err != nil {
				t.Fatal(err)
			}
			next := CollectionNext(url, page, 1)
			if next == "" {
				t.Fatal("page 1 has no next")
			}

			for n := 2; n <= 3; n++ {
				if style == "wanderer-old" && strings.Contains(next, "/outbox") {
					t.Fatalf("next %q was not resolved onto the collection", next)
				}
				counter.n.Store(0)
				got, err := FetchCollectionCursor(app, ctx, url, next)
				if err != nil {
					t.Fatalf("page %d: %v", n, err)
				}
				if c := counter.n.Load(); c != 1 {
					t.Errorf("page %d made %d requests, want 1", n, c)
				}
				want, err := FetchCollectionPage(app, ctx, url, n)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(itemIDs(got), itemIDs(want)) {
					t.Errorf("page %d = %v, want %v", n, itemIDs(got), itemIDs(want))
				}
				if got.TotalItems != 7 {
					t.Errorf("page %d: totalItems = %d, want 7", n, got.TotalItems)
				}
				next = CollectionNext(url, got, n)
			}
			if next != "" {
				t.Errorf("next after the last page = %q, want empty", next)
			}
		})
	}
}

func TestFetchCollectionCursorRejects(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	counter := countRequests(t)

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	collURL := collectionServer(t, "mastodon", 3)
	srvURL := strings.TrimSuffix(collURL, "/users/alice/followers")
	host := strings.TrimPrefix(srvURL, "http://")
	hostname, _, _ := strings.Cut(host, ":")

	cursors := map[string]string{
		"other host":   "http://elsewhere.example/users/alice/followers?page=2",
		"other path":   srvURL + "/users/alice/outbox?page=2",
		"other scheme": strings.Replace(collURL, "http://", "https://", 1) + "?page=2",
		"relative":     "/users/alice/followers?page=2",
		"other port":   "http://" + hostname + ":1/users/alice/followers?page=2",
		"userinfo":     "http://evil@" + host + "/users/alice/followers?page=2",
		"escaped path": srvURL + "/users/alice%2Ffollowers?page=2",
		"longer path":  collURL + "/../../admin?page=2",
		"path suffix":  collURL + "-evil?page=2",
		"fragment":     collURL + "?page=2#x",
		"newline":      collURL + "?page=2\nHost: elsewhere.example",
	}
	for name, cursor := range cursors {
		t.Run(name, func(t *testing.T) {
			counter.n.Store(0)
			_, err := FetchCollectionCursor(app, context.Background(), collURL, cursor)
			if !errors.Is(err, ErrInvalidCursor) {
				t.Errorf("err = %v, want ErrInvalidCursor", err)
			}
			if c := counter.n.Load(); c != 0 {
				t.Errorf("made %d requests, want 0", c)
			}
		})
	}
}

func TestCollectionNextEmpty(t *testing.T) {
	url := "http://example.test/users/alice/followers"
	for name, page := range map[string]*pub.OrderedCollectionPage{
		"nil page": nil,
		"nil next": {},
		"empty":    {Next: pub.IRI("")},
	} {
		if got := CollectionNext(url, page, 1); got != "" {
			t.Errorf("%s: next = %q, want empty", name, got)
		}
	}
}

func TestCollectionNextPassesValidation(t *testing.T) {
	collURL := "https://remote.example/users/alice/followers"
	for _, next := range []string{
		"?page=2",
		"/users/alice/followers?max_id=9",
		"https://remote.example/users/alice/followers?page=2",
		"http://remote.example/users/alice/followers?page=2",
		"https://u@remote.example/users/alice/followers?page=2",
		"https://remote.example/users/alice/followers?page=2#x",
		"https://remote.example/users/%61lice/followers?page=2",
		"https://remote.example/users/alice/outbox?page=2",
		"https://elsewhere.example/users/alice/followers?page=2",
	} {
		t.Run(next, func(t *testing.T) {
			cursor := CollectionNext(collURL, &pub.OrderedCollectionPage{Next: pub.IRI(next)}, 1)
			if cursor == "" {
				t.Fatal("next = \"\", want a cursor")
			}
			if err := validateCursor(collURL, cursor); err != nil {
				t.Errorf("cursor %q: %v", cursor, err)
			}
		})
	}
}
