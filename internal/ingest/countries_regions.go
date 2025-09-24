package ingest

import (
	"context"
	"encoding/csv"
	"log"
	"os"
	"time"

	mdb "SepTaf/internal/mongo"
)

// ─── Countries ────────────────────────────────────────────────────────────────
func ParseCountriesStreamAndUpsert(ctx context.Context, path string, mc *mdb.Client) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return err
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}

	get := func(row []string, k string) string {
		if p, ok := idx[k]; ok && p < len(row) {
			return row[p]
		}
		return ""
	}

	batch := make([]mdb.CountryDoc, 0, 1000)
	rows := 0
	last := time.Now()

	for {
		row, err := r.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		rows++

		doc := mdb.CountryDoc{
			Code:      get(row, "code"),
			Name:      get(row, "name"),
			Continent: get(row, "continent"),
			Keywords:  get(row, "keywords"),
		}
		if doc.Code == "" {
			continue
		}

		batch = append(batch, doc)
		if len(batch) >= 1000 {
			if err := mc.BulkUpsertCountries(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}

		if time.Since(last) > 3*time.Second {
			log.Printf(`{"msg":"countries-progress","rows":%d}`, rows)
			last = time.Now()
		}
	}
	if len(batch) > 0 {
		if err := mc.BulkUpsertCountries(ctx, batch); err != nil {
			return err
		}
	}
	log.Printf(`{"msg":"countries-upsert-done","rows":%d}`, rows)
	return nil
}

// ─── Regions ────────────────────────────────────────────────────────────────
func ParseRegionsStreamAndUpsert(ctx context.Context, path string, mc *mdb.Client) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return err
	}

	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}

	get := func(row []string, k string) string {
		if p, ok := idx[k]; ok && p < len(row) {
			return row[p]
		}
		return ""
	}

	batch := make([]mdb.RegionDoc, 0, 1000)
	rows := 0
	last := time.Now()

	for {
		row, err := r.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		rows++

		doc := mdb.RegionDoc{
			Code:       get(row, "code"),       // مثل US-CA
			LocalCode:  get(row, "local_code"), // مثل CA
			Name:       get(row, "name"),
			ISOCountry: get(row, "iso_country"),
			Continent:  get(row, "continent"),
		}
		if doc.Code == "" {
			continue
		}

		batch = append(batch, doc)
		if len(batch) >= 1000 {
			if err := mc.BulkUpsertRegions(ctx, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}

		if time.Since(last) > 3*time.Second {
			log.Printf(`{"msg":"regions-progress","rows":%d}`, rows)
			last = time.Now()
		}
	}
	if len(batch) > 0 {
		if err := mc.BulkUpsertRegions(ctx, batch); err != nil {
			return err
		}
	}
	log.Printf(`{"msg":"regions-upsert-done","rows":%d}`, rows)
	return nil
}
