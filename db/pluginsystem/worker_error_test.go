package pluginsystem

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	extism "github.com/extism/go-sdk"
)

func TestPluginErrorForResult(t *testing.T) {
	callErr := errors.New("exit_code: 1")
	// structuredOfLen builds a valid structured error that is n bytes long.
	structuredOfLen := func(n int) string {
		const head, tail = `{"code":"auth_failed","message":"`, `"}`
		return head + strings.Repeat("x", n-len(head)-len(tail)) + tail
	}
	oversized := structuredOfLen(64*1024 + 34)
	tests := []struct {
		name     string
		code     uint32
		raw      string
		callErr  error
		wantCode string // empty means nil (not a plugin error)
	}{
		{"success", 0, "", nil, ""},
		{"success with stray error text", 0, `{"code":"auth_failed"}`, callErr, ""},
		{"code without call error", 3, "", nil, "plugin_error"},
		{"structured error", 1, `{"code":"auth_failed","message":"nope"}`, callErr, "auth_failed"},
		{"guest sets plugin_error itself", 1, `{"code":"plugin_error","message":"x"}`, callErr, "plugin_error"},
		{"plain text error is fatal", 1, "boom", callErr, ""},
		{"empty error is fatal", 1, "", callErr, ""},
		{"json without code is fatal", 1, `{"message":"boom"}`, callErr, ""},
		{"oversized error is fatal", 1, oversized, callErr, ""},
		{"error of exactly 64 KiB is accepted", 1, structuredOfLen(64 * 1024), callErr, "auth_failed"},
		{"error of 64 KiB plus one byte is fatal", 1, structuredOfLen(64*1024 + 1), callErr, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pluginErrorForResult("list_routes_v1", tt.code, tt.raw, tt.callErr)
			if tt.wantCode == "" {
				if got != nil {
					t.Fatalf("expected no plugin error, got %#v", got)
				}
				return
			}
			if got == nil || got.Code != tt.wantCode {
				t.Fatalf("expected code %q, got %#v", tt.wantCode, got)
			}
		})
	}
}

// failingWasm is a hand-assembled module that sets the plugin error to the
// bytes testenv.msg returns and exits with code 1, as a TinyGo plugin does
// after pdk.SetError. It lets the call site run without a TinyGo build.
//
//	(import "extism:host/env" "error_set" (func (param i64)))
//	(import "testenv" "msg" (func (result i64)))
//	(func (export "run") (result i32) call 1 call 0 i32.const 1)
var failingWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00,
	0x01, 0x0d, 0x03, 0x60, 0x01, 0x7e, 0x00, 0x60, 0x00, 0x01, 0x7e, 0x60, 0x00, 0x01, 0x7f,
	0x02, 0x2b, 0x02,
	0x0f, 'e', 'x', 't', 'i', 's', 'm', ':', 'h', 'o', 's', 't', '/', 'e', 'n', 'v',
	0x09, 'e', 'r', 'r', 'o', 'r', '_', 's', 'e', 't', 0x00, 0x00,
	0x07, 't', 'e', 's', 't', 'e', 'n', 'v', 0x03, 'm', 's', 'g', 0x00, 0x01,
	0x03, 0x02, 0x01, 0x02,
	0x07, 0x07, 0x01, 0x03, 'r', 'u', 'n', 0x00, 0x02,
	0x0a, 0x0a, 0x01, 0x08, 0x00, 0x10, 0x01, 0x10, 0x00, 0x41, 0x01, 0x0b,
}

func callFailingPlugin(t *testing.T, guestError string) (*workerCallResult, error) {
	t.Helper()
	msgFn := extism.NewHostFunctionWithStack("msg", func(_ context.Context, p *extism.CurrentPlugin, stack []uint64) {
		offset, err := p.WriteBytes([]byte(guestError))
		if err != nil {
			t.Error(err)
		}
		stack[0] = offset
	}, nil, []extism.ValueType{extism.ValueTypeI64})
	msgFn.SetNamespace("testenv")

	ctx := context.Background()
	instance, err := extism.NewPlugin(ctx, extism.Manifest{
		Wasm: []extism.Wasm{extism.WasmData{Data: failingWasm}},
	}, extism.PluginConfig{EnableWasi: true}, []extism.HostFunction{msgFn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = instance.Close(ctx) })

	var stdout bytes.Buffer
	worker := &pluginWorkerProcess{
		ctx: ctx, stdout: &stdout, responseMaxBytes: 1 << 20,
		instance: instance, wasmPath: "/plugins/a.wasm",
	}
	msg, err := workerMessageWithData(workerMessageCallExport, workerCallExport{
		WASMPath: "/plugins/a.wasm", Export: "run", SessionID: "sess-err",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.handleCallExport(msg); err != nil {
		return nil, err
	}
	reply, err := readWorkerMessage(&stdout, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	result, err := workerData[workerCallResult](reply)
	if err != nil {
		t.Fatal(err)
	}
	return &result, nil
}

func TestHandleCallExportGuestError(t *testing.T) {
	for _, code := range []string{"auth_failed", "rate_limited", "plugin_error"} {
		t.Run("structured "+code, func(t *testing.T) {
			result, err := callFailingPlugin(t, `{"code":"`+code+`","message":"nope"}`)
			if err != nil {
				t.Fatalf("expected a plugin error, got fatal: %v", err)
			}
			if result.PluginError == nil || result.PluginError.Code != code {
				t.Fatalf("expected plugin error %q, got %#v", code, result)
			}
		})
	}
	t.Run("plain text is fatal", func(t *testing.T) {
		_, err := callFailingPlugin(t, "boom")
		if err == nil || !strings.Contains(err.Error(), "call run:") {
			t.Fatalf("expected fatal call error, got %v", err)
		}
	})
	t.Run("oversized is fatal", func(t *testing.T) {
		raw := `{"code":"auth_failed","message":"` + strings.Repeat("x", 64*1024) + `"}`
		_, err := callFailingPlugin(t, raw)
		if err == nil || !strings.Contains(err.Error(), "call run:") {
			t.Fatalf("expected fatal call error, got %v", err)
		}
	})
}
