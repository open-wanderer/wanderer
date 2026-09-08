package srch0

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/meilisearch/meilisearch-go"
)

type Request struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Body   any    `json:"body"`
}

// Meili captures HTTP payloads from the real SDK and applies document replace,
// merge and delete semantics for hook observations. Engine qualification is a
// separate harness; this adapter deliberately does not emulate search/ranking.
type Meili struct {
	mu         sync.Mutex
	Server     *httptest.Server
	Client     meilisearch.ServiceManager
	Requests   []Request
	Documents  map[string]map[string]Object
	Missing    bool
	FailDelete bool
	Before     func(Request)
}

func NewMeili(t TestingT) *Meili {
	t.Helper()
	m := &Meili{Documents: map[string]map[string]Object{"trails": {}, "lists": {}, "actors": {}}}
	m.Server = httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(m.Server.Close)
	m.Client = meilisearch.New(m.Server.URL)
	return m
}

func (m *Meili) serve(w http.ResponseWriter, r *http.Request) {
	var body any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	req := Request{r.Method, r.URL.Path, body}
	if m.Before != nil {
		m.Before(req)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Requests = append(m.Requests, req)
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/keys" {
		fmt.Fprint(w, `{"results":[{"name":"Default Search API Key","uid":"00000000-0000-4000-8000-000000000001","key":"synthetic-srch0-search-key","actions":["search"],"indexes":["*"]}],"offset":0,"limit":20,"total":1}`)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/tasks/") {
		fmt.Fprint(w, `{"uid":1,"status":"succeeded","type":"indexCreation","enqueuedAt":"2026-09-07T00:00:00Z"}`)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 2 && parts[0] == "indexes" && r.Method == "GET" {
		if m.Missing {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"message":"missing","code":"index_not_found","type":"invalid_request","link":"https://docs.meilisearch.com/errors#index_not_found"}`)
		} else {
			fmt.Fprintf(w, `{"uid":%q,"primaryKey":"id","createdAt":"2026-09-07T00:00:00Z","updatedAt":"2026-09-07T00:00:00Z"}`, parts[1])
		}
		return
	}
	if len(parts) >= 3 && parts[0] == "indexes" && parts[2] == "documents" {
		index := parts[1]
		if m.Documents[index] == nil {
			m.Documents[index] = map[string]Object{}
		}
		if r.Method == "GET" && len(parts) == 4 {
			document, exists := m.Documents[index][parts[3]]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"message":"missing document","code":"document_not_found","type":"invalid_request"}`)
			} else {
				_ = json.NewEncoder(w).Encode(document)
			}
			return
		}
		if r.Method == "DELETE" {
			if m.FailDelete {
				w.WriteHeader(400)
				fmt.Fprint(w, `{"message":"synthetic delete rejection","code":"invalid_document_id","type":"invalid_request"}`)
				return
			}
			if len(parts) == 3 {
				m.Documents[index] = map[string]Object{}
			} else {
				delete(m.Documents[index], parts[3])
			}
		} else if r.Method == "POST" || r.Method == "PUT" {
			docs, ok := body.([]any)
			if !ok {
				docs = []any{body}
			}
			for _, value := range docs {
				doc, ok := value.(map[string]any)
				if !ok {
					continue
				}
				id, _ := doc["id"].(string)
				if r.Method == "POST" || m.Documents[index][id] == nil {
					m.Documents[index][id] = Object{}
				}
				for k, v := range doc {
					m.Documents[index][id][k] = v
				}
			}
		}
	}
	w.WriteHeader(202)
	fmt.Fprint(w, `{"taskUid":1,"indexUid":"trails","status":"enqueued","type":"documentAdditionOrUpdate","enqueuedAt":"2026-09-07T00:00:00Z"}`)
}

func (m *Meili) Snapshot() Object {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, _ := json.Marshal(m.Documents)
	var result Object
	json.Unmarshal(b, &result)
	return result
}

func (m *Meili) Calls() []Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Request{}, m.Requests...)
}

func (m *Meili) ClearCalls() { m.mu.Lock(); defer m.mu.Unlock(); m.Requests = nil }
