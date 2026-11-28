package assets_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"pocketbase/hooks"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type countedPhotoReader struct {
	reader    filesystem.FileReader
	size      int64
	opens     atomic.Int32
	fullReads atomic.Int32
}

func (r *countedPhotoReader) Open() (io.ReadSeekCloser, error) {
	r.opens.Add(1)
	reader, err := r.reader.Open()
	if err != nil {
		return nil, err
	}
	return &countedPhotoStream{ReadSeekCloser: reader, owner: r}, nil
}

type countedPhotoStream struct {
	io.ReadSeekCloser
	owner *countedPhotoReader
	read  int64
}

func (r *countedPhotoStream) Read(p []byte) (int, error) {
	n, err := r.ReadSeekCloser.Read(p)
	r.read += int64(n)
	return n, err
}

func (r *countedPhotoStream) Close() error {
	if r.read >= r.owner.size {
		r.owner.fullReads.Add(1)
	}
	return r.ReadSeekCloser.Close()
}

func countedPhotoFile(t *testing.T, body []byte, name string) (*filesystem.File, *countedPhotoReader) {
	t.Helper()
	file, err := filesystem.NewFileFromBytes(body, name)
	if err != nil {
		t.Fatal(err)
	}
	reader := &countedPhotoReader{reader: file.Reader, size: int64(len(body))}
	file.Reader = reader
	return file, reader
}

func assertTransientHashIsPrivate(t *testing.T, app core.App, record *core.Record) {
	t.Helper()
	const privateKey = "__unsaved_photo_hash"
	if _, exported := record.WithCustomData(true).PublicExport()[privateKey]; exported {
		t.Fatal("internal hash identity appeared in a public export")
	}
	encoded, err := json.Marshal(record)
	if err != nil || bytes.Contains(encoded, []byte(privateKey)) {
		t.Fatalf("internal hash identity was serialized: %v %s", err, encoded)
	}
	dbData, err := record.DBExport(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, persisted := dbData[privateKey]; persisted {
		t.Fatal("internal hash identity appeared in a database export")
	}
}

func TestPhotoHashMarkerRequiresServerValueAndDistinctFiles(t *testing.T) {
	collection := core.NewBaseCollection("assets")
	collection.Fields.Add(&core.TextField{Name: "type"}, &core.FileField{Name: "file"}, &core.JSONField{Name: "metadata"})
	record := core.NewRecord(collection)
	record.Set("type", "photo")
	file, reader := countedPhotoFile(t, []byte("server bytes"), "first.png")
	record.Set("file", file)
	record.Set("metadata", map[string]any{"content_hash": "client value"})
	record.SetRaw("__unsaved_photo_hash", map[string]any{"file": file, "hash": "client value"})
	if err := util.HashUnsavedPhotoFile(record); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		record.Set("metadata", map[string]any{"content_hash": "client value again"})
		if err := util.HashUnsavedPhotoFile(record); err != nil {
			t.Fatal(err)
		}
	}
	metadata := map[string]any{}
	if err := record.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	if reader.opens.Load() != 1 || metadata["content_hash"] != fmt.Sprintf("%x", sha256.Sum256([]byte("server bytes"))) {
		t.Fatalf("untrusted marker/hash was reused or same file was read again: opens%d metadata%v", reader.opens.Load(), metadata)
	}
	// Reusing the filename is insufficient: a distinct file pointer gets its
	// own hash even when clients keep the old metadata value.
	other, otherReader := countedPhotoFile(t, []byte("new server bytes"), "second.png")
	other.Name = file.Name
	record.Set("file", other)
	if err := util.HashUnsavedPhotoFile(record); err != nil {
		t.Fatal(err)
	}
	if err := record.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	if otherReader.opens.Load() != 1 || metadata["content_hash"] != fmt.Sprintf("%x", sha256.Sum256([]byte("new server bytes"))) {
		t.Fatalf("new pointer reused the old hash: opens%d metadata%v", otherReader.opens.Load(), metadata)
	}
}

func TestCreatePhotoAssetAndSaveHookShareOneFullHashRead(t *testing.T) {
	app := newAssetFilterTestApp(t)
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{
		"username": "read-count", "preferred_username": "read-count", "domain": "a.example",
		"public_key": "key", "is_local": true, "iri": "https://a.example/read-count", "inbox": "https://a.example/read-count/inbox",
	})
	photo, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	// Sniffing only reads the prefix, while hashing and storage each read
	// the full byte stream. Padding keeps those operations distinguishable.
	photo = append(photo, make([]byte, 1<<20)...)
	file, reader := countedPhotoFile(t, photo, "counted.png")
	var beforeCreateHook int32
	app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
		beforeCreateHook = reader.opens.Load()
		return e.Next()
	})
	app.OnRecordCreate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
	app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
		if reader.opens.Load() != beforeCreateHook {
			t.Errorf("create hook reopened an already hashed file: before%d after%d", beforeCreateHook, reader.opens.Load())
		}
		return e.Next()
	})
	asset, err := util.CreatePhotoAsset(app, util.PhotoAssetInput{
		Author: actor.Id, File: file,
		Metadata: map[string]any{"content_hash": "client value", "label": "retained"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reader.fullReads.Load() != 2 {
		t.Errorf("creation fully read the file%d times through%d opens; want one hash and one storage read", reader.fullReads.Load(), reader.opens.Load())
	}
	want := fmt.Sprintf("%x", sha256.Sum256(photo))
	metadata := map[string]any{}
	if err := asset.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["content_hash"] != want || metadata["label"] != "retained" {
		t.Fatalf("creation trusted client metadata or lost unrelated metadata: %v", metadata)
	}
	assertTransientHashIsPrivate(t, app, asset)
	stored, err := app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetRaw("__unsaved_photo_hash") != nil {
		t.Fatal("internal hash identity survived a database round trip")
	}

	// A new file object must be read again. Pre-hashing and the ordinary
	// invalidation/save hooks must still read that replacement only once.
	replacement := append(bytes.Clone(photo), []byte("changed bytes")...)
	replacementFile, replacementReader := countedPhotoFile(t, replacement, "replaced.png")
	stored.Set("file", replacementFile)
	stored.Set("metadata", map[string]any{"content_hash": "replacement value", "label": "retained"})
	if err := util.HashUnsavedPhotoFile(stored); err != nil {
		t.Fatal(err)
	}
	if replacementReader.opens.Load() != 1 {
		t.Fatal("replacement did not perform its own fresh hash read")
	}
	app.OnRecordUpdate("assets").BindFunc(hooks.InvalidateAssetContentHashOnFileChange())
	var beforeUpdateHook int32
	app.OnRecordUpdate("assets").BindFunc(func(e *core.RecordEvent) error {
		beforeUpdateHook = replacementReader.opens.Load()
		cleared := map[string]any{}
		if err := e.Record.UnmarshalJSONField("metadata", &cleared); err != nil {
			return err
		}
		if _, exists := cleared["content_hash"]; exists {
			t.Error("physical file replacement did not invalidate metadata hash")
		}
		return e.Next()
	})
	app.OnRecordUpdate("assets").BindFunc(hooks.HashAssetPhotoOnWrite())
	app.OnRecordUpdate("assets").BindFunc(func(e *core.RecordEvent) error {
		if replacementReader.opens.Load() != beforeUpdateHook {
			t.Errorf("update hook reopened a prehashed replacement: before%d after%d", beforeUpdateHook, replacementReader.opens.Load())
		}
		return e.Next()
	})
	if err := app.Save(stored); err != nil {
		t.Fatal(err)
	}
	if replacementReader.fullReads.Load() != 2 {
		t.Errorf("replacement fully read the file%d times through%d opens; want one hash and one storage read", replacementReader.fullReads.Load(), replacementReader.opens.Load())
	}
	assertTransientHashIsPrivate(t, app, stored)
	stored, err = app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	if err := stored.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["content_hash"] != fmt.Sprintf("%x", sha256.Sum256(replacement)) || metadata["content_hash"] == want || metadata["label"] != "retained" {
		t.Fatalf("replacement hash/metadata is incorrect: %v", metadata)
	}
}
