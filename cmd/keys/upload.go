package keys

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/utils"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [keyID]",
	Short: "Upload a public key to S3 as JWKS",
	Long: `Upload a public key to S3 as a JWKS file.
The key must exist in the keys directory and will be uploaded to the specified S3 bucket.`,
	Args: cobra.ExactArgs(1),
	RunE: runUpload,
}

func init() {
	Cmd.AddCommand(uploadCmd)
}

func runUpload(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	keyID := args[0]

	cloudConnector, err := utils.NewS3Connector(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 connector: %w", err)
	}

	// Load the public key
	pubKeyPath := filepath.Join(viper.GetString("keys.dir"), fmt.Sprintf("%s.public.jwk", keyID))
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key file: %w", err)
	}

	pubKey, err := jwk.ParseKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	bucket, keyPath, err := cloudConnector.GetBucketAndKeyPath(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bucket and key path: %w", err)
	}

	tokenGenerator := utils.NewTokenGenerator(cloudConnector)

	var existingJWKS *utils.JWKS
	// Try to download existing JWKS
	data, err := cloudConnector.Download(ctx, bucket, keyPath)
	if err != nil {
		// If there's an error (like 404), start with empty JWKS
		existingJWKS = &utils.JWKS{
			Keys: []jwk.Key{},
		}
	} else {
		existingJWKS, err = tokenGenerator.ParseJWKS(ctx, data)
		if err != nil {
			return fmt.Errorf("failed to parse existing JWKS: %w", err)
		}
	}

	// Check if key with same ID already exists
	keyExists := false
	for i, key := range existingJWKS.Keys {
		kid, _ := key.Get(jwk.KeyIDKey)
		if kid == keyID {
			// Replace existing key
			existingJWKS.Keys[i] = pubKey
			keyExists = true
			break
		}
	}

	// Append new key if not found
	if !keyExists {
		existingJWKS.Keys = append(existingJWKS.Keys, pubKey)
	}

	// Upload combined JWKS to S3
	if err := tokenGenerator.UploadPublicKey(ctx, existingJWKS, bucket, keyPath); err != nil {
		return fmt.Errorf("failed to upload key to S3: %w", err)
	}

	return nil
}
