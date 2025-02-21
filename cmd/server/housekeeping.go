package server

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/altacoda/tailbone/core"
)

var housekeepingCmd = &cobra.Command{
	Use:   "housekeeping",
	Short: "Housekeeping for keys",
	RunE:  runHousekeeping,
}

func runHousekeeping(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	housekeeper, err := core.NewHouseKeeper(ctx)
	if err != nil {
		return err
	}

	return housekeeper.Run(ctx)
}

func init() {
	Cmd.AddCommand(housekeepingCmd)
}
