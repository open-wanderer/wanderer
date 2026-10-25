package pluginsystem

import "fmt"

// A protocol-limit rejection is separate from provider transport failures.
// The host rejects the whole oversized batch before importing any items.
type SyncItemLimitError struct{ Returned, MaxItems int }

func (e SyncItemLimitError) Error() string {
	return fmt.Sprintf("plugin returned %d sync items, exceeding maxItems %d", e.Returned, e.MaxItems)
}

const SyncItemLimitMessage = "The plugin returned too many sync items. Update or report the plugin so it respects limits.maxItems."
