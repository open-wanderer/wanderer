package srch0

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestSRCH0MissingMigrationExclusion(t *testing.T) {
	for _, missing := range externalSettingsMigrations {
		t.Run(missing, func(t *testing.T) {
			registered := []*core.Migration{{File: "database_schema.go"}}
			for _, name := range externalSettingsMigrations {
				if name == missing {
					// A rename must not silently turn an excluded external
					// migration into an executable database migration.
					name = "renamed_" + name
				}
				registered = append(registered, &core.Migration{File: name})
			}
			selected, err := databaseMigrations(registered)
			if err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf("expected targeted error naming %s, got %v", missing, err)
			}
			if len(selected.Items()) != 0 {
				t.Fatal("incomplete exclusion list produced executable migrations")
			}
		})
	}
}
