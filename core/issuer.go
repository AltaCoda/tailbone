package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// IssuerConfig holds the configuration for the token issuer
type IssuerConfig struct {
	KeyDir string // Directory containing the JWK files
}

// TokenIssuer handles JWT token issuance and verification
type TokenIssuer struct {
	keySet jwk.Set
	config IssuerConfig
	logger zerolog.Logger
}

// TokenClaims represents the custom claims in our JWT
type TokenClaims struct {
	jwt.RegisteredClaims
	User        string `json:"user"`
	DisplayName string `json:"display_name"`
}

// NewTokenIssuer creates a new JWT issuer with keys loaded from files
func NewTokenIssuer(ctx context.Context, cfg IssuerConfig) (Issuer, error) {
	logger := utils.GetLogger("issuer")

	issuer := &TokenIssuer{
		config: cfg,
		logger: logger,
	}

	key, err := issuer.loadLatestKey(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load latest key: %w", err)
	}

	if key == nil {
		logger.Warn().Str("dir", cfg.KeyDir).Msg("no valid key files found in directory. issue function will fail")
	} else {
		issuer.keySet = jwk.NewSet()
	}

	return issuer, nil
}

func (i *TokenIssuer) loadLatestKey(_ context.Context) (jwk.Key, error) {
	i.logger.Info().Str("dir", i.config.KeyDir).Msg("loading latest key")
	// Find the most recent private key in the directory
	entries, err := os.ReadDir(i.config.KeyDir)
	if err != nil {
		i.logger.Error().Err(err).Msg("failed to read key directory")
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
		ts, err := utils.ParseCreatedAt(entry.Name())
		if err != nil {
			continue
		}
		timestamp := ts.Unix()
		if timestamp > latestTime {
			latestTime = timestamp
			latestKey = entry.Name()
		}
	}

	if latestKey == "" {
		i.logger.Warn().Str("dir", i.config.KeyDir).Msg("no valid key files found in directory. issue function will fail")
		return nil, nil
	}

	// Load the private key
	keyBytes, err := os.ReadFile(filepath.Join(i.config.KeyDir, latestKey))
	if err != nil {
		i.logger.Error().Err(err).Str("key", latestKey).Msg("failed to read key file")
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	key, err := jwk.ParseKey(keyBytes)
	if err != nil {
		i.logger.Error().Err(err).Str("key", latestKey).Msg("failed to parse key")
		return nil, fmt.Errorf("failed to parse key: %w", err)
	}

	// Create key set
	keySet := jwk.NewSet()
	if err := keySet.AddKey(key); err != nil {
		i.logger.Error().Err(err).Str("key", latestKey).Msg("failed to add key to set")
		return nil, fmt.Errorf("failed to add key to set: %w", err)
	}

	return key, nil
}

// IssueToken creates a new JWT token for a Tailscale user
func (i *TokenIssuer) IssueToken(ctx context.Context, tailscaleUser, displayName string) (string, error) {
	i.logger.Debug().
		Str("user", tailscaleUser).
		Str("display_name", displayName).
		Msg("issuing new token")

	key, err := i.loadLatestKey(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load latest key: %w", err)
	}

	if key == nil {
		return "", fmt.Errorf("no valid key files found in directory. issue function will fail")
	}

	// Get the raw private key for signing
	var privateKey interface{}
	if err := key.Raw(&privateKey); err != nil {
		i.logger.Error().Err(err).Str("key", key.KeyID()).Msg("failed to get raw private key")
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
	token.Header["kid"] = key.KeyID()

	// Sign the token
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		i.logger.Error().Err(err).Str("key", key.KeyID()).Msg("failed to sign token")
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	i.logger.Info().
		Str("user", tailscaleUser).
		Str("kid", key.KeyID()).
		Msg("issued new token")

	i.logger.Info().Str("token", signedToken).Msg("issued new token")

	return signedToken, nil
}

// GetJWKS returns the JSON Web Key Set
func (i *TokenIssuer) GetJWKS(ctx context.Context) jwk.Set {
	publicKeySet := jwk.NewSet()
	for it := i.keySet.Keys(ctx); it.Next(ctx); {
		key := it.Pair().Value.(jwk.Key)
		public, err := jwk.PublicKeyOf(key)
		if err != nil {
			i.logger.Error().Err(err).Str("key", key.KeyID()).Msg("failed to get public key")
			continue
		}
		if err := publicKeySet.AddKey(public); err != nil {
			i.logger.Error().Err(err).Str("key", key.KeyID()).Msg("failed to add public key to set")
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
			i.logger.Error().Msg("token has no key ID")
			return nil, fmt.Errorf("token has no key ID")
		}

		// Find the key in our key set
		key, found := i.keySet.LookupKeyID(kid)
		if !found {
			i.logger.Error().Str("key", kid).Msg("key not found")
			return nil, fmt.Errorf("key %s not found", kid)
		}

		// Get the public key
		var publicKey interface{}
		if err := key.Raw(&publicKey); err != nil {
			i.logger.Error().Err(err).Str("key", kid).Msg("failed to get public key")
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		i.logger.Error().Err(err).Msg("failed to parse token")
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		i.logger.Error().Msg("invalid token")
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
