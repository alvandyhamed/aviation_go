package http

import (
	mdb "SepTaf/internal/mongo"
	"encoding/json"
	"net/http"
	"time"
)

func NewRouter(mc *mdb.Client) http.Handler {
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
	return mux
}
