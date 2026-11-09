package util

import "github.com/pocketbase/pocketbase/core"

func validatePhotoAssetInputTargets(app core.App, author string, input PhotoAssetInput) error {
	if input.ExternalProvider == FederationAssetProvider {
		return validateFederatedAssetTargets(app, author, PhotoAssetLinkTargets(input.Trail, input.Waypoint, input.SummitLog))
	}
	return nil
}
