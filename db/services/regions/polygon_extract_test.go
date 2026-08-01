package regions

import (
	"encoding/json"
	"os"
	"testing"
)

func TestWritePolygonTempFile(t *testing.T) {
	t.Run("valid Polygon returns a readable temp file with type Polygon", func(t *testing.T) {
		polygon := map[string]any{
			"type": "Polygon",
			"coordinates": [][][]float64{
				{{11.3, 48.0}, {11.7, 48.0}, {11.3, 48.2}, {11.3, 48.0}},
			},
		}

		path, err := writePolygonTempFile(polygon)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("temp file contents did not parse as JSON: %v", err)
		}
		if decoded["type"] != "Polygon" {
			t.Fatalf("got type %v, want Polygon", decoded["type"])
		}
	})

	t.Run("valid MultiPolygon returns a file with type MultiPolygon", func(t *testing.T) {
		polygon := map[string]any{
			"type": "MultiPolygon",
			"coordinates": [][][][]float64{
				{{{11.3, 48.0}, {11.7, 48.0}, {11.3, 48.2}, {11.3, 48.0}}},
			},
		}

		path, err := writePolygonTempFile(polygon)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read temp file: %v", err)
		}

		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("temp file contents did not parse as JSON: %v", err)
		}
		if decoded["type"] != "MultiPolygon" {
			t.Fatalf("got type %v, want MultiPolygon", decoded["type"])
		}
	})

	t.Run("nil polygon returns an error and creates no file", func(t *testing.T) {
		path, err := writePolygonTempFile(nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if path != "" {
			t.Fatalf("expected empty path, got %q", path)
		}
	})

	t.Run("empty map polygon returns an error and creates no file", func(t *testing.T) {
		path, err := writePolygonTempFile(map[string]any{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if path != "" {
			t.Fatalf("expected empty path, got %q", path)
		}
	})

	t.Run("two successive calls return two distinct paths", func(t *testing.T) {
		polygon := map[string]any{
			"type": "Polygon",
			"coordinates": [][][]float64{
				{{11.3, 48.0}, {11.7, 48.0}, {11.3, 48.2}, {11.3, 48.0}},
			},
		}

		path1, err := writePolygonTempFile(polygon)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path1)

		path2, err := writePolygonTempFile(polygon)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(path2)

		if path1 == path2 {
			t.Fatalf("expected distinct paths, got the same path twice: %q", path1)
		}
	})
}
