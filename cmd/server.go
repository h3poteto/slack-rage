package cmd

import (
	"github.com/h3poteto/slack-rage/server"
	"github.com/spf13/cobra"
)

type runServer struct {
	threshold int
	channel   string
}

func runServerCmd() *cobra.Command {
	r := &runServer{}
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run webhook server for slack Event API",
		Run:   r.run,
	}

	flags := cmd.Flags()
	flags.IntVarP(&r.threshold, "threshold", "t", 10, "Threshold for rage judgement.")
	flags.StringVarP(&r.channel, "channel", "c", "", "Notify channel.")

	return cmd

}

func (r *runServer) run(cmd *cobra.Command, args []string) {
	s := server.NewServer(r.threshold, r.channel)
	s.Serve()
}
