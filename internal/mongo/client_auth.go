package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ClientSecretVersion struct {
	Version   string `bson:"version"`    // مثل "v1","v2"
	SecretEnc string `bson:"secret_enc"` // secret رمز شده (Base64)
	Active    bool   `bson:"active"`
}

type APIClient struct {
	ClientID      string                `bson:"client_id"`
	Status        string                `bson:"status"`                 // "active" | "disabled"
	AllowedIPs    []string              `bson:"allowed_ips,omitempty"`  // IP یا CIDR
	RatePerMinute int                   `bson:"rate_per_min,omitempty"` // پیش‌فرض 29
	Secrets       []ClientSecretVersion `bson:"secrets"`                // حداقل یک نسخه فعال
	Scopes        []string              `bson:"scopes,omitempty"`
	CreatedAt     time.Time             `bson:"created_at"`
	UpdatedAt     time.Time             `bson:"updated_at"`
	Notes         string                `bson:"notes,omitempty"`
}

func (c *Client) clientsCol() *mongo.Collection { return c.DB.Collection("api_clients") }

func (c *Client) GetAPIClient(ctx context.Context, clientID string) (*APIClient, error) {
	var out APIClient
	err := c.clientsCol().FindOne(ctx, bson.M{"client_id": clientID}).Decode(&out)
	if err != nil {
		return nil, err
	}
	if out.Status != "active" {
		return nil, errors.New("client disabled")
	}
	return &out, nil
}

// کمکی برای گرفتن secret نسخه مورد نظر (یا فعال)
func (cl *APIClient) FindSecret(version string) (ver string, enc string, ok bool) {
	if version != "" {
		for _, v := range cl.Secrets {
			if v.Version == version && v.Active {
				return v.Version, v.SecretEnc, true
			}
		}
		return "", "", false
	}
	// اگر نسخه مشخص نشد، اولین secret فعال
	for _, v := range cl.Secrets {
		if v.Active {
			return v.Version, v.SecretEnc, true
		}
	}
	return "", "", false
}

/*
CANONICAL
<METHOD>
<lowercase PATH>
<sorted query>
<sha256(body)>
<X-Date>
<X-Nonce>
<X-Key-Version>

*/

//UUID=c4f2c1c6-8e82-4a3a-9a0f-0e6f1b8d2c7e Random
//Unix time=1759240841
//=>e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 =>Body if Get ```echo -n "" | sha256sum```

//OPEN SSL
/*
SECRET=$(echo "SHksSSBhbSBHcmlmZmluIGZvcm0gSEBtZWQ=" | base64 -d)
CANON="GET
/airportslist
limit=20&page=1&q=hamed
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
1759240841
c4f2c1c6-8e82-4a3a-9a0f-0e6f1b8d2c7e
v1"

SIG=$(printf "%b" "$CANON" | openssl dgst -sha256 -hmac "$SECRET" -binary | openssl base64 -A)
echo "X-Signature: $SIG"


*/
