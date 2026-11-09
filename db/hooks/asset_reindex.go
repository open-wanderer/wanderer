package hooks

import (
	"fmt"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
	"reflect"
	"sync"
	"time"
)

const assetLinkReindexDelay = 500 * time.Millisecond

func ReindexTrailOnAssetChange(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if e.Type == core.ModelEventTypeUpdate {
			changed := false
			for _, field := range []string{"type", "file", "storage_mode"} {
				if !reflect.DeepEqual(e.Record.Get(field), e.Record.Original().Get(field)) {
					changed = true
					break
				}
			}
			if !changed && util.IsGeneratedRoutePreviewAsset(e.Record) == util.IsGeneratedRoutePreviewAsset(e.Record.Original()) {
				return e.Next()
			}
		}
		// Search thumbnails use direct trail photos, never child galleries.
		links, err := e.App.FindRecordsByFilter("trail_assets", "asset={:asset}", "", -1, 0, map[string]any{"asset": e.Record.Id})
		if err != nil || len(links) == 0 {
			return e.Next()
		}
		trailIDs := []string{}
		seen := map[string]bool{}
		for _, link := range links {
			if id := link.GetString("trail"); id != "" && !seen[id] {
				seen[id] = true
				trailIDs = append(trailIDs, id)
			}
		}
		trails := make([]*core.Record, 0, len(trailIDs))
		for _, trailID := range trailIDs {
			trail, err := e.App.FindRecordById("trails", trailID)
			if err == nil {
				trails = append(trails, trail)
			}
		}
		if len(trails) > 0 {
			if err := util.IndexTrails(e.App, trails, client); err != nil {
				e.App.Logger().Warn("failed to reindex trails after asset change", "error", err)
			}
		}
		return e.Next()
	}
}

func ReindexTrailOnAssetLinkChange(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	reindexer := newTrailReindexDebouncer(client, assetLinkReindexDelay)
	return func(e *core.RecordEvent) error {
		if e.Record.Collection().Name != "trail_assets" {
			return e.Next()
		}
		trailID := e.Record.GetString("trail")
		if trailID != "" {
			reindexer.Enqueue(e.App, trailID)
		}
		if e.Type == core.ModelEventTypeUpdate {
			if previous := e.Record.Original().GetString("trail"); previous != "" && previous != trailID {
				reindexer.Enqueue(e.App, previous)
			}
		}
		return e.Next()
	}
}

type trailReindexDebouncer struct {
	client meilisearch.ServiceManager
	delay  time.Duration
	mu     sync.Mutex
	timers map[string]*time.Timer
}

func newTrailReindexDebouncer(client meilisearch.ServiceManager, delay time.Duration) *trailReindexDebouncer {
	return &trailReindexDebouncer{
		client: client,
		delay:  delay,
		timers: map[string]*time.Timer{},
	}
}

func (d *trailReindexDebouncer) Enqueue(app core.App, trailID string) {
	if trailID == "" {
		return
	}

	d.mu.Lock()
	if timer, ok := d.timers[trailID]; ok {
		timer.Stop()
	}
	d.timers[trailID] = time.AfterFunc(d.delay, func() {
		d.flush(app, trailID)
	})
	d.mu.Unlock()
}

func (d *trailReindexDebouncer) flush(app core.App, trailID string) {
	d.mu.Lock()
	delete(d.timers, trailID)
	d.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			app.Logger().Error(fmt.Sprintf("Recovered from panic while reindexing trail %s: %v", trailID, r))
		}
	}()

	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return
	}
	if err := util.IndexTrails(app, []*core.Record{trail}, d.client); err != nil {
		app.Logger().Warn("failed to reindex trail after asset link change", "trail", trailID, "error", err)
	}
}
