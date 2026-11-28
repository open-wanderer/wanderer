package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pocketbase/pocketbase/tools/logger"
	"io"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/util"
)

const placeholderGPX = `<gpx version="1.1"><trk><trkseg><trkpt lat="46.6" lon="7.2"/><trkpt lat="46.7" lon="7.3"/></trkseg></trk></gpx>`
const changedPlaceholderGPX = `<gpx version="1.1"><rte><rtept lat="47" lon="8"/><rtept lat="47.2" lon="8.1"/></rte></gpx>`

func placeholderFixture(t *testing.T) (*recordHooksFixture, *core.Record) {
	t.Helper()
	t.Setenv("ORIGIN", "https://a.example")
	f := newRecordHooksFixture(t)
	actor := f.save("activitypub_actors", map[string]any{"preferred_username": "walker", "iri": "https://b.example/users/walker", "is_local": false})
	f.bindHooks()
	return f, actor
}

func savePlaceholderTarget(t *testing.T, f *recordHooksFixture, actor *core.Record, collection, content string) *core.Record {
	t.Helper()
	values := map[string]any{"author": actor.Id, "name": "Route", "public": false}
	if content != "" {
		file, err := filesystem.NewFileFromBytes([]byte(content), "route.gpx")
		if err != nil {
			t.Fatal(err)
		}
		values["gpx"] = file
	}
	if collection == "summit_logs" {
		trail := f.save("trails", map[string]any{"author": actor.Id, "name": "Parent"})
		values["trail"] = trail.Id
	}
	record := f.save(collection, values)
	fresh, err := f.app.FindRecordById(collection, record.Id)
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func placeholderLinks(t *testing.T, app core.App, target *core.Record) []*core.Record {
	t.Helper()
	collection, field := "trail_assets", "trail"
	if target.Collection().Name == "summit_logs" {
		collection, field = "summit_log_assets", "summit_log"
	}
	links, err := app.FindRecordsByFilter(collection, field+"={:target}", "created,id", -1, 0, dbx.Params{"target": target.Id})
	if err != nil {
		t.Fatal(err)
	}
	return links
}

func onlyGeneratedPlaceholder(t *testing.T, app core.App, target *core.Record) *core.Record {
	t.Helper()
	links := placeholderLinks(t, app, target)
	if len(links) != 1 {
		t.Fatalf("links = %d, want one generated asset", len(links))
	}
	asset, err := app.FindRecordById("assets", links[0].GetString("asset"))
	if err != nil {
		t.Fatal(err)
	}
	if !util.IsGeneratedRoutePreviewAsset(asset) || asset.GetString("author") != target.GetString("author") || asset.GetString("file") == "" {
		t.Fatalf("invalid generated asset: %v", asset)
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	reader, err := fsys.GetReader(asset.BaseFilesPath() + "/" + asset.GetString("file"))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(content, []byte(`class="route"`)) {
		t.Fatal("placeholder has no route")
	}
	return asset
}

func TestRoutePlaceholderAssetsFollowTargetSaves(t *testing.T) {
	for _, collection := range []string{"trails", "summit_logs"} {
		t.Run(collection, func(t *testing.T) {
			f, actor := placeholderFixture(t)
			target := savePlaceholderTarget(t, f, actor, collection, placeholderGPX)
			old := onlyGeneratedPlaceholder(t, f.app, target)
			target.Set("name", "Renamed")
			if err := f.app.SaveNoValidate(target); err != nil {
				t.Fatal(err)
			}
			if got := onlyGeneratedPlaceholder(t, f.app, target); got.Id != old.Id {
				t.Fatal("metadata edit redrew placeholder")
			}
			file, _ := filesystem.NewFileFromBytes([]byte(changedPlaceholderGPX), "route.gpx")
			target.Set("gpx", file)
			if err := f.app.SaveNoValidate(target); err != nil {
				t.Fatal(err)
			}
			if got := onlyGeneratedPlaceholder(t, f.app, target); got.Id == old.Id {
				t.Fatal("route replacement kept old image")
			}
			if f.exists(old) {
				t.Fatal("unshared replaced image remains")
			}
		})
	}
}

func TestRoutePlaceholderAssetsFollowLinkWrites(t *testing.T) {
	for _, collection := range []string{"trails", "summit_logs"} {
		t.Run(collection, func(t *testing.T) {
			f, actor := placeholderFixture(t)
			target := savePlaceholderTarget(t, f, actor, collection, placeholderGPX)
			generated := onlyGeneratedPlaceholder(t, f.app, target)
			linksName, field := "trail_assets", "trail"
			if collection == "summit_logs" {
				linksName, field = "summit_log_assets", "summit_log"
			}
			photo := f.photo(actor)
			link := f.save(linksName, map[string]any{"asset": photo.Id, field: target.Id, "is_thumbnail": true})
			links := placeholderLinks(t, f.app, target)
			if len(links) != 1 || links[0].Id != link.Id {
				t.Fatal("real attachment retained generated link")
			}
			if collection == "trails" && !links[0].GetBool("is_thumbnail") {
				t.Fatal("real thumbnail changed")
			}
			if f.exists(generated) {
				t.Fatal("unlinked generated asset remains")
			}
			file, _ := filesystem.NewFileFromBytes([]byte(changedPlaceholderGPX), "route.gpx")
			target.Set("gpx", file)
			if err := f.app.SaveNoValidate(target); err != nil {
				t.Fatal(err)
			}
			if links := placeholderLinks(t, f.app, target); len(links) != 1 || links[0].Id != link.Id {
				t.Fatal("route change affected real photo")
			}
			if err := f.app.Delete(link); err != nil {
				t.Fatal(err)
			}
			onlyGeneratedPlaceholder(t, f.app, target)
		})
	}
}

func TestRoutePlaceholderSharedAssetsAndLinkMoves(t *testing.T) {
	f, actor := placeholderFixture(t)
	first := savePlaceholderTarget(t, f, actor, "trails", placeholderGPX)
	generated := onlyGeneratedPlaceholder(t, f.app, first)
	second := savePlaceholderTarget(t, f, actor, "trails", "")
	shared := f.save("trail_assets", map[string]any{"asset": generated.Id, "trail": second.Id})
	photo := f.photo(actor)
	realLink := f.save("trail_assets", map[string]any{"asset": photo.Id, "trail": first.Id, "is_thumbnail": true})
	if !f.exists(generated) || !f.exists(shared) {
		t.Fatal("removing placeholder deleted another target's asset/link")
	}
	realLink, err := f.app.FindRecordById("trail_assets", realLink.Id)
	if err != nil {
		t.Fatal(err)
	}
	realLink.Set("trail", second.Id)
	if err := f.app.SaveNoValidate(realLink); err != nil {
		t.Fatal(err)
	}
	onlyGeneratedPlaceholder(t, f.app, first)
	links := placeholderLinks(t, f.app, second)
	if len(links) != 1 || links[0].Id != realLink.Id || !links[0].GetBool("is_thumbnail") {
		t.Fatal("link move did not preserve real thumbnail")
	}
	if f.exists(generated) {
		t.Fatal("last generated link removal left orphan")
	}
}

func TestRoutePlaceholderRequiresUsableGPX(t *testing.T) {
	for _, content := range []string{"", "<gpx", `<gpx version="1.1"/>`} {
		t.Run(content, func(t *testing.T) {
			f, actor := placeholderFixture(t)
			target := savePlaceholderTarget(t, f, actor, "trails", content)
			if len(placeholderLinks(t, f.app, target)) != 0 {
				t.Fatal("unusable route produced placeholder")
			}
		})
	}
}

func TestRoutePlaceholderTargetDeletionDoesNotRegenerate(t *testing.T) {
	f, actor := placeholderFixture(t)
	target := savePlaceholderTarget(t, f, actor, "trails", placeholderGPX)
	asset := onlyGeneratedPlaceholder(t, f.app, target)
	if err := f.app.Delete(target); err != nil {
		t.Fatal(err)
	}
	if f.exists(target) || f.exists(asset) {
		t.Fatal("target deletion left generated asset")
	}
	if len(placeholderLinks(t, f.app, target)) != 0 {
		t.Fatal("target deletion regenerated placeholder")
	}
}

func TestRoutePlaceholderActorDeletionDoesNotRender(t *testing.T) {
	for _, realPhotos := range []bool{false, true} {
		t.Run(fmt.Sprintf("real_photos=%t", realPhotos), func(t *testing.T) {
			f, actor := placeholderFixture(t)
			for _, collection := range []string{"trails", "summit_logs"} {
				target := savePlaceholderTarget(t, f, actor, collection, placeholderGPX)
				if realPhotos {
					links, field := "trail_assets", "trail"
					if collection == "summit_logs" {
						links, field = "summit_log_assets", "summit_log"
					}
					f.save(links, map[string]any{field: target.Id, "asset": f.photo(actor).Id})
				}
			}
			f.app.Settings().Logs.MaxDays = 1
			attempts := 0
			f.app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
				if util.IsGeneratedRoutePreviewAsset(e.Record) {
					attempts++
				}
				return e.Next()
			})
			started := time.Now()
			if err := f.app.Delete(actor); err != nil {
				t.Fatal(err)
			}
			t.Logf("actor delete: %s", time.Since(started))
			if attempts != 0 {
				t.Fatalf("account deletion attempted %d generated asset creates", attempts)
			}
			if err := f.app.Logger().Handler().(*logger.BatchHandler).WriteAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			var logs []*core.Log
			if err := f.app.LogQuery().AndWhere(dbx.HashExp{"message": "could not generate route placeholder"}).All(&logs); err != nil {
				t.Fatal(err)
			}
			if len(logs) != 0 {
				t.Fatalf("account deletion logged %d placeholder errors", len(logs))
			}
		})
	}
}

func TestRoutePlaceholderAssetCascadeDoesNotRender(t *testing.T) {
	f, actor := placeholderFixture(t)
	target := savePlaceholderTarget(t, f, actor, "trails", placeholderGPX)
	asset := onlyGeneratedPlaceholder(t, f.app, target)
	attempts := 0
	f.app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
		if util.IsGeneratedRoutePreviewAsset(e.Record) {
			attempts++
		}
		return e.Next()
	})
	if err := f.app.Delete(asset); err != nil {
		t.Fatal(err)
	}
	if attempts != 0 || len(placeholderLinks(t, f.app, target)) != 0 {
		t.Fatal("asset cascade regenerated the removed photo")
	}
}
