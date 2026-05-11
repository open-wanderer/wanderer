package hooks

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func CreateAPITokenHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		rawToken := "wanderer_key_" + security.RandomString(32)

		hashedKey := security.SHA256(rawToken)

		e.Record.Set("token", hashedKey)

		// Temporarily store rawToken so we can display it once to the user
		e.Record.WithCustomData(true)
		e.Record.Set("rawToken", rawToken)

		return e.Next()
	}
}
