package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	"pocketbase/util"
)

// The track resync lets a user fetch the track of one imported trail again
// from the plugin it came from, immediately and on request, for example after
// a plugin fix changed how tracks are generated. Only the track and the
// values derived from it are replaced; everything the user edited or added
// stays.

type pluginTrackResyncRequest struct {
	TrailID            string `json:"trailId"`
	ExpectedProvider   string `json:"expectedProvider,omitempty"`
	ExpectedExternalID string `json:"expectedExternalId,omitempty"`
	// Kind is required when the reference does not record whether the item
	// is a planned route or a completed activity; the user states it.
	Kind string `json:"kind,omitempty"`
}

// pluginTrackResyncPreview tells the client whether a trail can be resynced
// and, if not, why (a reason key the client translates).
type pluginTrackResyncPreview struct {
	Available  bool   `json:"available"`
	Reason     string `json:"reason,omitempty"`
	Provider   string `json:"provider,omitempty"`
	ExternalID string `json:"externalId,omitempty"`
	// KindRequired asks the client to let the user state the item kind,
	// SuggestedKind being the likely answer from the trail's completed flag.
	KindRequired  bool   `json:"kindRequired,omitempty"`
	SuggestedKind string `json:"suggestedKind,omitempty"`
	// RetryAfterSeconds is how long the provider asked to wait after an
	// earlier failure; a request before that is refused without a call.
	RetryAfterSeconds int `json:"retryAfterSeconds,omitempty"`
	// OriginUnverified tells that the reference predates merge tracking:
	// it looks like the trail's own, but a merge could have moved it here,
	// so the user should check provider and item before replacing the track.
	OriginUnverified bool `json:"originUnverified,omitempty"`
}

type pluginTrackResyncResult struct {
	TrailID    string `json:"trailId"`
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	Kind       string `json:"kind"`
	// Warning is set when the track was stored but a follow-up such as the
	// search index update failed; a later edit of the trail catches it up.
	Warning string `json:"warning,omitempty"`
	// Track carries the fields ReplaceTrailTrack changed, so the client can
	// bring its copy of the trail up to date without a reload.
	Track pluginTrackResyncTrackFields `json:"track"`
}

// pluginTrackResyncTrackFields are the trail fields a new track changes,
// including the geometry ReplaceTrailTrack derives from the same parsed GPX.
type pluginTrackResyncTrackFields struct {
	GPX                 string  `json:"gpx"`
	Distance            float64 `json:"distance"`
	ElevationGain       float64 `json:"elevation_gain"`
	ElevationLoss       float64 `json:"elevation_loss"`
	Duration            float64 `json:"duration"`
	Lat                 float64 `json:"lat"`
	Lon                 float64 `json:"lon"`
	Polyline            string  `json:"polyline"`
	MinLat              float64 `json:"min_lat"`
	MaxLat              float64 `json:"max_lat"`
	MinLon              float64 `json:"min_lon"`
	MaxLon              float64 `json:"max_lon"`
	BoundingBoxDiagonal float64 `json:"bounding_box_diagonal"`
	Updated             string  `json:"updated"`
}

// trackResyncOutcome is what executeTrackResync reports on success.
type trackResyncOutcome struct {
	kind    string
	warning string
	track   pluginTrackResyncTrackFields
}

func trackFieldsOf(trail *core.Record) pluginTrackResyncTrackFields {
	return pluginTrackResyncTrackFields{
		GPX:                 trail.GetString("gpx"),
		Distance:            trail.GetFloat("distance"),
		ElevationGain:       trail.GetFloat("elevation_gain"),
		ElevationLoss:       trail.GetFloat("elevation_loss"),
		Duration:            trail.GetFloat("duration"),
		Lat:                 trail.GetFloat("lat"),
		Lon:                 trail.GetFloat("lon"),
		Polyline:            trail.GetString("polyline"),
		MinLat:              trail.GetFloat("min_lat"),
		MaxLat:              trail.GetFloat("max_lat"),
		MinLon:              trail.GetFloat("min_lon"),
		MaxLon:              trail.GetFloat("max_lon"),
		BoundingBoxDiagonal: trail.GetFloat("bounding_box_diagonal"),
		Updated:             trail.GetString("updated"),
	}
}

// trackResyncTarget is a trail the requesting user may resync, with the
// reference and plugin instance that serve it and the kind of item to fetch.
type trackResyncTarget struct {
	trail    *core.Record
	ref      *core.Record
	instance *core.Record
	plugin   pluginsystem.LocalPlugin
	// kind is the recorded reference kind, or the kind the user stated for
	// a reference that has none; kindStated tells which.
	kind       string
	kindStated bool
}

// Reasons a trail cannot be resynced, exposed to the client.
const (
	trackResyncReasonNotFound      = "trail_not_found"
	trackResyncReasonNotOwner      = "not_owner"
	trackResyncReasonNotImported   = "not_imported"
	trackResyncReasonAmbiguous     = "ambiguous"
	trackResyncReasonDisabled      = "instance_disabled"
	trackResyncReasonNoPlugin      = "plugin_unavailable"
	trackResyncReasonNoCapability  = "no_detail_capability"
	trackResyncReasonKindRequired  = "kind_required"
	trackResyncErrProviderRejected = "provider_rejected"
)

// PluginSystemTrackResyncPreview reports whether the trail can be resynced.
func PluginSystemTrackResyncPreview(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var data pluginTrackResyncRequest
	if err := e.BindBody(&data); err != nil || data.TrailID == "" {
		return apis.NewBadRequestError("trailId is required", err)
	}

	target, reason, err := resolveTrackResyncTarget(e.App, e.Auth.Id, data.TrailID, "", localPlugin)
	if err != nil {
		return err
	}
	if reason != "" && reason != trackResyncReasonKindRequired {
		return e.JSON(http.StatusOK, pluginTrackResyncPreview{Reason: reason})
	}
	preview := pluginTrackResyncPreview{
		Available:         true,
		Provider:          target.ref.GetString("provider"),
		ExternalID:        target.ref.GetString("external_id"),
		RetryAfterSeconds: trackResyncBackoffSeconds(target.instance),
		OriginUnverified:  target.ref.GetString("track_source") == util.TrackSourceLegacy,
	}
	if reason == trackResyncReasonKindRequired {
		preview.KindRequired = true
		preview.SuggestedKind = suggestedTrackResyncKind(target.trail)
	}
	return e.JSON(http.StatusOK, preview)
}

// PluginSystemTrackResync fetches the trail's track from its plugin and
// replaces the stored one, synchronously.
func PluginSystemTrackResync(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var data pluginTrackResyncRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("invalid track resync request", err)
	}
	if data.TrailID == "" {
		return apis.NewBadRequestError("trailId is required", nil)
	}
	if data.ExpectedProvider == "" || data.ExpectedExternalID == "" {
		return apis.NewBadRequestError("expectedProvider and expectedExternalId are required", nil)
	}

	target, reason, err := resolveTrackResyncTarget(e.App, e.Auth.Id, data.TrailID, data.Kind, localPlugin)
	if err != nil {
		return err
	}
	if reason != "" {
		return apis.NewBadRequestError(reason, nil)
	}
	if err := verifyTrackResyncPreviewIdentity(target, data.ExpectedProvider, data.ExpectedExternalID); err != nil {
		return err
	}

	// The scheduled sync honours the backoff a provider asked for; so does a
	// manual request, telling the client how long to wait.
	if wait := trackResyncBackoffSeconds(target.instance); wait > 0 {
		return e.JSON(http.StatusBadGateway, providerRejectedBody(trackResyncProviderError{err: trackResyncBackoffError(target.instance, wait)}))
	}

	// Instance status is written like the scheduled sync writes it, on a
	// freshly loaded record and only if nobody touched the instance since
	// this request read it, so changes made meanwhile survive.
	instanceSeen := target.instance.GetDateTime("updated")
	auth, err := decryptedInstanceAuth(target.instance)
	if err != nil {
		recordTrackResyncInstanceStatus(e.App, target.instance.Id, instanceSeen, func(app core.App, instance *core.Record) {
			setPluginInstanceStatus(app, instance, "needs_reauth", "auth_failed", err.Error())
		})
		return e.JSON(http.StatusBadGateway, providerRejectedBody(trackResyncProviderError{err: pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: "auth_failed", Message: err.Error()}}}))
	}
	auth, err = pluginsystem.RefreshOAuthAuthIfNeeded(e.Request.Context(), e.App, target.plugin, target.instance, auth)
	if err != nil {
		if errors.Is(err, pluginsystem.ErrPluginInstanceChanged) {
			return apis.NewApiError(http.StatusConflict, "plugin instance changed meanwhile", nil)
		}
		// The token endpoint's answer decides what the failure means. A
		// rejected grant or client and a rate limit concern the instance
		// and are recorded like the scheduled sync records them; a
		// transport problem, an outage or a plugin mistake says nothing
		// about the credentials, so the instance keeps its state.
		refreshErr := pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: pluginsystem.OAuthRefreshFailureCode(err), Message: err.Error()}}
		if pluginErrorConcernsInstance(refreshErr) {
			recordTrackResyncInstanceStatus(e.App, target.instance.Id, instanceSeen, func(app core.App, instance *core.Record) {
				setPluginInstanceStatusForError(app, instance, refreshErr)
			})
		} else {
			e.App.Logger().Warn("token refresh failed before track resync", "instance", target.instance.Id, "error", err)
		}
		return e.JSON(http.StatusBadGateway, providerRejectedBody(trackResyncProviderError{err: refreshErr}))
	}
	// A refresh may just have stored a new token; later status writes must
	// compare against that state, not the one before it.
	instanceSeen = target.instance.GetDateTime("updated")
	config := effectivePluginConfig(e.App, target.plugin.Manifest.ID, target.instance)
	pluginConfig := pluginRuntimeConfig(config)
	policy := pluginInstancePolicy(target.plugin, config).WithHostAuth(auth)

	runtime, err := pluginsystem.NewRuntimeRegistry().RuntimeFor(target.plugin)
	if err != nil {
		recordTrackResyncInstanceStatus(e.App, target.instance.Id, instanceSeen, func(app core.App, instance *core.Record) {
			setPluginInstanceStatusForError(app, instance, err)
		})
		return err
	}
	session, err := runtime.OpenSession(e.Request.Context(), target.plugin, policy)
	if err != nil {
		recordTrackResyncInstanceStatus(e.App, target.instance.Id, instanceSeen, func(app core.App, instance *core.Record) {
			setPluginInstanceStatusForError(app, instance, err)
		})
		return err
	}
	defer func() {
		_ = session.Close(context.Background())
	}()

	outcome, err := executeTrackResync(e.Request.Context(), e.App, session, target, auth, pluginConfig)
	if err != nil {
		var rejected trackResyncProviderError
		if errors.As(err, &rejected) {
			// A credential or rate-limit problem concerns the whole instance:
			// record it there like the scheduled sync does.
			if pluginErrorConcernsInstance(rejected.err) {
				recordTrackResyncInstanceStatus(e.App, target.instance.Id, instanceSeen, func(app core.App, instance *core.Record) {
					setPluginInstanceStatusForError(app, instance, rejected.err)
				})
			}
			// Written as plain JSON on purpose: apis.NewApiError rewrites
			// scalar data values into validation errors and would destroy
			// code, message and retry hint.
			return e.JSON(http.StatusBadGateway, providerRejectedBody(rejected))
		}
		return err
	}
	// The provider answered: an error state left by an earlier run no
	// longer applies. Changes made meanwhile are protected by the guard.
	recordTrackResyncInstanceStatus(e.App, target.instance.Id, instanceSeen, clearPluginInstanceErrorAfterSuccess)
	return e.JSON(http.StatusOK, pluginTrackResyncResult{
		TrailID:    target.trail.Id,
		Provider:   target.ref.GetString("provider"),
		ExternalID: target.ref.GetString("external_id"),
		Kind:       outcome.kind,
		Warning:    outcome.warning,
		Track:      outcome.track,
	})
}

// verifyTrackResyncPreviewIdentity makes applying a preview conditional on
// the import reference still naming the provider item the user confirmed.
// It runs before OAuth refresh or any provider call, so a changed reference
// cannot silently fetch a different track.
func verifyTrackResyncPreviewIdentity(target trackResyncTarget, expectedProvider string, expectedExternalID string) error {
	if target.ref.GetString("provider") != expectedProvider ||
		target.ref.GetString("external_id") != expectedExternalID {
		return apis.NewApiError(http.StatusConflict, "import reference changed since preview", nil)
	}
	return nil
}

// clearPluginInstanceErrorAfterSuccess resets whatever error state a
// successful provider call has just disproved: the status, a remembered
// error and a backoff. A rate limit can only be present here once its
// backoff has passed, since the request is refused before that. A disabled
// instance stays as it is.
func clearPluginInstanceErrorAfterSuccess(app core.App, instance *core.Record) {
	if !instance.GetBool("enabled") {
		return
	}
	if instance.GetString("status") == "configured" &&
		len(pluginsystem.JSONMapFromRecord(instance, "last_error")) == 0 &&
		instance.GetDateTime("retry_not_before").IsZero() {
		return
	}
	instance.Set("status", "configured")
	instance.Set("last_error", map[string]any{})
	instance.Set("retry_not_before", "")
	instance.IgnoreUnchangedFields(true)
	if err := app.Save(instance); err != nil {
		app.Logger().Warn("failed to clear plugin instance status after track resync", "instance", instance.Id, "error", err)
	}
}

// trackResyncBackoffSeconds is how long the instance's retry_not_before still
// asks to wait, 0 when it does not.
func trackResyncBackoffSeconds(instance *core.Record) int {
	retryNotBefore := instance.GetDateTime("retry_not_before")
	if retryNotBefore.IsZero() {
		return 0
	}
	wait := time.Until(retryNotBefore.Time())
	if wait <= 0 {
		return 0
	}
	return int(math.Ceil(wait.Seconds()))
}

// trackResyncBackoffError describes a refused request with the error that
// caused the backoff, so the client shows the right reason and countdown.
func trackResyncBackoffError(instance *core.Record, wait int) error {
	lastError := pluginsystem.JSONMapFromRecord(instance, "last_error")
	code := pluginsystem.StringFromAny(lastError["code"])
	if code == "" {
		code = "rate_limited"
	}
	message := pluginsystem.StringFromAny(lastError["message"])
	if message == "" {
		message = code
	}
	return pluginsystem.PluginCapabilityError{Err: &pluginsystem.PluginError{Code: code, Message: message, RetryAfterSeconds: &wait}}
}

// suggestedTrackResyncKind is the likely item kind of a reference without
// recorded kind: imports set the trail's completed flag from the item kind.
func suggestedTrackResyncKind(trail *core.Record) string {
	if trail.GetBool("completed") {
		return util.ExternalReferenceKindCompleted
	}
	return util.ExternalReferenceKindPlanned
}

// recordTrackResyncInstanceStatus applies a status update to a freshly loaded
// copy of the plugin instance, and only if the instance is unchanged since
// this request read it (seen): a status derived from an older state must not
// overwrite what a reconnect, a disable or another run wrote meanwhile. Check
// and write run in one transaction; PocketBase serialises every write
// through a single connection, so no other save can slip in between.
func recordTrackResyncInstanceStatus(app core.App, instanceID string, seen types.DateTime, update func(app core.App, instance *core.Record)) {
	err := app.RunInTransaction(func(txApp core.App) error {
		instance, err := txApp.FindRecordById("plugin_instances", instanceID)
		if err != nil {
			txApp.Logger().Warn("plugin instance vanished before its status could be recorded", "instance", instanceID, "error", err)
			return nil
		}
		// Compared as stored: the database keeps millisecond precision.
		if instance.GetDateTime("updated").String() != seen.String() {
			txApp.Logger().Info("plugin instance changed during track resync, leaving its status alone", "instance", instanceID)
			return nil
		}
		update(txApp, instance)
		return nil
	})
	if err != nil {
		app.Logger().Warn("failed to record plugin instance status after track resync", "instance", instanceID, "error", err)
	}
}

// resolveTrackResyncTarget checks that the user owns the trail, that the
// trail has exactly one reference that may describe its track, that the
// kind of item is known (recorded on the reference, or stated by the user
// for references imported before it was recorded; it is never guessed,
// because the built-in plugins report a fetch on the wrong endpoint like an
// outage and an unrelated item could share the id), and that an enabled
// instance of a plugin with the matching detail capability exists. A reason
// is returned instead of an error for conditions the client should explain
// to the user; for kind_required the target is returned as well so the
// preview can describe the reference.
func resolveTrackResyncTarget(app core.App, userID string, trailID string, statedKind string, loadPlugin func(core.App, string) (pluginsystem.LocalPlugin, error)) (trackResyncTarget, string, error) {
	var target trackResyncTarget

	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return target, trackResyncReasonNotFound, nil
		}
		return target, "", err
	}
	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return target, trackResyncReasonNotOwner, nil
		}
		return target, "", err
	}
	if trail.GetString("author") != actor.Id {
		return target, trackResyncReasonNotOwner, nil
	}

	refs, err := app.FindRecordsByFilter(
		"trail_external_reference",
		"trail={:trail} && user={:user}",
		"",
		-1,
		0,
		dbx.Params{"trail": trail.Id, "user": userID},
	)
	if err != nil {
		return target, "", err
	}
	if len(refs) == 0 {
		return target, trackResyncReasonNotImported, nil
	}
	// A merge moves the source's references onto the target while the target
	// keeps its own track: moved references never describe the track. Trails
	// with several remaining references were merged before moves were
	// recorded; which reference produced the track is unknown, so none may
	// replace it.
	candidates := make([]*core.Record, 0, len(refs))
	for _, ref := range refs {
		if ref.GetString("track_source") != util.TrackSourceMoved {
			candidates = append(candidates, ref)
		}
	}
	if len(candidates) != 1 {
		return target, trackResyncReasonAmbiguous, nil
	}
	ref := candidates[0]

	instance, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id} && enabled=true",
		dbx.Params{"user": userID, "plugin_id": ref.GetString("plugin_id")},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return target, trackResyncReasonDisabled, nil
		}
		return target, "", err
	}
	plugin, err := loadPlugin(app, ref.GetString("plugin_id"))
	if err != nil {
		return target, trackResyncReasonNoPlugin, nil
	}
	target = trackResyncTarget{trail: trail, ref: ref, instance: instance, plugin: plugin, kind: ref.GetString("kind")}
	if target.kind == "" {
		switch statedKind {
		case util.ExternalReferenceKindPlanned, util.ExternalReferenceKindCompleted:
			target.kind, target.kindStated = statedKind, true
		case "":
			if !pluginHasAnyDetailCapability(plugin) {
				return target, trackResyncReasonNoCapability, nil
			}
			return target, trackResyncReasonKindRequired, nil
		default:
			return target, trackResyncReasonKindRequired, nil
		}
	}
	if _, ok := detailCapabilityForKind(plugin, target.kind); !ok {
		return target, trackResyncReasonNoCapability, nil
	}
	return target, "", nil
}

func pluginHasAnyDetailCapability(plugin pluginsystem.LocalPlugin) bool {
	for _, descriptor := range syncCapabilityDescriptors {
		if pluginHasCapability(plugin, descriptor.DetailName, descriptor.Version) {
			return true
		}
	}
	return false
}

// trackResyncProviderError marks a failure of the provider or plugin to
// deliver the item, as opposed to a failure on our side.
type trackResyncProviderError struct {
	err error
}

func (e trackResyncProviderError) Error() string {
	return e.err.Error()
}

func (e trackResyncProviderError) Unwrap() error {
	return e.err
}

// providerRejectedBody is the response body for a provider failure. It follows
// the shape of PocketBase API errors ({status, message, data}) so the web
// proxy forwards data as the error detail, where the client reads the plugin
// error code, message and retry hint.
func providerRejectedBody(rejected trackResyncProviderError) map[string]any {
	return map[string]any{
		"status":  http.StatusBadGateway,
		"message": trackResyncErrProviderRejected,
		"data":    rejected.data(),
	}
}

// data is the client-facing payload: the plugin's structured error when
// there is one; anything else (transport, an unusable answer) is reported as
// provider_unavailable so the client always has a code to translate.
func (e trackResyncProviderError) data() map[string]any {
	if pluginErr := pluginErrorOf(e.err); pluginErr != nil && pluginErr.Code != "" {
		data := map[string]any{"code": pluginErr.Code, "message": pluginErr.Message}
		if pluginErr.RetryAfterSeconds != nil && *pluginErr.RetryAfterSeconds > 0 {
			data["retryAfterSeconds"] = *pluginErr.RetryAfterSeconds
		} else if update := pluginsystem.InstanceStatusForPluginError(*pluginErr, time.Now()); update.RetryNotBefore != nil {
			// The instance gets the default backoff for this error; the
			// client should know it right away instead of on its next try.
			data["retryAfterSeconds"] = int(math.Ceil(time.Until(*update.RetryNotBefore).Seconds()))
		}
		return data
	}
	return map[string]any{"code": "provider_unavailable", "message": e.err.Error()}
}

// pluginErrorOf extracts the structured plugin error carried by a capability
// or call error.
func pluginErrorOf(err error) *pluginsystem.PluginError {
	var capabilityErr pluginsystem.PluginCapabilityError
	if errors.As(err, &capabilityErr) && capabilityErr.Err != nil {
		return capabilityErr.Err
	}
	var callErr pluginsystem.PluginCallError
	if errors.As(err, &callErr) {
		return &callErr.PluginError
	}
	return nil
}

// pluginErrorConcernsInstance reports whether a plugin error is about the
// instance's credentials or rate limit rather than the requested item.
// Trying another detail capability would only repeat such a failure.
// Outages are deliberately not included: the built-in plugins report a 404
// of the wrong route/activity endpoint as provider_unavailable too, and a
// reference without recorded kind needs that second endpoint.
func pluginErrorConcernsInstance(err error) bool {
	pluginErr := pluginErrorOf(err)
	if pluginErr == nil {
		return false
	}
	switch pluginErr.Code {
	case "auth_failed", "invalid_grant", "unauthorized", "rate_limited":
		return true
	}
	return false
}

// executeTrackResync fetches the item through the detail capability matching
// the target kind, checks that the answer identifies the item and carries a
// usable track, then replaces the track of the freshly loaded trail. A kind
// the user stated is recorded on the reference once the fetch succeeded.
func executeTrackResync(ctx context.Context, app core.App, session pluginsystem.RuntimeSession, target trackResyncTarget, auth map[string]any, pluginConfig map[string]any) (trackResyncOutcome, error) {
	served := target.kind
	capability, ok := detailCapabilityForKind(target.plugin, served)
	if !ok {
		return trackResyncOutcome{}, apis.NewBadRequestError(trackResyncReasonNoCapability, nil)
	}
	summary := pluginsystem.TrailSummary{
		Source: pluginsystem.TrailImportSource{
			Provider:   target.ref.GetString("provider"),
			ExternalID: target.ref.GetString("external_id"),
		},
		Kind: served,
	}
	item, err := pluginDetail(ctx, session, target.plugin, capability, target.instance, auth, pluginConfig, summary)
	if err == nil {
		err = validateTrackResyncItem(item, summary)
	}
	if err == nil {
		err = importer.ValidateTrack(item.Track)
	}
	if err != nil {
		return trackResyncOutcome{}, trackResyncProviderError{err: err}
	}

	// The provider call took time: re-read reference and trail and make sure
	// the reference still describes this trail for this user (a merge may
	// have moved it meanwhile) and that nobody replaced the track in between,
	// before anything is written. Track and reference are written in one
	// transaction.
	written := false
	err = app.RunInTransaction(func(txApp core.App) error {
		ref, err := txApp.FindRecordById("trail_external_reference", target.ref.Id)
		if err != nil {
			return apis.NewBadRequestError("import reference disappeared meanwhile", err)
		}
		if ref.GetString("trail") != target.trail.Id ||
			ref.GetString("user") != target.ref.GetString("user") ||
			ref.GetString("provider") != target.ref.GetString("provider") ||
			ref.GetString("external_id") != target.ref.GetString("external_id") ||
			ref.GetString("track_source") == util.TrackSourceMoved {
			return apis.NewBadRequestError("import reference changed meanwhile", nil)
		}
		// A kind recorded meanwhile (by an import or another resync) must
		// agree with the capability that served this item.
		if recorded := ref.GetString("kind"); recorded != "" && recorded != served {
			return apis.NewApiError(http.StatusConflict, "import reference kind changed meanwhile", nil)
		}
		trail, err := txApp.FindRecordById("trails", target.trail.Id)
		if err != nil {
			return apis.NewBadRequestError("trail disappeared meanwhile", err)
		}
		if trail.GetString("author") != target.trail.GetString("author") {
			return apis.NewForbiddenError("trail changed owner meanwhile", nil)
		}
		// Every upload gets a new file name: a different name means someone
		// replaced the track while the provider call was running, and that
		// edit must not be lost. Other fields are preserved anyway, since
		// only the track and its derived values are written.
		if trail.GetString("gpx") != target.trail.GetString("gpx") {
			return apis.NewApiError(http.StatusConflict, "trail track changed meanwhile", nil)
		}

		if err := importer.ReplaceTrailTrack(txApp, trail, item); err != nil {
			return err
		}
		if ref.GetString("kind") == "" {
			ref.Set("kind", served)
		}
		ref.Set("track_resynced_at", time.Now())
		ref.IgnoreUnchangedFields(true)
		if err := txApp.Save(ref); err != nil {
			return err
		}
		written = true
		return nil
	})
	if err != nil && !written {
		return trackResyncOutcome{}, err
	}
	// Geometry is part of the primary trail write. The after-update hooks still
	// verify it against the stored GPX and update the search index/federation
	// after the transaction commits, so read the trail again for the answer.
	stored, readErr := app.FindRecordById("trails", target.trail.Id)
	if readErr != nil {
		return trackResyncOutcome{}, errors.Join(err, readErr)
	}
	if stored.GetString("gpx") == target.trail.GetString("gpx") {
		// Our writes did not reach the database after all.
		if err == nil {
			err = fmt.Errorf("trail track unchanged after resync")
		}
		return trackResyncOutcome{}, err
	}
	outcome := trackResyncOutcome{kind: served, track: trackFieldsOf(stored)}
	if err != nil {
		// PocketBase reports a failed post-commit hook through the same
		// error. The new track is in place, so the resync did succeed; the
		// failed follow-up is reported as a warning instead of sending the
		// user into a pointless retry.
		app.Logger().Warn("track resynced, but a follow-up of the trail update failed", "trail", target.trail.Id, "error", err)
		outcome.warning = "trail_update_followup_failed"
		return outcome, nil
	}
	app.Logger().Info("resynced plugin trail track", "plugin", target.plugin.Manifest.ID, "instance", target.instance.Id, "provider", target.ref.GetString("provider"), "external_id", target.ref.GetString("external_id"), "kind", served, "trail", target.trail.Id)
	return outcome, nil
}

// validateTrackResyncItem rejects detail answers that do not identify the
// requested item exactly.
func validateTrackResyncItem(item pluginsystem.TrailImport, summary pluginsystem.TrailSummary) error {
	if item.Source.ExternalID != summary.Source.ExternalID {
		return fmt.Errorf("detail answered for external id %q instead of %q", item.Source.ExternalID, summary.Source.ExternalID)
	}
	if item.Source.Provider != summary.Source.Provider {
		return fmt.Errorf("detail answered for provider %q instead of %q", item.Source.Provider, summary.Source.Provider)
	}
	if item.Kind != summary.Kind {
		return fmt.Errorf("detail answered with kind %q instead of %q", item.Kind, summary.Kind)
	}
	return nil
}

// detailCapabilityForKind resolves the detail capability that serves
// references of the given kind.
func detailCapabilityForKind(plugin pluginsystem.LocalPlugin, kind string) (pluginsystem.CapabilityManifest, bool) {
	for _, descriptor := range syncCapabilityDescriptors {
		if descriptor.OptionKey != kind || !pluginHasCapability(plugin, descriptor.DetailName, descriptor.Version) {
			continue
		}
		capability, err := pluginCapability(plugin, descriptor.DetailName, descriptor.Version)
		if err != nil {
			return pluginsystem.CapabilityManifest{}, false
		}
		return capability, true
	}
	return pluginsystem.CapabilityManifest{}, false
}
