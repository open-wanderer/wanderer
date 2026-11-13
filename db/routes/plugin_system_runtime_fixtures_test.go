package routes

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"pocketbase/pluginsystem"
	"strconv"
	"strings"
	"testing"
	"time"
)

const syncWorkerTrack = `<gpx version="1.1" creator="test"><trk><trkseg><trkpt lat="46" lon="8"><time>2026-01-01T10:00:00Z</time></trkpt><trkpt lat="46.01" lon="8.01"><time>2026-01-01T10:10:00Z</time></trkpt></trkseg></trk></gpx>`

// This process speaks the real worker protocol. Its deterministic provider
// payloads keep the test about worker admission/session lifecycle and DB import.
func TestPluginSyncWorkerRPCFixture(t *testing.T) {
	if os.Getenv("WANDERER_SYNC_TEST_RPC") != "1" {
		return
	}
	for {
		var header [4]byte
		if _, err := io.ReadFull(os.Stdin, header[:]); err != nil {
			os.Exit(0)
		}
		size := binary.BigEndian.Uint32(header[:])
		if size == 0 || size > 32*1024*1024 {
			os.Exit(2)
		}
		body := make([]byte, size)
		if _, err := io.ReadFull(os.Stdin, body); err != nil {
			os.Exit(2)
		}
		var message struct {
			Type string `json:"type"`
			Data struct {
				Export string `json:"export"`
				Input  string `json:"inputBase64"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &message); err != nil {
			os.Exit(2)
		}
		if message.Type == "shutdown" {
			os.Exit(0)
		}
		input, err := base64.StdEncoding.DecodeString(message.Data.Input)
		if err != nil {
			os.Exit(2)
		}
		if path := os.Getenv("WANDERER_SYNC_RPC_LOG"); path != "" {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				os.Exit(2)
			}
			_, _ = io.WriteString(file, strconv.Itoa(os.Getpid())+" "+message.Data.Export+"\n")
			_ = file.Close()
		}
		kind := "completed"
		if os.Getenv("WANDERER_SYNC_SCENARIO") == "planned" || os.Getenv("WANDERER_SYNC_TTL_PHASE") != "" {
			kind = "planned"
		}
		var output any
		switch message.Data.Export {
		case "list":
			if os.Getenv("WANDERER_SYNC_TTL_PHASE") == "list" {
				var request pluginSystemListInput
				_ = json.Unmarshal(input, &request)
				if request.State["page"] == nil {
					output = pluginSystemListOutput{Items: []pluginsystem.TrailSummary{{Kind: kind, Source: pluginsystem.TrailImportSource{Provider: "hikes", ExternalID: "first"}}}, State: map[string]any{"page": 2}, HasMore: true}
				} else {
					output = pluginSystemListOutput{Items: []pluginsystem.TrailSummary{{Kind: kind, Source: pluginsystem.TrailImportSource{Provider: "hikes", ExternalID: "second"}}}}
				}
				break
			}
			output = pluginSystemListOutput{Items: []pluginsystem.TrailSummary{
				{Kind: kind, Source: pluginsystem.TrailImportSource{Provider: "hikes", ExternalID: "first"}},
				{Kind: kind, Source: pluginsystem.TrailImportSource{Provider: "hikes", ExternalID: "second"}},
			}}
		case "detail":
			if os.Getenv("WANDERER_SYNC_TTL_PHASE") == "renew" {
				time.Sleep(225 * time.Millisecond)
			}
			var request pluginSystemDetailInput
			_ = json.Unmarshal(input, &request)
			output = pluginSystemDetailOutput{Item: pluginsystem.TrailImport{Name: "Imported " + request.Summary.Source.ExternalID, Kind: kind, Source: request.Summary.Source,
				Track: pluginsystem.Track{Format: "gpx", ContentBase64: base64.StdEncoding.EncodeToString([]byte(syncWorkerTrack))}}}
		default:
			os.Exit(2)
		}
		payload, err := json.Marshal(output)
		if err != nil {
			os.Exit(2)
		}
		response, _ := json.Marshal(map[string]any{"type": "call_result", "data": map[string]any{"outputBase64": base64.StdEncoding.EncodeToString(payload)}})
		binary.BigEndian.PutUint32(header[:], uint32(len(response)))
		_, _ = os.Stdout.Write(header[:])
		_, _ = os.Stdout.Write(response)
	}
}

func syncTestWorker(t *testing.T) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "worker")
	quoted := "'" + strings.ReplaceAll(executable, "'", "'\"'\"'") + "'"
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexport WANDERER_SYNC_TEST_RPC=1\nexec "+quoted+" -test.run='^TestPluginSyncWorkerRPCFixture$'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WANDERER_PLUGIN_WORKER_BIN", path)
	return path
}

type assetImportBatchTestSession struct {
	call func(context.Context, string, []byte, pluginsystem.RuntimeCallOptions) ([]byte, error)
}

func (s assetImportBatchTestSession) Call(ctx context.Context, export string, input []byte, options pluginsystem.RuntimeCallOptions) ([]byte, error) {
	return s.call(ctx, export, input, options)
}
func (assetImportBatchTestSession) Close(context.Context) error { return nil }

type assetImportLoggingApp struct {
	core.App
	logger *slog.Logger
}

func (app assetImportLoggingApp) Logger() *slog.Logger { return app.logger }
func newSyncRuntimeTestApp(t *testing.T) (*pbtests.TestApp, *core.Record, *core.Record) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app.Settings().Logs.MaxDays = 0
	t.Cleanup(app.Cleanup)
	for name, fields := range map[string][]core.Field{
		"activitypub_actors": {&core.TextField{Name: "user"}},
		"plugin_instances":   {&core.TextField{Name: "user"}, &core.TextField{Name: "plugin_id"}, &core.BoolField{Name: "enabled"}, &core.JSONField{Name: "config"}, &core.JSONField{Name: "auth"}},
	} {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(fields...)
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	collection, _ := app.FindCollectionByNameOrId("plugin_instances")
	instance := core.NewRecord(collection)
	instance.Load(map[string]any{"user": "user", "plugin_id": "hikes", "enabled": true})
	if err := app.Save(instance); err != nil {
		t.Fatal(err)
	}
	collection, _ = app.FindCollectionByNameOrId("activitypub_actors")
	actor := core.NewRecord(collection)
	actor.Set("user", "user")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	return app, instance, actor
}
