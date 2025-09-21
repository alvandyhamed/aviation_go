package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AirportDoc struct {
	IDCSV        *int   `bson:"id_csv,omitempty"`
	Ident        string `bson:"ident,omitempty"`
	GPSCode      string `bson:"gps_code,omitempty"`
	IATACode     string `bson:"iata_code,omitempty"`
	Name         string `bson:"name,omitempty"`
	Type         string `bson:"type,omitempty"`
	Municipality string `bson:"municipality,omitempty"`
	ISOCountry   string `bson:"iso_country,omitempty"`
	ISORegion    string `bson:"iso_region,omitempty"`
	ElevationFt  *int   `bson:"elevation_ft,omitempty"`
	Continent    string `bson:"continent,omitempty"`
	Location     any    `bson:"location,omitempty"` // GeoJSON point
	IcaoCode     string `bson:"icao_code,omitempty"`
	HomeLink     string `bson:"home_link,omitempty"`
	WikipediaURL string `bson:"wikipedia_url,omitempty"`
}

func (c *Client) EnsureAirportIndexes(ctx context.Context) error {
	col := c.DB.Collection("airports")
	_, err := col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "ident", Value: 1}}},
		{Keys: bson.D{{Key: "gps_code", Value: 1}}},
		{Keys: bson.D{{Key: "iata_code", Value: 1}}},
		{Keys: bson.D{{Key: "icao_code", Value: 1}}},
		{Keys: bson.D{{Key: "home_link", Value: 1}}},
		{Keys: bson.D{{Key: "wikipedia_url", Value: 1}}},
		{Keys: bson.D{{Key: "iso_country", Value: 1}, {Key: "type", Value: 1}}},
		{Keys: bson.D{{Key: "location", Value: "2dsphere"}}},
		{
			Keys:    bson.D{{Key: "name", Value: "text"}, {Key: "municipality", Value: "text"}},
			Options: options.Index().SetWeights(bson.M{"name": 5, "municipality": 2}),
		},
	})
	return err
}

func (c *Client) BulkUpsertAirports(ctx context.Context, docs []AirportDoc) error {
	col := c.DB.Collection("airports")
	var writes []mongo.WriteModel
	for _, d := range docs {
		// کلید upsert: اگر id_csv داریم از آن؛ وگرنه ident
		filter := bson.M{}
		if d.IDCSV != nil {
			filter["id_csv"] = *d.IDCSV
		} else if d.Ident != "" {
			filter["ident"] = d.Ident
		}

		w := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(bson.M{"$set": d}).
			SetUpsert(true)
		writes = append(writes, w)
	}
	opts := options.BulkWrite().SetOrdered(false)
	_, err := col.BulkWrite(ctx, writes, opts)
	return err
}
