package cmd

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:           "slack-rage",
	Short:         "Notify the rage of slack channel",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	cobra.OnInitialize()

	RootCmd.AddCommand(versionCmd(), runServerCmd(), runRTMCmd())
}
