package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func UserEmailChange(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var data struct {
		Email string `json:"email"`
	}
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.Email == "" {
		return apis.NewBadRequestError("Email is required", nil)
	}

	e.Auth.Set("email", data.Email)
	if err := e.App.Save(e.Auth); err != nil {
		return err
	}

	token, err := e.Auth.NewAuthToken()
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]any{
		"token":  token,
		"record": e.Auth,
	})
}
