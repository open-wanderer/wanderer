package main

import (
	"fmt"
	"math"
)

// deriveAddedTrails builds a deterministic add batch from the benchmark
// corpus. The source trails are sampled across the whole corpus (including its
// rare tail scenarios), receive new IDs, and are moved slightly so an add is
// not merely another primary key for identical geometry.
//
// The returned trails do not share coordinate slices with dataset.Trails.
func deriveAddedTrails(dataset Dataset, count int) []Trail {
	if count <= 0 || len(dataset.Trails) == 0 {
		return nil
	}

	existingIDs := make(map[string]struct{}, len(dataset.Trails)+count)
	for _, trail := range dataset.Trails {
		existingIDs[trail.ID] = struct{}{}
	}

	added := make([]Trail, 0, count)
	for index := 0; index < count; index++ {
		sourceIndex := mutationSourceIndex(len(dataset.Trails), count, index)
		source := dataset.Trails[sourceIndex]
		trail := Trail{
			ID:       uniqueAddedTrailID(existingIDs, dataset.Seed, index),
			Scenario: source.Scenario,
			Parts:    make([][]Coordinate, len(source.Parts)),
		}

		// A small, deterministic displacement keeps the generated route in the
		// same workload region while making spatial-index writes observable.
		distance := 70 + float64(index%11)*19
		bearing := math.Mod(float64(dataset.Seed%360)+float64(index)*137.507764, 360)
		if bearing < 0 {
			bearing += 360
		}
		for partIndex, part := range source.Parts {
			trail.Parts[partIndex] = make([]Coordinate, len(part))
			for coordinateIndex, coordinate := range part {
				trail.Parts[partIndex][coordinateIndex] = destinationPoint(coordinate, distance, bearing)
			}
		}
		added = append(added, trail)
	}
	return added
}

// selectDeletionIDs deterministically samples existing trail IDs across the
// complete slice. It returns at most count distinct, non-empty IDs and never
// modifies trails.
func selectDeletionIDs(trails []Trail, count int) []string {
	if count <= 0 || len(trails) == 0 {
		return nil
	}
	if count > len(trails) {
		count = len(trails)
	}

	ids := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for index := 0; index < count; index++ {
		sourceIndex := mutationSourceIndex(len(trails), count, index)
		id := trails[sourceIndex].ID
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func mutationSourceIndex(sourceCount, mutationCount, mutationIndex int) int {
	if sourceCount <= 1 || mutationCount <= 1 {
		return sourceCount - 1
	}
	if mutationCount <= sourceCount {
		return mutationIndex * (sourceCount - 1) / (mutationCount - 1)
	}
	return mutationIndex % sourceCount
}

func uniqueAddedTrailID(existing map[string]struct{}, seed int64, index int) string {
	base := fmt.Sprintf("trail-added-%d-%06d", seed, index)
	candidate := base
	for suffix := 1; ; suffix++ {
		if _, exists := existing[candidate]; !exists {
			existing[candidate] = struct{}{}
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, suffix)
	}
}
