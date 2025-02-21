package keys

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"

	"github.com/altacoda/tailbone/proto"
	"github.com/altacoda/tailbone/utils"
)

var (
	localKeys bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List available signing keys",
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
		fmt.Fprintln(os.Stderr, "Failed to list keys")
		return err
	}

	if len(resp.Keys) == 0 {
		fmt.Fprintln(os.Stderr, "No keys found")
		return nil
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
