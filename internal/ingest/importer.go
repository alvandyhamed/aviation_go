package ingest

import (
	"SepTaf/internal/config"
	mdb "SepTaf/internal/mongo"
	"context"
)

func RunAll(ctx context.Context, cfg config.Config, mc *mdb.Client) error {
	// === Airports ===
	//apFile, err := downloadToTemp(cfg.URLAirports)
	//if err != nil {
	//	return err
	//}
	//defer os.Remove(apFile)
	//
	//if err := mc.EnsureAirportIndexes(ctx); err != nil {
	//	return err
	//}
	//if err := ParseAirportsStreamAndUpsert(ctx, apFile, mc); err != nil {
	//	return err
	//}

	// === Countries ===
	//ctFile, err := downloadToTemp(cfg.URLCountries)
	//if err != nil {
	//	return err
	//}
	//defer os.Remove(ctFile)
	//
	//if err := mc.EnsureCountriesIndexes(ctx); err != nil {
	//	return err
	//}
	//if err := ParseCountriesStreamAndUpsert(ctx, ctFile, mc); err != nil {
	//	return err
	//}

	//=== Regions ===
	//rgFile, err := downloadToTemp(cfg.URLRegions)
	//if err != nil {
	//	return err
	//}
	//defer os.Remove(rgFile)
	//
	//if err := mc.EnsureRegionsIndexes(ctx); err != nil {
	//	return err
	//}
	//
	//if err := ParseRegionsStreamAndUpsert(ctx, rgFile, mc); err != nil {
	//	return err
	//}
	//// === FIRs ===  👇 بخش جدید
	//if cfg.URLFIRs != "" {
	//	firFile, err := downloadToTemp(cfg.URLFIRs)
	//	if err != nil {
	//		return err
	//	}
	//	defer os.Remove(firFile)
	//
	//	if err := mc.EnsureFIRIndexes(ctx); err != nil {
	//		return err
	//	}
	//	if err := ParseFIRsStreamAndUpsert(ctx, firFile, mc); err != nil {
	//		return err
	//	}
	//}
	// FIRs (بدون پکیج اضافی)
	// === FIR (Country ↔ FIR) از ویکی‌پدیا
	if err := ParseWikipediaFIRsAndUpsert(ctx, mc); err != nil {
		return err
	}
	return nil
}
