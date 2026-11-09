package routes

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"io"
	"net/http"
	"net/url"
	"pocketbase/util"
)

type remoteTrailShareCandidate struct {
	token  string
	stored bool
}

// A viewer's optional share is tried after every stored directed share. Only
// a credential-free confirmation can retract a copy when all candidates fail.
func fetchRemoteTrailWithShares(app core.App, ctx context.Context, requestURL *url.URL, trail *core.Record) (context.Context, *remoteSyncDocument, bool, error) {
	candidates := []remoteTrailShareCandidate{}
	seen := map[string]bool{}
	for _, token := range util.RemoteTrailShareTokens(app, trail.Id) {
		if !seen[token] {
			candidates = append(candidates, remoteTrailShareCandidate{token: token, stored: true})
			seen[token] = true
		}
	}
	if token := remoteSyncQueryParameters(requestURL).Get("share"); token != "" && !seen[token] {
		candidates = append(candidates, remoteTrailShareCandidate{token: token})
	}
	candidates = append(candidates, remoteTrailShareCandidate{})
	denied := map[string]bool{}
	client := newRemoteSyncHTTPClient()
	chosenCtx := ctx
	var document *remoteSyncDocument
	var fetchErr error
	assetsRequested := true
	for _, candidate := range candidates {
		candidateCtx := util.WithFederatedShareToken(ctx, trail.GetString("iri"), candidate.token)
		if document != nil {
			if candidate.stored {
				probeURL, _ := url.Parse(trail.GetString("iri"))
				probeURL.RawQuery = url.Values{"fields": {"id"}}.Encode()
				probeURL, _ = url.Parse(util.FederatedShareURL(candidateCtx, probeURL.String()))
				shareDenied, probeErr := probeRemoteTrailShare(candidateCtx, client, probeURL)
				if probeErr == nil && shareDenied {
					denied[candidate.token] = true
				}
			}
			continue
		}
		remoteURL, _ := url.Parse(trail.GetString("iri"))
		remoteURL.RawQuery = remoteTrailSyncQuery(nil).Encode()
		remoteURL, _ = url.Parse(util.FederatedShareURL(candidateCtx, remoteURL.String()))
		assetsRequested = true
		document, fetchErr = fetchRemoteSyncDocument(candidateCtx, client, remoteURL, "trail")
		if shouldRetryRemoteFetchWithoutAssetExpands(fetchErr) {
			remoteURL.RawQuery = remoteTrailSyncAssetQuery(nil, false).Encode()
			remoteURL, _ = url.Parse(util.FederatedShareURL(candidateCtx, remoteURL.String()))
			document, fetchErr = fetchRemoteSyncDocument(candidateCtx, client, remoteURL, "trail")
			if fetchErr == nil {
				assetsRequested = false
			}
		}
		if fetchErr == nil {
			chosenCtx = candidateCtx
			continue
		}
		var status *remoteFetchStatusError
		if !errors.As(fetchErr, &status) || !remoteTrailShareFallbackStatus(status.StatusCode) {
			break
		}
		if candidate.stored && status.ShareDenied {
			denied[candidate.token] = true
		}
		if candidate.token != "" {
			// A denial of one fetch credential says nothing about the trail's lifetime.
			fetchErr = util.RedactFederatedShareError(status)
		}
	}
	if err := removeDeniedRemoteTrailShares(app, trail.Id, denied); err != nil {
		if document != nil {
			document.close()
		}
		return ctx, nil, assetsRequested, util.RedactFederatedShareError(err)
	}
	return chosenCtx, document, assetsRequested, fetchErr
}

// These outcomes permit the next credential, but only an explicit origin
// share denial confirms that a stored grant should be removed.
func remoteTrailShareFallbackStatus(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden || status == http.StatusNotFound || status == http.StatusGone
}

// Re-read the token in the writer: a re-share received during the fetch may
// have replaced it, and must keep its newly granted access.
func removeDeniedRemoteTrailShares(app core.App, trailID string, denied map[string]bool) error {
	if len(denied) == 0 {
		return nil
	}
	return app.RunInTransaction(func(tx core.App) error {
		shares, err := tx.FindRecordsByFilter("trail_share", "trail={:trail}", "", -1, 0, dbx.Params{"trail": trailID})
		if err != nil {
			return err
		}
		for _, share := range shares {
			if denied[share.GetString("remote_token")] {
				if err := tx.Delete(share); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// Later recipients need only an access check; keep the first full document
// and inspect only bounded forbidden responses for an explicit share denial.
func probeRemoteTrailShare(ctx context.Context, client *http.Client, remoteURL *url.URL) (bool, error) {
	release, err := acquireRemoteSyncFetch(ctx, remoteURL.String())
	if err != nil {
		return false, util.RedactFederatedShareError(err)
	}
	defer release()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL.String(), nil)
	if err != nil {
		return false, util.RedactFederatedShareError(err)
	}
	response, err := client.Do(request)
	if err != nil {
		return false, util.RedactFederatedShareError(err)
	}
	defer response.Body.Close()
	return readRemoteTrailShareDenial(response), nil
}

// Only the origin trail access check emits this code. Proxy responses and
// malformed or oversized error documents leave the grant unconfirmed.
func readRemoteTrailShareDenial(response *http.Response) bool {
	if response.StatusCode != http.StatusForbidden {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 4*1024))
	if err != nil || len(body) == 4*1024 {
		return false
	}
	var denial struct {
		Data struct {
			Share struct {
				Code string `json:"code"`
			} `json:"share"`
		} `json:"data"`
	}
	return json.Unmarshal(body, &denial) == nil && denial.Data.Share.Code == "federated_share_denied"
}
