package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/utils"
)

// Issuer defines the interface for JWT token operations
type Issuer interface {
	// IssueToken creates a new JWT token for a Tailscale user
	IssueToken(ctx context.Context, tailscaleUser, displayName string) (string, error)
	// GetJWKS returns the JSON Web Key Set
	GetJWKS(ctx context.Context) jwk.Set
	// VerifyToken verifies and parses a JWT token
	VerifyToken(ctx context.Context, tokenString string) (*TokenClaims, error)
}

// TokenIssuer handles JWT token issuance and verification
type TokenIssuer struct {
	keySet    jwk.Set
	activeKey jwk.Key
	logger    zerolog.Logger
}

// TokenClaims represents the custom claims in our JWT
type TokenClaims struct {
	jwt.RegisteredClaims
	User        string `json:"user"`
	DisplayName string `json:"display_name"`
}

// IssuerConfig holds the configuration for the token issuer
type IssuerConfig struct {
	KeyDir string // Directory containing the JWK files
}

// NewTokenIssuer creates a new JWT issuer with keys loaded from files
func NewTokenIssuer(ctx context.Context, cfg IssuerConfig) (Issuer, error) {
	// Find the most recent private key in the directory
	entries, err := os.ReadDir(cfg.KeyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var latestKey string
	var latestTime int64
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".private.jwk") {
			continue
		}
		// Extract timestamp from key-{timestamp}.private.jwk
		parts := strings.Split(strings.TrimSuffix(entry.Name(), ".private.jwk"), "-")
		if len(parts) != 2 {
			continue
		}
		timestamp, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		if timestamp > latestTime {
			latestTime = timestamp
			latestKey = entry.Name()
		}
	}

	if latestKey == "" {
		return nil, fmt.Errorf("no valid key files found in %s", cfg.KeyDir)
	}

	// Load the private key
	keyBytes, err := os.ReadFile(filepath.Join(cfg.KeyDir, latestKey))
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	key, err := jwk.ParseKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	// Create key set
	keySet := jwk.NewSet()
	if err := keySet.AddKey(key); err != nil {
		return nil, fmt.Errorf("failed to add key to set: %w", err)
	}

	return &TokenIssuer{
		keySet:    keySet,
		activeKey: key,
		logger:    utils.GetLogger("issuer"),
	}, nil
}

// IssueToken creates a new JWT token for a Tailscale user
func (i *TokenIssuer) IssueToken(ctx context.Context, tailscaleUser, displayName string) (string, error) {
	logger := i.logger
	logger.Debug().
		Str("user", tailscaleUser).
		Str("display_name", displayName).
		Msg("issuing new token")

	// Get the raw private key for signing
	var privateKey interface{}
	if err := i.activeKey.Raw(&privateKey); err != nil {
		return "", fmt.Errorf("failed to get raw private key: %w", err)
	}

	// Create the claims
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    viper.GetString("keys.issuer"),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(viper.GetDuration("keys.expiry"))),
		},
		User:        tailscaleUser,
		DisplayName: displayName,
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = i.activeKey.KeyID()

	// Sign the token
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	logger.Info().
		Str("user", tailscaleUser).
		Str("kid", i.activeKey.KeyID()).
		Msg("issued new token")

	return signedToken, nil
}

// GetJWKS returns the JSON Web Key Set
func (i *TokenIssuer) GetJWKS(ctx context.Context) jwk.Set {
	logger := i.logger
	publicKeySet := jwk.NewSet()
	for it := i.keySet.Keys(ctx); it.Next(ctx); {
		key := it.Pair().Value.(jwk.Key)
		public, err := jwk.PublicKeyOf(key)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get public key")
			continue
		}
		if err := publicKeySet.AddKey(public); err != nil {
			logger.Error().Err(err).Msg("failed to add public key to set")
		}
	}
	return publicKeySet
}

// VerifyToken verifies and parses a JWT token
func (i *TokenIssuer) VerifyToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Get the key ID from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("token has no key ID")
		}

		// Find the key in our key set
		key, found := i.keySet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key %s not found", kid)
		}

		// Get the public key
		var publicKey interface{}
		if err := key.Raw(&publicKey); err != nil {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
