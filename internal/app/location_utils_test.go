package app

import "testing"

func TestSnakeCaseLocation(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "Vienna Austria", want: "vienna_austria"},
		{in: "München, Bayern", want: "muenchen_bayern"},
		{in: "St. Pölten", want: "st_poelten"},
		{in: "   ", want: "unknown_location"},
	}

	for _, tt := range tests {
		got := snakeCaseLocation(tt.in)
		if got != tt.want {
			t.Fatalf("snakeCaseLocation(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDominantPoint(t *testing.T) {
	points := []GeoPoint{
		{Lat: 48.2082, Lon: 16.3738},
		{Lat: 48.2083, Lon: 16.3739},
		{Lat: 48.2090, Lon: 16.3741},
		{Lat: 47.0707, Lon: 15.4395},
	}
	got := dominantPoint(points)
	if got.Lat < 48.20 || got.Lat > 48.21 {
		t.Fatalf("dominant point lat out of expected range: %.6f", got.Lat)
	}
}
