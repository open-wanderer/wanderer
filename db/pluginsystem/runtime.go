package pluginsystem

import (
	"context"
	"errors"
	"fmt"
)

var ErrRuntimeUnavailable = errors.New("plugin runtime is not available")

const (
	DefaultMaxHostRequestsPerCall  = 64
	AbsoluteMaxHostRequestsPerCall = 512
)

type RuntimeCallOptions struct {
	MaxHostRequests int
}

type Runtime interface {
	Call(ctx context.Context, plugin LocalPlugin, export string, input []byte, policy RequestPolicyContext, options RuntimeCallOptions) ([]byte, error)
	OpenSession(ctx context.Context, plugin LocalPlugin, policy RequestPolicyContext) (RuntimeSession, error)
}

type RuntimeSession interface {
	Call(ctx context.Context, export string, input []byte, options RuntimeCallOptions) ([]byte, error)
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

func (UnavailableRuntime) Call(context.Context, LocalPlugin, string, []byte, RequestPolicyContext, RuntimeCallOptions) ([]byte, error) {
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

type HostRequestBudgetError struct {
	Limit int
}

func (e HostRequestBudgetError) Error() string {
	return fmt.Sprintf("plugin host request budget exceeded (limit %d)", e.Limit)
}

func EffectiveMaxHostRequests(options RuntimeCallOptions) int {
	limit := options.MaxHostRequests
	if limit <= 0 {
		limit = DefaultMaxHostRequestsPerCall
	}
	if limit > AbsoluteMaxHostRequestsPerCall {
		limit = AbsoluteMaxHostRequestsPerCall
	}
	return limit
}

func (e PluginCallError) Error() string {
	if e.PluginError.Message == "" {
		return fmt.Sprintf("call %s.%s: %s", e.PluginID, e.Export, e.PluginError.Code)
	}
	return fmt.Sprintf("call %s.%s: %s: %s", e.PluginID, e.Export, e.PluginError.Code, e.PluginError.Message)
}
