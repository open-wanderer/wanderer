package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type preparedFederatedFilesKey struct{}
type federatedMediaByteBudgetKey struct{}

var ErrFederatedMediaBudget = errors.New("federated media byte budget exhausted")

type federatedMediaByteBudget struct {
	mu        sync.Mutex
	remaining int64
}

// WithFederatedMediaByteBudget limits bytes read across successful and failed
// media responses in one sync, including over-limit response probes.
func WithFederatedMediaByteBudget(ctx context.Context, bytes int64) context.Context {
	return context.WithValue(ctx, federatedMediaByteBudgetKey{}, &federatedMediaByteBudget{remaining: bytes})
}

type budgetedFederatedReader struct {
	body   io.Reader
	budget *federatedMediaByteBudget
}

func (r budgetedFederatedReader) Read(p []byte) (int, error) {
	r.budget.mu.Lock()
	defer r.budget.mu.Unlock()
	if r.budget.remaining <= 0 {
		var probe [1]byte
		n, err := r.body.Read(probe[:])
		if n > 0 {
			return 0, ErrFederatedMediaBudget
		}
		return 0, err
	}
	if int64(len(p)) > r.budget.remaining+1 {
		p = p[:r.budget.remaining+1]
	}
	n, err := r.body.Read(p)
	r.budget.remaining -= int64(n)
	if r.budget.remaining < 0 {
		return n, ErrFederatedMediaBudget
	}
	return n, err
}

func boundedFederatedBody(ctx context.Context, body io.Reader) io.Reader {
	if budget, ok := ctx.Value(federatedMediaByteBudgetKey{}).(*federatedMediaByteBudget); ok {
		return budgetedFederatedReader{body, budget}
	}
	return body
}

// PreparedFederatedFile holds bytes fetched before a database transaction.
// A nil File records an unavailable or omitted download.
type PreparedFederatedFile struct {
	URL, OriginIRI string
	File           *filesystem.File
}

func federatedFileKey(rawURL, originIRI string) string {
	origin, err := url.Parse(originIRI)
	if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "http" && origin.Scheme != "https") {
		return ""
	}
	return rawURL + "\x00" + strings.ToLower(origin.Host)
}

// WithPreparedFederatedFiles closes the network path for this context. A
// later file lookup can only return prepared bytes, including when the
// origin document references files omitted by a download budget.
func WithPreparedFederatedFiles(ctx context.Context, files []PreparedFederatedFile) context.Context {
	prepared := make(map[string]*filesystem.File, len(files))
	for _, file := range files {
		prepared[federatedFileKey(file.URL, file.OriginIRI)] = file.File
	}
	return context.WithValue(ctx, preparedFederatedFilesKey{}, prepared)
}

func preparedFederatedFile(ctx context.Context, rawURL, originIRI string, maxBytes int64) (*filesystem.File, bool, error) {
	prepared, enabled := ctx.Value(preparedFederatedFilesKey{}).(map[string]*filesystem.File)
	if !enabled {
		return nil, false, nil
	}
	key := federatedFileKey(rawURL, originIRI)
	if key == "" {
		return nil, true, fmt.Errorf("invalid prepared federation origin")
	}
	file := prepared[key]
	if file == nil {
		return nil, true, fmt.Errorf("federated file was not prepared")
	}
	if maxBytes > 0 && file.Size > maxBytes {
		return nil, true, fmt.Errorf("prepared federated file exceeds size limit")
	}
	return file, true, nil
}

// PreparedRemoteSyncFile only returns downloads prepared before a sync transaction.
func PreparedRemoteSyncFile(ctx context.Context, rawURL, originIRI string, maxBytes int64) (*filesystem.File, error) {
	file, enabled, err := preparedFederatedFile(ctx, rawURL, originIRI, maxBytes)
	if !enabled {
		return nil, fmt.Errorf("remote sync file was not prepared")
	}
	return file, err
}
