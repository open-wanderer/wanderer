package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

func remoteCapacityJSON(tokens int) string {
	return `{"values":[` + strings.Repeat("0,", tokens-6) + `0]}`
}

func TestRemoteDocumentJSONTokenCounting(t *testing.T) {
	for _, wire := range []string{`{}`, `{"values":[0,true,null,"a\\\"b",{"x":"y"}]}`, "[123, -2.4, 1e3]", `{"a":"` + strings.Repeat("long ", 10000) + `"}`} {
		decoder := json.NewDecoder(strings.NewReader(wire))
		want := 0
		for {
			_, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			want++
		}
		got, err := remoteSyncJSONTokens(t.Context(), strings.NewReader(wire))
		if err != nil || got != want {
			t.Fatalf("tokens=%d want=%d err=%v", got, want, err)
		}
	}
	for _, tokens := range []int{500000, 500001, 1000000, 1000001} {
		var value map[string]any
		err := decodeRemoteSyncJSON(t.Context(), strings.NewReader(remoteCapacityJSON(tokens)), util.FederationSyncBodyLimit, &value)
		if tokens <= 1000000 && err != nil {
			t.Fatalf("%d tokens: %v", tokens, err)
		}
		if tokens > 1000000 && !errors.Is(err, errRemoteSyncJSONComplexity) {
			t.Fatalf("%d tokens: %v", tokens, err)
		}
	}
	padding := bytes.Repeat([]byte("x"), 12<<20)
	var value map[string]any
	if err := decodeRemoteSyncJSON(t.Context(), io.MultiReader(strings.NewReader(`{"text":"`), bytes.NewReader(padding), strings.NewReader(`"}`)), util.FederationSyncBodyLimit, &value); err != nil {
		t.Fatal(err)
	}
	if util.FederationSyncBodyLimit != 32<<20 {
		t.Fatalf("document limit=%d", util.FederationSyncBodyLimit)
	}
}

func TestRemoteDocumentProcessingReservations(t *testing.T) {
	for _, tokens := range []int{500000, 500001} {
		release, err := acquireRemoteSyncProcessing(t.Context(), tokens)
		if err != nil {
			t.Fatal(err)
		}
		want := 1
		if tokens > 500000 {
			want = 2
		}
		if got := len(remoteSyncSlots); got != want {
			release()
			t.Fatalf("reserved=%d want=%d", got, want)
		}
		release()
	}
	release, err := acquireRemoteSyncProcessing(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if unexpected, err := acquireRemoteSyncProcessing(ctx, remoteSyncJSONSharedTokenLimit+1); err == nil {
		unexpected()
		t.Fatal("large document entered alongside existing document")
	}
	if got := len(remoteSyncSlots); got != 1 {
		t.Fatalf("cancelled reservation retained permits: %d", got)
	}
	release()
	release, err = acquireRemoteSyncProcessing(t.Context(), remoteSyncJSONSharedTokenLimit+1)
	if err != nil {
		t.Fatal(err)
	}
	release()
	if got := len(remoteSyncSlots); got != 0 {
		t.Fatalf("remaining permits=%d", got)
	}
	cancelled, stop := context.WithCancel(t.Context())
	stop()
	if release, err := acquireRemoteSyncProcessing(cancelled, 1); err == nil {
		release()
		t.Fatal("cancelled request reserved processing")
	}
	if got := len(remoteSyncSlots); got != 0 {
		t.Fatalf("cancelled request left %d permits", got)
	}
}

func TestRemoteLargeDocumentProcessesAlone(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "document-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(remoteCapacityJSON(remoteSyncJSONSharedTokenLimit + 1)); err != nil {
		t.Fatal(err)
	}
	document := &remoteSyncDocument{file: file}
	defer document.close()
	_, err = document.process(t.Context(), func(data map[string]any) (*core.Record, error) {
		if got := len(remoteSyncSlots); got != remoteSyncConcurrency {
			t.Errorf("large document retained %d permits", got)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()
		if release, err := acquireRemoteSyncProcessing(ctx, 1); err == nil {
			release()
			t.Error("second document entered large document processing")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
