//go:build tinygo

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

type komootClient struct {
	userID string
	token  string
}

func login(email string, password string) (*komootClient, error) {
	endpoint := "https://api.komoot.de/v006/account/email/" + url.PathEscape(email) + "/"
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		URL:    endpoint,
		Headers: map[string]string{
			sdk.AuthHeaderAuthorization: basicAuth(email, password),
			"Accept":                    "application/json",
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1048576,
		},
	})
	if err != nil {
		return nil, err
	}
	if response.Status != 200 {
		return nil, fmt.Errorf("komoot login failed (%d): %s", response.Status, string(body))
	}

	var parsed loginResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if parsed.Username == "" || parsed.Password == "" {
		return nil, fmt.Errorf("komoot login response did not contain credentials")
	}
	return &komootClient{userID: parsed.Username, token: parsed.Password}, nil
}

func loginClient(auth map[string]any) (*komootClient, error) {
	email := stringField(auth, "email")
	password := stringField(auth, "password")
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password are required")
	}
	return login(email, password)
}

func (c *komootClient) get(endpoint string, out any) error {
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		URL:    endpoint,
		Headers: map[string]string{
			sdk.AuthHeaderAuthorization: basicAuth(c.userID, c.token),
			"Accept":                    "application/json",
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1048576,
		},
	})
	if err != nil {
		return err
	}
	if response.Status != 200 {
		return fmt.Errorf("komoot request failed (%d): %s", response.Status, string(body))
	}
	return json.Unmarshal(body, out)
}

func (c *komootClient) tours(page int, limit int) ([]tour, int, error) {
	endpoint := fmt.Sprintf("https://api.komoot.de/v007/users/%s/tours/?page=%d&sort_field=date&sort_direction=desc&limit=%d", url.PathEscape(c.userID), page, limit)
	var data toursResponse
	err := c.get(endpoint, &data)
	return data.Embedded.Tours, data.Page.TotalPages, err
}

func (c *komootClient) detailedTour(id int64) (*detailedTour, error) {
	endpoint := fmt.Sprintf("https://api.komoot.de/v007/tours/%d?_embedded=coordinates,way_types,surfaces,directions,participants,timeline,cover_images&directions=v2&fields=timeline&format=coordinate_array&timeline_highlights_fields=tips,recommenders&page=2", id)
	var data detailedTour
	err := c.get(endpoint, &data)
	return &data, err
}

func syncTours(client *komootClient, input listInput, wantKind string) (listOutput, error) {
	page := intState(input.State, "page", 0)
	known := knownIDs(input.RecentExternalIDs)
	maxItems := limit(input)
	rows, totalPages, err := client.tours(page, maxItems)
	if err != nil {
		return listOutput{}, err
	}

	items := make([]trailImport, 0, maxItems)
	for _, row := range rows {
		externalID := strconv.FormatInt(row.ID, 10)
		if known[externalID] {
			continue
		}
		if !changedAfter(row.ChangedAt, stringOption(input.Options, "after")) {
			continue
		}
		if wantKind == "planned" && row.Type != "tour_planned" {
			continue
		}
		if wantKind == "completed" && row.Type != "tour_recorded" {
			continue
		}

		detail, err := client.detailedTour(row.ID)
		if err != nil {
			continue
		}
		item, err := tourImport(detail)
		if err == nil {
			items = append(items, item)
		}
		if len(items) >= maxItems {
			break
		}
	}

	nextPage := page + 1
	return listOutput{
		Items:   items,
		State:   map[string]any{"page": nextPage},
		HasMore: nextPage < totalPages,
	}, nil
}

func basicAuth(username string, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}
