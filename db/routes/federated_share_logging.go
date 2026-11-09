package routes

import (
	"errors"
	"net/url"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
)

// RedactFederatedShareRequestLogs leaves the credential available to handlers,
// then removes it before PocketBase records the completed request.
func RedactFederatedShareRequestLogs() *hook.Handler[*core.RequestEvent] {
	return &hook.Handler[*core.RequestEvent]{
		Id:       "redactShareRequestLog",
		Priority: apis.DefaultActivityLoggerMiddlewarePriority + 1,
		Func: func(e *core.RequestEvent) error {
			err := e.Next()
			request := e.Request
			shareInURL := request.URL.Query().Has("share")
			referer, parseErr := url.Parse(request.Referer())
			shareInReferer := parseErr == nil && referer.Query().Has("share")
			if !shareInURL && !shareInReferer {
				return err
			}

			// Keep URLs already handed to background work intact. Only the
			// completed request's log view loses its query credential.
			logged := request.Clone(request.Context())
			if shareInURL {
				query := logged.URL.Query()
				query.Del("share")
				logged.URL.RawQuery = query.Encode()
				logged.RequestURI = logged.URL.RequestURI()
			}
			if shareInReferer {
				logged.Header.Set("Referer", util.WithoutFederatedShareToken(request.Referer()))
			}
			e.Request = logged
			// PocketBase records ApiError.RawData separately from Error().
			// Retain the public status/data while redacting its internal cause.
			var apiError *router.ApiError
			if errors.As(err, &apiError) {
				detail := apiError.RawData()
				if cause, ok := detail.(error); ok {
					detail = util.RedactFederatedShareError(cause)
				}
				clean := router.NewApiError(apiError.Status, util.RedactFederatedShareError(errors.New(apiError.Message)).Error(), detail)
				clean.Data = apiError.Data
				return clean
			}
			return util.RedactFederatedShareError(err)
		},
	}
}
