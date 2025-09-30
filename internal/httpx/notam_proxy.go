package httpx

import (
	"SepTaf/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const faaBase = "https://external-api.faa.gov/notamapi/v1/notams"

type GeoMetry struct {
	Type string `json:"type"`
}
type NotamEventType struct {
	Scenario string `json:"type"`
}
type NotamType struct {
	Id             string `json:"id"`
	Series         string `json:"series"`
	Number         string `json:"number"`
	Type           string `json:"type"`
	Issued         string `json:"issued"`
	AffectedFIR    string `json:"affectedFIR"`
	SelectionCode  string `json:"selectionCode"`
	Traffic        string `json:"traffic"`
	Purpose        string `json:"purpose"`
	Scope          string `json:"scope"`
	MinimumFL      string `json:"minimumFL"`
	MaximumFL      string `json:"maximumFL"`
	Location       string `json:"location"`
	EffectiveStart string `json:"effectiveStart"`
	EffectiveEnd   string `json:"effectiveEnd"`
	Text           string `json:"text"`
	Classification string `json:"classification"`
	AccountId      string `json:"accountId"`
	LastUpdated    string `json:"lastUpdated"`
	IcaoLocation   string `json:"icaoLocation"`
	Schedule       string `json:"schedule"`
	LowerLimit     string `json:"lowerLimit"`
	UpperLimit     string `json:"upperLimit"`
}
type ArrayOfNotamTranslationType struct {
	Type          string `json:"type"`
	FormattedText string `json:"formattedText"`
}
type NotamTranslationType struct {
	Items []ArrayOfNotamTranslationType `json:"items"`
}

type CoreNOTAMDataType struct {
	NotamEvent       NotamEventType       `json:"notam_event"`
	Notam            NotamType            `json:"notam"`
	NotamTranslation NotamTranslationType `json:"notam_translation"`
}
type PropertiesType struct {
	CoreNOTAMData CoreNOTAMDataType `json:"coreNOTAMData"`
}

type NotamFeature struct {
	Type       string           `json:"type"`               // "Point" | ...
	Geometry   []GeoMetry       `json:"geometry,omitempty"` // GeoJSON-like
	Properties []PropertiesType `json:"properties,omitempty"`
}

type NotamResponse struct {
	PageSize   int            `json:"pageSize"`
	PageNum    int            `json:"pageNum"`
	TotalCount int            `json:"totalCount"`
	TotalPages int            `json:"totalPages"`
	Items      []NotamFeature `json:"items"`
}

// ENV/Config خواندن کلیدها
func getFAAKeys() (id, secret string) {
	cfg := config.Load()

	return cfg.FAACLIENTID, cfg.FAACLIENTSECRET
}

// enum helpers
func inSet(v string, set ...string) bool {
	if v == "" {
		return true
	} // اختیاری
	for _, s := range set {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
func clampInt(v string, def, min, max int) int {
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if i < min {
		return min
	}
	if i > max {
		return max
	}
	return i
}

// Build query allowed params + validate
func buildFAAQuery(r *http.Request) (url.Values, error) {
	q := url.Values{}

	// Required/optional params
	domesticLocation := strings.TrimSpace(r.URL.Query().Get("domesticLocation"))
	if domesticLocation != "" {
		q.Set("domesticLocation", domesticLocation)
	}

	// Enums
	notamType := strings.TrimSpace(r.URL.Query().Get("notamType")) // N,R,C
	if !inSet(notamType, "N", "R", "C", "") {
		return nil, fmt.Errorf("invalid notamType")
	}
	if notamType != "" {
		q.Set("notamType", strings.ToUpper(notamType))
	}

	classification := strings.TrimSpace(r.URL.Query().Get("classification")) // INTL,MIL,DOM,LMIL,FDC
	if !inSet(classification, "INTL", "MIL", "DOM", "LMIL", "FDC", "") {
		return nil, fmt.Errorf("invalid classification")
	}
	if classification != "" {
		q.Set("classification", strings.ToUpper(classification))
	}

	notamNumber := strings.TrimSpace(r.URL.Query().Get("notamNumber"))
	if notamNumber != "" {
		q.Set("notamNumber", notamNumber)
	}

	effStart := strings.TrimSpace(r.URL.Query().Get("effectiveStartDate"))
	if effStart != "" {
		q.Set("effectiveStartDate", effStart)
	}

	effEnd := strings.TrimSpace(r.URL.Query().Get("effectiveEndDate"))
	if effEnd != "" {
		q.Set("effectiveEndDate", effEnd)
	}

	featureType := strings.TrimSpace(r.URL.Query().Get("featureType"))
	if !inSet(featureType,
		"RWY", "TWY", "APRON", "AD", "OBST", "NAV", "COM", "SVC", "AIRSPACE",
		"ODP", "SID", "STAR", "CHART", "DATA", "DVA", "IAP", "VFP", "ROUTE",
		"SPECIAL", "SECURITY", "MILITARY", "INTERNATIONAL", "") {
		return nil, fmt.Errorf("invalid featureType")
	}
	if featureType != "" {
		q.Set("featureType", strings.ToUpper(featureType))
	}

	sortBy := strings.TrimSpace(r.URL.Query().Get("sortBy"))
	if !inSet(sortBy, "icaoLocation", "domesticLocation", "notamType", "notamNumber", "effectiveStartDate", "effectiveEndDate", "featureType", "") {
		return nil, fmt.Errorf("invalid sortBy")
	}
	if sortBy != "" {
		q.Set("sortBy", sortBy)
	}

	sortOrder := strings.TrimSpace(r.URL.Query().Get("sortOrder"))
	if !inSet(sortOrder, "Asc", "Desc", "") {
		return nil, fmt.Errorf("invalid sortOrder")
	}
	if sortOrder != "" {
		q.Set("sortOrder", sortOrder)
	}

	pageSize := clampInt(r.URL.Query().Get("pageSize"), 50, 1, 1000)
	q.Set("pageSize", strconv.Itoa(pageSize))
	pageNum := clampInt(r.URL.Query().Get("pageNum"), 1, 1, 1_000_000)
	q.Set("pageNum", strconv.Itoa(pageNum))

	return q, nil
}

// ---------- Handler ----------

// GetNOTAM godoc
// @Summary      FAA NOTAM proxy (rate-limited 29/min)
// @Description  Pass-through to FAA NOTAM API with input validation & global rate limit.
// @Tags         NOTAM
// @Produce      json
// @Param        domesticLocation  query  string  false  "Domestic/FIR/ICAO location (e.g., OIIX)"
// @Param        notamType         query  string  false  "N | R | C"
// @Param        classification    query  string  false  "INTL | MIL | DOM | LMIL | FDC"
// @Param        notamNumber       query  string  false  "e.g., CK0000/01"
// @Param        effectiveStartDate query string false   "ISO date/time"
// @Param        effectiveEndDate   query string false   "ISO date/time"
// @Param        featureType       query  string  false  "RWY,TWY,APRON,AD,OBST,NAV,COM,SVC,AIRSPACE,ODP,SID,STAR,CHART,DATA,DVA,IAP,VFP,ROUTE,SPECIAL,SECURITY,MILITARY,INTERNATIONAL"
// @Param        sortBy            query  string  false  "icaoLocation,domesticLocation,notamType,notamNumber,effectiveStartDate,effectiveEndDate,featureType"
// @Param        sortOrder         query  string  false  "Asc | Desc"
// @Param        pageSize          query  int     false  "Default 50 (max 1000)"
// @Param        pageNum           query  int     false  "Default 1"
// @Success      200  {object}  httpx.NotamResponse
// @Failure      400  {object}  httpx.HTTPError
// @Failure      401  {object}  httpx.HTTPError
// @Failure      502  {object}  httpx.HTTPError
// @Router       /faa/notams [get]
func GetNOTAM(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// validate query
	q, err := buildFAAQuery(r)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
		return
	}

	// credentials
	id, secret := getFAAKeys()
	if id == "" || secret == "" {
		http.Error(w, `{"error":"missing FAA client_id/client_secret"}`, http.StatusUnauthorized)
		return
	}

	// build upstream request
	u := fmt.Sprintf("%s?%s", faaBase, q.Encode())
	req, _ := http.NewRequestWithContext(r.Context(), "GET", u, nil)
	req.Header.Set("client_id", id)
	req.Header.Set("client_secret", secret)
	req.Header.Set("User-Agent", "SepTaf-NOTAM/1.0 (contact: you@example.com)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf(`{"error":"upstream %d","upstream":%q}`, resp.StatusCode, string(body)), http.StatusBadGateway)
		return
	}

	// sanity JSON + marshal به مدل برای Swagger (optional)
	var out NotamResponse
	if err := json.Unmarshal(body, &out); err != nil {
		// اگر اسکیمای FAA تغییر کرد، پاس‌ترو خام بده ولی 200 نگه دار
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}
