package keys

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/altacoda/tailbone/proto"
)

var Cmd = &cobra.Command{
	Use:   "keys",
	Short: "Key management commands",
	Long: `Key management commands for the Tailbone identity server.
These commands allow you to generate, upload, and manage signing keys.`,
}

func init() {
	Cmd.PersistentFlags().String("admin-address", "localhost:50051", "Address of the admin server")

	viper.BindPFlag("admin.address", Cmd.PersistentFlags().Lookup("admin-address"))
}

// getAdminClient creates a new gRPC client connection to the admin server
func getAdminClient(_ context.Context) (proto.AdminServiceClient, error) {
	conn, err := grpc.Dial(viper.GetString("admin.address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to admin server: %w", err)
	}

	return proto.NewAdminServiceClient(conn), nil
}
