package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	DB *mongo.Database
	c  *mongo.Client
}

func NewClient(ctx context.Context, uri, db string) (*Client, error) {
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &Client{DB: cl.Database(db), c: cl}, nil

}
func (c *Client) Close(ctx context.Context) { _ = c.c.Disconnect(ctx) }
