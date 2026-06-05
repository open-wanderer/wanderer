package pluginsystem

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const (
	workerMessageCallExport       = "call_export"
	workerMessageShutdown         = "shutdown"
	workerMessageHostHTTPResponse = "host_http_response"
	workerMessageHostHTTPRequest  = "host_http_request"
	workerMessageHostLog          = "host_log"
	workerMessageCallResult       = "call_result"
	workerMessageError            = "error"

	defaultWorkerRequestMaxBytes  = 32 * 1024 * 1024
	defaultWorkerResponseMaxBytes = 64 * 1024 * 1024
)

// workerMessage is one framed RPC message on the worker stdio protocol. The
// protocol is strictly synchronous (one call_export in flight at a time, with
// host HTTP RPC nested synchronously), so messages carry no correlation ID.
type workerMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type workerCallExport struct {
	WASMPath    string `json:"wasmPath"`
	Export      string `json:"export"`
	InputBase64 string `json:"inputBase64,omitempty"`
	SessionID   string `json:"sessionId,omitempty"`
}

type workerCallResult struct {
	OutputBase64 string       `json:"outputBase64,omitempty"`
	PluginError  *PluginError `json:"pluginError,omitempty"`
}

type workerHostHTTPRequest struct {
	RequestBase64 string `json:"requestBase64"`
}

type workerHostHTTPResponse struct {
	ResponseBase64 string `json:"responseBase64"`
}

type workerHostLog struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	SessionID string `json:"sessionId,omitempty"`
}

type workerError struct {
	Message string `json:"message"`
}

func writeWorkerMessage(w io.Writer, maxBytes int, msg workerMessage) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(payload) > maxBytes {
		return fmt.Errorf("worker rpc frame too large: %d > %d", len(payload), maxBytes)
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if err := writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, payload)
}

func readWorkerMessage(r io.Reader, maxBytes int) (workerMessage, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return workerMessage{}, err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return workerMessage{}, fmt.Errorf("worker rpc frame is empty")
	}
	if int(size) > maxBytes {
		return workerMessage{}, fmt.Errorf("worker rpc frame too large: %d > %d", size, maxBytes)
	}
	payload := make([]byte, int(size))
	if _, err := io.ReadFull(r, payload); err != nil {
		return workerMessage{}, err
	}
	var msg workerMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return workerMessage{}, err
	}
	if msg.Type == "" {
		return workerMessage{}, fmt.Errorf("worker rpc message type is empty")
	}
	return msg, nil
}

func encodeWorkerBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func decodeWorkerBytes(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	return base64.StdEncoding.DecodeString(encoded)
}

func workerData[T any](msg workerMessage) (T, error) {
	var value T
	if len(msg.Data) == 0 {
		return value, nil
	}
	if err := json.Unmarshal(msg.Data, &value); err != nil {
		return value, err
	}
	return value, nil
}

func workerMessageWithData[T any](typ string, data T) (workerMessage, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return workerMessage{}, err
	}
	return workerMessage{Type: typ, Data: raw}, nil
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}
