package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"pocketbase/commands"
	"pocketbase/hooks"
	"pocketbase/pluginsystem"
	"pocketbase/routes"
	"pocketbase/services/regions"

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
	if len(os.Args) > 1 && os.Args[1] == "plugin-worker" {
		os.Exit(pluginsystem.RunPluginWorker(context.Background(), os.Stdin, os.Stdout, os.Stderr))
	}

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

	app.OnRecordAfterCreateSuccess("activitypub_actors").BindFunc(hooks.CreateActorHandler(client))
	app.OnRecordAfterUpdateSuccess("activitypub_actors").BindFunc(hooks.UpdateActorHandler(client))
	app.OnRecordAfterDeleteSuccess("activitypub_actors").BindFunc(hooks.DeleteActorHandler(client))

	app.OnRecordCreateRequest("categories").BindFunc(hooks.ValidateCategoryHandler())
	app.OnRecordUpdateRequest("categories").BindFunc(hooks.ValidateCategoryHandler())
	app.OnRecordAfterCreateSuccess("categories").BindFunc(hooks.BackfillRemoteTrailCategoryHandler())
	app.OnRecordAfterUpdateSuccess("categories").BindFunc(hooks.BackfillRemoteTrailCategoryHandler())
	app.OnRecordCreateRequest("subcategories").BindFunc(hooks.ValidateSubcategoryHandler())
	app.OnRecordUpdateRequest("subcategories").BindFunc(hooks.ValidateSubcategoryHandler())
	app.OnRecordAfterCreateSuccess("subcategories").BindFunc(hooks.BackfillRemoteTrailSubcategoryHandler())
	app.OnRecordAfterUpdateSuccess("subcategories").BindFunc(hooks.BackfillRemoteTrailSubcategoryHandler())

	app.OnRecordCreateRequest("user_category_preferences").BindFunc(hooks.ValidateUserCategoryPreferenceHandler())
	app.OnRecordUpdateRequest("user_category_preferences").BindFunc(hooks.ValidateUserCategoryPreferenceHandler())
	app.OnRecordCreateRequest("user_subcategory_preferences").BindFunc(hooks.ValidateUserSubcategoryPreferenceHandler())
	app.OnRecordUpdateRequest("user_subcategory_preferences").BindFunc(hooks.ValidateUserSubcategoryPreferenceHandler())

	app.OnRecordCreateRequest("trails").BindFunc(hooks.ValidateTrailSubcategoryHandler())
	app.OnRecordUpdateRequest("trails").BindFunc(hooks.ValidateTrailSubcategoryHandler())
	app.OnRecordAfterCreateSuccess("trails").BindFunc(hooks.CreateTrailHandler(client))
	app.OnRecordAfterUpdateSuccess("trails").BindFunc(hooks.UpdateTrailHandler(client))
	app.OnRecordAfterDeleteSuccess("trails").BindFunc(hooks.DeleteTrailHandler(client))

	app.OnRecordCreateRequest("summit_logs").BindFunc(hooks.CreateSummitLogHandler(client))
	app.OnRecordUpdateRequest("summit_logs").BindFunc(hooks.UpdateSummitLogHandler())
	app.OnRecordDeleteRequest("summit_logs").BindFunc(hooks.DeleteSummitLogHandler(client))

	app.OnRecordCreateRequest("waypoints").BindFunc(hooks.CreateWaypointHandler())

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

	app.OnRecordsListRequest("plugin_instances").BindFunc(hooks.ListPluginInstanceHandler())
	app.OnRecordViewRequest("plugin_instances").BindFunc(hooks.ViewPluginInstanceHandler())
	app.OnRecordCreate("plugin_instances").BindFunc(hooks.CreatePluginInstanceHandler())
	app.OnRecordAfterCreateSuccess("plugin_instances").BindFunc(hooks.CreateUpdatePluginInstanceSuccessHandler())
	app.OnRecordUpdate("plugin_instances").BindFunc(hooks.UpdatePluginInstanceHandler())
	app.OnRecordAfterUpdateSuccess("plugin_instances").BindFunc(hooks.CreateUpdatePluginInstanceSuccessHandler())

	app.OnRecordsListRequest("feed", "profile_feed").BindFunc(hooks.ListFeedHandler())

	app.OnRecordCreate("api_tokens").BindFunc(hooks.CreateAPITokenHandler())

	// Path-based referential integrity: region_archives/region_geometry rows
	// must reference an existing regions.path (there is no PocketBase relation
	// between them — the join is on the stable path natural key). Model hooks
	// (not *Request) so internal builder/geometry-fetch writes are covered too.
	app.OnRecordCreate("region_archives", "region_geometry").BindFunc(hooks.ValidateRegionPathReferenceHandler())
	app.OnRecordUpdate("region_archives", "region_geometry").BindFunc(hooks.ValidateRegionPathReferenceHandler())

	// Cache a leaf's boundary geometry as soon as it is enabled, from whichever
	// path did the enabling (admin picker, REST PATCH, collection editor). Runs
	// post-commit so ResolveGeometry reads back an enabled record and takes its
	// persist branch; the upstream fetch happens off the request goroutine.
	app.OnRecordAfterUpdateSuccess("regions").BindFunc(hooks.CacheGeometryOnEnableHandler(app))

	app.OnRecordCreateRequest().BindFunc(util.SanitizeHTML())
	app.OnRecordUpdateRequest().BindFunc(util.SanitizeHTML())

	app.OnServe().BindFunc(onBeforeServeHandler(client))

	app.OnBootstrap().BindFunc(hooks.OnBootstrapHandler())
}

func setupCommands(app *pocketbase.PocketBase) {
	app.RootCmd.AddCommand(commands.Dedup(app))
	app.RootCmd.AddCommand(commands.SeedRegions())
}

func onBeforeServeHandler(client meilisearch.ServiceManager) func(se *core.ServeEvent) error {
	return func(se *core.ServeEvent) error {
		registerRoutes(se, client)
		registerCronJobs(se.App, client)
		initData(se.App, client)

		// Startup GC: drop region_archives rows whose backing pmtiles files
		// have vanished from disk (manual deletion / wiped cache volume). Never
		// fatal — a failure here must not block serving.
		if err := regions.ReconcileArchives(se.App); err != nil {
			se.App.Logger().Warn("failed to reconcile region archives on startup", "error", err)
		}

		return se.Next()
	}

}

func registerRoutes(se *core.ServeEvent, client meilisearch.ServiceManager) {
	se.Router.GET("/health", routes.Health)

	se.Router.POST("/auth/token", routes.AuthToken)
	se.Router.POST("/user/email", routes.UserEmailChange)
	se.Router.POST("/waypoint/cluster", routes.WaypointCluster)
	se.Router.POST("/category-preferences/reorder", routes.CategoryPreferencesReorder)
	se.Router.POST("/subcategory-preferences/reorder", routes.SubcategoryPreferencesReorder)

	se.Router.POST("/trail-merge/suggest", routes.TrailMergeSuggest)
	se.Router.POST("/trail-merge", routes.TrailMerge(client))

	se.Router.GET("/search/token", routes.SearchToken(client))

	se.Router.GET("/plugins", routes.PluginSystemPluginsList)
	se.Router.POST("/plugins/trail-send", routes.PluginSystemTrailSend)
	se.Router.POST("/plugins/auth/validate", routes.PluginSystemSessionAuthValidate)
	se.Router.POST("/plugins/category-remap/preview", routes.PluginSystemCategoryRemapPreview)
	se.Router.POST("/plugins/category-remap/apply", routes.PluginSystemCategoryRemapApply)
	se.Router.POST("/plugins/oauth/start", routes.PluginSystemOAuthStart)
	se.Router.POST("/plugins/oauth/callback", routes.PluginSystemOAuthCallback)
	se.Router.POST("/plugins/oauth/revoke", routes.PluginSystemOAuthRevoke)

	se.Router.POST("/activitypub/activity/process", routes.ActivitypubActivityProcess)
	se.Router.GET("/activitypub/actor", routes.ActivitypubActor)
	se.Router.GET("/activitypub/actor/{id}/{follow}", routes.ActivitypubActorFollow)
	se.Router.GET("/activitypub/trail/{id}", routes.ActivitypubTrail)
	se.Router.GET("/activitypub/comment/{id}", routes.ActivitypubComment)

	se.Router.GET("/remote/trail/{id}", routes.RemoteTrailGet)
	se.Router.GET("/remote/trail/{id}/comments", routes.RemoteTrailCommentsList)

	se.Router.GET("/remote/list/{id}", routes.RemoteListGet)

	se.Router.GET("/remote/profile/{handle}/follows", routes.RemoteProfileFollowsList)

	// Custom PocketBase admin extension: a standalone, superuser-gated page
	// at a distinct top-level path (NOT nested under /regions below, which
	// is bound to apis.RequireAuth() for any logged-in user). Auth is enforced
	// entirely by the page's own content: it reads the PocketBase
	// dashboard's own superuser JWT from localStorage and talks directly
	// to PocketBase's built-in collection REST API, which is superuser-
	// only by default on regions/region_geometry.
	se.Router.GET("/region-catalog/", routes.RegionsDashboard)

	// Destructive, admin-only: unlike the page above (which relies on the
	// PocketBase collection API's own superuser-only rules), this is a
	// custom Go route with no collection rules of its own, so it must
	// enforce superuser auth explicitly.
	se.Router.DELETE("/region-catalog/{id}/archive", routes.RegionArchiveDelete).Bind(apis.RequireSuperuserAuth())

	// "Sync now": manually trigger the same BuildAll pass the nightly cron
	// runs, plus a status check the admin page polls. Same superuser-only
	// posture as the delete route above, and for the same reason (custom Go
	// routes with no collection rules of their own).
	se.Router.POST("/region-catalog/sync", routes.RegionSyncStart).Bind(apis.RequireSuperuserAuth())
	se.Router.GET("/region-catalog/sync", routes.RegionSyncStatus).Bind(apis.RequireSuperuserAuth())

	se.UIExtensions = append(se.UIExtensions, core.UIExtension{
		Name: "wanderer-region-catalog",
		FS:   routes.RegionsExtFS(),
	})

	// /regions is an internal-only contract: the literal /api/v1 prefix
	// deviates from every other unprefixed custom Go route in this file. It
	// is reachable only from inside the docker network — a SvelteKit proxy
	// under the same public path forwards external requests to it
	// (web/src/routes/regions/**). Auth is ENABLED: any logged-in user, for
	// both the catalog listing and the archive downloads.
	regionsGroup := se.Router.Group("/regions")
	regionsGroup.Bind(apis.RequireAuth())

	regionsGroup.GET("", routes.RegionsList)
	regionsGroup.GET("/{id}/download", routes.RegionArchiveDownload)
	regionsGroup.GET("/{id}/download-dem", routes.RegionArchiveDownloadDem)

	// Standalone, NOT a member of regionsGroup above: this route triggers
	// outbound third-party requests (CoMaps, via ResolveGeometry), so an
	// authenticated-user gate would make it both an open proxy and a way to
	// burn CoMaps' rate limit from outside. It lives under /regions
	// only for URL coherence with its siblings — not because it belongs to
	// the group's weaker RequireAuth() trust class. Mirrors the standalone
	// superuser-bound registrations above (RegionArchiveDelete,
	// RegionSyncStart, RegionSyncStatus).
	se.Router.GET("/regions/{id}/geometry", routes.RegionGeometryGet).Bind(apis.RequireSuperuserAuth())
}

func registerCronJobs(app core.App, client meilisearch.ServiceManager) {
	schedule := os.Getenv("POCKETBASE_CRON_SYNC_SCHEDULE")
	if len(schedule) == 0 {
		schedule = "0 2 * * *"
	}

	app.Cron().MustAdd("plugin-sync", schedule, func() {
		if err := routes.PluginSystemSyncConfigured(context.Background(), app, client); err != nil {
			warning := fmt.Sprintf("Error syncing with WASM plugins: %v", err)
			fmt.Println(warning)
			app.Logger().Error(warning)
		}
	})

	app.Cron().MustAdd("region-archive-build", regions.CronSchedule(), func() {
		regions.BuildAll(app)
	})
}

func initData(app core.App, client meilisearch.ServiceManager) error {
	initCategories(app)
	if err := util.SeedDefaultSubcategories(app); err != nil {
		return err
	}
	initPlugins(app)
	initMeilisearchConfig(client)
	go func() {
		backfillPolylines(app)
		initMeilisearchDocuments(app, client)
	}()
	return nil
}

func initPlugins(app core.App) {
	manager := pluginsystem.NewManager(app, "")
	if err := manager.SyncInstalledPlugins(context.Background()); err != nil {
		warning := fmt.Sprintf("Error discovering WASM plugins: %v", err)
		fmt.Println(warning)
		app.Logger().Error(warning)
	}
}

func backfillPolylines(app core.App) {
	const pageSize int64 = 100
	var lastID string
	var processed int
	var failed int

	log.Printf("backfill polyline started")
	defer func() {
		log.Printf("backfill polyline completed: processed=%d failed=%d", processed, failed)
	}()

	for {
		trails := []*core.Record{}
		query := app.RecordQuery("trails").
			AndWhere(dbx.NewExp("(polyline IS NULL OR polyline = '') AND gpx != ''")).
			OrderBy("id ASC").
			Limit(pageSize)

		if lastID != "" {
			query = query.AndWhere(dbx.NewExp("id > {:lastID}", dbx.Params{"lastID": lastID}))
		}

		err := query.All(&trails)
		if err != nil {
			log.Printf("backfill polyline query failed after trail %q: %v", lastID, err)
			break
		}
		if len(trails) == 0 {
			break
		}
		for _, r := range trails {
			if err := util.SavePolyline(app, r); err != nil {
				failed++
				log.Printf("backfill polyline failed for trail %s (%q), gpx=%q: %v", r.Id, r.GetString("name"), r.GetString("gpx"), err)
			} else {
				processed++
			}
			lastID = r.Id
		}
	}
}

func initCategories(app core.App) error {
	query := app.RecordQuery("categories")
	records := []*core.Record{}

	if err := query.All(&records); err != nil {
		return err
	}
	collection, err := app.FindCollectionByNameOrId("categories")
	if err != nil {
		return err
	}

	if len(records) == 0 {
		for _, element := range util.DefaultCategoryNames() {
			record := core.NewRecord(collection)
			record.Set("name", element)
			record.Set("settings", map[string]any{
				"wp_merge_enabled": true,
				"wp_merge_radius":  50,
			})
			err := app.Save(record)
			if err != nil {
				return err
			}
		}
	}
	if err := util.PrepopulateDefaultCategoryTranslations(app); err != nil {
		return err
	}
	if err := util.PrepopulateDefaultCategoryIcons(app); err != nil {
		return err
	}
	return util.PrepopulateDefaultCategoryValhallaProfiles(app)
}

func initMeilisearchConfig(client meilisearch.ServiceManager) {
	configs := map[string]meilisearch.Settings{
		"trails": {
			SearchableAttributes: []string{"author_name", "name", "description", "location", "tags"},
			FilterableAttributes: []string{
				"id", "_geo", "author", "category_id", "subcategory_id",
				"is_federated", "completed", "date", "difficulty", "distance",
				"elevation_gain", "elevation_loss", "likes", "public", "shares",
				"tags", "min_lat", "max_lat", "min_lon", "max_lon", "bounding_box_diagonal",
			},
			SortableAttributes: []string{
				"author", "created", "date", "difficulty", "distance",
				"duration", "elevation_gain", "elevation_loss", "like_count", "name",
				"min_lat", "max_lat", "min_lon", "max_lon",
			},
			RankingRules: []string{"words", "typo", "proximity", "attribute", "sort", "exactness"},
		},
		"lists": {
			SearchableAttributes: []string{"*"},
			FilterableAttributes: []string{"author", "public", "shares"},
			SortableAttributes:   []string{"created", "name"},
			RankingRules:         []string{"words", "typo", "proximity", "attribute", "sort", "exactness"},
		},
		"actors": {
			SearchableAttributes: []string{"username", "preferred_username", "domain"},
			FilterableAttributes: []string{"id"},
			SortableAttributes:   []string{},
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

	// --- Actors ---
	if _, err := client.Index("actors").DeleteAllDocuments(nil); err != nil {
		return err
	}

	page = 0
	for {
		actors := []*core.Record{}
		err := app.RecordQuery("activitypub_actors").
			Limit(pageSize).
			Offset(page * pageSize).
			All(&actors)
		if err != nil {
			return err
		}
		if len(actors) == 0 {
			break
		}

		if err := util.IndexActors(actors, client); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to index actor page %d: %v", page, err))
			continue
		}

		page++
	}

	return nil
}
