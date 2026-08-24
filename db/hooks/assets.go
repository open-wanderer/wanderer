package hooks

import (
	"encoding/json"
	"pocketbase/util"
	"sync"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

const assetLinkReindexDelay = 500 * time.Millisecond

func InvalidateAssetContentHashOnFileChange() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if e.Record.Original().Id == "" || e.Record.GetString("file") == e.Record.Original().GetString("file") {
			return e.Next()
		}
		metadata, err := assetMetadataMap(e.Record)
		if err != nil {
			return err
		}
		if _, ok := metadata["content_hash"]; !ok {
			return e.Next()
		}
		delete(metadata, "content_hash")
		e.Record.Set("metadata", metadata)
		return e.Next()
	}
}

func assetMetadataMap(record *core.Record) (map[string]any, error) {
	result := map[string]any{}
	rawBytes, err := json.Marshal(record.Get("metadata"))
	if err != nil {
		return nil, err
	}
	if len(rawBytes) == 0 || string(rawBytes) == "null" {
		return result, nil
	}
	if err := json.Unmarshal(rawBytes, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func ReindexTrailOnAssetChange(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		trailIDs, err := util.TrailIDsForAsset(e.App, e.Record.Id)
		if err != nil || len(trailIDs) == 0 {
			return e.Next()
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
		trailID := trailIDForAssetLinkRecord(e.App, e.Record)
		if trailID == "" {
			return e.Next()
		}
		reindexer.Enqueue(e.App, trailID)
		return e.Next()
	}
}

func trailIDForAssetLinkRecord(app core.App, record *core.Record) string {
	trailID := record.GetString("trail")
	if trailID == "" {
		if waypointID := record.GetString("waypoint"); waypointID != "" {
			if waypoint, err := app.FindRecordById("waypoints", waypointID); err == nil {
				trailID = waypoint.GetString("trail")
			}
		}
	}
	if trailID == "" {
		if summitLogID := record.GetString("summit_log"); summitLogID != "" {
			if summitLog, err := app.FindRecordById("summit_logs", summitLogID); err == nil {
				trailID = summitLog.GetString("trail")
			}
		}
	}
	return trailID
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

	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return
	}
	if err := util.IndexTrails(app, []*core.Record{trail}, d.client); err != nil {
		app.Logger().Warn("failed to reindex trail after asset link change", "trail", trailID, "error", err)
	}
}
