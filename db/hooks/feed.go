package hooks

import (
	"fmt"
	"pocketbase/util"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func ListFeedHandler() func(e *core.RecordsListRequestEvent) error {
	return func(e *core.RecordsListRequestEvent) error {
		requestInfo, err := e.RequestInfo()
		if err != nil {
			return err
		}
		fetchReadable := func(collection *core.Collection, ids []string) ([]*core.Record, error) {
			records, err := e.App.FindRecordsByIds(collection, ids)
			if err != nil {
				return nil, err
			}
			readable := make([]*core.Record, 0, len(records))
			for _, record := range records {
				if allowed, err := e.App.CanAccessRecord(record, requestInfo, collection.ViewRule); err == nil && allowed {
					readable = append(readable, record)
				}
			}
			return readable, nil
		}

		for _, r := range e.Records {
			expand := r.Expand()
			delete(expand, "item")
			r.SetExpand(expand)
			var item *core.Record
			var err error

			typ := r.GetString("type")
			typ = strings.Trim(typ, "\"")

			itemId := r.GetString("item")
			itemId = strings.Trim(itemId, "\"")

			switch typ {
			case string(util.TrailFeed):
				item, err = e.App.FindRecordById("trails", itemId)
			case string(util.ListFeed):
				item, err = e.App.FindRecordById("lists", itemId)
			case string(util.SummitLogFeed):
				item, err = e.App.FindRecordById("summit_logs", itemId)
			}

			if err != nil {
				continue
			}

			if item == nil {
				continue
			}
			if allowed, err := e.App.CanAccessRecord(item, requestInfo, item.Collection().ViewRule); err != nil || !allowed {
				continue
			}

			errs := e.App.ExpandRecord(item, []string{"author"}, fetchReadable)
			if len(errs) > 0 {
				return fmt.Errorf("failed to expand author: %v", errs)
			}

			if typ == string(util.TrailFeed) {
				errs := e.App.ExpandRecord(item, []string{"category"}, fetchReadable)
				if len(errs) > 0 {
					return fmt.Errorf("failed to expand trail feed item: %v", errs)
				}
			}

			if typ == string(util.SummitLogFeed) {
				errs := e.App.ExpandRecord(item, []string{"trail"}, fetchReadable)
				if len(errs) > 0 {
					return fmt.Errorf("failed to expand summit log feed item: %v", errs)
				}
			}

			r.MergeExpand(map[string]any{"item": item})
		}
		return e.Next()
	}
}
