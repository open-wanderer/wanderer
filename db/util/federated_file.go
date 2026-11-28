package util

import (
	"context"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"io"
	"net/http"
)

func FetchFederatedFile(ctx context.Context, rawURL, originIRI, fallbackName string, maxBytes int64) (*filesystem.File, func(), error) {
	noop := func() {}
	if file, prepared, err := preparedFederatedFile(ctx, rawURL, originIRI, maxBytes); prepared {
		return file, noop, err
	}
	file, cleanup, err := DownloadRemoteFile(ctx, rawURL, maxBytes)
	if err != nil {
		return nil, cleanup, RedactFederatedShareError(err)
	}
	reader, err := file.Reader.Open()
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	var prefix [512]byte
	n, err := io.ReadFull(reader, prefix[:])
	closeErr := reader.Close()
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err == nil {
		err = closeErr
	}
	if err != nil {
		cleanup()
		return nil, noop, err
	}
	file.Name = MediaFileName(file.Name, http.DetectContentType(prefix[:n]))
	return file, cleanup, nil
}

// FederatedPhotoFileLimit follows the asset file field configured by the admin.
func FederatedPhotoFileLimit(app core.App) int64 {
	collection, err := app.FindCachedCollectionByNameOrId("assets")
	if err == nil && collection != nil {
		if field, ok := collection.Fields.GetByName("file").(*core.FileField); ok && field.MaxSize > 0 {
			return field.MaxSize
		}
	}
	return RemotePhotoMaxBytes
}
