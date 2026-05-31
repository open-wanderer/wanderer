package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type instanceRef struct {
	ID       string `json:"id"`
	PluginID string `json:"pluginId"`
}

type refreshSessionInput struct {
	Instance instanceRef    `json:"instance"`
	Auth     map[string]any `json:"auth,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type refreshSessionOutput struct {
	Token     string `json:"token"`
	Scheme    string `json:"scheme,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

type sendRouteInput struct {
	Instance instanceRef    `json:"instance"`
	Auth     map[string]any `json:"auth,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Name     string         `json:"name,omitempty"`
	Route    track          `json:"route"`
}

type listInput struct {
	Instance          instanceRef    `json:"instance"`
	Auth              map[string]any `json:"auth,omitempty"`
	State             map[string]any `json:"state,omitempty"`
	Options           map[string]any `json:"options,omitempty"`
	Limits            syncLimits     `json:"limits,omitempty"`
	RecentExternalIDs []string       `json:"recentExternalIds,omitempty"`
}

type syncLimits struct {
	MaxItems int `json:"maxItems,omitempty"`
}

type listOutput struct {
	Items   []trailImport  `json:"items"`
	State   map[string]any `json:"state,omitempty"`
	HasMore bool           `json:"hasMore"`
	Error   *pluginError   `json:"error,omitempty"`
}

type trailImport struct {
	Source       trailImportSource `json:"source"`
	Kind         string            `json:"kind,omitempty"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	StartedAt    string            `json:"startedAt,omitempty"`
	ActivityType string            `json:"activityType,omitempty"`
	Privacy      *string           `json:"privacy,omitempty"`
	Track        track             `json:"track"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

type trailImportSource struct {
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	URL        string `json:"url,omitempty"`
}

type track struct {
	Format        string `json:"format"`
	ContentBase64 string `json:"contentBase64"`
}

type uploadPlan struct {
	Request sdk.HostRequestSpec `json:"request"`
}

type loginResponse struct {
	Token string `json:"access_token"`
}

type toursResponse struct {
	TotalPages int            `json:"totalPages"`
	Data       []tourResponse `json:"data"`
}

type tourResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type activitiesResponse struct {
	TotalPages int                `json:"totalPages"`
	Data       []activityResponse `json:"data"`
}

type activityResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type tour struct {
	ID            string    `json:"id"`
	CreatedAt     string    `json:"createdAt"`
	Name          string    `json:"name"`
	Distance      float64   `json:"distance"`
	Elevation     elevation `json:"elevation"`
	StartLocation location  `json:"startLocation"`
	RoutePolyline string    `json:"routePolyline"`
	IsPublic      bool      `json:"isPublic"`
}

type elevation struct {
	Gain     float64 `json:"gain"`
	Loss     float64 `json:"loss"`
	Polyline string  `json:"polyline"`
}

type location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type activity struct {
	ActivityData activityData `json:"activityData"`
	RecordData   recordData   `json:"recordData"`
}

type activityData struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	CreatedAt    string      `json:"createdAt"`
	ActivityInfo []info      `json:"activityInfo"`
	Laps         []lapDetail `json:"laps"`
	ActivityType string      `json:"activityType"`
}

type info struct {
	Key   string    `json:"key"`
	Value infoValue `json:"value"`
}

type infoValue struct {
	Value float64 `json:"value"`
}

type lapDetail struct {
	ActiveTime int `json:"activeTime"`
}

type recordData struct {
	Timestamp []int     `json:"timestamp"`
	Elevation []float64 `json:"elevation"`
	Lat       []float64 `json:"lat"`
	Lng       []float64 `json:"lng"`
}

type pluginError = sdk.PluginError
