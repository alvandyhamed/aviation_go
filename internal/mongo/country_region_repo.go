package mongo

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ===== Countries =====
type CountryDoc struct {
	Code      string `bson:"code"`                // e.g. "US"
	Name      string `bson:"name,omitempty"`      // "United States"
	Continent string `bson:"continent,omitempty"` // "NA"
	Keywords  string `bson:"keywords,omitempty"`
}

func (c *Client) EnsureCountriesIndexes(ctx context.Context) error {
	col := c.DB.Collection("countries")
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "name", Value: 1}}},
		{Keys: bson.D{{Key: "keywords", Value: 1}}},
	})
	return err
}

func (c *Client) BulkUpsertCountries(ctx context.Context, docs []CountryDoc) error {
	if len(docs) == 0 {
		return nil
	}
	col := c.DB.Collection("countries")
	writes := make([]mongo.WriteModel, 0, len(docs))
	for _, d := range docs {
		if d.Code == "" {
			continue
		}
		w := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"code": d.Code}).
			SetUpdate(bson.M{"$set": d}).
			SetUpsert(true)
		writes = append(writes, w)
	}
	res, err := col.BulkWrite(ctx, writes, options.BulkWrite().SetOrdered(false))
	if err != nil {
		return err
	}
	log.Printf(`{"msg":"countries-bulkwrite","matched":%d,"modified":%d,"upserted":%d}`,
		res.MatchedCount, res.ModifiedCount, res.UpsertedCount)
	return nil
}

// ===== Regions =====
type RegionDoc struct {
	Code       string `bson:"code"`                  // e.g. "US-CA"
	LocalCode  string `bson:"local_code,omitempty"`  // e.g. "CA"
	Name       string `bson:"name,omitempty"`        // "California"
	ISOCountry string `bson:"iso_country,omitempty"` // "US"
	Continent  string `bson:"continent,omitempty"`   // "NA"
}

func (c *Client) EnsureRegionsIndexes(ctx context.Context) error {
	col := c.DB.Collection("regions")
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "code", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "iso_country", Value: 1}}},
		{Keys: bson.D{{Key: "name", Value: 1}}},
	})
	return err
}

func (c *Client) BulkUpsertRegions(ctx context.Context, docs []RegionDoc) error {
	if len(docs) == 0 {
		return nil
	}
	col := c.DB.Collection("regions")
	writes := make([]mongo.WriteModel, 0, len(docs))
	for _, d := range docs {
		if d.Code == "" {
			continue
		}
		w := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"code": d.Code}).
			SetUpdate(bson.M{"$set": d}).
			SetUpsert(true)
		writes = append(writes, w)
	}
	res, err := col.BulkWrite(ctx, writes, options.BulkWrite().SetOrdered(false))
	if err != nil {
		return err
	}
	log.Printf(`{"msg":"regions-bulkwrite","matched":%d,"modified":%d,"upserted":%d}`,
		res.MatchedCount, res.ModifiedCount, res.UpsertedCount)
	return nil
}
