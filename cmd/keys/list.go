package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/altacoda/tailbone/proto"
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
}

func runList(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	client, err := getAdminClient(ctx)
	if err != nil {
		return err
	}

	resp, err := client.ListKeys(ctx, &proto.ListKeysRequest{})
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(resp.Keys) == 0 {
		location := "in remote JWKS"
		if localKeys {
			location = "in local directory"
		}
		fmt.Printf("No keys found %s\n", location)
		return nil
	}

	fmt.Println("----------")

	for _, key := range resp.Keys {
		fmt.Printf("Key ID: %s\n", key.KeyId)
		fmt.Printf("Algorithm: %s\n", key.Algorithm)
		if key.CreatedAt > 0 {
			fmt.Printf("Created: %v\n", time.Unix(key.CreatedAt, 0).Format(time.RFC3339))
		}
		if key.Filename != "" {
			fmt.Printf("File: %s\n", key.Filename)
		}
		fmt.Println("----------")
	}

	return nil
}
