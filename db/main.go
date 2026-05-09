package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"pocketbase/commands"
	"pocketbase/hooks"
	"pocketbase/integrations/hammerhead"
	"pocketbase/integrations/komoot"
	"pocketbase/integrations/strava"
	"pocketbase/routes"

	_ "pocketbase/migrations"
	"pocketbase/util"
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
		log.Fatal("POCKETBASE_ENCRYPTION_KEY must be exactly 32 bytes long- See https://wanderer.to/run/installation/docker#prerequisites for more information")
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
	client := initializeMeilisearch()

	verifySettings(app)

	registerMigrations(app)
	setupEventHandlers(app, client)

	setupCommands(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

func initializeMeilisearch() meilisearch.ServiceManager {
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
	app.OnRecordAfterCreateSuccess("users").BindFunc(hooks.CreateUserHandler(client))
	app.OnRecordAfterUpdateSuccess("users").BindFunc(hooks.UpdateUserHandler(client))
	app.OnRecordRequestEmailChangeRequest("users").BindFunc(hooks.ChangeUserEmailHandler())

	app.OnRecordAfterCreateSuccess("trails").BindFunc(hooks.CreateTrailHandler(client))
	app.OnRecordAfterUpdateSuccess("trails").BindFunc(hooks.UpdateTrailHandler(client))
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(hooks.DeleteTrailHandler(client))

	app.OnRecordCreateRequest("summit_logs").BindFunc(hooks.CreateSummitLogHandler(client))
	app.OnRecordUpdateRequest("summit_logs").BindFunc(hooks.UpdateSummitLogHandler())
	app.OnRecordDeleteRequest("summit_logs").BindFunc(hooks.DeleteSummitLogHandler(client))

	app.OnRecordCreateRequest("comments").BindFunc(hooks.CreateCommentHandler())
	app.OnRecordUpdateRequest("comments").BindFunc(hooks.UpdateCommentHandler())
	app.OnRecordDeleteRequest("comments").BindFunc(hooks.DeleteCommentHandler(client))

	app.OnRecordCreateRequest("trail_share").BindFunc(hooks.CreateTrailShareHandler(client))
	app.OnRecordDeleteRequest("trail_share").BindFunc(hooks.DeleteTrailShareHandler(client))

	app.OnRecordAfterCreateSuccess("trail_like").BindFunc(hooks.CreateTrailLikeHandler(client))
	app.OnRecordAfterDeleteSuccess("trail_like").BindFunc(hooks.DeleteTrailLikeHandler(client))

	app.OnRecordAfterCreateSuccess("lists").BindFunc(hooks.CreateListHandler(client))
	app.OnRecordAfterUpdateSuccess("lists").BindFunc(hooks.UpdateListHandler(client))
	app.OnRecordAfterDeleteSuccess("lists").BindFunc(hooks.DeleteListHandler(client))

	app.OnRecordCreateRequest("list_share").BindFunc(hooks.CreateListShareHandler(client))
	app.OnRecordDeleteRequest("list_share").BindFunc(hooks.DeleteListShareHandler(client))

	app.OnRecordCreateRequest("follows").BindFunc(hooks.CreateFollowHandler())
	app.OnRecordDeleteRequest("follows").BindFunc(hooks.DeleteFollowHandler())

	app.OnRecordsListRequest("integrations").BindFunc(hooks.ListIntegrationHandler())
	app.OnRecordCreate("integrations").BindFunc(hooks.CreateIntegrationHandler())
	app.OnRecordAfterCreateSuccess("integrations").BindFunc(hooks.CreateUpdateIntegrationSuccessHandler())
	app.OnRecordUpdate("integrations").BindFunc(hooks.UpdateIntegrationHandler())
	app.OnRecordAfterUpdateSuccess("integrations").BindFunc(hooks.CreateUpdateIntegrationSuccessHandler())

	app.OnRecordsListRequest("feed", "profile_feed").BindFunc(hooks.ListFeedHandler())

	app.OnRecordCreate("api_tokens").BindFunc(hooks.CreateAPITokenHandler())

	app.OnRecordCreateRequest().BindFunc(util.SanitizeHTML())
	app.OnRecordUpdateRequest().BindFunc(util.SanitizeHTML())

	app.OnServe().BindFunc(onBeforeServeHandler(client))

	app.OnBootstrap().BindFunc(hooks.OnBootstrapHandler())
}

func setupCommands(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(commands.Dedup(app))
}

func onBeforeServeHandler(client meilisearch.ServiceManager) func(se *core.ServeEvent) error {
	return func(se *core.ServeEvent) error {
		registerRoutes(se, client)
		registerCronJobs(se.App, client)
		initData(se.App, client)

		return se.Next()
	}

}

func registerRoutes(se *core.ServeEvent, client meilisearch.ServiceManager) {
	se.Router.GET("/health", routes.Health)

	se.Router.POST("/auth/token", routes.AuthToken)
	se.Router.POST("/waypoint/cluster", routes.WaypointCluster)

	se.Router.POST("/trail-merge/suggest", routes.TrailMergeSuggest)
	se.Router.POST("/trail-merge", routes.TrailMerge(client))

	se.Router.GET("/search/token", routes.SearchToken(client))

	se.Router.POST("/integration/strava/token", routes.IntegrationStravaToken)
	se.Router.POST("/integration/hammerhead/upload", routes.IntegrationHammerheadUpload)
	se.Router.GET("/integration/hammerhead/login", routes.IntegrationHammerheadLogin)
	se.Router.GET("/integration/komoot/login", routes.IntegrationKommotLogin)

	se.Router.POST("/activitypub/activity/process", routes.ActivitypubActivityProcess)
	se.Router.GET("/activitypub/actor", routes.ActivitypubActor)
	se.Router.GET("/activitypub/actor/{id}/{follow}", routes.ActivitypubActorFollow)
	se.Router.GET("/activitypub/trail/{id}", routes.ActivitypubTrail)
	se.Router.GET("/activitypub/comment/{id}", routes.ActivitypubComment)

	se.Router.GET("/remote/trail/{id}", routes.RemoteTrailGet)
	se.Router.GET("/remote/trail/{id}/comments", routes.RemoteTrailCommentsList)

	se.Router.GET("/remote/list/{id}", routes.RemoteListGet)

	se.Router.GET("/remote/profile/{handle}/follows", routes.RemoteProfileFollowsList)

}

func registerCronJobs(app core.App, client meilisearch.ServiceManager) {
	schedule := os.Getenv("POCKETBASE_CRON_SYNC_SCHEDULE")
	if len(schedule) == 0 {
		schedule = "0 2 * * *"
	}

	app.Cron().MustAdd("integrations", schedule, func() {
		err := strava.SyncStrava(app, client)
		if err != nil {
			warning := fmt.Sprintf("Error syncing with strava: %v", err)
			fmt.Println(warning)
			app.Logger().Error(warning)
		}
		err = komoot.SyncKomoot(app, client)
		if err != nil {
			warning := fmt.Sprintf("Error syncing with komoot: %v", err)
			fmt.Println(warning)
			app.Logger().Error(warning)
		}
		err = hammerhead.SyncHammerhead(app, client)
		if err != nil {
			warning := fmt.Sprintf("Error syncing with hammerhead: %v", err)
			fmt.Println(warning)
			app.Logger().Error(warning)
		}
	})
}

func initData(app core.App, client meilisearch.ServiceManager) error {
	initCategories(app)
	initMeilisearchConfig(client)
	go initMeilisearchDocuments(app, client)
	return nil
}

func initCategories(app core.App) error {
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
			record.Set("settings", map[string]any{
				"wp_merge_enabled": true,
				"wp_merge_radius":  50,
			})
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

func initMeilisearchConfig(client meilisearch.ServiceManager) {
	configs := map[string]meilisearch.Settings{
		"trails": {
			SearchableAttributes: []string{"author_name", "name", "description", "location", "tags"},
			FilterableAttributes: []string{
				"_geo", "author", "category", "completed", "date", "difficulty",
				"distance", "elevation_gain", "elevation_loss", "likes", "public",
				"shares", "tags",
			},
			SortableAttributes: []string{
				"author", "created", "date", "difficulty", "distance",
				"duration", "elevation_gain", "elevation_loss", "like_count", "name",
			},
			RankingRules: []string{"words", "typo", "proximity", "attribute", "sort", "exactness"},
		},
		"lists": {
			SearchableAttributes: []string{"*"},
			FilterableAttributes: []string{"author", "public", "shares"},
			SortableAttributes:   []string{"created", "name"},
			RankingRules:         []string{"words", "typo", "proximity", "attribute", "sort", "exactness"},
		},
	}

	for indexName, settings := range configs {
		_, err := client.GetIndex(indexName)
		if err != nil {
			log.Printf("Index [%s] not found, creating it...", indexName)
			task, err := client.CreateIndex(&meilisearch.IndexConfig{
				Uid:        indexName,
				PrimaryKey: "id",
			})
			if err != nil {
				log.Printf("Failed to create index [%s]: %v", indexName, err)
				continue
			}

			_, err = client.WaitForTask(task.TaskUID, 0)
			if err != nil {
				log.Printf("Error waiting for index creation [%s]: %v", indexName, err)
				continue
			}
		}

		_, err = client.Index(indexName).UpdateSettings(&settings)
		if err != nil {
			log.Printf("Failed to sync settings for index [%s]: %v", indexName, err)
		} else {
			log.Printf("Settings synced for index [%s]", indexName)
		}
	}
}

func initMeilisearchDocuments(app core.App, client meilisearch.ServiceManager) error {
	// --- Trails ---
	const pageSize int64 = 100
	var page int64 = 0

	// Clear index before re-indexing
	if _, err := client.Index("trails").DeleteAllDocuments(nil); err != nil {
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
	if _, err := client.Index("lists").DeleteAllDocuments(nil); err != nil {
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
