package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	komootMappings := map[string][]string{
		"Hiking": {
			"hike",
			"mountaineering",
		},
		"Biking": {
			"racebike",
			"e_racebike",
			"touringbicycle",
			"e_touringbicycle",
			"mtb",
			"e_mtb",
			"mtb_easy",
			"e_mtb_easy",
			"mtb_advanced",
			"e_mtb_advanced",
			"downhillbike",
			"unicycle",
			"citybike",
		},
		"Walking": {
			"jogging",
			"nordicwalking",
			"skaten",
			"other",
		},
		"Climbing": {
			"climbing",
		},
		"Skiing": {
			"nordic",
			"skialpin",
			"skitour",
			"sled",
			"snowboard",
			"snowshoe",
		},
	}

	stravaMappings := map[string][]string{
		"Hiking": {
			"Hike",
			"Snowshoe",
		},
		"Biking": {
			"EBikeRide",
			"Handcycle",
			"Ride",
			"Velomobile",
			"VirtualRide",
		},
		"Walking": {
			"InlineSkate",
			"Run",
			"Skateboard",
			"VirtualRun",
			"Walk",
			"Wheelchair",
		},
		"Climbing": {
			"RockClimbing",
		},
		"Skiing": {
			"AlpineSki",
			"BackcountrySki",
			"IceSkate",
			"NordicSki",
			"RollerSki",
			"Snowboard",
		},
		"Canoeing": {
			"Canoeing",
			"Kayaking",
			"Kitesurf",
			"Rowing",
			"Sail",
			"StandUpPaddling",
			"Surfing",
			"Windsurf",
		},
		"Other": {
			"Crossfit",
			"Elliptical",
			"Golf",
			"Soccer",
			"StairStepper",
			"Swim",
			"WeightTraining",
			"Workout",
			"Yoga",
		},
	}

	m.Register(func(app core.App) error {
		if err := applyCategoryIntegrationMappings(app, "komoot", komootMappings, true); err != nil {
			return err
		}
		return applyCategoryIntegrationMappings(app, "strava", stravaMappings, true)
	}, func(app core.App) error {
		if err := applyCategoryIntegrationMappings(app, "strava", stravaMappings, false); err != nil {
			return err
		}
		return applyCategoryIntegrationMappings(app, "komoot", komootMappings, false)
	})
}
