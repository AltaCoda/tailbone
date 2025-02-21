package keys

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/proto"
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

	client, err := getAdminClient(ctx)
	if err != nil {
		return err
	}

	resp, err := client.GenerateNewKeys(ctx, &proto.GenerateNewKeysRequest{})
	if err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}

	fmt.Printf("Generated new key pair with ID: %s\n", resp.KeyId)
	return nil
}
