package cmd

import (
	"github.com/h3poteto/slack-rage/rtm"
	"github.com/spf13/cobra"
)

type runRTM struct {
}

func runRTMCmd() *cobra.Command {
	r := &runRTM{}
	cmd := &cobra.Command{
		Use:   "rtm",
		Short: "Run bot using Real Time Message",
		Run:   r.run,
	}
	return cmd
}

func (r *runRTM) run(cmd *cobra.Command, args []string) {
	s := rtm.New()
	s.Start()
}
