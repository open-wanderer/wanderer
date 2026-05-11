package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"pocketbase/integrations/komoot"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func IntegrationKommotLogin(e *core.RequestEvent) error {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
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
	komootString := integration.GetString("komoot")
	if len(komootString) == 0 {
		return apis.NewBadRequestError("komoot integration missing", nil)
	}
	var komootIntegration komoot.KomootIntegration
	err = json.Unmarshal([]byte(komootString), &komootIntegration)
	if err != nil {
		return err
	}
	decryptedPassword, err := security.Decrypt(komootIntegration.Password, encryptionKey)
	if err != nil {
		return err
	}

	k := &komoot.KomootApi{}

	err = k.Login(komootIntegration.Email, string(decryptedPassword))
	if err != nil {
		return apis.NewUnauthorizedError("invalid credentials", nil)
	}

	return e.JSON(http.StatusOK, nil)
}
