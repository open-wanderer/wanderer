package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// NodeInfo handles GET /.well-known/nodeinfo — returns the JRD discovery document.
func NodeInfo(e *core.RequestEvent) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}
	return e.JSON(http.StatusOK, buildNodeInfoDiscovery(origin))
}

// NodeInfo21 handles GET /.well-known/nodeinfo/2.1 — returns the NodeInfo 2.1 payload.
func NodeInfo21(e *core.RequestEvent) error {
	payload, err := buildNodeInfo21(e.App)
	if err != nil {
		return err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.Response.Header().Set("Content-Type", `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`)
	e.Response.WriteHeader(http.StatusOK)
	_, err = e.Response.Write(b)
	return err
}

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
// It queries the trails collection for local public trails and the users collection
// for total user count. The version field is overwritten by the SvelteKit proxy at
// /.well-known/nodeinfo/2.1, which injects the version from web/package.json.
func buildNodeInfo21(app core.App) (map[string]any, error) {
	version := os.Getenv("WANDERER_VERSION")
	if version == "" {
		version = "dev"
	}

	// localPosts = public trails authored by local actors only. Federated trails
	// (is_local=false actors) are excluded so the count reflects what this instance
	// produces, matching fediverse convention.
	var localPosts int64
	if err := app.DB().
		Select("COUNT(*) AS c").
		From("trails t").
		InnerJoin("activitypub_actors aa", dbx.NewExp("t.author = aa.id")).
		Where(dbx.NewExp("t.public = 1 AND aa.is_local = 1")).
		Row(&localPosts); err != nil {
		return nil, fmt.Errorf("counting local trails: %w", err)
	}

	// users.total = count of all users (the instance actor is in activitypub_actors, not users)
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
