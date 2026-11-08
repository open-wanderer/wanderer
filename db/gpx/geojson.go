package gpx

import (
	"errors"
	"io"
)

// EncodeGeoJSON is a compile-only stub for the RED commit.
func (r *Result) EncodeGeoJSON(w io.Writer) error {
	return errors.New("gpx: EncodeGeoJSON not implemented")
}
