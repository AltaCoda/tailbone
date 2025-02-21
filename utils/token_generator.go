package utils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rs/zerolog"
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

// TokenGenerator interface defines the methods for managing JWT keys
type TokenGenerator interface {
	GenerateKeyPair(ctx context.Context, keySize int) (*KeyPair, error)
	SaveLocally(ctx context.Context, kp *KeyPair, keyDir string) error
	UploadPublicKey(ctx context.Context, jwks *JWKS, bucket, keyPath string) error
	DownloadJWKS(ctx context.Context, bucket, keyPath string) (*JWKS, error)
	RemoveKeyFromJWKS(jwks *JWKS, keyID string) *JWKS
	ParseJWKS(ctx context.Context, data []byte) (*JWKS, error)
}

// tokenGenerator implements the TokenGenerator interface
type tokenGenerator struct {
	logger          zerolog.Logger
	cloudConnector  CloudConnector
	localKeyStorage ILocalKeyStorage
}

// NewTokenGenerator creates a new instance of TokenGenerator
func NewTokenGenerator(cloudConnector CloudConnector, localKeyStorage ILocalKeyStorage) TokenGenerator {
	return &tokenGenerator{
		logger:          GetLogger("token_generator"),
		cloudConnector:  cloudConnector,
		localKeyStorage: localKeyStorage,
	}
}

// GenerateKeyPair creates a new RSA key pair with the specified size
func (t *tokenGenerator) GenerateKeyPair(ctx context.Context, keySize int) (*KeyPair, error) {
	// Generate a new RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create a JWK from the RSA key
	key, err := jwk.FromRaw(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWK: %w", err)
	}

	// Set key metadata
	kid := fmt.Sprintf("key-%d", time.Now().Unix())
	if err := key.Set(jwk.KeyIDKey, kid); err != nil {
		return nil, fmt.Errorf("failed to set key ID: %w", err)
	}
	if err := key.Set(jwk.AlgorithmKey, "RS256"); err != nil {
		return nil, fmt.Errorf("failed to set algorithm: %w", err)
	}

	// Get the public key
	publicKey, err := jwk.PublicKeyOf(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	t.logger.Info().Str("kid", kid).Msg("generated new key pair")

	return &KeyPair{
		PrivateKey: key,
		PublicKey:  publicKey,
		KeyID:      kid,
	}, nil
}

// SaveLocally saves the key pair to files in the specified directory
func (t *tokenGenerator) SaveLocally(ctx context.Context, kp *KeyPair, keyDir string) error {
	// Create JWKS with the public key
	jwks := &JWKS{
		Keys: []jwk.Key{kp.PublicKey},
	}

	// Save public key using local storage
	if err := t.localKeyStorage.SaveLocalJWKs(ctx, jwks); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	// Save private key separately
	privBytes, err := json.MarshalIndent(kp.PrivateKey, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	privPath := filepath.Join(keyDir, fmt.Sprintf("%s.private.jwk", kp.KeyID))
	if err := os.WriteFile(privPath, privBytes, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	t.logger.Info().
		Str("kid", kp.KeyID).
		Str("private_key", privPath).
		Msg("saved key pair")

	return nil
}

// UploadPublicKey uploads the JWKS to the specified S3 bucket
func (t *tokenGenerator) UploadPublicKey(ctx context.Context, jwks *JWKS, bucket, keyPath string) error {
	jwksBytes, err := json.Marshal(jwks)
	if err != nil {
		return fmt.Errorf("failed to marshal JWKS: %w", err)
	}

	// Upload to S3
	if err := t.cloudConnector.Upload(ctx, bucket, keyPath, jwksBytes); err != nil {
		return fmt.Errorf("failed to upload JWKS to S3: %w", err)
	}

	t.logger.Info().
		Str("bucket", bucket).
		Str("key_path", keyPath).
		Int("total_keys", len(jwks.Keys)).
		Msg("uploaded JWKS to S3")

	return nil
}

// DownloadJWKS downloads and parses a JWKS from a given URL
func (t *tokenGenerator) DownloadJWKS(ctx context.Context, bucket, keyPath string) (*JWKS, error) {
	data, err := t.cloudConnector.Download(ctx, bucket, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download JWKS: %w", err)
	}

	return t.ParseJWKS(ctx, data)
}

// RemoveKeyFromJWKS removes a key from the JWKS by its ID and returns the updated JWKS
func (t *tokenGenerator) RemoveKeyFromJWKS(jwks *JWKS, keyID string) *JWKS {
	// Create new slice for remaining keys
	remainingKeys := make([]jwk.Key, 0, len(jwks.Keys))
	removed := false

	// Filter out the key with matching ID
	for _, key := range jwks.Keys {
		kid, _ := key.Get(jwk.KeyIDKey)
		if kid != keyID {
			remainingKeys = append(remainingKeys, key)
		} else {
			removed = true
		}
	}

	if removed {
		t.logger.Info().
			Str("kid", keyID).
			Int("remaining_keys", len(remainingKeys)).
			Msg("removed key from JWKS")
	} else {
		t.logger.Warn().
			Str("kid", keyID).
			Msg("key not found in JWKS")
	}

	return &JWKS{
		Keys: remainingKeys,
	}
}

func (t *tokenGenerator) ParseJWKS(ctx context.Context, data []byte) (*JWKS, error) {
	set, err := jwk.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWKS: %w", err)
	}

	keys := make([]jwk.Key, 0, set.Len())
	iter := set.Keys(ctx)
	for iter.Next(ctx) {
		pair := iter.Pair()
		keys = append(keys, pair.Value.(jwk.Key))
	}

	return &JWKS{
		Keys: keys,
	}, nil
}
