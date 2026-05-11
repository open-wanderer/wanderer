package routes

import (
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

	if err != nil {
		return apis.NewNotFoundError("Invalid or revoked API token", nil)
	}
	if !tokenRecord.GetDateTime("expiration").IsZero() &&
		tokenRecord.GetDateTime("expiration").Time().Before(time.Now()) {
		return apis.NewBadRequestError("Key has expired", nil)
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
