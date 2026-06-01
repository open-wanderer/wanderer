package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type instanceRef = sdk.InstanceRef
type refreshSessionInput = sdk.RefreshSessionInput
type refreshSessionOutput = sdk.RefreshSessionOutput
type listInput = sdk.ListInput
type listOutput = sdk.ListOutput
type trailImport = sdk.TrailImport
type trailImportSource = sdk.TrailImportSource
type track = sdk.Track
type waypoint = sdk.Waypoint
type photo = sdk.Photo
type mediaSource = sdk.MediaSource

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
	ID            int64                `json:"id"`
	Type          string               `json:"type"`
	Name          string               `json:"name"`
	Status        string               `json:"status"`
	Date          string               `json:"date"`
	Sport         string               `json:"sport"`
	Distance      float64              `json:"distance"`
	Duration      int                  `json:"duration"`
	ElevationUp   float64              `json:"elevation_up"`
	ElevationDown float64              `json:"elevation_down"`
	MapImage      mapImage             `json:"map_image"`
	Difficulty    difficulty           `json:"difficulty"`
	ChangedAt     string               `json:"changed_at"`
	Embedded      detailedTourEmbedded `json:"_embedded"`
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
	Tips   tips        `json:"tips"`
	Images coverImages `json:"images"`
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
