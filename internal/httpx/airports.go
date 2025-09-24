package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	_ "SepTaf/internal/docs"
	mdb "SepTaf/internal/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AirportDTO struct {
	Ident        string `json:"ident,omitempty"`
	GPSCode      string `json:"gps_code,omitempty"`
	IATACode     string `json:"iata_code,omitempty"`
	Name         string `json:"name,omitempty"`
	Type         string `json:"type,omitempty"`
	Municipality string `json:"municipality,omitempty"`
	ISOCountry   string `json:"iso_country,omitempty"`
	ISORegion    string `json:"iso_region,omitempty"`
	Location     any    `json:"location,omitempty"` // GeoJSON point
}
type AirportsResponse struct {
	Items []AirportDTO `json:"items"`
	Meta  PageMeta     `json:"meta"`
}

func airportsListHandler(mc *mdb.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
		atype := strings.TrimSpace(r.URL.Query().Get("type"))

		filter := bson.M{}
		var sort bson.D

		if q != "" {
			// جستجوی ترکیبی: کدها یا نام/شهر با regex
			up := strings.ToUpper(q)
			filter["$or"] = []bson.M{
				{"ident": up},
				{"gps_code": up},
				{"iata_code": up},
				{"name": bson.M{"$regex": q, "$options": "i"}},
				{"municipality": bson.M{"$regex": q, "$options": "i"}},
			}
			// اولویت با کدها
			sort = bson.D{{Key: "ident", Value: 1}}
		}
		if country != "" {
			filter["iso_country"] = country
		}
		if atype != "" {
			filter["type"] = atype
		}

		page := getPage(r)
		limit := getLimit(r, 20, 200)
		skip := int64((page - 1)) * limit

		opts := options.Find().
			SetProjection(bson.M{
				"_id":          0,
				"id_csv":       0,
				"continent":    0,
				"elevation_ft": 0,
			}).
			SetSkip(skip).
			SetLimit(limit)

		if len(sort) > 0 {
			opts.SetSort(sort)
		} else {
			opts.SetSort(bson.D{{Key: "name", Value: 1}})
		}

		cur, err := mc.DB.Collection("airports").Find(ctx, filter, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer cur.Close(ctx)

		var items []AirportDTO
		if err := cur.All(ctx, &items); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		total, _ := mc.DB.Collection("airports").CountDocuments(ctx, filter)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items": items,
			"meta":  PageMeta{Page: page, Limit: int(limit), Total: total},
		})
	}
}

// AirportsList godoc
// @Summary      List airports
// @Description  Search & paginate airports
// @Tags         airports
// @Param        q        query   string  false  "free-text (name/municipality) or codes (ICAO/IATA)"
// @Param        country  query   string  false  "ISO country (e.g. US, DE)"
// @Param        type     query   string  false  "large_airport|medium_airport|small_airport|heliport|seaplane_base"
// @Param        page     query   int     false  "page (>=1)"      default(1)
// @Param        limit    query   int     false  "items per page"  default(20)  minimum(1)  maximum(200)
// @Success      200      {object}  AirportsResponse
// @Router       /airportsList [get]
func airportsList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
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

// test godoc
// @Summary      Test Airport Search
// @Description  A simple test endpoint to search and paginate airports by name or municipality.
// @Tags         airports
// @Produce      json
// @Param        q        query   string  false  "Search term for airport name or municipality"
// @Param        page     query   int     false  "Page number for pagination" default(1)
// @Param        limit    query   int     false  "Number of items per page"   default(20)
// @Success      200      {object}  AirportsResponse
// @Failure      500      {object}  map[string]string "Internal Server Error"
// @Router       /test [get]
func Test(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	q := strings.TrimSpace(r.URL.Query().Get("q"))

	filter := bson.M{}
	var sort bson.D

	if q != "" {

		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": q, "$options": "i"}},
			{"municipality": bson.M{"$regex": q, "$options": "i"}},
		}
	}

	page := getPage(r)
	limit := getLimit(r, 20, 200)
	skip := int64(page-1) * limit

	opts := options.Find().SetProjection(bson.M{"_id": 0, "id_csv": 0, "continent": 0, "elevation_ft": 0}).
		SetSkip(skip).SetLimit(limit)
	if len(sort) == 0 {
		sort = bson.D{{Key: "name", Value: 1}}
	}
	opts.SetSort(sort)
	cur, err := depMC.DB.Collection("airports").Find(ctx, filter, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)
	var items []AirportDTO
	if err := cur.All(ctx, &items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	total, _ := depMC.DB.Collection("airports").CountDocuments(ctx, filter)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(AirportsResponse{
		Items: items,
		Meta:  PageMeta{Page: page, Limit: int(limit), Total: total},
	})

}
