package main

import (
	"SepTaf/internal/config"
	httpx "SepTaf/internal/http"
	"SepTaf/internal/ingest"
	mdb "SepTaf/internal/mongo"
	"context"
	"github.com/robfig/cron/v3"
	"log"
	"net/http"
	"time"
)

func main() {

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

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      httpx.NewRouter(mc),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Printf(`{"msg":"listening","port":%q}`, cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

}
