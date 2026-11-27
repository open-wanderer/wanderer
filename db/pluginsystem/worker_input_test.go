package pluginsystem

import (
	"bytes"
	"context"
	"strings"
	"testing"

	extism "github.com/extism/go-sdk"
)

func TestWorkerInputLimitBytes(t *testing.T) {
	if version := strings.TrimSpace(extism.RuntimeVersion()); version != "v1.7.0" {
		t.Fatalf("revalidate the input allowance for Extism kernel %q", version)
	}
	const mib = uint64(1024 * 1024)
	for _, tt := range []struct {
		name         string
		pages        uint32
		initialBytes uint64
		want         uint64
	}{
		{"minimum admin cap", 2 * 16, mib, mib},
		{"default cap", 256 * 16, mib, 255 * mib},
		{"maximum cap without integer wrap", 4096 * 16, mib, 4095 * mib},
		{"one-page kernel", 2 * 16, wasmPageBytes, 2*mib - wasmPageBytes},
		{"no memory", 2 * 16, 0, 0},
		{"undersized kernel", 2 * 16, wasmPageBytes - 1, 0},
		{"no input budget", 16, mib, 0},
		{"minimum exceeds cap", 16, 2 * mib, 0},
		{"no configured cap", 0, mib, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := workerInputLimitBytes(tt.pages, tt.initialBytes); got != tt.want {
				t.Fatalf("input limit=%d bytes, want %d", got, tt.want)
			}
		})
	}
	if workerInputLimitBytes(256*16, mib) <= defaultWorkerRequestMaxBytes {
		t.Fatal("new input guard unexpectedly narrows the default RPC request limit")
	}
}

func TestHandleCallExportRejectsUnsafeInputBeforeSDK(t *testing.T) {
	// This uninitialized plugin would panic if the SDK call were reached.
	// The tiny artificial budget tests guard placement without exercising an
	// actual refused allocation or an unchecked SDK write.
	worker := &pluginWorkerProcess{
		ctx: context.Background(), instance: &extism.Plugin{},
		wasmPath: "/fixture.wasm", inputMaxBytes: 3,
	}
	msg, err := workerMessageWithData(workerMessageCallExport, workerCallExport{
		WASMPath: "/fixture.wasm", Export: "echo", SessionID: "input-fixture",
		InputBase64: encodeWorkerBytes([]byte("four")),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = worker.handleCallExport(msg)
	if err == nil || !strings.Contains(err.Error(), "safe worker input limit is 3 bytes") ||
		!strings.Contains(err.Error(), "input-fixture export echo") {
		t.Fatalf("expected contextual input safety rejection before SDK access, got %v", err)
	}
}

func TestWorkerRuntimeInputSafetyBoundaries(t *testing.T) {
	t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", "2")
	session := openMemoryTestSession(t, workerCompatibilityTestWASM())
	const inputLimit = 1024 * 1024 // 2 MiB cap minus the kernel's initial 1 MiB.
	input := bytes.Repeat([]byte("x"), inputLimit+1)
	for _, size := range []int{0, wasmPageBytes - 1, wasmPageBytes, wasmPageBytes + 1, inputLimit - 1, inputLimit} {
		output, err := session.Call(context.Background(), "echo", input[:size], RuntimeCallOptions{})
		if err != nil || !bytes.Equal(output, input[:size]) {
			t.Fatalf("ordinary input of %d bytes: output=%d bytes error=%v", size, len(output), err)
		}
	}
	// One byte above the conservative bound still fits the pinned kernel's
	// metadata headroom: even a regressed guard cannot exercise its null-handle
	// corruption path with this test input.
	_, err := session.Call(context.Background(), "echo", input, RuntimeCallOptions{})
	if !IsRuntimeSessionFatalError(err) || !strings.Contains(err.Error(), "safe worker input limit is 1048576 bytes") {
		t.Fatalf("expected generic fatal input safety rejection, got %v", err)
	}
	_, err = session.Call(context.Background(), "echo", nil, RuntimeCallOptions{})
	if !IsRuntimeSessionFatalError(err) || !strings.Contains(err.Error(), "worker session is invalid") {
		t.Fatalf("input-rejected worker was reused: %v", err)
	}
}
