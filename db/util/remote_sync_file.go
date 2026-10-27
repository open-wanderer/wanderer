package util

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/doyensec/safeurl"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

var newRemoteSyncFileClient = func(config *safeurl.Config) HTTPDoer { return safeurl.Client(config) }

func federatedMediaHTTPConfig(originIRI string) (*safeurl.Config, error) {
	origin, err := url.Parse(originIRI)
	if err != nil || origin.Hostname() == "" || origin.User != nil || (origin.Scheme != "http" && origin.Scheme != "https") {
		return nil, fmt.Errorf("invalid federation origin IRI")
	}
	if raw := origin.Port(); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid federation origin port")
		}
	}
	return federatedFileHTTPConfig, nil
}

// DownloadRemoteSyncFile streams a canonical origin's file to temporary storage.
// Call cleanup after the record transaction has consumed the file.
func DownloadRemoteSyncFile(ctx context.Context, rawURL, originIRI, name string, maxBytes int64) (*filesystem.File, func(), error) {
	noop := func() {}
	if maxBytes <= 0 {
		return nil, noop, fmt.Errorf("file size limit must be positive")
	}
	config, err := federatedMediaHTTPConfig(originIRI)
	if err != nil {
		return nil, noop, err
	}
	ctx, cancel := context.WithTimeout(ctx, RemoteAttachmentTimeout)
	defer cancel()
	client := newRemoteSyncFileClient(config)
	defer closeIdleHTTPConnections(client)
	dir, err := os.MkdirTemp("", "wanderer-sync-file-*")
	if err != nil {
		return nil, noop, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	payload := filepath.Join(dir, "payload")
	out, err := os.OpenFile(payload, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	_, finalURL, err := fetchBoundedTo(ctx, client, rawURL, maxBytes, out)
	closeErr := out.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	filename := attachmentFileName(finalURL)
	if filename == "attachment" {
		filename = attachmentFileName(name)
	}
	named := filepath.Join(dir, filename)
	if err := os.Rename(payload, named); err != nil {
		cleanup()
		return nil, noop, err
	}
	file, err := filesystem.NewFileFromPath(named)
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	if file.Size == 0 {
		cleanup()
		return nil, noop, fmt.Errorf("cannot create an empty file")
	}
	return file, cleanup, nil
}
