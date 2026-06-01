//go:build tinygo

package main

import (
	"encoding/json"
	"fmt"

	"github.com/extism/go-pdk"
	"github.com/open-wanderer/wanderer/plugins/sdk"
)

const tokenURL = "https://dashboard.hammerhead.io/v1/auth/token"

type hammerheadClient struct {
	userID string
	token  string
}

func login(email string, password string) (string, error) {
	spec := sdk.HostRequestSpec{
		Method: "POST",
		URL:    tokenURL,
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

func (c hammerheadClient) get(path string, out any) error {
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		URL:    "https://dashboard.hammerhead.io/v1/users/" + c.userID + path,
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
	err := c.get(fmt.Sprintf("/activities?perPage=%d&page=%d&search=&orderBy=NEWEST&ascending=true", perPage, page), &data)
	return data.Data, data.TotalPages, err
}

func (c hammerheadClient) tours(page int, perPage int) ([]tourResponse, int, error) {
	var data toursResponse
	err := c.get(fmt.Sprintf("/routes?perPage=%d&page=%d&search=&orderBy=NEWEST&ascending=true&exclude=archive", perPage, page), &data)
	return data.Data, data.TotalPages, err
}

func (c hammerheadClient) activity(id string) (*activity, error) {
	var data activity
	err := c.get("/activities/"+id+"/details", &data)
	return &data, err
}

func (c hammerheadClient) tour(id string) (*tour, error) {
	var data tour
	err := c.get("/routes/"+id, &data)
	return &data, err
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
