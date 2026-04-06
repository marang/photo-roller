package app

import "sort"

type EventCount struct {
	Label string
	Count int
}

type EventStats struct {
	TotalSegments int
	DistinctTypes int
	ByType        []EventCount
}

func BuildEventStats(result ScanResult) EventStats {
	counts := map[string]int{}
	totalSegments := 0
	for _, day := range result.Days {
		for _, segment := range day.Segments {
			totalSegments++
			label := segment.Label
			if label == "" {
				label = "unknown"
			}
			counts[label]++
		}
	}

	byType := make([]EventCount, 0, len(counts))
	for label, count := range counts {
		byType = append(byType, EventCount{Label: label, Count: count})
	}
	sort.Slice(byType, func(i, j int) bool {
		if byType[i].Count == byType[j].Count {
			return byType[i].Label < byType[j].Label
		}
		return byType[i].Count > byType[j].Count
	})

	return EventStats{
		TotalSegments: totalSegments,
		DistinctTypes: len(byType),
		ByType:        byType,
	}
}
