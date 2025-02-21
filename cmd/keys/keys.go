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
	Use:     "keys",
	Aliases: []string{"key"},
	Short:   "Key management commands",
	Long: `Key management commands for the Tailbone identity server.
These commands allow you to generate, upload, and manage signing keys.`,
}

func init() {
	Cmd.PersistentFlags().String("host", "", "Address of the admin server")
	Cmd.PersistentFlags().Int("port", 50051, "Port of the admin server")

	viper.BindPFlag("admin.client.host", Cmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("admin.client.port", Cmd.PersistentFlags().Lookup("port"))
}

// getAdminClient creates a new gRPC client connection to the admin server
func getAdminClient(_ context.Context) (proto.AdminServiceClient, error) {
	addr := fmt.Sprintf("%s:%d", viper.GetString("admin.client.host"), viper.GetInt("admin.client.port"))
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to admin server on %s: %w", addr, err)
	}

	return proto.NewAdminServiceClient(conn), nil
}
