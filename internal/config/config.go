package config

import (
	"os"
	"strconv"
	"strings"
)

func getenvBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key) // فرقش با Getenv اینه که می‌فهمیم اصلاً ست شده یا نه
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return b
}

type Config struct {
	Port            string
	MongoURI        string
	MongoDB         string
	URLAirports     string
	URLCountries    string
	URLRegions      string
	IngestSchedule  string
	URLFIRs         string
	FIRCountry      string
	WIKIAPI         string
	FAACLIENTID     string
	FAACLIENTSECRET string
	//AUTH
	AuthStrictMode    bool   // true: فقط HMAC؛ false: حالت permissive (HMAC یا raw secret)
	DateSkewSeconds   int    // مثلا 60
	NonceTTLSeconds   int    // مثلا 600
	DefaultRatePerMin int    // fallback اگر در داکیومنت مشتری نبود (مثلا 29)
	MasterKeyBase64   string // برای رمزکردن secret‌ها (فعلا می‌تونه خالی باشه)

}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func getenvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return def
	}
	return n
}

func Load() Config {
	return Config{
		Port:              getenv("PORT", "8086"),
		MongoURI:          getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:           getenv("MONGO_DB", "aviation"),
		URLAirports:       getenv("DATA_URL_AIRPORTS", "https://ourairports.com/data/airports.csv"),
		URLCountries:      getenv("DATA_URL_COUNTRIES", "https://ourairports.com/data/countries.csv"),
		URLRegions:        getenv("DATA_URL_REGIONS", "https://ourairports.com/data/regions.csv"),
		IngestSchedule:    getenv("INGEST_SCHEDULE", "@every 240h"), // 10 روز
		FIRCountry:        getenv("FIR_COUNTRY", "IR"),
		WIKIAPI:           getenv("WIKI_API", "https://www.wikiapi.com/"),
		FAACLIENTID:       getenv("FAACLIENTID", ""),
		FAACLIENTSECRET:   getenv("FAACLIENTSECRET", ""),
		AuthStrictMode:    getenvBool("AUTH_STRICT_MODE", false),
		DateSkewSeconds:   getenvInt("DATE_SKETW_EXTRACT_SECONDS", 60),
		NonceTTLSeconds:   getenvInt("NONCE_TTL_SECONDS", 60),
		DefaultRatePerMin: getenvInt("DEFAULT_RATE_PER_MIN", 0),
		MasterKeyBase64:   getenv("MASTER_KEY_BASE64", ""),
	}
}
