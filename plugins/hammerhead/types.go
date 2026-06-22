package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type instanceRef = sdk.InstanceRef
type refreshSessionInput = sdk.RefreshSessionInput
type refreshSessionOutput = sdk.RefreshSessionOutput
type trailSendInput = sdk.TrailSendInput
type listInput = sdk.ListInput
type listOutput = sdk.ListOutput
type detailInput = sdk.DetailInput
type detailOutput = sdk.DetailOutput
type trailSummary = sdk.TrailSummary
type trailImport = sdk.TrailImport
type trailImportSource = sdk.TrailImportSource
type track = sdk.Track
type trailSendPlan = sdk.TrailSendPlan

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
	Duration     duration    `json:"duration"`
	ActivityInfo []info      `json:"activityInfo"`
	Laps         []lapDetail `json:"laps"`
	ActivityType string      `json:"activityType"`
}

type duration struct {
	ElapsedTime int `json:"elapsedTime"`
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
