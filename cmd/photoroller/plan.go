package photoroller

import (
	"context"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/marang/photo-roller/internal/app"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Preview grouped day folders and file counts",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := app.BuildPlan(context.Background(), cfg)
		if err != nil {
			return err
		}

		title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("PhotoRoller Plan")
		meta := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

		fmt.Println(title)
		fmt.Println(meta.Render("Source: " + cfg.Source))
		fmt.Println(meta.Render("Target: " + cfg.Target))
		fmt.Println(meta.Render("Geocoder: " + cfg.Geocoder))
		fmt.Println(meta.Render("Collision mode: " + cfg.CollisionMode))
		fmt.Printf("Geohash precision: %d | Segment gap: %dmin\n", cfg.GeohashPrecision, cfg.SegmentGapMinutes)
		fmt.Printf("Days: %d | Files: %d | Geocode requests: %d\n\n", len(result.Days), result.TotalFiles, result.GeocodeRequests)
		eventStats := app.BuildEventStats(result)
		fmt.Printf("Events: %d segments | %d distinct event types\n", eventStats.TotalSegments, eventStats.DistinctTypes)
		if len(eventStats.ByType) > 0 {
			max := min(len(eventStats.ByType), 8)
			for i := 0; i < max; i++ {
				e := eventStats.ByType[i]
				fmt.Printf("- %s: %d\n", e.Label, e.Count)
			}
		}
		fmt.Println()

		for _, day := range result.Days {
			fmt.Printf("%s (%d files, gps: %d, segments: %d)\n", day.FolderName, day.FilesCount, day.GPSPoints, len(day.Segments))
			for _, segment := range day.Segments {
				fmt.Printf("  %s | %s | files=%d gps=%d\n", segment.FolderName, segment.Mode, len(segment.Files), segment.GPSPoints)
			}
		}

		if len(result.Days) == 0 {
			fmt.Println("No media files found.")
		}
		if len(result.Warnings) > 0 {
			fmt.Printf("\nWarnings: %d (showing up to 5)\n", len(result.Warnings))
			max := min(len(result.Warnings), 5)
			for i := 0; i < max; i++ {
				fmt.Printf("- %s\n", result.Warnings[i])
			}
		}

		collisions := app.FindCollisions(result)
		if len(collisions) > 0 {
			fmt.Printf("\nPreflight collisions: %d (showing up to 10)\n", len(collisions))
			max := min(len(collisions), 10)
			for i := 0; i < max; i++ {
				c := collisions[i]
				fmt.Printf("- day=%s segment=%s file=%s count=%d\n", c.Day, c.Segment, c.FileName, c.Occurrences)
			}
			fmt.Println("Recommendation: rename edge cases on SD card before apply, or use --collision-mode=suffix.")
		}
		return nil
	},
}
