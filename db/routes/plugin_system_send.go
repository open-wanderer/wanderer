package routes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"

	"pocketbase/pluginsystem"
	"pocketbase/util"
)

type pluginSystemSendRouteRequest struct {
	PluginID string `json:"pluginId"`
	TrailID  string `json:"trailId"`
	Share    string `json:"share,omitempty"`
}

type pluginSystemSendRouteInput struct {
	Instance pluginsystem.InstanceRef `json:"instance"`
	Auth     map[string]any           `json:"auth,omitempty"`
	Config   map[string]any           `json:"config,omitempty"`
	Name     string                   `json:"name,omitempty"`
	Route    pluginsystem.Track       `json:"route"`
}

// PluginSystemSendRoute asks a plugin to prepare an upload request for an
// existing trail and then executes that request through the host policy layer.
func PluginSystemSendRoute(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginSystemSendRouteRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.PluginID == "" || data.TrailID == "" {
		return apis.NewBadRequestError("pluginId and trailId are required", nil)
	}

	instance, err := e.App.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id} && enabled=true",
		dbx.Params{"user": e.Auth.Id, "plugin_id": data.PluginID},
	)
	if err != nil {
		return apis.NewBadRequestError("no enabled plugin instance configured for this plugin", nil)
	}

	plugin, capability, err := localPluginCapability(e.App, data.PluginID, "prepare_send_route", "v1")
	if err != nil {
		return err
	}

	trail, err := e.App.FindRecordById("trails", data.TrailID)
	if err != nil {
		return apis.NewNotFoundError("trail not found", nil)
	}
	if !util.TrailViewableByUser(e.App, trail, e.Auth.Id, data.Share) {
		return apis.NewForbiddenError("not allowed to send this trail", nil)
	}

	gpx, err := readTrailGPX(e.App, trail)
	if err != nil {
		return err
	}
	if len(gpx) == 0 {
		return apis.NewBadRequestError("trail has no GPX track", nil)
	}

	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		return err
	}

	input := pluginSystemSendRouteInput{
		Instance: pluginsystem.InstanceRef{
			ID:       instance.Id,
			PluginID: instance.GetString("plugin_id"),
		},
		Auth: pluginsystem.PluginInputAuth(plugin, auth),
		Name: trail.GetString("name"),
		Route: pluginsystem.Track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpx),
		},
	}
	config := pluginRuntimeConfig(effectivePluginConfig(e.App, plugin.Manifest.ID, instance))
	policy := pluginInstancePolicy(plugin, config)
	input.Config = config
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return err
	}

	runtime, err := pluginsystem.NewRuntimeRegistry().RuntimeFor(plugin)
	if err != nil {
		return err
	}
	output, err := runtime.Call(e.Request.Context(), plugin, capability.Export, inputBytes, policy)
	if err != nil {
		return err
	}

	var plan pluginsystem.UploadPlan
	if err := json.Unmarshal(output, &plan); err != nil {
		return apis.NewBadRequestError("plugin returned an invalid upload plan", err)
	}
	if plan.Request.Method == "" {
		return apis.NewBadRequestError("plugin returned an empty upload request", nil)
	}
	if err := pluginsystem.ValidateHostRequestSpec(plugin.Manifest, plan.Request, policy); err != nil {
		return apis.NewBadRequestError("plugin upload request is not permitted by manifest", err)
	}

	if err := pluginsystem.InjectHostRequestAuth(e.Request.Context(), pluginsystem.AuthInjectionInput{
		App:      e.App,
		Runtime:  runtime,
		Plugin:   plugin,
		Instance: instance,
		Auth:     auth,
		Config:   config,
		Spec:     &plan.Request,
		Policy:   policy,
	}); err != nil {
		return apis.NewBadRequestError("plugin auth injection failed", err)
	}
	if err := executeHostRequest(e.Request.Context(), plugin.Manifest, policy, plan.Request, gpx); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]any{"ok": true})
}

// executeHostRequest runs a plugin upload plan through the shared host request
// executor and maps provider failures to API errors.
func executeHostRequest(ctx context.Context, manifest pluginsystem.Manifest, policy pluginsystem.RequestPolicyContext, spec pluginsystem.HostRequestSpec, gpx []byte) error {
	resp, err := pluginsystem.ExecuteHostRequest(ctx, manifest, policy, spec, pluginsystem.HostRequestOptions{
		Route: gpx,
	})
	if err != nil {
		return err
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return apis.NewBadRequestError(
			fmt.Sprintf("provider upload failed: %d", resp.Status),
			strings.TrimSpace(string(resp.Body)),
		)
	}
	return nil
}

// readTrailGPX loads the trail GPX file that can be inserted into a plugin's
// multipart upload plan.
func readTrailGPX(app core.App, trail *core.Record) ([]byte, error) {
	gpxPath := trail.GetString("gpx")
	if gpxPath == "" {
		return nil, nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(trail.BaseFilesPath() + "/" + gpxPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// decryptedInstanceAuth returns auth fields in the shape expected by host-side
// auth injection and plugin input preparation.
func decryptedInstanceAuth(instance *core.Record) (map[string]any, error) {
	auth := pluginsystem.JSONMapFromRecord(instance, "auth")
	if len(auth) == 0 {
		return map[string]any{}, nil
	}
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if encryptionKey == "" {
		return nil, apis.NewBadRequestError("POCKETBASE_ENCRYPTION_KEY not set", nil)
	}
	for key, value := range auth {
		secret, ok := value.(string)
		if !ok || secret == "" || !util.CanDecryptSecret(secret) {
			continue
		}
		decrypted, err := security.Decrypt(secret, encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt %s: %w", key, err)
		}
		auth[key] = string(decrypted)
	}
	return auth, nil
}
