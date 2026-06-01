package polyline

import (
	"fmt"
	"math"
)

func Decode(encoded string, precision float64) ([][2]float64, error) {
	if precision == 0 {
		return nil, fmt.Errorf("precision must not be zero")
	}
	var coords [][2]float64
	index := 0
	lat := 0
	lon := 0
	for index < len(encoded) {
		dlat, next, err := decodeValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = next
		dlon, next, err := decodeValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = next
		lat += dlat
		lon += dlon
		coords = append(coords, [2]float64{float64(lat) / precision, float64(lon) / precision})
	}
	return coords, nil
}

func DecodeValues(encoded string, precision float64) ([]float64, error) {
	if precision == 0 {
		return nil, fmt.Errorf("precision must not be zero")
	}
	var values []float64
	index := 0
	value := 0
	for index < len(encoded) {
		delta, next, err := decodeValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = next
		value += delta
		values = append(values, float64(value)/precision)
	}
	return values, nil
}

func NormalizeCoordinateScale(coords [][2]float64) {
	if len(coords) == 0 {
		return
	}
	maxLat := 0.0
	maxLon := 0.0
	for _, coord := range coords {
		if abs := math.Abs(coord[0]); abs > maxLat {
			maxLat = abs
		}
		if abs := math.Abs(coord[1]); abs > maxLon {
			maxLon = abs
		}
	}
	for (maxLat > 90 || maxLon > 180) && maxLat > 0 && maxLon > 0 {
		for i := range coords {
			coords[i][0] /= 10
			coords[i][1] /= 10
		}
		maxLat /= 10
		maxLon /= 10
	}
}

func ShouldSwapCoordinates(coords [][2]float64) bool {
	validAsLat := 0
	validAsLon := 0
	for _, coord := range coords {
		if validLatLon(coord[0], coord[1]) {
			validAsLat++
		}
		if validLatLon(coord[1], coord[0]) {
			validAsLon++
		}
	}
	return validAsLon > validAsLat
}

func ProportionalIndex(i int, sourceLen int, targetLen int) int {
	if targetLen <= 1 || sourceLen <= 1 {
		return 0
	}
	j := int(math.Round(float64(i) * float64(targetLen-1) / float64(sourceLen-1)))
	if j < 0 {
		return 0
	}
	if j >= targetLen {
		return targetLen - 1
	}
	return j
}

func validLatLon(lat float64, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func decodeValue(encoded string, index int) (int, int, error) {
	result := 0
	shift := uint(0)
	for {
		if index >= len(encoded) {
			return 0, index, fmt.Errorf("invalid polyline encoding")
		}
		b := int(encoded[index]) - 63
		index++
		result |= (b & 0x1F) << shift
		shift += 5
		if b < 0x20 {
			break
		}
	}
	return (result >> 1) ^ (-(result & 1)), index, nil
}
