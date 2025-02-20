package keys

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var Cmd = &cobra.Command{
	Use:   "keys",
	Short: "Key management commands",
	Long: `Key management commands for the Tailbone identity server.
These commands allow you to generate, upload, and manage signing keys.`,
}

func init() {
	Cmd.PersistentFlags().String("dir", "keys", "Directory containing the keys")
	Cmd.PersistentFlags().String("bucket", "", "S3 bucket for JWKS storage")
	Cmd.PersistentFlags().String("key-path", ".well-known/jwks.json", "Path/key for the JWKS file in S3")

	viper.BindPFlag("keys.dir", Cmd.PersistentFlags().Lookup("dir"))
	viper.BindPFlag("keys.bucket", Cmd.PersistentFlags().Lookup("bucket"))
	viper.BindPFlag("keys.keyPath", Cmd.PersistentFlags().Lookup("key-path"))
}
