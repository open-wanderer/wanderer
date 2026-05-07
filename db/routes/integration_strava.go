package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"pocketbase/integrations/strava"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func IntegrationStravaToken(e *core.RequestEvent) error {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
	}

	var data strava.TokenRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}

	userId := ""
	if e.Auth != nil {
		userId = e.Auth.Id
	} else {
		return e.UnauthorizedError("authentication required", nil)
	}

	integrations, err := e.App.FindAllRecords("integrations", dbx.NewExp("user = {:id}", dbx.Params{"id": userId}))
	if err != nil {
		return err
	}
	if len(integrations) == 0 {
		return apis.NewBadRequestError("user has no integration", nil)
	}
	integration := integrations[0]
	stravaString := integration.GetString("strava")
	if len(stravaString) == 0 {
		return apis.NewBadRequestError("strava integration missing", nil)
	}
	var stravaIntegration strava.StravaIntegration
	err = json.Unmarshal([]byte(stravaString), &stravaIntegration)
	if err != nil {
		return err
	}
	decryptedSecret, err := security.Decrypt(stravaIntegration.ClientSecret, encryptionKey)
	if err != nil {
		return err
	}

	request := strava.TokenRequest{
		ClientID:     stravaIntegration.ClientID,
		ClientSecret: string(decryptedSecret),
		Code:         data.Code,
		GrantType:    "authorization_code",
	}
	r, err := strava.GetStravaToken(request)
	if err != nil {
		return err
	}
	if r.AccessToken != "" {
		stravaIntegration.AccessToken = r.AccessToken
	}
	if r.RefreshToken != "" {
		stravaIntegration.RefreshToken = r.RefreshToken
	}
	if r.AccessToken != "" {
		stravaIntegration.ExpiresAt = r.ExpiresAt
	}

	stravaIntegration.Active = true

	b, err := json.Marshal(stravaIntegration)
	if err != nil {
		return err
	}
	integration.Set("strava", string(b))
	err = e.App.Save(integration)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, nil)
}
