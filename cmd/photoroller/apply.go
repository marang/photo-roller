package photoroller

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/marang/photo-roller/internal/app"
	"github.com/marang/photo-roller/internal/config"
	"github.com/marang/photo-roller/internal/ui"
)

var assumeYes bool

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Copy files into day folders",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		result, err := app.BuildPlan(ctx, cfg)
		if err != nil {
			return err
		}
		if result.TotalFiles == 0 {
			fmt.Println("No media files found. Nothing to copy.")
			return nil
		}
		segments := 0
		for _, day := range result.Days {
			segments += len(day.Segments)
		}
		fmt.Printf("Plan summary: %d days, %d segments, %d files, %d geocode requests\n", len(result.Days), segments, result.TotalFiles, result.GeocodeRequests)
		fmt.Printf("Collision mode: %s\n", cfg.CollisionMode)
		if len(result.Warnings) > 0 {
			fmt.Printf("Warnings during scan: %d\n", len(result.Warnings))
		}

		resolvedMode, err := resolveCollisionMode(cfg.CollisionMode, result)
		if err != nil {
			return err
		}
		cfg.CollisionMode = resolvedMode

		if cfg.Confirm && !assumeYes {
			ok, err := confirmPrompt()
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		applyCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		events := make(chan app.ProgressEvent, 64)
		errCh := make(chan error, 1)
		go func() {
			_, applyErr := app.ApplyPlanWithSummary(applyCtx, cfg, result, events)
			errCh <- applyErr
		}()

		model := ui.NewProgressModel(events, result.TotalFiles, "", "Copying planned files to target")
		p := tea.NewProgram(model)
		finalModel, err := p.Run()
		if err != nil {
			return err
		}
		if progressModel, ok := finalModel.(ui.Model); ok && progressModel.Canceled() {
			cancel()
		}

		if err := <-errCh; err != nil {
			return err
		}
		fmt.Println("Apply finished successfully.")
		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&assumeYes, "yes", false, "run without confirmation prompt")
}

func confirmPrompt() (bool, error) {
	fmt.Print("Proceed with copy? type 'yes' to continue: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(line), "yes"), nil
}

func resolveCollisionMode(mode string, result app.ScanResult) (string, error) {
	collisions := app.FindCollisions(result)
	if len(collisions) == 0 {
		if mode == config.CollisionModeAsk {
			return config.CollisionModeSuffix, nil
		}
		return mode, nil
	}
	if mode == config.CollisionModeFail {
		return "", fmt.Errorf("detected %d filename collisions; rerun with --collision-mode=suffix or --collision-mode=ask", len(collisions))
	}
	if mode == config.CollisionModeAsk {
		fmt.Printf("Detected %d filename collisions across segments.\n", len(collisions))
		fmt.Print("Use suffix strategy (_2, _3, ...) for this run? type 'yes' to continue: ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(strings.TrimSpace(line), "yes") {
			return "", fmt.Errorf("aborted due to filename collisions")
		}
		return config.CollisionModeSuffix, nil
	}
	return mode, nil
}
