package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/spf13/viper"
)

// KeyPair represents a generated key pair with metadata
type KeyPair struct {
	PrivateKey jwk.Key
	PublicKey  jwk.Key
	KeyID      string
}

// JWKS represents a JSON Web Key Set
type JWKS struct {
	Keys []jwk.Key `json:"keys"`
}

func ParseCreatedAt(keyID string) (time.Time, error) {
	if parts := strings.Split(keyID, "-"); len(parts) > 1 {
		if ts, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
			return time.Time{}, err
		} else {
			return time.Unix(ts, 0), nil
		}
	}

	return time.Time{}, errors.New("invalid key ID")
}

func (kp *KeyPair) CreatedAt() time.Time {
	ts, err := ParseCreatedAt(kp.KeyID)
	if err != nil {
		return time.Time{}
	}

	return ts
}

func GetKeyId(t time.Time) string {
	return fmt.Sprintf("%s-%s", viper.GetString("key.prefix"), strconv.FormatInt(t.Unix(), 10))
}
