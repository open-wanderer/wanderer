package routes

import (
	"errors"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/mails"
	"github.com/pocketbase/pocketbase/tools/routine"
)

func UserEmailChange(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("Authentication required", nil)
	}

	var data struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.Email == "" {
		return apis.NewBadRequestError("Email is required", nil)
	}
	if data.Password == "" {
		return apis.NewBadRequestError("Current password is required", nil)
	}
	if !e.Auth.ValidatePassword(data.Password) {
		return apis.NewBadRequestError("Invalid password", nil)
	}

	e.Auth.Set("email", data.Email)
	e.Auth.Set("verified", false)
	if err := e.App.Save(e.Auth); err != nil {
		var verr validation.Errors
		if errors.As(err, &verr) {
			return apis.NewBadRequestError("Validation failed", verr)
		}
		return err
	}

	app := e.App
	routine.FireAndForget(func() {
		if err := mails.SendRecordVerification(app, e.Auth); err != nil {
			app.Logger().Error("Failed to send verification email", "error", err)
		}
	})

	token, err := e.Auth.NewAuthToken()
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]any{
		"token":  token,
		"record": e.Auth,
	})
}
