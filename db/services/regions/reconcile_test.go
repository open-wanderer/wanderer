package regions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchiveFileExists(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "vector.pmtiles")
	if err := os.WriteFile(file, []byte("pm"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	t.Run("true for an existing regular file", func(t *testing.T) {
		if !archiveFileExists(file) {
			t.Fatalf("archiveFileExists(%q) = false, want true", file)
		}
	})

	t.Run("false for a missing path", func(t *testing.T) {
		missing := filepath.Join(dir, "nope.pmtiles")
		if archiveFileExists(missing) {
			t.Fatalf("archiveFileExists(%q) = true, want false", missing)
		}
	})

	t.Run("false for a directory (not a regular file)", func(t *testing.T) {
		if archiveFileExists(dir) {
			t.Fatalf("archiveFileExists(%q) = true for a dir, want false", dir)
		}
	})
}
