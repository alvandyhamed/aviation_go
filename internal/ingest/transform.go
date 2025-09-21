package ingest

import (
	"context"
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"time"

	mdb "SepTaf/internal/mongo"
)

func ParseAirportsStreamAndUpsert(ctx context.Context, path string, mc *mdb.Client) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	// header
	header, err := r.Read()
	if err != nil {
		return err
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[h] = i
	}

	batch := make([]mdb.AirportDoc, 0, 1000)
	rowNum := 0
	lastLog := time.Now()

	for {
		row, err := r.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		rowNum++

		get := func(k string) string {
			if p, ok := idx[k]; ok && p < len(row) {
				return row[p]
			}
			return ""
		}

		var idCSV *int
		if v := get("id"); v != "" {
			if n, e := strconv.Atoi(v); e == nil {
				idCSV = &n
			}
		}
		var elev *int
		if v := get("elevation_ft"); v != "" {
			if n, e := strconv.Atoi(v); e == nil {
				elev = &n
			}
		}
		lat, _ := strconv.ParseFloat(get("latitude_deg"), 64)
		lon, _ := strconv.ParseFloat(get("longitude_deg"), 64)

		doc := mdb.AirportDoc{
			IDCSV:        idCSV,
			Ident:        get("ident"),
			GPSCode:      get("gps_code"),
			IATACode:     get("iata_code"),
			Name:         get("name"),
			Type:         get("type"),
			Municipality: get("municipality"),
			ISOCountry:   get("iso_country"),
			ISORegion:    get("iso_region"),
			ElevationFt:  elev,
			Continent:    get("continent"),
		}
		if lat != 0 || lon != 0 {
			doc.Location = map[string]any{
				"type":        "Point",
				"coordinates": []float64{lon, lat},
			}
		}

		batch = append(batch, doc)
		if len(batch) >= 1000 {
			if err := mc.BulkUpsertAirports(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}

		// لاگ پیشرفت هر ~5 ثانیه
		if time.Since(lastLog) > 5*time.Second {
			log.Printf(`{"msg":"airports-progress","rows":%d}`, rowNum)
			lastLog = time.Now()
		}
	}
	if len(batch) > 0 {
		if err := mc.BulkUpsertAirports(ctx, batch); err != nil {
			return err
		}
	}
	log.Printf(`{"msg":"airports-upsert-done","rows":%d}`, rowNum)
	return nil
}
