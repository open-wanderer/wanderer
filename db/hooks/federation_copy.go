package hooks

import "github.com/pocketbase/pocketbase/core"

// Only an internal merge may mark a copied entry. Requests cannot remove that
// restriction and announce another author's private source to their followers.
func PreserveSummitLogFederationSuppression() func(*core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		e.Record.Set("federation_suppressed", e.Record.Original().GetBool("federation_suppressed"))
		return e.Next()
	}
}
