package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"pocketbase/integrations/hammerhead"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func IntegrationHammerheadUpload(e *core.RequestEvent) error {
	h, err := loginHammerhead(e)
	if err != nil {
		return err
	}

	if err := h.UploadActivities(e); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, nil)
}

func IntegrationHammerheadLogin(e *core.RequestEvent) error {
	_, err := loginHammerhead(e)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, nil)
}

func loginHammerhead(e *core.RequestEvent) (*hammerhead.HammerheadApi, error) {

	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return nil, apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
	}

	userId := ""
	if e.Auth != nil {
		userId = e.Auth.Id
	} else {
		return nil, e.UnauthorizedError("authentication required", nil)
	}

	integrations, err := e.App.FindAllRecords("integrations", dbx.NewExp("user = {:id}", dbx.Params{"id": userId}))
	if err != nil {
		return nil, err
	}
	if len(integrations) == 0 {
		return nil, apis.NewBadRequestError("user has no integration", nil)
	}
	integration := integrations[0]
	hammerheadString := integration.GetString("hammerhead")
	if len(hammerheadString) == 0 {
		return nil, apis.NewBadRequestError("hammerhead integration missing", nil)
	}
	var hammerheadIntegration hammerhead.HammerheadIntegration
	err = json.Unmarshal([]byte(hammerheadString), &hammerheadIntegration)
	if err != nil {
		return nil, err
	}
	decryptedPassword, err := security.Decrypt(hammerheadIntegration.Password, encryptionKey)
	if err != nil {
		return nil, err
	}

	k := &hammerhead.HammerheadApi{}

	err = k.Login(hammerheadIntegration.Email, string(decryptedPassword))
	if err != nil {
		return nil, apis.NewUnauthorizedError("invalid credentials", nil)
	}

	return k, e.JSON(http.StatusOK, nil)
}
