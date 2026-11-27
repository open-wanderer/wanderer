package hooks

import (
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const assetMetadataOwnerMemoKey = "__assetMetadataOwnerMemo"

type assetMetadataOwnerMemo struct {
	user, database string
	once           sync.Once
	actors         map[string]bool
}

var assetMetadataOwnerMemoLock sync.Mutex

// PocketBase loads a trusted auth record for each request. Its enrichment
// contexts share that record and this private typed memo. JSON input cannot
// construct the memo type.
func assetAuthorOwnedForRequest(app core.App, info *core.RequestInfo, author string) bool {
	if info == nil || info.Auth == nil || info.Auth.Id == "" {
		return false
	}
	assetMetadataOwnerMemoLock.Lock()
	memo, _ := info.Auth.Get(assetMetadataOwnerMemoKey).(*assetMetadataOwnerMemo)
	if memo == nil || memo.user != info.Auth.Id || memo.database != app.DataDir() {
		memo = &assetMetadataOwnerMemo{user: info.Auth.Id, database: app.DataDir()}
		info.Auth.Set(assetMetadataOwnerMemoKey, memo)
		info.Auth.Hide(assetMetadataOwnerMemoKey)
	}
	assetMetadataOwnerMemoLock.Unlock()
	memo.once.Do(func() {
		memo.actors = map[string]bool{}
		actors, err := app.FindRecordsByFilter("activitypub_actors", "user={:user}", "", -1, 0, dbx.Params{"user": memo.user})
		if err != nil {
			return
		}
		for _, actor := range actors {
			memo.actors[actor.Id] = true
		}
	})
	return memo.actors[author]
}
