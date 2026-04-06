package config

type Config struct {
	Source            string
	Target            string
	Lang              string
	Confirm           bool
	Geocoder          string
	GeocodeCache      string
	GeohashPrecision  int
	SegmentGapMinutes int
	CollisionMode     string
}

const (
	DefaultSource            = "/media/camera/DCIM/"
	DefaultTarget            = "/mnt/data/assets/__albums/"
	DefaultLang              = "de"
	DefaultGeocoder          = "nominatim"
	DefaultGeocodeCache      = ".cache/geocode.json"
	DefaultGeohashPrecision  = 7
	DefaultSegmentGapMinutes = 90
	MinGeohashPrecision      = 1
	MaxGeohashPrecision      = 12
	MinSegmentGapMinutes     = 1
	CollisionModeSuffix      = "suffix"
	CollisionModeAsk         = "ask"
	CollisionModeFail        = "fail"
	DefaultCollisionMode     = CollisionModeSuffix
)
