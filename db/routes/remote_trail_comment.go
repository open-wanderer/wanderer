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

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func RemoteTrailCommentsList(e *core.RequestEvent) error {
	trailID := e.Request.PathValue("id")
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

	trailAuthor, err := e.App.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return err
	}

	// Sync remote data first (Fetch + Save)
	if trail.GetString("iri") != "" && !trailAuthor.GetBool("is_local") {
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

	reqInfo, err := e.RequestInfo()
	if err != nil {
		return err
	}

	filteredRecords := []*core.Record{}
	for _, record := range records {
		canAccess, _ := e.App.CanAccessRecord(record, reqInfo, record.Collection().ListRule)
		if canAccess {
			filteredRecords = append(filteredRecords, record)
		}
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

	// 4. Apply view rules to every requested relation, including nested expands.
	if err := enrichRemoteRecords(e, filteredRecords...); err != nil {
		return err
	}

	// 5. Manually construct the response object
	return e.JSON(http.StatusOK, map[string]any{
		"page":       page,
		"perPage":    perPage,
		"totalItems": totalItems,
		"totalPages": (totalItems + perPage - 1) / perPage,
		"items":      filteredRecords,
	})
}

func syncRemoteComments(e *core.RequestEvent, trail *core.Record) error {
	client := newRemoteSyncHTTPClient()

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
				if remoteID == "" {
					continue
				}
				remoteIRI = fmt.Sprintf("%s://%s/api/v1/comment/%s", u.Scheme, u.Host, remoteID)
			}

			skip := func(err error) {
				txApp.Logger().Warn("skipping pulled object", "collection", "comments", "iri", remoteIRI, "trail", trailIRI, "error", err)
			}

			authorIRI := ""
			if expand, ok := raw["expand"].(map[string]any); ok {
				if author, ok := expand["author"].(map[string]any); ok {
					authorIRI, _ = author["iri"].(string)
				}
			}

			// Only accept objects from the trail's origin, before fetching or writing.
			commentRecord, err := federation.CheckPulledObject(txApp, "comments", remoteIRI, authorIRI, trail)
			if err != nil {
				skip(err)
				continue
			}
			if authorIRI == "" {
				skip(errors.New("comment has no author"))
				continue
			}

			actor, err := federation.GetActorByIRI(txApp, ctx, authorIRI, false)
			if err != nil {
				skip(err)
				continue
			}
			if _, err := federation.CheckRemoteObjectOwnership(txApp, "comments", remoteIRI, actor); err != nil {
				skip(err)
				continue
			}

			if commentRecord == nil {
				commentRecord = core.NewRecord(collection)
				commentRecord.Set("iri", remoteIRI)
				commentRecord.Set("trail", trail.Id)
			}
			raw["author"] = actor.Id

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
