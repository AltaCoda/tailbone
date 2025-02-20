package keys

import (
	"context"
	"fmt"

	"github.com/altacoda/tailbone/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new signing key pair",
	Long: `Generate a new RSA key pair for signing JWTs.
The keys will be saved in JWK format with the key ID and timestamp in the filename.`,
	RunE: runGenerate,
}

func init() {
	Cmd.AddCommand(generateCmd)
	generateCmd.Flags().IntP("size", "s", 2048, "RSA key size in bits")
	viper.BindPFlag("keys.size", generateCmd.Flags().Lookup("size"))
}

func runGenerate(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Create S3 connector
	cloudConnector, err := utils.NewS3Connector(ctx)
	if err != nil {
		return fmt.Errorf("failed to create S3 connector: %w", err)
	}

	tokenGenerator := utils.NewTokenGenerator(cloudConnector)

	// Generate the key pair
	keyPair, err := tokenGenerator.GenerateKeyPair(ctx, viper.GetInt("keys.size"))
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Save the key pair
	if err := tokenGenerator.SaveLocally(ctx, keyPair, viper.GetString("keys.dir")); err != nil {
		return fmt.Errorf("failed to save key pair: %w", err)
	}

	return nil
}
