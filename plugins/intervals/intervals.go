//go:build tinygo

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

type rateLimitError struct {
	retryAfter int
}

func (e rateLimitError) Error() string {
	return "intervals.icu rate limit exceeded"
}

func parseRetryAfter(response sdk.HostResponse) int {
	for _, key := range []string{"Retry-After", "retry-after"} {
		if values, ok := response.HeaderValues[key]; ok && len(values) > 0 {
			if seconds, err := strconv.Atoi(values[0]); err == nil && seconds > 0 {
				return seconds
			}
		}
	}
	return 0
}

const intervalsDownloadMaxBytes int64 = 16 * 1024 * 1024

type intervalsClient struct {
	athleteID string
	apiKey    string
}

func newClient(auth map[string]any) (*intervalsClient, error) {
	athleteID := sdk.StringField(auth, "athleteId")
	if athleteID == "" {
		athleteID = "0"
	}
	apiKey := sdk.StringField(auth, "apiKey")
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}
	return &intervalsClient{
		athleteID: athleteID,
		apiKey:    apiKey,
	}, nil
}

func (c *intervalsClient) activities(oldest string) ([]intervalsActivity, error) {
	var activities []intervalsActivity
	err := c.getJSON("/athlete/"+c.athleteID+"/activities", []sdk.QueryParam{
		{Name: "oldest", Value: oldest},
	}, &activities)
	return activities, err
}

func (c *intervalsClient) activity(id string) (*intervalsActivity, error) {
	var activity intervalsActivity
	err := c.getJSON("/activity/"+id, nil, &activity)
	return &activity, err
}

func (c *intervalsClient) activityGPX(id string) ([]byte, error) {
	return c.getBytes("/activity/" + id + "/gpx-file")
}

func (c *intervalsClient) getJSON(path string, query []sdk.QueryParam, out any) error {
	response, body, err := c.request(path, query, []string{"application/json"})
	if err != nil {
		return err
	}
	if response.Status == 429 {
		return rateLimitError{retryAfter: parseRetryAfter(response)}
	}
	if response.Status < 200 || response.Status >= 300 {
		return fmt.Errorf("intervals.icu request failed (%d): %s", response.Status, string(body))
	}
	return json.Unmarshal(body, out)
}

func (c *intervalsClient) getBytes(path string) ([]byte, error) {
	response, body, err := c.request(path, nil, []string{"application/gpx+xml", "application/octet-stream", "text/xml", "application/xml"})
	if err != nil {
		return nil, err
	}
	if response.Status == 429 {
		return nil, rateLimitError{retryAfter: parseRetryAfter(response)}
	}
	if response.Status < 200 || response.Status >= 300 {
		return nil, fmt.Errorf("intervals.icu request failed (%d): %s", response.Status, string(body))
	}
	return body, nil
}

func (c *intervalsClient) request(path string, query []sdk.QueryParam, contentTypes []string) (sdk.HostResponse, []byte, error) {
	accept := "application/json"
	if len(contentTypes) > 0 {
		accept = contentTypes[0]
	}

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte("API_KEY:"+c.apiKey))

	return sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      path,
			Query:     query,
		},
		Headers: map[string]string{
			sdk.AuthHeaderAuthorization: authHeader,
			"Accept":                    accept,
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: contentTypes,
			MaxBytes:     intervalsDownloadMaxBytes,
		},
	})
}

func syncActivities(client *intervalsClient, input listInput) (listOutput, error) {
	oldest := getOldestDate(input)

	rows, err := client.activities(oldest)
	if err != nil {
		return listOutput{}, err
	}

	sortActivitiesAscending(rows)

	limit := sdk.SyncLimit(input)
	items := make([]trailSummary, 0, limit)
	var nextOldest string
	hasMore := false

	for i, row := range rows {
		if !timeAfterDate(row.StartDateLocal, oldest) {
			continue
		}

		if len(items) >= limit {
			nextOldest = parseDateOnly(rows[i].StartDateLocal)
			hasMore = true
			break
		}

		items = append(items, trailSummary{
			Source: trailImportSource{Provider: "intervals", ExternalID: row.ID},
			Kind:   "completed",
		})
	}

	state := map[string]any{}
	if hasMore && nextOldest != "" {
		state["oldest"] = nextOldest
	}

	return listOutput{
		Items:   items,
		State:   state,
		HasMore: hasMore,
	}, nil
}

func parseDateOnly(isoDate string) string {
	if len(isoDate) >= 10 {
		return isoDate[:10]
	}
	return isoDate
}

func sortActivitiesAscending(activities []intervalsActivity) {
	for i := 1; i < len(activities); i++ {
		key := activities[i]
		j := i - 1
		for j >= 0 && compareDates(activities[j].StartDateLocal, key.StartDateLocal) > 0 {
			activities[j+1] = activities[j]
			j = j - 1
		}
		activities[j+1] = key
	}
}

func compareDates(d1, d2 string) int {
	if d1 < d2 {
		return -1
	}
	if d1 > d2 {
		return 1
	}
	return 0
}
