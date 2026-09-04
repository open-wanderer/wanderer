package routes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"pocketbase/pluginsystem"
	"pocketbase/util"
)

const providerGPX = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="wanderer Hammerhead plugin">
  <trk><trkseg>
    <trkpt lat="46.000000" lon="8.000000"><ele>100</ele><time>2026-01-01T10:00:00Z</time></trkpt>
    <trkpt lat="46.001000" lon="8.001000"><ele>120</ele><time>2026-01-01T10:10:00Z</time></trkpt>
  </trkseg></trk>
</gpx>`

const (
	routeDetailExport    = "get_route_detail_v1"
	activityDetailExport = "get_activity_detail_v1"
)

// fakeRuntimeSession answers plugin exports from canned outputs or errors and
// can run a side effect while the "provider call" is in flight.
type fakeRuntimeSession struct {
	outputs map[string][]byte
	errors  map[string]error
	onCall  func()
	calls   []string
}

func (s *fakeRuntimeSession) Call(_ context.Context, export string, _ []byte) ([]byte, error) {
	s.calls = append(s.calls, export)
	if s.onCall != nil {
		s.onCall()
	}
	if err := s.errors[export]; err != nil {
		return nil, err
	}
	out, ok := s.outputs[export]
	if !ok {
		return nil, fmt.Errorf("unexpected export %q", export)
	}
	return out, nil
}

func (s *fakeRuntimeSession) Close(context.Context) error { return nil }

var trackResyncTestPlugin = pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
	ID: "hammerhead",
	Capabilities: []pluginsystem.CapabilityManifest{
		{Name: "list_routes", Version: "v1", Export: "list_routes_v1"},
		{Name: "get_route_detail", Version: "v1", Export: routeDetailExport},
		{Name: "list_activities", Version: "v1", Export: "list_activities_v1"},
		{Name: "get_activity_detail", Version: "v1", Export: activityDetailExport},
	},
}}

func loadTestPlugin(core.App, string) (pluginsystem.LocalPlugin, error) {
	return trackResyncTestPlugin, nil
}

type trackResyncFixture struct {
	app      *pbtests.TestApp
	actor    *core.Record
	trail    *core.Record
	ref      *core.Record
	instance *core.Record
	refs     *core.Collection
}

func newTrackResyncFixture(t *testing.T) trackResyncFixture {
	t.Helper()

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "user"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "author"},
		&core.BoolField{Name: "completed"},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 5 << 20},
		&core.NumberField{Name: "distance"},
		&core.NumberField{Name: "elevation_gain"},
		&core.NumberField{Name: "elevation_loss"},
		&core.NumberField{Name: "duration"},
		&core.NumberField{Name: "lat"},
		&core.NumberField{Name: "lon"},
		&core.TextField{Name: "polyline", Max: util.PolylineMaxLength},
		&core.NumberField{Name: "min_lat"},
		&core.NumberField{Name: "max_lat"},
		&core.NumberField{Name: "min_lon"},
		&core.NumberField{Name: "max_lon"},
		&core.NumberField{Name: "bounding_box_diagonal"},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	refs := core.NewBaseCollection("trail_external_reference")
	refs.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1, Required: true},
		&core.TextField{Name: "user"},
		&core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"},
		&core.TextField{Name: "plugin_id"},
		&core.TextField{Name: "kind"},
		&core.TextField{Name: "track_source"},
		&core.DateField{Name: "track_resynced_at"},
	)
	if err := app.Save(refs); err != nil {
		t.Fatal(err)
	}

	instances := core.NewBaseCollection("plugin_instances")
	instances.Fields.Add(
		&core.TextField{Name: "plugin_id"},
		&core.TextField{Name: "user"},
		&core.BoolField{Name: "enabled"},
		&core.TextField{Name: "status"},
		&core.JSONField{Name: "last_error"},
		&core.DateField{Name: "retry_not_before"},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(instances); err != nil {
		t.Fatal(err)
	}

	actor := core.NewRecord(actors)
	actor.Set("user", "user-1")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}

	oldGPX, err := filesystem.NewFileFromBytes([]byte("<gpx></gpx>"), "old.gpx")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("name", "Edited by user")
	trail.Set("author", actor.Id)
	trail.Set("completed", true)
	trail.Set("gpx", oldGPX)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}

	ref := core.NewRecord(refs)
	ref.Load(map[string]any{
		"trail":       trail.Id,
		"user":        "user-1",
		"provider":    "hammerhead",
		"external_id": "act-1",
		"plugin_id":   "hammerhead",
		"kind":        util.ExternalReferenceKindCompleted,
	})
	if err := app.Save(ref); err != nil {
		t.Fatal(err)
	}

	instance := core.NewRecord(instances)
	instance.Set("plugin_id", "hammerhead")
	instance.Set("user", "user-1")
	instance.Set("enabled", true)
	if err := app.Save(instance); err != nil {
		t.Fatal(err)
	}

	return trackResyncFixture{app: app, actor: actor, trail: trail, ref: ref, instance: instance, refs: refs}
}

func (f trackResyncFixture) resolve(t *testing.T) (trackResyncTarget, string) {
	t.Helper()
	target, reason, err := resolveTrackResyncTarget(f.app, "user-1", f.trail.Id, "", loadTestPlugin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return target, reason
}

func (f trackResyncFixture) execute(t *testing.T, session *fakeRuntimeSession) (string, error) {
	t.Helper()
	outcome, err := f.executeOutcome(t, session)
	return outcome.kind, err
}

func (f trackResyncFixture) executeOutcome(t *testing.T, session *fakeRuntimeSession) (trackResyncOutcome, error) {
	t.Helper()
	target, reason := f.resolve(t)
	if reason != "" {
		t.Fatalf("expected a resyncable trail, got reason %q", reason)
	}
	return executeTrackResync(context.Background(), f.app, session, target, nil, nil)
}

func pluginError(code string) error {
	return pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: code, Message: code}}
}

func (f trackResyncFixture) setRef(t *testing.T, values map[string]any) {
	t.Helper()
	ref := f.reloadRef(t)
	for key, value := range values {
		ref.Set(key, value)
	}
	if err := f.app.Save(ref); err != nil {
		t.Fatal(err)
	}
}

func (f trackResyncFixture) reloadRef(t *testing.T) *core.Record {
	t.Helper()
	ref, err := f.app.FindRecordById("trail_external_reference", f.ref.Id)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func (f trackResyncFixture) reloadTrail(t *testing.T) *core.Record {
	t.Helper()
	trail, err := f.app.FindRecordById("trails", f.trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	return trail
}

func detailOutput(t *testing.T, kind string, externalID string) []byte {
	t.Helper()
	body, err := json.Marshal(pluginSystemDetailOutput{
		Item: pluginsystem.TrailImport{
			Source: pluginsystem.TrailImportSource{Provider: "hammerhead", ExternalID: externalID},
			Kind:   kind,
			Name:   "Provider name",
			Track: pluginsystem.Track{
				Format:        "gpx",
				ContentBase64: base64.StdEncoding.EncodeToString([]byte(providerGPX)),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func activitySession(t *testing.T) *fakeRuntimeSession {
	t.Helper()
	return &fakeRuntimeSession{outputs: map[string][]byte{activityDetailExport: detailOutput(t, "completed", "act-1")}}
}

func TestResolveTrackResyncTarget(t *testing.T) {
	t.Run("own imported trail is resyncable", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		target, reason := f.resolve(t)
		if reason != "" || target.ref.Id != f.ref.Id || target.instance.Id != f.instance.Id {
			t.Fatalf("expected the trail to resolve, got reason %q", reason)
		}
	})
	t.Run("foreign trail", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		if _, reason, err := resolveTrackResyncTarget(f.app, "someone-else", f.trail.Id, "", loadTestPlugin); err != nil || reason != trackResyncReasonNotOwner {
			t.Fatalf("expected %q, got %q (%v)", trackResyncReasonNotOwner, reason, err)
		}
	})
	t.Run("unknown trail", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		if _, reason, err := resolveTrackResyncTarget(f.app, "user-1", "missing", "", loadTestPlugin); err != nil || reason != trackResyncReasonNotFound {
			t.Fatalf("expected %q, got %q (%v)", trackResyncReasonNotFound, reason, err)
		}
	})
	t.Run("trail without import", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		if err := f.app.Delete(f.ref); err != nil {
			t.Fatal(err)
		}
		if _, reason := f.resolve(t); reason != trackResyncReasonNotImported {
			t.Fatalf("expected %q, got %q", trackResyncReasonNotImported, reason)
		}
	})
	t.Run("moved reference is no source", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		f.setRef(t, map[string]any{"track_source": util.TrackSourceMoved})
		if _, reason := f.resolve(t); reason != trackResyncReasonAmbiguous {
			t.Fatalf("expected %q, got %q", trackResyncReasonAmbiguous, reason)
		}
	})
	t.Run("merged trail with unknown source is ambiguous", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		second := core.NewRecord(f.refs)
		second.Load(map[string]any{"trail": f.trail.Id, "user": "user-1", "provider": "komoot", "external_id": "tour-9", "plugin_id": "komoot"})
		if err := f.app.Save(second); err != nil {
			t.Fatal(err)
		}
		if _, reason := f.resolve(t); reason != trackResyncReasonAmbiguous {
			t.Fatalf("expected %q, got %q", trackResyncReasonAmbiguous, reason)
		}
		// Once the second reference is known to be moved, the trail's own one serves it.
		second.Set("track_source", util.TrackSourceMoved)
		if err := f.app.Save(second); err != nil {
			t.Fatal(err)
		}
		if target, reason := f.resolve(t); reason != "" || target.ref.Id != f.ref.Id {
			t.Fatalf("expected the trail's own reference, got reason %q", reason)
		}
	})
	t.Run("disabled instance", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		f.instance.Set("enabled", false)
		if err := f.app.Save(f.instance); err != nil {
			t.Fatal(err)
		}
		if _, reason := f.resolve(t); reason != trackResyncReasonDisabled {
			t.Fatalf("expected %q, got %q", trackResyncReasonDisabled, reason)
		}
	})
	t.Run("legacy reference is accepted but reported as unverified", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		f.setRef(t, map[string]any{"track_source": util.TrackSourceLegacy})
		target, reason := f.resolve(t)
		if reason != "" || target.ref.GetString("track_source") != util.TrackSourceLegacy {
			t.Fatalf("a legacy reference must resolve, got %q", reason)
		}
	})

	t.Run("reference without recorded kind needs the user to state it", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		f.setRef(t, map[string]any{"kind": ""})
		target, reason := f.resolve(t)
		if reason != trackResyncReasonKindRequired || target.ref == nil {
			t.Fatalf("expected %q with the reference, got %q", trackResyncReasonKindRequired, reason)
		}
		if suggestedTrackResyncKind(target.trail) != util.ExternalReferenceKindCompleted {
			t.Fatal("a completed trail suggests the activity kind")
		}
		if _, reason, err := resolveTrackResyncTarget(f.app, "user-1", f.trail.Id, "bogus", loadTestPlugin); err != nil || reason != trackResyncReasonKindRequired {
			t.Fatalf("an unknown stated kind must be refused, got %q (%v)", reason, err)
		}
		stated, reason, err := resolveTrackResyncTarget(f.app, "user-1", f.trail.Id, util.ExternalReferenceKindPlanned, loadTestPlugin)
		if err != nil || reason != "" || stated.kind != util.ExternalReferenceKindPlanned || !stated.kindStated {
			t.Fatalf("a stated kind must be used, got %q %+v (%v)", reason, stated, err)
		}
	})
	t.Run("plugin without detail capability", func(t *testing.T) {
		f := newTrackResyncFixture(t)
		noDetail := func(core.App, string) (pluginsystem.LocalPlugin, error) {
			return pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "hammerhead"}}, nil
		}
		if _, reason, err := resolveTrackResyncTarget(f.app, "user-1", f.trail.Id, "", noDetail); err != nil || reason != trackResyncReasonNoCapability {
			t.Fatalf("expected %q, got %q (%v)", trackResyncReasonNoCapability, reason, err)
		}
	})
}

func TestVerifyTrackResyncPreviewIdentity(t *testing.T) {
	f := newTrackResyncFixture(t)
	target, reason := f.resolve(t)
	if reason != "" {
		t.Fatalf("expected a resyncable trail, got reason %q", reason)
	}

	if err := verifyTrackResyncPreviewIdentity(target, "hammerhead", "act-1"); err != nil {
		t.Fatalf("the reference matching the confirmed preview must be accepted: %v", err)
	}

	for name, expectation := range map[string]struct {
		provider   string
		externalID string
	}{
		"provider changed":    {provider: "komoot", externalID: "act-1"},
		"external id changed": {provider: "hammerhead", externalID: "act-2"},
	} {
		t.Run(name, func(t *testing.T) {
			err := verifyTrackResyncPreviewIdentity(target, expectation.provider, expectation.externalID)
			if apiErr := apis.ToApiError(err); err == nil || apiErr.Status != http.StatusConflict {
				t.Fatalf("a reference differing from the preview must return 409, got %v", err)
			}
		})
	}
}

func TestExecuteTrackResyncReplacesTrackAndKeepsUserFields(t *testing.T) {
	f := newTrackResyncFixture(t)
	oldFile := f.trail.GetString("gpx")
	session := activitySession(t)
	// An edit while the provider call is in flight must survive.
	session.onCall = func() {
		edited := f.reloadTrail(t)
		edited.Set("name", "Renamed meanwhile")
		if err := f.app.Save(edited); err != nil {
			t.Fatal(err)
		}
	}

	kind, err := f.execute(t, session)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != util.ExternalReferenceKindCompleted {
		t.Fatalf("expected the activity detail to serve a completed trail, got %q", kind)
	}
	if len(session.calls) != 1 || session.calls[0] != activityDetailExport {
		t.Fatalf("expected one activity detail call, got %v", session.calls)
	}

	trail := f.reloadTrail(t)
	if trail.GetString("name") != "Renamed meanwhile" {
		t.Fatalf("concurrent edit was overwritten: %q", trail.GetString("name"))
	}
	if gpx := trail.GetString("gpx"); gpx == "" || gpx == oldFile {
		t.Fatalf("expected replaced gpx file, got %q", gpx)
	}
	if trail.GetFloat("elevation_gain") != 20 {
		t.Fatalf("expected metrics from new track, got elevation gain %v", trail.GetFloat("elevation_gain"))
	}

	ref := f.reloadRef(t)
	if ref.GetString("kind") != util.ExternalReferenceKindCompleted || ref.GetDateTime("track_resynced_at").IsZero() {
		t.Fatal("expected the served kind and the completion to be recorded")
	}
}

func TestExecuteTrackResyncRefusesReferenceMovedDuringProviderCall(t *testing.T) {
	f := newTrackResyncFixture(t)
	oldFile := f.trail.GetString("gpx")
	target := core.NewRecord(f.trail.Collection())
	target.Set("name", "Merge target")
	target.Set("author", f.actor.Id)
	if err := f.app.Save(target); err != nil {
		t.Fatal(err)
	}
	session := activitySession(t)
	// A merge moves the reference onto another trail while the provider call is in flight.
	session.onCall = func() {
		if err := util.ReassignTrailExternalReferences(f.app, f.trail.Id, target.Id); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := f.execute(t, session); err == nil {
		t.Fatal("a reference moved meanwhile must not replace any track")
	}
	if f.reloadTrail(t).GetString("gpx") != oldFile {
		t.Fatal("the original trail must stay untouched")
	}
	moved, err := f.app.FindRecordById("trails", target.Id)
	if err != nil {
		t.Fatal(err)
	}
	if moved.GetString("gpx") != "" {
		t.Fatal("the merge target must stay untouched")
	}
	if !f.reloadRef(t).GetDateTime("track_resynced_at").IsZero() {
		t.Fatal("nothing must be recorded on the moved reference")
	}
}

func TestExecuteTrackResyncRefusesToOverwriteTrackEditedMeanwhile(t *testing.T) {
	f := newTrackResyncFixture(t)
	session := activitySession(t)
	var editedFile string
	// The user uploads a new track while the provider call is in flight.
	session.onCall = func() {
		edited := f.reloadTrail(t)
		file, err := filesystem.NewFileFromBytes([]byte(`<?xml version="1.0"?><gpx version="1.1" creator="editor"><trk><trkseg><trkpt lat="1" lon="2"></trkpt></trkseg></trk></gpx>`), "edited.gpx")
		if err != nil {
			t.Fatal(err)
		}
		edited.Set("gpx", file)
		if err := f.app.Save(edited); err != nil {
			t.Fatal(err)
		}
		editedFile = edited.GetString("gpx")
	}

	if _, err := f.execute(t, session); err == nil {
		t.Fatal("a track replaced meanwhile must not be overwritten")
	}
	if f.reloadTrail(t).GetString("gpx") != editedFile {
		t.Fatal("the user's edited track must be kept")
	}
	if !f.reloadRef(t).GetDateTime("track_resynced_at").IsZero() {
		t.Fatal("nothing must be recorded when the resync was refused")
	}
}

func TestExecuteTrackResyncStopsAtInstanceLevelErrorsAndReportsThem(t *testing.T) {
	retry := 90
	f := newTrackResyncFixture(t)
	session := &fakeRuntimeSession{errors: map[string]error{
		activityDetailExport: pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: "rate_limited", Message: "slow down", RetryAfterSeconds: &retry}},
		routeDetailExport:    pluginError("not_importable"),
	}}

	_, err := f.execute(t, session)
	var rejected trackResyncProviderError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a provider error, got %v", err)
	}
	if len(session.calls) != 1 {
		t.Fatalf("expected a single detail call, got %v", session.calls)
	}
	data := rejected.data()
	if data["code"] != "rate_limited" || data["retryAfterSeconds"] != 90 {
		t.Fatalf("expected the structured plugin error to be reported, got %v", data)
	}
}

func TestProviderRejectedBodySurvivesSerialization(t *testing.T) {
	retry := 90
	rejected := trackResyncProviderError{err: pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: "rate_limited", Message: "slow down", RetryAfterSeconds: &retry}}}

	raw, err := json.Marshal(providerRejectedBody(rejected))
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
		Data    struct {
			Code              string `json:"code"`
			Message           string `json:"message"`
			RetryAfterSeconds int    `json:"retryAfterSeconds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != 502 || body.Message != trackResyncErrProviderRejected {
		t.Fatalf("unexpected envelope: %s", raw)
	}
	if body.Data.Code != "rate_limited" || body.Data.Message != "slow down" || body.Data.RetryAfterSeconds != 90 {
		t.Fatalf("plugin error must reach the client intact, got %s", raw)
	}

	plain := trackResyncProviderError{err: errors.New("connection reset")}
	raw, err = json.Marshal(providerRejectedBody(plain))
	if err != nil {
		t.Fatal(err)
	}
	var plainBody struct {
		Data struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &plainBody); err != nil || plainBody.Data.Message != "connection reset" || plainBody.Data.Code != "provider_unavailable" {
		t.Fatalf("plain errors are reported as provider_unavailable with their message, got %s (%v)", raw, err)
	}
}

func TestExecuteTrackResyncRefusesKindRecordedMeanwhile(t *testing.T) {
	f := newTrackResyncFixture(t)
	oldFile := f.trail.GetString("gpx")
	session := activitySession(t)
	// Another resync records the other kind while the provider call is in flight.
	session.onCall = func() {
		f.setRef(t, map[string]any{"kind": util.ExternalReferenceKindPlanned})
	}

	if _, err := f.execute(t, session); err == nil {
		t.Fatal("a kind recorded meanwhile that disagrees with the served item must refuse the replacement")
	}
	if f.reloadTrail(t).GetString("gpx") != oldFile {
		t.Fatal("stored track must stay untouched")
	}
}

func TestExecuteTrackResyncReportsFailedFollowUpAsWarning(t *testing.T) {
	f := newTrackResyncFixture(t)
	oldFile := f.trail.GetString("gpx")
	hookID := f.app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
		return errors.New("search index unavailable")
	})
	defer f.app.OnRecordAfterUpdateSuccess("trails").Unbind(hookID)

	outcome, err := f.executeOutcome(t, activitySession(t))
	if err != nil {
		t.Fatalf("a failed post-commit follow-up must not read as a failed resync: %v", err)
	}
	if outcome.kind != util.ExternalReferenceKindCompleted || outcome.warning == "" {
		t.Fatalf("expected the served kind and a warning, got %+v", outcome)
	}
	stored := f.reloadTrail(t)
	if stored.GetString("gpx") == oldFile {
		t.Fatal("the committed track replacement must be reported as done")
	}
	if stored.GetString("polyline") == "" || outcome.track.Polyline != stored.GetString("polyline") {
		t.Fatalf("track geometry must be committed before a follow-up hook can fail: %+v", outcome.track)
	}
}

func TestExecuteTrackResyncRefusesRouteOnlyTrack(t *testing.T) {
	f := newTrackResyncFixture(t)
	oldFile := f.trail.GetString("gpx")
	body, err := json.Marshal(pluginSystemDetailOutput{Item: pluginsystem.TrailImport{
		Source: pluginsystem.TrailImportSource{Provider: "hammerhead", ExternalID: "act-1"},
		Kind:   "completed",
		Track: pluginsystem.Track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString([]byte(`<?xml version="1.0"?><gpx version="1.1" creator="t"><rte><rtept lat="46" lon="8"></rtept><rtept lat="46.001" lon="8.001"></rtept></rte></gpx>`)),
		},
	}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.execute(t, &fakeRuntimeSession{outputs: map[string][]byte{activityDetailExport: body}})
	var rejected trackResyncProviderError
	if !errors.As(err, &rejected) {
		t.Fatalf("a route-only document must be reported as a provider error, got %v", err)
	}
	if f.reloadTrail(t).GetString("gpx") != oldFile {
		t.Fatal("stored track must stay untouched")
	}
}

func TestExecuteTrackResyncRefusesEmptyTrack(t *testing.T) {
	f := newTrackResyncFixture(t)
	oldFile := f.trail.GetString("gpx")
	body, err := json.Marshal(pluginSystemDetailOutput{Item: pluginsystem.TrailImport{
		Source: pluginsystem.TrailImportSource{Provider: "hammerhead", ExternalID: "act-1"},
		Kind:   "completed",
		Track: pluginsystem.Track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString([]byte(`<?xml version="1.0"?><gpx version="1.1" creator="t"><trk><trkseg></trkseg></trk></gpx>`)),
		},
	}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = f.execute(t, &fakeRuntimeSession{outputs: map[string][]byte{activityDetailExport: body}})
	var rejected trackResyncProviderError
	if !errors.As(err, &rejected) {
		t.Fatalf("an empty track must be reported as a provider error, got %v", err)
	}
	if f.reloadTrail(t).GetString("gpx") != oldFile {
		t.Fatal("stored track must stay untouched")
	}
}

func TestExecuteTrackResyncRecordsStatedKindAndReportsTrackFields(t *testing.T) {
	f := newTrackResyncFixture(t)
	f.setRef(t, map[string]any{"kind": ""})
	target, reason, err := resolveTrackResyncTarget(f.app, "user-1", f.trail.Id, util.ExternalReferenceKindPlanned, loadTestPlugin)
	if err != nil || reason != "" {
		t.Fatalf("unexpected resolve result %q (%v)", reason, err)
	}
	session := &fakeRuntimeSession{outputs: map[string][]byte{routeDetailExport: detailOutput(t, "planned", "act-1")}}

	outcome, err := executeTrackResync(context.Background(), f.app, session, target, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.reloadRef(t).GetString("kind") != util.ExternalReferenceKindPlanned {
		t.Fatal("the stated kind must be recorded once the fetch succeeded")
	}
	trail := f.reloadTrail(t)
	if outcome.track.GPX != trail.GetString("gpx") || outcome.track.ElevationGain != 20 || outcome.track.Updated != trail.GetString("updated") {
		t.Fatalf("expected the stored track fields to be reported, got %+v", outcome.track)
	}
}

func TestRecordTrackResyncInstanceStatusSkipsInstancesChangedMeanwhile(t *testing.T) {
	f := newTrackResyncFixture(t)
	seen := f.instance.GetDateTime("updated")

	applied := false
	recordTrackResyncInstanceStatus(f.app, f.instance.Id, seen, func(core.App, *core.Record) { applied = true })
	if !applied {
		t.Fatal("an unchanged instance must receive the status update")
	}

	// Someone disables the instance while the provider call runs.
	changed, err := f.app.FindRecordById("plugin_instances", f.instance.Id)
	if err != nil {
		t.Fatal(err)
	}
	changed.Set("enabled", false)
	// The autodate field stamps the save itself; make sure it lands in a
	// later millisecond than the creation.
	time.Sleep(3 * time.Millisecond)
	if err := f.app.Save(changed); err != nil {
		t.Fatal(err)
	}
	applied = false
	recordTrackResyncInstanceStatus(f.app, f.instance.Id, seen, func(core.App, *core.Record) { applied = true })
	if applied {
		t.Fatal("an instance changed meanwhile must keep its state")
	}
}

func TestClearPluginInstanceErrorAfterSuccess(t *testing.T) {
	f := newTrackResyncFixture(t)
	reload := func() *core.Record {
		instance, err := f.app.FindRecordById("plugin_instances", f.instance.Id)
		if err != nil {
			t.Fatal(err)
		}
		return instance
	}
	set := func(status string) {
		instance := reload()
		instance.Set("status", status)
		instance.Set("retry_not_before", time.Now().Add(time.Hour))
		if err := f.app.Save(instance); err != nil {
			t.Fatal(err)
		}
	}

	set("needs_reauth")
	clearPluginInstanceErrorAfterSuccess(f.app, reload())
	if got := reload(); got.GetString("status") != "configured" || !got.GetDateTime("retry_not_before").IsZero() {
		t.Fatalf("a credential error is cleared by a successful call, got %q", got.GetString("status"))
	}

	// A token refresh sets the status back to configured but leaves the
	// remembered error and backoff behind; the success clears those too.
	set("configured")
	stale := reload()
	stale.Set("last_error", map[string]any{"code": "rate_limited", "message": "earlier"})
	if err := f.app.Save(stale); err != nil {
		t.Fatal(err)
	}
	clearPluginInstanceErrorAfterSuccess(f.app, reload())
	if got := reload(); len(pluginsystem.JSONMapFromRecord(got, "last_error")) != 0 || !got.GetDateTime("retry_not_before").IsZero() {
		t.Fatalf("a remembered error is cleared by a successful call, got %s / %s", got.GetString("last_error"), got.GetString("retry_not_before"))
	}

	disabled := reload()
	disabled.Set("enabled", false)
	disabled.Set("status", "error")
	if err := f.app.Save(disabled); err != nil {
		t.Fatal(err)
	}
	clearPluginInstanceErrorAfterSuccess(f.app, reload())
	if got := reload(); got.GetString("status") != "error" {
		t.Fatal("a disabled instance keeps its state")
	}
}

func TestProviderRejectedBodyCarriesDefaultBackoff(t *testing.T) {
	rejected := trackResyncProviderError{err: pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: "rate_limited", Message: "slow down"}}}
	data := rejected.data()
	wait, ok := data["retryAfterSeconds"].(int)
	if !ok || wait < 3590 || wait > 3600 {
		t.Fatalf("expected the default one hour backoff to be reported, got %v", data["retryAfterSeconds"])
	}

	// A non-positive hint is ignored by the status mapping; the answer must
	// report the same backoff the instance receives.
	zero := 0
	hinted := trackResyncProviderError{err: pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: "rate_limited", Message: "slow down", RetryAfterSeconds: &zero}}}
	if wait, ok := hinted.data()["retryAfterSeconds"].(int); !ok || wait < 3590 {
		t.Fatalf("expected the default backoff for a non-positive hint, got %v", hinted.data()["retryAfterSeconds"])
	}

	unavailable := trackResyncProviderError{err: pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: "provider_unavailable", Message: "down"}}}
	if _, ok := unavailable.data()["retryAfterSeconds"]; ok {
		t.Fatal("an outage without hint carries no backoff")
	}
}

func TestTrackResyncBackoffFollowsRetryNotBefore(t *testing.T) {
	f := newTrackResyncFixture(t)
	if trackResyncBackoffSeconds(f.instance) != 0 {
		t.Fatal("an instance without backoff must not wait")
	}

	f.instance.Set("retry_not_before", time.Now().Add(90*time.Second))
	f.instance.Set("last_error", map[string]any{"code": "rate_limited", "message": "slow down"})
	wait := trackResyncBackoffSeconds(f.instance)
	if wait < 85 || wait > 90 {
		t.Fatalf("expected the remaining backoff, got %d", wait)
	}
	pluginErr := pluginErrorOf(trackResyncBackoffError(f.instance, wait))
	if pluginErr == nil || pluginErr.Code != "rate_limited" || pluginErr.Message != "slow down" || pluginErr.RetryAfterSeconds == nil || *pluginErr.RetryAfterSeconds != wait {
		t.Fatalf("expected the causing error with the countdown, got %+v", pluginErr)
	}

	f.instance.Set("retry_not_before", time.Now().Add(-time.Second))
	if trackResyncBackoffSeconds(f.instance) != 0 {
		t.Fatal("an expired backoff must not wait")
	}
}

func TestExecuteTrackResyncCallsOnlyTheRecordedKind(t *testing.T) {
	f := newTrackResyncFixture(t)
	session := &fakeRuntimeSession{
		errors:  map[string]error{activityDetailExport: pluginError("provider_unavailable")},
		outputs: map[string][]byte{routeDetailExport: detailOutput(t, "planned", "act-1")},
	}

	_, err := f.execute(t, session)
	var rejected trackResyncProviderError
	if !errors.As(err, &rejected) {
		t.Fatalf("expected a provider error, got %v", err)
	}
	if len(session.calls) != 1 || session.calls[0] != activityDetailExport {
		t.Fatalf("the other endpoint must never be tried, got %v", session.calls)
	}
	if f.reloadRef(t).GetString("kind") != util.ExternalReferenceKindCompleted {
		t.Fatal("the recorded kind must not change")
	}
}

func TestExecuteTrackResyncRejectsAnswersThatDoNotIdentifyTheItem(t *testing.T) {
	cases := map[string]map[string][]byte{
		"other external id": {activityDetailExport: detailOutput(t, "completed", "act-999"), routeDetailExport: detailOutput(t, "planned", "act-999")},
		"wrong kind":        {activityDetailExport: detailOutput(t, "planned", "act-1")},
	}
	for name, outputs := range cases {
		t.Run(name, func(t *testing.T) {
			f := newTrackResyncFixture(t)
			oldFile := f.trail.GetString("gpx")
			_, err := f.execute(t, &fakeRuntimeSession{outputs: outputs})
			var rejected trackResyncProviderError
			if !errors.As(err, &rejected) {
				t.Fatalf("an answer that does not identify the item must be reported as a provider error, got %v", err)
			}
			if f.reloadTrail(t).GetString("gpx") != oldFile {
				t.Fatal("track must stay untouched")
			}
		})
	}
}
