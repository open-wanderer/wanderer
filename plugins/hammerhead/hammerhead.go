//go:build tinygo

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/extism/go-pdk"
	"github.com/open-wanderer/wanderer/plugins/sdk"
)

type hammerheadClient struct {
	userID string
	token  string
}

func login(email string, password string) (string, error) {
	spec := sdk.HostRequestSpec{
		Method: "POST",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      "/v1/auth/token",
		},
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Body: &sdk.HostRequestBody{
			Type: sdk.HostRequestBodyTypeJSON,
			JSON: map[string]string{
				"grant_type": "password",
				"username":   email,
				"password":   password,
			},
		},
		Expect: sdk.ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1048576,
		},
	}
	response, body, err := sdk.HostRequest(spec)
	if err != nil {
		return "", err
	}
	if response.Status != 200 {
		return "", fmt.Errorf("hammerhead login failed (%d): %s", response.Status, string(body))
	}

	var parsed loginResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if parsed.Token == "" {
		return "", fmt.Errorf("hammerhead login returned no access token")
	}

	pdk.SetVar("hammerhead_access_token", []byte(parsed.Token))
	return parsed.Token, nil
}

func loginClient(auth map[string]any) (hammerheadClient, error) {
	email := sdk.StringField(auth, "email")
	password := sdk.StringField(auth, "password")
	if email == "" || password == "" {
		return hammerheadClient{}, fmt.Errorf("email and password are required")
	}
	token, err := login(email, password)
	if err != nil {
		return hammerheadClient{}, err
	}
	userID, err := userIDFromJWT(token)
	if err != nil {
		return hammerheadClient{}, err
	}
	return hammerheadClient{userID: userID, token: token}, nil
}

func (c hammerheadClient) get(path string, query []sdk.QueryParam, out any) error {
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      "/v1/users/" + c.userID + path,
			Query:     query,
		},
		Headers: map[string]string{
			sdk.AuthHeaderAuthorization: sdk.AuthSchemeBearer + " " + c.token,
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
		return fmt.Errorf("hammerhead request failed (%d): %s", response.Status, string(body))
	}
	return json.Unmarshal(body, out)
}

func (c hammerheadClient) activities(page int, perPage int) ([]activityResponse, int, error) {
	var data activitiesResponse
	err := c.get("/activities", hammerheadListQuery(page, perPage), &data)
	return data.Data, data.TotalPages, err
}

func (c hammerheadClient) tours(page int, perPage int) ([]tourResponse, int, error) {
	var data toursResponse
	err := c.get("/routes", hammerheadListQuery(page, perPage), &data)
	return data.Data, data.TotalPages, err
}

func (c hammerheadClient) activity(id string) (*activity, error) {
	var data activity
	err := c.get("/activities/"+id+"/details", nil, &data)
	return &data, err
}

func (c hammerheadClient) tour(id string) (*tour, error) {
	var data tour
	err := c.get("/routes/"+id, nil, &data)
	return &data, err
}

func hammerheadListQuery(page int, perPage int) []sdk.QueryParam {
	return []sdk.QueryParam{
		{Name: "page", Value: strconv.Itoa(page)},
		{Name: "perPage", Value: strconv.Itoa(perPage)},
		{Name: "orderBy", Value: "NEWEST"},
		{Name: "ascending", Value: "true"},
	}
}

func userIDForUpload(auth map[string]any) (string, error) {
	token := string(pdk.GetVar("hammerhead_access_token"))
	if token == "" {
		email := sdk.StringField(auth, "email")
		password := sdk.StringField(auth, "password")
		if email == "" || password == "" {
			return "", fmt.Errorf("email and password are required")
		}
		var err error
		token, err = login(email, password)
		if err != nil {
			return "", err
		}
	}
	return userIDFromJWT(token)
}

func userIDFromSession() (string, error) {
	token := string(pdk.GetVar("hammerhead_access_token"))
	if token == "" {
		return "", fmt.Errorf("session token is not available")
	}
	return userIDFromJWT(token)
}
