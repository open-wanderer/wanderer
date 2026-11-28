package util

import (
	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"net/url"
	"path"
)

const federatedGeneratedKindTag = "wanderer:generated-kind"

const MaxFederatedTrailPhotos = 3
const MaxFederatedSummitLogPhotos = 20

// FederatedPhotoImage preserves the generated-photo marker in an attachment's
// ActivityStreams tags, alongside its canonical asset identity.
func FederatedPhotoImage(asset *core.Record, origin string) pub.Image {
	image := pub.Image{
		ID:        pub.ID(CanonicalFederatedAssetID(asset, origin)),
		Type:      pub.ImageType,
		MediaType: "image/jpeg",
		URL:       pub.IRI(AssetPublicMediaURL(asset, origin)),
	}
	if IsGeneratedRoutePreviewAsset(asset) {
		image.Tag = pub.ItemCollection{pub.Object{
			Type:    pub.NoteType,
			Name:    pub.DefaultNaturalLanguageValue(federatedGeneratedKindTag),
			Content: pub.DefaultNaturalLanguageValue("route-preview"),
		}}
	}
	return image
}

// FederatedPhotoFromAttachment reads the marker without relying on filenames:
// canonical photos can retain their identity across several instance hops.
func FederatedPhotoFromAttachment(attachment *pub.Object) (FederatedPhoto, bool) {
	if attachment == nil || attachment.Type != pub.ImageType || pub.IsNil(attachment.URL) {
		return FederatedPhoto{}, false
	}
	photo := FederatedPhoto{CanonicalID: WithoutFederatedShareToken(attachment.ID.String()), FileURL: attachment.URL.GetLink().String()}
	if photo.FileURL == "" {
		return FederatedPhoto{}, false
	}
	if parsed, err := url.Parse(photo.FileURL); err == nil {
		photo.SourceName = path.Base(parsed.Path)
	}
	if photo.CanonicalID == "" {
		photo.CanonicalID = WithoutFederatedShareToken(photo.FileURL)
	}
	for _, tag := range attachment.Tag {
		object, err := pub.ToObject(tag)
		if err == nil && object.Name.First().Value.String() == federatedGeneratedKindTag && object.Content.First().Value.String() == "route-preview" {
			return FederatedPhoto{}, false
		}
	}
	return photo, true
}

// Generated route previews stay local to each instance.
func FederatedPhotosForActivity(assets []*core.Record, limit int) []*core.Record {
	photos := make([]*core.Record, 0, len(assets))

	for _, asset := range assets {
		if IsGeneratedRoutePreviewAsset(asset) {
			continue
		} else {
			photos = append(photos, asset)
		}
	}

	if limit >= 0 && len(photos) > limit {
		photos = photos[:limit]
	}
	return photos
}
