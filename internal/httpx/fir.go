package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type FIR struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"    json:"id"`
	Country string             `bson:"country,omitempty" json:"country,omitempty"`
	FirName string             `bson:"fir_name,omitempty" json:"fir_name,omitempty"`
	FirCode string             `bson:"fir_code,omitempty" json:"fir_code,omitempty"`
}

type FirResponse struct {
	Items []FIR    `json:"items"`
	Meta  PageMeta `json:"meta"`
}
type FIRSimpleDTO struct {
	CountryName string `json:"country_name"`
	FIRCode     string `json:"fir_code"`
	FIRName     string `json:"fir_name"`
}
type HTTPError struct {
	Message string `json:"message"`
}

// FIR LIST godoc
// @Summary      List of FIR
// @Description  Search FIRs by country, name, or code
// @Tags         Firs
// @Produce      json
// @Param        country   query   string  false  "Find FIRs for country (name or ISO code)"
// @Param        fir_name  query   string  false  "Find by FIR name (e.g., Tehran)"
// @Param        fir_code  query   string  false  "Find by FIR ICAO code (e.g., OIIX)"
// @Success      200  {object}  FirResponse
// @Failure      400  {object}  HTTPError
// @Failure      500  {object}  HTTPError
// @Router       /firList  [get]
func firList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	country := strings.TrimSpace(r.URL.Query().Get("country"))
	firName := strings.TrimSpace(r.URL.Query().Get("fir_name"))
	firCode := strings.TrimSpace(r.URL.Query().Get("fir_code"))

	filter := bson.M{}

	if country != "" {
		filter["country"] = country
	}
	if firName != "" {
		filter["fir_name"] = bson.M{"$regex": regexp.QuoteMeta(firName), "$options": "i"}
	}
	if firCode != "" {
		filter["fir_code"] = strings.ToUpper(firCode)
	}

	page := getPage(r)
	limit := getLimit(r, 20, 200)
	skip := int64(page-1) * limit

	opts := options.Find().
		SetSkip(skip).
		SetLimit(limit).
		SetProjection(bson.M{"_id": 0}).
		SetSort(bson.D{{Key: "fir_name", Value: 1}})

	cur, err := depMC.DB.Collection("firs").Find(ctx, filter, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	var items []FIR
	if err := cur.All(ctx, &items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	total, _ := depMC.DB.Collection("firs").CountDocuments(ctx, filter)

	w.Header().Set("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(FirResponse{
		Items: items,
		Meta:  PageMeta{Page: page, Limit: int(limit), Total: total},
	})

}
