package httpx

import (
	"SepTaf/internal/config"
	docs "SepTaf/internal/docs"
	mdb "SepTaf/internal/mongo"
	"encoding/json"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"
	"time"
)

// @title           SepTaf API
// @version         1.0
// @description     This is the API documentation for the SepTaf application.
// @host            localhost:8086
// @BasePath        /
func NewRouter(mc *mdb.Client, cfg config.Config) http.Handler {
	SetDeps(mc, cfg)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		type resp struct {
			Status string    `json:"status"`
			Time   time.Time `json:"time"`
			DB     string    `json:"db"`
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp{
			Status: "ok",
			Time:   time.Now().UTC(),
			DB:     mc.DB.Name(),
		})
	})
	mux.HandleFunc("/countries_find", findacountries)
	mux.HandleFunc("/airportsList", airportsList)
	mux.HandleFunc("/regions", regionsListHandler(mc)) // GET ?q=&country=&page=&limit=

	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	mux.HandleFunc("/firList", firList)

	//Proxy
	mux.HandleFunc("/wx/metar", http.HandlerFunc(GetMETAR))
	mux.HandleFunc("/wx/taf", http.HandlerFunc(GetTAF))
	mux.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		doc := docs.SwaggerInfo.ReadDoc()
		_, _ = w.Write([]byte(doc))
	})

	return mux
}
