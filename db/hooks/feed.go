package hooks

import (
	"fmt"
	"pocketbase/util"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func ListFeedHandler() func(e *core.RecordsListRequestEvent) error {
	return func(e *core.RecordsListRequestEvent) error {

		for _, r := range e.Records {
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

			errs := e.App.ExpandRecord(item, []string{"author"}, nil)
			if len(errs) > 0 {
				return fmt.Errorf("failed to expand author: %v", errs)
			}

			if typ == string(util.TrailFeed) {
				errs := e.App.ExpandRecord(item, []string{"category", "trail_assets_via_trail.asset"}, nil)
				if len(errs) > 0 {
					return fmt.Errorf("failed to expand trail feed item: %v", errs)
				}
			}

			if typ == string(util.SummitLogFeed) {
				errs := e.App.ExpandRecord(item, []string{"trail", "summit_log_assets_via_summit_log.asset"}, nil)
				if len(errs) > 0 {
					return fmt.Errorf("failed to expand summit log feed item: %v", errs)
				}
			}

			r.MergeExpand(map[string]any{"item": item})
		}
		return e.Next()
	}
}
