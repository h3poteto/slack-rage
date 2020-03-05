package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RootCmd = &cobra.Command{
	Use:           "slack-rage",
	Short:         "Notify the rage of slack channel",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	cobra.OnInitialize()

	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose mode")
	RootCmd.AddCommand(versionCmd(), runServerCmd())
}

func generalConfig() bool {
	return viper.GetBool("verbose")
}
