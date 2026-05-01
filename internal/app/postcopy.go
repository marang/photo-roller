package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PlannedCopy struct {
	Source string
	Target string
}

type VerifySummary struct {
	Planned         int
	Verified        int
	MissingTargets  int
	SizeMismatches  int
	SourceMissing   int
	SourceStatError int
	TargetStatError int
}

type DeleteSummary struct {
	DeletedFiles int
	DeletedDirs  int
}

func IteratePlannedCopies(cfgTarget, collisionMode string, result ScanResult, fn func(PlannedCopy) error) error {
	for _, day := range result.Days {
		destDayDir := filepath.Join(cfgTarget, day.FolderName)
		for _, segment := range day.Segments {
			segmentDir := filepath.Join(destDayDir, segment.FolderName)
			nameCounts := map[string]int{}
			for _, src := range segment.Files {
				base := filepath.Base(src)
				nameCounts[base]++
				targetName, err := resolveTargetName(base, nameCounts[base], collisionMode)
				if err != nil {
					return err
				}
				copyItem := PlannedCopy{
					Source: src,
					Target: filepath.Join(segmentDir, targetName),
				}
				if err := fn(copyItem); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func VerifyPlannedCopies(result ScanResult, target, collisionMode string) (VerifySummary, error) {
	summary := VerifySummary{}
	err := IteratePlannedCopies(target, collisionMode, result, func(item PlannedCopy) error {
		summary.Planned++
		srcInfo, srcErr := os.Stat(item.Source)
		if srcErr != nil {
			if os.IsNotExist(srcErr) {
				summary.SourceMissing++
				return nil
			}
			summary.SourceStatError++
			return nil
		}

		dstInfo, dstErr := os.Stat(item.Target)
		if dstErr != nil {
			if os.IsNotExist(dstErr) {
				summary.MissingTargets++
				return nil
			}
			summary.TargetStatError++
			return nil
		}

		if srcInfo.Size() != dstInfo.Size() {
			summary.SizeMismatches++
			return nil
		}

		summary.Verified++
		return nil
	})
	if err != nil {
		return summary, err
	}
	return summary, nil
}

func DeleteVerifiedSources(copied []PlannedCopy, sourceRoot string) (DeleteSummary, error) {
	summary := DeleteSummary{}
	for _, item := range copied {
		if !isPathUnderRoot(item.Source, sourceRoot) {
			continue
		}
		srcInfo, srcErr := os.Stat(item.Source)
		if srcErr != nil {
			continue
		}
		dstInfo, dstErr := os.Stat(item.Target)
		if dstErr != nil {
			continue
		}
		if srcInfo.Size() != dstInfo.Size() {
			continue
		}
		if err := os.Remove(item.Source); err != nil {
			continue
		}
		summary.DeletedFiles++
	}

	dirs, dirErr := removeEmptyDirsUnder(sourceRoot)
	if dirErr != nil {
		return summary, dirErr
	}
	summary.DeletedDirs = dirs
	return summary, nil
}

func isPathUnderRoot(path, root string) bool {
	absPath, pathErr := filepath.Abs(path)
	absRoot, rootErr := filepath.Abs(root)
	if pathErr != nil || rootErr != nil {
		return false
	}
	rel, relErr := filepath.Rel(absRoot, absPath)
	if relErr != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func removeEmptyDirsUnder(root string) (int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(root, entry.Name())
		subDeleted, subErr := removeEmptyDirsUnder(dirPath)
		deleted += subDeleted
		if subErr != nil {
			continue
		}
		empty, checkErr := isDirEmpty(dirPath)
		if checkErr != nil || !empty {
			continue
		}
		if err := os.Remove(dirPath); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

func isDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func (s VerifySummary) FailedCount() int {
	return s.MissingTargets + s.SizeMismatches + s.SourceMissing + s.SourceStatError + s.TargetStatError
}

func (s VerifySummary) String() string {
	return fmt.Sprintf(
		"planned=%d verified=%d missing_target=%d size_mismatch=%d source_missing=%d source_stat_error=%d target_stat_error=%d",
		s.Planned,
		s.Verified,
		s.MissingTargets,
		s.SizeMismatches,
		s.SourceMissing,
		s.SourceStatError,
		s.TargetStatError,
	)
}
