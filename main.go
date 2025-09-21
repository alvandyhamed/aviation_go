package main

import (
	"context"
	"log"

	"SepTaf/internal/config"
	"SepTaf/internal/ingest"
	mdb "SepTaf/internal/mongo"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	mc, err := mdb.NewClient(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatal(err)
	}
	defer mc.Close(ctx)

	if err := ingest.RunAll(ctx, cfg, mc); err != nil {
		log.Fatal(err)
	}
	log.Println("ingest done")
}
