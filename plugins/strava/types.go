package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type instanceRef struct {
	ID       string `json:"id"`
	PluginID string `json:"pluginId"`
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
	Waypoints    []waypoint        `json:"waypoints,omitempty"`
	Photos       []photo           `json:"photos,omitempty"`
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

type waypoint struct {
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Icon        string  `json:"icon,omitempty"`
}

type photo struct {
	ExternalID string      `json:"externalId,omitempty"`
	Filename   string      `json:"filename,omitempty"`
	Source     mediaSource `json:"source"`
}

type mediaSource struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
}

type pluginError = sdk.PluginError

type route struct {
	Description         string          `json:"description"`
	IDStr               string          `json:"id_str"`
	Name                string          `json:"name"`
	Private             bool            `json:"private"`
	Timestamp           int64           `json:"timestamp"`
	Type                int             `json:"type"`
	CreatedAt           string          `json:"created_at"`
	EstimatedMovingTime int             `json:"estimated_moving_time"`
	Waypoints           []routeWaypoint `json:"waypoints"`
}

type routeWaypoint struct {
	Latlng      []float64 `json:"latlng"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

type activity struct {
	ID int64 `json:"id"`
}

type detailedActivity struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Private     bool      `json:"private"`
	StartDate   string    `json:"start_date"`
	StartLatlng []float64 `json:"start_latlng"`
	Type        string    `json:"type"`
	SportType   string    `json:"sport_type"`
	Photos      photos    `json:"photos"`
}

type photos struct {
	Count   int          `json:"count"`
	Primary primaryPhoto `json:"primary"`
}

type primaryPhoto struct {
	ID   int64     `json:"id"`
	Urls photoURLs `json:"urls"`
}

type photoURLs struct {
	Num100 string `json:"100"`
	Num600 string `json:"600"`
}

type activityPhoto struct {
	UniqueID string    `json:"unique_id"`
	Urls     photoURLs `json:"urls"`
}

type activityStreamResponse struct {
	LatLng   streamLatLng  `json:"latlng"`
	Time     streamInt     `json:"time"`
	Altitude streamFloat64 `json:"altitude"`
}

type streamLatLng struct {
	Data [][]float64 `json:"data"`
}

type streamInt struct {
	Data []int `json:"data"`
}

type streamFloat64 struct {
	Data []float64 `json:"data"`
}
