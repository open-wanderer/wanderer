package main

import (
	"encoding/json"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

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

type pluginError struct {
	Code              string `json:"code"`
	Message           string `json:"message,omitempty"`
	RetryAfterSeconds *int   `json:"retryAfterSeconds,omitempty"`
}
type refreshSessionInput = sdk.RefreshSessionInput
type refreshSessionOutput = sdk.RefreshSessionOutput

type intervalsActivity struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	Type               string   `json:"type"`
	StartDateLocal     string   `json:"start_date_local"`
	StartDate          string   `json:"start_date"` // UTC
	Distance           float64  `json:"distance"`
	MovingTime         int      `json:"moving_time"`
	ElapsedTime        int      `json:"elapsed_time"`
	AverageWatts       float64  `json:"average_watts"`
	TotalElevationGain float64  `json:"total_elevation_gain"`
	StartLatitude      *float64 `json:"start_latitude"`
	StartLongitude     *float64 `json:"start_longitude"`
}

type intervalStream struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
