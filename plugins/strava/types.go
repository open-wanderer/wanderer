package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type instanceRef = sdk.InstanceRef
type listInput = sdk.ListInput
type listOutput = sdk.ListOutput
type detailInput = sdk.DetailInput
type detailOutput = sdk.DetailOutput
type trailSummary = sdk.TrailSummary
type trailImport = sdk.TrailImport
type trailImportSource = sdk.TrailImportSource
type track = sdk.Track
type waypoint = sdk.Waypoint
type photo = sdk.Photo
type mediaSource = sdk.MediaSource

type pluginError = sdk.PluginError

type route struct {
	Description         string          `json:"description"`
	Distance            float64         `json:"distance"`
	ElevationGain       float64         `json:"elevation_gain"`
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
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Distance           float64   `json:"distance"`
	ElapsedTime        int       `json:"elapsed_time"`
	TotalElevationGain float64   `json:"total_elevation_gain"`
	Private            bool      `json:"private"`
	StartDate          string    `json:"start_date"`
	StartLatlng        []float64 `json:"start_latlng"`
	Type               string    `json:"type"`
	SportType          string    `json:"sport_type"`
	Photos             photos    `json:"photos"`
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
