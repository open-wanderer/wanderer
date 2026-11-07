package main

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"pocketbase/services/regions"
)

func TestBackupsLeaveRegionArchivesOut(t *testing.T) {
	f := newRecordHooksFixture(t)
	f.bindHooks()

	archive := regions.RegionArchivePath(f.app, "monaco")
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("pmtiles"), 0o644); err != nil {
		t.Fatal(err)
	}
	upload := filepath.Join(f.app.DataDir(), "storage", "collection", "record", "photo.webp")
	if err := os.MkdirAll(filepath.Dir(upload), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(upload, []byte("webp"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := f.app.CreateBackup(context.Background(), "test.zip"); err != nil {
		t.Fatal(err)
	}

	backup, err := zip.OpenReader(filepath.Join(f.app.DataDir(), "backups", "test.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer backup.Close()

	var names []string
	for _, file := range backup.File {
		names = append(names, file.Name)
		if strings.HasPrefix(file.Name, "region_archives/") {
			t.Errorf("backup contains %s", file.Name)
		}
	}
	for _, want := range []string{"data.db", "storage/collection/record/photo.webp"} {
		if !slices.Contains(names, want) {
			t.Errorf("backup lacks %s; it holds %v", want, names)
		}
	}
	if _, err := os.Stat(archive); err != nil {
		t.Error("the backup removed the archive from disk")
	}
}
