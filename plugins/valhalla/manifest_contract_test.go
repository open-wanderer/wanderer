package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestManifestAlternativeLimitMatchesAdapter(t *testing.T) {
	payload, err := os.ReadFile("plugin.json")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		Metadata struct {
			Routing struct {
				MaxAlternatives int `json:"maxAlternatives"`
			} `json:"routing"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}
	if manifest.Metadata.Routing.MaxAlternatives != valhallaMaxCandidates {
		t.Fatalf("manifest maxAlternatives = %d, adapter limit = %d", manifest.Metadata.Routing.MaxAlternatives, valhallaMaxCandidates)
	}
}

func TestManifestDeclaresManeuverContract(t *testing.T) {
	payload, err := os.ReadFile("plugin.json")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		Capabilities []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Export  string `json:"export"`
		} `json:"capabilities"`
		Permissions struct {
			Network struct {
				Connectors []struct {
					AllowedPathPrefixes []string `json:"allowedPathPrefixes"`
				} `json:"connectors"`
			} `json:"network"`
		} `json:"permissions"`
		Metadata struct {
			Routing struct {
				Roles []string `json:"roles"`
			} `json:"routing"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}
	foundCapability := false
	for _, capability := range manifest.Capabilities {
		foundCapability = foundCapability || (capability.Name == "maneuvers" && capability.Version == "v1" && capability.Export == "maneuvers_v1")
	}
	if !foundCapability {
		t.Fatal("manifest does not declare maneuvers.v1")
	}
	foundRole := false
	for _, role := range manifest.Metadata.Routing.Roles {
		foundRole = foundRole || role == "maneuvers"
	}
	if !foundRole {
		t.Fatal("manifest does not declare the maneuver routing role")
	}
	foundPath := false
	for _, connector := range manifest.Permissions.Network.Connectors {
		for _, path := range connector.AllowedPathPrefixes {
			foundPath = foundPath || path == "/trace_route"
		}
	}
	if !foundPath {
		t.Fatal("manifest does not permit /trace_route")
	}
}
