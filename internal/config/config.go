package config

import (
	"os"
)

type Config struct {
	Port           string
	MongoURI       string
	MongoDB        string
	URLAirports    string
	URLCountries   string
	URLRegions     string
	IngestSchedule string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func Load() Config {
	return Config{
		Port:           getenv("PORT", "8080"),
		MongoURI:       getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:        getenv("MONGO_DB", "aviation"),
		URLAirports:    getenv("DATA_URL_AIRPORTS", "https://ourairports.com/data/airports.csv"),
		URLCountries:   getenv("DATA_URL_COUNTRIES", "https://ourairports.com/data/countries.csv"),
		URLRegions:     getenv("DATA_URL_REGIONS", "https://ourairports.com/data/regions.csv"),
		IngestSchedule: getenv("INGEST_SCHEDULE", "@every 240h"), // 10 روز

	}
}
