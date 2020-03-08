package cmd

import (
	"github.com/h3poteto/slack-rage/rtm"
	"github.com/spf13/cobra"
)

type runRTM struct {
	threshold int
	period    int
	channel   string
	verbose   bool
}

func runRTMCmd() *cobra.Command {
	r := &runRTM{}
	cmd := &cobra.Command{
		Use:   "rtm",
		Short: "Run bot using Real Time Message",
		Run:   r.run,
	}

	flags := cmd.Flags()
	flags.IntVarP(&r.threshold, "threshold", "t", 10, "Threshold for rage judgement.")
	flags.IntVarP(&r.period, "period", "p", 60, "Observation period seconds for rage judgement. This CLI notify when there are more than threshold posts per period.")
	flags.StringVarP(&r.channel, "channel", "c", "", "Notify channel.")
	flags.BoolVarP(&r.verbose, "verbose", "v", false, "Enable verbose mode")

	return cmd
}

func (r *runRTM) run(cmd *cobra.Command, args []string) {
	s := rtm.New(r.threshold, r.period, r.channel, r.verbose)
	s.Start()
}
