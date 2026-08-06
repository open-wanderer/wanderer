package main

import (
	"math"
	"sort"
)

type immichConfig struct {
	TimeWindowMinutes int
	MaxDistanceMeters int
	ImportSize        string
	OwnedOnly         bool
	UserID            string
}

func matchAssetCandidates(assets []immichAsset, req assetLibraryRequest, cfg immichConfig) []assetCandidate {
	maxDistance := float64(cfg.MaxDistanceMeters)
	if req.DoubleRadius {
		maxDistance *= 2
	}
	if maxDistance <= 0 {
		maxDistance = 150
	}
	matches := []assetCandidate{}
	for _, asset := range assets {
		if cfg.OwnedOnly && cfg.UserID != "" && asset.OwnerID != cfg.UserID {
			continue
		}
		if asset.ExifInfo.Latitude == nil || asset.ExifInfo.Longitude == nil {
			continue
		}
		candidate, ok := candidateForAsset(asset, req, maxDistance)
		if ok {
			matches = append(matches, candidate)
		}
	}
	return matches
}

func sortMatches(matches []assetCandidate) []assetCandidate {
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Distance == matches[j].Distance {
			return matches[i].TakenAt > matches[j].TakenAt
		}
		return matches[i].Distance < matches[j].Distance
	})
	return matches
}

func candidateForAsset(asset immichAsset, req assetLibraryRequest, maxDistance float64) (assetCandidate, bool) {
	lat := *asset.ExifInfo.Latitude
	lon := *asset.ExifInfo.Longitude
	bestLat := req.Lat
	bestLon := req.Lon
	bestDistance := math.MaxFloat64
	bestFromStart := 0.0

	if len(req.Points) > 0 {
		for _, point := range req.Points {
			distance := haversineMeters(lat, lon, point.Lat, point.Lon)
			if distance < bestDistance {
				bestDistance = distance
				bestLat = point.Lat
				bestLon = point.Lon
				bestFromStart = point.Distance
			}
		}
	} else if req.Lat != 0 || req.Lon != 0 {
		bestDistance = haversineMeters(lat, lon, req.Lat, req.Lon)
	}

	if bestDistance > maxDistance {
		return assetCandidate{}, false
	}
	return assetCandidate{
		AssetID:           asset.ID,
		OriginalFileName:  asset.OriginalFileName,
		TakenAt:           asset.FileCreatedAt,
		Lat:               lat,
		Lon:               lon,
		Distance:          bestDistance,
		PointLat:          bestLat,
		PointLon:          bestLon,
		DistanceFromStart: bestFromStart,
		City:              asset.ExifInfo.City,
		Country:           asset.ExifInfo.Country,
	}, true
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)
	rLat1 := degreesToRadians(lat1)
	rLat2 := degreesToRadians(lat2)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}
