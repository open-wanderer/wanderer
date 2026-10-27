package routes

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"pocketbase/util"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Stage the bounded response on disk. Neither queue waiters nor the optional
// author and media phase retain a decoded copy of the complete document.
type remoteSyncDocument struct {
	file    *os.File
	tokens  int
	counted bool
}

func fetchRemoteSyncDocument(ctx context.Context, client *http.Client, remoteURL *url.URL, kind string) (*remoteSyncDocument, error) {
	releaseFetch, err := acquireRemoteSyncFetch(ctx, remoteURL.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	defer releaseFetch()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, remoteURL.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err := &remoteFetchStatusError{Kind: kind, URL: remoteURL.String(), StatusCode: response.StatusCode}
		if response.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
		}
		if isGoneStatus(response.StatusCode) {
			return nil, fmt.Errorf("%w: %w", errRemoteGone, err)
		}
		return nil, err
	}
	file, err := os.CreateTemp("", "wanderer-sync-document-*")
	if err != nil {
		return nil, err
	}
	document := &remoteSyncDocument{file: file}
	count, err := io.Copy(file, io.LimitReader(response.Body, util.FederationSyncBodyLimit+1))
	if err == nil && count > util.FederationSyncBodyLimit {
		err = fmt.Errorf("%w: %w", errRemoteUnavailable, util.ErrFederationBodyTooLarge)
	}
	if err != nil {
		document.close()
		return nil, err
	}
	return document, nil
}

func (document *remoteSyncDocument) close() {
	name := document.file.Name()
	_ = document.file.Close()
	_ = os.Remove(name)
}

func (document *remoteSyncDocument) process(ctx context.Context, run func(map[string]any) (*core.Record, error)) (*core.Record, error) {
	if !document.counted {
		if _, err := document.file.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		tokens, err := remoteSyncJSONTokens(ctx, document.file)
		if err != nil {
			if isRemoteDocumentLimitError(err) {
				err = fmt.Errorf("%w: %w", errRemoteUnavailable, err)
			}
			return nil, err
		}
		document.tokens, document.counted = tokens, true
	}
	release, err := acquireRemoteSyncProcessing(ctx, document.tokens)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	defer release()
	if _, err := document.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	var data map[string]any
	if err := decodeRemoteSyncJSON(ctx, document.file, util.FederationSyncBodyLimit, &data); err != nil {
		if errors.Is(err, util.ErrFederationBodyTooLarge) || errors.Is(err, errRemoteSyncJSONComplexity) {
			err = fmt.Errorf("%w: %w", errRemoteUnavailable, err)
		}
		return nil, err
	}
	return run(data)
}

// Prepare only identity fields before the existing author and object checks.
// Enforce the collection id contract before deriving repeated network URLs.
func (document *remoteSyncDocument) identity(app core.App, ctx context.Context, record *core.Record) (map[string]any, error) {
	var identity map[string]any
	_, err := document.process(ctx, func(data map[string]any) (*core.Record, error) {
		id, _ := data["id"].(string)
		if strings.TrimSpace(id) == "" || !remoteSyncIDFitsCollection(app, record.Collection().Name, id) {
			return nil, fmt.Errorf("%w: remote response record id does not match the collection field", errRemoteUnavailable)
		}
		identity = remoteSyncNetworkRecord(data)
		return nil, nil
	})
	return identity, err
}

type remoteSyncNetworkPlan struct {
	data  map[string]any
	media remoteMediaCollector
}

func (document *remoteSyncDocument) networkPlan(app core.App, ctx context.Context, record *core.Record, origin, kind string) (remoteSyncNetworkPlan, error) {
	plan := remoteSyncNetworkPlan{}
	_, err := document.process(ctx, func(data map[string]any) (*core.Record, error) {
		id, _ := data["id"].(string)
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("remote %s response is missing its record id", kind)
		}
		if kind == "list" {
			capRemoteSyncChildren(app, data, origin, "trails")
			plan.media = remoteMediaCollector{seen: map[string]bool{}}
			plan.media.record(app, "list", id, origin, record.GetString("iri"), data)
		} else {
			capRemoteSyncChildren(app, data, origin, "waypoints_via_trail", "summit_logs_via_trail")
			plan.media = collectRemoteTrailMedia(app, record, origin, id, data)
		}
		plan.data = remoteSyncNetworkRecord(data)
		if public, ok := data["public"].(bool); ok && kind == "trail" {
			plan.data["public"] = public
		}
		expand, _ := data["expand"].(map[string]any)
		plannedExpand, _ := plan.data["expand"].(map[string]any)
		relation := "summit_logs_via_trail"
		if kind == "list" {
			relation = "trails"
		}
		if children, ok := expand[relation].([]any); ok {
			planned := make([]any, 0, len(children))
			for _, value := range children {
				if raw, ok := value.(map[string]any); ok {
					child := remoteSyncNetworkRecord(raw)
					iri, _ := raw["iri"].(string)
					if iri != "" && !remoteSyncIRIMatchesOrigin(iri, origin) {
						delete(child, "id")
					}
					planned = append(planned, child)
				}
			}
			plannedExpand[relation] = planned
		}
		return nil, nil
	})
	return plan, err
}

// Only scalar identities are needed during network preparation. File requests
// have already been reduced to the existing per-run request limit.
func remoteSyncNetworkRecord(data map[string]any) map[string]any {
	result := canonicalScalarFields(data, "id", "iri", "updated")
	if _, present := data["iri"]; present {
		if _, ok := data["iri"].(string); !ok {
			result["iri"] = nil
		}
	}
	expand := map[string]any{}
	result["expand"] = expand
	if raw, ok := data["expand"].(map[string]any); ok {
		if value, present := raw["author"]; present {
			expand["author"] = nil
			if author, ok := value.(map[string]any); ok {
				expand["author"] = canonicalScalarFields(author, "iri")
			}
		}
	}
	return result
}
