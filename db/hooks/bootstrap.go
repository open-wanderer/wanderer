package hooks

import (
	"cmp"
	"os"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cast"
)

func OnBootstrapHandler() func(se *core.BootstrapEvent) error {
	return func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		if e.App.Settings().Meta.AppName == "Acme" {
			e.App.Settings().Meta.AppName = "wanderer"
		}
		if v := os.Getenv("ORIGIN"); v != "" {
			e.App.Settings().Meta.AppURL = v
		}
		if v := cmp.Or(os.Getenv("POCKETBASE_SMTP_SENDER_ADDRESS"), os.Getenv("POCKETBASE_SMTP_SENDER_ADRESS")); v != "" {
			e.App.Settings().Meta.SenderAddress = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_SENDER_NAME"); v != "" {
			e.App.Settings().Meta.SenderName = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_ENABLED"); v != "" {
			e.App.Settings().SMTP.Enabled = cast.ToBool(v)
		}
		if v := os.Getenv("POCKETBASE_SMTP_HOST"); v != "" {
			e.App.Settings().SMTP.Host = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_PORT"); v != "" {
			e.App.Settings().SMTP.Port = cast.ToInt(v)
		}
		if v := os.Getenv("POCKETBASE_SMTP_USERNAME"); v != "" {
			e.App.Settings().SMTP.Username = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_PASSWORD"); v != "" {
			e.App.Settings().SMTP.Password = v
		}

		if err := e.App.Save(e.App.Settings()); err != nil {
			return err
		}

		createSuperuserFromEnv(e.App)

		return nil
	}
}

// createSuperuserFromEnv provisions an initial PocketBase superuser from the
// POCKETBASE_SUPERUSER_EMAIL / POCKETBASE_SUPERUSER_PASSWORD environment
// variables, so a fresh instance can be bootstrapped without the interactive
// installer link.
//
// It is idempotent and conservative: if a superuser with that email already
// exists it is left untouched, so a password later changed in the UI is never
// reset on restart. Both variables must be set; if only one is provided this is
// treated as a misconfiguration and logged. Failures are logged but never abort
// startup - the operator can always fall back to the installer.
func createSuperuserFromEnv(app core.App) {
	email := os.Getenv("POCKETBASE_SUPERUSER_EMAIL")
	password := os.Getenv("POCKETBASE_SUPERUSER_PASSWORD")

	if email == "" && password == "" {
		return
	}
	if email == "" || password == "" {
		app.Logger().Warn("Skipping superuser bootstrap: both POCKETBASE_SUPERUSER_EMAIL and POCKETBASE_SUPERUSER_PASSWORD must be set")
		return
	}

	superusers, err := app.FindCachedCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		app.Logger().Error("Failed to bootstrap superuser: cannot load the _superusers collection", "error", err)
		return
	}

	if _, err := app.FindAuthRecordByEmail(superusers, email); err == nil {
		// Already present - do not touch the existing account or its password.
		return
	}

	superuser := core.NewRecord(superusers)
	superuser.SetEmail(email)
	superuser.SetPassword(password)

	if err := app.Save(superuser); err != nil {
		// Most commonly an invalid email or a password shorter than the
		// collection minimum - Save returns a descriptive validation error.
		app.Logger().Error("Failed to bootstrap superuser from environment", "email", email, "error", err)
		return
	}

	app.Logger().Info("Created initial superuser from environment", "email", email)
}
