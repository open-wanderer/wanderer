package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"pocketbase/util"
)

// Bound decoded object overhead independently of the wire limit. Counting
// needs only a fixed buffer, before nested objects reserve processing capacity.
const remoteSyncJSONTokenLimit = 1000000
const remoteSyncJSONSharedTokenLimit = 500000

var errRemoteSyncJSONComplexity = errors.New("remote sync document has too many JSON values")

func remoteSyncJSONTokens(ctx context.Context, body io.Reader) (int, error) {
	buffer := make([]byte, 32<<10)
	count := 0
	quoted, escaped, scalar := false, false, false
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		n, readErr := body.Read(buffer)
		for _, c := range buffer[:n] {
			if quoted {
				if escaped {
					escaped = false
				} else if c == '\\' {
					escaped = true
				} else if c == '"' {
					quoted = false
				}
				continue
			}
			switch c {
			case ' ', '\t', '\r', '\n', ',', ':':
				scalar = false
			case '{', '}', '[', ']':
				count++
				scalar = false
			case '"':
				count++
				quoted = true
				scalar = false
			default:
				if !scalar {
					count++
					scalar = true
				}
			}
			if count > remoteSyncJSONTokenLimit {
				return count, errRemoteSyncJSONComplexity
			}
		}
		if readErr == io.EOF {
			return count, nil
		}
		if readErr != nil {
			return 0, readErr
		}
	}
}

func decodeRemoteSyncJSON(ctx context.Context, body io.Reader, limit int64, value any) error {
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > limit {
		return util.ErrFederationBodyTooLarge
	}
	if _, err := remoteSyncJSONTokens(ctx, bytes.NewReader(data)); err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
