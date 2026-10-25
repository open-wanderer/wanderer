package hooks

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"pocketbase/util"
	"strings"

	"github.com/go-ozzo/ozzo-validation/v4/is"

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
//
// trusted reports whether the claims of a provider slot are trusted (see
// OAuth2EmailHandler). Only for those, a missing username falls back to the
// display name: it is part of the actor IRI and the federated handle of the
// account and cannot be changed afterwards, so it must not be taken from
// providers the administrator has not opted in.
func OAuth2UsernameHandler(trusted func(provider string) bool) func(e *core.RecordAuthWithOAuth2RequestEvent) error {
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

		username := e.OAuth2User.Username

		// generic OIDC providers without a preferred_username claim, such as
		// Cloudflare Access, would otherwise get a generated name or their
		// opaque subject id; the display name is more recognisable
		if (username == "" || username == e.OAuth2User.Id) && trusted != nil && trusted(e.ProviderName) {
			username = e.OAuth2User.Name
		}

		// only rewrite a value the mapped field would reject
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

// OAuth2EmailHandler keeps the email of users of generic OIDC providers that
// do not send an email_verified claim, such as Cloudflare Access.
//
// PocketBase only takes the email from the normalised OAuth2 user when the
// provider marks it as verified, so these accounts end up without one. For the
// provider slots the administrator trusts (trusted), the raw email claim is
// handed to PocketBase as if the provider had verified it, both for new
// accounts and for existing ones that have no email yet.
//
// It has to be the OAuth2 user email, not create data: PocketBase only creates
// the account as verified when the two match. An unverified account is treated
// as possibly pre-created by an attacker on every later OAuth2 login - its
// password is reset, which invalidates the sessions of other devices - and
// confirming its email unlinks the provider again.
//
// An email_verified claim of any value, false and null included, means the
// provider has an opinion and is left alone. The email is also not taken when
// another account already uses it: PocketBase links accounts by verified
// emails only, and trusting a claim must not grant access to an existing
// account or fail with a uniqueness error.
func OAuth2EmailHandler(trusted func(provider string) bool) func(e *core.RecordAuthWithOAuth2RequestEvent) error {
	return func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		if e.OAuth2User == nil || e.Collection == nil || e.OAuth2User.Email != "" || trusted == nil || !trusted(e.ProviderName) {
			return e.Next()
		}

		if _, present := e.OAuth2User.RawUser["email_verified"]; present {
			return e.Next()
		}

		// a value submitted by the client for a new account takes precedence
		if e.Record == nil {
			if v, _ := e.CreateData[core.FieldNameEmail].(string); v != "" {
				return e.Next()
			}
		}

		email, _ := e.OAuth2User.RawUser["email"].(string)
		email = strings.TrimSpace(email)
		if email == "" || is.EmailFormat.Validate(email) != nil {
			return e.Next()
		}

		// an account that already has a different email keeps it; PocketBase
		// compares the addresses exactly as well
		if e.Record != nil && e.Record.Email() != "" && e.Record.Email() != email {
			return e.Next()
		}

		// the account being authenticated may already have the address
		owner, err := e.App.FindAuthRecordByEmail(e.Collection, email)
		if err == nil {
			if e.Record == nil || owner.Id != e.Record.Id {
				e.App.Logger().Warn(
					"OIDC email not assigned because another account uses it",
					"provider", e.ProviderName,
				)
				return e.Next()
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		e.OAuth2User.Email = email

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
