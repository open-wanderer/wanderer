package main

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/spf13/cast"

	"pocketbase/commands"
	"pocketbase/federation"
	"pocketbase/integrations/komoot"
	"pocketbase/integrations/strava"

	_ "pocketbase/migrations"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/microcosm-cc/bluemonday"
)

const (
	defaultPocketBaseEncryptionKey = "fde406459dc1f6ca6f348e1f44a9a2af"
	defaultMeiliMasterKey          = "vODkljPcfFANYNepCHyDyGjzAMPcdHnrb6X5KyXQPWo"
)

// verifySettings checks if the required environment variables are set.
// If they are not set, it logs a warning.
func verifySettings(app core.App) {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")

	if len(encryptionKey) != 32 {
		// terminate if the encryption key is not set or is not exactly 32 bytes long,
		// as this is a requirement for PocketBase to function properly.
		log.Fatal("POCKETBASE_ENCRYPTION_KEY must be exactly 32 bytes long- See https://wanderer.to/run/installation/#prerequisites for more information")
	}

	if encryptionKey == defaultPocketBaseEncryptionKey {
		app.Logger().Warn("POCKETBASE_ENCRYPTION_KEY is still set to the default value. Please change it to a secure value")
	}

	meiliMasterKey := os.Getenv("MEILI_MASTER_KEY")

	if len(meiliMasterKey) < 32 {
		app.Logger().Warn("MEILI_MASTER_KEY not set or is shorter than 32 bytes")
	}

	if meiliMasterKey == defaultMeiliMasterKey {
		app.Logger().Warn("MEILI_MASTER_KEY is still set to the default value. Please change it to a secure value")
	}
}

func main() {

	app := pocketbase.New()
	client := initializeMeiliSearch()

	verifySettings(app)

	registerMigrations(app)
	setupEventHandlers(app, client)

	setupCommands(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func initializeMeiliSearch() meilisearch.ServiceManager {
	return meilisearch.New(
		os.Getenv("MEILI_URL"),
		meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")),
	)
}

func registerMigrations(app *pocketbase.PocketBase) {
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Dir:         "migrations",
		Automigrate: true,
	})
}

func setupEventHandlers(app *pocketbase.PocketBase, client meilisearch.ServiceManager) {
	app.OnRecordAfterCreateSuccess("users").BindFunc(createUserHandler(client))

	app.OnRecordAfterCreateSuccess("trails").BindFunc(createTrailHandler(client))
	app.OnRecordAfterUpdateSuccess("trails").BindFunc(updateTrailHandler(client))
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(deleteTrailHandler(client))

	app.OnRecordCreateRequest("summit_logs").BindFunc(createSummitLogHandler(client))
	app.OnRecordUpdateRequest("summit_logs").BindFunc(updateSummitLogHandler())
	app.OnRecordDeleteRequest("summit_logs").BindFunc(deleteSummitLogHandler(client))

	app.OnRecordCreateRequest("comments").BindFunc(createCommentHandler())
	app.OnRecordUpdateRequest("comments").BindFunc(updateCommentHandler())
	app.OnRecordDeleteRequest("comments").BindFunc(deleteCommentHandler(client))

	app.OnRecordCreateRequest("trail_share").BindFunc(createTrailShareHandler(client))
	app.OnRecordDeleteRequest("trail_share").BindFunc(deleteTrailShareHandler(client))

	app.OnRecordAfterCreateSuccess("trail_like").BindFunc(createTrailLikeHandler(client))
	app.OnRecordAfterDeleteSuccess("trail_like").BindFunc(deleteTrailLikeHandler(client))

	app.OnRecordAfterCreateSuccess("lists").BindFunc(createListHandler(client))
	app.OnRecordAfterUpdateSuccess("lists").BindFunc(updateListHandler(client))
	app.OnRecordAfterDeleteSuccess("lists").BindFunc(deleteListHandler(client))

	app.OnRecordCreateRequest("list_share").BindFunc(createListShareHandler(client))
	app.OnRecordDeleteRequest("list_share").BindFunc(deleteListShareHandler(client))

	app.OnRecordCreateRequest("follows").BindFunc(createFollowHandler())
	app.OnRecordDeleteRequest("follows").BindFunc(deleteFollowHandler())

	app.OnRecordsListRequest("integrations").BindFunc(listIntegrationHandler())
	app.OnRecordCreate("integrations").BindFunc(createIntegrationHandler())
	app.OnRecordAfterCreateSuccess("integrations").BindFunc(createUpdateIntegrationSuccessHandler())
	app.OnRecordUpdate("integrations").BindFunc(updateIntegrationHandler())
	app.OnRecordAfterUpdateSuccess("integrations").BindFunc(createUpdateIntegrationSuccessHandler())

	app.OnRecordsListRequest("feed", "profile_feed").BindFunc(listFeedHandler())

	app.OnRecordCreateRequest().BindFunc(sanitizeHTML())
	app.OnRecordUpdateRequest().BindFunc(sanitizeHTML())

	app.OnRecordRequestEmailChangeRequest("users").BindFunc(changeUserEmailHandler())
	app.OnServe().BindFunc(onBeforeServeHandler(client))

	app.OnBootstrap().BindFunc(onBootstrapHandler())
}

func setupCommands(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(commands.Dedup(app))
}

func sanitizeHTML() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		fieldsToSanitize := map[string][]string{
			"lists":       {"description"},
			"settings":    {"bio"},
			"summit_logs": {"text"},
			"trails":      {"description"},
			"comments":    {"text"},
			"waypoints":   {"description"},
		}
		collection := e.Collection.Name
		fields, ok := fieldsToSanitize[collection]
		if !ok {
			return e.Next()
		}

		p := bluemonday.NewPolicy()
		p.AllowStandardAttributes()
		p.AllowStandardURLs()
		p.AllowLists()
		p.AllowElements("br", "div", "hr", "p", "span", "wbr")
		p.AllowElements("b", "strong", "em", "u", "blockquote", "a")
		p.AllowAttrs("href").OnElements("a")
		p.AllowAttrs("target").OnElements("a")
		p.AllowAttrs("class").OnElements("a")

		for _, field := range fields {
			if val, ok := e.Record.Get(field).(string); ok {
				sanitizedValue := p.Sanitize(val)
				e.Record.Set(field, sanitizedValue)
			}
		}

		return e.Next()
	}
}

func createUserHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		userId := e.Record.Id

		err := createDefaultUserSettings(e.App, e.Record.Id)
		if err != nil {
			return err
		}

		actor, err := util.ActorFromUser(e.App, e.Record)
		if err != nil {
			return err
		}

		searchRules := map[string]interface{}{
			"lists": map[string]string{
				"filter": "public = true OR author = " + actor.Id + " OR shares = " + userId,
			},
			"trails": map[string]string{
				"filter": "public = true OR author = " + actor.Id + " OR shares = " + userId,
			},
		}

		token, err := util.GenerateMeilisearchToken(searchRules, client)
		if err != nil {
			return err
		}
		e.Record.Set("token", token)
		if err := e.App.Save(e.Record); err != nil {
			return err
		}

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

func createTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}
		if err := util.IndexTrails(e.App, []*core.Record{record}, client); err != nil {
			return err
		}
		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote trail
			// we create a stub trail record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, author, e.Record, pub.CreateType)
		if err != nil {
			return err
		}

		_, err = util.InsertIntoFeed(e.App, author.Id, author.Id, record.Id, util.TrailFeed)
		if err != nil {
			return err
		}

		return nil
	}
}

func updateTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}
		err = util.UpdateTrail(e.App, record, author, client)
		if err != nil {
			return err
		}
		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote trail
			// we create a stub trail record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, author, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}

		return nil
	}
}

func deleteTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		task, err := client.Index("trails").DeleteDocument(record.Id)
		if err != nil {
			return err
		}

		interval := 500 * time.Millisecond
		_, err = client.WaitForTask(task.TaskUID, interval)
		if err != nil {
			log.Fatalf("Error waiting for task completion: %v", err)
		}

		err = federation.CreateTrailDeleteActivity(e.App, e.Record)
		if err != nil {
			return err
		}

		err = util.DeleteFromFeed(e.App, record.Id)
		if err != nil {
			return err
		}

		return e.Next()
	}
}

func createSummitLogHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		err := e.Next()
		if err != nil {
			return err
		}

		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		trail, err := e.App.FindRecordById("trails", e.Record.GetString("trail"))
		if err != nil {
			return err
		}

		if err := util.IndexTrails(e.App, []*core.Record{trail}, client); err != nil {
			return err
		}

		err = federation.CreateSummitLogActivity(e.App, userActor, e.Record, pub.CreateType)
		if err != nil {
			return err
		}

		return nil
	}
}

func updateSummitLogHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		err := e.Next()
		if err != nil {
			return err
		}

		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		err = federation.CreateSummitLogActivity(e.App, userActor, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}
		return nil
	}
}

func deleteSummitLogHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		trail, err := e.App.FindRecordById("trails", e.Record.GetString("trail"))
		if err != nil {
			return err
		}

		if err := util.IndexTrails(e.App, []*core.Record{trail}, client); err != nil {
			return err
		}

		err = federation.CreateSummitLogDeleteActivity(e.App, e.Record)
		if err != nil {
			return err
		}
		return nil
	}
}

func createCommentHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		e.Next()

		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		err = federation.CreateCommentActivity(e.App, userActor, e.Record, pub.CreateType)
		if err != nil {
			return err
		}
		return nil
	}
}

func updateCommentHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		userActor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
		if err != nil {
			return err
		}

		err = federation.CreateCommentActivity(e.App, userActor, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}
		return e.Next()

	}
}

func deleteCommentHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {

		err := federation.CreateCommentDeleteActivity(e.App, client, e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	}
}

func createTrailShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		record := e.Record

		trailId := record.GetString("trail")
		shares, err := e.App.FindAllRecords("trail_share",
			dbx.NewExp("trail = {:trailId}", dbx.Params{"trailId": trailId}),
		)
		if err != nil {
			return err
		}
		actorIds := make([]string, len(shares))
		for i, r := range shares {
			actorIds[i] = r.GetString("actor")
		}
		err = util.UpdateTrailShares(trailId, actorIds, client)
		if err != nil {
			return err
		}

		err = federation.CreateAnnounceActivity(e.App, record, federation.TrailAnnounceType)
		if err != nil {
			return err
		}

		return nil
	}
}

func deleteTrailShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		record := e.Record

		trailId := record.GetString("trail")
		err := util.UpdateTrailShares(trailId, []string{}, client)
		if err != nil {
			return err
		}
		return e.Next()
	}
}

func createTrailLikeHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		record := e.Record

		trailId := record.GetString("trail")
		actorId := record.GetString("actor")
		actor, err := e.App.FindRecordById("activitypub_actors", actorId)
		if err != nil {
			return err
		}
		trail, err := e.App.FindRecordById("trails", trailId)
		if err != nil {
			return err
		}
		likes, err := e.App.FindAllRecords("trail_like",
			dbx.NewExp("trail = {:trailId}", dbx.Params{"trailId": trailId}),
		)
		if err != nil {
			return err
		}

		trail.Set("like_count", len(likes))
		err = e.App.UnsafeWithoutHooks().Save(trail)
		if err != nil {
			return err
		}

		actorIds := make([]string, len(likes))
		for i, r := range likes {
			actorIds[i] = r.GetString("actor")
		}
		err = util.UpdateTrailLikes(trailId, actorIds, client)
		if err != nil {
			return err
		}

		if !actor.GetBool("isLocal") {
			// this happens if someone likes a remote trail
			// we create a local copy
			// no need to create an activity for that
			return nil
		}

		err = federation.CreateLikeActivity(e.App, record)
		if err != nil {
			return err
		}

		return nil
	}
}

func deleteTrailLikeHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {

		record := e.Record

		trailId := record.GetString("trail")
		actorId := record.GetString("actor")
		actor, err := e.App.FindRecordById("activitypub_actors", actorId)
		if err != nil {
			return err
		}
		// trail might deleted be already if this is called as part of a cascade
		trail, err := e.App.FindRecordById("trails", trailId)
		if err != nil && err == sql.ErrNoRows {
			return nil
		} else if err != nil {
			return err
		}
		likes, err := e.App.CountRecords("trail_like", dbx.NewExp("trail={:trail}", dbx.Params{"trail": trailId}))
		if err != nil {
			return err
		}

		trail.Set("like_count", likes)
		err = e.App.UnsafeWithoutHooks().Save(trail)
		if err != nil {
			return err
		}

		err = util.UpdateTrailLikes(trailId, []string{}, client)
		if err != nil {
			return err
		}

		if !actor.GetBool("isLocal") {
			// this happens if someone likes a remote trail
			// we create a local copy
			// no need to create an activity for that
			return nil
		}

		err = federation.CreateUnlikeActivity(e.App, record)
		if err != nil {
			return err
		}

		return e.Next()
	}
}

func createListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record

		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}

		if err := util.IndexLists(e.App, []*core.Record{record}, client); err != nil {
			return err
		}

		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateListActivity(e.App, e.Record, pub.CreateType)
		if err != nil {
			return err
		}

		_, err = util.InsertIntoFeed(e.App, author.Id, author.Id, record.Id, util.ListFeed)
		if err != nil {
			return err
		}

		return nil
	}
}

func updateListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		author, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}

		err = util.UpdateList(e.App, record, author, client)
		if err != nil {
			return err
		}

		if !author.GetBool("isLocal") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		err = federation.CreateListActivity(e.App, e.Record, pub.CreateType)
		if err != nil {
			return err
		}

		return nil
	}
}

func deleteListHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		_, err := client.Index("lists").DeleteDocument(record.Id)
		if err != nil {
			return err
		}

		err = federation.CreateListDeleteActivity(e.App, record)
		if err != nil {
			return err
		}

		err = util.DeleteFromFeed(e.App, record.Id)
		if err != nil {
			return err
		}

		return e.Next()
	}
}

func createListShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		record := e.Record
		listId := record.GetString("list")
		shares, err := e.App.FindAllRecords("list_share",
			dbx.NewExp("list = {:listId}", dbx.Params{"listId": listId}),
		)
		if err != nil {
			return err
		}
		actorIds := make([]string, len(shares))
		for i, r := range shares {
			actorIds[i] = r.GetString("actor")
		}
		err = util.UpdateListShares(listId, actorIds, client)

		if err != nil {
			return err
		}

		err = federation.CreateAnnounceActivity(e.App, record, federation.ListAnnounceType)
		if err != nil {
			return err
		}

		return nil
	}
}

func deleteListShareHandler(client meilisearch.ServiceManager) func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		record := e.Record
		listId := record.GetString("list")
		err := util.UpdateListShares(listId, []string{}, client)
		if err != nil {
			return err
		}
		return e.Next()
	}
}

func createFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		e.Next()
		federation.CreateFollowActivity(e.App, e.Record)

		return nil
	}
}

func deleteFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		federation.CreateUnfollowActivity(e.App, e.Record)

		return e.Next()
	}
}

func listIntegrationHandler() func(e *core.RecordsListRequestEvent) error {
	return func(e *core.RecordsListRequestEvent) error {
		if e.HasSuperuserAuth() {
			return e.Next()
		}
		for _, r := range e.Records {

			err := censorIntegrationSecrets(r)
			if err != nil {
				return err
			}
		}

		return e.Next()
	}
}

func createIntegrationHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := encryptIntegrationSecrets(e.App, e.Record)
		if err != nil {
			return err
		}

		return e.Next()
	}
}

func createUpdateIntegrationSuccessHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := censorIntegrationSecrets(e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	}
}

func updateIntegrationHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		err := encryptIntegrationSecrets(e.App, e.Record)
		if err != nil {
			return err
		}

		return e.Next()
	}
}
func censorIntegrationSecrets(r *core.Record) error {
	secrets := map[string][]string{
		"strava": {"clientSecret", "refreshToken", "accessToken", "expiresAt"},
		"komoot": {"password"},
	}
	for key, secretKeys := range secrets {
		if integrationString := r.GetString(key); integrationString != "" {
			var integration map[string]interface{}
			if err := json.Unmarshal([]byte(integrationString), &integration); err != nil {
				return err
			}
			if integration == nil {
				continue
			}
			for _, secretKey := range secretKeys {
				integration[secretKey] = ""
			}
			b, err := json.Marshal(integration)
			if err != nil {
				return err
			}
			r.Set(key, string(b))
		}
	}

	return nil
}

func encryptIntegrationSecrets(app core.App, r *core.Record) error {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
	}

	secrets := map[string][]string{
		"strava": {"clientSecret", "refreshToken", "accessToken", "expiresAt"},
		"komoot": {"password"},
	}

	original, _ := app.FindRecordById("integrations", r.Id)

	for key, secretKeys := range secrets {
		if integrationString := r.GetString(key); integrationString != "" {
			var integration map[string]interface{}
			if err := json.Unmarshal([]byte(integrationString), &integration); err != nil {
				return err
			}

			for _, secretKey := range secretKeys {
				// If the secret is already encrypted, we don't re-encrypt it.
				// TODO: This is a bit of a hack, we should handle this in a more robust way (e.g.
				// storing flag on the record or prefixing encrypted strings with enc: or smilar).
				// Doing that would also potentially allow us to support key rotation in the future.
				if secret, ok := integration[secretKey].(string); ok && len(secret) > 0 && !util.CanDecryptSecret(secret) {
					encryptedSecret, err := security.Encrypt([]byte(secret), encryptionKey)
					if err != nil {
						return err
					}
					integration[secretKey] = encryptedSecret
				} else if original != nil {

					originalString := original.GetString(key)
					var originalIntegration map[string]interface{}
					if err := json.Unmarshal([]byte(originalString), &originalIntegration); err != nil {
						return err
					}
					if integration == nil {
						continue
					}
					integration[secretKey] = originalIntegration[secretKey]
				}
			}

			b, err := json.Marshal(integration)
			if err != nil {
				return err
			}
			r.Set(key, string(b))
		}
	}

	return nil
}

func changeUserEmailHandler() func(e *core.RecordRequestEmailChangeRequestEvent) error {
	return func(e *core.RecordRequestEmailChangeRequestEvent) error {

		e.Record.Set("email", e.NewEmail)
		if err := e.App.Save(e.Record); err != nil {
			return err
		}
		return nil
	}
}

func listFeedHandler() func(e *core.RecordsListRequestEvent) error {
	return func(e *core.RecordsListRequestEvent) error {

		for _, r := range e.Records {
			var item *core.Record
			var err error

			typ := r.GetString("type")
			typ = strings.Trim(typ, "\"")

			itemId := r.GetString("item")
			itemId = strings.Trim(itemId, "\"")

			switch typ {
			case string(util.TrailFeed):
				item, err = e.App.FindRecordById("trails", itemId)
			case string(util.ListFeed):
				item, err = e.App.FindRecordById("lists", itemId)
			case string(util.SummitLogFeed):
				item, err = e.App.FindRecordById("summit_logs", itemId)
			}

			if err != nil {
				continue
			}

			if item == nil {
				continue
			}

			errs := e.App.ExpandRecord(item, []string{"author"}, nil)
			if len(errs) > 0 {
				return fmt.Errorf("failed to expand author: %v", errs)
			}

			if typ == string(util.TrailFeed) {
				errs := e.App.ExpandRecord(item, []string{"category"}, nil)
				if len(errs) > 0 {
					return fmt.Errorf("failed to expand category: %v", errs)
				}
			}

			r.MergeExpand(map[string]any{"item": item})
		}
		return e.Next()
	}
}

func onBeforeServeHandler(client meilisearch.ServiceManager) func(se *core.ServeEvent) error {
	return func(se *core.ServeEvent) error {
		registerRoutes(se, client)
		registerCronJobs(se.App)
		bootstrapData(se.App, client)

		return se.Next()
	}

}

func onBootstrapHandler() func(se *core.BootstrapEvent) error {
	return func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		if e.App.Settings().Meta.AppName == "Acme" {
			e.App.Settings().Meta.AppName = "wanderer"
		}
		if v := os.Getenv("ORIGIN"); v != "" {
			e.App.Settings().Meta.AppURL = v
		}
		if v := cmp.Or(os.Getenv("POCKETBASE_SMTP_SENDER_ADDRESS"), os.Getenv("POCKETBASE_SMTP_SENDER_ADRESS")); v != "" {
			e.App.Settings().Meta.SenderAddress = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_SENDER_NAME"); v != "" {
			e.App.Settings().Meta.SenderName = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_ENABLED"); v != "" {
			e.App.Settings().SMTP.Enabled = cast.ToBool(v)
		}
		if v := os.Getenv("POCKETBASE_SMTP_HOST"); v != "" {
			e.App.Settings().SMTP.Host = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_PORT"); v != "" {
			e.App.Settings().SMTP.Port = cast.ToInt(v)
		}
		if v := os.Getenv("POCKETBASE_SMTP_USERNAME"); v != "" {
			e.App.Settings().SMTP.Username = v
		}
		if v := os.Getenv("POCKETBASE_SMTP_PASSWORD"); v != "" {
			e.App.Settings().SMTP.Password = v
		}

		return e.App.Save(e.App.Settings())
	}
}

func registerRoutes(se *core.ServeEvent, client meilisearch.ServiceManager) {
	se.Router.GET("/health", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	se.Router.GET("/public/search/token", func(e *core.RequestEvent) error {
		searchRules := map[string]interface{}{
			"lists": map[string]string{
				"filter": "public = true",
			},
			"trails": map[string]string{
				"filter": "public = true",
			},
		}
		token, err := util.GenerateMeilisearchToken(searchRules, client)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]string{"token": token})
	})

	se.Router.POST("/integration/strava/token", func(e *core.RequestEvent) error {
		encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
		if len(encryptionKey) == 0 {
			return apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
		}

		var data strava.TokenRequest
		if err := e.BindBody(&data); err != nil {
			return apis.NewBadRequestError("Failed to read request data", err)
		}

		userId := ""
		if e.Auth != nil {
			userId = e.Auth.Id
		}

		integrations, err := e.App.FindAllRecords("integrations", dbx.NewExp("user = {:id}", dbx.Params{"id": userId}))
		if err != nil {
			return err
		}
		if len(integrations) == 0 {
			return apis.NewBadRequestError("user has no integration", nil)
		}
		integration := integrations[0]
		stravaString := integration.GetString("strava")
		if len(stravaString) == 0 {
			return apis.NewBadRequestError("strava integration missing", nil)
		}
		var stravaIntegration strava.StravaIntegration
		err = json.Unmarshal([]byte(stravaString), &stravaIntegration)
		if err != nil {
			return err
		}
		decryptedSecret, err := security.Decrypt(stravaIntegration.ClientSecret, encryptionKey)
		if err != nil {
			return err
		}

		request := strava.TokenRequest{
			ClientID:     stravaIntegration.ClientID,
			ClientSecret: string(decryptedSecret),
			Code:         data.Code,
			GrantType:    "authorization_code",
		}
		r, err := strava.GetStravaToken(request)
		if err != nil {
			return err
		}
		if r.AccessToken != "" {
			stravaIntegration.AccessToken = r.AccessToken
		}
		if r.RefreshToken != "" {
			stravaIntegration.RefreshToken = r.RefreshToken
		}
		if r.AccessToken != "" {
			stravaIntegration.ExpiresAt = r.ExpiresAt
		}

		stravaIntegration.Active = true

		b, err := json.Marshal(stravaIntegration)
		if err != nil {
			return err
		}
		integration.Set("strava", string(b))
		err = e.App.Save(integration)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, nil)
	})

	se.Router.GET("/integration/komoot/login", func(e *core.RequestEvent) error {
		encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
		if len(encryptionKey) == 0 {
			return apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
		}

		userId := ""
		if e.Auth != nil {
			userId = e.Auth.Id
		}

		integrations, err := e.App.FindAllRecords("integrations", dbx.NewExp("user = {:id}", dbx.Params{"id": userId}))
		if err != nil {
			return err
		}
		if len(integrations) == 0 {
			return apis.NewBadRequestError("user has no integration", nil)
		}
		integration := integrations[0]
		komootString := integration.GetString("komoot")
		if len(komootString) == 0 {
			return apis.NewBadRequestError("komoot integration missing", nil)
		}
		var komootIntegration komoot.KomootIntegration
		err = json.Unmarshal([]byte(komootString), &komootIntegration)
		if err != nil {
			return err
		}
		decryptedPassword, err := security.Decrypt(komootIntegration.Password, encryptionKey)
		if err != nil {
			return err
		}

		k := &komoot.KomootApi{}

		err = k.Login(komootIntegration.Email, string(decryptedPassword))
		if err != nil {
			return apis.NewUnauthorizedError("invalid credentials", nil)
		}

		return e.JSON(http.StatusOK, nil)
	})
	se.Router.POST("/activitypub/activity/process", federation.ProcessActivity)
	se.Router.GET("/activitypub/actor", func(e *core.RequestEvent) error {
		resource := e.Request.URL.Query().Get("resource")
		resource = strings.TrimPrefix(resource, "acct:")

		iri := e.Request.URL.Query().Get("iri")
		follows := e.Request.URL.Query().Get("follows") == "true"

		var userActor *core.Record
		var err error
		if e.Auth != nil {
			userActor, err = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
			if err != nil {
				return err
			}
		}

		var actor *core.Record
		if resource != "" {
			actor, err = federation.GetActorByHandle(e.App, userActor, resource, follows)
		} else {
			actor, err = federation.GetActorByIRI(e.App, userActor, iri, follows)
		}
		if err != nil && actor == nil {
			if strings.HasPrefix(err.Error(), "webfinger") {
				return e.NotFoundError("Not found", err)
			}
			return err
		} else if err != nil && actor != nil {
			if errors.Is(err, federation.ErrProfilePrivate) {
				// this is our own profile
				if e.Auth != nil && actor.GetString("user") == e.Auth.Id {
					return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": nil})
				} else {
					return e.JSON(http.StatusNotFound, map[string]any{"error": "profile is private"})
				}
			}
			// we could not fetch the remote actor so we return our local cached copy
			return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{"actor": actor, "error": nil})
	})
	se.Router.GET("/activitypub/actor/{id}/{follow}", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		followType := e.Request.PathValue("follow")
		page := e.Request.URL.Query().Get("page")
		intPage := 0

		if page != "" {
			var err error
			intPage, err = strconv.Atoi(page)
			if err != nil {
				return err
			}
		}

		actor, err := e.App.FindRecordById("activitypub_actors", id)
		if err != nil {
			return err
		}

		var userActor *core.Record
		if e.Auth != nil {
			userActor, err = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
			if err != nil {
				return err
			}
		}

		url := actor.GetString(followType)

		if url == "" {
			return e.BadRequestError("unknown type: "+followType, nil)
		}
		collection, err := federation.FetchCollection(userActor, fmt.Sprintf("%s?page=%d", url, intPage))
		if err != nil {
			if errors.Is(err, federation.ErrProfilePrivate) {
				return e.JSON(http.StatusNotFound, map[string]any{"error": "profile is private"})
			}
			return err
		}
		return e.JSON(http.StatusOK, collection)
	})
	se.Router.GET("/activitypub/trail/{id}", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		trail, err := e.App.FindRecordById("trails", id)
		if err != nil {
			return err
		}

		trailObject, err := util.ObjectFromTrail(e.App, trail, nil)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, trailObject)
	})
	se.Router.GET("/activitypub/comment/{id}", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		comment, err := e.App.FindRecordById("comments", id)
		if err != nil {
			return err
		}

		commentObject, err := util.ObjectFromComment(e.App, comment, nil)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, commentObject)
	})
}

func registerCronJobs(app core.App) {
	schedule := os.Getenv("POCKETBASE_CRON_SYNC_SCHEDULE")
	if len(schedule) == 0 {
		schedule = "0 2 * * *"
	}

	app.Cron().MustAdd("integrations", schedule, func() {
		err := strava.SyncStrava(app)
		if err != nil {
			warning := fmt.Sprintf("Error syncing with strava: %v", err)
			fmt.Println(warning)
			app.Logger().Error(warning)
		}
		err = komoot.SyncKomoot(app)
		if err != nil {
			warning := fmt.Sprintf("Error syncing with komoot: %v", err)
			fmt.Println(warning)
			app.Logger().Error(warning)
		}
	})
}

func bootstrapData(app core.App, client meilisearch.ServiceManager) error {
	bootstrapCategories(app)
	go bootstrapMeilisearchDocuments(app, client)
	return nil
}

func bootstrapCategories(app core.App) error {
	query := app.RecordQuery("categories")
	records := []*core.Record{}

	if err := query.All(&records); err != nil {
		return err
	}
	if len(records) == 0 {
		collection, _ := app.FindCollectionByNameOrId("categories")

		categories := []string{"Hiking", "Walking", "Climbing", "Skiing", "Canoeing", "Biking"}
		for _, element := range categories {
			record := core.NewRecord(collection)
			record.Set("name", element)
			f, _ := filesystem.NewFileFromPath("migrations/initial_data/" + strings.ToLower(element) + ".jpg")
			record.Set("img", f)
			err := app.Save(record)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func bootstrapMeilisearchDocuments(app core.App, client meilisearch.ServiceManager) error {
	// --- Trails ---
	const pageSize int64 = 100
	var page int64 = 0

	// Clear index before re-indexing
	if _, err := client.Index("trails").DeleteAllDocuments(); err != nil {
		return err
	}

	for {
		trails := []*core.Record{}
		err := app.RecordQuery("trails").
			Limit(pageSize).
			Offset(page * pageSize).
			All(&trails)
		if err != nil {
			return err
		}
		if len(trails) == 0 {
			break
		}

		if err := util.IndexTrails(app, trails, client); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to index trails page %d: %v", page, err))
			continue
		}

		page++
	}

	// --- Lists ---
	if _, err := client.Index("lists").DeleteAllDocuments(); err != nil {
		return err
	}

	page = 0
	for {
		lists := []*core.Record{}
		err := app.RecordQuery("lists").
			Limit(pageSize).
			Offset(page * pageSize).
			All(&lists)
		if err != nil {
			return err
		}
		if len(lists) == 0 {
			break
		}

		if err := util.IndexLists(app, lists, client); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to index list page %d: %v", page, err))
			continue
		}

		page++
	}

	return nil
}
