package pluginsystem

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	extism "github.com/extism/go-sdk"
)

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
	instance         *extism.Plugin
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
			if err == io.EOF || err == io.ErrUnexpectedEOF {
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
	if w.instance == nil {
		if err := w.openPlugin(call.WASMPath); err != nil {
			return err
		}
	} else if call.WASMPath != w.wasmPath {
		return fmt.Errorf("worker session cannot switch wasm path")
	}

	input, err := decodeWorkerBytes(call.InputBase64)
	if err != nil {
		return err
	}
	w.fatalErr = nil
	code, output, err := w.instance.CallWithContext(w.ctx, call.Export, input)
	if w.fatalErr != nil {
		return w.fatalErr
	}
	if err != nil {
		return fmt.Errorf("call %s: %w", call.Export, err)
	}
	if code != 0 {
		pluginErr := w.instance.GetErrorWithContext(w.ctx)
		var parsed PluginError
		if pluginErr == "" || json.Unmarshal([]byte(pluginErr), &parsed) != nil || parsed.Code == "" {
			return w.sendCallResult(workerCallResult{PluginError: &PluginError{
				Code:    "plugin_error",
				Message: fmt.Sprintf("call %s failed with code %d", call.Export, code),
			}})
		}
		return w.sendCallResult(workerCallResult{PluginError: &parsed})
	}
	return w.sendCallResult(workerCallResult{OutputBase64: encodeWorkerBytes(output)})
}

func (w *pluginWorkerProcess) openPlugin(wasmPath string) error {
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: wasmPath},
		},
	}
	instance, err := extism.NewPlugin(w.ctx, manifest, extism.PluginConfig{
		EnableWasi: true,
	}, w.hostFunctions())
	if err != nil {
		return fmt.Errorf("create wasm plugin: %w", err)
	}
	w.wasmPath = wasmPath
	w.instance = instance
	return nil
}

func (w *pluginWorkerProcess) hostFunctions() []extism.HostFunction {
	fn := extism.NewHostFunctionWithStack(
		"http_request",
		func(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64) {
			requestBytes, err := plugin.ReadBytes(stack[0])
			if err != nil {
				w.writeHostFunctionResponse(ctx, plugin, stack, hostHTTPResponse{
					Error: &PluginError{Code: "invalid_request", Message: err.Error()},
				})
				return
			}
			msg, err := workerMessageWithData(workerMessageHostHTTPRequest, workerHostHTTPRequest{
				RequestBase64: encodeWorkerBytes(requestBytes),
			})
			if err != nil {
				w.writeHostFunctionResponse(ctx, plugin, stack, hostHTTPResponse{
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
			offset, err := plugin.WriteBytes(responseBytes)
			if err != nil {
				plugin.Log(extism.LogLevelError, "write host http response: "+err.Error())
				stack[0] = 0
				return
			}
			stack[0] = offset
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)
	fn.SetNamespace("wanderer")
	return []extism.HostFunction{fn}
}

func (w *pluginWorkerProcess) writeHostFunctionResponse(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64, response hostHTTPResponse) {
	_ = ctx
	writeHostHTTPResponse(ctx, plugin, stack, response)
}

func (w *pluginWorkerProcess) failHostRPC(stack []uint64, err error) {
	w.fatalErr = err
	stack[0] = 0
}

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
