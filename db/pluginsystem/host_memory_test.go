package pluginsystem

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tetratelabs/wazero/api"
)

type recordingHostMemory struct {
	api.Memory
	writes  int
	offset  uint32
	data    []byte
	writeOK bool
}

func (m *recordingHostMemory) Write(offset uint32, data []byte) bool {
	m.writes++
	m.offset = offset
	m.data = bytes.Clone(data)
	return m.writeOK
}

type fakeExtismHostMemory struct {
	memory      *recordingHostMemory
	handle      uint64
	allocErr    error
	freeErr     error
	allocations []uint64
	freed       []uint64
	memoryReads int
}

func (p *fakeExtismHostMemory) AllocWithContext(_ context.Context, n uint64) (uint64, error) {
	p.allocations = append(p.allocations, n)
	return p.handle, p.allocErr
}

func (p *fakeExtismHostMemory) FreeWithContext(_ context.Context, offset uint64) error {
	p.freed = append(p.freed, offset)
	return p.freeErr
}

func (p *fakeExtismHostMemory) Memory() api.Memory {
	p.memoryReads++
	if p.memory == nil {
		return nil
	}
	return p.memory
}

func TestWriteHostMemory(t *testing.T) {
	allocErr := errors.New("allocator call failed")
	freeErr := errors.New("free call failed")
	for _, tt := range []struct {
		name        string
		handle      uint64
		data        []byte
		writeOK     bool
		noMemory    bool
		allocErr    error
		freeErr     error
		wantErr     string
		wantCause   error
		wantAllocs  int
		wantWrites  int
		wantCleanup bool
	}{
		{name: "empty payload", handle: 128, wantErr: "payload size"},
		{name: "null handle", data: []byte("response"), wantErr: "allocation handle", wantAllocs: 1},
		{name: "unrepresentable handle", handle: 1 << 32, data: []byte("response"), wantErr: "allocation handle", wantAllocs: 1},
		{name: "allocation error", data: []byte("response"), allocErr: allocErr, wantErr: "allocate host response", wantCause: allocErr, wantAllocs: 1},
		{name: "address range overflow", handle: 1<<32 - 1, data: []byte("response"), wantErr: "address range", wantAllocs: 1, wantCleanup: true},
		{name: "failed write", handle: 128, data: []byte("response"), wantErr: "write host response memory", wantAllocs: 1, wantWrites: 1, wantCleanup: true},
		{name: "missing memory", handle: 128, data: []byte("response"), noMemory: true, wantErr: "write host response memory", wantAllocs: 1, wantCleanup: true},
		{name: "failed cleanup", handle: 128, data: []byte("response"), freeErr: freeErr, wantErr: "write host response memory", wantCause: freeErr, wantAllocs: 1, wantWrites: 1, wantCleanup: true},
		{name: "success", handle: 128, data: []byte("response"), writeOK: true, wantAllocs: 1, wantWrites: 1},
		{name: "last representable byte", handle: 1<<32 - 1, data: []byte("x"), writeOK: true, wantAllocs: 1, wantWrites: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			memory := &recordingHostMemory{writeOK: tt.writeOK}
			plugin := &fakeExtismHostMemory{
				memory: memory, handle: tt.handle, allocErr: tt.allocErr, freeErr: tt.freeErr,
			}
			if tt.noMemory {
				plugin.memory = nil
			}
			offset, err := writeHostMemory(context.Background(), plugin, tt.data)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || offset != 0 {
					t.Fatalf("offset=%d error=%v, want %q", offset, err, tt.wantErr)
				}
			} else if err != nil || offset != tt.handle {
				t.Fatalf("offset=%d error=%v", offset, err)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Fatalf("lost underlying error: %v", err)
			}
			if len(plugin.allocations) != tt.wantAllocs || memory.writes != tt.wantWrites {
				t.Fatalf("allocations=%v writes=%d", plugin.allocations, memory.writes)
			}
			if len(plugin.allocations) != 0 && plugin.allocations[0] != uint64(len(tt.data)) {
				t.Fatalf("unexpected allocation size: %v", plugin.allocations)
			}
			if tt.wantCleanup {
				if len(plugin.freed) != 1 || plugin.freed[0] != tt.handle {
					t.Fatalf("allocation not cleaned up: %v", plugin.freed)
				}
			} else if len(plugin.freed) != 0 {
				t.Fatalf("unexpected free: %v", plugin.freed)
			}
			if memory.writes != 0 && (memory.offset != uint32(tt.handle) || !bytes.Equal(memory.data, tt.data)) {
				t.Fatalf("incorrect write offset=%d data=%q", memory.offset, memory.data)
			}
			if (len(tt.data) == 0 || tt.handle == 0 || tt.handle >= 1<<32 || tt.allocErr != nil) && plugin.memoryReads != 0 {
				t.Fatal("memory was accessed after an invalid/refused allocation")
			}
		})
	}
}

func TestWorkerHostResponseWriteFailureIsFatal(t *testing.T) {
	for _, formatted := range []bool{false, true} {
		name := "normal response"
		if formatted {
			name = "preformatted host error"
		}
		t.Run(name, func(t *testing.T) {
			for _, tt := range []struct {
				name    string
				handle  uint64
				writeOK bool
			}{
				{"null allocation", 0, false},
				{"failed checked write", 128, false},
				{"success", 128, true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					// Both a null allocation and a failed checked write are safe
					// fakes: no real kernel allocation is refused or corrupted.
					memory := &recordingHostMemory{writeOK: tt.writeOK}
					plugin := &fakeExtismHostMemory{memory: memory, handle: tt.handle}
					worker := &pluginWorkerProcess{}
					stack := []uint64{99}
					if formatted {
						worker.writeHostHTTPResponse(context.Background(), plugin, stack, hostHTTPResponse{
							Error: &PluginError{Code: "invalid_request", Message: "fixture error"},
						})
					} else {
						worker.writeHostHTTPResponseBytes(context.Background(), plugin, stack, []byte(`{"status":200}`))
					}
					if tt.writeOK {
						if stack[0] != tt.handle || worker.fatalErr != nil || memory.writes != 1 || len(plugin.freed) != 0 {
							t.Fatalf("valid response failed or was freed: stack=%v fatal=%v", stack, worker.fatalErr)
						}
						return
					}
					if stack[0] != 0 || worker.fatalErr == nil || !strings.Contains(worker.fatalErr.Error(), "write host http response") {
						t.Fatalf("invalid response did not fail the worker RPC: stack=%v fatal=%v", stack, worker.fatalErr)
					}
					if tt.handle == 0 && (memory.writes != 0 || plugin.memoryReads != 0 || len(plugin.freed) != 0) {
						t.Fatal("null allocation touched memory")
					}
					if tt.handle != 0 && (memory.writes != 1 || len(plugin.freed) != 1 || plugin.freed[0] != tt.handle) {
						t.Fatal("failed write was not checked and cleaned up")
					}
				})
			}
		})
	}
}

func TestWorkerHTTPStopsAfterFatalResponseWrite(t *testing.T) {
	t.Setenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB", "2")
	ctx := context.Background()
	var stdin, stdout, stderr bytes.Buffer
	reply, err := workerMessageWithData(workerMessageHostHTTPResponse, workerHostHTTPResponse{
		ResponseBase64: encodeWorkerBytes([]byte(`{"status":200}`)),
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := writeWorkerMessage(&stdin, defaultWorkerResponseMaxBytes, reply); err != nil {
			t.Fatal(err)
		}
	}
	unreadBytes := stdin.Len()
	worker := &pluginWorkerProcess{
		ctx: ctx, stdin: &stdin, stdout: &stdout, stderr: &stderr,
		requestMaxBytes: defaultWorkerRequestMaxBytes, responseMaxBytes: defaultWorkerResponseMaxBytes,
	}
	if err := worker.openPlugin(writeMemoryTestWASM(t, workerCompatibilityTestWASM())); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = worker.instance.Close(ctx) })

	// Inject a null allocation through the checked writer, never through a
	// real kernel. The callback's stdio streams deliberately remain healthy.
	memory := &recordingHostMemory{writeOK: true}
	plugin := &fakeExtismHostMemory{memory: memory}
	stack := []uint64{99}
	worker.writeHostHTTPResponseBytes(ctx, plugin, stack, []byte(`{"status":200}`))
	firstFatal := worker.fatalErr
	if firstFatal == nil || stack[0] != 0 || memory.writes != 0 {
		t.Fatal("expected a safe injected response allocation failure")
	}

	// Call the real HTTP callback using tiny ordinary guest requests. Direct
	// SDK calls retain the export's fatal flag; handleCallExport would reset
	// that flag before a new export, which is not what is being tested.
	for range 2 {
		code, output, err := worker.instance.CallWithContext(ctx, "host_error", []byte("not json"))
		if err != nil || code != 0 || len(output) != 0 {
			t.Fatalf("fatal HTTP callback did not return a zero handle: code=%d output=%q error=%v", code, output, err)
		}
	}
	if stdout.Len() != 0 || stdin.Len() != unreadBytes {
		t.Fatalf("forwarded more HTTP requests after fatal: stdout=%d unread=%d, want %d", stdout.Len(), stdin.Len(), unreadBytes)
	}
	worker.failHostRPC(stack, errors.New("later RPC failure"))
	if worker.fatalErr != firstFatal || stack[0] != 0 {
		t.Fatalf("earliest fatal error was replaced: %v", worker.fatalErr)
	}
}
