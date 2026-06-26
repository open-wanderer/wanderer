package routes

import (
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// buildNodeInfoDiscovery returns the JRD discovery document for /.well-known/nodeinfo.
// The links array contains one entry pointing to the NodeInfo 2.1 schema endpoint.
func buildNodeInfoDiscovery(origin string) map[string]any {
	return map[string]any{
		"links": []map[string]string{
			{
				"rel":  "http://nodeinfo.diaspora.software/ns/schema/2.1",
				"href": origin + "/.well-known/nodeinfo/2.1",
			},
		},
	}
}

// buildNodeInfo21 returns the NodeInfo 2.1 payload with live user and post counts.
// It queries the trails collection for public-only trails (D-02) and the users
// collection for total user count (D-03). The software version is sourced from
// the WANDERER_VERSION environment variable, falling back to "dev" when unset (D-01).
func buildNodeInfo21(app core.App) (map[string]any, error) {
	// D-01: version from env, fall back to "dev"
	version := os.Getenv("WANDERER_VERSION")
	if version == "" {
		version = "dev"
	}

	// D-02: localPosts = count of public trails only (private trails never leave instance)
	localPosts, err := app.CountRecords("trails", dbx.NewExp("public = true"))
	if err != nil {
		return nil, err
	}

	// D-03: users.total = count of all users (instance actor is in activitypub_actors, not users)
	usersTotal, err := app.CountRecords("users", nil)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"version": "2.1",
		"software": map[string]any{
			"name":       "wanderer",
			"version":    version,
			"homepage":   "https://wanderer.to",
			"repository": "https://github.com/Flomp/wanderer",
		},
		"protocols": []string{"activitypub"},
		"services": map[string]any{
			"inbound":  []string{},
			"outbound": []string{},
		},
		"openRegistrations": false,
		"usage": map[string]any{
			"users": map[string]any{
				"total": usersTotal,
			},
			"localPosts": localPosts,
		},
		"metadata": map[string]any{},
	}

	return payload, nil
}
