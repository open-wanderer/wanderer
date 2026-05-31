package manifestcheck

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func PrintFile(w io.Writer, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(encoded))
	return err
}
