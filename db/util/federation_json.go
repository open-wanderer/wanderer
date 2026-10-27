package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	WebfingerBodyLimit        int64 = 100 << 10
	FederationActorBodyLimit  int64 = 1 << 20
	FederationObjectBodyLimit int64 = 2 << 20
	FederationSyncBodyLimit   int64 = 32 << 20
)

var ErrFederationBodyTooLarge = errors.New("federation response exceeds body limit")

// DecodeFederationJSON bounds the complete, decompressed response before JSON
// decoding can allocate its nested objects. It also rejects trailing content.
func DecodeFederationJSON(body io.Reader, limit int64, value any) error {
	if limit <= 0 {
		return fmt.Errorf("invalid federation response limit")
	}
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > limit {
		return ErrFederationBodyTooLarge
	}
	return json.Unmarshal(data, value)
}
