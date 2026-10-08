package routes

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"pocketbase/federation"
	"pocketbase/util"
	"strconv"
	"sync"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func RemoteProfileFollowsList(e *core.RequestEvent) error {
	handle := e.Request.PathValue("handle")
	if handle == "" {
		return e.BadRequestError("Missing required parameter 'handle'", nil)
	}

	followType := e.Request.URL.Query().Get("type")
	if followType != "following" {
		followType = "followers"
	}

	pageQuery := e.Request.URL.Query().Get("page")
	if pageQuery == "" {
		pageQuery = "1"
	}
	page, _ := strconv.Atoi(pageQuery)

	var userActor *core.Record
	if e.Auth != nil {
		userActor, _ = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	}

	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	// 1. Resolve Target Actor
	actor, err := federation.GetActorByHandle(e.App, ctx, handle, false)
	if err != nil {
		return e.NotFoundError("Actor not found", err)
	}

	collectionIRI := actor.GetString(followType)
	if collectionIRI == "" {
		return e.BadRequestError(fmt.Sprintf("Actor has no %s collection", followType), nil)
	}

	// 2. Fetch the requested page, following the collection's own links
	// A cursor names the page directly and costs one remote request
	var collection *pub.OrderedCollectionPage
	if cursor := e.Request.URL.Query().Get("cursor"); cursor != "" {
		collection, err = federation.FetchCollectionCursor(e.App, ctx, collectionIRI, cursor)
	} else {
		collection, err = federation.FetchCollectionPage(e.App, ctx, collectionIRI, page)
	}
	if err != nil {
		if errors.Is(err, federation.ErrInvalidCursor) {
			return e.BadRequestError("Invalid cursor", err)
		}
		if errors.Is(err, util.ErrRateLimited) {
			return e.TooManyRequestsError("Too many requests", err)
		}
		return e.InternalServerError("Failed to fetch remote collection", err)
	}
	items := collection.OrderedItems
	totalItems := collection.TotalItems

	// 5. Resolve IRIs to Local Records
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var mu sync.Mutex
	var wg sync.WaitGroup
	resolvedItems := make([]*core.Record, 0, len(items))

	for _, item := range items {
		iri := item.GetLink().String()
		if iri == "" {
			continue
		}

		wg.Add(1)
		go func(actorIRI string) {
			defer wg.Done()

			// We use a channel to wrap the GetActorByIRI call
			// so we can respect the context timeout
			done := make(chan *core.Record, 1)
			go func() {
				// Pass false to sync to prevent deep recursion/heavy syncing if possible
				res, err := federation.GetActorByIRI(e.App, timeoutCtx, actorIRI, false)
				if err == nil {
					done <- res
				} else {
					done <- nil
				}
			}()

			select {
			case itemActor := <-done:
				if itemActor != nil {
					mu.Lock()
					resolvedItems = append(resolvedItems, itemActor)
					mu.Unlock()
				}
			case <-ctx.Done():
				// Timeout reached for this specific resolution
				return
			}
		}(iri)
	}

	wg.Wait()

	// 6. Pagination Metadata
	perPage := 10
	if len(items) > 0 {
		perPage = len(items)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"page":       page,
		"perPage":    perPage,
		"totalItems": totalItems,
		"totalPages": math.Ceil(float64(totalItems) / float64(perPage)),
		"items":      resolvedItems,
		"next":       federation.CollectionNext(collectionIRI, collection, page),
	})
}
