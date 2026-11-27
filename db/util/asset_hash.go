package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"weak"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const unsavedPhotoHashKey = "__unsaved_photo_hash"

// Only server code can populate this type. Keep the identity weak so a saved
// record does not retain an upload's bytes after PocketBase releases the file.
type unsavedPhotoHash struct {
	file weak.Pointer[filesystem.File]
	hash string
}

func PhotoContentHash(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func HashUnsavedPhotoFile(record *core.Record) error {
	files := record.GetUnsavedFiles("file")
	if record.GetString("type") != "photo" || len(files) == 0 {
		return nil
	}
	computed, ok := record.GetRaw(unsavedPhotoHashKey).(unsavedPhotoHash)
	if !ok || computed.file.Value() != files[0] {
		reader, err := files[0].Reader.Open()
		if err != nil {
			return err
		}
		defer reader.Close()
		hash, err := PhotoContentHash(reader)
		if err != nil {
			return err
		}
		computed = unsavedPhotoHash{file: weak.Make(files[0]), hash: hash}
		record.SetRaw(unsavedPhotoHashKey, computed)
	}
	record.Hide(unsavedPhotoHashKey)
	metadata := map[string]any{}
	if err := record.UnmarshalJSONField("metadata", &metadata); err != nil {
		return err
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["content_hash"] = computed.hash
	record.Set("metadata", metadata)
	return nil
}
