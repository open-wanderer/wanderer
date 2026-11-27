package pluginsystem

import (
	"context"
	"errors"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

// extismHostMemory is the existing Extism allocator and linear memory, not an
// alternative allocator. The interface also permits safe refusal tests without
// exhausting or corrupting a real kernel.
type extismHostMemory interface {
	AllocWithContext(context.Context, uint64) (uint64, error)
	FreeWithContext(context.Context, uint64) error
	Memory() api.Memory
}

// writeHostMemory avoids go-sdk v1.7.1 WriteBytes, which writes at address zero
// when the kernel refuses an allocation by returning a null handle.
func writeHostMemory(ctx context.Context, plugin extismHostMemory, data []byte) (uint64, error) {
	const addressSpaceBytes = uint64(1) << 32
	length := uint64(len(data))
	if length == 0 || length >= addressSpaceBytes {
		return 0, fmt.Errorf("invalid host response payload size")
	}
	offset, err := plugin.AllocWithContext(ctx, length)
	if err != nil {
		return 0, fmt.Errorf("allocate host response: %w", err)
	}
	// Do not write, truncate, or free a null/non-Wasm32 allocation handle.
	if offset == 0 || offset >= addressSpaceBytes {
		return 0, fmt.Errorf("invalid host response allocation handle")
	}

	var writeErr error
	if length > addressSpaceBytes-offset {
		writeErr = fmt.Errorf("host response allocation exceeds Wasm32 address range")
	} else if memory := plugin.Memory(); memory == nil || !memory.Write(uint32(offset), data) {
		writeErr = fmt.Errorf("write host response memory failed")
	}
	if writeErr != nil {
		if err := plugin.FreeWithContext(ctx, offset); err != nil {
			writeErr = errors.Join(writeErr, fmt.Errorf("free failed host response allocation: %w", err))
		}
		return 0, writeErr
	}
	return offset, nil
}
