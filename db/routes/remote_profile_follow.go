package routes

import (
	"context"
	"errors"
	"fmt"
	"io"
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

	// 2. Fetch Remote Content
	client := util.SafeHTTPClient()
	req, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s?page=%d", collectionIRI, page), nil)
	req.Header.Set("Accept", "application/activity+json")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if errors.Is(err, util.ErrRateLimited) {
			return e.TooManyRequestsError("Too many requests", err)
		}
		return e.InternalServerError("Failed to fetch remote collection", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return e.InternalServerError("Failed to read response body", err)
	}

	// 3. Proper Unmarshaling using go-ap
	// This returns a pub.Item interface which could be an OrderedCollection,
	// OrderedCollectionPage, or even a simple Object.
	data, err := pub.UnmarshalJSON(body)
	if err != nil {
		return e.InternalServerError("Failed to unmarshal ActivityPub JSON", err)
	}

	var items pub.ItemCollection
	var totalItems uint = 0

	// 4. Type assertion using go-ap's type switch pattern
	err = pub.OnOrderedCollectionPage(data, func(p *pub.OrderedCollectionPage) error {
		items = p.OrderedItems
		totalItems = p.TotalItems
		return nil
	})

	// Fallback: some instances might return a plain OrderedCollection
	// if the page isn't strictly formatted as a Page object
	if err != nil || items == nil {
		_ = pub.OnOrderedCollection(data, func(c *pub.OrderedCollection) error {
			items = c.OrderedItems
			totalItems = c.TotalItems
			return nil
		})
	}

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
	})
}
