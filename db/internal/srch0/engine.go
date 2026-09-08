package srch0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// AssertRealMaterialization replays the actual hook HTTP stream to dedicated
// indexes in the selected disposable Meilisearch instance. Waiting here is a
// test barrier, not a change to application startup or mutation semantics.
func (m *Meili) AssertRealMaterialization(t TestingT, c Case) {
	t.Helper()
	endpoint := os.Getenv("SRCH0_MEILI_URL")
	if endpoint == "" {
		t.Log("materialization=http-recorder; real-engine replay runs in the engine matrix")
		return
	}
	key := os.Getenv("SRCH0_MEILI_KEY")
	client := meilisearch.New(endpoint, meilisearch.WithAPIKey(key))
	prefix := fmt.Sprintf("srch0_%d_%s_", os.Getpid(), strings.ToLower(strings.ReplaceAll(c.ID, "-", "_")))
	wait := func(task *meilisearch.TaskInfo) {
		t.Helper()
		done, err := client.WaitForTask(task.TaskUID, 10*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
		if done.Status != "succeeded" {
			t.Fatalf("real engine task failed: %+v", done.Error)
		}
	}
	for _, index := range []string{"trails", "lists", "actors"} {
		task, err := client.CreateIndex(&meilisearch.IndexConfig{Uid: prefix + index, PrimaryKey: "id"})
		if err != nil {
			t.Fatal(err)
		}
		wait(task)
		index := index
		t.Cleanup(func() {
			task, err := client.DeleteIndex(prefix + index)
			if err != nil {
				t.Error(err)
				return
			}
			wait(task)
		})
	}
	// The recorder starts empty and records seeds before mutations. Track which
	// documents should exist at each read, so a guard cannot silently observe a
	// different 200/404 result when replayed against the actual engine.
	present := map[string]map[string]bool{"trails": {}, "lists": {}, "actors": {}}
	for _, call := range m.Calls() {
		parts := strings.Split(strings.Trim(call.Path, "/"), "/")
		if len(parts) < 3 || parts[0] != "indexes" || parts[2] != "documents" {
			continue
		}
		index := parts[1]
		parts[1] = prefix + index
		var body io.Reader
		if call.Method != http.MethodGet {
			encoded, err := json.Marshal(call.Body)
			if err != nil {
				t.Fatal(err)
			}
			body = bytes.NewReader(encoded)
		}
		req, err := http.NewRequest(call.Method, strings.TrimRight(endpoint, "/")+"/"+strings.Join(parts, "/"), body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		response, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if call.Method == http.MethodGet {
			if len(parts) != 4 {
				t.Fatalf("unsupported replay document read: %s", call.Path)
			}
			id := parts[3]
			status := http.StatusNotFound
			if present[index][id] {
				status = http.StatusOK
			}
			if resp.StatusCode != status {
				t.Fatalf("case=%s real engine GET %s status=%d, want %d: %s", c.ID, call.Path, resp.StatusCode, status, response)
			}
			var document Object
			if err := json.Unmarshal(response, &document); err != nil {
				t.Fatal(err)
			}
			if status == http.StatusOK && document["id"] != id {
				t.Fatalf("real engine guard read returned another document: %s", response)
			}
			if status == http.StatusNotFound && document["code"] != "document_not_found" {
				t.Fatalf("real engine guard read returned another error: %s", response)
			}
			continue
		}
		if resp.StatusCode != 202 {
			t.Fatalf("case=%s real engine %s %s status=%d: %s", c.ID, call.Method, call.Path, resp.StatusCode, response)
		}
		var task meilisearch.TaskInfo
		if err := json.Unmarshal(response, &task); err != nil {
			t.Fatal(err)
		}
		wait(&task)
		switch call.Method {
		case http.MethodDelete:
			if len(parts) == 3 {
				present[index] = map[string]bool{}
			} else {
				delete(present[index], parts[3])
			}
		case http.MethodPost, http.MethodPut:
			documents, array := call.Body.([]any)
			if !array {
				documents = []any{call.Body}
			}
			for _, value := range documents {
				document, ok := value.(map[string]any)
				if !ok {
					t.Fatalf("unsupported replay document body: %T", value)
				}
				id, ok := document["id"].(string)
				if !ok || id == "" {
					t.Fatal("replay document has no ID")
				}
				present[index][id] = true
			}
		}
	}
	materialized := Object{}
	for _, index := range []string{"trails", "lists", "actors"} {
		req, _ := http.NewRequest("GET", strings.TrimRight(endpoint, "/")+"/indexes/"+prefix+index+"/documents?limit=1000", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("read materialized %s documents: HTTP %d", index, resp.StatusCode)
		}
		var body struct {
			Results []Object `json:"results"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		docs := Object{}
		for _, doc := range body.Results {
			docs[doc["id"].(string)] = doc
		}
		materialized[index] = docs
	}
	if diff := materializationDiff("$", Readable(materialized), Readable(m.Snapshot())); diff != "" {
		t.Fatalf("case=%s materialization=real-meilisearch: %s", c.ID, diff)
	}
	t.Logf("case=%s materialization=real-meilisearch", c.ID)
}

// A JSON numeric roundtrip through Meilisearch may change a nonintegral float
// by machine precision (20.490000000000002 -> 20.49). Only this engine boundary
// allows that difference; fixture assertions continue to compare exact values.
func materializationDiff(path string, got, want any) string {
	actual, actualNumber := got.(float64)
	expected, expectedNumber := want.(float64)
	if actualNumber && expectedNumber && expected != math.Trunc(expected) &&
		!math.IsNaN(actual) && !math.IsNaN(expected) && !math.IsInf(actual, 0) && !math.IsInf(expected, 0) {
		epsilon := math.Nextafter(1, 2) - 1
		tolerance := epsilon * math.Max(1, math.Max(math.Abs(actual), math.Abs(expected)))
		if math.Abs(actual-expected) <= tolerance {
			return ""
		}
	}
	if actual, ok := got.(map[string]any); ok {
		if expected, ok := want.(map[string]any); ok {
			keys := make([]string, 0, len(actual)+len(expected))
			for key := range actual {
				keys = append(keys, key)
			}
			for key := range expected {
				if _, exists := actual[key]; !exists {
					keys = append(keys, key)
				}
			}
			sort.Strings(keys)
			for _, key := range keys {
				_, actualExists := actual[key]
				_, expectedExists := expected[key]
				if actualExists != expectedExists {
					return fmt.Sprintf("%s.%s: field present=%t; recorded present=%t", path, key, actualExists, expectedExists)
				}
				if diff := materializationDiff(path+"."+key, actual[key], expected[key]); diff != "" {
					return diff
				}
			}
			return ""
		}
	}
	if actual, ok := got.([]any); ok {
		if expected, ok := want.([]any); ok && len(actual) == len(expected) {
			for i := range actual {
				if diff := materializationDiff(fmt.Sprintf("%s[%d]", path, i), actual[i], expected[i]); diff != "" {
					return diff
				}
			}
			return ""
		}
	}
	return firstDiff(path, got, want)
}
