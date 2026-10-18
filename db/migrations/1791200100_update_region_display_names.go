package migrations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Region names used to be derived from CoMaps ids by stripping the parent's id,
// which left 491 names such as "Styria_Graz" or "Russia_Altai Krai". The seed
// now carries CoMaps' English display names ("Styria — Graz", "Altai Krai").
// 1785000000 only seeds an empty table, so instances that were seeded before
// get the new names here. A row is only renamed while it still has the
// derived name, so names an admin changed are kept.
func init() {
	m.Register(func(app core.App) error {
		var catalog SeedCatalog
		if err := json.Unmarshal(regionsSeedJSON, &catalog); err != nil {
			return fmt.Errorf("parse regions seed: %w", err)
		}

		renamed := 0
		for _, row := range catalog.Rows {
			derived := derivedRegionName1791200100(row.ComapsID, row.ParentComapsID)
			if derived == row.Name {
				continue
			}
			params := dbx.Params{"path": row.Path, "derived": derived, "name": row.Name}
			result, err := app.DB().NewQuery("UPDATE regions SET name = {:name} WHERE path = {:path} AND name = {:derived}").Bind(params).Execute()
			if err != nil {
				return err
			}
			if count, err := result.RowsAffected(); err == nil {
				renamed += int(count)
			}
			if _, err := app.DB().NewQuery("UPDATE region_archives SET name = {:name} WHERE path = {:path} AND name = {:derived}").Bind(params).Execute(); err != nil {
				return err
			}
		}
		if renamed > 0 {
			app.Logger().Info("updated region names to CoMaps display names", "count", renamed)
		}
		return nil
	}, nil)
}

// derivedRegionName1791200100 is the name the seed command produced before it
// used CoMaps' display names.
func derivedRegionName1791200100(comapsID, parentComapsID string) string {
	if parentComapsID != "" {
		if name, ok := strings.CutPrefix(comapsID, parentComapsID+"_"); ok {
			return name
		}
	}
	return comapsID
}
