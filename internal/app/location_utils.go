package app

import (
	"fmt"
	"math"
	"strings"
	"unicode"
)

func dominantPoint(points []GeoPoint) GeoPoint {
	if len(points) == 0 {
		return GeoPoint{}
	}
	type cluster struct {
		count int
		point GeoPoint
	}
	byCell := map[string]cluster{}
	for _, point := range points {
		key := fmt.Sprintf("%.3f,%.3f", point.Lat, point.Lon)
		entry := byCell[key]
		entry.count++
		if entry.count == 1 {
			entry.point = point
		}
		byCell[key] = entry
	}

	bestCount := -1
	bestKey := ""
	bestPoint := points[0]
	for key, entry := range byCell {
		if entry.count > bestCount || (entry.count == bestCount && key < bestKey) {
			bestCount = entry.count
			bestKey = key
			bestPoint = entry.point
		}
	}
	return bestPoint
}

func snakeCaseLocation(input string) string {
	replacer := strings.NewReplacer(
		"ä", "ae",
		"ö", "oe",
		"ü", "ue",
		"ß", "ss",
	)
	normalized := strings.ToLower(replacer.Replace(strings.TrimSpace(input)))

	var b strings.Builder
	underscore := false
	for _, r := range normalized {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			underscore = false
		default:
			if !underscore && b.Len() > 0 {
				b.WriteByte('_')
				underscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "unknown_location"
	}
	return out
}

func haversineKm(a, b GeoPoint) float64 {
	const earthRadiusKm = 6371.0
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180

	sinDLat := math.Sin(dLat / 2)
	sinDLon := math.Sin(dLon / 2)
	h := sinDLat*sinDLat + math.Cos(lat1)*math.Cos(lat2)*sinDLon*sinDLon
	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
	return earthRadiusKm * c
}
