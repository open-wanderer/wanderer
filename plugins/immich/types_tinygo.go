//go:build tinygo

package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type pluginError = sdk.PluginError

type assetLibraryInput struct {
	Instance sdk.InstanceRef       `json:"instance"`
	Auth     map[string]any        `json:"auth,omitempty"`
	Config   map[string]any        `json:"config,omitempty"`
	Limits   sdk.PhotoImportLimits `json:"limits,omitempty"`
	Search   sdk.AssetSearchLimits `json:"search,omitempty"`
	State    map[string]any        `json:"state,omitempty"`
	Request  assetLibraryRequest   `json:"request"`
}

type assetLibraryOutput struct {
	UserID        string                `json:"userId,omitempty"`
	Candidates    []assetCandidate      `json:"candidates,omitempty"`
	Photos        []sdk.Photo           `json:"photos,omitempty"`
	OmittedAssets []sdk.OmittedAsset    `json:"omittedAssetIds,omitempty"`
	State         map[string]any        `json:"state,omitempty"`
	HasMore       bool                  `json:"hasMore,omitempty"`
	Stats         *sdk.AssetSearchStats `json:"stats,omitempty"`
	TakenAfter    string                `json:"takenAfter,omitempty"`
	HasTimestamps bool                  `json:"hasTimestamps,omitempty"`
	Error         *pluginError          `json:"error,omitempty"`
}

type metadataSearchRequest struct {
	TakenAfter  string   `json:"takenAfter,omitempty"`
	TakenBefore string   `json:"takenBefore,omitempty"`
	Type        string   `json:"type,omitempty"`
	WithExif    bool     `json:"withExif,omitempty"`
	IsArchived  bool     `json:"isArchived,omitempty"`
	IsFavorite  *bool    `json:"isFavorite,omitempty"`
	Page        int      `json:"page,omitempty"`
	Size        int      `json:"size,omitempty"`
	IDs         []string `json:"ids,omitempty"`
}

type metadataSearchResponse struct {
	Assets struct {
		Items    []immichAsset `json:"items"`
		NextPage *string       `json:"nextPage"`
	} `json:"assets"`
}

type currentUserResponse struct {
	ID string `json:"id"`
}
