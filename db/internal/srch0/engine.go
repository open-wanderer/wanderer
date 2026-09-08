package srch0

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	for _, call := range m.Calls() {
		parts := strings.Split(strings.Trim(call.Path, "/"), "/")
		if len(parts) < 3 || parts[0] != "indexes" || parts[2] != "documents" {
			continue
		}
		parts[1] = prefix + parts[1]
		body, _ := json.Marshal(call.Body)
		req, err := http.NewRequest(call.Method, strings.TrimRight(endpoint, "/")+"/"+strings.Join(parts, "/"), bytes.NewReader(body))
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
		if resp.StatusCode != 202 {
			t.Fatalf("case=%s real engine %s %s status=%d: %s", c.ID, call.Method, call.Path, resp.StatusCode, response)
		}
		var task meilisearch.TaskInfo
		if err := json.Unmarshal(response, &task); err != nil {
			t.Fatal(err)
		}
		wait(&task)
	}
	materialized := Object{}
	for _, index := range []string{"trails", "lists", "actors"} {
		req, _ := http.NewRequest("GET", strings.TrimRight(endpoint, "/")+"/indexes/"+prefix+index+"/documents?limit=1000", nil)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
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
	Assert(t, c, materialized, m.Snapshot())
	t.Logf("case=%s materialization=real-meilisearch", c.ID)
}
