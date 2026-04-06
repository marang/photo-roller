package app

import "path/filepath"

type CollisionStat struct {
	Day         string
	Segment     string
	FileName    string
	Occurrences int
}

func FindCollisions(result ScanResult) []CollisionStat {
	stats := []CollisionStat{}
	for _, day := range result.Days {
		for _, segment := range day.Segments {
			counts := map[string]int{}
			for _, src := range segment.Files {
				base := filepath.Base(src)
				counts[base]++
			}
			for name, count := range counts {
				if count > 1 {
					stats = append(stats, CollisionStat{
						Day:         day.FolderName,
						Segment:     segment.FolderName,
						FileName:    name,
						Occurrences: count,
					})
				}
			}
		}
	}
	return stats
}
