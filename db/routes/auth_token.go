package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func AuthToken(e *core.RequestEvent) error {
	var data struct {
		APIToken string `json:"api_token"`
	}
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}

	hashedAPIToken := security.SHA256(data.APIToken)

	tokenRecord, err := e.App.FindFirstRecordByFilter(
		"api_tokens",
		"token = {:hash}",
		map[string]any{"hash": hashedAPIToken},
	)

	if errors.Is(err, sql.ErrNoRows) {
		return apis.NewUnauthorizedError("Invalid or revoked API token", nil)
	}
	if err != nil {
		// A failed lookup says nothing about the token; a 401 would make
		// clients discard a token that is still valid.
		return apis.NewInternalServerError("Failed to look up API token", err)
	}
	if !tokenRecord.GetDateTime("expiration").IsZero() &&
		tokenRecord.GetDateTime("expiration").Time().Before(time.Now()) {
		return apis.NewUnauthorizedError("Key has expired", nil)
	}

	tokenRecord.Set("last_used", time.Now())
	if err := e.App.Save(tokenRecord); err != nil {
		return err
	}

	userRecord, _ := e.App.FindRecordById("users", tokenRecord.GetString("user"))
	token, err := userRecord.NewAuthToken()
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]any{
		"token":  token,
		"record": userRecord,
	})
}
