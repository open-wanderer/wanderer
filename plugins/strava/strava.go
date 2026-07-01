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

const stravaDownloadMaxBytes int64 = 16 * 1024 * 1024

// Strava is migrating its API host: the new host "https://www.api-v3.strava.com"
// is available from 2027-01-04 and the old one is retired on 2027-06-01 (June
// 2026 Developer Program update). We cut over on 2027-03-01 — after the new host
// has had time to stabilize, well before the old one disappears — so no manual
// change or release is needed at the deadline.
func stravaConnector() string {
	return pickStravaConnector(time.Now())
}

func pickStravaConnector(now time.Time) string {
	cutover := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	if now.Before(cutover) {
		return "api"
	}
	return "api_next"
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
	var routes []route
	err := c.getJSON("/athlete/routes", []sdk.QueryParam{
		{Name: "page", Value: strconv.Itoa(page)},
		{Name: "per_page", Value: strconv.Itoa(perPage)},
	}, &routes)
	return routes, err
}

func (c *stravaClient) route(id string) (*route, error) {
	var route route
	err := c.getJSON("/routes/"+url.PathEscape(id), nil, &route)
	return &route, err
}

func (c *stravaClient) routeGPX(id string) ([]byte, error) {
	return c.getBytes("/routes/" + url.PathEscape(id) + "/export_gpx")
}

func (c *stravaClient) activities(page int, perPage int, after int64) ([]activity, error) {
	var activities []activity
	err := c.getJSON("/athlete/activities", []sdk.QueryParam{
		{Name: "page", Value: strconv.Itoa(page)},
		{Name: "per_page", Value: strconv.Itoa(perPage)},
		{Name: "after", Value: strconv.FormatInt(after, 10)},
	}, &activities)
	return activities, err
}

func (c *stravaClient) activity(id int64) (*detailedActivity, error) {
	var activity detailedActivity
	err := c.getJSON(fmt.Sprintf("/activities/%d", id), nil, &activity)
	return &activity, err
}

func (c *stravaClient) activityStreams(id int64) (*activityStreamResponse, error) {
	var streams activityStreamResponse
	err := c.getJSON(fmt.Sprintf("/activities/%d/streams", id), []sdk.QueryParam{
		{Name: "keys", Value: "latlng,time,altitude"},
		{Name: "key_by_type", Value: "true"},
	}, &streams)
	return &streams, err
}

func (c *stravaClient) activityPhotos(id int64) ([]activityPhoto, error) {
	var photos []activityPhoto
	err := c.getJSON(fmt.Sprintf("/activities/%d/photos", id), []sdk.QueryParam{{Name: "size", Value: "600"}}, &photos)
	return photos, err
}

func (c *stravaClient) getJSON(path string, query []sdk.QueryParam, out any) error {
	response, body, err := c.request(path, query, []string{"application/json"})
	if err != nil {
		return err
	}
	if response.Status < 200 || response.Status >= 300 {
		return fmt.Errorf("strava request failed (%d): %s", response.Status, string(body))
	}
	return json.Unmarshal(body, out)
}

func (c *stravaClient) getBytes(path string) ([]byte, error) {
	response, body, err := c.request(path, nil, []string{"application/gpx+xml", "application/octet-stream", "text/xml", "application/xml"})
	if err != nil {
		return nil, err
	}
	if response.Status < 200 || response.Status >= 300 {
		return nil, fmt.Errorf("strava request failed (%d): %s", response.Status, string(body))
	}
	return body, nil
}

func (c *stravaClient) request(path string, query []sdk.QueryParam, contentTypes []string) (sdk.HostResponse, []byte, error) {
	accept := "application/json"
	if len(contentTypes) > 0 {
		accept = contentTypes[0]
	}
	return sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: stravaConnector(),
			Path:      path,
			Query:     query,
		},
		Headers: map[string]string{
			sdk.AuthHeaderAuthorization: sdk.AuthSchemeBearer + " " + c.accessToken,
			"Accept":                    accept,
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: contentTypes,
			MaxBytes:     stravaDownloadMaxBytes,
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
