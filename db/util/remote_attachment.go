package util

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
)

const (
	// Per-file size limits, equal to the maxSize of the matching collection fields.
	RemoteGPXMaxBytes    int64 = 5 << 20
	RemotePhotoMaxBytes  int64 = 20 << 20
	RemoteAvatarMaxBytes int64 = 5 << 20

	// RemoteAttachmentTimeout bounds a single attachment download.
	RemoteAttachmentTimeout = 30 * time.Second

	// RemoteMaxPhotosPerObject is the photo cap used when the collection's photos
	// field cannot be read (maxSelect is 99 on trails and summit_logs).
	RemoteMaxPhotosPerObject = 99

	// Attachment budget of one inbound activity, shared by every object it stores.
	// MaxFiles counts download attempts, MaxBytes counts stored bytes.
	RemoteActivityAttachmentMaxBytes int64 = 512 << 20
	RemoteActivityAttachmentMaxFiles       = 120
	RemoteActivityAttachmentTimeout        = 5 * time.Minute
)

// ErrRemoteAttachmentBudgetExhausted is returned when the activity's attachment
// budget has no room left. No request is sent.
var ErrRemoteAttachmentBudgetExhausted = errors.New("attachment budget of this activity is exhausted")

// SetRemoteAttachmentBudgetLimitsForTesting overrides the limits of budgets
// created afterwards and returns a restore function. Tests only.
func SetRemoteAttachmentBudgetLimitsForTesting(maxBytes int64, maxFiles int) (restore func()) {
	remoteAttachmentBudgetMu.Lock()
	previousBytes, previousFiles := remoteAttachmentBudgetBytes, remoteAttachmentBudgetFiles
	remoteAttachmentBudgetBytes, remoteAttachmentBudgetFiles = maxBytes, maxFiles
	remoteAttachmentBudgetMu.Unlock()
	return func() {
		remoteAttachmentBudgetMu.Lock()
		remoteAttachmentBudgetBytes, remoteAttachmentBudgetFiles = previousBytes, previousFiles
		remoteAttachmentBudgetMu.Unlock()
	}
}

var (
	remoteAttachmentBudgetMu    sync.RWMutex
	remoteAttachmentBudgetBytes = RemoteActivityAttachmentMaxBytes
	remoteAttachmentBudgetFiles = RemoteActivityAttachmentMaxFiles
)

// WithRemoteAttachmentBudget returns a context carrying a fresh attachment
// budget, bounded by RemoteActivityAttachmentTimeout. If ctx already carries a
// budget, it is returned unchanged with a no-op cancel.
func WithRemoteAttachmentBudget(ctx context.Context) (context.Context, context.CancelFunc) {
	if remoteAttachmentBudgetFrom(ctx) != nil {
		return ctx, func() {}
	}
	remoteAttachmentBudgetMu.RLock()
	b := &remoteAttachmentBudget{bytesLeft: remoteAttachmentBudgetBytes, filesLeft: remoteAttachmentBudgetFiles}
	remoteAttachmentBudgetMu.RUnlock()
	ctx, cancel := context.WithTimeout(ctx, RemoteActivityAttachmentTimeout)
	return context.WithValue(ctx, remoteAttachmentBudgetKey{}, b), cancel
}

type remoteAttachmentBudgetKey struct{}

// remoteAttachmentBudget tracks the attempts and bytes left for one activity.
type remoteAttachmentBudget struct {
	mu        sync.Mutex
	bytesLeft int64
	filesLeft int
}

func remoteAttachmentBudgetFrom(ctx context.Context) *remoteAttachmentBudget {
	b, _ := ctx.Value(remoteAttachmentBudgetKey{}).(*remoteAttachmentBudget)
	return b
}

// reserve takes one attempt and returns the size limit for it, capped at the
// bytes left.
func (b *remoteAttachmentBudget) reserve(maxBytes int64) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.filesLeft <= 0 || b.bytesLeft <= 0 {
		return 0, ErrRemoteAttachmentBudgetExhausted
	}
	b.filesLeft--
	return min(maxBytes, b.bytesLeft), nil
}

// charge subtracts the size of a stored file, refusing it if it no longer fits.
func (b *remoteAttachmentBudget) charge(size int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if size > b.bytesLeft {
		return ErrRemoteAttachmentBudgetExhausted
	}
	b.bytesLeft -= size
	return nil
}

// ErrAttachmentPortNotAllowed is returned when an attachment URL uses a port
// that is not allowed.
var ErrAttachmentPortNotAllowed = errors.New("attachment URL port is not allowed")

var (
	remoteAttachmentClientMu sync.RWMutex
	remoteAttachmentClient   HTTPDoer

	// remoteAttachmentAllowedIPsForTest widens the IP allow list of the attachment
	// client. Set only by tests.
	remoteAttachmentAllowedIPsForTest []string
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

// setAttachmentName names f after name, keeping a random suffix like
// PocketBase does. Without an extension in name, the extension detected from
// the content is used.
func setAttachmentName(f *filesystem.File, name string) {
	ext := filepath.Ext(name)
	if ext == "" {
		ext = filepath.Ext(f.Name)
	}
	f.OriginalName = name
	f.Name = strings.TrimSuffix(name, ext) + "_" + security.RandomStringWithAlphabet(10, "abcdefghijklmnopqrstuvwxyz0123456789") + ext
}

// attachmentOriginPort returns the hostname and port of origin. ok is false for
// ports 80 and 443.
func attachmentOriginPort(origin string) (hostname string, port int, ok bool) {
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", 0, false
	}
	port, err = effectiveURLPort(u)
	if err != nil || port == 80 || port == 443 {
		return "", 0, false
	}
	return u.Hostname(), port, true
}

// effectiveURLPort returns the explicit port of u, or the scheme default.
func effectiveURLPort(u *url.URL) (int, error) {
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return 0, fmt.Errorf("invalid port %q", p)
		}
		return n, nil
	}
	if u.Scheme == "https" {
		return 443, nil
	}
	return 80, nil
}

// checkAttachmentURLPort allows ports 80 and 443 on any host, and any other port
// only for origin's own hostname and port.
func checkAttachmentURLPort(u *url.URL, origin string) error {
	port, err := effectiveURLPort(u)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrAttachmentPortNotAllowed, err)
	}
	if port == 80 || port == 443 {
		return nil
	}
	if hostname, originPort, ok := attachmentOriginPort(origin); ok &&
		strings.EqualFold(u.Hostname(), hostname) && port == originPort {
		return nil
	}
	return fmt.Errorf("%w: %d", ErrAttachmentPortNotAllowed, port)
}

// attachmentRedirectPolicy is publicMediaRedirectPolicy plus the port rule.
func attachmentRedirectPolicy(origin string) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if err := publicMediaRedirectPolicy(req, via); err != nil {
			return err
		}
		return checkAttachmentURLPort(req.URL, origin)
	}
}

// newRemoteAttachmentClient returns an SSRF-safe client for attachments of an
// object stored under the actor at origin. Besides ports 80 and 443 it allows
// the origin's own port on the origin's hostname, including on redirects.
func newRemoteAttachmentClient(origin string) HTTPDoer {
	ports := []int{80, 443}
	if _, port, ok := attachmentOriginPort(origin); ok {
		ports = append(ports, port)
	}
	return newSafeURLClient(RemoteAttachmentTimeout, attachmentRedirectPolicy(origin), ports, remoteAttachmentAllowedIPsForTest)
}

// DownloadRemoteFile downloads rawURL to a temp file and returns it with a
// cleanup function (never nil) that removes the temp file. origin is the IRI of
// the actor the object is stored under.
//
// Only public addresses are fetched, ports are restricted as in
// checkAttachmentURLPort, and the body is limited to maxBytes and
// RemoteAttachmentTimeout. Each call draws from the attachment budget in ctx, or
// from a fresh budget if ctx has none; when it is exhausted no request is sent
// and ErrRemoteAttachmentBudgetExhausted is returned.
func DownloadRemoteFile(ctx context.Context, rawURL string, maxBytes int64, origin string) (*filesystem.File, func(), error) {
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

	// A test override client skips the port check.
	client := installedRemoteAttachmentClient()
	if client == nil {
		if err := checkAttachmentURLPort(parsed, origin); err != nil {
			return nil, noop, err
		}
		client = newRemoteAttachmentClient(origin)
	}

	if remoteAttachmentBudgetFrom(ctx) == nil {
		var cancelBudget context.CancelFunc
		ctx, cancelBudget = WithRemoteAttachmentBudget(ctx)
		defer cancelBudget()
	}
	budget := remoteAttachmentBudgetFrom(ctx)
	maxBytes, err = budget.reserve(maxBytes)
	if err != nil {
		return nil, noop, err
	}

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

	f, err := filesystem.NewFileFromPath(payload)
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	setAttachmentName(f, attachmentFileName(finalURL))
	if f.Size == 0 {
		cleanup()
		return nil, noop, fmt.Errorf("cannot create an empty file")
	}
	if err := budget.charge(f.Size); err != nil {
		cleanup()
		return nil, noop, err
	}
	return f, cleanup, nil
}
