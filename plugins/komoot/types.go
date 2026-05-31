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
	Token  string `json:"token"`
	Scheme string `json:"scheme,omitempty"`
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
	ExternalID  string   `json:"externalId,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Lat         float64  `json:"lat"`
	Lon         float64  `json:"lon"`
	Ele         *float64 `json:"ele,omitempty"`
	Icon        string   `json:"icon,omitempty"`
}

type photo struct {
	ExternalID  string      `json:"externalId,omitempty"`
	Filename    string      `json:"filename,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	Lat         *float64    `json:"lat,omitempty"`
	Lon         *float64    `json:"lon,omitempty"`
	Source      mediaSource `json:"source"`
}

type mediaSource struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
}

type pluginError = sdk.PluginError

type loginResponse struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type toursResponse struct {
	Embedded toursEmbedded `json:"_embedded"`
	Page     page          `json:"page"`
}

type toursEmbedded struct {
	Tours []tour `json:"tours"`
}

type page struct {
	TotalPages int `json:"totalPages"`
}

type tour struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	Date      string `json:"date"`
	Sport     string `json:"sport"`
	ChangedAt string `json:"changed_at"`
}

type detailedTour struct {
	ID         int64                `json:"id"`
	Type       string               `json:"type"`
	Name       string               `json:"name"`
	Status     string               `json:"status"`
	Date       string               `json:"date"`
	Sport      string               `json:"sport"`
	MapImage   mapImage             `json:"map_image"`
	Difficulty difficulty           `json:"difficulty"`
	ChangedAt  string               `json:"changed_at"`
	Embedded   detailedTourEmbedded `json:"_embedded"`
}

type difficulty struct {
	Grade string `json:"grade"`
}

type mapImage struct {
	Src string `json:"src"`
}

type detailedTourEmbedded struct {
	Coordinates coordinates `json:"coordinates"`
	Timeline    timeline    `json:"timeline"`
	CoverImages coverImages `json:"cover_images"`
}

type coordinates struct {
	Items []coordinate `json:"items"`
}

type coordinate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Alt float64 `json:"alt"`
	T   int     `json:"t"`
}

type timeline struct {
	Embedded timelineEmbedded `json:"_embedded"`
}

type timelineEmbedded struct {
	Items []timelineItem `json:"items"`
}

type timelineItem struct {
	Embedded timelineItemEmbedded `json:"_embedded"`
}

type timelineItemEmbedded struct {
	Reference waypointReference `json:"reference"`
}

type waypointReference struct {
	ID         int64               `json:"id"`
	Name       string              `json:"name"`
	StartPoint point               `json:"start_point"`
	Embedded   waypointSubEmbedded `json:"_embedded"`
}

type point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	Alt float64 `json:"alt"`
}

type waypointSubEmbedded struct {
	Tips tips `json:"tips"`
}

type tips struct {
	Embedded tipsEmbedded `json:"_embedded"`
}

type tipsEmbedded struct {
	Items []tipItem `json:"items"`
}

type tipItem struct {
	Text string `json:"text"`
}

type coverImages struct {
	Embedded imagesEmbedded `json:"_embedded"`
}

type imagesEmbedded struct {
	Items []imageItem `json:"items"`
}

type imageItem struct {
	ID       int64    `json:"id"`
	Src      string   `json:"src"`
	Location location `json:"location"`
	Type     string   `json:"type"`
}

type location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
