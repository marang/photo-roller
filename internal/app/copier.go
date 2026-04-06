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

const (
	EventDayStart     = "day_start"
	EventSegmentStart = "segment_start"
	EventFileDone     = "file_done"
	EventWarning      = "warning"
	EventDone         = "done"
)

const (
	dirPerm = 0o755

	rcloneArgCopyTo         = "copyto"
	rcloneArgIgnoreExisting = "--ignore-existing"
)

func ApplyPlan(ctx context.Context, cfg config.Config, result ScanResult, events chan<- ProgressEvent) error {
	defer close(events)

	if _, err := exec.LookPath("rclone"); err != nil {
		return fmt.Errorf("rclone is required but was not found in PATH: %w", err)
	}

	var errs []error
	total := result.TotalFiles
	done := 0

	for _, day := range result.Days {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		destDayDir := filepath.Join(cfg.Target, day.FolderName)
		if err := os.MkdirAll(destDayDir, dirPerm); err != nil {
			errs = append(errs, fmt.Errorf("create destination %s: %w", destDayDir, err))
			continue
		}

		events <- ProgressEvent{
			Kind: EventDayStart,
			Day:  day.FolderName,
		}

		for _, segment := range day.Segments {
			events <- ProgressEvent{
				Kind:    EventSegmentStart,
				Day:     day.FolderName,
				Segment: segment.FolderName,
				Done:    done,
				Total:   total,
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
					return ctx.Err()
				default:
				}

				base := filepath.Base(src)
				nameCounts[base]++
				targetName, nameErr := resolveTargetName(base, nameCounts[base], cfg.CollisionMode)
				if nameErr != nil {
					errs = append(errs, fmt.Errorf("collision handling in segment %s: %w", segment.FolderName, nameErr))
					events <- ProgressEvent{
						Kind:    EventWarning,
						Day:     day.FolderName,
						Segment: segment.FolderName,
						Message: nameErr.Error(),
						Done:    done,
						Total:   total,
					}
					continue
				}
				dst := filepath.Join(segmentDir, targetName)

				args := []string{
					rcloneArgCopyTo,
					src,
					dst,
					rcloneArgIgnoreExisting,
				}
				cmd := exec.CommandContext(ctx, "rclone", args...)
				if out, err := cmd.CombinedOutput(); err != nil {
					errs = append(errs, fmt.Errorf("copy %s -> %s: %w (%s)", src, dst, err, strings.TrimSpace(string(out))))
					events <- ProgressEvent{
						Kind:    EventWarning,
						Day:     day.FolderName,
						Segment: segment.FolderName,
						Message: fmt.Sprintf("copy failed: %s", base),
						Done:    done,
						Total:   total,
					}
					continue
				}

				done++
				events <- ProgressEvent{
					Kind:    EventFileDone,
					Day:     day.FolderName,
					Segment: segment.FolderName,
					Done:    done,
					Total:   total,
				}
			}
		}
	}

	events <- ProgressEvent{
		Kind:  EventDone,
		Done:  done,
		Total: total,
	}
	return errors.Join(errs...)
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
