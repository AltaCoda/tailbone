package keys

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/altacoda/tailbone/utils"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	localKeys bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available signing keys",
	Long: `List all available signing keys.
By default, lists keys from the configured JWKS endpoint.
Use --local flag to list keys from the local filesystem instead.`,
	RunE: runList,
}

func init() {
	Cmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&localKeys, "local", "l", false, "List keys from local filesystem")
}

func runList(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	if localKeys {
		return listLocalKeys(ctx)
	}
	return listRemoteKeys(ctx)
}

func listLocalKeys(_ context.Context) error {
	keyDir := viper.GetString("keys.dir")
	// Check if key directory exists
	if _, err := os.Stat(keyDir); os.IsNotExist(err) {
		return fmt.Errorf("key directory %s does not exist", keyDir)
	}

	// Find all public key files
	files, err := filepath.Glob(filepath.Join(keyDir, "*.public.jwk"))
	if err != nil {
		return fmt.Errorf("failed to list key files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No keys found in local directory")
		return nil
	}

	fmt.Println("Local keys:")
	fmt.Println("----------")

	for _, file := range files {
		// Read and parse the key file
		keyBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read key file %s: %w", file, err)
		}

		key, err := jwk.ParseKey(keyBytes)
		if err != nil {
			return fmt.Errorf("failed to parse key file %s: %w", file, err)
		}

		// Extract key metadata
		kid, _ := key.Get(jwk.KeyIDKey)
		alg, _ := key.Get(jwk.AlgorithmKey)

		// Extract timestamp from key ID (assuming format "key-{timestamp}")
		var timestamp time.Time
		if kidStr, ok := kid.(string); ok {
			if parts := strings.Split(kidStr, "-"); len(parts) > 1 {
				if ts, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					timestamp = time.Unix(ts, 0)
				}
			}
		}

		fmt.Printf("Key ID: %v\n", kid)
		fmt.Printf("Algorithm: %v\n", alg)
		if !timestamp.IsZero() {
			fmt.Printf("Created: %v\n", timestamp.Format(time.RFC3339))
		}
		fmt.Printf("File: %s\n", filepath.Base(file))
		fmt.Println("----------")
	}

	return nil
}

func listRemoteKeys(ctx context.Context) error {
	cloudConnector, err := utils.NewS3Connector(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 connector: %w", err)
	}

	tokenGenerator := utils.NewTokenGenerator(cloudConnector)

	// Check if we have either JWKS URL or S3 bucket
	if viper.GetString("keys.bucket") == "" {
		return fmt.Errorf("keys.bucket is required")
	}

	// Get JWKS URL
	bucket, keyPath, err := cloudConnector.GetBucketAndKeyPath(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bucket and key path: %w", err)
	}

	// Download JWKS
	jwks, err := tokenGenerator.DownloadJWKS(ctx, bucket, keyPath)
	if err != nil {
		return fmt.Errorf("failed to download JWKS: %w", err)
	}

	if len(jwks.Keys) == 0 {
		fmt.Println("No keys found in remote JWKS")
		return nil
	}

	fmt.Printf("Remote keys from %s:\n", keyPath)
	fmt.Println("----------")

	for _, key := range jwks.Keys {
		// Extract key metadata
		kid, _ := key.Get(jwk.KeyIDKey)
		alg, _ := key.Get(jwk.AlgorithmKey)

		// Extract timestamp from key ID (assuming format "key-{timestamp}")
		var timestamp time.Time
		if kidStr, ok := kid.(string); ok {
			if parts := strings.Split(kidStr, "-"); len(parts) > 1 {
				if ts, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					timestamp = time.Unix(ts, 0)
				}
			}
		}

		fmt.Printf("Key ID: %v\n", kid)
		fmt.Printf("Algorithm: %v\n", alg)
		if !timestamp.IsZero() {
			fmt.Printf("Created: %v\n", timestamp.Format(time.RFC3339))
		}
		fmt.Println("----------")
	}

	return nil
}
