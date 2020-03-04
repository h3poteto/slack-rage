package cmd

import (
	"github.com/h3poteto/slack-rage/server"
	"github.com/spf13/cobra"
)

type runServer struct {
	channel string
}

func runServerCmd() *cobra.Command {
	r := &runServer{}
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run webhook server for slack Event API",
		Run:   r.run,
	}

	flags := cmd.Flags()
	flags.StringVarP(&r.channel, "channel", "c", ".*", "Event watching channel name. Please provide regexp format.")

	return cmd

}

func (r *runServer) run(cmd *cobra.Command, args []string) {
	s := &server.Server{
		Channel: r.channel,
	}
	s.Serve()
}
