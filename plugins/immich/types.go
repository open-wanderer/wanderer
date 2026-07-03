package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type pluginError = sdk.PluginError

type assetLibraryInput struct {
	Instance sdk.InstanceRef     `json:"instance"`
	Auth     map[string]any      `json:"auth,omitempty"`
	Config   map[string]any      `json:"config,omitempty"`
	Limits   sdk.SyncLimits      `json:"limits,omitempty"`
	Request  assetLibraryRequest `json:"request"`
}

type assetLibraryRequest struct {
	Action       string       `json:"action"`
	Lat          float64      `json:"lat,omitempty"`
	Lon          float64      `json:"lon,omitempty"`
	Points       []trackPoint `json:"points,omitempty"`
	StartedAt    string       `json:"startedAt,omitempty"`
	EndedAt      string       `json:"endedAt,omitempty"`
	TakenAfter   string       `json:"takenAfter,omitempty"`
	TakenBefore  string       `json:"takenBefore,omitempty"`
	DoubleRadius bool         `json:"doubleRadius,omitempty"`
	AssetIDs     []string     `json:"assetIds,omitempty"`
}

type assetLibraryOutput struct {
	OK            bool             `json:"ok"`
	UserID        string           `json:"userId,omitempty"`
	Candidates    []assetCandidate `json:"candidates,omitempty"`
	Photos        []sdk.Photo      `json:"photos,omitempty"`
	HasMore       bool             `json:"hasMore,omitempty"`
	TakenAfter    string           `json:"takenAfter,omitempty"`
	HasTimestamps bool             `json:"hasTimestamps,omitempty"`
	Error         *pluginError     `json:"error,omitempty"`
}

type trackPoint struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Distance  float64 `json:"distance,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
}

type assetCandidate struct {
	AssetID           string  `json:"assetId"`
	OriginalFileName  string  `json:"originalFileName"`
	TakenAt           string  `json:"takenAt"`
	Lat               float64 `json:"lat"`
	Lon               float64 `json:"lon"`
	Distance          float64 `json:"distance"`
	PointLat          float64 `json:"pointLat"`
	PointLon          float64 `json:"pointLon"`
	DistanceFromStart float64 `json:"distanceFromStart"`
	City              string  `json:"city,omitempty"`
	Country           string  `json:"country,omitempty"`
}

type metadataSearchRequest struct {
	TakenAfter  string   `json:"takenAfter,omitempty"`
	TakenBefore string   `json:"takenBefore,omitempty"`
	Type        string   `json:"type,omitempty"`
	WithExif    bool     `json:"withExif,omitempty"`
	IsArchived  bool     `json:"isArchived,omitempty"`
	IsFavorite  *bool    `json:"isFavorite,omitempty"`
	Page        int      `json:"page,omitempty"`
	Size        int      `json:"size,omitempty"`
	IDs         []string `json:"ids,omitempty"`
}

type metadataSearchResponse struct {
	Assets struct {
		Items    []immichAsset `json:"items"`
		NextPage *string       `json:"nextPage"`
	} `json:"assets"`
}

type immichAsset struct {
	ID               string         `json:"id"`
	OwnerID          string         `json:"ownerId"`
	FileCreatedAt    string         `json:"fileCreatedAt"`
	OriginalFileName string         `json:"originalFileName"`
	ExifInfo         immichExifInfo `json:"exifInfo"`
	Thumbhash        string         `json:"thumbhash,omitempty"`
	Type             string         `json:"type,omitempty"`
	OriginalPath     string         `json:"originalPath,omitempty"`
}

type immichExifInfo struct {
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	City        string   `json:"city"`
	Country     string   `json:"country"`
	Description string   `json:"description"`
}

type currentUserResponse struct {
	ID string `json:"id"`
}
