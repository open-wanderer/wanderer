//go:build tinygo

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

const komootJSONMaxBytes int64 = 16 * 1024 * 1024
const komootMaxHighlightTipRequests = 20

var komootJSONContentTypes = []string{"application/json", "application/hal+json"}

var errTourKindMismatch = errors.New("tour kind mismatch")

func login(email string, password string) (*komootClient, error) {
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      "/v006/account/email/" + url.PathEscape(email) + "/",
		},
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
	client := &komootClient{userID: parsed.Username, token: parsed.Password, locale: parsed.Locale}
	if client.locale == "" {
		client.locale = client.profileLocale()
	}
	return client, nil
}

func loginClient(auth map[string]any) (*komootClient, error) {
	email := sdk.StringField(auth, "email")
	password := sdk.StringField(auth, "password")
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password are required")
	}
	return login(email, password)
}

func (c *komootClient) get(path string, query []sdk.QueryParam, out any) error {
	body, err := c.getRawFromConnector("api", path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (c *komootClient) getFromConnector(connector string, path string, query []sdk.QueryParam, out any) error {
	body, err := c.getRawFromConnector(connector, path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (c *komootClient) getRawFromConnector(connector string, path string, query []sdk.QueryParam) ([]byte, error) {
	headers := c.requestHeaders(connector)
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: connector,
			Path:      path,
			Query:     query,
		},
		Headers: headers,
		Expect: sdk.ResponseExpect{
			ContentTypes: komootJSONContentTypes,
			MaxBytes:     komootJSONMaxBytes,
		},
	})
	if err != nil {
		return nil, err
	}
	if response.Status != 200 {
		return body, fmt.Errorf("komoot request failed (%d): %s", response.Status, string(body))
	}
	return body, nil
}

func (c *komootClient) profileLocale() string {
	var data userProfile
	if err := c.get("/v007/users/"+url.PathEscape(c.userID), nil, &data); err != nil {
		return ""
	}
	return data.Locale
}

func (c *komootClient) tours(page int, limit int) ([]tour, int, error) {
	var data toursResponse
	err := c.get("/v007/users/"+url.PathEscape(c.userID)+"/tours/", []sdk.QueryParam{
		{Name: "page", Value: strconv.Itoa(page)},
		{Name: "sort_field", Value: "date"},
		{Name: "sort_direction", Value: "desc"},
		{Name: "limit", Value: strconv.Itoa(limit)},
	}, &data)
	return data.Embedded.Tours, data.Page.TotalPages, err
}

func (c *komootClient) detailedTour(id int64) (*detailedTour, error) {
	var data detailedTour
	err := c.get(fmt.Sprintf("/v007/tours/%d", id), []sdk.QueryParam{
		{Name: "_embedded", Value: "coordinates,way_types,surfaces,directions,participants,timeline,cover_images"},
		{Name: "directions", Value: "v2"},
		{Name: "fields", Value: "timeline"},
		{Name: "format", Value: "coordinate_array"},
		{Name: "timeline_highlights_fields", Value: "tips,recommenders"},
		{Name: "page", Value: "2"},
	}, &data)
	if err != nil {
		return &data, err
	}
	if len(data.Embedded.WayPoints.Embedded.Items) == 0 && len(data.Embedded.Timeline.Embedded.Items) == 0 {
		if timeline, err := c.webTimeline(id); err == nil {
			data.Embedded.WayPoints = timeline
		}
	}
	return &data, nil
}

func (c *komootClient) webTimeline(id int64) (timeline, error) {
	var data timeline
	token := c.shareToken(id)
	var query []sdk.QueryParam
	if token != "" {
		query = []sdk.QueryParam{{Name: "share_token", Value: token}}
	}
	err := c.getFromConnector("web", fmt.Sprintf("/webapi/v007/tours/%d/timeline/", id), query, &data)
	if err != nil {
		return data, err
	}
	c.addHighlightTips(data.Embedded.Items)
	return data, nil
}

func (c *komootClient) shareToken(id int64) string {
	token, err := c.shareTokenWithQuery(id, nil)
	if err == nil && token != "" {
		return token
	}
	token, _ = c.shareTokenWithQuery(id, []sdk.QueryParam{{Name: "token_name", Value: "invite"}})
	return token
}

func (c *komootClient) shareTokenWithQuery(id int64, query []sdk.QueryParam) (string, error) {
	body, err := c.getRawFromConnector("api", fmt.Sprintf("/v007/tours/%d/share_token", id), query)
	if err != nil {
		return "", err
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return "", err
	}
	if token, ok := value.(string); ok {
		return token, nil
	}
	return findShareToken(value), nil
}

func findShareToken(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"token", "share_token", "shareToken"} {
			if token, ok := typed[key].(string); ok {
				return token
			}
		}
		for _, nested := range typed {
			if token := findShareToken(nested); token != "" {
				return token
			}
		}
	case []any:
		for _, nested := range typed {
			if token := findShareToken(nested); token != "" {
				return token
			}
		}
	}
	return ""
}

func (c *komootClient) addHighlightTips(items []timelineItem) {
	requests := 0
	for i := range items {
		if items[i].Type != "highlight" {
			continue
		}
		ref := &items[i].Embedded.Reference
		if ref.ID.String() == "" || len(ref.Embedded.Tips.Embedded.Items) > 0 {
			continue
		}
		if requests >= komootMaxHighlightTipRequests {
			return
		}
		requests++
		var data tips
		if err := c.get(fmt.Sprintf("/v007/highlights/%s/tips/", url.PathEscape(ref.ID.String())), nil, &data); err == nil {
			ref.Embedded.Tips = data
		}
	}
}

func (c *komootClient) coverImages(id int64) ([]imageItem, error) {
	var data coverImages
	err := c.get(fmt.Sprintf("/v007/tours/%d/cover_images/", id), nil, &data)
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
		if !tourDateAfter(row.Date, sdk.StringOption(input.Options, "after")) {
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
