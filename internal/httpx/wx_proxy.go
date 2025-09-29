package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const awcBase = "https://aviationweather.gov/api/data"

// validate ICAO (چهار کاراکتر A-Z/0-9؛ معمولاً حروف)
var icaoRe = regexp.MustCompile(`^[A-Z0-9]{4}$`)

func parseHours(q string, def int) int {
	if q == "" {
		return def
	}
	h, err := strconv.Atoi(q)
	if err != nil || h <= 0 {
		return def
	}
	return h
}

func fetchAWC(ctx context.Context, resource, icao string, hours int) ([]byte, int, error) {
	q := url.Values{}
	q.Set("format", "json")
	q.Set("ids", strings.ToUpper(icao))
	if hours > 0 {
		q.Set("hours", fmt.Sprintf("%d", hours))
	}
	q.Set("mostRecent", "true") // optional

	u := fmt.Sprintf("%s/%s?%s", awcBase, resource, q.Encode())
	req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
	req.Header.Set("User-Agent", "SepTaf-WX/1.0 (contact: you@example.com)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return b, resp.StatusCode, fmt.Errorf("awc %s http %d", resource, resp.StatusCode)
	}
	return b, resp.StatusCode, nil
}

// ----------------- Handlers -----------------

// FIR LIST godoc
// @Summary      Get METAR
// @Description  Returns METAR JSON from AWC for a given ICAO (no storage)
// @Tags         Weather
// @Produce      json
// @Param        icao   query   string  true   "ICAO code (e.g., OIII, KJFK)"
// @Param        hours  query   int     false  "Lookback hours (default 2)"
// @Success      200    {object}  map[string]any
// @Failure      400    {object}  map[string]string
// @Failure      502    {object}  map[string]string
// @Router       /wx/metar [get]
func GetMETAR(w http.ResponseWriter, r *http.Request) {
	icao := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("icao")))
	if !icaoRe.MatchString(icao) {
		http.Error(w, `{"error":"invalid ICAO"}`, http.StatusBadRequest)
		return
	}
	hours := parseHours(r.URL.Query().Get("hours"), 2)

	body, code, err := fetchAWC(r.Context(), "metar", icao, hours)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if len(body) > 0 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"%s","upstream":%q}`, err.Error(), string(body))))
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	// ✅ قبول هر نوع JSON (object/array)
	var js any
	if e := json.Unmarshal(body, &js); e != nil {
		http.Error(w,
			fmt.Sprintf(`{"error":"invalid upstream json","upstream":%q}`, string(body)),
			http.StatusBadGateway)
		return
	}

	w.WriteHeader(code)
	_, _ = w.Write(body)
}

// @Summary      Get TAF
// @Description  Returns TAF JSON from AWC for a given ICAO (no storage)
// @Tags         Weather
// @Produce      json
// @Param        icao   query   string  true   "ICAO code (e.g., OIII, KJFK)"
// @Param        hours  query   int     false  "Lookback hours (default 24)"
// @Success      200    {object}  map[string]any
// @Failure      400    {object}  map[string]string
// @Failure      502    {object}  map[string]string
// @Router       /wx/taf [get]
func GetTAF(w http.ResponseWriter, r *http.Request) {
	icao := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("icao")))
	if !icaoRe.MatchString(icao) {
		http.Error(w, `{"error":"invalid ICAO"}`, http.StatusBadRequest)
		return
	}
	hours := parseHours(r.URL.Query().Get("hours"), 24)

	body, code, err := fetchAWC(r.Context(), "taf", icao, hours)
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		if len(body) > 0 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"error":"%s","upstream":%q}`, err.Error(), string(body))))
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	var js any
	if e := json.Unmarshal(body, &js); e != nil {
		http.Error(w,
			fmt.Sprintf(`{"error":"invalid upstream json","upstream":%q}`, string(body)),
			http.StatusBadGateway)
		return
	}

	w.WriteHeader(code)
	_, _ = w.Write(body)
}
