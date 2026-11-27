package pluginsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	extism "github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
)

const (
	defaultWorkerWASMMemoryMiB = 256
	minWorkerWASMMemoryMiB     = 2
	maxWorkerWASMMemoryMiB     = 4096 // wazero's Wasm32 maximum: 65536 pages.
	wasmPagesPerMiB            = 16   // A WebAssembly page is 64 KiB.
	wasmPageBytes              = 64 * 1024
)

// workerWASMMemoryLimitPages always returns a nonzero, supported per-memory
// limit. Invalid settings fall back to the default, like other worker settings.
func workerWASMMemoryLimitPages() uint32 {
	memoryMiB, err := strconv.ParseInt(strings.TrimSpace(os.Getenv("WANDERER_PLUGIN_WORKER_WASM_MEMORY_MIB")), 10, 16)
	if err != nil || memoryMiB < minWorkerWASMMemoryMiB || memoryMiB > maxWorkerWASMMemoryMiB {
		return defaultWorkerWASMMemoryMiB * wasmPagesPerMiB
	}
	return uint32(memoryMiB) * wasmPagesPerMiB
}

// RunPluginWorker runs the stdio worker process. It is called by the main
// binary's plugin-worker subcommand before PocketBase is initialized.
func RunPluginWorker(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	_ = os.Unsetenv("EXTISM_ENABLE_WASI_OUTPUT")

	worker := &pluginWorkerProcess{
		ctx:              ctx,
		stdin:            stdin,
		stdout:           stdout,
		stderr:           stderr,
		requestMaxBytes:  envInt("WANDERER_PLUGIN_WORKER_REQUEST_BYTES", defaultWorkerRequestMaxBytes),
		responseMaxBytes: envInt("WANDERER_PLUGIN_WORKER_RESPONSE_BYTES", defaultWorkerResponseMaxBytes),
	}
	if err := worker.run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "plugin worker: %v\n", err)
		return 1
	}
	return 0
}

type pluginWorkerProcess struct {
	ctx              context.Context
	stdin            io.Reader
	stdout           io.Writer
	stderr           io.Writer
	requestMaxBytes  int
	responseMaxBytes int
	wasmPath         string
	sessionID        string
	instance         *extism.Plugin
	inputMaxBytes    uint64
	fatalErr         error
}

func (w *pluginWorkerProcess) run() error {
	defer func() {
		if w.instance != nil {
			_ = w.instance.Close(w.ctx)
		}
	}()

	for {
		msg, err := readWorkerMessage(w.stdin, w.requestMaxBytes)
		if err != nil {
			// A clean io.EOF means the parent closed stdin without a
			// shutdown frame (e.g. it crashed); exit quietly. An
			// io.ErrUnexpectedEOF means stdin was cut mid-frame, which is a
			// truncated/corrupt frame and should surface as an error.
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch msg.Type {
		case workerMessageShutdown:
			return nil
		case workerMessageCallExport:
			if err := w.handleCallExport(msg); err != nil {
				_ = w.sendError(err.Error())
				return err
			}
		default:
			err := fmt.Errorf("unexpected worker message %q", msg.Type)
			_ = w.sendError(err.Error())
			return err
		}
	}
}

func (w *pluginWorkerProcess) handleCallExport(msg workerMessage) error {
	call, err := workerData[workerCallExport](msg)
	if err != nil {
		return err
	}
	if call.WASMPath == "" || call.Export == "" {
		return fmt.Errorf("call_export requires wasmPath and export")
	}
	// The session ID is set by the parent once per worker process and reused
	// for every call. It carries no routing semantics here (a worker serves a
	// single wasm path) but is threaded into errors so captured stderr can be
	// tied back to a specific session during diagnosis.
	w.sessionID = call.SessionID
	if w.instance == nil {
		if err := w.openPlugin(call.WASMPath); err != nil {
			return w.errCtx(call.Export, err)
		}
	} else if call.WASMPath != w.wasmPath {
		return w.errCtx(call.Export, fmt.Errorf("worker session cannot switch wasm path (have %q, got %q)", w.wasmPath, call.WASMPath))
	}

	input, err := decodeWorkerBytes(call.InputBase64)
	if err != nil {
		return w.errCtx(call.Export, fmt.Errorf("decode call input: %w", err))
	}
	// go-sdk v1.7.1 SetInput ignores null allocation handles and failed writes.
	// Reject unsafe sizes before it resets/allocates/writes kernel memory.
	if uint64(len(input)) > w.inputMaxBytes {
		return w.errCtx(call.Export, fmt.Errorf("call input is %d bytes; safe worker input limit is %d bytes", len(input), w.inputMaxBytes))
	}
	w.fatalErr = nil
	code, output, err := w.instance.CallWithContext(w.ctx, call.Export, input)
	if w.fatalErr != nil {
		return w.errCtx(call.Export, w.fatalErr)
	}
	// Extism also returns a Go error when pdk.SetError accompanies a nonzero
	// export code. A structured guest rejection belongs on call_result and
	// must not be mistaken for a fatal WASM trap.
	if pluginErr := pluginErrorForResult(call.Export, code, w.instance.GetErrorWithContext(w.ctx), err); pluginErr != nil {
		return w.sendCallResult(workerCallResult{PluginError: pluginErr})
	}
	if err != nil {
		return w.errCtx(call.Export, fmt.Errorf("call %s: %w", call.Export, err))
	}
	return w.sendCallResult(workerCallResult{OutputBase64: encodeWorkerBytes(output)})
}

// pluginErrorForResult distinguishes a business rejection from a runtime trap.
// A Go call error is compatible with guest rejection only when the guest set a
// bounded structured error; malformed trap text stays on the fatal channel.
func pluginErrorForResult(export string, code uint32, rawErr string, callErr error) *PluginError {
	if code == 0 {
		return nil
	}
	if callErr != nil {
		var guest PluginError
		if len(rawErr) > 64*1024 || json.Unmarshal([]byte(rawErr), &guest) != nil || guest.Code == "" {
			return nil
		}
	}
	normalized := pluginErrorForCode(export, code, rawErr)
	return &normalized
}

// pluginErrorForCode maps a non-zero export return code into the PluginError
// reported to the parent. It prefers the structured error JSON the plugin set
// via the host error API, and falls back to a generic plugin_error when that
// payload is missing, malformed, or has no code.
func pluginErrorForCode(export string, code uint32, rawErr string) PluginError {
	var parsed PluginError
	if rawErr == "" || json.Unmarshal([]byte(rawErr), &parsed) != nil || parsed.Code == "" {
		return PluginError{
			Code:    "plugin_error",
			Message: fmt.Sprintf("call %s failed with code %d", export, code),
		}
	}
	return parsed
}

func (w *pluginWorkerProcess) openPlugin(wasmPath string) error {
	memoryLimitPages := workerWASMMemoryLimitPages()
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: wasmPath},
		},
	}
	instance, err := extism.NewPlugin(w.ctx, manifest, extism.PluginConfig{
		EnableWasi: true,
		// Keep Manifest.Memory nil: its zero-valued fields disable Extism
		// variables and change HTTP defaults. This cap is per linear memory,
		// not a hard limit on the worker's RSS or Go backing-buffer capacity.
		RuntimeConfig: wazero.NewRuntimeConfig().WithMemoryLimitPages(memoryLimitPages),
	}, w.hostFunctions())
	if err != nil {
		return fmt.Errorf("create wasm plugin (per-memory limit %d MiB): %w", memoryLimitPages/wasmPagesPerMiB, err)
	}
	w.wasmPath = wasmPath
	w.instance = instance
	w.inputMaxBytes = workerInputLimitBytes(memoryLimitPages, uint64(instance.Memory().Size()))
	return nil
}

// workerInputLimitBytes bounds the fresh allocation made after Extism's reset.
// The pinned v1.7.0 kernel initially accounts for only one 64 KiB page despite
// starting with 1 MiB of memory; subsequent growth updates both sizes equally.
// Reserving its entire observed initial memory leaves one extra page for root,
// block metadata and boundary rounding, rather than depending on exact struct
// sizes. This is a conservative size check, not a promise that allocation can
// never fail for other reasons or that later guest/host allocations will fit.
func workerInputLimitBytes(memoryLimitPages uint32, kernelInitialBytes uint64) uint64 {
	limitBytes := uint64(memoryLimitPages) * wasmPageBytes
	if kernelInitialBytes < wasmPageBytes || kernelInitialBytes >= limitBytes {
		return 0
	}
	return limitBytes - kernelInitialBytes
}

func (w *pluginWorkerProcess) hostFunctions() []extism.HostFunction {
	httpFn := extism.NewHostFunctionWithStack(
		"http_request",
		func(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64) {
			// A guest may try fallback requests after a fatal host response
			// failure while stdio is still healthy. Do no more work for it.
			if w.fatalErr != nil {
				stack[0] = 0
				return
			}
			requestBytes, err := plugin.ReadBytes(stack[0])
			if err != nil {
				w.writeHostHTTPResponse(ctx, plugin, stack, hostHTTPResponse{
					Error: &PluginError{Code: "invalid_request", Message: err.Error()},
				})
				return
			}
			msg, err := workerMessageWithData(workerMessageHostHTTPRequest, workerHostHTTPRequest{
				RequestBase64: encodeWorkerBytes(requestBytes),
			})
			if err != nil {
				w.writeHostHTTPResponse(ctx, plugin, stack, hostHTTPResponse{
					Error: &PluginError{Code: "internal_error", Message: err.Error()},
				})
				return
			}
			if err := writeWorkerMessage(w.stdout, w.responseMaxBytes, msg); err != nil {
				w.failHostRPC(stack, fmt.Errorf("write host http request: %w", err))
				return
			}
			responseMsg, err := readWorkerMessage(w.stdin, w.responseMaxBytes)
			if err != nil {
				w.failHostRPC(stack, fmt.Errorf("read host http response: %w", err))
				return
			}
			if responseMsg.Type != workerMessageHostHTTPResponse {
				w.failHostRPC(stack, fmt.Errorf("unexpected host http response message %q", responseMsg.Type))
				return
			}
			response, err := workerData[workerHostHTTPResponse](responseMsg)
			if err != nil {
				w.failHostRPC(stack, fmt.Errorf("decode host http response: %w", err))
				return
			}
			responseBytes, err := decodeWorkerBytes(response.ResponseBase64)
			if err != nil {
				w.failHostRPC(stack, fmt.Errorf("decode host http response bytes: %w", err))
				return
			}
			w.writeHostHTTPResponseBytes(ctx, plugin, stack, responseBytes)
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)
	httpFn.SetNamespace("wanderer")

	logFn := extism.NewHostFunctionWithStack(
		"log",
		func(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64) {
			message, err := readBoundedHostLogPayload(plugin, stack[0])
			if err != nil {
				plugin.Log(extism.LogLevelError, "read host log message: "+err.Error())
				return
			}
			entry, err := parseHostLogEntry(message)
			if err != nil {
				_, _ = fmt.Fprintf(w.stderr, "plugin log invalid: session %s: %v\n", w.sessionID, err)
				return
			}
			msg, err := workerMessageWithData(workerMessageHostLog, workerHostLog{
				Level:     entry.Level,
				Message:   entry.Message,
				SessionID: w.sessionID,
			})
			if err != nil {
				_, _ = fmt.Fprintf(w.stderr, "plugin log encode failed: session %s: %v\n", w.sessionID, err)
				return
			}
			if err := writeWorkerMessage(w.stdout, w.responseMaxBytes, msg); err != nil {
				_, _ = fmt.Fprintf(w.stderr, "plugin log write failed: session %s: %v\n", w.sessionID, err)
			}
			_ = ctx
		},
		[]extism.ValueType{extism.ValueTypePTR},
		nil,
	)
	logFn.SetNamespace("wanderer")

	return []extism.HostFunction{httpFn, logFn}
}

func (w *pluginWorkerProcess) writeHostHTTPResponse(ctx context.Context, plugin extismHostMemory, stack []uint64, response hostHTTPResponse) {
	if err := writeHostHTTPResponse(ctx, plugin, stack, response); err != nil {
		w.failHostRPC(stack, err)
	}
}

func (w *pluginWorkerProcess) writeHostHTTPResponseBytes(ctx context.Context, plugin extismHostMemory, stack []uint64, responseBytes []byte) {
	offset, err := writeHostMemory(ctx, plugin, responseBytes)
	if err != nil {
		w.failHostRPC(stack, fmt.Errorf("write host http response: %w", err))
		return
	}
	stack[0] = offset
}

func (w *pluginWorkerProcess) failHostRPC(stack []uint64, err error) {
	if w.fatalErr == nil {
		w.fatalErr = err
	}
	stack[0] = 0
}

// errCtx annotates a fatal worker error with the active session and export so
// the message that the parent captures from stderr can be tied back to a
// specific call during diagnosis.
func (w *pluginWorkerProcess) errCtx(export string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("session %s export %s: %w", w.sessionID, export, err)
}

// Worker error channels follow a strict convention:
//
//   - sendCallResult with a PluginError reports a business-level rejection from
//     the plugin (a bad call code). The session stays alive and reusable; the
//     parent surfaces it as a PluginCallError.
//   - sendError reports a broken protocol or runtime (corrupt frame, host RPC
//     failure, unexpected message). The parent treats it as fatal and tears the
//     session down.
//
// Keep new failure paths on the correct channel: recoverable plugin outcomes
// use sendCallResult, anything that invalidates the session uses sendError.
func (w *pluginWorkerProcess) sendCallResult(result workerCallResult) error {
	msg, err := workerMessageWithData(workerMessageCallResult, result)
	if err != nil {
		return err
	}
	return writeWorkerMessage(w.stdout, w.responseMaxBytes, msg)
}

func (w *pluginWorkerProcess) sendError(message string) error {
	msg, err := workerMessageWithData(workerMessageError, workerError{Message: message})
	if err != nil {
		return err
	}
	return writeWorkerMessage(w.stdout, w.responseMaxBytes, msg)
}
