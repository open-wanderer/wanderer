package main

import (
	"encoding/base64"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

func (c *komootClient) requestHeaders(connector string) map[string]string {
	headers := map[string]string{
		"Accept": "application/hal+json",
	}
	if connector == "api" {
		headers[sdk.AuthHeaderAuthorization] = basicAuth(c.userID, c.token)
	}
	if language := acceptLanguage(c.locale); language != "" {
		headers["Accept-Language"] = language
	}
	return headers
}

func basicAuth(username string, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}
