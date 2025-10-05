package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	mdb "SepTaf/internal/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// -------- Countries --------
type CountryDTO struct {
	Code      string `json:"code"`
	Name      string `json:"name,omitempty"`
	Continent string `json:"continent,omitempty"`
	Keywords  string `json:"keywords,omitempty"`
}
type CountriesResponse struct {
	Items []CountryDTO `json:"items"`
	Meta  PageMeta     `json:"meta"`
}

// -------- Regions --------
type RegionDTO struct {
	Code       string `json:"code"`
	LocalCode  string `json:"local_code,omitempty"`
	Name       string `json:"name,omitempty"`
	ISOCountry string `json:"iso_country,omitempty"`
	Continent  string `json:"continent,omitempty"`
}
type RegionsResponse struct {
	Items []RegionDTO `json:"items"`
	Meta  PageMeta    `json:"meta"`
}

// RegionsList godoc
// @Summary     List regions
// @Tags        geo
// @Param       q        query   string  false  "code/local_code/name"
// @Param       country  query   string  false  "ISO country (e.g. US)"
// @Param       page     query   int     false  "page"  default(1)
// @Param       limit    query   int     false  "limit" default(50) minimum(1) maximum(500)
/*Headers Params*/
// @Param        X-Client-Id     header  string  true   "Client ID (e.g., client-42)"
// @Param        X-Key-Version   header  string  true   "Key version (e.g., v1)"
// @Param        X-Date          header  string  true   "Request time (RFC3339 or epoch seconds)"
// @Param        X-Nonce         header  string  true   "Random nonce (UUID/base64)"
// @Param        X-Signature     header  string  true   "Base64(HMAC-SHA256(canonical, secret_vN))"
// @Security     ClientIDAuth
// @Security     KeyVersionAuth
// @Security     DateAuth
// @Security     NonceAuth
// @Security     SignatureAuth
// @Success     200      {object}  RegionsResponse
// @Failure      400  {object}  HTTPError
// @Failure      500  {object}  HTTPError
// @Router      /regions [get]
func regionsListHandler(mc *mdb.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		q := strings.TrimSpace(r.URL.Query().Get("q"))
		country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
		filter := bson.M{}
		if q != "" {
			up := strings.ToUpper(q)
			filter["$or"] = []bson.M{
				{"code": up},
				{"local_code": up},
				{"name": bson.M{"$regex": q, "$options": "i"}},
			}
		}
		if country != "" {
			filter["iso_country"] = country
		}

		page := getPage(r)
		limit := getLimit(r, 50, 500)
		skip := int64(page-1) * limit

		opts := options.Find().
			SetProjection(bson.M{"_id": 0}).
			SetSort(bson.D{{Key: "name", Value: 1}}).
			SetSkip(skip).
			SetLimit(limit)

		cur, err := mc.DB.Collection("regions").Find(ctx, filter, opts)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer cur.Close(ctx)

		var items []RegionDTO
		if err := cur.All(ctx, &items); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		total, _ := mc.DB.Collection("regions").CountDocuments(ctx, filter)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items": items,
			"meta":  PageMeta{Page: page, Limit: int(limit), Total: total},
		})
	}
}

// Find Countries godoc
// @Summary      Search Countries Code
// @Description  Search Countries Code
// @Tags         Countries
// @Produce      json
// @Param        q     query  string  false  "Search term"
// @Param        page  query  int     false  "Page number"       default(1)
// @Param        limit query  int     false  "Items per page"    default(20)
/*Headers Params*/
// @Param        X-Client-Id     header  string  true   "Client ID (e.g., client-42)"
// @Param        X-Key-Version   header  string  true   "Key version (e.g., v1)"
// @Param        X-Date          header  string  true   "Request time (RFC3339 or epoch seconds)"
// @Param        X-Nonce         header  string  true   "Random nonce (UUID/base64)"
// @Param        X-Signature     header  string  true   "Base64(HMAC-SHA256(canonical, secret_vN))"
// @Security     ClientIDAuth
// @Security     KeyVersionAuth
// @Security     DateAuth
// @Security     NonceAuth
// @Security     SignatureAuth
// @Success      200   {object}  CountriesResponse
// @Failure      400  {object}  HTTPError
// @Failure      500  {object}  HTTPError
// @Router       /countries_find [get]
func findacountries(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page := getPage(r)
	if page < 1 {
		page = 1
	}
	limit := getLimit(r, 20, 200)
	if limit <= 0 {
		limit = 20
	}
	skip := int64(page-1) * limit

	// فیلتر
	filter := bson.M{}
	if q != "" {

		pattern := regexp.QuoteMeta(q)

		filter["$or"] = []bson.M{
			{"keywords": bson.M{"$regex": pattern, "$options": "i"}},
			{"name": bson.M{"$regex": pattern, "$options": "i"}},
			{"name": bson.M{"$regex": pattern, "$options": "i"}},
		}
	}

	sort := bson.D{{Key: "name", Value: 1}}
	opts := options.Find().
		SetProjection(bson.M{
			"_id":          0,
			"id_csv":       0,
			"continent":    0,
			"elevation_ft": 0,
		}).
		SetSkip(skip).
		SetLimit(limit).
		SetSort(sort)

	cur, err := depMC.DB.Collection("countries").Find(ctx, filter, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var items []CountryDTO
	if err := cur.All(ctx, &items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	total, err := depMC.DB.Collection("countries").CountDocuments(ctx, filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// گزینه ۱: struct واضح
	resp := struct {
		Items []CountryDTO `json:"items"`
		Meta  PageMeta     `json:"meta"`
	}{
		Items: items,
		Meta:  PageMeta{Page: page, Limit: int(limit), Total: total},
	}
	_ = json.NewEncoder(w).Encode(resp)

	// گزینه ۲ (جایگزین): اگر map می‌خوای
	// _ = json.NewEncoder(w).Encode(map[string]interface{}{
	// 	"items": items,
	// 	"meta":  PageMeta{Page: page, Limit: int(limit), Total: total},
	// })
}
