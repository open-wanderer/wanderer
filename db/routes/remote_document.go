package routes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pocketbase/util"
)

func fetchRemoteDocument(ctx context.Context, client *http.Client, remoteURL *url.URL, kind string, limits ...int64) (map[string]any, error) {
	releaseFetch, err := acquireRemoteSyncFetch(ctx, remoteURL.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	defer releaseFetch()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		statusErr := &remoteFetchStatusError{
			Kind:       kind,
			URL:        remoteURL.String(),
			StatusCode: res.StatusCode,
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, statusErr)
		}
		if isGoneStatus(res.StatusCode) {
			return nil, fmt.Errorf("%w: %w", errRemoteGone, statusErr)
		}
		return nil, statusErr
	}

	limit := util.FederationObjectBodyLimit
	if len(limits) > 0 {
		limit = limits[0]
	}
	var data map[string]any
	decode := decodeRemoteFederationJSON
	if limit == util.FederationSyncBodyLimit {
		decode = decodeRemoteSyncJSON
	}
	if ctx.Value(remoteSyncStagingKey{}) != nil {
		decode = decodeStagedRemoteSyncJSON
	}
	if err := decode(ctx, res.Body, limit, &data); err != nil {
		if errors.Is(err, util.ErrFederationBodyTooLarge) || errors.Is(err, errRemoteSyncJSONComplexity) {
			return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
		}
		return nil, err
	}
	return data, nil
}

type remoteFetchStatusError struct {
	Kind       string
	URL        string
	StatusCode int
}

func (err *remoteFetchStatusError) Error() string {
	return fmt.Sprintf("remote %s fetch %s returned: %d", err.Kind, err.URL, err.StatusCode)
}

// Small canonical responses share processing capacity too, after the entire
// bounded body has arrived. A slow response never holds a processing permit.
func decodeStagedRemoteSyncJSON(ctx context.Context, body io.Reader, limit int64, value any) error {
	if budget, ok := ctx.Value(canonicalRemoteBodyBudgetKey{}).(*canonicalRemoteBodyBudget); ok {
		body = canonicalRemoteBudgetReader{Reader: body, budget: budget}
	}
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > limit {
		return util.ErrFederationBodyTooLarge
	}
	tokens := 0
	if limit == util.FederationSyncBodyLimit {
		tokens, err = remoteSyncJSONTokens(ctx, bytes.NewReader(data))
		if err != nil {
			return err
		}
	}
	release, err := acquireRemoteSyncProcessing(ctx, tokens)
	if err != nil {
		return err
	}
	defer release()
	if limit == util.FederationSyncBodyLimit {
		return decodeRemoteSyncJSON(ctx, bytes.NewReader(data), limit, value)
	}
	return util.DecodeFederationJSON(bytes.NewReader(data), limit, value)
}
