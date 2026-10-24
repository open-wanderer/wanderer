package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
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

var newRemoteCommentSyncHTTPClient = util.SafeHTTPClient

func syncRemoteComments(e *core.RequestEvent, trail *core.Record) error {
	if util.IsLocalIRI(trail.GetString("iri")) || remoteRecordHasLocalAuthor(e.App, trail) {
		return nil
	}
	client := newRemoteCommentSyncHTTPClient()

	var userActor *core.Record
	if e.Auth != nil {
		userActor, _ = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	}

	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	trailIRI := trail.GetString("iri")
	u, err := url.Parse(trailIRI)
	if err != nil || !remoteSyncIRIMatchesOrigin(trailIRI, trailIRI) {
		return fmt.Errorf("invalid remote trail origin")
	}

	remoteTrailID := path.Base(u.Path)

	remoteURL := fmt.Sprintf("%s://%s/api/v1/comment?filter=trail='%s'&expand=author", u.Scheme, u.Host, remoteTrailID)

	req, _ := http.NewRequestWithContext(ctx, "GET", remoteURL, nil)
	res, err := client.Do(req)
	if err != nil {
		if errors.Is(err, util.ErrRateLimited) {
			return e.TooManyRequestsError("Too many requests", err)
		}
		return fmt.Errorf("remote fetch failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("remote fetch returned status %d", res.StatusCode)
	}

	var remoteData struct {
		Items      []map[string]any `json:"items"`
		TotalItems *int             `json:"totalItems"`
		TotalPages *int             `json:"totalPages"`
	}
	if err := util.DecodeFederationJSON(res.Body, util.FederationObjectBodyLimit, &remoteData); err != nil {
		return err
	}

	collection, err := e.App.FindCollectionByNameOrId("comments")
	if err != nil {
		return err
	}
	type preparedComment struct {
		iri, author string
		data        map[string]any
		canonical   bool
	}
	prepared := []preparedComment{}
	complete := remoteData.Items != nil
	if remoteData.TotalItems != nil || remoteData.TotalPages != nil {
		complete = complete && (remoteData.TotalItems == nil || *remoteData.TotalItems == len(remoteData.Items)) &&
			(remoteData.TotalPages == nil || *remoteData.TotalPages == 1 || *remoteData.TotalPages == 0 && len(remoteData.Items) == 0)
	} else {
		complete = complete && len(remoteData.Items) < canonicalRemoteRecordLimit
	}
	listed := map[string]bool{}
	references := []canonicalCommentReference{}
	referenced := map[string]struct{}{}
	// Actor refreshes run before the transaction; no provider lookup may hold
	// the write lock needed to save a comment.
	for _, raw := range remoteData.Items {
		iri, _ := raw["iri"].(string)
		if iri == "" {
			id, _ := raw["id"].(string)
			if id == "" {
				complete = false
				continue
			}
			iri = fmt.Sprintf("%s://%s/api/v1/comment/%s", u.Scheme, u.Host, id)
		}
		if !remoteSyncIRIMatchesOrigin(iri, iri) {
			complete = false
			continue
		}
		listed[iri] = true
		if util.IsLocalIRI(iri) {
			continue
		}
		if !remoteSyncIRIMatchesOrigin(iri, trailIRI) {
			if _, duplicate := referenced[iri]; duplicate {
				continue
			}
			referenced[iri] = struct{}{}
			references = append(references, canonicalCommentReference{iri: iri, updated: canonicalLogUpdatedHint(raw)})
			prepared = append(prepared, preparedComment{iri: iri, canonical: true})
			continue
		}
		author, allowed, authorErr := resolveRemoteSyncAuthor(e.App, ctx, trail.GetString("author"), iri, raw)
		if authorErr != nil || !allowed {
			continue
		}
		actor, actorErr := e.App.FindRecordById("activitypub_actors", author)
		if actorErr != nil || actor.GetBool("is_local") {
			continue
		}
		for _, key := range []string{"id", "trail", "author", "expand", "iri"} {
			delete(raw, key)
		}
		stripLocalSyncFields(raw)
		prepared = append(prepared, preparedComment{iri: iri, author: author, data: raw})
	}
	if complete {
		for _, reference := range omittedRemoteCommentReferences(e.App, trail, listed) {
			references = append(references, reference)
			prepared = append(prepared, preparedComment{iri: reference.iri, canonical: true})
		}
	}
	canonical := fetchCanonicalRemoteComments(e.App, ctx, client, trail, references)
	return e.App.RunInTransaction(func(txApp core.App) error {
		for _, item := range prepared {
			var trusted canonicalRemoteComment
			if item.canonical {
				var ok bool
				if trusted, ok = canonical[item.iri]; !ok {
					continue
				}
				item.author, item.data = trusted.author, trusted.data
			}
			comment, err := txApp.FindFirstRecordByData("comments", "iri", item.iri)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if item.canonical && trusted.removed {
				if err := removeConfirmedGoneComment(txApp, trail, comment, trusted); err != nil {
					return err
				}
				continue
			}
			if comment != nil {
				if !remoteSyncRecordCanUpdate(txApp, comment, item.author, trail.Id) {
					if item.canonical {
						canonicalRemoteLogCache.failure(trusted.cacheKey, trusted.updatedHint)
					}
					continue
				}
			} else {
				comment = core.NewRecord(collection)
			}
			loadRemoteRecordFields(comment, item.data)
			comment.Set("iri", item.iri)
			comment.Set("trail", trail.Id)
			comment.Set("author", item.author)
			util.SanitizeHTMLFieldsWithLimits(comment)
			if err := txApp.Save(comment); err != nil {
				txApp.Logger().Warn("skipped remote comment that failed validation", "iri", item.iri, "error", err)
				if item.canonical {
					canonicalRemoteLogCache.failure(trusted.cacheKey, trusted.updatedHint)
				}
				continue
			}
			if item.canonical {
				canonicalRemoteLogCache.successAfterCommit(txApp, trusted.cacheKey, trusted.updatedHint, comment.GetString("updated"))
			}
		}
		return nil
	})
}
