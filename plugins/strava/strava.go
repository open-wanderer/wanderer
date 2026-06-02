//go:build tinygo

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

// Strava is migrating its API host: the new host "https://www.api-v3.strava.com"
// is available from 2027-01-04 and the old one is retired on 2027-06-01 (June
// 2026 Developer Program update). We cut over on 2027-03-01 — after the new host
// has had time to stabilize, well before the old one disappears — so no manual
// change or release is needed at the deadline.
func stravaAPIBase() string {
	return pickStravaAPIBase(time.Now())
}

func pickStravaAPIBase(now time.Time) string {
	cutover := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	if now.Before(cutover) {
		return "https://www.strava.com/api/v3"
	}
	return "https://www.api-v3.strava.com"
}

type stravaClient struct {
	accessToken string
}

func newClient(auth map[string]any) (*stravaClient, error) {
	token := sdk.StringField(auth, "accessToken")
	if token == "" {
		return nil, fmt.Errorf("accessToken is required")
	}
	return &stravaClient{accessToken: token}, nil
}

func (c *stravaClient) routes(page int, perPage int) ([]route, error) {
	endpoint := fmt.Sprintf("%s/athlete/routes?page=%d&per_page=%d", stravaAPIBase(), page, perPage)
	var routes []route
	err := c.getJSON(endpoint, &routes)
	return routes, err
}

func (c *stravaClient) route(id string) (*route, error) {
	endpoint := fmt.Sprintf("%s/routes/%s", stravaAPIBase(), url.PathEscape(id))
	var route route
	err := c.getJSON(endpoint, &route)
	return &route, err
}

func (c *stravaClient) routeGPX(id string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/routes/%s/export_gpx", stravaAPIBase(), url.PathEscape(id))
	return c.getBytes(endpoint)
}

func (c *stravaClient) activities(page int, perPage int, after int64) ([]activity, error) {
	endpoint := fmt.Sprintf("%s/athlete/activities?page=%d&per_page=%d&after=%d", stravaAPIBase(), page, perPage, after)
	var activities []activity
	err := c.getJSON(endpoint, &activities)
	return activities, err
}

func (c *stravaClient) activity(id int64) (*detailedActivity, error) {
	endpoint := fmt.Sprintf("%s/activities/%d", stravaAPIBase(), id)
	var activity detailedActivity
	err := c.getJSON(endpoint, &activity)
	return &activity, err
}

func (c *stravaClient) activityStreams(id int64) (*activityStreamResponse, error) {
	endpoint := fmt.Sprintf("%s/activities/%d/streams?keys=latlng,time,altitude&key_by_type=true", stravaAPIBase(), id)
	var streams activityStreamResponse
	err := c.getJSON(endpoint, &streams)
	return &streams, err
}

func (c *stravaClient) activityPhotos(id int64) ([]activityPhoto, error) {
	endpoint := fmt.Sprintf("%s/activities/%d/photos?size=600", stravaAPIBase(), id)
	var photos []activityPhoto
	err := c.getJSON(endpoint, &photos)
	return photos, err
}

func (c *stravaClient) getJSON(endpoint string, out any) error {
	response, body, err := c.request(endpoint, []string{"application/json"})
	if err != nil {
		return err
	}
	if response.Status < 200 || response.Status >= 300 {
		return fmt.Errorf("strava request failed (%d): %s", response.Status, string(body))
	}
	return json.Unmarshal(body, out)
}

func (c *stravaClient) getBytes(endpoint string) ([]byte, error) {
	response, body, err := c.request(endpoint, []string{"application/gpx+xml", "application/octet-stream", "text/xml", "application/xml"})
	if err != nil {
		return nil, err
	}
	if response.Status < 200 || response.Status >= 300 {
		return nil, fmt.Errorf("strava request failed (%d): %s", response.Status, string(body))
	}
	return body, nil
}

func (c *stravaClient) request(endpoint string, contentTypes []string) (sdk.HostResponse, []byte, error) {
	accept := "application/json"
	if len(contentTypes) > 0 {
		accept = contentTypes[0]
	}
	return sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		URL:    endpoint,
		Headers: map[string]string{
			sdk.AuthHeaderAuthorization: sdk.AuthSchemeBearer + " " + c.accessToken,
			"Accept":                    accept,
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: contentTypes,
			MaxBytes:     1048576,
		},
	})
}

func syncRoutes(client *stravaClient, input listInput) (listOutput, error) {
	page := sdk.IntState(input.State, "page", 1)
	if page <= 0 {
		page = 1
	}
	rows, err := client.routes(page, sdk.SyncLimit(input))
	if err != nil {
		return listOutput{}, err
	}
	after := dateOption(input.Options, "after")
	items := make([]trailSummary, 0, sdk.SyncLimit(input))
	for _, row := range rows {
		if !timeAfterDate(row.CreatedAt, after) {
			continue
		}
		items = append(items, trailSummary{
			Source: trailImportSource{Provider: "strava", ExternalID: row.IDStr},
			Kind:   "planned",
		})
		if len(items) >= sdk.SyncLimit(input) {
			break
		}
	}
	nextPage := page + 1
	hasMore := len(rows) >= sdk.SyncLimit(input)
	return listOutput{
		Items:   items,
		State:   sdk.NextPageState(nextPage, hasMore),
		HasMore: hasMore,
	}, nil
}

func syncActivities(client *stravaClient, input listInput) (listOutput, error) {
	page := sdk.IntState(input.State, "page", 1)
	if page <= 0 {
		page = 1
	}
	rows, err := client.activities(page, sdk.SyncLimit(input), unixAfter(input.Options))
	if err != nil {
		return listOutput{}, err
	}
	items := make([]trailSummary, 0, sdk.SyncLimit(input))
	for _, row := range rows {
		externalID := strconv.FormatInt(row.ID, 10)
		items = append(items, trailSummary{
			Source: trailImportSource{Provider: "strava", ExternalID: externalID},
			Kind:   "completed",
		})
		if len(items) >= sdk.SyncLimit(input) {
			break
		}
	}
	nextPage := page + 1
	hasMore := len(rows) >= sdk.SyncLimit(input)
	return listOutput{
		Items:   items,
		State:   sdk.NextPageState(nextPage, hasMore),
		HasMore: hasMore,
	}, nil
}
