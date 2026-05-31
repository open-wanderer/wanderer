//go:build tinygo

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

const stravaAPIBase = "https://www.strava.com/api/v3"

type stravaClient struct {
	accessToken string
}

func newClient(auth map[string]any) (*stravaClient, error) {
	token := stringField(auth, "accessToken")
	if token == "" {
		return nil, fmt.Errorf("accessToken is required")
	}
	return &stravaClient{accessToken: token}, nil
}

func (c *stravaClient) routes(page int, perPage int) ([]route, error) {
	endpoint := fmt.Sprintf("%s/athlete/routes?page=%d&per_page=%d", stravaAPIBase, page, perPage)
	var routes []route
	err := c.getJSON(endpoint, &routes)
	return routes, err
}

func (c *stravaClient) routeGPX(id string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/routes/%s/export_gpx", stravaAPIBase, url.PathEscape(id))
	return c.getBytes(endpoint)
}

func (c *stravaClient) activities(page int, perPage int, after int64) ([]activity, error) {
	endpoint := fmt.Sprintf("%s/athlete/activities?page=%d&per_page=%d&after=%d", stravaAPIBase, page, perPage, after)
	var activities []activity
	err := c.getJSON(endpoint, &activities)
	return activities, err
}

func (c *stravaClient) activity(id int64) (*detailedActivity, error) {
	endpoint := fmt.Sprintf("%s/activities/%d", stravaAPIBase, id)
	var activity detailedActivity
	err := c.getJSON(endpoint, &activity)
	return &activity, err
}

func (c *stravaClient) activityStreams(id int64) (*activityStreamResponse, error) {
	endpoint := fmt.Sprintf("%s/activities/%d/streams?keys=latlng,time,altitude&key_by_type=true", stravaAPIBase, id)
	var streams activityStreamResponse
	err := c.getJSON(endpoint, &streams)
	return &streams, err
}

func (c *stravaClient) activityPhotos(id int64) ([]activityPhoto, error) {
	endpoint := fmt.Sprintf("%s/activities/%d/photos?size=600", stravaAPIBase, id)
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
	page := intState(input.State, "page", 1)
	if page <= 0 {
		page = 1
	}
	rows, err := client.routes(page, limit(input))
	if err != nil {
		return listOutput{}, err
	}
	known := knownIDs(input.RecentExternalIDs)
	after := dateOption(input.Options, "after")
	items := make([]trailImport, 0, limit(input))
	for _, row := range rows {
		if known[row.IDStr] {
			continue
		}
		if !timeAfterDate(row.CreatedAt, after) {
			continue
		}
		gpxData, err := client.routeGPX(row.IDStr)
		if err != nil {
			continue
		}
		item, err := routeImport(row, gpxData)
		if err == nil {
			items = append(items, item)
		}
		if len(items) >= limit(input) {
			break
		}
	}
	nextPage := page + 1
	return listOutput{
		Items:   items,
		State:   map[string]any{"page": nextPage},
		HasMore: len(rows) >= limit(input),
	}, nil
}

func syncActivities(client *stravaClient, input listInput) (listOutput, error) {
	page := intState(input.State, "page", 1)
	if page <= 0 {
		page = 1
	}
	rows, err := client.activities(page, limit(input), unixAfter(input.Options))
	if err != nil {
		return listOutput{}, err
	}
	known := knownIDs(input.RecentExternalIDs)
	items := make([]trailImport, 0, limit(input))
	for _, row := range rows {
		externalID := strconv.FormatInt(row.ID, 10)
		if known[externalID] {
			continue
		}
		detail, err := client.activity(row.ID)
		if err != nil {
			continue
		}
		var photos []activityPhoto
		if detail.Photos.Count > 0 {
			photos, _ = client.activityPhotos(row.ID)
		}
		streams, err := client.activityStreams(row.ID)
		if err != nil {
			continue
		}
		item, err := activityImport(detail, streams, photos)
		if err == nil {
			items = append(items, item)
		}
		if len(items) >= limit(input) {
			break
		}
	}
	nextPage := page + 1
	return listOutput{
		Items:   items,
		State:   map[string]any{"page": nextPage},
		HasMore: len(rows) >= limit(input),
	}, nil
}
