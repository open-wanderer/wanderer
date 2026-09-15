package main

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
