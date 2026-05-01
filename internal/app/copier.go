package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/marang/photo-roller/internal/config"
)

type ProgressEvent struct {
	Kind    string
	Message string
	Day     string
	Segment string
	Done    int
	Total   int
}

type ApplySummary struct {
	Copied          []PlannedCopy
	SkippedExisting []PlannedCopy
	Processed       int
}

const (
	EventDayStart     = "day_start"
	EventSegmentStart = "segment_start"
	EventFileDone     = "file_done"
	EventWarning      = "warning"
	EventDone         = "done"
)

const (
	dirPerm = 0o755

	rcloneArgCopyTo = "copyto"
)

func ApplyPlan(ctx context.Context, cfg config.Config, result ScanResult, events chan<- ProgressEvent) error {
	_, err := ApplyPlanWithSummary(ctx, cfg, result, events)
	return err
}

func ApplyPlanWithSummary(ctx context.Context, cfg config.Config, result ScanResult, events chan<- ProgressEvent) (ApplySummary, error) {
	defer close(events)

	if _, err := exec.LookPath("rclone"); err != nil {
		return ApplySummary{}, fmt.Errorf("rclone is required but was not found in PATH: %w", err)
	}

	summary := ApplySummary{}
	var errs []error
	total := result.TotalFiles
	done := 0

	for _, day := range result.Days {
		select {
		case <-ctx.Done():
			return summary, ctx.Err()
		default:
		}

		destDayDir := filepath.Join(cfg.Target, day.FolderName)
		if err := os.MkdirAll(destDayDir, dirPerm); err != nil {
			errs = append(errs, fmt.Errorf("create destination %s: %w", destDayDir, err))
			continue
		}

		if err := sendProgressEvent(ctx, events, ProgressEvent{
			Kind: EventDayStart,
			Day:  day.FolderName,
		}); err != nil {
			return summary, err
		}

		for _, segment := range day.Segments {
			if err := sendProgressEvent(ctx, events, ProgressEvent{
				Kind:    EventSegmentStart,
				Day:     day.FolderName,
				Segment: segment.FolderName,
				Done:    done,
				Total:   total,
			}); err != nil {
				return summary, err
			}

			segmentDir := filepath.Join(destDayDir, segment.FolderName)
			if err := os.MkdirAll(segmentDir, dirPerm); err != nil {
				errs = append(errs, fmt.Errorf("create destination %s: %w", segmentDir, err))
				continue
			}

			nameCounts := map[string]int{}
			for _, src := range segment.Files {
				select {
				case <-ctx.Done():
					return summary, ctx.Err()
				default:
				}

				base := filepath.Base(src)
				nameCounts[base]++
				targetName, nameErr := resolveTargetName(base, nameCounts[base], cfg.CollisionMode)
				if nameErr != nil {
					errs = append(errs, fmt.Errorf("collision handling in segment %s: %w", segment.FolderName, nameErr))
					if err := sendProgressEvent(ctx, events, ProgressEvent{
						Kind:    EventWarning,
						Day:     day.FolderName,
						Segment: segment.FolderName,
						Message: nameErr.Error(),
						Done:    done,
						Total:   total,
					}); err != nil {
						return summary, err
					}
					continue
				}
				dst := filepath.Join(segmentDir, targetName)
				copyItem := PlannedCopy{
					Source: src,
					Target: dst,
				}

				existingInfo, existingErr := os.Stat(dst)
				if existingErr == nil {
					srcInfo, statErr := os.Stat(src)
					if statErr != nil {
						errs = append(errs, fmt.Errorf("stat source %s: %w", src, statErr))
						if err := sendProgressEvent(ctx, events, ProgressEvent{
							Kind:    EventWarning,
							Day:     day.FolderName,
							Segment: segment.FolderName,
							Message: fmt.Sprintf("source stat failed: %s", base),
							Done:    done,
							Total:   total,
						}); err != nil {
							return summary, err
						}
						continue
					}
					if existingInfo.Size() != srcInfo.Size() {
						errs = append(errs, fmt.Errorf("target exists with different size: %s", dst))
						if err := sendProgressEvent(ctx, events, ProgressEvent{
							Kind:    EventWarning,
							Day:     day.FolderName,
							Segment: segment.FolderName,
							Message: fmt.Sprintf("target size mismatch: %s", base),
							Done:    done,
							Total:   total,
						}); err != nil {
							return summary, err
						}
						continue
					}
					summary.SkippedExisting = append(summary.SkippedExisting, copyItem)
					done++
					summary.Processed = done
					if err := sendProgressEvent(ctx, events, ProgressEvent{
						Kind:    EventFileDone,
						Day:     day.FolderName,
						Segment: segment.FolderName,
						Done:    done,
						Total:   total,
					}); err != nil {
						return summary, err
					}
					continue
				}
				if existingErr != nil && !os.IsNotExist(existingErr) {
					errs = append(errs, fmt.Errorf("stat target %s: %w", dst, existingErr))
					if err := sendProgressEvent(ctx, events, ProgressEvent{
						Kind:    EventWarning,
						Day:     day.FolderName,
						Segment: segment.FolderName,
						Message: fmt.Sprintf("target stat failed: %s", base),
						Done:    done,
						Total:   total,
					}); err != nil {
						return summary, err
					}
					continue
				}

				args := []string{
					rcloneArgCopyTo,
					src,
					dst,
				}
				cmd := exec.CommandContext(ctx, "rclone", args...)
				if out, err := cmd.CombinedOutput(); err != nil {
					errs = append(errs, fmt.Errorf("copy %s -> %s: %w (%s)", src, dst, err, strings.TrimSpace(string(out))))
					if err := sendProgressEvent(ctx, events, ProgressEvent{
						Kind:    EventWarning,
						Day:     day.FolderName,
						Segment: segment.FolderName,
						Message: fmt.Sprintf("copy failed: %s", base),
						Done:    done,
						Total:   total,
					}); err != nil {
						return summary, err
					}
					continue
				}

				summary.Copied = append(summary.Copied, copyItem)
				done++
				summary.Processed = done
				if err := sendProgressEvent(ctx, events, ProgressEvent{
					Kind:    EventFileDone,
					Day:     day.FolderName,
					Segment: segment.FolderName,
					Done:    done,
					Total:   total,
				}); err != nil {
					return summary, err
				}
			}
		}
	}

	if err := sendProgressEvent(ctx, events, ProgressEvent{
		Kind:  EventDone,
		Done:  done,
		Total: total,
	}); err != nil {
		return summary, err
	}
	return summary, errors.Join(errs...)
}

func sendProgressEvent(ctx context.Context, events chan<- ProgressEvent, event ProgressEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case events <- event:
		return nil
	}
}

func uniqueName(base string, count int) string {
	if count <= 1 {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	return fmt.Sprintf("%s_%d%s", stem, count, ext)
}

func resolveTargetName(base string, occurrence int, mode string) (string, error) {
	if occurrence <= 1 {
		return base, nil
	}
	switch mode {
	case config.CollisionModeSuffix, config.CollisionModeAsk:
		return uniqueName(base, occurrence), nil
	case config.CollisionModeFail:
		return "", fmt.Errorf("filename collision: %s", base)
	default:
		return "", fmt.Errorf("unsupported collision mode %q", mode)
	}
}
