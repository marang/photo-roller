package photoroller

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/marang/photo-roller/internal/config"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "photoroller",
	Short: "Import photos/videos from DCIM into day-based album folders",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Lang != "de" && cfg.Lang != "en" {
			return fmt.Errorf("unsupported --lang %q (use de|en)", cfg.Lang)
		}
		if cfg.GeohashPrecision < config.MinGeohashPrecision || cfg.GeohashPrecision > config.MaxGeohashPrecision {
			return fmt.Errorf("unsupported --geohash-precision %d (use %d..%d)", cfg.GeohashPrecision, config.MinGeohashPrecision, config.MaxGeohashPrecision)
		}
		if cfg.SegmentGapMinutes < config.MinSegmentGapMinutes {
			return fmt.Errorf("unsupported --segment-gap-minutes %d (must be >= %d)", cfg.SegmentGapMinutes, config.MinSegmentGapMinutes)
		}
		switch cfg.CollisionMode {
		case config.CollisionModeSuffix, config.CollisionModeAsk, config.CollisionModeFail:
		default:
			return fmt.Errorf("unsupported --collision-mode %q (use suffix|ask|fail)", cfg.CollisionMode)
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfg.Source, "source", config.DefaultSource, "source directory")
	rootCmd.PersistentFlags().StringVar(&cfg.Target, "target", config.DefaultTarget, "target directory")
	rootCmd.PersistentFlags().StringVar(&cfg.Lang, "lang", config.DefaultLang, "location language (de|en)")
	rootCmd.PersistentFlags().BoolVar(&cfg.Confirm, "confirm", false, "ask for confirmation before executing apply")
	rootCmd.PersistentFlags().StringVar(&cfg.Geocoder, "geocoder", config.DefaultGeocoder, "reverse geocoder provider (nominatim|none)")
	rootCmd.PersistentFlags().StringVar(&cfg.GeocodeCache, "geocode-cache", config.DefaultGeocodeCache, "path to geocode cache json")
	rootCmd.PersistentFlags().IntVar(&cfg.GeohashPrecision, "geohash-precision", config.DefaultGeohashPrecision, "geohash precision for clustering (1..12)")
	rootCmd.PersistentFlags().IntVar(&cfg.SegmentGapMinutes, "segment-gap-minutes", config.DefaultSegmentGapMinutes, "time gap in minutes that starts a new segment")
	rootCmd.PersistentFlags().StringVar(&cfg.CollisionMode, "collision-mode", config.DefaultCollisionMode, "filename collision handling: suffix|ask|fail")

	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(runCmd)
}
