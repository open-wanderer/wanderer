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
				errs := e.App.ExpandRecord(item, []string{"category"}, nil)
				if len(errs) > 0 {
					return fmt.Errorf("failed to expand category: %v", errs)
				}
			}

			r.MergeExpand(map[string]any{"item": item})
		}
		return e.Next()
	}
}
