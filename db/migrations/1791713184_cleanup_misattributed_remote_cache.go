package migrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func init() {
	m.Register(cleanupMisattributedRemoteCache1791713184, func(core.App) error {
		return fmt.Errorf("cleaned remote cache cannot be restored; restore the database and storage backup")
	})
}

func cleanupMisattributedRemoteCache1791713184(app core.App) (result error) {
	actors, err := app.FindAllRecords("activitypub_actors", dbx.HashExp{"is_local": true})
	if err != nil {
		return err
	}
	local := map[string]*core.Record{}
	ids := make([]string, 0, len(actors))
	for _, actor := range actors {
		local[actor.Id] = actor
		ids = append(ids, actor.Id)
	}
	if len(ids) == 0 {
		return nil
	}
	releaseIndexes, err := cleanupLookupIndexes1791713184(app)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, releaseIndexes()) }()
	app, err = newCoreOnlyCleanupApp1791713184(app)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	removed, preserved := 0, 0
	for _, collection := range []string{"trails", "lists", "summit_logs", "comments"} {
		cursor := ""
		for {
			records := []*core.Record{}
			err := app.RecordQuery(collection).AndWhere(dbx.NewExp(collection+".author IN (SELECT value FROM json_each({:actors})) AND "+collection+".id > {:cursor}", dbx.Params{"actors": string(encoded), "cursor": cursor})).OrderBy(collection + ".id").Limit(200).All(&records)
			if err != nil {
				return err
			}
			if len(records) == 0 {
				break
			}
			for _, record := range records {
				cursor = record.Id
				iri := record.GetString("iri")
				if iri == "" || isLocalIRI1791713184(iri) || iriHasRecordIdentity1791713184(iri, collection, record.Id) || historicalLocalActorOrigin1791713184(local[record.GetString("author")], iri) {
					continue
				}
				// Older upgrades set full_sync_completed for local objects too.
				// Only a different remote object ID can establish a cache copy;
				// preserve the stable local identity across any ORIGIN changes.
				if !record.GetBool("needs_full_sync") && !record.GetBool("full_sync_completed") {
					preserved++
					continue
				}
				protected, err := hasLocalChildren1791713184(app, record, local)
				if err != nil {
					return err
				}
				if protected {
					preserved++
					fmt.Fprintf(os.Stderr, "Remote cache cleanup: preserved %s/%s with local contributions.\n", collection, record.Id)
					continue
				}
				if collection == "trails" || collection == "lists" {
					kind := "trail"
					if collection == "lists" {
						kind = "list"
					}
					if _, err := app.DB().NewQuery("DELETE FROM feed WHERE item = {:id} AND type = {:type}").Bind(dbx.Params{"id": record.Id, "type": kind}).Execute(); err != nil {
						return err
					}
				}
				if err := app.Delete(record); err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Remote cache cleanup: removed %s/%s.\n", collection, record.Id)
				removed++
			}
		}
	}
	if removed > 0 || preserved > 0 {
		// Migration-runner transactions must not call the buffered DB logger.
		fmt.Fprintf(os.Stderr, "Remote cache cleanup: removed %d remote cache records; preserved %d ambiguous historical records.\n", removed, preserved)
	}
	return nil
}

// Native cascades and contribution checks repeatedly look up children by
// trail, and feed cleanup by item. Keep those migration-only lookups indexed
// without adding permanent schema indexes to every new installation.
func cleanupLookupIndexes1791713184(app core.App) (func() error, error) {
	created := []string{}
	release := func() error {
		var result error
		for _, name := range created {
			_, err := app.DB().NewQuery("DROP INDEX " + name).Execute()
			result = errors.Join(result, err)
		}
		return result
	}
	for _, index := range []struct{ name, table, columns string }{
		{"_cleanup1791713184_feed", "feed", "item,type"},
		{"_cleanup1791713184_waypoints", "waypoints", "trail"},
		{"_cleanup1791713184_summit_logs", "summit_logs", "trail"},
		{"_cleanup1791713184_comments", "comments", "trail"},
	} {
		var exists int
		if err := app.DB().NewQuery("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name={:name}").Bind(dbx.Params{"name": index.name}).Row(&exists); err != nil {
			return nil, errors.Join(err, release())
		}
		if exists > 0 {
			continue
		}
		if _, err := app.DB().NewQuery("CREATE INDEX " + index.name + " ON " + index.table + "(" + index.columns + ")").Execute(); err != nil {
			return nil, errors.Join(err, release())
		}
		created = append(created, index.name)
	}
	return release, nil
}

// UnsafeWithoutHooks also removes PocketBase's cascades and file interceptors.
// Restore only its pristine model/record hooks, retaining the migration's DB
// and storage configuration. Application hooks must not do network I/O or use
// the buffered DB logger while the runner holds its auxiliary transaction.
func newCoreOnlyCleanupApp1791713184(app core.App) (core.App, error) {
	defaults := core.NewBaseApp(core.BaseAppConfig{DataDir: app.DataDir(), EncryptionEnv: app.EncryptionEnv()})
	settings, err := app.Settings().Clone()
	if err != nil {
		return nil, err
	}
	// Settings itself contains a mutex. The remaining default handlers use
	// e.App's original settings; copy only the filesystem backend snapshot.
	defaults.Settings().S3 = settings.S3
	// The v0.38 native handler captures defaults.App and logs through its
	// global logger. Replace that handler on this private app with equivalent
	// committed-prefix cleanup that writes errors directly to stderr.
	defaults.OnModelAfterDeleteSuccess().Unbind("__pbFilesManagerDelete__")
	defaults.OnModelAfterDeleteSuccess().BindFunc(func(e *core.ModelEvent) error {
		if record, ok := e.Model.(*core.Record); ok {
			hasFiles := false
			for _, field := range record.Collection().Fields {
				hasFiles = hasFiles || field.Type() == core.FieldTypeFile
			}
			if !hasFiles {
				return e.Next()
			}
		}
		if files, ok := e.Model.(core.FilesManager); ok && files.BaseFilesPath() != "" {
			fs, err := e.App.NewFilesystem()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Remote cache cleanup: could not open storage: %v\n", err)
			} else {
				for _, err := range fs.DeletePrefix(strings.TrimRight(files.BaseFilesPath(), "/") + "/") {
					fmt.Fprintf(os.Stderr, "Remote cache cleanup: could not delete storage prefix: %v\n", err)
				}
				_ = fs.Close()
			}
		}
		return e.Next()
	})
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	var mu sync.Mutex
	clones := map[core.App]core.App{}
	var isolate func(core.App) core.App
	isolate = func(source core.App) core.App {
		mu.Lock()
		defer mu.Unlock()
		if clone := clones[source]; clone != nil {
			return clone
		}
		base := source.UnsafeWithoutHooks()
		clone := &cleanupCoreApp1791713184{App: base, logger: logger}
		clones[source], clones[base], clones[clone] = clone, clone, clone
		model := func(target, pristine *hook.TaggedHook[*core.ModelEvent]) {
			target.BindFunc(func(e *core.ModelEvent) error {
				previous := *e
				e.App = isolate(e.App)
				return pristine.Trigger(e, func(*core.ModelEvent) error { return previous.Next() })
			})
		}
		modelError := func(target, pristine *hook.TaggedHook[*core.ModelErrorEvent]) {
			target.BindFunc(func(e *core.ModelErrorEvent) error {
				previous := *e
				e.App = isolate(e.App)
				return pristine.Trigger(e, func(*core.ModelErrorEvent) error { return previous.Next() })
			})
		}
		record := func(target, pristine *hook.TaggedHook[*core.RecordEvent]) {
			target.BindFunc(func(e *core.RecordEvent) error {
				previous := *e
				e.App = isolate(e.App)
				return pristine.Trigger(e, func(*core.RecordEvent) error { return previous.Next() })
			})
		}
		recordError := func(target, pristine *hook.TaggedHook[*core.RecordErrorEvent]) {
			target.BindFunc(func(e *core.RecordErrorEvent) error {
				previous := *e
				e.App = isolate(e.App)
				return pristine.Trigger(e, func(*core.RecordErrorEvent) error { return previous.Next() })
			})
		}
		model(base.OnModelDelete(), defaults.OnModelDelete())
		model(base.OnModelDeleteExecute(), defaults.OnModelDeleteExecute())
		model(base.OnModelAfterDeleteSuccess(), defaults.OnModelAfterDeleteSuccess())
		modelError(base.OnModelAfterDeleteError(), defaults.OnModelAfterDeleteError())
		model(base.OnModelUpdate(), defaults.OnModelUpdate())
		model(base.OnModelUpdateExecute(), defaults.OnModelUpdateExecute())
		model(base.OnModelAfterUpdateSuccess(), defaults.OnModelAfterUpdateSuccess())
		modelError(base.OnModelAfterUpdateError(), defaults.OnModelAfterUpdateError())
		record(base.OnRecordDelete(), defaults.OnRecordDelete())
		record(base.OnRecordDeleteExecute(), defaults.OnRecordDeleteExecute())
		record(base.OnRecordAfterDeleteSuccess(), defaults.OnRecordAfterDeleteSuccess())
		recordError(base.OnRecordAfterDeleteError(), defaults.OnRecordAfterDeleteError())
		record(base.OnRecordUpdate(), defaults.OnRecordUpdate())
		record(base.OnRecordUpdateExecute(), defaults.OnRecordUpdateExecute())
		record(base.OnRecordAfterUpdateSuccess(), defaults.OnRecordAfterUpdateSuccess())
		recordError(base.OnRecordAfterUpdateError(), defaults.OnRecordAfterUpdateError())
		return clone
	}
	return isolate(app), nil
}

type cleanupCoreApp1791713184 struct {
	core.App
	logger *slog.Logger
}

func (app *cleanupCoreApp1791713184) Logger() *slog.Logger { return app.logger }

func hasLocalChildren1791713184(app core.App, record *core.Record, local map[string]*core.Record) (bool, error) {
	if record.Collection().Name != "trails" {
		return false, nil
	}
	for _, collection := range []string{"waypoints", "summit_logs", "comments"} {
		children, err := app.FindRecordsByFilter(collection, "trail={:trail} && author.is_local=true", "", -1, 0, dbx.Params{"trail": record.Id})
		if err != nil {
			return false, err
		}
		for _, child := range children {
			iri := child.GetString("iri")
			if iri == "" || isLocalIRI1791713184(iri) || iriHasRecordIdentity1791713184(iri, collection, child.Id) || historicalLocalActorOrigin1791713184(local[child.GetString("author")], iri) {
				return true, nil
			}
		}
	}
	return false, nil
}

func historicalLocalActorOrigin1791713184(actor *core.Record, iri string) bool {
	if actor == nil {
		return false
	}
	if matchingRemoteHosts1791713184(actor.GetString("iri"), iri) {
		return true
	}
	parsed, err := url.Parse(iri)
	domain := strings.TrimSpace(actor.GetString("domain"))
	return err == nil && parsed.Hostname() != "" && domain != "" && (strings.EqualFold(parsed.Host, domain) || strings.EqualFold(parsed.Hostname(), domain))
}

func matchingRemoteHosts1791713184(a, b string) bool {
	first, errA := url.Parse(a)
	second, errB := url.Parse(b)
	return errA == nil && errB == nil && first.User == nil && second.User == nil &&
		(first.Scheme == "http" || first.Scheme == "https") && (second.Scheme == "http" || second.Scheme == "https") &&
		first.Host != "" && strings.EqualFold(first.Host, second.Host)
}

// These migration-local identity rules are frozen independently of runtime helpers.
func isLocalIRI1791713184(iri string) bool {
	if iri == "" {
		return false
	}
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return false
	}
	o, err := url.Parse(origin)
	if err != nil {
		return false
	}
	u, err := url.Parse(iri)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Host, o.Host)
}

func iriHasRecordIdentity1791713184(iri, collection, id string) bool {
	kind := map[string]string{"trails": "trail", "lists": "list", "summit_logs": "summit-log", "comments": "comment", "waypoints": "waypoint"}[collection]
	parsed, err := url.Parse(iri)
	return err == nil && kind != "" && id != "" && parsed.Host != "" && parsed.User == nil &&
		(parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.RawQuery == "" && parsed.Fragment == "" &&
		parsed.EscapedPath() == parsed.Path && strings.HasSuffix(parsed.Path, "/api/v1/"+kind+"/"+id)
}
