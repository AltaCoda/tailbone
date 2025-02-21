package keys

import (
	"context"
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/proto"
	"github.com/altacoda/tailbone/utils"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new signing key pair",
	Long: `Generate a new RSA key pair for signing JWTs.
The keys will be saved in JWK format with the key ID and timestamp in the filename.`,
	RunE: runGenerate,
	PreRun: func(cmd *cobra.Command, _ []string) {
		viper.BindPFlag("host", cmd.PersistentFlags().Lookup("host"))
	},
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

	out := utils.OutData{
		Headers: table.Row{"KeyId", "Algorithm"},
		Rows:    []table.Row{},
	}

	out.Rows = append(out.Rows, table.Row{resp.Key.KeyId, resp.Key.Algorithm})
	out.RawData = append(out.RawData, resp)

	return utils.Print(out)
}
