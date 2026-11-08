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

		items := map[string]*core.Record{}
		targetIDs := map[string][]string{}
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

			items[r.Id] = item
			if typ == string(util.TrailFeed) {
				targetIDs["trail"] = append(targetIDs["trail"], item.Id)
			}
			if typ == string(util.SummitLogFeed) {
				targetIDs["summit_log"] = append(targetIDs["summit_log"], item.Id)
			}
		}
		previews := map[string]map[string][]*core.Record{}
		for _, field := range []string{"trail", "summit_log"} {
			rows, err := util.CardPhotoLinks(e.App, requestInfo, field, targetIDs[field])
			if err != nil {
				return err
			}
			previews[field] = rows
		}
		for _, r := range e.Records {
			item := items[r.Id]
			if item == nil {
				continue
			}
			typ := strings.Trim(r.GetString("type"), "\"")

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

			field := ""
			if typ == string(util.TrailFeed) {
				field = "trail"
			}
			if typ == string(util.SummitLogFeed) {
				field = "summit_log"
			}
			if field != "" {
				rows := previews[field][item.Id]
				if rows == nil {
					rows = []*core.Record{}
				}
				item.MergeExpand(map[string]any{field + "_assets_via_" + field: rows})
			}

			r.MergeExpand(map[string]any{"item": item})
		}
		return e.Next()
	}
}
