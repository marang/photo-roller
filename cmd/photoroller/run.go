package photoroller

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/marang/photo-roller/internal/app"
	"github.com/marang/photo-roller/internal/config"
	"github.com/marang/photo-roller/internal/ui"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Interactive end-to-end flow (preflight, decisions, execute)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		selectSourceAndTarget := func() error {
			for {
				source, err := ui.RunDirectoryPicker(
					"Step 1 - Source Directory",
					"Path to camera/SD DCIM root",
					cfg.Source,
				)
				if err != nil {
					return err
				}
				cfg.Source = source

				target, err := ui.RunTargetDirectoryPicker(
					"Step 2 - Target Directory",
					"Source left (with analysis), destination browser right",
					cfg.Target,
					cfg.Source,
				)
				if err != nil {
					if err == ui.ErrBackToSource {
						continue
					}
					return err
				}
				cfg.Target = target
				break
			}
			return nil
		}

		selectTargetFromCurrentSource := func() error {
			for {
				target, err := ui.RunTargetDirectoryPicker(
					"Step 2 - Target Directory",
					"Source left (with analysis), destination browser right",
					cfg.Target,
					cfg.Source,
				)
				if err != nil {
					if err == ui.ErrBackToSource {
						source, srcErr := ui.RunDirectoryPicker(
							"Step 1 - Source Directory",
							"Path to camera/SD DCIM root",
							cfg.Source,
						)
						if srcErr != nil {
							return srcErr
						}
						cfg.Source = source
						continue
					}
					return err
				}
				cfg.Target = target
				return nil
			}
		}

		if err := selectSourceAndTarget(); err != nil {
			return err
		}

		for {
			result, err := app.BuildPlan(ctx, cfg)
			if err != nil {
				return err
			}
			if result.TotalFiles == 0 {
				fmt.Println("No media files found. Nothing to copy.")
				return nil
			}

			collisions := app.FindCollisions(result)
			wizard := newWizardState(&cfg, result, collisions)

			if len(collisions) > 0 && cfg.CollisionMode == config.CollisionModeAsk {
				modeChoice, modeErr := ui.RunWizardSelectPrompt(
					"Step 3 - Collision Mode",
					"Review detected filename collisions",
					wizard.LeftSummary(),
					fmt.Sprintf("Detected %d filename collisions.\nChoose how duplicates should be handled in this run.", len(collisions)),
					[]ui.SelectOption{
						{Title: "Confirm (Suffix Recommended)", Description: "Keep all files by appending _2, _3, ...", Value: config.CollisionModeSuffix},
						{Title: "Confirm (Fail)", Description: "Abort run when duplicate basename occurs", Value: config.CollisionModeFail},
					},
				)
				if modeErr != nil {
					return modeErr
				}
				if modeChoice == ui.WizardBackChoice {
					if err := selectTargetFromCurrentSource(); err != nil {
						return err
					}
					continue
				}
				cfg.CollisionMode = modeChoice
				wizard.AddConfirmedStep("Step 3 - Collision Mode", "Selected mode: "+cfg.CollisionMode)
			}

			preflightSummary := wizard.PreflightSummary()
			previewChoice, err := ui.RunWizardSelectPrompt(
				"Step 3 - Preflight Review",
				"Validate the planned import before copying",
				wizard.LeftSummary(),
				preflightSummary,
				[]ui.SelectOption{
					{Title: "Confirm", Description: "", Value: "confirm"},
				},
			)
			if err != nil {
				return err
			}
			if previewChoice == ui.WizardBackChoice {
				if err := selectTargetFromCurrentSource(); err != nil {
					return err
				}
				continue
			}
			if previewChoice != "confirm" {
				continue
			}
			wizard.AddConfirmedStep("Step 3 - Preflight Review", preflightSummary)

			executeSummary := wizard.ExecuteSummary()
			execChoice, err := ui.RunWizardSelectPrompt(
				"Step 4 - Execute Import",
				"Confirm and start copy/verify workflow",
				wizard.LeftSummary(),
				executeSummary,
				[]ui.SelectOption{
					{Title: "Confirm", Description: "", Value: "start"},
				},
			)
			if err != nil {
				return err
			}
			if execChoice == ui.WizardBackChoice {
				continue
			}
			if execChoice != "start" {
				continue
			}
			if execChoice == "start" {
				events := make(chan app.ProgressEvent, 64)
				errCh := make(chan error, 1)
				go func() {
					errCh <- app.ApplyPlan(ctx, cfg, result, events)
				}()

				model := ui.NewProgressModel(events, result.TotalFiles, wizard.LeftSummary(), "Copying planned files to target (no source deletion yet)")
				p := tea.NewProgram(model, tea.WithAltScreen())
				if _, err := p.Run(); err != nil {
					return err
				}
				applyErr := <-errCh

				verifySummary, verifyErr := app.VerifyPlannedCopies(result, cfg.Target, cfg.CollisionMode)
				if verifyErr != nil {
					return verifyErr
				}
				verifyText := wizard.VerifySummary(verifySummary, applyErr)
				afterChoice, promptErr := ui.RunWizardSelectPrompt(
					"Step 4 - Execute Import",
					"Review verification and choose SD-card cleanup",
					wizard.LeftSummary(),
					verifyText,
					[]ui.SelectOption{
						{Title: "Leave all files on SD card", Description: "", Value: "keep"},
						{Title: "Remove verified files from SD card", Description: "", Value: "delete_verified"},
					},
				)
				if promptErr != nil {
					return promptErr
				}
				if afterChoice == ui.WizardBackChoice {
					continue
				}
				if afterChoice == "delete_verified" {
					deleteSummary, delErr := app.DeleteVerifiedSources(result, cfg.Source, cfg.Target, cfg.CollisionMode)
					if delErr != nil {
						return delErr
					}
					fmt.Printf("Cleanup finished: deleted %d files and %d empty directories from source.\n", deleteSummary.DeletedFiles, deleteSummary.DeletedDirs)
				}

				if applyErr != nil {
					return fmt.Errorf("apply finished with errors: %w", applyErr)
				}
				fmt.Println("Run finished successfully.")
				return nil
			}
		}
	},
}
