package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type ILocalKeyStorage interface {
	GetLocalJWKs(ctx context.Context) (*JWKS, error)
	SaveLocalJWKs(ctx context.Context, jwks *JWKS) error
	DeleteLocalJWK(ctx context.Context, kid string) error
}

// LocalKeyStorage handles storage and retrieval of JWKs from local filesystem
type LocalKeyStorage struct {
	keyDir string
	logger zerolog.Logger
}

// NewLocalKeyStorage creates a new instance of LocalKeyStorage
func NewLocalKeyStorage() *LocalKeyStorage {
	return &LocalKeyStorage{
		keyDir: viper.GetString("keys.dir"),
		logger: GetLogger("local_key_storage"),
	}
}

// GetLocalJWKs retrieves all public keys from the local storage directory and returns them as JWKS
func (l *LocalKeyStorage) GetLocalJWKs(ctx context.Context) (*JWKS, error) {
	// Create key directory if it doesn't exist
	if err := os.MkdirAll(l.keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	// Read all files in the directory
	files, err := os.ReadDir(l.keyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	// Collect all public keys
	keys := make([]jwk.Key, 0)
	for _, file := range files {
		// Skip non-public key files
		if !strings.HasSuffix(file.Name(), ".public.jwk") && !strings.HasSuffix(file.Name(), ".private.jwk") {
			continue
		}

		// Read and parse the public key file
		keyPath := filepath.Join(l.keyDir, file.Name())
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			l.logger.Error().Err(err).Str("file", keyPath).Msg("failed to read public key file")
			continue
		}

		key, err := jwk.ParseKey(keyData)
		if err != nil {
			l.logger.Error().Err(err).Str("file", keyPath).Msg("failed to parse public key")
			continue
		}

		keys = append(keys, key)
	}

	// Create and marshal JWKS
	jwks := &JWKS{Keys: keys}

	l.logger.Info().Int("total_keys", len(keys)).Msg("retrieved local JWKS")
	return jwks, nil
}

// SaveLocalJWKs saves the provided JWKS to the local storage
func (l *LocalKeyStorage) SaveLocalJWKs(ctx context.Context, jwks *JWKS) error {
	// Create key directory if it doesn't exist
	if err := os.MkdirAll(l.keyDir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Save each key
	for _, key := range jwks.Keys {
		// Get key ID
		kid, ok := key.Get(jwk.KeyIDKey)
		if !ok {
			l.logger.Warn().Msg("skipping key without ID")
			continue
		}

		// Marshal the key
		keyBytes, err := json.MarshalIndent(key, "", "  ")
		if err != nil {
			l.logger.Error().Err(err).Interface("kid", kid).Msg("failed to marshal key")
			continue
		}

		// Save the key
		keyPath := filepath.Join(l.keyDir, fmt.Sprintf("%s.public.jwk", kid))
		if err := os.WriteFile(keyPath, keyBytes, 0644); err != nil {
			l.logger.Error().Err(err).Str("path", keyPath).Msg("failed to write key file")
			continue
		}

		l.logger.Debug().Interface("kid", kid).Str("path", keyPath).Msg("saved key")
	}

	l.logger.Info().Int("total_keys", len(jwks.Keys)).Msg("saved JWKS locally")
	return nil
}

// DeleteLocalJWKs removes all JWK files from the local storage
func (l *LocalKeyStorage) DeleteLocalJWK(ctx context.Context, kid string) error {
	publicKeyPath := filepath.Join(l.keyDir, fmt.Sprintf("%s.public.jwk", kid))
	privateKeyPath := filepath.Join(l.keyDir, fmt.Sprintf("%s.private.jwk", kid))

	// Delete each JWK pair
	l.logger.Info().Str("public_key", publicKeyPath).Str("private_key", privateKeyPath).Msg("deleting local JWK pair")
	if err := os.Remove(publicKeyPath); err != nil && !os.IsNotExist(err) {
		l.logger.Error().Err(err).Str("public_key", publicKeyPath).Msg("failed to delete public key")
		return fmt.Errorf("failed to delete public key: %w", err)
	}

	if err := os.Remove(privateKeyPath); err != nil && !os.IsNotExist(err) {
		l.logger.Error().Err(err).Str("private_key", privateKeyPath).Msg("failed to delete private key")
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	l.logger.Info().Str("kid", kid).Msg("deleted local JWK pair")
	return nil
}

var _ ILocalKeyStorage = &LocalKeyStorage{}
