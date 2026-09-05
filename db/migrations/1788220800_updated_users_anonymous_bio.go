package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// bio lives on the settings collection, but the users_anonymous view selected
// it unqualified. SQLite resolves that to settings.bio, while PocketBase's view
// field inference only looks at the main table (users), finds no bio field and
// falls back to defaultViewField() -- a "json" field.
//
// Reading a plain-text bio through a json field runs it through strconv.Quote
// (core.JSONField.PrepareValue), and Go escapes non-printable runes above
// U+FFFF as \U0001abcd -- an 8 digit escape that is not valid JSON. Marshalling
// the record then fails after the 200 header has already been written, so the
// client receives an empty body and an empty record.
//
// Bios containing tag characters (U+E0020-U+E007F, used by subdivision flag
// emoji such as the Scottish flag) hit this, which broke the ActivityPub actor
// endpoint with "Invalid time value" once the empty record lost its `created`.
//
// Qualifying bio with its table lets the field resolve to the underlying text
// field. PocketBase regenerates a view's fields from its query on every save,
// so saving the collection is what applies the fix.
const usersAnonymousViewQuery1788220800 = "SELECT users.id, username, avatar, settings.bio, users.created, CAST(COALESCE(json_extract(privacy, '$.account') = 'private', false) as BOOL) as private FROM users LEFT JOIN settings ON settings.user = users.id"

const usersAnonymousViewQueryOld1788220800 = "SELECT users.id, username, avatar, bio, users.created, CAST(COALESCE(json_extract(privacy, '$.account') = 'private', false) as BOOL) as private FROM users LEFT JOIN settings ON settings.user = users.id"

func init() {
	m.Register(func(app core.App) error {
		return setUsersAnonymousViewQuery1788220800(app, usersAnonymousViewQuery1788220800)
	}, func(app core.App) error {
		return setUsersAnonymousViewQuery1788220800(app, usersAnonymousViewQueryOld1788220800)
	})
}

func setUsersAnonymousViewQuery1788220800(app core.App, query string) error {
	collection, err := app.FindCollectionByNameOrId("users_anonymous")
	if err != nil {
		return err
	}

	collection.ViewQuery = query

	return app.Save(collection)
}
