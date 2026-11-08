package assets_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"testing"

	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/hooks"
)

func TestPhotosAreHashedOnWrite(t *testing.T) {
	app := newAssetFilterTestApp(t)
	app.OnRecordCreate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
	app.OnRecordUpdate("assets").BindFunc(hooks.InvalidateAssetContentHashOnFileChange())
	app.OnRecordUpdate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{
		"username": "hash", "preferred_username": "hash", "domain": "a.example", "public_key": "key",
		"is_local": true, "iri": "https://a.example/hash", "inbox": "https://a.example/hash/inbox",
	})
	photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	for range 2 {
		file, err := filesystem.NewFileFromBytes(photo, "photo.png")
		if err != nil {
			t.Fatal(err)
		}
		asset := saveAssetFilterRecord(t, app, "assets", map[string]any{"author": actor.Id, "type": "photo", "storage_mode": "copy", "file": file})
		metadata := map[string]any{}
		if err := asset.UnmarshalJSONField("metadata", &metadata); err != nil {
			t.Fatal(err)
		}
		if metadata["content_hash"] != fmt.Sprintf("%x", sha256.Sum256(photo)) {
			t.Fatalf("photo hash missing after upload: %v", metadata)
		}
		if err := fsys.Delete(asset.BaseFilesPath() + "/" + asset.GetString("file")); err != nil {
			t.Fatal(err)
		}
	}

}

func TestAssetPhotoFileReplacementStoresHashOfNewPhysicalBytes(t *testing.T) {
	app := newAssetFilterTestApp(t)
	app.OnRecordCreate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
	app.OnRecordUpdate("assets").BindFunc(hooks.InvalidateAssetContentHashOnFileChange())
	app.OnRecordUpdate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{
		"username": "replacement", "preferred_username": "replacement", "domain": "a.example", "public_key": "key",
		"is_local": true, "iri": "https://a.example/replacement", "inbox": "https://a.example/replacement/inbox",
	})
	photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(photo, "original.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := saveAssetFilterRecord(t, app, "assets", map[string]any{
		"author": actor.Id, "type": "photo", "storage_mode": "copy", "file": file,
		"metadata": map[string]any{"label": "preserved"},
	})
	oldFilename := asset.GetString("file")
	asset, err = app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	// Trailing bytes keep this valid PNG's type while changing its exact hash.
	replacement := append(bytes.Clone(photo), []byte("replacement bytes")...)
	file, err = filesystem.NewFileFromBytes(replacement, "replacement.png")
	if err != nil {
		t.Fatal(err)
	}
	asset.Set("file", file)
	if err := app.Save(asset); err != nil {
		t.Fatal(err)
	}
	stored, err := app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	metadata := map[string]any{}
	if err := stored.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	gotHash := metadata["content_hash"]
	if gotHash != fmt.Sprintf("%x", sha256.Sum256(replacement)) || gotHash == fmt.Sprintf("%x", sha256.Sum256(photo)) {
		t.Fatalf("replacement retained the old hash or lost the new one: %q", gotHash)
	}
	if metadata["label"] != "preserved" {
		t.Fatal("hash replacement discarded unrelated metadata")
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	reader, err := fsys.GetReader(stored.BaseFilesPath() + "/" + stored.GetString("file"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	actualBytes, err := io.ReadAll(reader)
	if err != nil || !bytes.Equal(actualBytes, replacement) {
		t.Fatalf("physical replacement does not match hashed bytes: %v", err)
	}
	if stored.GetString("file") == oldFilename {
		t.Fatal("replacement did not update the stored file")
	}
	oldReader, err := fsys.GetReader(stored.BaseFilesPath() + "/" + oldFilename)
	if err == nil {
		oldReader.Close()
		t.Fatal("old physical file was not removed")
	}
}

func TestAssetFileRemovalAndRenameInvalidateHash(t *testing.T) {
	for _, name := range []string{"clear", "rename"} {
		t.Run(name, func(t *testing.T) {
			app := newAssetFilterTestApp(t)
			app.OnRecordCreate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
			app.OnRecordUpdate("assets").BindFunc(hooks.InvalidateAssetContentHashOnFileChange())
			app.OnRecordUpdate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
			actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{"username": "photographer", "preferred_username": "photographer", "is_local": true, "iri": "https://a.example/photographer", "public_key": "key", "inbox": "https://a.example/inbox"})
			photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
			if err != nil {
				t.Fatal(err)
			}
			file, err := filesystem.NewFileFromBytes(photo, "photo.png")
			if err != nil {
				t.Fatal(err)
			}
			asset := saveAssetFilterRecord(t, app, "assets", map[string]any{"author": actor.Id, "type": "photo", "storage_mode": "copy", "file": file, "metadata": map[string]any{"label": "preserved", "nested": map[string]any{"value": true}}})
			asset, err = app.FindRecordById("assets", asset.Id)
			if err != nil {
				t.Fatal(err)
			}
			metadata := map[string]any{}
			if err := asset.UnmarshalJSONField("metadata", &metadata); err != nil || metadata["content_hash"] == nil {
				t.Fatalf("fixture has no hash: %v", err)
			}
			filename := ""
			if name == "rename" {
				filename = "renamed.png"
			}
			asset.Set("file", filename)
			// Renaming an existing file is a server operation without an upload.
			if err := app.SaveNoValidate(asset); err != nil {
				t.Fatal(err)
			}
			stored, err := app.FindRecordById("assets", asset.Id)
			if err != nil {
				t.Fatal(err)
			}
			metadata = map[string]any{}
			if err := stored.UnmarshalJSONField("metadata", &metadata); err != nil {
				t.Fatal(err)
			}
			if _, ok := metadata["content_hash"]; ok {
				t.Fatal("old content hash survived the file change")
			}
			if metadata["label"] != "preserved" || metadata["nested"].(map[string]any)["value"] != true {
				t.Fatal("unrelated metadata changed")
			}
		})
	}
}
