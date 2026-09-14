package hooks

import (
	"fmt"
	"os"
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func CreateUserHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := createDefaultUserSettings(e.App, e.Record.Id)
		if err != nil {
			return err
		}

		if err := util.EnsureUserCategoryPriority(e.App, e.Record.Id, ""); err != nil {
			return err
		}

		_, err = util.ActorFromUser(e.App, e.Record)
		if err != nil {
			return err
		}

		return e.Next()
	}
}

func UpdateUserHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Record.Id)
		if err != nil {
			return e.Next()
		}

		icon := ""
		origin := os.Getenv("ORIGIN")
		if origin != "" && e.Record.GetString("avatar") != "" {
			icon = fmt.Sprintf("%s/api/v1/files/_pb_users_auth_/%s/%s", origin, e.Record.Id, e.Record.GetString("avatar"))
		}
		actor.Set("icon", icon)
		if err := e.App.Save(actor); err != nil {
			return err
		}

		trails, err := e.App.FindRecordsByFilter("trails", "author={:author}", "", -1, 0, dbx.Params{"author": actor.Id})
		if err != nil {
			return err
		}
		if len(trails) > 0 {
			if err := util.IndexTrails(e.App, trails, client); err != nil {
				return err
			}
		}

		lists, err := e.App.FindRecordsByFilter("lists", "author={:author}", "", -1, 0, dbx.Params{"author": actor.Id})
		if err != nil {
			return err
		}
		if len(lists) > 0 {
			if err := util.IndexLists(e.App, lists, client); err != nil {
				return err
			}
		}

		return e.Next()
	}
}

// OAuth2UsernameHandler assigns the username reported by the OAuth2 provider
// to accounts created through OAuth2.
//
// PocketBase maps the provider username itself, but only when the raw value
// already satisfies the users.username field. Providers whose usernames are
// display names - OpenStreetMap, for instance - therefore end up with a
// generated "usersNNNNNN" name. Sanitising the value first keeps the name the
// user signed up with recognisable.
func OAuth2UsernameHandler() func(e *core.RecordAuthWithOAuth2RequestEvent) error {
	return func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		if !e.IsNewRecord || e.OAuth2User == nil || e.Collection == nil {
			return e.Next()
		}

		// PocketBase only assigns the provider username when the collection
		// maps it, so respect a disabled or redirected mapping
		mapped := e.Collection.OAuth2.MappedFields.Username
		if mapped == "" {
			return e.Next()
		}

		// a value submitted by the client takes precedence
		if _, ok := e.CreateData[mapped]; ok {
			return e.Next()
		}

		field, ok := e.Collection.Fields.GetByName(mapped).(*core.TextField)
		if !ok {
			return e.Next()
		}

		// only rewrite a value the mapped field would reject
		username := e.OAuth2User.Username
		if field.ValidatePlainValue(username) != nil {
			username = util.SanitizeUsername(username, field.Min, field.Max)
		}

		username = util.UniqueUsername(e.App, e.Collection, field, username)
		if username == "" {
			// nothing usable; leave the provider value to PocketBase
			return e.Next()
		}

		// only the provider value is replaced: PocketBase still validates it and
		// checks it is free inside the create transaction, and generates a
		// username if it is not
		e.OAuth2User.Username = username

		return e.Next()
	}
}

func createDefaultUserSettings(app core.App, userId string) error {
	collection, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		return err
	}
	settings := core.NewRecord(collection)
	settings.Set("language", "en")
	settings.Set("unit", "metric")
	settings.Set("mapFocus", "trails")
	settings.Set("user", userId)
	return app.Save(settings)
}
