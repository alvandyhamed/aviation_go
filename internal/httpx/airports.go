package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	_ "SepTaf/internal/docs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GeoJSONPoint struct {
	Type        string     `bson:"type"        json:"type"`
	Coordinates [2]float64 `bson:"coordinates" json:"coordinates"`
}

type AirportDTO struct {
	Ident        string        `bson:"ident,omitempty"        json:"ident,omitempty"`
	Continent    string        `bson:"continent,omitempty"    json:"continent,omitempty"`
	GPSCode      string        `bson:"gps_code,omitempty"     json:"gps_code,omitempty"` // <-- fix tag
	IATACode     string        `bson:"iata_code,omitempty"    json:"iata_code,omitempty"`
	IcaoCode     string        `bson:"icao_code,omitempty"    json:"icao_code,omitempty"`
	Name         string        `bson:"name,omitempty"         json:"name,omitempty"`
	Type         string        `bson:"type,omitempty"         json:"type,omitempty"`
	Municipality string        `bson:"municipality,omitempty" json:"municipality,omitempty"`
	ISOCountry   string        `bson:"iso_country,omitempty"  json:"iso_country,omitempty"`
	ISORegion    string        `bson:"iso_region,omitempty"   json:"iso_region,omitempty"`
	Location     *GeoJSONPoint `bson:"location,omitempty"     json:"location,omitempty"`
	// اگر elevation هم می‌خواهید:
	// ElevationFT  int          `bson:"elevation_ft,omitempty" json:"elevation_ft,omitempty"`
}
type AirportsResponse struct {
	Items []AirportDTO `json:"items"`
	Meta  PageMeta     `json:"meta"`
}

// AirportsList godoc
// @Summary      List airports
// @Description  Search & paginate airports
// @Tags         airports
// @Param        q        query   string  false  "free-text (name/municipality) or codes (ICAO/IATA)"
// @Param 		 ICAO     query   string  false   "Find ICAO"
// @Param        IATA     query   string  false    "Find IATA"
// @Param        country  query   string  false  "ISO country (e.g. US, DE)"
// @Param        type     query   string  false  "large_airport|medium_airport|small_airport|heliport|seaplane_base"
// @Param        page     query   int     false  "page (>=1)"      default(1)
// @Param        limit    query   int     false  "items per page"  default(20)  minimum(1)  maximum(200)
/*Headers Params*/
// @Param        X-Client-Id     header  string  true   "Client ID (e.g., client-42)"
// @Param        X-Key-Version   header  string  true   "Key version (e.g., v1)"
// @Param        X-Date          header  string  true   "Request time (RFC3339 or epoch seconds)"
// @Param        X-Nonce         header  string  true   "Random nonce (UUID/base64)"
// @Param        X-Signature     header  string  false   "Base64(HMAC-SHA256(canonical, secret_vN))"
// @Security     ClientIDAuth
// @Security     KeyVersionAuth
// @Security     DateAuth
// @Security     NonceAuth
// @Security     SignatureAuth
// @Success      200      {object}  AirportsResponse
// @Failure      400  {object}  HTTPError
// @Failure      500  {object}  HTTPError
// @Router       /airports_list [get]
func airportsList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	icao := strings.TrimSpace(r.URL.Query().Get("ICAO"))
	iata := strings.TrimSpace(r.URL.Query().Get("IATA"))
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	atype := strings.TrimSpace(r.URL.Query().Get("type"))

	filter := bson.M{}
	var sort bson.D

	if q != "" {
		up := strings.ToUpper(q)
		filter["$or"] = []bson.M{
			{"ident": up},
			{"gps_code": up},
			{"iata_code": up},
			{"name": bson.M{"$regex": q, "$options": "i"}},
			{"municipality": bson.M{"$regex": q, "$options": "i"}},
		}
		sort = bson.D{{Key: "ident", Value: 1}}
	}
	if country != "" {
		filter["iso_country"] = country
	}
	if icao != "" {
		filter["icao_code"] = icao
	}
	if iata != "" {
		filter["iata_code"] = iata
	}
	if atype != "" {
		filter["type"] = atype
	}

	page := getPage(r)
	limit := getLimit(r, 20, 200)
	skip := int64(page-1) * limit

	opts := options.Find().
		SetProjection(bson.M{"_id": 0, "id_csv": 0, "continent": 0, "elevation_ft": 0}).
		SetSkip(skip).SetLimit(limit)
	if len(sort) == 0 {
		sort = bson.D{{Key: "name", Value: 1}}
	}
	opts.SetSort(sort)

	cur, err := depMC.DB.Collection("airports").Find(ctx, filter, opts)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer cur.Close(ctx)

	var items []AirportDTO
	if err := cur.All(ctx, &items); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	total, _ := depMC.DB.Collection("airports").CountDocuments(ctx, filter)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AirportsResponse{
		Items: items,
		Meta:  PageMeta{Page: page, Limit: int(limit), Total: total},
	})
}
