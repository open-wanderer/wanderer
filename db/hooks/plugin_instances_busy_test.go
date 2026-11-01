package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
)

func TestPluginInstanceBusyReadsUseLockAndOnlyExposeItToOwner(t *testing.T) {
	for _, list := range []bool{false, true} {
		for _, owner := range []bool{false, true} {
			collection := core.NewBaseCollection("plugin_instances")
			collection.Fields.Add(&core.TextField{Name: "user"})
			record := core.NewRecord(collection)
			record.Id = t.Name()
			record.Set("user", "owner")
			auth := core.NewRecord(core.NewAuthCollection("users"))
			auth.Id = "other"
			if owner {
				auth.Id = "owner"
			}
			request := &core.RequestEvent{Auth: auth}
			read := func() {
				t.Helper()
				var err error
				if list {
					err = ListPluginInstanceHandler()(&core.RecordsListRequestEvent{RequestEvent: request, Records: []*core.Record{record}})
				} else {
					err = ViewPluginInstanceHandler()(&core.RecordRequestEvent{RequestEvent: request, Record: record})
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			release, ok := pluginsystem.TryLockInstance(record.Id)
			if !ok {
				t.Fatal("could not acquire lock")
			}
			read()
			busy, visible := record.PublicExport()["busy"]
			release()
			if owner && busy != true {
				t.Fatalf("list=%v: owner did not see held lock: %v", list, busy)
			}
			if !owner && visible {
				t.Fatalf("list=%v: another user saw busy", list)
			}
			read()
			if owner && record.PublicExport()["busy"] != false {
				t.Fatal("released lock still showed busy")
			}
		}
	}
}
