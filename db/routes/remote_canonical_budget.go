package routes

import (
	"context"
	"errors"
	"io"
	"sync/atomic"

	"pocketbase/util"
)

const canonicalRemoteBodyLimit int64 = 256 << 10
const canonicalRemoteByteBudget int64 = 2 << 20

var errCanonicalRemoteByteBudget = errors.New("canonical response byte budget exhausted")

type canonicalRemoteBodyBudgetKey struct{}
type canonicalRemoteBodyBudget struct{ remaining atomic.Int64 }

func withCanonicalRemoteBodyBudget(ctx context.Context) context.Context {
	if _, ok := ctx.Value(canonicalRemoteBodyBudgetKey{}).(*canonicalRemoteBodyBudget); ok {
		return ctx
	}
	budget := &canonicalRemoteBodyBudget{}
	budget.remaining.Store(canonicalRemoteByteBudget)
	return context.WithValue(ctx, canonicalRemoteBodyBudgetKey{}, budget)
}

func canonicalRemoteBodyBudgetExhausted(ctx context.Context) bool {
	budget, _ := ctx.Value(canonicalRemoteBodyBudgetKey{}).(*canonicalRemoteBodyBudget)
	return budget != nil && budget.remaining.Load() <= 0
}

type canonicalRemoteBudgetReader struct {
	io.Reader
	budget *canonicalRemoteBodyBudget
}

func (r canonicalRemoteBudgetReader) Read(p []byte) (int, error) {
	for {
		remaining := r.budget.remaining.Load()
		if remaining < 0 {
			return 0, errCanonicalRemoteByteBudget
		}
		// Reserve actual read capacity across concurrent responses. One extra byte
		// distinguishes a complete document at the boundary from an oversized one.
		reserved := min(int64(len(p)), remaining+1)
		if !r.budget.remaining.CompareAndSwap(remaining, remaining-reserved) {
			continue
		}
		n, err := r.Reader.Read(p[:reserved])
		r.budget.remaining.Add(reserved - int64(n))
		if int64(n) > remaining {
			return 0, errCanonicalRemoteByteBudget
		}
		return n, err
	}
}

func decodeRemoteFederationJSON(ctx context.Context, body io.Reader, limit int64, value any) error {
	if budget, ok := ctx.Value(canonicalRemoteBodyBudgetKey{}).(*canonicalRemoteBodyBudget); ok {
		body = canonicalRemoteBudgetReader{Reader: body, budget: budget}
	}
	return util.DecodeFederationJSON(body, limit, value)
}

// Keep the schema values used by the import, rather than retaining unrelated
// nested response objects for all contributions until the transaction starts.
func canonicalSummitLogData(raw map[string]any) map[string]any {
	result := canonicalScalarFields(raw, "id", "iri", "name", "text", "date", "gpx", "distance", "elevation_gain", "elevation_loss", "duration")
	if photos, ok := raw["photos"].([]any); ok {
		values := make([]any, 0, len(photos))
		for _, photo := range photos {
			if filename, ok := photo.(string); ok {
				values = append(values, filename)
			} else {
				values = append(values, nil)
			}
		}
		result["photos"] = values
	}
	return result
}

func canonicalScalarFields(raw map[string]any, fields ...string) map[string]any {
	result := map[string]any{}
	for _, field := range fields {
		switch value := raw[field].(type) {
		case string, float64, bool:
			result[field] = value
		}
	}
	return result
}
