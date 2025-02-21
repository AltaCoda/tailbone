package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/altacoda/tailbone/cmd/keys"
	"github.com/altacoda/tailbone/cmd/server"
	"github.com/altacoda/tailbone/utils"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:     "tailbone",
		Short:   "Tailbone is an identity provider based on JWT and using Tailscale for authentication",
		Long:    `Tailbone is an identity provider based on JWT and using Tailscale for authentication. It is used to authenticate users and provide them with a JWT token.`,
		Version: fmt.Sprintf("%s-%s", utils.Version, utils.Commit),
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tailbone.*)")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "output format (json, yaml, text)")

	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.SetDefault("key.prefix", "tb") // WARNING: Changing this can lead to keys being left behind

	// Add commands
	rootCmd.AddCommand(server.Cmd)
	rootCmd.AddCommand(keys.Cmd)

	// Set environment variable bindings
	viper.SetEnvPrefix("TB")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName(".tailbone")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
