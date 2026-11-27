package assets_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

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
