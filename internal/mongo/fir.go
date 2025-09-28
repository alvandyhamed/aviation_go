package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// اگر قبلاً Client و c.db رو داری، همونو نگه دار
func (c *Client) FIRsCollection() *mongo.Collection {
	return c.DB.Collection("firs")
}

type FIRDoc struct {
	// _id لازم نیست دستی ست کنی
	Country   string    `bson:"country,omitempty"`    // "IR"
	FIRName   string    `bson:"fir_name,omitempty"`   // "Tehran FIR"
	FIRCode   string    `bson:"fir_code,omitempty"`   // "OIIX" (ممکنه خالی باشه)
	Geometry  any       `bson:"geometry,omitempty"`   // GeoJSON (map[string]any)
	Source    string    `bson:"source,omitempty"`     // "openAIP" یا ...
	UpdatedAt time.Time `bson:"updated_at,omitempty"` // زمان upsert
}

// ایندکس‌ها
func (c *Client) EnsureFIRIndexes(ctx context.Context) error {
	col := c.FIRsCollection()

	idxes := []mongo.IndexModel{
		{
			// جلوگیری از تکراری شدن رکوردها
			Keys:    bson.D{{Key: "country", Value: 1}, {Key: "fir_name", Value: 1}},
			Options: options.Index().SetName("uniq_country_firname").SetUnique(true),
		},
		{
			// برای کوئری‌های مکانی
			Keys:    bson.D{{Key: "geometry", Value: "2dsphere"}},
			Options: options.Index().SetName("geo_geometry"),
		},
	}
	_, err := col.Indexes().CreateMany(ctx, idxes)
	return err
}

// Upsert گروهی
func (c *Client) BulkUpsertFIRs(ctx context.Context, items []FIRDoc) error {
	if len(items) == 0 {
		return nil
	}
	col := c.FIRsCollection()

	now := time.Now().UTC()
	models := make([]mongo.WriteModel, 0, len(items))

	for _, it := range items {
		// زمان به‌روزرسانی
		it.UpdatedAt = now

		filter := bson.M{
			"country":  it.Country,
			"fir_name": it.FIRName,
		}

		update := bson.M{
			"$set": bson.M{
				"country":    it.Country, // ✅ تایپو قبلی اینجا بود
				"fir_name":   it.FIRName,
				"fir_code":   it.FIRCode,
				"geometry":   it.Geometry,
				"source":     it.Source,
				"updated_at": it.UpdatedAt,
			},
		}

		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true))
	}

	_, err := col.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false))
	return err
}
