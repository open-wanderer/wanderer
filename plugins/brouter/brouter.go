//go:build tinygo

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/open-wanderer/wanderer/plugins/brouter/core"
	"github.com/open-wanderer/wanderer/plugins/sdk"
)

const (
	brouterConnector       = "api"
	brouterJSONBytes       = 4 * 1024 * 1024
	errProviderUnavailable = "provider_unavailable"
	errNoRoute             = "no_route"
)

var brouterJSONTypes = []string{"application/json", "application/geo+json", "application/vnd.geo+json", "text/plain"}

func errorCode(err error) string {
	if coded, ok := err.(codedError); ok {
		return coded.code
	}
	return errProviderUnavailable
}

func handleRoute(input routingRouteInput) (routeOutput, error) {
	req := input.Request
	if len(req.Anchors) < 2 {
		return routeOutput{}, codedError{code: "invalid_request", message: "at least two anchors are required"}
	}
	profileKey, err := brouterProfileKey(req)
	if err != nil {
		return routeOutput{}, err
	}
	candidateCount := req.Options.Alternatives
	if candidateCount < 1 {
		candidateCount = 1
	}
	if candidateCount > core.MaxRouteCandidates {
		candidateCount = core.MaxRouteCandidates
	}
	candidates := make([]routeCandidate, 0, candidateCount)
	for alternativeIndex := 0; alternativeIndex < candidateCount; alternativeIndex++ {
		profileErrorCode := errProviderUnavailable
		if req.Profile.PreparedKey != "" {
			profileErrorCode = errUnsupportedProfile
		}
		response, err := requestBRouterRoute(req, profileKey, alternativeIndex, profileErrorCode)
		if err != nil {
			if alternativeIndex > 0 {
				continue
			}
			return routeOutput{}, err
		}
		if len(response.Features) == 0 {
			if alternativeIndex > 0 {
				continue
			}
			return routeOutput{}, codedError{code: errNoRoute, message: "BRouter returned no route feature"}
		}
		candidate, err := candidateFromBRouter(req, profileKey, response.Features[0])
		if err != nil {
			if alternativeIndex > 0 {
				continue
			}
			return routeOutput{}, err
		}
		if alternativeIndex == 0 {
			candidate.ID = "primary"
		} else {
			candidate.ID = fmt.Sprintf("alternative-%d", alternativeIndex)
		}
		candidates = append(candidates, candidate)
	}
	return routeOutput{Candidates: candidates}, nil
}

func handleRoundTrip(input routingRoundTripInput) (roundTripOutput, error) {
	req := input.Request
	if req.Start.Lat < -90 || req.Start.Lat > 90 || req.Start.Lon < -180 || req.Start.Lon > 180 {
		return roundTripOutput{}, codedError{code: "invalid_coordinate", message: "round-trip start must be a valid WGS84 coordinate"}
	}
	if req.TargetDistance < 1000 || req.TargetDistance > 300000 || math.IsNaN(req.TargetDistance) || math.IsInf(req.TargetDistance, 0) {
		return roundTripOutput{}, codedError{code: "invalid_request", message: "round-trip targetDistance must be between 1000 and 300000 meters"}
	}
	if req.Direction != nil && (math.IsNaN(*req.Direction) || math.IsInf(*req.Direction, 0)) {
		return roundTripOutput{}, codedError{code: "invalid_request", message: "round-trip direction must be finite"}
	}
	if len(req.Seed) > 128 {
		return roundTripOutput{}, codedError{code: "invalid_request", message: "round-trip seed exceeds 128 characters"}
	}
	profileKey, err := brouterProfileKey(routeRequest{
		Mode:                req.Mode,
		Profile:             req.Profile,
		Preferences:         req.Preferences,
		RequiredPreferences: req.RequiredPreferences,
	})
	if err != nil {
		return roundTripOutput{}, err
	}
	direction := core.RoundTripDirection(req.Direction, req.Seed)
	best, err := core.CalibrateRoundTrip(req.TargetDistance, func(radius float64, attempt int) (routeCandidate, error) {
		response, requestErr := requestBRouterRoundTrip(req.Start, profileKey, radius, direction)
		if requestErr != nil {
			return routeCandidate{}, requestErr
		}
		feature, suggestions, parseErr := parseBRouterRoundTrip(response, req.Start)
		if parseErr != nil {
			return routeCandidate{}, parseErr
		}
		candidate, candidateErr := core.RoundTripCandidateFromFeature(profileKey, feature, suggestions, req.TargetDistance, direction, attempt, parseFloat)
		if candidateErr != nil {
			return routeCandidate{}, codedError{code: errNoRoute, message: candidateErr.Error()}
		}
		return candidate, nil
	})
	if err != nil {
		return roundTripOutput{}, err
	}
	return roundTripOutput{Candidates: []routeCandidate{best}}, nil
}

func requestBRouterRoundTrip(start anchor, profileKey string, radius float64, direction int) (brouterRoundTripFeatureCollection, error) {
	query := []sdk.QueryParam{
		{Name: "lonlats", Value: fmt.Sprintf("%.6f,%.6f", start.Lon, start.Lat)},
		{Name: "profile", Value: profileKey},
		{Name: "engineMode", Value: "4"},
		{Name: "roundTripDistance", Value: strconv.Itoa(int(math.Floor(radius + 0.5)))},
		{Name: "roundTripPoints", Value: strconv.Itoa(core.RoundTripPointCount)},
		{Name: "exportWaypoints", Value: "1"},
		{Name: "format", Value: "geojson"},
	}
	if direction >= 0 {
		query = append(query, sdk.QueryParam{Name: "direction", Value: strconv.Itoa(direction)})
	}
	response, body, err := sdk.Get(brouterConnector, "/brouter", query, map[string]string{
		"Accept": "application/json",
	}, sdk.ResponseExpect{ContentTypes: brouterJSONTypes, MaxBytes: brouterJSONBytes})
	if err != nil {
		return brouterRoundTripFeatureCollection{}, err
	}
	if response.Status == 404 {
		return brouterRoundTripFeatureCollection{}, codedError{code: errNoRoute, message: strings.TrimSpace(string(body))}
	}
	if response.Status < 200 || response.Status >= 300 {
		return brouterRoundTripFeatureCollection{}, brouterProviderError(response.Status, body, errProviderUnavailable)
	}
	var parsed brouterRoundTripFeatureCollection
	if err := json.Unmarshal(body, &parsed); err != nil {
		return brouterRoundTripFeatureCollection{}, err
	}
	return parsed, nil
}

func parseBRouterRoundTrip(response brouterRoundTripFeatureCollection, start anchor) (brouterFeature, []anchor, error) {
	feature, suggestions, err := core.ParseRoundTripFeatures(response.Features, start)
	if err != nil {
		return brouterFeature{}, nil, codedError{code: errNoRoute, message: err.Error()}
	}
	return feature, suggestions, nil
}

func handleProfileIntrospect(input routingProfileIntrospectInput) (profileIntrospectOutput, error) {
	profile := input.Request.Profile
	if profile.ContentBase64 == "" {
		return profileIntrospectOutput{}, nil
	}
	content, err := base64.StdEncoding.DecodeString(profile.ContentBase64)
	if err != nil {
		return profileIntrospectOutput{}, codedError{code: "invalid_request", message: "BRouter profile content is not valid base64"}
	}
	if err := validateBRouterProfileContent(content); err != nil {
		return profileIntrospectOutput{}, err
	}
	mode := core.DetectProfileMode(string(content))
	capabilityMode := mode
	if capabilityMode == "" {
		capabilityMode = profile.Mode
	}
	parameters := parseBRouterProfileParameters(string(content))
	output := profileIntrospectOutput{
		Mode:                 mode,
		SupportedPreferences: core.SupportedStandardPreferences(brouterProfileParameterTypes(string(content)), capabilityMode),
	}
	controls := brouterControlsFromProfileMetadata(parameters, profile.NativeConfig)
	if len(controls) == 0 {
		return output, nil
	}
	output.NativeControlGroups = []nativeControlGroup{{
		Key:   "profile",
		Label: "Profile",
		Labels: map[string]string{
			"de": "Profil",
			"en": "Profile",
		},
		Controls: controls,
	}}
	return output, nil
}

func handleProfilePrepare(input routingProfilePrepareInput) (profilePrepareOutput, error) {
	req := routeRequest{
		Mode:                input.Request.Mode,
		Profile:             input.Request.Profile,
		Preferences:         input.Request.Preferences,
		RequiredPreferences: input.Request.RequiredPreferences,
	}
	req.Profile.PreparedKey = ""
	profileKey, err := brouterProfileKey(req)
	if err != nil {
		return profilePrepareOutput{}, err
	}
	return profilePrepareOutput{PreparedKey: profileKey}, nil
}

func brouterProfileKey(req routeRequest) (string, error) {
	if req.Profile.PreparedKey != "" {
		return req.Profile.PreparedKey, nil
	}
	if req.Profile.ContentBase64 != "" {
		content, err := base64.StdEncoding.DecodeString(req.Profile.ContentBase64)
		if err != nil {
			return "", codedError{code: "invalid_request", message: "BRouter profile content is not valid base64"}
		}
		if err := validateBRouterProfileContent(content); err != nil {
			return "", err
		}
		// Uploaded copies of the well-known profiles get the same translation as
		// the curated ones; anything unrecognized falls back to the generic names.
		templateKey := core.TemplateKeyForParameters(brouterProfileParameterTypes(string(content)))
		nativeConfig := core.NativeConfigWithPreferences(templateKey, req.Profile.NativeConfig, req.Preferences, req.Mode)
		rendered, err := renderBRouterCustomProfile(string(content), nativeConfig)
		if err != nil {
			return "", err
		}
		return uploadBRouterProfile(base64.StdEncoding.EncodeToString([]byte(rendered)))
	}
	if req.Profile.Kind == "generated" {
		templateKey := stringValue(req.Profile.Metadata["templateKey"])
		content, err := renderBRouterGeneratedProfile(req.Profile.Metadata, core.NativeConfigWithPreferences(templateKey, req.Profile.NativeConfig, req.Preferences, req.Mode))
		if err != nil {
			return "", err
		}
		return uploadBRouterProfile(base64.StdEncoding.EncodeToString([]byte(content)))
	}
	if req.Profile.Key != "" {
		// Curated profiles always render and upload their embedded base. Routing
		// with the bare key would silently drop the category preset and would use
		// the provider's own copy of that key, which can differ from ours.
		if templateKey := brouterTemplateKeyForProfile(req.Profile.Key); templateKey != "" {
			metadata := map[string]any{"templateKey": templateKey}
			nativeConfig := core.NativeConfigWithPreferences(templateKey, req.Profile.NativeConfig, req.Preferences, req.Mode)
			content, err := renderBRouterGeneratedProfile(metadata, nativeConfig)
			if err != nil {
				return "", err
			}
			return uploadBRouterProfile(base64.StdEncoding.EncodeToString([]byte(content)))
		}
		return req.Profile.Key, nil
	}
	switch req.Mode {
	case "foot":
		return "hiking-mountain", nil
	case "bike":
		return "trekking", nil
	case "motor":
		return "car-vario", nil
	default:
		return "", codedError{code: errUnsupportedProfile, message: "BRouter routing profile key is required"}
	}
}

func uploadBRouterProfile(contentBase64 string) (string, error) {
	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return "", codedError{code: "invalid_request", message: "BRouter profile content is not valid base64"}
	}
	if err := validateBRouterProfileContent(content); err != nil {
		return "", err
	}
	response, body, err := sdk.PostText(brouterConnector, "/brouter/profile", nil, map[string]string{
		"Accept": "application/json",
	}, string(content), sdk.ResponseExpect{ContentTypes: []string{"application/json", "text/plain"}, MaxBytes: 64 * 1024})
	if err != nil {
		return "", err
	}
	if response.Status < 200 || response.Status >= 300 {
		return "", brouterProviderError(response.Status, body, errUnsupportedProfile)
	}
	var parsed brouterProfileUploadResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if message := brouterUploadErrorMessage(parsed.Error, body); message != "" {
		return "", codedError{code: errUnsupportedProfile, message: message}
	}
	if parsed.ProfileID == "" {
		return "", codedError{code: errUnsupportedProfile, message: "BRouter profile upload returned no profile id"}
	}
	return parsed.ProfileID, nil
}

func brouterControlsFromProfileMetadata(parameters []brouterProfileParameter, nativeConfig map[string]any) []nativeControl {
	if len(parameters) == 0 {
		return nil
	}
	values := mapValue(nativeConfig["parameters"])
	controls := make([]nativeControl, 0, len(parameters))
	for _, parameter := range parameters {
		current := values[parameter.Key]
		if current == nil {
			current = parameter.Default
		}
		control := nativeControl{
			Key:       parameter.Key,
			Label:     parameter.Label,
			Type:      parameter.ValueType,
			ValueType: parameter.ValueType,
			Default:   parameter.Default,
			Current:   current,
			Target:    "native_config",
			Path:      []string{"parameters", parameter.Key},
			Options:   parameter.Options,
		}
		if parameter.ValueType == "boolean" {
			control.Type = "boolean"
			control.ValueType = "boolean"
		} else {
			control.Type = "number"
			control.ValueType = "number"
			if parameter.Bounded {
				control.UI = "slider"
				min := parameter.Min
				max := parameter.Max
				step := parameter.Step
				control.Min = &min
				control.Max = &max
				control.Step = &step
			} else {
				// Without a declared range a slider would invent one and reject
				// values the profile itself uses.
				control.UI = "number"
			}
		}
		controls = append(controls, control)
	}
	return controls
}

func requestBRouterRoute(req routeRequest, profileKey string, alternativeIndex int, profileErrorCode string) (brouterFeatureCollection, error) {
	query := []sdk.QueryParam{
		{Name: "lonlats", Value: brouterLonLats(req.Anchors)},
		{Name: "profile", Value: profileKey},
		{Name: "alternativeidx", Value: strconv.Itoa(alternativeIndex)},
		{Name: "format", Value: "geojson"},
	}
	response, body, err := sdk.Get(brouterConnector, "/brouter", query, map[string]string{
		"Accept": "application/json",
	}, sdk.ResponseExpect{ContentTypes: brouterJSONTypes, MaxBytes: brouterJSONBytes})
	if err != nil {
		return brouterFeatureCollection{}, err
	}
	if response.Status == 404 {
		return brouterFeatureCollection{}, codedError{code: errNoRoute, message: strings.TrimSpace(string(body))}
	}
	if response.Status < 200 || response.Status >= 300 {
		return brouterFeatureCollection{}, brouterProviderError(response.Status, body, profileErrorCode)
	}
	var parsed brouterFeatureCollection
	if err := json.Unmarshal(body, &parsed); err != nil {
		return brouterFeatureCollection{}, err
	}
	return parsed, nil
}

func brouterProviderError(status int, body []byte, clientCode string) error {
	code := brouterProviderErrorCode(status, body, clientCode)
	return codedError{code: code, message: fmt.Sprintf("BRouter request failed (%d): %s", status, brouterProviderMessage(body))}
}

func brouterUploadErrorMessage(errorValue any, body []byte) string {
	switch errorValue := errorValue.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(errorValue)
	case map[string]any:
		if msg := stringValue(errorValue["message"]); msg != "" {
			return msg
		}
		if code := stringValue(errorValue["code"]); code != "" {
			return code
		}
	}
	return brouterProviderMessage(body)
}

func brouterProviderMessage(body []byte) string {
	message := strings.TrimSpace(string(body))
	if message == "" {
		return "empty provider response"
	}
	var parsed map[string]any
	if json.Unmarshal(body, &parsed) != nil {
		return message
	}
	if msg := stringValue(parsed["message"]); msg != "" {
		return msg
	}
	if errValue, ok := parsed["error"]; ok {
		switch errValue := errValue.(type) {
		case string:
			if errValue != "" {
				return errValue
			}
		case map[string]any:
			if msg := stringValue(errValue["message"]); msg != "" {
				return msg
			}
			if code := stringValue(errValue["code"]); code != "" {
				return code
			}
		}
	}
	if code := stringValue(parsed["code"]); code != "" {
		return code
	}
	return message
}

func candidateFromBRouter(req routeRequest, profileKey string, feature brouterFeature) (routeCandidate, error) {
	candidate, err := core.CandidateFromFeature(req, profileKey, feature, parseFloat)
	if err != nil {
		return routeCandidate{}, codedError{code: errNoRoute, message: err.Error()}
	}
	return candidate, nil
}

func brouterLonLats(anchors []anchor) string {
	parts := make([]string, 0, len(anchors))
	for _, a := range anchors {
		parts = append(parts, fmt.Sprintf("%.6f,%.6f", a.Lon, a.Lat))
	}
	return strings.Join(parts, "|")
}
