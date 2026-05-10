package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"pocketbase/federation"
	"pocketbase/util"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func RemoteTrailCommentsList(e *core.RequestEvent) error {
	trailID := e.Request.PathValue("id")
	expandQuery := e.Request.URL.Query().Get("expand")
	sort := e.Request.URL.Query().Get("sort")

	if sort == "" {
		sort = "-created"
	}

	page, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(e.Request.URL.Query().Get("perPage"))
	if perPage < 1 {
		perPage = 30
	}

	trail, err := e.App.FindRecordById("trails", trailID)
	if err != nil {
		return err
	}

	// Sync remote data first (Fetch + Save)
	if trail.GetString("iri") != "" {
		_ = syncRemoteComments(e, trail)
	}

	// 1. Calculate Offset
	offset := (page - 1) * perPage

	// 2. Fetch the records using FindRecordsByFilter
	records, err := e.App.FindRecordsByFilter(
		"comments",
		"trail = {:trailId}",
		sort,
		perPage,
		offset,
		dbx.Params{"trailId": trail.Id},
	)
	if err != nil {
		return err
	}

	// 3. Get total count for pagination metadata
	var totalItems int
	err = e.App.DB().
		Select("count(*)").
		From("comments").
		Where(dbx.HashExp{"trail": trail.Id}).
		Row(&totalItems)
	if err != nil {
		return err
	}

	// 4. Handle Expand
	if expandQuery != "" {
		errs := e.App.ExpandRecords(records, strings.Split(expandQuery, ","), nil)
		if len(errs) > 0 {
			fmt.Printf("Expand errors: %v\n", errs)
		}
	}

	// 5. Manually construct the response object
	return e.JSON(http.StatusOK, map[string]any{
		"page":       page,
		"perPage":    perPage,
		"totalItems": totalItems,
		"totalPages": (totalItems + perPage - 1) / perPage,
		"items":      records,
	})
}

func syncRemoteComments(e *core.RequestEvent, trail *core.Record) error {
	client := util.SafeHTTPClient()

	var userActor *core.Record
	if e.Auth != nil {
		userActor, _ = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	}

	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	trailIRI := trail.GetString("iri")
	u, _ := url.Parse(trailIRI)

	remoteTrailID := path.Base(u.Path)

	remoteURL := fmt.Sprintf("%s://%s/api/v1/comment?filter=trail='%s'&expand=author", u.Scheme, u.Host, remoteTrailID)

	req, _ := http.NewRequestWithContext(ctx, "GET", remoteURL, nil)
	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		if errors.Is(err, util.ErrRateLimited) {
			return e.TooManyRequestsError("Too many requests", err)
		}
		return fmt.Errorf("remote fetch failed: %w", err)
	}
	defer res.Body.Close()

	var remoteData struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&remoteData); err != nil {
		return err
	}

	collection, _ := e.App.FindCollectionByNameOrId("comments")

	return e.App.RunInTransaction(func(txApp core.App) error {
		for _, raw := range remoteData.Items {
			remoteIRI, _ := raw["iri"].(string)
			if remoteIRI == "" {
				remoteID, _ := raw["id"].(string)
				remoteIRI = fmt.Sprintf("%s://%s/api/v1/comment/%s", u.Scheme, u.Host, remoteID)
			}

			remoteCommentUrl, _ := url.Parse(remoteIRI)
			possibleLocalId := path.Base(remoteCommentUrl.Path)

			// Find existing record by IRI or ID to avoid duplicates
			commentRecord, _ := txApp.FindFirstRecordByFilter("comments", "iri={:iri} || id={:id}", dbx.Params{"id": possibleLocalId, "iri": remoteIRI})
			if commentRecord == nil {
				commentRecord = core.NewRecord(collection)
				commentRecord.Set("iri", remoteIRI)
				commentRecord.Set("trail", trail.Id)
			}

			// Resolve federated author
			if expand, ok := raw["expand"].(map[string]any); ok {
				if author, ok := expand["author"].(map[string]any); ok {
					authorIRI, _ := author["iri"].(string)
					actor, err := federation.GetActorByIRI(txApp, ctx, authorIRI, false)
					if err == nil {
						raw["author"] = actor.Id
					}
				}
			}

			delete(raw, "id")
			delete(raw, "trail")
			delete(raw, "expand")
			delete(raw, "iri")
			commentRecord.Load(raw)

			if err := txApp.Save(commentRecord); err != nil {
				continue
			}
		}
		return nil
	})
}

func expandAndReturnList(e *core.RequestEvent, records []*core.Record, query string) error {
	if query != "" {
		expandPaths := strings.Split(query, ",")

		errs := e.App.ExpandRecords(records, expandPaths, nil)
		if len(errs) > 0 {
			fmt.Printf("Expand errors: %v\n", errs)
		}
	}

	return e.JSON(http.StatusOK, records)
}
