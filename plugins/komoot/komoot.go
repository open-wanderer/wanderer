//go:build tinygo

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

type komootClient struct {
	userID string
	token  string
}

const komootJSONMaxBytes int64 = 16 * 1024 * 1024

var komootJSONContentTypes = []string{"application/json", "application/hal+json"}

var errTourKindMismatch = errors.New("tour kind mismatch")

func login(email string, password string) (*komootClient, error) {
	endpoint := "https://api.komoot.de/v006/account/email/" + url.PathEscape(email) + "/"
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		URL:    endpoint,
		Headers: map[string]string{
			sdk.AuthHeaderAuthorization: basicAuth(email, password),
			"Accept":                    "application/hal+json",
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: komootJSONContentTypes,
			MaxBytes:     komootJSONMaxBytes,
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
	email := sdk.StringField(auth, "email")
	password := sdk.StringField(auth, "password")
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
			"Accept":                    "application/hal+json",
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: komootJSONContentTypes,
			MaxBytes:     komootJSONMaxBytes,
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

func (c *komootClient) coverImages(id int64) ([]imageItem, error) {
	endpoint := fmt.Sprintf("https://api.komoot.de/v007/tours/%d/cover_images/", id)
	var data coverImages
	err := c.get(endpoint, &data)
	return data.Embedded.Items, err
}

func syncTours(client *komootClient, input listInput, wantKind string) (listOutput, error) {
	page := sdk.IntState(input.State, "page", 0)
	maxItems := sdk.SyncLimit(input)
	rows, totalPages, err := client.tours(page, maxItems)
	if err != nil {
		return listOutput{}, err
	}

	items := make([]trailSummary, 0, maxItems)
	for _, row := range rows {
		if !changedAfter(row.ChangedAt, sdk.StringOption(input.Options, "after")) {
			continue
		}
		if wantKind == "planned" && row.Type != "tour_planned" {
			continue
		}
		if wantKind == "completed" && row.Type != "tour_recorded" {
			continue
		}

		items = append(items, trailSummary{
			Source: trailImportSource{Provider: "komoot", ExternalID: strconv.FormatInt(row.ID, 10)},
			Kind:   kindFromType(row.Type),
		})
		if len(items) >= maxItems {
			break
		}
	}

	nextPage := page + 1
	hasMore := nextPage < totalPages
	return listOutput{
		Items:   items,
		State:   sdk.NextPageState(nextPage, hasMore),
		HasMore: hasMore,
	}, nil
}

func tourDetail(client *komootClient, externalID string, wantKind string) (trailImport, error) {
	id, err := strconv.ParseInt(externalID, 10, 64)
	if err != nil {
		return trailImport{}, fmt.Errorf("invalid tour external id")
	}
	detail, err := client.detailedTour(id)
	if err != nil {
		return trailImport{}, fmt.Errorf("fetch tour %d details: %w", id, err)
	}
	if wantKind == "planned" && detail.Type != "tour_planned" {
		return trailImport{}, fmt.Errorf("%w: tour %d is not planned", errTourKindMismatch, id)
	}
	if wantKind == "completed" && detail.Type != "tour_recorded" {
		return trailImport{}, fmt.Errorf("%w: tour %d is not completed", errTourKindMismatch, id)
	}
	var routeImages []imageItem
	if len(detail.Embedded.CoverImages.Embedded.Items) > 0 {
		routeImages, _ = client.coverImages(detail.ID)
	}
	item, err := tourImport(detail, routeImages)
	if err != nil {
		return trailImport{}, fmt.Errorf("map tour %d: %w", id, err)
	}
	return item, nil
}

func basicAuth(username string, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}
