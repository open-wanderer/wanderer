package routes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	routingPreparedProfileTTL       = 15 * time.Minute
	routingPreparedProfileCacheSize = 512
	routingPreparedProfileKeyMaxLen = 1024
)

var (
	routingProfilePrepareRateLimiter  = util.NewRateLimiter(20, time.Minute)
	routingProfilePreparePluginCaller = callRoutingProfilePreparePlugin
	routingPreparedProfiles           = newRoutingPreparedProfileCache(routingPreparedProfileTTL, routingPreparedProfileCacheSize)
)

type routingPreparedProfileEntry struct {
	PreparedKey string
	ExpiresAt   time.Time
}

type routingPreparedProfileCall struct {
	Done        chan struct{}
	PreparedKey string
	Err         error
}

type routingPreparedProfileRefresh struct {
	once sync.Once
	key  string
}

type routingPreparedProfileCache struct {
	mu         sync.Mutex
	ttl        time.Duration
	maxEntries int
	entries    map[string]routingPreparedProfileEntry
	inflight   map[string]*routingPreparedProfileCall
}

func newRoutingPreparedProfileCache(ttl time.Duration, maxEntries int) *routingPreparedProfileCache {
	return &routingPreparedProfileCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		entries:    map[string]routingPreparedProfileEntry{},
		inflight:   map[string]*routingPreparedProfileCall{},
	}
}

func (c *routingPreparedProfileCache) getOrPrepare(ctx context.Context, key string, prepare func(context.Context) (string, error)) (string, error) {
	now := time.Now()
	c.mu.Lock()
	if entry, ok := c.entries[key]; ok {
		if now.Before(entry.ExpiresAt) {
			c.mu.Unlock()
			return entry.PreparedKey, nil
		}
		delete(c.entries, key)
	}
	call, ok := c.inflight[key]
	if !ok {
		call = &routingPreparedProfileCall{Done: make(chan struct{})}
		c.inflight[key] = call
		prepareCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), routingPluginTimeout)
		go c.runPreparation(prepareCtx, cancel, key, call, prepare)
	}
	c.mu.Unlock()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-call.Done:
		return call.PreparedKey, call.Err
	}
}

func (c *routingPreparedProfileCache) runPreparation(ctx context.Context, cancel context.CancelFunc, key string, call *routingPreparedProfileCall, prepare func(context.Context) (string, error)) {
	var preparedKey string
	var err error
	defer func() {
		cancel()
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("routing profile preparation panicked: %v", recovered)
		}
		c.finishPreparation(key, call, preparedKey, err)
	}()
	preparedKey, err = prepare(ctx)
	if err == nil && strings.TrimSpace(preparedKey) == "" {
		err = fmt.Errorf("routing profile preparation returned no prepared key")
	}
	if err == nil && len(preparedKey) > routingPreparedProfileKeyMaxLen {
		err = fmt.Errorf("routing profile preparation key exceeds host limit")
	}
}

func (c *routingPreparedProfileCache) finishPreparation(key string, call *routingPreparedProfileCall, preparedKey string, err error) {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	call.PreparedKey = preparedKey
	call.Err = err
	if err == nil {
		c.pruneLocked(now)
		c.entries[key] = routingPreparedProfileEntry{PreparedKey: preparedKey, ExpiresAt: now.Add(c.ttl)}
	}
	if c.inflight[key] == call {
		delete(c.inflight, key)
	}
	close(call.Done)
}

func (c *routingPreparedProfileCache) invalidate(key string, preparedKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if ok && (preparedKey == "" || entry.PreparedKey == preparedKey) {
		delete(c.entries, key)
	}
}

func (c *routingPreparedProfileCache) pruneLocked(now time.Time) {
	for key, entry := range c.entries {
		if !now.Before(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
	for c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		oldestKey := ""
		var oldestExpiry time.Time
		for key, entry := range c.entries {
			if oldestKey == "" || entry.ExpiresAt.Before(oldestExpiry) {
				oldestKey = key
				oldestExpiry = entry.ExpiresAt
			}
		}
		if oldestKey == "" {
			break
		}
		delete(c.entries, oldestKey)
	}
}

func PluginSystemRoutingProfilePrepare(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	if err := routingProfilePrepareRateLimiter.CheckRateLimit(e.Auth.Id, "profile-prepare"); err != nil {
		return e.TooManyRequestsError("too many routing profile preparation requests", err)
	}
	var data pluginRoutingRouteHTTPInput
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.EngineMode != "" && data.EngineMode != "single" && data.EngineMode != "parallel" {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "engineMode must be single or parallel"), nil)
	}
	if err := validateRoutingInlineProfileContent(data.Profile.ContentBase64); err != nil {
		return routingJSONError(e, err, nil)
	}
	if err := validateRoutingPreferences(data.Preferences, false); err != nil {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "invalid routing preferences: "+err.Error()), nil)
	}
	if err := validateRoutingConfig(data.Profile.NativeConfig, "profile.nativeConfig"); err != nil {
		return routingJSONError(e, routingErrorFromCode("invalid_request", err.Error()), nil)
	}
	if err := validateRoutingRequiredPreferences(data.RequiredPreferences); err != nil {
		return routingJSONError(e, routingErrorFromCode("invalid_request", err.Error()), nil)
	}
	settings, err := ResolveRoutingSettings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	if !routingFeatureEnabled(settings, "routing", true) {
		return routingJSONError(e, routingErrorFromCode("routing_disabled", "routing is not enabled"), nil)
	}
	routingMode := strings.TrimSpace(data.RoutingMode)
	if routingMode == "" {
		routingMode = settings.DefaultRoutingMode
	}
	if routingMode == "" {
		routingMode = "segment"
	}
	if routingMode != "segment" && routingMode != "via" {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "routingMode must be segment or via"), nil)
	}
	parallel := routingUsesParallelEngines(data, data.RequestVariants)
	if parallel && !routingFeatureEnabled(settings, "parallelRouting", true) {
		parallel = false
		data.EngineMode = "single"
		data.Engines = nil
	}
	selections, _, err := routingSelectionsForRequest(e.App, e.Auth.Id, data, settings, parallel)
	if err != nil {
		return routingJSONError(e, err, nil)
	}
	baseRequest := data.pluginRoutingRouteRequest
	baseRequest.Anchors = nil
	baseRequest.RoutingMode = routingMode
	baseRequest.Options = pluginRoutingOptions{}
	baseRequest.Profile.PreparedKey = ""
	runtimes, engineErrors := routingRuntimesForMode(e, selections, baseRequest, routingMode, 1, false, &settings)
	if len(runtimes) == 0 {
		return routingJSONError(e, routingNoCandidateError(engineErrors), map[string]any{"engineErrors": engineErrors})
	}
	routingCtx, closeRoutingSessions := withRoutingPluginSessions(e.Request.Context())
	defer closeRoutingSessions()
	preparedRuntimes, preparationErrors := prepareRoutingRuntimeProfiles(routingCtx, runtimes)
	engineErrors = append(engineErrors, preparationErrors...)
	preparedCount := 0
	for _, runtime := range preparedRuntimes {
		if runtime.Request.Profile.PreparedKey != "" {
			preparedCount++
		}
	}
	return e.JSON(http.StatusOK, map[string]any{
		"prepared":     preparedCount,
		"engineErrors": compactRoutingEngineErrors(engineErrors),
	})
}

func prepareRoutingRuntimeProfiles(ctx context.Context, runtimes []routingEngineRuntime) ([]routingEngineRuntime, []pluginRoutingEngineError) {
	prepared := append([]routingEngineRuntime(nil), runtimes...)
	var errorsMu sync.Mutex
	engineErrors := []pluginRoutingEngineError{}
	var wg sync.WaitGroup
	for index := range runtimes {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			runtime, err := prepareRoutingRuntimeProfile(ctx, runtimes[index])
			if err != nil {
				prepared[index].Request.Profile.PreparedKey = ""
				prepared[index].PreparedProfileFingerprint = ""
				errorsMu.Lock()
				engineErrors = append(engineErrors, routingEngineErrorFromError(err, runtimes[index].Plugin, runtimes[index].Instance))
				errorsMu.Unlock()
				return
			}
			prepared[index] = runtime
		}()
	}
	wg.Wait()
	return prepared, engineErrors
}

func prepareRoutingRuntimeProfile(ctx context.Context, runtime routingEngineRuntime) (routingEngineRuntime, error) {
	capability, ok := routingProfilePrepareCapability(runtime.Plugin)
	if !ok {
		return runtime, nil
	}
	fingerprint, err := routingPreparedProfileFingerprint(runtime)
	if err != nil {
		return runtime, err
	}
	preparedKey, err := routingPreparedProfiles.getOrPrepare(ctx, fingerprint, func(ctx context.Context) (string, error) {
		request := pluginRoutingProfilePrepareRequest{
			Mode:                runtime.Request.Mode,
			Profile:             runtime.Request.Profile,
			Preferences:         cloneRoutingMap(runtime.Request.Preferences),
			RequiredPreferences: append([]string(nil), runtime.Request.RequiredPreferences...),
		}
		request.Profile.PreparedKey = ""
		output, callErr := routingProfilePreparePluginCaller(ctx, runtime.Plugin, capability, runtime.Instance, runtime.Auth, runtime.Config, request)
		if callErr != nil {
			return "", callErr
		}
		if output.Error != nil {
			return "", routingErrorFromPluginError(*output.Error)
		}
		return output.PreparedKey, nil
	})
	if err != nil {
		return runtime, err
	}
	runtime.Request.Profile.PreparedKey = preparedKey
	runtime.PreparedProfileFingerprint = fingerprint
	return runtime, nil
}

func callPreparedRoutingRoutePlugin(ctx context.Context, runtime routingEngineRuntime, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
	output, err := routingRoutePluginCaller(ctx, runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, request)
	if request.Profile.PreparedKey == "" || !routingPreparedProfileRetryable(output, err) {
		return output, err
	}

	refresh := refreshPreparedRoutingProfile(ctx, runtime, request.Profile.PreparedKey, request)
	// All segment callers wait for the same refresh. An empty key deliberately
	// selects the adapter's canonical inline-profile fallback.
	request.Profile.PreparedKey = refresh.key
	return routingRoutePluginCaller(ctx, runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, request)
}

func refreshPreparedRoutingProfile(ctx context.Context, runtime routingEngineRuntime, rejectedKey string, request pluginRoutingRouteRequest) *routingPreparedProfileRefresh {
	refresh := runtime.PreparedProfileRefresh
	if refresh == nil {
		refresh = &routingPreparedProfileRefresh{}
	}
	refresh.once.Do(func() {
		fingerprint := runtime.PreparedProfileFingerprint
		if fingerprint == "" {
			fingerprint, _ = routingPreparedProfileFingerprint(runtime)
		}
		routingPreparedProfiles.invalidate(fingerprint, rejectedKey)

		refreshRuntime := runtime
		refreshRuntime.Request = cloneRoutingRouteRequest(request)
		refreshRuntime.Request.Profile.PreparedKey = ""
		refreshRuntime.PreparedProfileFingerprint = ""
		refreshed, prepareErr := prepareRoutingRuntimeProfile(ctx, refreshRuntime)
		if prepareErr == nil {
			refresh.key = refreshed.Request.Profile.PreparedKey
		}
	})
	return refresh
}

func routingPreparedProfileRetryable(output pluginRoutingRouteOutput, err error) bool {
	if output.Error != nil {
		return output.Error.Code == "unsupported_profile"
	}
	if err == nil {
		return false
	}
	var callErr pluginsystem.PluginCallError
	if errors.As(err, &callErr) {
		return callErr.PluginError.Code == "unsupported_profile"
	}
	var routingErr *routingError
	if errors.As(err, &routingErr) {
		return routingErr.Code == "unsupported_profile"
	}
	return false
}

func routingProfilePrepareCapability(plugin pluginsystem.LocalPlugin) (pluginsystem.CapabilityManifest, bool) {
	for _, capability := range plugin.Manifest.Capabilities {
		if capability.Name == "profile_prepare" && capability.Version == "v1" {
			return capability, true
		}
	}
	return pluginsystem.CapabilityManifest{}, false
}

func routingPreparedProfileFingerprint(runtime routingEngineRuntime) (string, error) {
	profile := runtime.Request.Profile
	profile.PreparedKey = ""
	payload, err := json.Marshal(map[string]any{
		"pluginId":            runtime.Plugin.Manifest.ID,
		"pluginVersion":       runtime.Plugin.Manifest.Version,
		"instanceId":          instanceID(runtime.Instance),
		"auth":                runtime.Auth,
		"config":              runtime.Config,
		"mode":                runtime.Request.Mode,
		"profile":             profile,
		"preferences":         runtime.Request.Preferences,
		"requiredPreferences": runtime.Request.RequiredPreferences,
	})
	if err != nil {
		return "", fmt.Errorf("encode routing profile preparation fingerprint: %w", err)
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}
