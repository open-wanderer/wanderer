package pluginsystem

import (
	"encoding/json"
	"time"
)

type InstanceRef struct {
	ID       string `json:"id"`
	PluginID string `json:"pluginId"`
}

type TrailImport struct {
	Source       TrailImportSource `json:"source"`
	Kind         string            `json:"kind,omitempty"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	StartedAt    *time.Time        `json:"startedAt,omitempty"`
	ActivityType string            `json:"activityType,omitempty"`
	Privacy      *string           `json:"privacy,omitempty"`
	Track        Track             `json:"track"`
	Waypoints    []Waypoint        `json:"waypoints,omitempty"`
	Photos       []Photo           `json:"photos,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`

	// Difficulty is the provider's coarse rating: easy, moderate, or difficult.
	// Omit it when unknown; this is not a universal physical or technical scale.
	Difficulty string `json:"difficulty,omitempty"`
}

// UnmarshalJSON keeps malformed optional difficulty values from rejecting an
// otherwise valid import. All other fields retain their normal JSON validation.
func (item *TrailImport) UnmarshalJSON(data []byte) error {
	type trailImport TrailImport
	var decoded trailImport
	wire := struct {
		*trailImport
		Difficulty json.RawMessage `json:"difficulty"`
	}{trailImport: &decoded}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var difficulty string
	if err := json.Unmarshal(wire.Difficulty, &difficulty); err == nil {
		decoded.Difficulty = difficulty
	}
	*item = TrailImport(decoded)
	return nil
}

type TrailSummary struct {
	Source TrailImportSource `json:"source"`
	Kind   string            `json:"kind,omitempty"`
}

type TrailImportSource struct {
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	URL        string `json:"url,omitempty"`
}

type Track struct {
	Format        string `json:"format"`
	ContentBase64 string `json:"contentBase64"`
}

type Waypoint struct {
	ExternalID  string     `json:"externalId,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Lat         float64    `json:"lat"`
	Lon         float64    `json:"lon"`
	Ele         *float64   `json:"ele,omitempty"`
	Time        *time.Time `json:"time,omitempty"`
	Icon        string     `json:"icon,omitempty"`
	Photos      []Photo    `json:"photos,omitempty"`
}

type Photo struct {
	ExternalID  string      `json:"externalId,omitempty"`
	Filename    string      `json:"filename,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	TakenAt     *time.Time  `json:"takenAt,omitempty"`
	Lat         *float64    `json:"lat,omitempty"`
	Lon         *float64    `json:"lon,omitempty"`
	Source      MediaSource `json:"source"`
}

type MediaSource struct {
	Type      string     `json:"type"`
	URL       string     `json:"url,omitempty"`
	MediaRef  *MediaRef  `json:"mediaRef,omitempty"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

type MediaRef struct {
	Connector string       `json:"connector"`
	Auth      string       `json:"auth,omitempty"`
	Path      string       `json:"path,omitempty"`
	Query     []QueryParam `json:"query,omitempty"`
	AssetID   string       `json:"assetId,omitempty"`
}
