package keys

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/altacoda/tailbone/utils"
)

var removeCmd = &cobra.Command{
	Use:   "remove [keyID]",
	Short: "Remove a key from the JWKS in S3",
	Long: `Remove a key from the JSON Web Key Set (JWKS) stored in S3.
This will download the current JWKS, remove the specified key, and upload the updated JWKS.`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	Cmd.AddCommand(removeCmd)
	removeCmd.MarkFlagRequired("bucket")
}

func runRemove(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	keyID := args[0]

	cloudConnector, err := utils.NewS3Connector(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 connector: %w", err)
	}

	tokenGenerator := utils.NewTokenGenerator(cloudConnector)

	bucket, keyPath, err := cloudConnector.GetBucketAndKeyPath(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bucket and key path: %w", err)
	}

	// Download existing JWKS
	jwks, err := tokenGenerator.DownloadJWKS(ctx, bucket, keyPath)
	if err != nil {
		return fmt.Errorf("failed to download JWKS: %w", err)
	}

	// Remove the specified key
	updatedJWKS := tokenGenerator.RemoveKeyFromJWKS(jwks, keyID)

	// Upload the updated JWKS back to S3
	if err := tokenGenerator.UploadPublicKey(ctx, updatedJWKS, bucket, keyPath); err != nil {
		return fmt.Errorf("failed to upload updated JWKS: %w", err)
	}

	return nil
}
