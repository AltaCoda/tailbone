package keys

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/altacoda/tailbone/proto"
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

	client, err := getAdminClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.RemoveKey(ctx, &proto.RemoveKeyRequest{
		KeyId: keyID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove key: %w", err)
	}

	fmt.Printf("Successfully removed key %s from JWKS\n", keyID)
	return nil
}
