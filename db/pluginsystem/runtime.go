package pluginsystem

import (
	"context"
	"errors"
	"fmt"
)

var ErrRuntimeUnavailable = errors.New("plugin runtime is not available")

type Runtime interface {
	Call(ctx context.Context, plugin LocalPlugin, export string, input []byte, policy RequestPolicyContext) ([]byte, error)
	OpenSession(ctx context.Context, plugin LocalPlugin, policy RequestPolicyContext) (RuntimeSession, error)
}

type RuntimeSession interface {
	Call(ctx context.Context, export string, input []byte) ([]byte, error)
	Close(ctx context.Context) error
}

type RuntimeRegistry struct {
	wasm Runtime
}

// NewRuntimeRegistry wires available runtime implementations behind the common
// Runtime interface.
func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		wasm: NewWorkerRuntime(),
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

func (UnavailableRuntime) OpenSession(context.Context, LocalPlugin, RequestPolicyContext) (RuntimeSession, error) {
	return nil, ErrRuntimeUnavailable
}

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
