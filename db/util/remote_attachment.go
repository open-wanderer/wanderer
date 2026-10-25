package util

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/doyensec/safeurl"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const (
	// Default per-file limits when the matching collection field is unavailable.
	RemoteGPXMaxBytes    int64 = 5 << 20
	RemotePhotoMaxBytes  int64 = 20 << 20
	RemoteAvatarMaxBytes int64 = 5 << 20

	// RemoteAttachmentTimeout bounds a single attachment download.
	RemoteAttachmentTimeout = 30 * time.Second

	// RemoteMaxPhotosPerObject is the photo cap used when the collection's photos
	// field cannot be read (maxSelect is 99 on trails and summit_logs).
	RemoteMaxPhotosPerObject = 99
)

// RemoteFileLimit uses the configured file field, including administrator
// changes, and falls back to the shipped limit when the field is unavailable.
func RemoteFileLimit(record *core.Record, name string, fallback int64) int64 {
	if field, ok := record.Collection().Fields.GetByName(name).(*core.FileField); ok && field.MaxSize > 0 {
		return field.MaxSize
	}
	return fallback
}

var (
	remoteAttachmentClientMu  sync.RWMutex
	remoteAttachmentClient    HTTPDoer
	newRemoteAttachmentClient = func(time.Duration, func(*http.Request, []*http.Request) error) HTTPDoer {
		return safeurl.Client(federatedFileHTTPConfig)
	}
)

// SetRemoteAttachmentClientForTesting replaces the HTTP client used by
// DownloadRemoteFile and returns a restore function. Tests only.
func SetRemoteAttachmentClientForTesting(c HTTPDoer) (restore func()) {
	remoteAttachmentClientMu.Lock()
	previous := remoteAttachmentClient
	remoteAttachmentClient = c
	remoteAttachmentClientMu.Unlock()
	return func() {
		remoteAttachmentClientMu.Lock()
		remoteAttachmentClient = previous
		remoteAttachmentClientMu.Unlock()
	}
}

func installedRemoteAttachmentClient() HTTPDoer {
	remoteAttachmentClientMu.RLock()
	defer remoteAttachmentClientMu.RUnlock()
	return remoteAttachmentClient
}

// RemotePhotoLimit returns the maxSelect of the collection's photos field, or
// RemoteMaxPhotosPerObject if it cannot be read.
func RemotePhotoLimit(app core.App, collection string) int {
	col, err := app.FindCachedCollectionByNameOrId(collection)
	if err != nil || col == nil {
		return RemoteMaxPhotosPerObject
	}
	field, ok := col.Fields.GetByName("photos").(*core.FileField)
	if !ok || field == nil {
		return RemoteMaxPhotosPerObject
	}
	if field.MaxSelect > 1 {
		return field.MaxSelect
	}
	return 1
}

var attachmentNameInvalidChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// attachmentFileName derives a file-system-safe name from the URL path,
// defaulting to "attachment".
func attachmentFileName(finalURL string) string {
	name := ""
	if u, err := url.Parse(finalURL); err == nil {
		name = path.Base(u.Path)
	}
	if name == "." || name == ".." || name == "/" {
		name = ""
	}
	name = attachmentNameInvalidChars.ReplaceAllString(name, "_")
	if len(name) > 200 {
		name = name[len(name)-200:]
	}
	if name == "" || name == "." || name == ".." {
		return "attachment"
	}
	return name
}

// DownloadRemoteFile downloads rawURL to a temp file and returns it with a
// cleanup function (never nil) that removes the temp file.
//
// Only public addresses are fetched, including on redirects, and the body
// is limited to maxBytes and RemoteAttachmentTimeout.
func DownloadRemoteFile(ctx context.Context, rawURL string, maxBytes int64) (*filesystem.File, func(), error) {
	noop := func() {}
	if maxBytes <= 0 {
		return nil, noop, fmt.Errorf("attachment size limit must be positive")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, noop, fmt.Errorf("attachment URL must be http or https")
	}

	ctx, cancel := context.WithTimeout(ctx, RemoteAttachmentTimeout)
	defer cancel()

	client := installedRemoteAttachmentClient()
	if client == nil {
		client = newRemoteAttachmentClient(RemoteAttachmentTimeout, publicMediaRedirectPolicy)
	}
	defer closeIdleHTTPConnections(client)

	dir, err := os.MkdirTemp("", "wanderer-attachment-*")
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

	named := filepath.Join(dir, attachmentFileName(finalURL))
	if err := os.Rename(payload, named); err != nil {
		cleanup()
		return nil, noop, err
	}
	f, err := filesystem.NewFileFromPath(named)
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	if f.Size == 0 {
		cleanup()
		return nil, noop, fmt.Errorf("cannot create an empty file")
	}
	return f, cleanup, nil
}
