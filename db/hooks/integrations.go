package hooks

import (
	"encoding/json"
	"os"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func ListIntegrationHandler() func(e *core.RecordsListRequestEvent) error {
	return func(e *core.RecordsListRequestEvent) error {
		if e.HasSuperuserAuth() {
			return e.Next()
		}
		for _, r := range e.Records {

			err := censorIntegrationSecrets(r)
			if err != nil {
				return err
			}
		}

		return e.Next()
	}
}

func CreateIntegrationHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := encryptIntegrationSecrets(e.App, e.Record)
		if err != nil {
			return err
		}

		return e.Next()
	}
}

func CreateUpdateIntegrationSuccessHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := censorIntegrationSecrets(e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	}
}

func UpdateIntegrationHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := encryptIntegrationSecrets(e.App, e.Record)
		if err != nil {
			return err
		}

		return e.Next()
	}
}

func censorIntegrationSecrets(r *core.Record) error {
	secrets := map[string][]string{
		"strava":     {"clientSecret", "refreshToken", "accessToken", "expiresAt"},
		"komoot":     {"password"},
		"hammerhead": {"password"},
	}
	for key, secretKeys := range secrets {
		if integrationString := r.GetString(key); integrationString != "" {
			var integration map[string]interface{}
			if err := json.Unmarshal([]byte(integrationString), &integration); err != nil {
				return err
			}
			if integration == nil {
				continue
			}
			for _, secretKey := range secretKeys {
				integration[secretKey] = ""
			}
			b, err := json.Marshal(integration)
			if err != nil {
				return err
			}
			r.Set(key, string(b))
		}
	}

	return nil
}

func encryptIntegrationSecrets(app core.App, r *core.Record) error {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
	}

	secrets := map[string][]string{
		"strava":     {"clientSecret", "refreshToken", "accessToken", "expiresAt"},
		"komoot":     {"password"},
		"hammerhead": {"password"},
	}

	original, _ := app.FindRecordById("integrations", r.Id)

	for key, secretKeys := range secrets {
		if integrationString := r.GetString(key); integrationString != "" {
			var integration map[string]interface{}
			if err := json.Unmarshal([]byte(integrationString), &integration); err != nil {
				return err
			}

			for _, secretKey := range secretKeys {
				// If the secret is already encrypted, we don't re-encrypt it.
				// TODO: This is a bit of a hack, we should handle this in a more robust way (e.g.
				// storing flag on the record or prefixing encrypted strings with enc: or smilar).
				// Doing that would also potentially allow us to support key rotation in the future.
				if secret, ok := integration[secretKey].(string); ok && len(secret) > 0 && !util.CanDecryptSecret(secret) {
					encryptedSecret, err := security.Encrypt([]byte(secret), encryptionKey)
					if err != nil {
						return err
					}
					integration[secretKey] = encryptedSecret
				} else if original != nil {

					originalString := original.GetString(key)
					var originalIntegration map[string]interface{}
					if err := json.Unmarshal([]byte(originalString), &originalIntegration); err != nil {
						return err
					}
					if integration == nil {
						continue
					}
					integration[secretKey] = originalIntegration[secretKey]
				}
			}

			b, err := json.Marshal(integration)
			if err != nil {
				return err
			}
			r.Set(key, string(b))
		}
	}

	return nil
}
