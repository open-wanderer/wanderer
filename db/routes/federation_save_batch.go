package routes

import (
	"net/http"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func FederationSaveBatch(begin, complete func(*core.Record, string) bool) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return apis.NewUnauthorizedError("authentication required", nil)
		}
		collection := map[string]string{"trail": "trails", "summit-log": "summit_logs"}[e.Request.PathValue("kind")]
		action := e.Request.PathValue("action")
		if collection == "" || (action != "begin" && action != "complete") {
			return apis.NewBadRequestError("invalid federation save action", nil)
		}
		var data struct {
			BatchID string `json:"batchId"`
		}
		if err := e.BindBody(&data); err != nil || !util.ValidFederationSaveBatchID(data.BatchID) {
			return apis.NewBadRequestError("invalid federation save batch", err)
		}
		record, err := e.App.FindRecordById(collection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("record not found", err)
		}
		info, err := e.RequestInfo()
		if err != nil {
			return err
		}
		info = info.Clone()
		info.Method = http.MethodPatch
		info.Body = map[string]any{}
		allowed, err := e.App.CanAccessRecord(record, info, record.Collection().UpdateRule)
		if err != nil || !allowed {
			return apis.NewForbiddenError("insufficient edit permission", err)
		}
		author, err := e.App.FindRecordById("activitypub_actors", record.GetString("author"))
		if err != nil || !author.GetBool("is_local") {
			return apis.NewForbiddenError("record is not locally authored", err)
		}
		if collection == "summit_logs" && record.GetBool("federation_suppressed") {
			return e.JSON(http.StatusOK, map[string]string{"status": "not_federated"})
		}
		// Editable copies with an unrelated object identity do not announce
		// through this instance.
		if iri := record.GetString("iri"); iri != "" && !util.RecordHasLocalIdentity(e.App, record) {
			return e.JSON(http.StatusOK, map[string]string{"status": "not_federated"})
		}
		if action == "begin" {
			if !begin(record, data.BatchID) {
				return apis.NewTooManyRequestsError("too many active federation saves", nil)
			}
		} else {
			complete(record, data.BatchID)
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}
