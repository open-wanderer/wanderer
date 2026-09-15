package routes

import (
	"errors"
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// enrichRemoteRecords rebuilds response expansions using the caller's view
// rules, including auth and share-token query parameters. Remote payloads and
// indexing hooks may have populated unfiltered expansions already. PocketBase
// retains those when no allowed relation is found, so clear them even when the
// caller did not request an expand.
func enrichRemoteRecords(e *core.RequestEvent, records ...*core.Record) error {
	for _, record := range records {
		record.SetExpand(nil)
	}

	if err := apis.EnrichRecords(e, records); err != nil {
		// PocketBase logs ordinary expansion failures itself. Request parsing
		// and enrichment hooks can still return errors.
		var apiErr *router.ApiError
		if errors.As(err, &apiErr) {
			return apiErr
		}
		return e.InternalServerError("Failed to enrich record", err)
	}
	return nil
}

func expandAndReturn(e *core.RequestEvent, record *core.Record) error {
	if err := enrichRemoteRecords(e, record); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, record)
}
