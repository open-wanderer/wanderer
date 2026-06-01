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

		return e.App.Save(e.App.Settings())
	}
}
