package app

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mmcloughlin/geohash"
	"github.com/rwcarlsen/goexif/exif"

	"github.com/marang/photo-roller/internal/config"
)

type SegmentMode string

const (
	SegmentModeStationary SegmentMode = "stationary"
	SegmentModeMoving     SegmentMode = "moving"
	SegmentModeUnknown    SegmentMode = "unknown"
)

const (
	labelNoGPS           = "no_gps"
	labelUnknownLocation = "unknown_location"
	labelStationary      = "stationary"
	labelHike            = "hike"
	labelOnRoute         = "on_route"

	exifDateTimeLayout = "2006:01:02 15:04:05"
	timeHHMMLayout     = "1504"

	movingMinClusters          = 4
	movingDominantRatioMax     = 0.60
	movingDistanceKmThreshold  = 8.0
	hikeDurationThreshold      = 2 * time.Hour
	initialDateOrderCapacity   = 16
	initialSegmentSliceCap     = 4
	initialAssetBufferCapacity = 1024
)

type SegmentPlan struct {
	Start      time.Time
	End        time.Time
	Mode       SegmentMode
	Label      string
	FolderName string
	Files      []string
	GPSPoints  int
}

type DayPlan struct {
	Date       string
	FolderName string
	Summary    string
	FilesCount int
	GPSPoints  int
	Segments   []SegmentPlan
}

type ScanResult struct {
	Days            []DayPlan
	TotalFiles      int
	GeocodeRequests int
	Warnings        []string
}

type assetMeta struct {
	Path        string
	CaptureTime time.Time
	HasGPS      bool
	Lat         float64
	Lon         float64
}

func BuildPlan(ctx context.Context, cfg config.Config) (ScanResult, error) {
	resolver, err := NewLocationResolver(cfg)
	if err != nil {
		return ScanResult{}, err
	}

	byDate, totalFiles, warnings, err := scanAssetsByDate(ctx, cfg.Source)
	if err != nil {
		_ = resolver.Close()
		return ScanResult{}, err
	}
	if totalFiles == 0 {
		if closeErr := resolver.Close(); closeErr != nil {
			warnings = append(warnings, fmt.Sprintf("failed to save geocode cache: %v", closeErr))
		}
		return ScanResult{
			TotalFiles: 0,
			Warnings:   warnings,
		}, nil
	}

	dateOrder := make([]string, 0, len(byDate))
	for date := range byDate {
		dateOrder = append(dateOrder, date)
	}
	sort.Strings(dateOrder)

	days := make([]DayPlan, 0, len(dateOrder))
	segmentGap := time.Duration(cfg.SegmentGapMinutes) * time.Minute
	for _, date := range dateOrder {
		assets := byDate[date]
		sort.SliceStable(assets, func(i, j int) bool {
			if assets[i].CaptureTime.Equal(assets[j].CaptureTime) {
				return assets[i].Path < assets[j].Path
			}
			return assets[i].CaptureTime.Before(assets[j].CaptureTime)
		})

		segmentsAssets := splitByTimeGap(assets, segmentGap)
		segments := make([]SegmentPlan, 0, len(segmentsAssets))
		summaryParts := make([]string, 0, len(segmentsAssets))
		gpsPoints := 0
		for _, chunk := range segmentsAssets {
			seg, segWarnings := buildSegmentPlan(ctx, cfg, resolver, date, chunk)
			warnings = append(warnings, segWarnings...)
			segments = append(segments, seg)
			summaryParts = append(summaryParts, summaryPartForSegment(seg))
			gpsPoints += seg.GPSPoints
		}
		daySummary := buildDaySummary(summaryParts)
		dayFolder := date + "_" + daySummary
		days = append(days, DayPlan{
			Date:       date,
			FolderName: dayFolder,
			Summary:    daySummary,
			FilesCount: len(assets),
			GPSPoints:  gpsPoints,
			Segments:   segments,
		})
	}

	if closeErr := resolver.Close(); closeErr != nil {
		warnings = append(warnings, fmt.Sprintf("failed to save geocode cache: %v", closeErr))
	}

	return ScanResult{
		Days:            days,
		TotalFiles:      totalFiles,
		GeocodeRequests: resolver.RequestCount(),
		Warnings:        warnings,
	}, nil
}

func scanAssetsByDate(ctx context.Context, source string) (map[string][]assetMeta, int, []string, error) {
	byDate := make(map[string][]assetMeta)
	warnings := []string{}
	total := 0

	err := filepath.WalkDir(source, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !isMediaExt(ext) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("stat failed for %s: %v", path, err))
			return nil
		}

		meta := extractMetadata(path, info.ModTime())
		date := meta.CaptureTime.Local().Format(time.DateOnly)
		if _, ok := byDate[date]; !ok {
			byDate[date] = make([]assetMeta, 0, initialAssetBufferCapacity)
		}
		byDate[date] = append(byDate[date], assetMeta{
			Path:        path,
			CaptureTime: meta.CaptureTime,
			HasGPS:      meta.HasGPS,
			Lat:         meta.Lat,
			Lon:         meta.Lon,
		})
		total++
		if meta.Warn != "" {
			warnings = append(warnings, meta.Warn)
		}
		return nil
	})
	if err != nil {
		return nil, 0, warnings, err
	}
	if len(byDate) == 0 {
		return byDate, 0, warnings, nil
	}

	// Keep map allocations tighter for downstream work.
	ordered := make(map[string][]assetMeta, len(byDate))
	dateOrder := make([]string, 0, initialDateOrderCapacity)
	for date := range byDate {
		dateOrder = append(dateOrder, date)
	}
	sort.Strings(dateOrder)
	for _, date := range dateOrder {
		ordered[date] = byDate[date]
	}
	return ordered, total, warnings, nil
}

func splitByTimeGap(assets []assetMeta, maxGap time.Duration) [][]assetMeta {
	if len(assets) == 0 {
		return nil
	}
	segments := make([][]assetMeta, 0, initialSegmentSliceCap)
	current := []assetMeta{assets[0]}
	for i := 1; i < len(assets); i++ {
		gap := assets[i].CaptureTime.Sub(assets[i-1].CaptureTime)
		if gap > maxGap {
			segments = append(segments, current)
			current = []assetMeta{assets[i]}
			continue
		}
		current = append(current, assets[i])
	}
	segments = append(segments, current)
	return segments
}

func buildSegmentPlan(ctx context.Context, cfg config.Config, resolver *LocationResolver, date string, assets []assetMeta) (SegmentPlan, []string) {
	warnings := []string{}
	files := make([]string, 0, len(assets))
	gps := make([]GeoPoint, 0, len(assets))
	for _, asset := range assets {
		files = append(files, asset.Path)
		if asset.HasGPS {
			gps = append(gps, GeoPoint{Lat: asset.Lat, Lon: asset.Lon})
		}
	}

	start := assets[0].CaptureTime.Local()
	end := assets[len(assets)-1].CaptureTime.Local()
	label := labelNoGPS
	mode := SegmentModeUnknown

	if len(gps) > 0 {
		isMoving, dominant := classifyMovement(gps, cfg.GeohashPrecision)
		if isMoving {
			mode = SegmentModeMoving
			segmentLabel, warn := movingSegmentLabel(ctx, cfg, resolver, gps, end.Sub(start))
			if warn != "" {
				warnings = append(warnings, warn)
			}
			label = segmentLabel
		} else {
			mode = SegmentModeStationary
			name, err := resolver.Resolve(ctx, dominant, cfg.Lang)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("reverse geocoding stationary segment on %s failed: %v", date, err))
				label = labelStationary
			} else {
				label = snakeCaseLocation(name)
			}
		}
	}

	folder := segmentFolderName(date, start, end, label)
	return SegmentPlan{
		Start:      start,
		End:        end,
		Mode:       mode,
		Label:      label,
		FolderName: folder,
		Files:      files,
		GPSPoints:  len(gps),
	}, warnings
}

func classifyMovement(points []GeoPoint, precision int) (bool, GeoPoint) {
	if len(points) == 0 {
		return false, GeoPoint{}
	}
	type cluster struct {
		count int
		point GeoPoint
	}
	clusters := map[string]cluster{}
	for _, point := range points {
		hash := geohash.EncodeWithPrecision(point.Lat, point.Lon, uint(precision))
		entry := clusters[hash]
		entry.count++
		if entry.count == 1 {
			entry.point = point
		}
		clusters[hash] = entry
	}

	dominant := cluster{count: -1, point: points[0]}
	for _, entry := range clusters {
		if entry.count > dominant.count {
			dominant = entry
		}
	}

	dominantRatio := float64(dominant.count) / float64(len(points))
	clusterCount := len(clusters)
	distanceKm := haversineKm(points[0], points[len(points)-1])
	isMoving := clusterCount >= movingMinClusters || dominantRatio < movingDominantRatioMax || distanceKm >= movingDistanceKmThreshold
	return isMoving, dominant.point
}

func movingSegmentLabel(ctx context.Context, cfg config.Config, resolver *LocationResolver, points []GeoPoint, duration time.Duration) (string, string) {
	startName, startErr := resolver.Resolve(ctx, points[0], cfg.Lang)
	endName, endErr := resolver.Resolve(ctx, points[len(points)-1], cfg.Lang)
	if startErr != nil {
		return labelHike, fmt.Sprintf("reverse geocoding moving-segment start failed: %v", startErr)
	}
	if endErr != nil {
		return labelHike, fmt.Sprintf("reverse geocoding moving-segment end failed: %v", endErr)
	}

	startSlug := snakeCaseLocation(startName)
	endSlug := snakeCaseLocation(endName)
	if startSlug != labelUnknownLocation && endSlug != labelUnknownLocation && startSlug != endSlug {
		return labelHike + "_" + startSlug + "_to_" + endSlug, ""
	}
	if duration >= hikeDurationThreshold {
		return labelHike, ""
	}
	return labelOnRoute, ""
}

func buildDaySummary(parts []string) string {
	if len(parts) == 0 {
		return labelNoGPS
	}
	unique := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		if part == "" {
			continue
		}
		if seen[part] {
			continue
		}
		seen[part] = true
		unique = append(unique, part)
	}
	if len(unique) == 0 {
		return labelNoGPS
	}
	return strings.Join(unique, "_and_")
}

func summaryPartForSegment(seg SegmentPlan) string {
	if seg.Mode == SegmentModeMoving {
		return labelHike
	}
	if seg.Label == "" {
		return labelNoGPS
	}
	return seg.Label
}

func segmentFolderName(date string, start, end time.Time, label string) string {
	return fmt.Sprintf("%s_%s-%s_%s", date, start.Format(timeHHMMLayout), end.Format(timeHHMMLayout), label)
}

type FileMetadata struct {
	CaptureTime time.Time
	HasGPS      bool
	Lat         float64
	Lon         float64
	Warn        string
}

func extractMetadata(path string, fallbackTime time.Time) FileMetadata {
	meta := FileMetadata{
		CaptureTime: fallbackTime,
	}
	if !isExifCandidate(path) {
		return meta
	}

	f, err := os.Open(path)
	if err != nil {
		meta.Warn = fmt.Sprintf("failed to open EXIF file %s: %v", path, err)
		return meta
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return meta
	}

	if ts, err := exifDateTimeOriginal(x); err == nil {
		meta.CaptureTime = ts
	}
	if lat, lon, err := x.LatLong(); err == nil {
		meta.HasGPS = true
		meta.Lat = lat
		meta.Lon = lon
	}
	return meta
}

func exifDateTimeOriginal(x *exif.Exif) (time.Time, error) {
	tag, err := x.Get(exif.DateTimeOriginal)
	if err == nil {
		if value, valueErr := tag.StringVal(); valueErr == nil {
			if t, parseErr := time.ParseInLocation(exifDateTimeLayout, value, time.Local); parseErr == nil {
				return t, nil
			}
		}
	}
	return x.DateTime()
}

func isExifCandidate(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".tif", ".tiff", ".dng", ".cr2", ".nef", ".arw", ".rw2", ".heic", ".heif":
		return true
	default:
		return false
	}
}

func isMediaExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".heic", ".heif", ".png", ".webp", ".dng", ".cr2", ".nef", ".arw", ".rw2", ".mp4", ".mov", ".avi", ".mkv":
		return true
	default:
		return false
	}
}
