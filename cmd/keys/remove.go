package keys

import (
	"context"
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/altacoda/tailbone/proto"
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

	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func runRemove(_ *cobra.Command, args []string) error {
	ctx := context.Background()
	keyID := args[0]

	yes, _ := removeCmd.Flags().GetBool("yes")

	if yes || utils.ExpectYes("Are you sure you want to remove key?. This operation is not reversible.") {
		client, err := getAdminClient(ctx)
		if err != nil {
			return err
		}

		resp, err := client.RemoveKey(ctx, &proto.RemoveKeyRequest{
			KeyId: keyID,
		})
		if err != nil {
			return fmt.Errorf("failed to remove key: %w", err)
		}

		out := utils.OutData{
			Headers: table.Row{"KeyId", "Algorithm", "Created"},
			Rows:    []table.Row{},
		}

		for _, key := range resp.Keys {
			out.Rows = append(out.Rows, table.Row{key.KeyId, key.Algorithm, time.Unix(key.CreatedAt, 0).Format(time.RFC3339)})
			out.RawData = append(out.RawData, key)
		}

		return utils.Print(out)
	}

	return nil
}
