package keys

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/core"
)

var housekeepingCmd = &cobra.Command{
	Use:   "housekeeping",
	Short: "Housekeeping for keys",
	RunE:  runHousekeeping,
}

func runHousekeeping(cmd *cobra.Command, args []string) error {
	// initialize here to avoid overlaps
	viper.BindPFlag("keys.dir", cmd.Flags().Lookup("dir"))
	viper.BindPFlag("keys.bucket", cmd.Flags().Lookup("bucket"))
	viper.BindPFlag("keys.keyPath", cmd.Flags().Lookup("key-path"))

	ctx := context.Background()
	housekeeper, err := core.NewHouseKeeper(ctx)
	if err != nil {
		return err
	}

	return housekeeper.Run(ctx)
}

func init() {
	Cmd.AddCommand(housekeepingCmd)

	housekeepingCmd.Flags().String("dir", "keys", "Directory containing the JWK files")
	housekeepingCmd.Flags().String("bucket", "", "S3 bucket for JWKS storage")
	housekeepingCmd.Flags().String("key-path", ".well-known/jwks.json", "Path/key for the JWKS file in S3")
}
