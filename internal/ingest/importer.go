package ingest

import (
	"context"
	"os"

	"SepTaf/internal/config"
	mdb "SepTaf/internal/mongo"
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
	ctFile, err := downloadToTemp(cfg.URLCountries)
	if err != nil {
		return err
	}
	defer os.Remove(ctFile)

	if err := mc.EnsureCountriesIndexes(ctx); err != nil {
		return err
	}
	if err := ParseCountriesStreamAndUpsert(ctx, ctFile, mc); err != nil {
		return err
	}

	// === Regions ===
	//rgFile, err := downloadToTemp(cfg.URLRegions)
	//if err != nil {
	//	return err
	//}
	//defer os.Remove(rgFile)
	//
	//if err := mc.EnsureRegionsIndexes(ctx); err != nil {
	//	return err
	//}
	//// این خط را اضافه کن 👇
	//if err := ParseRegionsStreamAndUpsert(ctx, rgFile, mc); err != nil {
	//	return err
	//}

	return nil
}
