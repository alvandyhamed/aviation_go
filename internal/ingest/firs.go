package ingest

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	mdb "SepTaf/internal/mongo"
)

// ساختارهای سبک برای GeoJSON
type featureCollection struct {
	Type     string            `json:"type"`
	Features []geoFeatureLight `json:"features"`
}

type geoFeatureLight struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Geometry   map[string]any `json:"geometry"`
}

// استخراج مقدار رشته با چند کلید ممکن
func strProp(props map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := props[k]; ok {
			if s, ok2 := v.(string); ok2 {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// این تابع ورودی هم FeatureCollection کامل را پشتیبانی می‌کند، هم NDJSON (هر خط یک Feature)
func ParseFIRsStreamAndUpsert(ctx context.Context, path string, mc *mdb.Client) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	// تلاش برای تشخیص نوع فایل: اگر اولین توکن '{' و "type":"FeatureCollection" بود، کل را به‌عنوان FC می‌خوانیم.
	// در غیر اینصورت، فایل را خط‌به‌خط (NDJSON) می‌خوانیم.
	// برای این کار باید یک peek کوچک داشته باشیم:
	headBuf := make([]byte, 4096)
	n, _ := f.Read(headBuf)
	head := strings.TrimSpace(string(headBuf[:n]))
	// به ابتدای فایل برگردیم
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}

	batch := make([]mdb.FIRDoc, 0, 500)
	commit := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := mc.BulkUpsertFIRs(ctx, batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	pushFeature := func(ft geoFeatureLight) error {
		if ft.Properties == nil {
			return nil
		}
		typ := strings.ToUpper(strProp(ft.Properties, "type", "airspaceType"))
		if typ != "FIR" {
			return nil
		}

		country := strProp(ft.Properties, "country", "Country", "iso_country", "ISO_COUNTRY")
		name := strProp(ft.Properties, "name", "Name", "fir_name", "FIR_NAME")
		if country == "" || name == "" {
			// اگر کشور/نام نداریم، بی‌خیال این فیچر شو
			return nil
		}

		// تلاش برای یافتن کد FIR (ممکن است در منابع مختلف کلیدهای مختلفی داشته باشد)
		code := strProp(ft.Properties, "icao_code", "code", "fir_code", "ICAO_CODE", "FIR_CODE")

		doc := mdb.FIRDoc{
			Country:  country,
			FIRName:  name,
			FIRCode:  code,
			Geometry: ft.Geometry, // همان GeoJSON نگه می‌داریم (bson سازگار است)
			Source:   "openAIP",
		}

		batch = append(batch, doc)
		if len(batch) >= 500 {
			return commit()
		}
		return nil
	}

	// تشخیص مود
	if strings.Contains(head, `"FeatureCollection"`) {
		var fc featureCollection
		if err := dec.Decode(&fc); err != nil {
			return err
		}
		if strings.ToLower(fc.Type) != "featurecollection" {
			return errors.New("not a FeatureCollection")
		}
		for _, ft := range fc.Features {
			if err := pushFeature(ft); err != nil {
				return err
			}
		}
		if err := commit(); err != nil {
			return err
		}
		return nil
	}

	// NDJSON (هر خط یک Feature)
	if _, err := f.Seek(0, 0); err != nil {
		return err
	}
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			var ft geoFeatureLight
			if e := json.Unmarshal(line, &ft); e == nil {
				if pushErr := pushFeature(ft); pushErr != nil {
					return pushErr
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read ndjson: %w", err)
		}
	}
	if err := commit(); err != nil {
		return err
	}
	return nil
}
