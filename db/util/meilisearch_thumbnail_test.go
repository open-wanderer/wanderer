package util

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestTrailSearchDocumentThumbnail(t *testing.T) {
	collection := core.NewBaseCollection("trails")
	collection.Fields.Add(
		&core.FileField{Name: "photos", MaxSelect: 999},
		&core.NumberField{Name: "thumbnail"},
	)
	author := core.NewRecord(core.NewBaseCollection("activitypub_actors"))
	author.Id = "author000000001"
	photos := []string{"first.jpg", "second.jpg"}

	for _, test := range []struct {
		name      string
		photos    []string
		thumbnail int
		want      string
	}{
		{"negative index uses first photo", photos, -1, "first.jpg"},
		{"first photo selected", photos, 0, "first.jpg"},
		{"last photo selected", photos, 1, "second.jpg"},
		{"index at length uses first photo", photos, 2, "first.jpg"},
		{"index beyond length uses first photo", photos, 99, "first.jpg"},
		{"no photos leaves thumbnail empty", nil, -1, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			trail := core.NewRecord(collection)
			trail.Set("photos", test.photos)
			trail.Set("thumbnail", test.thumbnail)
			document, err := documentFromTrailRecord(trail, author, true)
			if err != nil {
				t.Fatal(err)
			}
			if got := document["thumbnail"]; got != test.want {
				t.Errorf("thumbnail = %v; want %q", got, test.want)
			}
		})
	}
}
