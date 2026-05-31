package pluginsystem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	extism "github.com/extism/go-sdk"
)

var ErrRuntimeUnavailable = errors.New("plugin runtime is not available")

type Runtime interface {
	Call(ctx context.Context, plugin LocalPlugin, export string, input []byte, policy RequestPolicyContext) ([]byte, error)
}

type RuntimeRegistry struct {
	wasm Runtime
}

// NewRuntimeRegistry wires available runtime implementations behind the common
// Runtime interface.
func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		wasm: ExtismRuntime{},
	}
}

// RuntimeFor selects the runtime declared by a plugin manifest.
func (r *RuntimeRegistry) RuntimeFor(plugin LocalPlugin) (Runtime, error) {
	switch plugin.Manifest.Runtime.Type {
	case RuntimeWASM:
		return r.wasm, nil
	default:
		return nil, ErrRuntimeUnavailable
	}
}

type UnavailableRuntime struct{}

func (UnavailableRuntime) Call(context.Context, LocalPlugin, string, []byte, RequestPolicyContext) ([]byte, error) {
	return nil, ErrRuntimeUnavailable
}

type ExtismRuntime struct{}

type PluginCallError struct {
	PluginID    string
	Export      string
	PluginError PluginError
}

func (e PluginCallError) Error() string {
	if e.PluginError.Message == "" {
		return fmt.Sprintf("call %s.%s: %s", e.PluginID, e.Export, e.PluginError.Code)
	}
	return fmt.Sprintf("call %s.%s: %s: %s", e.PluginID, e.Export, e.PluginError.Code, e.PluginError.Message)
}

// Call loads the plugin WASM module, attaches wanderer host functions, invokes
// the requested export, and converts plugin-reported JSON errors into Go errors.
func (ExtismRuntime) Call(ctx context.Context, plugin LocalPlugin, export string, input []byte, policy RequestPolicyContext) ([]byte, error) {
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: plugin.WASMPath},
		},
	}
	instance, err := extism.NewPlugin(ctx, manifest, extism.PluginConfig{
		EnableWasi: true,
	}, extismHostFunctions(plugin.Manifest, policy))
	if err != nil {
		return nil, fmt.Errorf("create wasm plugin %s: %w", plugin.Manifest.ID, err)
	}
	defer func() {
		_ = instance.Close(ctx)
	}()

	code, output, err := instance.CallWithContext(ctx, export, input)
	if err != nil {
		return nil, fmt.Errorf("call %s.%s: %w", plugin.Manifest.ID, export, err)
	}
	if code != 0 {
		if pluginErr := instance.GetErrorWithContext(ctx); pluginErr != "" {
			var parsed PluginError
			if err := json.Unmarshal([]byte(pluginErr), &parsed); err == nil && parsed.Code != "" {
				return nil, PluginCallError{PluginID: plugin.Manifest.ID, Export: export, PluginError: parsed}
			}
			return nil, fmt.Errorf("call %s.%s failed with code %d: %s", plugin.Manifest.ID, export, code, pluginErr)
		}
		return nil, fmt.Errorf("call %s.%s failed with code %d", plugin.Manifest.ID, export, code)
	}
	return output, nil
}
