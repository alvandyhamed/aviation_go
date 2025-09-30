package main

import (
	"context"
	"github.com/joho/godotenv"
	"log"

	"SepTaf/internal/config"
	"SepTaf/internal/ingest"
	mdb "SepTaf/internal/mongo"
)

func init() {
	// اول تلاش می‌کنیم .env را لود کنیم؛ اگر نبود هم ادامه می‌دهیم
	_ = godotenv.Load() // .env
	// یا اگر چند فایل داری:
	// _ = godotenv.Load(".env.local", ".env")
}

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
