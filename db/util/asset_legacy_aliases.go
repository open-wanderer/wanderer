package util

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const LegacyPhotoAliasTable = "_asset_legacy_photo_aliases"

type LegacyPhotoAlias struct {
	Collection string
	Record     string
	File       string
}

// AssetLegacyPhotoAliases lists the historical file URLs associated with an asset.
func AssetLegacyPhotoAliases(record *core.Record) []LegacyPhotoAlias {
	metadata := AssetProvenanceMetadata(record)
	collection := AssetMetadataString(metadata, "source_collection")
	source := AssetMetadataString(metadata, "source_record")
	file := AssetMetadataString(metadata, "source_file")
	if (collection != "trails" && collection != "waypoints" && collection != "summit_logs") || source == "" || file == "" {
		return nil
	}
	return []LegacyPhotoAlias{{Collection: collection, Record: source, File: file}}
}

func CreateLegacyPhotoAliasTable(app core.App) error {
	_, err := app.DB().NewQuery(`CREATE TABLE IF NOT EXISTS _asset_legacy_photo_aliases (
		asset TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
		source_collection TEXT NOT NULL,
		source_record TEXT NOT NULL,
		source_file TEXT NOT NULL,
		PRIMARY KEY (asset, source_collection, source_record, source_file)
	);
	CREATE INDEX IF NOT EXISTS idx_asset_legacy_photo_alias_lookup ON _asset_legacy_photo_aliases (source_collection, source_record, source_file, asset);`).Execute()
	return err
}

// Call within the same transaction as the asset write. The private table has
// no records API and serves exact legacy URL lookups without parsing metadata.
func SyncAssetLegacyPhotoAliases(app core.App, record *core.Record) error {
	if _, err := app.DB().NewQuery("DELETE FROM " + LegacyPhotoAliasTable + " WHERE asset = {:asset}").Bind(dbx.Params{"asset": record.Id}).Execute(); err != nil {
		return err
	}
	aliases := AssetLegacyPhotoAliases(record)
	for start := 0; start < len(aliases); start += 200 {
		end := min(start+200, len(aliases))
		values := make([]string, 0, end-start)
		params := dbx.Params{"asset": record.Id}
		for index, alias := range aliases[start:end] {
			collection, source, file := fmt.Sprintf("c%d", index), fmt.Sprintf("r%d", index), fmt.Sprintf("f%d", index)
			values = append(values, fmt.Sprintf("({:asset}, {:%s}, {:%s}, {:%s})", collection, source, file))
			params[collection], params[source], params[file] = alias.Collection, alias.Record, alias.File
		}
		if _, err := app.DB().NewQuery("INSERT INTO " + LegacyPhotoAliasTable + " (asset, source_collection, source_record, source_file) VALUES " + strings.Join(values, ",")).Bind(params).Execute(); err != nil {
			return err
		}
	}
	return nil
}
