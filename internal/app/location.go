package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/marang/photo-roller/internal/config"
)

type GeoPoint struct {
	Lat float64
	Lon float64
}

const (
	geocoderTimeout         = 10 * time.Second
	geocoderRateLimitWindow = 1 * time.Second
	nominatimZoomLevel      = "10"
	nominatimFormat         = "jsonv2"
	nominatimAddressDetails = "1"
	userAgentValue          = "PhotoRoller/0.1"
	cacheCoordPrecision     = "%.4f,%.4f"
)

type LocationResolver struct {
	mu          sync.Mutex
	cachePath   string
	cache       map[string]string
	geocoder    reverseGeocoder
	lastRequest time.Time
	requests    int
}

type reverseGeocoder interface {
	Reverse(lat, lon float64, lang string) (string, error)
}

func NewLocationResolver(cfg config.Config) (*LocationResolver, error) {
	cache, err := loadCache(cfg.GeocodeCache)
	if err != nil {
		return nil, err
	}

	var g reverseGeocoder
	switch strings.ToLower(cfg.Geocoder) {
	case "", "nominatim":
		g = &nominatimGeocoder{
			client: &http.Client{Timeout: geocoderTimeout},
		}
	case "none", "off":
		g = nil
	default:
		return nil, fmt.Errorf("unsupported geocoder %q", cfg.Geocoder)
	}

	return &LocationResolver{
		cachePath: cfg.GeocodeCache,
		cache:     cache,
		geocoder:  g,
	}, nil
}

func (r *LocationResolver) Resolve(ctx context.Context, point GeoPoint, lang string) (string, error) {
	key := cacheKey(point, lang)

	r.mu.Lock()
	if value, ok := r.cache[key]; ok {
		r.mu.Unlock()
		return value, nil
	}
	r.mu.Unlock()

	if r.geocoder == nil {
		return "", nil
	}

	if err := r.waitForRateLimit(ctx); err != nil {
		return "", err
	}

	location, err := r.geocoder.Reverse(point.Lat, point.Lon, lang)
	if err != nil {
		return "", err
	}

	r.mu.Lock()
	r.cache[key] = location
	r.requests++
	r.lastRequest = time.Now()
	r.mu.Unlock()

	return location, nil
}

func (r *LocationResolver) waitForRateLimit(ctx context.Context) error {
	r.mu.Lock()
	since := time.Since(r.lastRequest)
	r.mu.Unlock()

	delay := geocoderRateLimitWindow - since
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r *LocationResolver) RequestCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests
}

func (r *LocationResolver) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return saveCache(r.cachePath, r.cache)
}

func loadCache(path string) (map[string]string, error) {
	if path == "" {
		return map[string]string{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return map[string]string{}, nil
	}
	cache := map[string]string{}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	return cache, nil
}

func saveCache(path string, cache map[string]string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func cacheKey(point GeoPoint, lang string) string {
	return fmt.Sprintf(cacheCoordPrecision, point.Lat, point.Lon) + "|" + strings.ToLower(strings.TrimSpace(lang))
}

type nominatimGeocoder struct {
	client *http.Client
}

type nominatimReverseResponse struct {
	DisplayName string `json:"display_name"`
	Address     struct {
		City         string `json:"city"`
		Town         string `json:"town"`
		Village      string `json:"village"`
		Municipality string `json:"municipality"`
		County       string `json:"county"`
		State        string `json:"state"`
		Country      string `json:"country"`
	} `json:"address"`
}

func (g *nominatimGeocoder) Reverse(lat, lon float64, lang string) (string, error) {
	query := url.Values{}
	query.Set("format", nominatimFormat)
	query.Set("lat", fmt.Sprintf("%.7f", lat))
	query.Set("lon", fmt.Sprintf("%.7f", lon))
	query.Set("zoom", nominatimZoomLevel)
	query.Set("addressdetails", nominatimAddressDetails)
	if lang != "" {
		query.Set("accept-language", lang)
	}

	endpoint := "https://nominatim.openstreetmap.org/reverse?" + query.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgentValue)

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("nominatim status %d", resp.StatusCode)
	}

	var payload nominatimReverseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	parts := []string{
		firstNonEmpty(payload.Address.City, payload.Address.Town, payload.Address.Village, payload.Address.Municipality, payload.Address.County, payload.Address.State),
		payload.Address.Country,
	}
	trimmed := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			trimmed = append(trimmed, p)
		}
	}
	if len(trimmed) > 0 {
		return strings.Join(trimmed, " "), nil
	}
	return strings.TrimSpace(payload.DisplayName), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
