package pluginsystem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	wasmbinary "github.com/tetratelabs/wabin/binary"
	"github.com/tetratelabs/wabin/leb128"
	"github.com/tetratelabs/wabin/wasm"
)

// Run the real worker entrypoint in the test executable, without starting
// PocketBase, opening a database, or building a separate application binary.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "plugin-worker" && os.Getenv("WANDERER_TEST_PLUGIN_WORKER") == "1" {
		os.Exit(RunPluginWorker(context.Background(), os.Stdin, os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

func TestWorkerWASMMemoryLimitPages(t *testing.T) {
	for _, tt := range []struct {
		raw     string
		wantMiB uint32
	}{
		{"", 256},
		{" \t ", 256},
		{"1", 256},
		{" 2 \n", 2},
		{"+32", 32},
		{"256", 256},
		{"4096", 4096},
		{"0", 256},
		{"-1", 256},
		{"4097", 256},
		{"1.5", 256},
		{"256MiB", 256},
		{"invalid", 256},
		{"4294967296", 256},
		{"99999999999999999999999999999999", 256},
	} {
		t.Run(strconv.Quote(tt.raw), func(t *testing.T) {
			t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", tt.raw)
			if got := workerWASMMemoryLimitPages(); got != tt.wantMiB*16 {
				t.Fatalf("memory limit = %d pages, want %d", got, tt.wantMiB*16)
			}
		})
	}
	t.Run("unset", func(t *testing.T) {
		t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", "")
		_ = os.Unsetenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB")
		if got := workerWASMMemoryLimitPages(); got != 256*16 {
			t.Fatalf("unset memory limit = %d pages", got)
		}
	})
}

func TestWorkerRuntimeWASMMemoryLimit(t *testing.T) {
	for _, memoryMiB := range []int{2, 4} {
		t.Run(strconv.Itoa(memoryMiB)+"MiB", func(t *testing.T) {
			t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", strconv.Itoa(memoryMiB))
			// An omitted or higher declared maximum is capped. An explicitly
			// lower maximum must remain in force, rather than being raised.
			for _, tt := range []struct {
				name       string
				maxPages   uint32
				hasMax     bool
				growRefuse int32
			}{
				{"no_declared_max", 0, false, int32(memoryMiB * 16)},
				{"higher_declared_max", 128, true, int32(memoryMiB * 16)},
				{"lower_declared_max", 8, true, 8},
			} {
				t.Run(tt.name, func(t *testing.T) {
					session := openMemoryTestSession(t, workerMemoryTestWASM(&wasm.Memory{
						Min: 1, Max: tt.maxPages, IsMaxEncoded: tt.hasMax,
					}, tt.growRefuse))
					for _, export := range []string{"grow", "refuse", "refuse"} {
						if _, err := session.Call(context.Background(), export, nil, RuntimeCallOptions{}); err != nil {
							t.Fatalf("%s: %v", export, err)
						}
					}
				})
			}
		})
	}
}

func TestWorkerRuntimeWASMMemoryMinimumFailureIsFatal(t *testing.T) {
	t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", "2")
	session := openMemoryTestSession(t, workerMemoryTestWASM(&wasm.Memory{
		Min: 33, Max: 64, IsMaxEncoded: true,
	}, 32))
	_, err := session.Call(context.Background(), "grow", nil, RuntimeCallOptions{})
	if !IsRuntimeSessionFatalError(err) {
		t.Fatalf("module minimum above cap should be session-fatal, got %v", err)
	}
	if !strings.Contains(err.Error(), "per-memory limit 2 MiB") || !strings.Contains(err.Error(), "min 33 pages") {
		t.Fatalf("missing configured limit or minimum failure: %v", err)
	}
	_, err = session.Call(context.Background(), "grow", nil, RuntimeCallOptions{})
	if !IsRuntimeSessionFatalError(err) || !strings.Contains(err.Error(), "worker session is invalid") {
		t.Fatalf("failed module session was reused: %v", err)
	}
}

func TestWorkerMemoryLimitCoversExtismWithoutChangingDefaults(t *testing.T) {
	// Extism's kernel starts with 1 MiB. Check the final page under a 2 MiB
	// cap, then refuse just one more page, without a large growth request.
	t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", "2")
	path := writeMemoryTestWASM(t, workerMemoryTestWASM(&wasm.Memory{Min: 1}, 16))
	ctx := context.Background()
	worker := &pluginWorkerProcess{ctx: ctx, stderr: io.Discard}
	if err := worker.openPlugin(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = worker.instance.Close(ctx) })
	if worker.instance.MaxVarBytes != 1024*1024 || worker.instance.MaxHttpResponseBytes != 50*1024*1024 {
		t.Fatalf("Extism defaults changed: vars=%d http=%d", worker.instance.MaxVarBytes, worker.instance.MaxHttpResponseBytes)
	}

	memory := worker.instance.Memory()
	initialSize := memory.Size()
	if _, ok := memory.Grow(15); !ok {
		t.Fatal("ordinary growth to one page below the cap failed")
	}
	finalPageOffset := memory.Size()
	if previousPages, ok := memory.Grow(1); !ok || previousPages != 31 || memory.Size() != 2*1024*1024 {
		t.Fatal("the final page within the cap could not grow")
	}
	if !memory.Write(finalPageOffset, []byte("within cap")) {
		t.Fatal("cannot write to the final page")
	}
	before, ok := memory.Read(0, memory.Size())
	if !ok {
		t.Fatal("cannot read memory")
	}
	// Request only one more page: small even if enforcement regresses. A refused
	// grow must not allocate or replace the backing buffer, not merely trap
	// after attempting a large host allocation.
	unexpectedGrowth := false
	allocs := testing.AllocsPerRun(3, func() {
		if !unexpectedGrowth {
			_, unexpectedGrowth = memory.Grow(1)
		}
	})
	if unexpectedGrowth || allocs != 0 {
		t.Fatalf("refused growth: grew=%v, Go allocations=%v", unexpectedGrowth, allocs)
	}
	if memory.Size() != uint32(len(before)) {
		t.Fatal("refused growth changed the memory size")
	}
	after, ok := memory.Read(0, memory.Size())
	if !ok || &before[0] != &after[0] || !bytes.Equal(after[finalPageOffset:finalPageOffset+10], []byte("within cap")) {
		t.Fatal("refused growth changed the backing buffer or its contents")
	}
	t.Logf("Extism memory: initial=%d bytes, grown=%d bytes; refused one-page grow: Go allocations=%v", initialSize, memory.Size(), allocs)
}

func TestWorkerRuntimeCappedMemoryCompatibility(t *testing.T) {
	t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", "2")
	session := openMemoryTestSession(t, workerCompatibilityTestWASM())
	value := []byte("ordinary session variable")
	if _, err := session.Call(context.Background(), "set_var", value, RuntimeCallOptions{}); err != nil {
		t.Fatalf("set variable: %v", err)
	}
	assertVariable := func() {
		t.Helper()
		output, err := session.Call(context.Background(), "get_var", nil, RuntimeCallOptions{})
		if err != nil || !bytes.Equal(output, value) {
			t.Fatalf("variable did not survive across exports: output=%q err=%v", output, err)
		}
	}
	assertVariable()

	_, err := session.Call(context.Background(), "reject", []byte(`{"code":"rate_limited","message":"try later"}`), RuntimeCallOptions{})
	var pluginErr PluginCallError
	if !errors.As(err, &pluginErr) || pluginErr.PluginError.Code != "rate_limited" || IsRuntimeSessionFatalError(err) {
		t.Fatalf("structured guest error should remain nonfatal: %v", err)
	}
	assertVariable()

	// Invalid JSON is rejected locally by the host before any network access.
	output, err := session.Call(context.Background(), "host_error", []byte("not json"), RuntimeCallOptions{MaxHostRequests: 1})
	if err != nil {
		t.Fatalf("host error response: %v", err)
	}
	var response hostHTTPResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Fatalf("decode host error response %q: %v", output, err)
	}
	if response.Error == nil || response.Error.Code != "invalid_request" {
		t.Fatalf("unexpected host error response: %#v", response)
	}
	assertVariable()
}

func openMemoryTestSession(t *testing.T, data []byte) RuntimeSession {
	t.Helper()
	t.Setenv("WANDERER_TEST_PLUGIN_WORKER", "1")
	t.Setenv("WANDERER_PLUGIN_WORKER_EXPORT_TIMEOUT", "10s")
	t.Setenv("WANDERER_PLUGIN_WORKER_SESSION_TIMEOUT", "1m")
	t.Setenv("WANDERER_PLUGIN_WORKER_SLOT_TIMEOUT", "5s")
	t.Setenv("WANDERER_PLUGIN_WORKER_REQUEST_BYTES", "3145728")
	t.Setenv("WANDERER_PLUGIN_WORKER_RESPONSE_BYTES", "3145728")
	t.Setenv("WANDERER_PLUGIN_WORKER_STDERR_BYTES", "65536")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	runtime := WorkerRuntime{Executable: executable}
	session, err := runtime.OpenSession(context.Background(), LocalPlugin{
		Manifest: Manifest{ID: "memory-test"},
		WASMPath: writeMemoryTestWASM(t, data),
	}, RequestPolicyContext{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := session.Close(ctx); err != nil {
			t.Errorf("close worker: %v", err)
		}
	})
	return session
}

func writeMemoryTestWASM(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plugin.wasm")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A valid, tiny module with an ordinary one-page allocation and a bounded
// refusal check. Tests request at most 66 pages (4.125 MiB), even without a cap.
//
// grow: expect memory.grow(1) == 1, store 42 at byte 65536, expect size == 2.
// refuse: expect memory.grow(limitPages) == -1, size == 2, stored byte == 42.
// Each export returns 0 on success, 1 if any assertion fails.
func workerMemoryTestWASM(memory *wasm.Memory, limitPages int32) []byte {
	refuse := append([]byte{wasm.OpcodeI32Const}, leb128.EncodeInt32(limitPages)...)
	refuse = append(refuse,
		wasm.OpcodeMemoryGrow, 0,
		wasm.OpcodeI32Const, 0x7f, wasm.OpcodeI32Ne, // -1
		wasm.OpcodeMemorySize, 0, wasm.OpcodeI32Const, 2, wasm.OpcodeI32Ne, wasm.OpcodeI32Or,
		wasm.OpcodeI32Const, 0x80, 0x80, 0x04, // 65536
		wasm.OpcodeI32Load8U, 0, 0, wasm.OpcodeI32Const, 42, wasm.OpcodeI32Ne, wasm.OpcodeI32Or,
		wasm.OpcodeEnd,
	)
	return wasmbinary.EncodeModule(&wasm.Module{
		TypeSection:     []*wasm.FunctionType{{Results: []wasm.ValueType{wasm.ValueTypeI32}}},
		FunctionSection: []wasm.Index{0, 0},
		MemorySection:   memory,
		ExportSection: []*wasm.Export{
			{Name: "grow", Type: wasm.ExternTypeFunc, Index: 0},
			{Name: "refuse", Type: wasm.ExternTypeFunc, Index: 1},
		},
		CodeSection: []*wasm.Code{
			{Body: []byte{
				wasm.OpcodeI32Const, 1, wasm.OpcodeMemoryGrow, 0,
				wasm.OpcodeI32Const, 1, wasm.OpcodeI32Ne,
				wasm.OpcodeI32Const, 0x80, 0x80, 0x04, // 65536
				wasm.OpcodeI32Const, 42, wasm.OpcodeI32Store8, 0, 0,
				wasm.OpcodeMemorySize, 0, wasm.OpcodeI32Const, 2, wasm.OpcodeI32Ne, wasm.OpcodeI32Or,
				wasm.OpcodeEnd,
			}},
			{Body: refuse},
		},
	})
}

// Only standard Extism PDK operations and wanderer.http_request are imported.
// set_var/get_var use the one-byte key "*"; reject sets the input as a guest
// error; host_error sends the input as a host request and outputs its response.
func workerCompatibilityTestWASM() []byte {
	i64 := wasm.ValueTypeI64
	module := &wasm.Module{
		TypeSection: []*wasm.FunctionType{
			{Results: []wasm.ValueType{i64}},
			{Params: []wasm.ValueType{i64}, Results: []wasm.ValueType{i64}},
			{Params: []wasm.ValueType{i64, wasm.ValueTypeI32}},
			{Params: []wasm.ValueType{i64, i64}},
			{Params: []wasm.ValueType{i64}},
			{Results: []wasm.ValueType{wasm.ValueTypeI32}},
		},
		ImportSection: []*wasm.Import{
			{Module: "extism:host/env", Name: "input_offset", Type: wasm.ExternTypeFunc, DescFunc: 0},
			{Module: "extism:host/env", Name: "alloc", Type: wasm.ExternTypeFunc, DescFunc: 1},
			{Module: "extism:host/env", Name: "store_u8", Type: wasm.ExternTypeFunc, DescFunc: 2},
			{Module: "extism:host/env", Name: "var_set", Type: wasm.ExternTypeFunc, DescFunc: 3},
			{Module: "extism:host/env", Name: "var_get", Type: wasm.ExternTypeFunc, DescFunc: 1},
			{Module: "extism:host/env", Name: "output_set", Type: wasm.ExternTypeFunc, DescFunc: 3},
			{Module: "extism:host/env", Name: "error_set", Type: wasm.ExternTypeFunc, DescFunc: 4},
			{Module: "wanderer", Name: "http_request", Type: wasm.ExternTypeFunc, DescFunc: 1},
			{Module: "extism:host/env", Name: "length", Type: wasm.ExternTypeFunc, DescFunc: 1},
		},
		FunctionSection: []wasm.Index{5, 5, 5, 5, 5},
		ExportSection: []*wasm.Export{
			{Name: "set_var", Type: wasm.ExternTypeFunc, Index: 9},
			{Name: "get_var", Type: wasm.ExternTypeFunc, Index: 10},
			{Name: "reject", Type: wasm.ExternTypeFunc, Index: 11},
			{Name: "host_error", Type: wasm.ExternTypeFunc, Index: 12},
			{Name: "echo", Type: wasm.ExternTypeFunc, Index: 13},
		},
	}
	key := []byte{
		wasm.OpcodeI64Const, 1, wasm.OpcodeCall, 1, wasm.OpcodeLocalTee, 0,
		wasm.OpcodeI32Const, 42, wasm.OpcodeCall, 2, wasm.OpcodeLocalGet, 0,
	}
	output := []byte{
		wasm.OpcodeLocalTee, 0, wasm.OpcodeLocalGet, 0, wasm.OpcodeCall, 8,
		wasm.OpcodeCall, 5, wasm.OpcodeI32Const, 0, wasm.OpcodeEnd,
	}
	module.CodeSection = []*wasm.Code{
		{LocalTypes: []wasm.ValueType{i64}, Body: append(bytes.Clone(key),
			wasm.OpcodeCall, 0, wasm.OpcodeCall, 3, wasm.OpcodeI32Const, 0, wasm.OpcodeEnd)},
		{LocalTypes: []wasm.ValueType{i64}, Body: append(append(bytes.Clone(key), wasm.OpcodeCall, 4), output...)},
		{Body: []byte{wasm.OpcodeCall, 0, wasm.OpcodeCall, 6, wasm.OpcodeI32Const, 1, wasm.OpcodeEnd}},
		{LocalTypes: []wasm.ValueType{i64}, Body: append([]byte{wasm.OpcodeCall, 0, wasm.OpcodeCall, 7}, output...)},
		{LocalTypes: []wasm.ValueType{i64}, Body: append([]byte{wasm.OpcodeCall, 0}, output...)},
	}
	return wasmbinary.EncodeModule(module)
}
