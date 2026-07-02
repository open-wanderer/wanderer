package main

func activityTypeFromProvider(value string) string {
	switch value {
	case "AlpineSki", "BackcountrySki", "IceSkate", "NordicSki", "RollerSki", "Snowboard":
		return "skiing"
	case "Canoeing", "Kayaking", "Kitesurf", "Rowing", "Sail", "StandUpPaddling", "Surfing", "Windsurf":
		return "canoeing"
	case "Hike", "Snowshoe":
		return "hiking"
	case "Run", "TrailRun", "VirtualRun":
		return "running"
	case "Walk", "Golf", "Skateboard", "Wheelchair":
		return "walking"
	case "Ride", "EBikeRide", "Handcycle", "InlineSkate", "Velomobile", "VirtualRide":
		return "biking"
	case "RockClimbing":
		return "climbing"
	default:
		return value
	}
}
