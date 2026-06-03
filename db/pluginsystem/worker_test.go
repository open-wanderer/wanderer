package pluginsystem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestWorkerRPCFrameRoundTrip(t *testing.T) {
	msg, err := workerMessageWithData(workerMessageCallExport, workerCallExport{
		WASMPath:    "/tmp/plugin.wasm",
		Export:      "list_routes_v1",
		InputBase64: encodeWorkerBytes([]byte(`{"ok":true}`)),
	})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := writeWorkerMessage(&buf, 1024, msg); err != nil {
		t.Fatalf("write message: %v", err)
	}
	got, err := readWorkerMessage(&buf, 1024)
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if got.Type != workerMessageCallExport {
		t.Fatalf("unexpected type: %q", got.Type)
	}
	payload, err := workerData[workerCallExport](got)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	input, err := decodeWorkerBytes(payload.InputBase64)
	if err != nil {
		t.Fatalf("decode input: %v", err)
	}
	if string(input) != `{"ok":true}` {
		t.Fatalf("unexpected input: %s", input)
	}
}

func TestWorkerRPCRejectsOversizedFrameBeforePayloadRead(t *testing.T) {
	msg, err := workerMessageWithData(workerMessageError, workerError{Message: "too large"})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := writeWorkerMessage(&buf, 1024, msg); err != nil {
		t.Fatalf("write message: %v", err)
	}

	if _, err := readWorkerMessage(&buf, 4); err == nil {
		t.Fatal("expected oversized frame error")
	}
}

func TestPluginWorkerExitsOnStdinEOF(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunPluginWorker(context.Background(), bytes.NewReader(nil), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("unexpected exit code %d, stderr %q", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestPluginWorkerHostRPCFatalSetsClearError(t *testing.T) {
	worker := &pluginWorkerProcess{}
	stack := []uint64{123}

	worker.failHostRPC(stack, errors.New("host RPC read failed"))

	if stack[0] != 0 {
		t.Fatalf("expected null response pointer, got %d", stack[0])
	}
	if worker.fatalErr == nil || worker.fatalErr.Error() != "host RPC read failed" {
		t.Fatalf("unexpected fatal error: %v", worker.fatalErr)
	}
}

func TestRuntimeSessionFatalErrorIsDetectableThroughWrapping(t *testing.T) {
	err := fmt.Errorf("outer: %w", RuntimeSessionFatalError{Err: errors.New("worker died")})
	if !IsRuntimeSessionFatalError(err) {
		t.Fatal("expected fatal session error")
	}
	if IsRuntimeSessionFatalError(errors.New("plugin error")) {
		t.Fatal("unexpected fatal session error")
	}
}

func TestChildEnvWithoutStripsKeys(t *testing.T) {
	t.Setenv("EXTISM_ENABLE_WASI_OUTPUT", "1")
	t.Setenv("WANDERER_TEST_KEEP", "yes")

	env := childEnvWithout("EXTISM_ENABLE_WASI_OUTPUT")
	for _, entry := range env {
		if entry == "EXTISM_ENABLE_WASI_OUTPUT=1" {
			t.Fatalf("unexpected stripped env entry in %#v", env)
		}
	}
	if os.Getenv("EXTISM_ENABLE_WASI_OUTPUT") != "1" {
		t.Fatal("childEnvWithout should not mutate the current process env")
	}
}

func TestInjectHostRequestAuthUsesExistingSessionForRefresh(t *testing.T) {
	spec := HostRequestSpec{Auth: "session"}
	session := &fakeRuntimeSession{
		output: []byte(`{"token":"session-token"}`),
	}
	err := InjectHostRequestAuth(context.Background(), AuthInjectionInput{
		Session: session,
		Plugin: LocalPlugin{Manifest: Manifest{
			Auth: AuthManifest{Contexts: map[string]AuthContext{
				"session": {
					Type:         AuthTypeSession,
					SecretFields: []string{"email", "password"},
					Refresh:      &AuthRefresh{Mode: AuthRefreshModePlugin, Function: "refresh_session_v1"},
				},
			}},
		}},
		Instance: testPluginInstance("inst1", "plugin.test"),
		Auth:     map[string]any{"email": "user@example.com", "password": "secret"},
		Spec:     &spec,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.export != "refresh_session_v1" {
		t.Fatalf("unexpected export: %q", session.export)
	}
	if got := spec.Headers[AuthHeaderAuthorization]; got != AuthSchemeBearer+" session-token" {
		t.Fatalf("unexpected auth header: %q", got)
	}

	var input map[string]any
	if err := json.Unmarshal(session.input, &input); err != nil {
		t.Fatalf("invalid refresh input: %v", err)
	}
	auth, ok := input["auth"].(map[string]any)
	if !ok {
		t.Fatalf("missing refresh auth: %#v", input)
	}
	if _, ok := auth["accessToken"]; ok {
		t.Fatalf("refresh auth leaked access token: %#v", auth)
	}
}

type fakeRuntimeSession struct {
	export string
	input  []byte
	output []byte
	err    error
}

func (s *fakeRuntimeSession) Call(_ context.Context, export string, input []byte) ([]byte, error) {
	s.export = export
	s.input = append([]byte(nil), input...)
	if s.err != nil {
		return nil, s.err
	}
	return s.output, nil
}

func (s *fakeRuntimeSession) Close(context.Context) error {
	return nil
}

func TestInjectHostRequestAuthDoesNotRequireRuntimeWhenSessionProvided(t *testing.T) {
	spec := HostRequestSpec{Auth: "session"}
	session := &fakeRuntimeSession{err: errors.New("session failed")}
	err := InjectHostRequestAuth(context.Background(), AuthInjectionInput{
		Session: session,
		Plugin: LocalPlugin{Manifest: Manifest{
			Auth: AuthManifest{Contexts: map[string]AuthContext{
				"session": {
					Type:         AuthTypeSession,
					SecretFields: []string{"email"},
					Refresh:      &AuthRefresh{Mode: AuthRefreshModePlugin, Function: "refresh_session_v1"},
				},
			}},
		}},
		Instance: testPluginInstance("inst1", "plugin.test"),
		Auth:     map[string]any{"email": "user@example.com"},
		Spec:     &spec,
	})
	if err == nil || err.Error() != "session failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func testPluginInstance(id string, pluginID string) *core.Record {
	collection := core.NewBaseCollection("plugin_instances")
	collection.Fields.Add(&core.TextField{Name: "plugin_id"})
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("plugin_id", pluginID)
	return record
}
