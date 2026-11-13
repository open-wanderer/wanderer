package util

import (
	"fmt"
	"io"

	"github.com/pocketbase/pocketbase/core"
)

func ReadTrailGPX(app core.App, trail *core.Record) ([]byte, error) {
	if trail == nil {
		return nil, nil
	}
	gpxName := trail.GetString("gpx")
	if gpxName == "" {
		return nil, nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, fmt.Errorf("open filesystem: %w", err)
	}
	defer fsys.Close()

	path := trail.BaseFilesPath() + "/" + gpxName
	reader, err := fsys.GetReader(path)
	if err != nil {
		return nil, fmt.Errorf("open gpx file %q: %w", path, err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read gpx file %q: %w", path, err)
	}
	return content, nil
}
