// @title           SepTaf API
// @version         0.1
// @description     Airports/Countries/Regions listing
// @BasePath        /
package main

import (
	"SepTaf/internal/config"
	_ "SepTaf/internal/docs"
	httpx "SepTaf/internal/httpx"
	"SepTaf/internal/ingest"
	mdb "SepTaf/internal/mongo"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/robfig/cron/v3"
)

func init() {
	// اول تلاش می‌کنیم .env را لود کنیم؛ اگر نبود هم ادامه می‌دهیم
	_ = godotenv.Load() // .env
	// یا اگر چند فایل داری:
	// _ = godotenv.Load(".env.local", ".env")
}

func main() {

	_ = godotenv.Load()
	fmt.Println("PORT=", os.Getenv("PORT"))
	fmt.Println("MONGO_URI=", os.Getenv("MONGO_URI"))

	cfg := config.Load()
	ctx := context.Background()

	mc, err := mdb.NewClient(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Close(ctx)

	c := cron.New()
	_, err = c.AddFunc(cfg.IngestSchedule, func() {
		if err := ingest.RunAll(ctx, cfg, mc); err != nil {
			log.Printf(`{"lvl":"error","msg":"ingest failed","err":%q}`, err.Error())
		} else {
			log.Printf(`{"lvl":"info","msg":"ingest completed"}`)

		}
	})
	if err != nil {
		log.Fatal(err)
	}
	c.Start()
	defer c.Stop()

	httpx.SetDeps(mc, cfg)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httpx.NewRouter(mc, cfg),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf(`{"msg":"listening","port":%q}`, cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
