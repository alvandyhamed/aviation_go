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
//
// Security (HMAC strict mode):
// برای هر درخواست محافظت‌شده، این هدرها لازم‌اند:
//   - X-Client-Id
//   - X-Key-Version
//   - X-Date
//   - X-Nonce
//   - X-Signature
//
// @securityDefinitions.apikey  ClientIDAuth
// @in                          header
// @name                        X-Client-Id
//
// @securityDefinitions.apikey  KeyVersionAuth
// @in                          header
// @name                        X-Key-Version
//
// @securityDefinitions.apikey  DateAuth
// @in                          header
// @name                        X-Date
//
// @securityDefinitions.apikey  NonceAuth
// @in                          header
// @name                        X-Nonce
//
// @securityDefinitions.apikey  SignatureAuth
// @in                          header
// @name                        X-Signature
//
// Legacy/Permissive (اختیاری):
// اگر AuthStrictMode=false باشد، می‌توانید به‌جای امضا از X-Client-Secret استفاده کنید.
//
// @securityDefinitions.apikey  ClientSecretAuth
// @in                          header
// @name                        X-Client-Secret
func NewRouter(mc *mdb.Client, cfg config.Config) http.Handler {
	SetDeps(mc, cfg)
	public := http.NewServeMux()
	public.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
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
	public.HandleFunc("/countries_find", findacountries)
	public.Handle("/swagger/", httpSwagger.WrapHandler)
	public.HandleFunc("/swagger/doc.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		doc := docs.SwaggerInfo.ReadDoc()
		_, _ = w.Write([]byte(doc))
	})
	protected := http.NewServeMux()

	protected.HandleFunc("/airportsList", airportsList)
	protected.HandleFunc("/regions", regionsListHandler(mc)) // GET ?q=&country=&page=&limit=

	protected.HandleFunc("/firList", firList)

	//Proxy
	protected.HandleFunc("/wx/metar", http.HandlerFunc(GetMETAR))
	protected.HandleFunc("/wx/taf", http.HandlerFunc(GetTAF))
	//rl := NewRateLimiter(29)
	//protected.Handle("/faa/notams", LimitMiddleware(rl, http.HandlerFunc(GetNOTAM)))
	protected.HandleFunc("/faa/notams", http.HandlerFunc(GetNOTAM))

	auth := NewAuthMiddleware(cfg, mc)
	root := http.NewServeMux()
	root.Handle("/", public) // آزاد
	root.Handle("/countries_find", auth.Handler(protected))
	root.Handle("/airportsList", auth.Handler(protected))
	root.Handle("/regions", auth.Handler(protected))
	root.Handle("/firList", auth.Handler(protected))
	root.Handle("/wx/", auth.Handler(protected))
	root.Handle("/faa/", auth.Handler(protected))

	return root
}
